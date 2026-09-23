package routehandlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

func DeleteRoutePlanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	salesAssociateID := r.URL.Query().Get("sales_associate_id")
	routeDay := r.URL.Query().Get("route_day")
	if salesAssociateID == "" {
		api.RequestErrorHandler(w, errors.New("sales_associate_id is required"))
		return
	}
	if !api.IsValidRouteDay(routeDay) {
		api.RequestErrorHandler(w, errors.New("route_day must be Monday through Saturday"))
		return
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		api.InternalErrorHandler(w)
		return
	}

	found, err := (*database).DeleteRoutePlan(salesAssociateID, routeDay)
	if err != nil {
		log.Error("Failed to delete route plan: ", err)
		api.InternalErrorHandler(w)
		return
	}
	if !found {
		api.RequestErrorHandler(w, errors.New("no route plan found for this associate/day"))
		return
	}

	actorID := r.Header.Get("userid")
	(*database).RecordAuditLog(&actorID, "route_plan_deleted", salesAssociateID+"/"+routeDay, "Route plan deleted by admin")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"sales_associate_id": salesAssociateID, "route_day": routeDay, "deleted": true})
}
