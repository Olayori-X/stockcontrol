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

// ============================================================================
// RESUMPTION LOGS
// ============================================================================

// GetResumptionLogsHandler lists daily resumption attempts (PASS/FAIL) in a
// date range, optionally filtered to one sales associate. Powers the
// Manager/Admin "resumption compliance" review.
func GetResumptionLogsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	salesAssociateID := r.URL.Query().Get("sales_associate_id")

	from, to, err := resolveDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		api.RequestErrorHandler(w, err)
		return
	}

	var database *sqltools.DatabaseInterface
	database, err = sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	logs, err := (*database).GetResumptionLogs(salesAssociateID, from, to)
	if err != nil {
		log.Error("Failed to fetch resumption logs: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(logs); err != nil {
		log.Error("Failed to encode response: ", err)
		api.InternalErrorHandler(w)
		return
	}
}

// ============================================================================
// OUTLET VISITS
// ============================================================================

// GetOutletVisitsHandler lists outlet visit attempts (PASS/FAIL geofence
// events) in a date range, optionally filtered to one sales associate.
func GetOutletVisitsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	salesAssociateID := r.URL.Query().Get("sales_associate_id")

	from, to, err := resolveDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		api.RequestErrorHandler(w, err)
		return
	}

	var database *sqltools.DatabaseInterface
	database, err = sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	visits, err := (*database).GetOutletVisits(salesAssociateID, from, to)
	if err != nil {
		log.Error("Failed to fetch outlet visits: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(visits); err != nil {
		log.Error("Failed to encode response: ", err)
		api.InternalErrorHandler(w)
		return
	}
}

// ============================================================================
// DATE HELPERS
// ============================================================================

// resolveDateRange parses optional YYYY-MM-DD from/to bounds, defaulting to
// the last 7 days if either is omitted.
func resolveDateRange(fromRaw, toRaw string) (time.Time, time.Time, error) {
	to := time.Now()
	from := to.AddDate(0, 0, -7)

	if fromRaw != "" {
		parsed, err := time.Parse("2006-01-02", fromRaw)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("from must be in YYYY-MM-DD format")
		}
		from = parsed
	}
	if toRaw != "" {
		parsed, err := time.Parse("2006-01-02", toRaw)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("to must be in YYYY-MM-DD format")
		}
		to = parsed
	}

	return from, to, nil
}
