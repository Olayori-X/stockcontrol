package models

import "time"

type DistributorAssignment struct {
	SalesAssociateID string    `db:"sales_associate_id" json:"sales_associate_id"`
	DistributorID    string    `db:"distributor_id" json:"distributor_id"`
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
