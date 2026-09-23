package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Olayori-X/stock-control-backend/api"
	sqltools "github.com/Olayori-X/stock-control-backend/internal/tools/sql"
	log "github.com/sirupsen/logrus"
)

var allowedIntegrationKeys = map[string]bool{
	"PICKUP_REQUESTS_SPREADSHEET_ID": true,
	"SALES_SPREADSHEET_ID":           true,
}

func SetIntegrationSettingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	var params api.SetIntegrationSettingInput
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		api.RequestErrorHandler(w, err)
		return
	}
	if !allowedIntegrationKeys[params.Key] {
		api.RequestErrorHandler(w, errors.New("unknown setting key"))
		return
	}
	if strings.TrimSpace(params.Value) == "" {
		api.RequestErrorHandler(w, errors.New("value is required"))
		return
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		api.InternalErrorHandler(w)
		return
	}

	if err := (*database).SetIntegrationSetting(params.Key, params.Value); err != nil {
		log.Error("Failed to save integration setting: ", err)
		api.InternalErrorHandler(w)
		return
	}

	actorID := r.Header.Get("userid")
	(*database).RecordAuditLog(&actorID, "integration_setting_updated", params.Key, "Spreadsheet ID updated by admin")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"key": params.Key, "status": "saved"})
}

// GetIntegrationSettingsHandler returns which keys are configured, masked
// (last 4 chars only) — never the full decrypted value over the wire,
// even to an admin, since this is displayed in a UI that could be
// screen-shared/logged.
func GetIntegrationSettingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.RequestErrorHandler(w, errors.New("method not allowed"))
		return
	}

	database, err := sqltools.NewDatabase()
	if err != nil {
		api.InternalErrorHandler(w)
		return
	}

	result := map[string]string{}
	for key := range allowedIntegrationKeys {
		value, err := (*database).GetIntegrationSetting(key)
		if err != nil || value == "" {
			result[key] = ""
			continue
		}
		masked := value
		if len(value) > 4 {
			masked = "••••" + value[len(value)-4:]
		}
		result[key] = masked
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}
