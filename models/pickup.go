package models

import "time"

type PickupRequest struct {
	RequestID        string        `db:"request_id" json:"request_id"`
	SalesAssociateID string        `db:"sales_associate_id" json:"sales_associate_id"`
	DistributorID    string        `db:"distributor_id" json:"distributor_id"`
	Products         []ProductItem `db:"-" json:"products"`
	Confirmed        bool          `db:"confirmed" json:"confirmed"`
	CreatedAt        time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time     `db:"updated_at" json:"updated_at"`
}

type ProductItem struct {
	SKU      string `db:"sku" json:"sku"`
	Name     string `db:"name" json:"name"`
	Quantity int    `db:"quantity" json:"quantity"`
}

type PendingPickupRequest struct {
	PickupRequest
	SalesAssociateName string `json:"sales_associate_name"`
}
