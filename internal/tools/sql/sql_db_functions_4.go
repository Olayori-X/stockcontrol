package sqltools

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Olayori-X/stock-control-backend/models"
)

// ============================================================================
// AUDIT LOG
// ============================================================================

// RecordAuditLog writes one audit entry. actorID is nil for system-
// initiated actions (e.g. the overdue-invoice scheduler) — every other
// caller should pass the acting user's ID.
func (db *RealDB) RecordAuditLog(actorID *string, action, target, details string) error {
	_, err := db.DB.Exec(`
		INSERT INTO audit_log (actor_id, action, target, details)
		VALUES ($1, $2, $3, $4);
	`, nullableStringPtr(actorID), action, target, details)
	if err != nil {
		return fmt.Errorf("could not record audit log entry: %w", err)
	}
	return nil
}

// nullableStringPtr converts a *string to a driver-friendly NULL when nil.
func nullableStringPtr(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

// GetAuditLog lists entries in a date range, optionally filtered to one
// actor. actorID == "" means "all actors" (including system entries).
func (db *RealDB) GetAuditLog(actorID string, from, to time.Time) ([]models.AuditLogEntry, error) {
	query := `
		SELECT actor_id, action, COALESCE(target, ''), COALESCE(details, ''), created_at
		FROM audit_log
		WHERE created_at >= $1 AND created_at < $2
	`
	args := []interface{}{from, to}
	if actorID != "" {
		query += ` AND actor_id = $3`
		args = append(args, actorID)
	}
	query += ` ORDER BY created_at DESC;`

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not fetch audit log: %w", err)
	}
	defer rows.Close()

	entries := make([]models.AuditLogEntry, 0)
	for rows.Next() {
		var e models.AuditLogEntry
		var actor sql.NullString
		if err := rows.Scan(&actor, &e.Action, &e.Target, &e.Details, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("could not scan audit log row: %w", err)
		}
		if actor.Valid {
			e.ActorID = &actor.String
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return entries, nil
}

// ============================================================================
// SALES — READ (for Manager/Admin sales review)
// ============================================================================

// GetSales lists sales in a date range, optionally filtered by sales
// associate, outlet, and/or SKU. Any filter left empty is skipped —
// callers can combine filters freely (e.g. one associate across all
// outlets, or one outlet across all associates).
func (db *RealDB) GetSales(salesAssociateID, outletID, sku string, from, to time.Time) ([]models.Sale, error) {
	query := `
		SELECT transaction_id, sales_associate_id, outlet_id, sku, quantity,
		       unit_value, total_value, latitude, longitude,
		       distance_from_outlet_m, geofence_status, route_day, synced_at, created_at
		FROM sales
		WHERE created_at >= $1 AND created_at < $2
	`
	args := []interface{}{from, to}
	argN := 3

	if salesAssociateID != "" {
		query += fmt.Sprintf(" AND sales_associate_id = $%d", argN)
		args = append(args, salesAssociateID)
		argN++
	}
	if outletID != "" {
		query += fmt.Sprintf(" AND outlet_id = $%d", argN)
		args = append(args, outletID)
		argN++
	}
	if sku != "" {
		query += fmt.Sprintf(" AND sku = $%d", argN)
		args = append(args, sku)
		argN++
	}

	query += " ORDER BY created_at DESC;"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not fetch sales: %w", err)
	}
	defer rows.Close()

	sales := make([]models.Sale, 0)
	for rows.Next() {
		var s models.Sale
		if err := rows.Scan(
			&s.TransactionID, &s.SalesAssociateID, &s.OutletID, &s.SKU, &s.Quantity,
			&s.UnitValue, &s.TotalValue, &s.Latitude, &s.Longitude,
			&s.DistanceFromOutletM, &s.GeofenceStatus, &s.RouteDay, &s.SyncedAt, &s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("could not scan sale row: %w", err)
		}
		sales = append(sales, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return sales, nil
}

// GetSalesSummary aggregates total quantity and value for the same filter
// set as GetSales — powers a quick "total sales this week" figure without
// the caller needing to sum the full row list client-side.
func (db *RealDB) GetSalesSummary(salesAssociateID, outletID, sku string, from, to time.Time) (totalQuantity int, totalValue float64, err error) {
	query := `
		SELECT COALESCE(SUM(quantity), 0), COALESCE(SUM(total_value), 0)
		FROM sales
		WHERE created_at >= $1 AND created_at < $2
	`
	args := []interface{}{from, to}
	argN := 3

	if salesAssociateID != "" {
		query += fmt.Sprintf(" AND sales_associate_id = $%d", argN)
		args = append(args, salesAssociateID)
		argN++
	}
	if outletID != "" {
		query += fmt.Sprintf(" AND outlet_id = $%d", argN)
		args = append(args, outletID)
		argN++
	}
	if sku != "" {
		query += fmt.Sprintf(" AND sku = $%d", argN)
		args = append(args, sku)
		argN++
	}

	err = db.DB.QueryRow(query, args...).Scan(&totalQuantity, &totalValue)
	if err != nil {
		return 0, 0, fmt.Errorf("could not compute sales summary: %w", err)
	}
	return totalQuantity, totalValue, nil
}
