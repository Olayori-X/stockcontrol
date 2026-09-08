package models

import "time"

type Outlet struct {
	OutletID                 string    `db:"outlet_id" json:"outlet_id"`
	Name                     string    `db:"name" json:"name"`
	Address                  string    `db:"address" json:"address"`
	OutletType               string    `db:"outlet_type" json:"outlet_type"`
	Phone                    string    `db:"phone" json:"phone"`
	Latitude                 float64   `db:"latitude" json:"latitude"`
	Longitude                float64   `db:"longitude" json:"longitude"`
	Area                     string    `db:"area" json:"area"`
	Zone                     string    `db:"zone" json:"zone"`
	AssignedSalesAssociateID string    `db:"assigned_sales_associate_id" json:"assigned_sales_associate_id"`
	RouteDay                 string    `db:"route_day" json:"route_day"`
	Priority                 string    `db:"priority" json:"priority"`
	Active                   bool      `db:"active" json:"active"`
	CreatedAt                time.Time `db:"created_at" json:"created_at"`
	UpdatedAt                time.Time `db:"updated_at" json:"updated_at"`
}
