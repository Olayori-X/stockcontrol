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

// CreateMyOutletInput — no assigned_sales_associate_id field: the backend
// derives it from the auth header, so a sales associate can never assign
// an outlet to anyone but themselves.
type CreateMyOutletInput struct {
	Name       string  `json:"name"`
	Address    string  `json:"address"`
	OutletType string  `json:"outlet_type"`
	Phone      string  `json:"phone"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Area       string  `json:"area"`
	Zone       string  `json:"zone"`
	RouteDay   string  `json:"route_day"`
	Priority   string  `json:"priority"`
}

// CreateMyOutletHandler lets a sales associate create a new outlet while
// physically there — coordinates come from the device's live GPS, not
// manual entry or geocoding. The associate is automatically assigned as
// responsible for the outlet they create; this is never client-supplied.
func CreateMyOutletHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	salesAssociateID := r.Header.Get("userid")
	if salesAssociateID == "" {
		api.RequestErrorHandler(w, errors.New("userid header is required"))
		return
	}

	var params CreateMyOutletInput
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
		AssignedSalesAssociateID: salesAssociateID, // always self — never trusted from the client
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

	actorID := salesAssociateID
	if err := (*database).RecordAuditLog(&actorID, "outlet_created", outlet.OutletID, "Outlet created by sales associate in the field"); err != nil {
		log.Error("Failed to record audit log entry: ", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(outlet)
}
