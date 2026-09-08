package models

import "time"

// RoutePlanStop is one outlet's position in an associate's day plan.
type RoutePlanStop struct {
	OutletID   string  `json:"outlet_id"`
	OutletName string  `json:"outlet_name"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Address    string  `json:"address"`
	Sequence   int     `json:"sequence"`
}

// RoutePlan is a sales associate's full plan for one route day.
type RoutePlan struct {
	SalesAssociateID string          `json:"sales_associate_id"`
	RouteDay         string          `json:"route_day"`
	Approved         bool            `json:"approved"`
	Stops            []RoutePlanStop `json:"stops"`
	UpdatedAt        time.Time       `json:"updated_at"`
}
