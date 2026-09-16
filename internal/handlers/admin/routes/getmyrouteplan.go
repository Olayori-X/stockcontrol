package routehandlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

// GetMyRoutePlanHandler returns the calling sales associate's own route
// plan for today. sales_associate_id comes only from the authenticated
// userid header — never a query param — so an associate can never
// request or accidentally trigger a lookup of someone else's route.
// Reuses the exact same GetRoutePlan sqltools function GetRoutePlanHandler
// already calls; this is purely a scoping wrapper.
func GetMyRoutePlanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	salesAssociateID := r.Header.Get("userid")
	if salesAssociateID == "" {
		api.RequestErrorHandler(w, errors.New("userid header is required"))
		return
	}

	routeDay := time.Now().Weekday().String()

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	plan, err := (*database).GetRoutePlan(salesAssociateID, routeDay)
	if err != nil {
		log.Error("Failed to fetch route plan: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(plan)
}
