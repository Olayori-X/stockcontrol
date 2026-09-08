package authhandlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Olayori-X/stock-control-backend/api"
	"github.com/Olayori-X/stock-control-backend/functions"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// SetPINHandler generates a fresh 8-digit PIN for a user, hashes and
// stores it, and returns the plaintext PIN exactly once — the admin must
// relay it to the associate now, since it's never retrievable again
// (only the hash is stored, same as passwords).
func SetPINHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	var params api.SetPINInput
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Error(err)
		api.RequestErrorHandler(w, err)
		return
	}

	if params.UserID == "" {
		api.RequestErrorHandler(w, errors.New("user_id is required"))
		return
	}

	pin, err := functions.GenerateNumericPIN(8)
	if err != nil {
		log.Error("Failed to generate PIN: ", err)
		api.InternalErrorHandler(w)
		return
	}

	hashedPin, err := functions.HashString(pin)
	if err != nil {
		log.Error("Failed to hash PIN: ", err)
		api.InternalErrorHandler(w)
		return
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	if err := (*database).SetUserPIN(params.UserID, hashedPin); err != nil {
		log.Error("Failed to set PIN: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(api.SetPINResponse{
		UserID: params.UserID,
		PIN:    pin,
	})
}

// PINLoginHandler authenticates a Sales Associate by user_id + PIN — the
// APK's login path, separate from the email/password LoginHandler used by
// other roles. PIN identifies/authenticates only; it never authorizes
// stock release (ConfirmPickupRequest does that).
func PINLoginHandler(w http.ResponseWriter, r *http.Request) {
	var params api.PINLoginInput
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Error(err)
		api.RequestErrorHandler(w, err)
		return
	}

	if strings.TrimSpace(params.UserID) == "" {
		api.RequestErrorHandler(w, errors.New("user_id cannot be empty"))
		return
	}
	if strings.TrimSpace(params.PIN) == "" {
		api.RequestErrorHandler(w, errors.New("pin cannot be empty"))
		return
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	details := (*database).GetUserPINLoginDetails(params.UserID)
	if details == nil {
		log.Warn("PIN login attempt with unknown/PIN-less user: ", params.UserID)
		api.UnAuthorizedError(w)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(details.PinHash), []byte(params.PIN)); err != nil {
		log.Warn("Invalid PIN login attempt for: ", params.UserID)
		api.UnAuthorizedError(w)
		return
	}

	// PIN auth is Sales Associate-specific — reject other roles even if
	// they somehow have a pin_hash set, so this endpoint can't become a
	// backdoor for admin/distributor accounts.
	if details.Role != "sales" {
		log.Warn("PIN login attempted for non-sales role: ", params.UserID)
		api.UnAuthorizedError(w)
		return
	}

	if err := checkSalesAssociateResumption(database, details.UserID, details.Role, params.Latitude, params.Longitude, params.DeviceRef); err != nil {
		if err.Error() == "internal error" {
			api.InternalErrorHandler(w)
		} else {
			api.RequestErrorHandler(w, err)
		}
		return
	}

	code, err := functions.GenerateAuthorizationCode()
	if err != nil {
		log.Error("Generating authorization code failed", err)
		api.InternalErrorHandler(w)
		return
	}

	token, err := functions.HashString(code)
	if err != nil {
		log.Error("Hashing authorization code failed", err)
		api.InternalErrorHandler(w)
		return
	}

	if err := (*database).UpsertLoggedInUser(details.UserID, token, details.Role); err != nil {
		log.Error("An error occurred with authorization table", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(api.LoginResponse{
		Code:     http.StatusOK,
		Message:  "Login successful",
		UserID:   details.UserID,
		Role:     details.Role,
		Verified: details.Verified,
		Token:    code,
	})
}
