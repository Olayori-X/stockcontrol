package outlethandlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

func SubmitSaleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	salesAssociateID := r.Header.Get("userid")
	if salesAssociateID == "" {
		api.RequestErrorHandler(w, errors.New("userid header is required"))
		return
	}

	var params api.SubmitSaleInput
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Error(err)
		api.RequestErrorHandler(w, err)
		return
	}

	if params.TransactionID == "" {
		api.RequestErrorHandler(w, errors.New("transaction_id is required"))
		return
	}
	if params.OutletID == "" {
		api.RequestErrorHandler(w, errors.New("outlet_id is required"))
		return
	}
	if !api.IsValidRouteDay(params.RouteDay) {
		api.RequestErrorHandler(w, errors.New("route_day must be Monday through Saturday"))
		return
	}
	if params.SKU == "" {
		api.RequestErrorHandler(w, errors.New("sku is required"))
		return
	}
	if params.Quantity <= 0 {
		api.RequestErrorHandler(w, errors.New("quantity must be greater than 0"))
		return
	}
	if !api.IsValidLatLng(params.Latitude, params.Longitude) {
		api.RequestErrorHandler(w, errors.New("latitude/longitude are out of range"))
		return
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	sale, alreadyExisted, blocked, err := (*database).SubmitSale(salesAssociateID, &params)
	if err != nil {
		log.Error("Failed to submit sale: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if blocked {
		// Geofence FAIL — sale was not created, matching the brief's
		// "normal on-site sale is blocked" rule. See the geofence-check
		// endpoint's design note: this is a result, not a client error,
		// so it stays 200 with the outcome in the body.
		log.Warnf("Sale blocked by geofence for %s at outlet %s (transaction %s)",
			salesAssociateID, params.OutletID, params.TransactionID)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"transaction_id":  params.TransactionID,
			"outlet_id":       params.OutletID,
			"geofence_status": "FAIL",
			"sale_recorded":   false,
			"message":         "You are too far from this outlet to submit a sale.",
		})
		return
	}

	status := http.StatusCreated
	if alreadyExisted {
		// Idempotent retry — same transaction_id already succeeded before.
		// 200, not 201: nothing new was created this time.
		status = http.StatusOK
	}

	w.WriteHeader(status)
	json.NewEncoder(w).Encode(sale)
}
