package assignmenthandlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

func AssignDistributorHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	var params api.AssignDistributorInput
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Error(err)
		api.RequestErrorHandler(w, err)
		return
	}

	if params.SalesAssociateID == "" {
		api.RequestErrorHandler(w, errors.New("sales_associate_id is required"))
		return
	}
	if params.DistributorID == "" {
		api.RequestErrorHandler(w, errors.New("distributor_id is required"))
		return
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	if err := (*database).AssignDistributor(params.SalesAssociateID, params.DistributorID); err != nil {
		log.Error("Failed to assign distributor: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sales_associate_id": params.SalesAssociateID,
		"distributor_id":     params.DistributorID,
		"assigned":           true,
	})
}

func UnassignDistributorHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	salesAssociateID := r.URL.Query().Get("sales_associate_id")
	distributorID := r.URL.Query().Get("distributor_id")

	if salesAssociateID == "" {
		api.RequestErrorHandler(w, errors.New("sales_associate_id is required"))
		return
	}
	if distributorID == "" {
		api.RequestErrorHandler(w, errors.New("distributor_id is required"))
		return
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	found, err := (*database).UnassignDistributor(salesAssociateID, distributorID)
	if err != nil {
		log.Error("Failed to unassign distributor: ", err)
		api.InternalErrorHandler(w)
		return
	}
	if !found {
		api.RequestErrorHandler(w, errors.New("assignment not found"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sales_associate_id": salesAssociateID,
		"distributor_id":     distributorID,
		"assigned":           false,
	})
}

func GetAssignedDistributorsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	salesAssociateID := r.URL.Query().Get("sales_associate_id")
	if salesAssociateID == "" {
		api.RequestErrorHandler(w, errors.New("sales_associate_id is required"))
		return
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		log.Error("Failed to connect to database: ", err)
		api.InternalErrorHandler(w)
		return
	}

	assignments, err := (*database).GetAssignedDistributors(salesAssociateID)
	if err != nil {
		log.Error("Failed to fetch distributor assignments: ", err)
		api.InternalErrorHandler(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(assignments)
}
