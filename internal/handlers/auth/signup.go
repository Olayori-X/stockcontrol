package authhandlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Olayori-X/stock-control-backend/api"
	"github.com/Olayori-X/stock-control-backend/functions"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	"github.com/Olayori-X/stock-control-backend/models"
	log "github.com/sirupsen/logrus"
)

func SignupHandler(w http.ResponseWriter, r *http.Request) {
	var params api.SignupParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Error(err)
		api.RequestErrorHandler(w, err)
		return
	}

	if params.Name == "" {
		api.RequestErrorHandler(w, errors.New("name is required"))
		return
	}
	if params.Email == "" {
		api.RequestErrorHandler(w, errors.New("email is required"))
		return
	}
	if params.Phone == "" {
		api.RequestErrorHandler(w, errors.New("phone is required"))
		return
	}
	if params.Role != "admin" && params.Role != "sales" && params.Role != "distributor" && params.Role != "supervisor" {
		api.RequestErrorHandler(w, errors.New("role must be admin, sales, distributor, or supervisor"))
		return
	}

	// Password is required for every role except sales — sales associates
	// authenticate via PIN instead, so there's nothing for them to
	// meaningfully set a password to.
	var passwordPtr *string
	if params.Role != "sales" {
		if params.Password == "" {
			api.RequestErrorHandler(w, errors.New("password is required"))
			return
		}
		hashed, err := functions.HashString(params.Password)
		if err != nil {
			log.Error("Failed to hash password: ", err)
			api.InternalErrorHandler(w)
			return
		}
		passwordPtr = &hashed
	}

	newUser := models.User{
		Name:     params.Name,
		Email:    params.Email,
		Phone:    params.Phone,
		Role:     params.Role,
		Password: passwordPtr,
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	if err := (*database).AddUser(&newUser); err != nil {
		log.Error("Failed to create user: ", err)
		api.InternalErrorHandler(w)
		return
	}

	response := api.SignupResponse{
		UserID: newUser.UserID,
		Name:   newUser.Name,
		Role:   newUser.Role,
	}

	// Sales associates get their PIN generated right here — no separate
	// admin follow-up step needed for the account to actually be usable.
	if newUser.Role == "sales" {
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
		if err := (*database).SetUserPIN(newUser.UserID, hashedPin); err != nil {
			log.Error("Failed to set PIN during signup: ", err)
			api.InternalErrorHandler(w)
			return
		}
		response.PIN = pin
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}
