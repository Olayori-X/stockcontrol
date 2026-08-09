package authhandlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Olayori-X/stock-control-backend/api"
	"github.com/Olayori-X/stock-control-backend/functions"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	"github.com/Olayori-X/stock-control-backend/models"
	log "github.com/sirupsen/logrus"
)

func SignupHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("SignupHandler called")
	var params = api.SignupParams{}
	// var decoder *schema.Decoder = schema.NewDecoder()
	var err error

	err = json.NewDecoder(r.Body).Decode(&params)

	if err != nil {
		log.Error(err)
		api.RequestErrorHandler(w, err)
		return
	}
	if strings.TrimSpace(params.Name) == "" {
		api.RequestErrorHandler(w, errors.New("name cannot be empty"))
		return
	}

	if strings.TrimSpace(params.Email) == "" {
		api.RequestErrorHandler(w, errors.New("email cannot be empty"))
		return
	}

	if strings.TrimSpace(params.Phone) == "" {
		api.RequestErrorHandler(w, errors.New("phone cannot be empty"))
		return
	}

	if strings.TrimSpace(params.Role) == "" {
		api.RequestErrorHandler(w, errors.New("role cannot be empty"))
		return
	}

	if strings.TrimSpace(params.Password) == "" {
		api.RequestErrorHandler(w, errors.New("password cannot be empty"))
		return
	}

	fmt.Printf("Received signup request: %+v\n", params)

	var database *sqltools.DatabaseInterface
	database, err = sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	hashedPassword, err := functions.HashString(params.Password)
	if err != nil {
		log.Error("Failed to hash password:", err)
		api.InternalErrorHandler(w)
		return
	}

	newUser := models.User{
		UserID:    "userID",
		Name:      params.Name,
		Phone:     params.Phone,
		Email:     params.Email,
		Role:      params.Role,
		Password:  hashedPassword,
		CreatedAt: time.Now(),
	}

	errChan := make(chan error, 2)
	var userid string

	// Add user concurrently
	go func() {
		var errAdd error
		userid, errAdd = (*database).AddUser(newUser)
		if errAdd != nil {
			log.Error("Failed to add user: ", errAdd)
			errChan <- errors.New("user already exists or could not be added")
			return
		}
		errChan <- nil
	}()

	// Wait for both operations to complete
	for i := 0; i < 1; i++ {
		if err := <-errChan; err != nil {
			api.RequestErrorHandler(w, err)
			return
		}
	}

	var response = api.SignupResponse{
		Code:     http.StatusOK,
		Message:  "Signup successful",
		Username: userid,
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)

	if err != nil {
		log.Error("Failed to encode response: ", err)
		api.InternalErrorHandler(w)
		return
	}
}
