package outlethandlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

// GetMySalesHandler returns the calling sales associate's own sales
// history. sales_associate_id comes only from the authenticated userid
// header, never a query param — same scoping principle as
// GetMyRoutePlanHandler. Optional outlet_id/sku/from/to filters are still
// accepted via query params, since narrowing your own history is a
// legitimate use case an associate might want (e.g. "show my sales at
// this outlet today").
func GetMySalesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	salesAssociateID := r.Header.Get("userid")
	if salesAssociateID == "" {
		api.RequestErrorHandler(w, errors.New("userid header is required"))
		return
	}

	outletID := r.URL.Query().Get("outlet_id")
	sku := r.URL.Query().Get("sku")

	to := time.Now()
	from := to.AddDate(0, 0, -30) // default: last 30 days

	if raw := r.URL.Query().Get("from"); raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			api.RequestErrorHandler(w, errors.New("from must be in YYYY-MM-DD format"))
			return
		}
		from = parsed
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			api.RequestErrorHandler(w, errors.New("to must be in YYYY-MM-DD format"))
			return
		}
		to = parsed
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
