package outlethandlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

func GetMyOutletsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	salesAssociateID := r.Header.Get("userid")
	if salesAssociateID == "" {
		log.Error("Failed to fetch outlets: ", errors.New("userid header is required"))
		api.RequestErrorHandler(w, errors.New("userid header is required"))
		return
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Internal error: ", err)
		api.InternalErrorHandler(w)
		return
	}

	outlets, err := (*database).GetOutlets(false, salesAssociateID)
	if err != nil {
		log.Error("Failed to fetch outlets: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Error("Failed to fetch outlets: ", err)
	log.Infof("Fetched %d outlets for sales associate %s (includeInactive=%v)", len(outlets), salesAssociateID, false)
	json.NewEncoder(w).Encode(outlets)
}
