package reportinghandlers

import (
	"encoding/json"
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
)

func GetRouteEfficiencyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	salesAssociateID := r.URL.Query().Get("sales_associate_id")
	if salesAssociateID == "" {
		api.RequestErrorHandler(w, errors.New("sales_associate_id is required"))
		return
	}

	routeDay := r.URL.Query().Get("route_day")
	if !api.IsValidRouteDay(routeDay) {
		api.RequestErrorHandler(w, errors.New("route_day must be Monday through Saturday"))
		return
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	report, err := (*database).GetRouteEfficiency(salesAssociateID, routeDay)
	if err != nil {
		log.Error("Failed to compute route efficiency: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(report)
}
