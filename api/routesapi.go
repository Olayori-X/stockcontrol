package api

var validRouteDays = map[string]bool{
	"Monday": true, "Tuesday": true, "Wednesday": true,
	"Thursday": true, "Friday": true, "Saturday": true,
}

func IsValidRouteDay(day string) bool {
	return validRouteDays[day]
}

// SetRoutePlanInput replaces the entire day's plan for a sales associate.
// OutletIDs order defines the visit sequence (1-indexed).
type SetRoutePlanInput struct {
	SalesAssociateID string   `json:"sales_associate_id"`
	RouteDay         string   `json:"route_day"`
	OutletIDs        []string `json:"outlet_ids"`
}

type ApproveRoutePlanInput struct {
	SalesAssociateID string `json:"sales_associate_id"`
	RouteDay         string `json:"route_day"`
}
