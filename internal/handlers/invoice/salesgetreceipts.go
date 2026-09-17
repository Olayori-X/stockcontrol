package invoicehandlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

func GetMyInvoicesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	salesAssociateID := r.Header.Get("userid")
	if salesAssociateID == "" {
		api.RequestErrorHandler(w, errors.New("userid header is required"))
		return
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	invoices, err := (*database).GetInvoicesForSalesAssociate(salesAssociateID)
	if err != nil {
		log.Error("Failed to fetch invoices: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(invoices)
}

func GetMyReceiptsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	salesAssociateID := r.Header.Get("userid")
	if salesAssociateID == "" {
		api.RequestErrorHandler(w, errors.New("userid header is required"))
		return
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	receipts, err := (*database).GetReceiptsForSalesAssociate(salesAssociateID)
	if err != nil {
		log.Error("Failed to fetch receipts: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(receipts)
}
