package models

import "time"

type Invoice struct {
	InvoiceID        string    `db:"invoice_id" json:"invoice_id"`
	RequestID        string    `db:"request_id" json:"request_id"`
	TotalValue       float64   `db:"total_value" json:"total_value"`
	OutstandingValue float64   `db:"outstanding_value" json:"outstanding_value"`
	Status           string    `db:"status" json:"status"`
	DueAt            time.Time `db:"due_at" json:"due_at"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at" json:"updated_at"`
}

type Receipt struct {
	ReceiptID  string    `db:"receipt_id" json:"receipt_id"`
	InvoiceID  string    `db:"invoice_id" json:"invoice_id"`
	AmountPaid float64   `db:"amount_paid" json:"amount_paid"`
	PaidAt     time.Time `db:"paid_at" json:"paid_at"`
}
