package pickuphandlers

import (
	"encoding/json"
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql" // adjust import path
)

// ConfirmPickupRequestHandler — updated for the new 3-value return.
func ConfirmPickupRequestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	var params api.ConfirmPickupRequestInput
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Error(err)
		api.RequestErrorHandler(w, err)
		return
	}

	if params.RequestID == "" {
		api.RequestErrorHandler(w, errors.New("request_id is required"))
		return
	}
	if params.DistributorID == "" {
		api.RequestErrorHandler(w, errors.New("distributor_id is required"))
		return
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	confirmed, invoice, err := (*database).ConfirmPickupRequest(params.RequestID, params.DistributorID)
	if err != nil {
		log.Error("failed to confirm pickup request: ", err)
		api.InternalErrorHandler(w)
		return
	}

	if !confirmed {
		api.RequestErrorHandler(w, errors.New("pickup request not found for this distributor"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"request_id": params.RequestID,
		"confirmed":  true,
		"invoice":    invoice,
	})
}
