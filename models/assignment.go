package models

import "time"

type DistributorAssignment struct {
	SalesAssociateID string    `db:"sales_associate_id" json:"sales_associate_id"`
	DistributorID    string    `db:"distributor_id" json:"distributor_id"`
	DistributorName  string    `db:"distributor_name" json:"distributor_name"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

type OutsideCoverageReport struct {
	SalesAssociateID     string    `json:"sales_associate_id"`
	WeekStart            time.Time `json:"week_start"`
	WeekEnd              time.Time `json:"week_end"`
	TotalPickupValue     float64   `json:"total_pickup_value"`
	OutsideCoverageValue float64   `json:"outside_coverage_value"`
	OutsideCoveragePct   float64   `json:"outside_coverage_pct"`
	ThresholdPct         float64   `json:"threshold_pct"`
	Exceeded             bool      `json:"exceeded"`
}

type ResumptionLog struct {
	SalesAssociateID string    `json:"sales_associate_id"`
	RouteDay         string    `json:"route_day"`
	Date             time.Time `json:"date"`
	Time             time.Time `json:"time"`
	Latitude         float64   `json:"latitude"`
	Longitude        float64   `json:"longitude"`
	DistanceToRouteM float64   `json:"distance_to_route_m"`
	Result           string    `json:"result"`
	DeviceRef        string    `json:"device_ref"`
	CreatedAt        time.Time `json:"created_at"`
}
