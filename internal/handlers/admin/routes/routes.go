package routehandlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

// SetRoutePlanHandler replaces an associate's entire route plan for a day.
func SetRoutePlanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	var params api.SetRoutePlanInput
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Error(err)
		api.RequestErrorHandler(w, err)
		return
	}

	if params.SalesAssociateID == "" {
		api.RequestErrorHandler(w, errors.New("sales_associate_id is required"))
		return
	}
	if !api.IsValidRouteDay(params.RouteDay) {
		api.RequestErrorHandler(w, errors.New("route_day must be Monday through Saturday"))
		return
	}
	if len(params.OutletIDs) == 0 {
		api.RequestErrorHandler(w, errors.New("outlet_ids cannot be empty"))
		return
	}

	// Reject duplicate outlets in the same day's plan — a stop appearing
	// twice would otherwise silently produce two sequence positions for
	// the same outlet.
	seen := make(map[string]bool, len(params.OutletIDs))
	for _, id := range params.OutletIDs {
		if seen[id] {
			api.RequestErrorHandler(w, errors.New("duplicate outlet in route plan: "+id))
			return
		}
		seen[id] = true
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	if err := (*database).SetRoutePlan(params.SalesAssociateID, params.RouteDay, params.OutletIDs); err != nil {
		log.Error("Failed to set route plan: ", err)
		api.InternalErrorHandler(w)
		return
	}

	plan, err := (*database).GetRoutePlan(params.SalesAssociateID, params.RouteDay)
	if err != nil {
		log.Error("Failed to fetch route plan after save: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(plan)
}

// GetRoutePlanHandler fetches an associate's route plan for a given day.
func GetRoutePlanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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

// ApproveRoutePlanHandler marks a day's route plan as approved, finalizing
// it for the sales associate's use.
func ApproveRoutePlanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	var params api.ApproveRoutePlanInput
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Error(err)
		api.RequestErrorHandler(w, err)
		return
	}

	if params.SalesAssociateID == "" {
		api.RequestErrorHandler(w, errors.New("sales_associate_id is required"))
		return
	}
	if !api.IsValidRouteDay(params.RouteDay) {
		api.RequestErrorHandler(w, errors.New("route_day must be Monday through Saturday"))
		return
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	approved, err := (*database).ApproveRoutePlan(params.SalesAssociateID, params.RouteDay)
	if err != nil {
		log.Error("Failed to approve route plan: ", err)
		api.InternalErrorHandler(w)
		return
	}

	if !approved {
		api.RequestErrorHandler(w, errors.New("no route plan found for this associate/day"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sales_associate_id": params.SalesAssociateID,
		"route_day":          params.RouteDay,
		"approved":           true,
	})
}
