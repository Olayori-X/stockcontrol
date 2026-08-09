package pickuphandlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql" // adjust import path
	"github.com/Olayori-X/stock-control-backend/models"                      // adjust import path
)

func CreatePickupRequestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Error("method not allowed")
		api.RequestErrorHandler(w, fmt.Errorf("method not allowed"))
		return
	}

	var params = api.CreatePickupRequestInput{}
	var err error

	err = json.NewDecoder(r.Body).Decode(&params)

	if err != nil {
		log.Error(err)
		api.RequestErrorHandler(w, err)
		return
	}

	if params.SalesAssociateID == "" {
		log.Error("sales_associate_id is required")
		api.RequestErrorHandler(w, fmt.Errorf("sales_associate_id is required"))
		return
	}
	if params.DistributorID == "" {
		log.Error("distributor_id is required")
		api.RequestErrorHandler(w, fmt.Errorf("distributor_id is required"))
		return
	}
	if len(params.Products) == 0 {
		log.Error("products cannot be empty")
		api.RequestErrorHandler(w, fmt.Errorf("products cannot be empty"))
		return
	}
	for _, p := range params.Products {
		if p.SKU == "" {
			log.Error("each product must have a sku")
			api.RequestErrorHandler(w, fmt.Errorf("each product must have a sku"))
			return
		}
		if p.Quantity <= 0 {
			log.Error("each product must have a quantity greater than 0")
			api.RequestErrorHandler(w, fmt.Errorf("each product must have a quantity greater than 0"))
			return
		}
	}

	req := &models.PickupRequest{
		RequestID:        uuid.NewString(),
		SalesAssociateID: params.SalesAssociateID,
		DistributorID:    params.DistributorID,
		Products:         params.Products,
		Confirmed:        false,
	}

	var database *sqltools.DatabaseInterface
	database, err = sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	if err = (*database).CreatePickupRequest(req); err != nil {
		log.Error("failed to create pickup request: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)
}
