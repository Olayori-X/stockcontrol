package outlethandlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Olayori-X/stock-control-backend/api"
	"github.com/Olayori-X/stock-control-backend/functions"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

// ConfirmOutletVisitHandler checks whether the sales associate is within
// the configured geofence radius of the given outlet. The attempt is
// recorded either way; the response tells the caller whether they may
// proceed to sales capture for this outlet.
func ConfirmOutletVisitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	salesAssociateID := r.Header.Get("userid")
	if salesAssociateID == "" {
		api.RequestErrorHandler(w, errors.New("userid header is required"))
		return
	}

	var params api.ConfirmOutletVisitInput
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Error(err)
		api.RequestErrorHandler(w, err)
		return
	}

	if params.OutletID == "" {
		api.RequestErrorHandler(w, errors.New("outlet_id is required"))
		return
	}
	if !api.IsValidRouteDay(params.RouteDay) {
		api.RequestErrorHandler(w, errors.New("route_day must be Monday through Saturday"))
		return
	}
	if !api.IsValidLatLng(params.Latitude, params.Longitude) {
		api.RequestErrorHandler(w, errors.New("latitude/longitude are out of range"))
		return
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	outlet, err := (*database).GetOutletByID(params.OutletID)
	if err != nil {
		log.Error("Failed to fetch outlet: ", err)
		api.InternalErrorHandler(w)
		return
	}
	if outlet == nil || !outlet.Active {
		api.RequestErrorHandler(w, errors.New("outlet not found"))
		return
	}

	radiusM, err := (*database).GetSettingFloat("outlet_geofence_m")
	if err != nil {
		log.Error("Failed to read outlet_geofence_m setting: ", err)
		api.InternalErrorHandler(w)
		return
	}

	distance := functions.HaversineMeters(params.Latitude, params.Longitude, outlet.Latitude, outlet.Longitude)

	status := "FAIL"
	if distance <= radiusM {
		status = "PASS"
	}

	if err := (*database).RecordOutletVisit(
		salesAssociateID, params.OutletID, params.RouteDay,
		params.Latitude, params.Longitude, distance, status,
	); err != nil {
		log.Error("Failed to record outlet visit: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"outlet_id":       params.OutletID,
		"distance_m":      distance,
		"radius_m":        radiusM,
		"geofence_status": status,
	})

	if status == "FAIL" {
		log.Warnf("Outlet geofence FAIL for %s at outlet %s: %.0fm (radius %.0fm)",
			salesAssociateID, params.OutletID, distance, radiusM)
	}
}
