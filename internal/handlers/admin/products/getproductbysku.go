package producthandlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
)

func GetProductBySKUHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Error("method not allowed")
		api.RequestErrorHandler(w, fmt.Errorf("method not allowed"))
		return
	}

	var params = api.GetProductBySKUInput{}
	var err error

	err = json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		log.Error(err)
		api.RequestErrorHandler(w, err)
		return
	}

	if params.SKU == "" {
		log.Error("sku is required")
		api.RequestErrorHandler(w, fmt.Errorf("sku is required"))
		return
	}

	var database *sqltools.DatabaseInterface
	database, err = sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	product, err := (*database).GetProductBySKU(params.SKU)
	if err != nil {
		log.Error("failed to get product: ", err)
		api.InternalErrorHandler(w)
		return
	}
	if product == nil {
		log.Error("product not found")
		api.RequestErrorHandler(w, fmt.Errorf("product not found"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(product)
}
