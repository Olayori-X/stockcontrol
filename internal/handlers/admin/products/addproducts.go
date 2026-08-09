package producthandlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	"github.com/Olayori-X/stock-control-backend/models"
)

func AddProductHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Error("method not allowed")
		api.RequestErrorHandler(w, fmt.Errorf("method not allowed"))
		return
	}

	var params = api.AddProductInput{}
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
	if params.Name == "" {
		log.Error("name is required")
		api.RequestErrorHandler(w, fmt.Errorf("name is required"))
		return
	}

	product := models.Products{
		SKU:  params.SKU,
		Name: params.Name,
	}

	var database *sqltools.DatabaseInterface
	database, err = sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	if err = (*database).AddProduct(product); err != nil {
		log.Error("failed to add product: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}
