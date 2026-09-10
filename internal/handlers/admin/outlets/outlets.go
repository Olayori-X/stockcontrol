package outlethandlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	"github.com/Olayori-X/stock-control-backend/models"
	log "github.com/sirupsen/logrus"
)

func AddOutletHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	var params api.CreateOutletInput
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Error(err)
		api.RequestErrorHandler(w, err)
		return
	}

	if params.Name == "" {
		api.RequestErrorHandler(w, errors.New("name is required"))
		return
	}
	if !api.IsValidLatLng(params.Latitude, params.Longitude) {
		api.RequestErrorHandler(w, errors.New("latitude/longitude are out of range"))
		return
	}
	if params.RouteDay != "" && !api.IsValidRouteDay(params.RouteDay) {
		api.RequestErrorHandler(w, errors.New("route_day must be Monday through Saturday"))
		return
	}

	outlet := &models.Outlet{
		Name:                     params.Name,
		Address:                  params.Address,
		OutletType:               params.OutletType,
		Phone:                    params.Phone,
		Latitude:                 params.Latitude,
		Longitude:                params.Longitude,
		Area:                     params.Area,
		Zone:                     params.Zone,
		AssignedSalesAssociateID: params.AssignedSalesAssociateID,
		RouteDay:                 params.RouteDay,
		Priority:                 params.Priority,
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	if err := (*database).AddOutlet(outlet); err != nil {
		log.Error("Failed to create outlet: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(outlet)
}

func GetOutletsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	includeInactive := r.URL.Query().Get("include_inactive") == "true"

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	outlets, err := (*database).GetOutlets(includeInactive)
	if err != nil {
		log.Error("Failed to fetch outlets: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(outlets)
}

func GetOutletByIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	outletID := r.URL.Query().Get("outlet_id")
	if outletID == "" {
		api.RequestErrorHandler(w, errors.New("outlet_id is required"))
		return
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	outlet, err := (*database).GetOutletByID(outletID)
	if err != nil {
		log.Error("Failed to fetch outlet: ", err)
		api.InternalErrorHandler(w)
		return
	}
	if outlet == nil {
		api.RequestErrorHandler(w, errors.New("outlet not found"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(outlet)
}

func EditOutletHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	var params api.EditOutletInput
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Error(err)
		api.RequestErrorHandler(w, err)
		return
	}

	if params.OutletID == "" {
		api.RequestErrorHandler(w, errors.New("outlet_id is required"))
		return
	}
	if params.Name == "" {
		api.RequestErrorHandler(w, errors.New("name is required"))
		return
	}
	if !api.IsValidLatLng(params.Latitude, params.Longitude) {
		api.RequestErrorHandler(w, errors.New("latitude/longitude are out of range"))
		return
	}
	if params.RouteDay != "" && !api.IsValidRouteDay(params.RouteDay) {
		api.RequestErrorHandler(w, errors.New("route_day must be Monday through Saturday"))
		return
	}

	outlet := &models.Outlet{
		OutletID:                 params.OutletID,
		Name:                     params.Name,
		Address:                  params.Address,
		OutletType:               params.OutletType,
		Phone:                    params.Phone,
		Latitude:                 params.Latitude,
		Longitude:                params.Longitude,
		Area:                     params.Area,
		Zone:                     params.Zone,
		AssignedSalesAssociateID: params.AssignedSalesAssociateID,
		RouteDay:                 params.RouteDay,
		Priority:                 params.Priority,
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	updated, err := (*database).EditOutlet(outlet)
	if err != nil {
		log.Error("Failed to update outlet: ", err)
		api.InternalErrorHandler(w)
		return
	}
	if updated == nil {
		api.RequestErrorHandler(w, errors.New("outlet not found"))
		return
	}

	actorID := r.Header.Get("userid")
	(*database).RecordAuditLog(&actorID, "outlet_edited", updated.OutletID, "Outlet details updated by admin")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updated)
}

// SetOutletActiveHandler deactivates (default) or reactivates an outlet.
// Pass ?active=true to reactivate; omitted or any other value deactivates.
func SetOutletActiveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	outletID := r.URL.Query().Get("outlet_id")
	if outletID == "" {
		api.RequestErrorHandler(w, errors.New("outlet_id is required"))
		return
	}
	active := r.URL.Query().Get("active") == "true"

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	found, err := (*database).SetOutletActive(outletID, active)
	if err != nil {
		log.Error("Failed to update outlet status: ", err)
		api.InternalErrorHandler(w)
		return
	}
	if !found {
		api.RequestErrorHandler(w, errors.New("outlet not found"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"outlet_id": outletID,
		"active":    active,
	})
}
