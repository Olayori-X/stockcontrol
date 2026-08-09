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
)

func ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	var params api.ChangePasswordParams

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Error("Invalid request body: ", err)
		api.RequestErrorHandler(w, err)
		return
	}

	if strings.TrimSpace(params.Email) == "" {
		api.RequestErrorHandler(w, errors.New("this email cannot be validated"))
		return
	}
	if strings.TrimSpace(params.Password) == "" {
		api.RequestErrorHandler(w, errors.New("password cannot be empty"))
		return
	}

	if strings.TrimSpace(params.ConfirmPassword) != strings.TrimSpace(params.Password) {
		api.RequestErrorHandler(w, errors.New("passwords do not match"))
		return
	}

	hashedPassword, err := functions.HashString(params.Password)
	if err != nil {
		log.Error("Failed to hash password: ", err)
		api.InternalErrorHandler(w)
		return
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Database connection failed: ", err)
		api.InternalErrorHandler(w)
		return
	}

	err = (*database).ChangeUserPassword(params.Email, string(hashedPassword))
	if err != nil {
		log.Error("Failed to change password: ", err)
		api.InternalErrorHandler(w)
		return
	}

	response := api.ChangePasswordResponse{
		Code:    http.StatusOK,
		Success: true,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
