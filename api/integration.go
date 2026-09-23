package api

type SetIntegrationSettingInput struct {
	Key   string `json:"key"` // "PICKUP_REQUESTS_SPREADSHEET_ID" | "SALES_SPREADSHEET_ID"
	Value string `json:"value"`
}
