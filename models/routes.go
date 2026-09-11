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

type RouteEfficiency struct {
	SalesAssociateID string `json:"sales_associate_id"`
	RouteDay         string `json:"route_day"`

	OutletCount int `json:"outlet_count"`

	PlannedDistanceM    float64 `json:"planned_distance_m"`
	AverageLegDistanceM float64 `json:"average_leg_distance_m"`

	CentroidLatitude  float64 `json:"centroid_latitude"`
	CentroidLongitude float64 `json:"centroid_longitude"`
	DispersionM       float64 `json:"dispersion_m"`

	Outliers []string `json:"outliers"`

	BacktrackCount int      `json:"backtrack_count"`
	BacktrackLegs  []string `json:"backtrack_legs"`

	DominantArea     string  `json:"dominant_area"`
	AreaAlignmentPct float64 `json:"area_alignment_pct"`
	DominantZone     string  `json:"dominant_zone"`
	ZoneAlignmentPct float64 `json:"zone_alignment_pct"`
}
