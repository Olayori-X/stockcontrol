package models

import "time"

type Sale struct {
	TransactionID       string     `db:"transaction_id" json:"transaction_id"`
	SalesAssociateID    string     `db:"sales_associate_id" json:"sales_associate_id"`
	OutletID            string     `db:"outlet_id" json:"outlet_id"`
	SKU                 string     `db:"sku" json:"sku"`
	Quantity            int        `db:"quantity" json:"quantity"`
	UnitValue           float64    `db:"unit_value" json:"unit_value"`
	TotalValue          float64    `db:"total_value" json:"total_value"`
	Latitude            float64    `db:"latitude" json:"latitude"`
	Longitude           float64    `db:"longitude" json:"longitude"`
	DistanceFromOutletM float64    `db:"distance_from_outlet_m" json:"distance_from_outlet_m"`
	GeofenceStatus      string     `db:"geofence_status" json:"geofence_status"`
	RouteDay            string     `db:"route_day" json:"route_day"`
	SyncedAt            *time.Time `db:"synced_at" json:"synced_at,omitempty"`
	CreatedAt           time.Time  `db:"created_at" json:"created_at"`
}

type PlannedVsActual struct {
	SalesAssociateID string    `json:"sales_associate_id"`
	RouteDay         string    `json:"route_day"`
	Date             time.Time `json:"date"`

	OutletsPlanned   int      `json:"outlets_planned"`
	OutletsVisited   int      `json:"outlets_visited"`   // distinct outlets with at least one PASS visit
	ProductiveVisits int      `json:"productive_visits"` // distinct outlets with at least one sale
	MissedOutlets    []string `json:"missed_outlets"`    // planned outlet_ids never visited

	ResumptionTime   *string    `json:"resumption_time,omitempty"` // "" if no resumption record for the day
	ResumptionResult *string    `json:"resumption_result,omitempty"`
	LastActivity     *time.Time `json:"last_activity,omitempty"`

	SalesCalls int     `json:"sales_calls"` // total PASS visit attempts (not distinct outlets)
	SalesValue float64 `json:"sales_value"`

	CoveragePct       float64 `json:"coverage_pct"`        // visited planned / total planned
	RouteAdherencePct float64 `json:"route_adherence_pct"` // visits to planned outlets / total visits (flags off-route activity)
}
