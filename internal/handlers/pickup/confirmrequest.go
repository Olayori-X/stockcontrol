package pickuphandlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql" // adjust import path
)

func ConfirmPickupRequestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Error("method not allowed")
		api.RequestErrorHandler(w, fmt.Errorf("method not allowed"))
		return
	}

	var params = api.ConfirmPickupRequestInput{}
	var err error

	err = json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		log.Error(err)
		api.RequestErrorHandler(w, err)
		return
	}

	if params.RequestID == "" {
		log.Error("request_id is required")
		api.RequestErrorHandler(w, fmt.Errorf("request_id is required"))
		return
	}
	if params.DistributorID == "" {
		log.Error("distributor_id is required")
		api.RequestErrorHandler(w, fmt.Errorf("distributor_id is required"))
		return
	}

	var database *sqltools.DatabaseInterface
	database, err = sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	confirmed, err := (*database).ConfirmPickupRequest(params.RequestID, params.DistributorID)
	if err != nil {
		log.Error("failed to confirm pickup request: ", err)
		api.InternalErrorHandler(w)
		return
	}

	if !confirmed {
		log.Error("pickup request not found or distributor mismatch: ", params.RequestID)
		api.RequestErrorHandler(w, fmt.Errorf("pickup request not found for this distributor"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"request_id": params.RequestID,
		"confirmed":  true,
	})
}
