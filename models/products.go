package models

import "time"

type Products struct {
	SKU       string    `db:"sku" json:"sku"`
	Name      string    `db:"name" json:"name"`
	Price     float64   `db:"price" json:"price"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
