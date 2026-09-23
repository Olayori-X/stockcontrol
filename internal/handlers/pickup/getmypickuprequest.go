package pickuphandlers

import (
	"encoding/json"
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql" // adjust import path
)

func GetMyPickupRequestsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}
	salesAssociateID := r.Header.Get("userid")
	if salesAssociateID == "" {
		api.RequestErrorHandler(w, errors.New("userid header is required"))
		return
	}
	database, err := sqltools.NewDatabase()
	if err != nil {
		api.InternalErrorHandler(w)
		return
	}
	requests, err := (*database).GetMyPickupRequests(salesAssociateID)
	if err != nil {
		log.Error("Failed to fetch pickup requests: ", err)
		api.InternalErrorHandler(w)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(requests)
}
