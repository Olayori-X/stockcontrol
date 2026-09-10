package audithandlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

func GetAuditLogHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	actorID := r.URL.Query().Get("actor_id")

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

	entries, err := (*database).GetAuditLog(actorID, from, to)
	if err != nil {
		log.Error("Failed to fetch audit log: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(entries)
}
