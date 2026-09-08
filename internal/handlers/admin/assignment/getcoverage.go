package assignmenthandlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

// GetOutsideCoverageHandler returns the outside-coverage report for one
// sales associate. week_start (YYYY-MM-DD, expected to be a Monday) is
// optional — defaults to the current week if omitted.
func GetOutsideCoverageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	salesAssociateID := r.URL.Query().Get("sales_associate_id")
	if salesAssociateID == "" {
		api.RequestErrorHandler(w, errors.New("sales_associate_id is required"))
		return
	}

	weekStart, err := resolveWeekStart(r.URL.Query().Get("week_start"))
	if err != nil {
		api.RequestErrorHandler(w, err)
		return
	}
	weekEnd := weekStart.AddDate(0, 0, 7)

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	report, err := (*database).GetOutsideCoverage(salesAssociateID, weekStart, weekEnd)
	if err != nil {
		log.Error("Failed to compute outside coverage: ", err)
		api.InternalErrorHandler(w)
		return
	}

	if report.Exceeded {
		log.Warnf("Outside-coverage exception: %s at %.1f%% (threshold %.1f%%) for week of %s",
			salesAssociateID, report.OutsideCoveragePct, report.ThresholdPct, weekStart.Format("2006-01-02"))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(report)
}

// resolveWeekStart parses an optional YYYY-MM-DD date, or defaults to the
// Monday of the current week (matching the brief's Monday–Saturday route
// week) if no value is given.
func resolveWeekStart(raw string) (time.Time, error) {
	if raw == "" {
		now := time.Now()
		offset := int(now.Weekday()) - int(time.Monday)
		if offset < 0 {
			offset += 7
		}
		monday := now.AddDate(0, 0, -offset)
		return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location()), nil
	}

	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, errors.New("week_start must be in YYYY-MM-DD format")
	}
	return parsed, nil
}
