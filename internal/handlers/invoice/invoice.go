package invoicehandlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

func RecordPaymentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	var params api.RecordPaymentInput
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Error(err)
		api.RequestErrorHandler(w, err)
		return
	}

	if params.InvoiceID == "" {
		api.RequestErrorHandler(w, errors.New("invoice_id is required"))
		return
	}
	if params.Amount <= 0 {
		api.RequestErrorHandler(w, errors.New("amount must be greater than 0"))
		return
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	receipt, invoice, err := (*database).RecordPayment(params.InvoiceID, params.Amount)
	if err != nil {
		log.Error("Failed to record payment: ", err)
		api.InternalErrorHandler(w)
		return
	}
	if receipt == nil {
		api.RequestErrorHandler(w, errors.New("invoice not found"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"receipt": receipt,
		"invoice": invoice,
	})
}

// GetOutstandingInvoicesForDistributorHandler — for /distributor/outstanding.
// The caller IS the distributor, so userid IS the distributor_id. No
// guessing, no query params needed.
func GetOutstandingInvoicesForDistributorHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	distributorID := r.Header.Get("userid")

	respondWithOutstandingInvoices(w, distributorID)
}

// GetOutstandingInvoicesForSupervisorHandler — for /supervisor (and
// /admin, if you wire it there too). The caller is NOT a distributor, so
// their userid header is meaningless here. They must explicitly say which
// distributor they want, or explicitly ask for everyone.
func GetOutstandingInvoicesForSupervisorHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	distributorID := r.URL.Query().Get("distributor_id")
	seeAll := r.URL.Query().Get("all") == "true"

	if distributorID == "" && !seeAll {
		api.RequestErrorHandler(w, errors.New("pass distributor_id=<id> to see one distributor, or all=true to see every distributor"))
		return
	}

	respondWithOutstandingInvoices(w, distributorID)
}

// respondWithOutstandingInvoices is the shared logic both handlers above
// call once they've each figured out the correct distributorID (or "" for
// "everyone", in the supervisor case).
func respondWithOutstandingInvoices(w http.ResponseWriter, distributorID string) {
	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	invoices, err := (*database).GetOutstandingInvoices(distributorID)
	if err != nil {
		log.Error("Failed to fetch outstanding invoices: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(invoices)
}
