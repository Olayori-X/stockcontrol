package api

// CreateOutletInput — outlet_id is server-generated, never supplied by the client.
type CreateOutletInput struct {
	Name                     string  `json:"name"`
	Address                  string  `json:"address"`
	OutletType               string  `json:"outlet_type"`
	Phone                    string  `json:"phone"`
	Latitude                 float64 `json:"latitude"`
	Longitude                float64 `json:"longitude"`
	Area                     string  `json:"area"`
	Zone                     string  `json:"zone"`
	AssignedSalesAssociateID string  `json:"assigned_sales_associate_id"`
	RouteDay                 string  `json:"route_day"`
	Priority                 string  `json:"priority"`
}

type EditOutletInput struct {
	OutletID                 string  `json:"outlet_id"`
	Name                     string  `json:"name"`
	Address                  string  `json:"address"`
	OutletType               string  `json:"outlet_type"`
	Phone                    string  `json:"phone"`
	Latitude                 float64 `json:"latitude"`
	Longitude                float64 `json:"longitude"`
	Area                     string  `json:"area"`
	Zone                     string  `json:"zone"`
	AssignedSalesAssociateID string  `json:"assigned_sales_associate_id"`
	RouteDay                 string  `json:"route_day"`
	Priority                 string  `json:"priority"`
}

func IsValidLatLng(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

type ConfirmOutletVisitInput struct {
	OutletID  string  `json:"outlet_id"`
	RouteDay  string  `json:"route_day"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
