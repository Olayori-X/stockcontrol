package reportinghandlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

func GetPlannedVsActualHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	salesAssociateID := r.URL.Query().Get("sales_associate_id")
	if salesAssociateID == "" {
		api.RequestErrorHandler(w, errors.New("sales_associate_id is required"))
		return
	}

	dateRaw := r.URL.Query().Get("date")
	date := time.Now()
	if dateRaw != "" {
		parsed, err := time.Parse("2006-01-02", dateRaw)
		if err != nil {
			api.RequestErrorHandler(w, errors.New("date must be in YYYY-MM-DD format"))
			return
		}
		date = parsed
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	report, err := (*database).GetPlannedVsActual(salesAssociateID, date)
	if err != nil {
		log.Error("Failed to compute planned vs actual: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(report)
}
