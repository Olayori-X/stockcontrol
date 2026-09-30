package outlethandlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

func DeactivateMyOutletHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	salesAssociateID := r.Header.Get("userid")
	if salesAssociateID == "" {
		api.RequestErrorHandler(w, errors.New("userid header is required"))
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
	if outlet.AssignedSalesAssociateID != salesAssociateID {
		api.RequestErrorHandler(w, errors.New("you can only remove outlets you created"))
		return
	}

	found, err := (*database).SetOutletActive(outletID, false)
	if err != nil {
		log.Error("Failed to deactivate outlet: ", err)
		api.InternalErrorHandler(w)
		return
	}
	if !found {
		api.RequestErrorHandler(w, errors.New("outlet not found"))
		return
	}

	actorID := salesAssociateID
	if err := (*database).RecordAuditLog(&actorID, "outlet_deactivated", outletID, "Outlet removed by the sales associate who created it"); err != nil {
		log.Error("Failed to record audit log entry: ", err)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"outlet_id": outletID, "active": false})
}
