package invoicehandlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

// GetReceiptsForDistributorHandler — /distributor/receipts.
// The caller IS the distributor; userid IS the distributor_id.
func GetReceiptsForDistributorHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	distributorID := r.Header.Get("userid")

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	receipts, err := (*database).GetReceipts(distributorID)
	if err != nil {
		log.Error("Failed to fetch receipts: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(receipts)
}
