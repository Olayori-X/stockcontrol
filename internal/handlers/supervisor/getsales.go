package reportinghandlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

// GetSalesHandler lists sales in a date range, optionally filtered by
// sales_associate_id, outlet_id, and/or sku. Also returns a summary
// (total quantity, total value) for the same filters, so the caller
// doesn't have to sum the row list itself.
func GetSalesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	salesAssociateID := r.URL.Query().Get("sales_associate_id")
	outletID := r.URL.Query().Get("outlet_id")
	sku := r.URL.Query().Get("sku")

	from, to, err := resolveDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		api.RequestErrorHandler(w, err)
		return
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	sales, err := (*database).GetSales(salesAssociateID, outletID, sku, from, to)
	if err != nil {
		log.Error("Failed to fetch sales: ", err)
		api.InternalErrorHandler(w)
		return
	}

	totalQuantity, totalValue, err := (*database).GetSalesSummary(salesAssociateID, outletID, sku, from, to)
	if err != nil {
		log.Error("Failed to compute sales summary: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sales":          sales,
		"total_quantity": totalQuantity,
		"total_value":    totalValue,
		"from":           from.Format("2006-01-02"),
		"to":             to.Format("2006-01-02"),
	})
}
