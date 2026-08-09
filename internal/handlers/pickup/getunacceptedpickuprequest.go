package pickuphandlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql" // adjust import path
)

func GetUnacceptedPickupRequestsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Error("method not allowed")
		api.RequestErrorHandler(w, fmt.Errorf("method not allowed"))
		return
	}

	salesAssociateID := r.Header.Get("userid")
	if salesAssociateID == "" {
		log.Error("userid header is required")
		api.RequestErrorHandler(w, fmt.Errorf("userid header is required"))
		return
	}

	var database *sqltools.DatabaseInterface
	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	requests, err := (*database).GetUnacceptedPickupRequests(salesAssociateID)
	if err != nil {
		log.Error("failed to fetch unaccepted pickup requests: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(requests)
}
