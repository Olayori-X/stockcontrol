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

func GetProductsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Error("method not allowed")
		api.RequestErrorHandler(w, fmt.Errorf("method not allowed"))
		return
	}

	var err error

	var database *sqltools.DatabaseInterface
	database, err = sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	products, err := (*database).GetProducts()
	if err != nil {
		log.Error("failed to get products: ", err)
		api.InternalErrorHandler(w)
		return
	}

	if products == nil {
		products = []models.Products{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(products)
}
