package api

// SubmitSaleInput — TransactionID is client-generated (APK generates it
// offline, before connectivity exists) and doubles as the idempotency key.
// A retried sync with the same TransactionID is safe: it returns the
// original result rather than creating a duplicate sale.
type SubmitSaleInput struct {
	TransactionID string  `json:"transaction_id"`
	OutletID      string  `json:"outlet_id"`
	RouteDay      string  `json:"route_day"`
	SKU           string  `json:"sku"`
	Quantity      int     `json:"quantity"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
}
