package admin

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

func EditUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	var params api.EditUserInput
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Error(err)
		api.RequestErrorHandler(w, err)
		return
	}

	if params.UserID == "" {
		api.RequestErrorHandler(w, errors.New("user_id is required"))
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

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	updated, err := (*database).EditUser(params.UserID, params.Name, params.Email, params.Phone)
	if err != nil {
		log.Error("Failed to update user: ", err)
		api.InternalErrorHandler(w)
		return
	}
	if updated == nil {
		api.RequestErrorHandler(w, errors.New("user not found"))
		return
	}

	actorID := r.Header.Get("userid")
	if err := (*database).RecordAuditLog(&actorID, "user_edited", updated.UserID, "Profile details updated by admin"); err != nil {
		log.Error("Failed to record audit log entry: ", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(api.ToUserSummary(*updated))
}

// SetUserActiveHandler deactivates (default) or reactivates a user.
// Pass ?active=true to reactivate; omitted or any other value deactivates.
func SetUserActiveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		api.RequestErrorHandler(w, errors.New("user_id is required"))
		return
	}
	active := r.URL.Query().Get("active") == "true"

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	found, err := (*database).SetUserActive(userID, active)
	if err != nil {
		log.Error("Failed to update user status: ", err)
		api.InternalErrorHandler(w)
		return
	}
	if !found {
		api.RequestErrorHandler(w, errors.New("user not found"))
		return
	}

	actorID := r.Header.Get("userid")
	action := "user_deactivated"
	if active {
		action = "user_reactivated"
	}
	if err := (*database).RecordAuditLog(&actorID, action, userID, "User status changed by admin"); err != nil {
		log.Error("Failed to record audit log entry: ", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"user_id": userID, "active": active})
}
