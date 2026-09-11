package sqltools

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/Olayori-X/stock-control-backend/functions"
	"github.com/Olayori-X/stock-control-backend/models"
)

func (db *RealDB) createInvoiceForPickup(tx *sql.Tx, requestID string) (*models.Invoice, error) {
	var total float64
	err := tx.QueryRow(`
		SELECT COALESCE(SUM(pri.quantity * p.price), 0)
		FROM pickup_request_items pri
		JOIN products p ON p.sku = pri.sku
		WHERE pri.request_id = $1
	`, requestID).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("could not compute invoice total: %w", err)
	}

	overdueDaysStr, err := db.getSettingTx(tx, "invoice_overdue_days")
	if err != nil {
		return nil, fmt.Errorf("could not read invoice_overdue_days setting: %w", err)
	}
	overdueDays, err := strconv.Atoi(overdueDaysStr)
	if err != nil {
		return nil, fmt.Errorf("invoice_overdue_days is not a valid integer: %w", err)
	}

	invoiceID, err := functions.GenerateTransactionID("INV", "LG")
	if err != nil {
		return nil, fmt.Errorf("could not generate invoice id: %w", err)
	}

	var inv models.Invoice
	err = tx.QueryRow(`
		INSERT INTO invoices (invoice_id, request_id, total_value, outstanding_value, status, due_at)
		VALUES ($1, $2, $3, $3, 'open', CURRENT_TIMESTAMP + ($4 || ' days')::interval)
		RETURNING invoice_id, request_id, total_value, outstanding_value, status, due_at, created_at, updated_at;
	`, invoiceID, requestID, total, overdueDays).Scan(
		&inv.InvoiceID, &inv.RequestID, &inv.TotalValue, &inv.OutstandingValue,
		&inv.Status, &inv.DueAt, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create invoice: %w", err)
	}

	return &inv, nil
}

func (db *RealDB) getSettingTx(tx *sql.Tx, key string) (string, error) {
	var value string
	err := tx.QueryRow(`SELECT value FROM settings WHERE key = $1`, key).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("could not fetch setting %q: %w", key, err)
	}
	return value, nil
}

func (db *RealDB) RecordPayment(invoiceID string, amount float64) (*models.Receipt, *models.Invoice, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, nil, fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	var outstanding float64
	var status string
	err = tx.QueryRow(`SELECT outstanding_value, status FROM invoices WHERE invoice_id = $1 FOR UPDATE`, invoiceID).
		Scan(&outstanding, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("could not fetch invoice: %w", err)
	}

	newOutstanding := outstanding - amount
	if newOutstanding < 0 {
		newOutstanding = 0
	}
	newStatus := status
	if newOutstanding == 0 {
		newStatus = "paid"
	}

	var inv models.Invoice
	err = tx.QueryRow(`
		UPDATE invoices
		SET outstanding_value = $2, status = $3, updated_at = CURRENT_TIMESTAMP
		WHERE invoice_id = $1
		RETURNING invoice_id, request_id, total_value, outstanding_value, status, due_at, created_at, updated_at;
	`, invoiceID, newOutstanding, newStatus).Scan(
		&inv.InvoiceID, &inv.RequestID, &inv.TotalValue, &inv.OutstandingValue,
		&inv.Status, &inv.DueAt, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("could not update invoice: %w", err)
	}

	receiptID, err := functions.GenerateTransactionID("RCT", "LG")
	if err != nil {
		return nil, nil, fmt.Errorf("could not generate receipt id: %w", err)
	}

	var rcpt models.Receipt
	err = tx.QueryRow(`
		INSERT INTO receipts (receipt_id, invoice_id, amount_paid)
		VALUES ($1, $2, $3)
		RETURNING receipt_id, invoice_id, amount_paid, paid_at;
	`, receiptID, invoiceID, amount).Scan(
		&rcpt.ReceiptID, &rcpt.InvoiceID, &rcpt.AmountPaid, &rcpt.PaidAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create receipt: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	return &rcpt, &inv, nil
}

func (db *RealDB) GetOutstandingInvoices(distributorID string) ([]models.Invoice, error) {
	query := `
		SELECT i.invoice_id, i.request_id, i.total_value, i.outstanding_value,
		       i.status, i.due_at, i.created_at, i.updated_at
		FROM invoices i
		JOIN pickup_requests pr ON pr.request_id = i.request_id
		WHERE i.outstanding_value > 0
	`
	args := []interface{}{}
	if distributorID != "" {
		query += ` AND pr.distributor_id = $1`
		args = append(args, distributorID)
	}
	query += ` ORDER BY i.due_at;`

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not fetch outstanding invoices: %w", err)
	}
	defer rows.Close()

	invoices := make([]models.Invoice, 0)
	for rows.Next() {
		var inv models.Invoice
		if err := rows.Scan(
			&inv.InvoiceID, &inv.RequestID, &inv.TotalValue, &inv.OutstandingValue,
			&inv.Status, &inv.DueAt, &inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("could not scan invoice row: %w", err)
		}
		invoices = append(invoices, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return invoices, nil
}

func (db *RealDB) SetUserPIN(userID, pinHash string) error {
	result, err := db.DB.Exec(
		`UPDATE users SET pin_hash = $2, updated_at = CURRENT_TIMESTAMP WHERE user_id = $1`,
		userID, pinHash,
	)
	if err != nil {
		return fmt.Errorf("could not set pin: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

type PINLoginDetails struct {
	UserID   string
	PinHash  string
	Role     string
	Verified bool
}

func (db *RealDB) GetUserPINLoginDetails(userID string) *PINLoginDetails {
	query := `
	SELECT user_id, pin_hash, role, verified
	FROM users
	WHERE user_id = $1 AND pin_hash IS NOT NULL;`

	var uid, pinHash, role string
	var verified bool

	err := db.DB.QueryRow(query, userID).Scan(&uid, &pinHash, &role, &verified)
	if err != nil {
		return nil
	}

	return &PINLoginDetails{
		UserID:   uid,
		PinHash:  pinHash,
		Role:     role,
		Verified: verified,
	}
}

func (db *RealDB) AssignDistributor(salesAssociateID, distributorID string) error {
	_, err := db.DB.Exec(`
		INSERT INTO distributor_assignments (sales_associate_id, distributor_id)
		VALUES ($1, $2)
		ON CONFLICT (sales_associate_id, distributor_id) DO NOTHING;
	`, salesAssociateID, distributorID)
	if err != nil {
		return fmt.Errorf("could not assign distributor: %w", err)
	}
	return nil
}

func (db *RealDB) UnassignDistributor(salesAssociateID, distributorID string) (bool, error) {
	result, err := db.DB.Exec(`
		DELETE FROM distributor_assignments
		WHERE sales_associate_id = $1 AND distributor_id = $2;
	`, salesAssociateID, distributorID)
	if err != nil {
		return false, fmt.Errorf("could not unassign distributor: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("could not check rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

func (db *RealDB) GetAssignedDistributors(salesAssociateID string) ([]models.DistributorAssignment, error) {
	rows, err := db.DB.Query(`
		SELECT sales_associate_id, distributor_id, created_at
		FROM distributor_assignments
		WHERE sales_associate_id = $1
		ORDER BY created_at;
	`, salesAssociateID)
	if err != nil {
		return nil, fmt.Errorf("could not fetch distributor assignments: %w", err)
	}
	defer rows.Close()

	assignments := make([]models.DistributorAssignment, 0)
	for rows.Next() {
		var a models.DistributorAssignment
		if err := rows.Scan(&a.SalesAssociateID, &a.DistributorID, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("could not scan assignment row: %w", err)
		}
		assignments = append(assignments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return assignments, nil
}

func (db *RealDB) IsDistributorAssigned(salesAssociateID, distributorID string) (bool, error) {
	var exists bool
	err := db.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM distributor_assignments
			WHERE sales_associate_id = $1 AND distributor_id = $2
		);
	`, salesAssociateID, distributorID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("could not check assignment: %w", err)
	}
	return exists, nil
}

func (db *RealDB) GetOutsideCoverage(salesAssociateID string, weekStart, weekEnd time.Time) (*models.OutsideCoverageReport, error) {
	query := `
		SELECT
			COALESCE(SUM(i.total_value), 0) AS total_value,
			COALESCE(SUM(
				CASE WHEN NOT EXISTS (
					SELECT 1 FROM distributor_assignments da
					WHERE da.sales_associate_id = pr.sales_associate_id
					  AND da.distributor_id = pr.distributor_id
				) THEN i.total_value ELSE 0 END
			), 0) AS outside_value
		FROM pickup_requests pr
		JOIN invoices i ON i.request_id = pr.request_id
		WHERE pr.sales_associate_id = $1
		  AND pr.confirmed = TRUE
		  AND pr.created_at >= $2
		  AND pr.created_at < $3;
	`

	var totalValue, outsideValue float64
	err := db.DB.QueryRow(query, salesAssociateID, weekStart, weekEnd).Scan(&totalValue, &outsideValue)
	if err != nil {
		return nil, fmt.Errorf("could not compute outside coverage: %w", err)
	}

	thresholdStr, err := db.GetSettingFloat("outside_coverage_pct")
	if err != nil {
		return nil, fmt.Errorf("could not read outside_coverage_pct setting: %w", err)
	}

	var pct float64
	if totalValue > 0 {
		pct = (outsideValue / totalValue) * 100
	}

	return &models.OutsideCoverageReport{
		SalesAssociateID:     salesAssociateID,
		WeekStart:            weekStart,
		WeekEnd:              weekEnd,
		TotalPickupValue:     totalValue,
		OutsideCoverageValue: outsideValue,
		OutsideCoveragePct:   pct,
		ThresholdPct:         thresholdStr,
		Exceeded:             pct > thresholdStr,
	}, nil
}

func (db *RealDB) GetResumptionLogs(salesAssociateID string, from, to time.Time) ([]models.ResumptionLog, error) {
	query := `
		SELECT sales_associate_id, route_day, date, time, latitude, longitude,
		       distance_to_route_m, result, COALESCE(device_ref, ''), created_at
		FROM resumption_log
		WHERE date >= $1 AND date < $2
	`
	args := []interface{}{from, to}
	if salesAssociateID != "" {
		query += ` AND sales_associate_id = $3`
		args = append(args, salesAssociateID)
	}
	query += ` ORDER BY date DESC, sales_associate_id;`

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not fetch resumption logs: %w", err)
	}
	defer rows.Close()

	logs := make([]models.ResumptionLog, 0)
	for rows.Next() {
		var l models.ResumptionLog
		if err := rows.Scan(
			&l.SalesAssociateID, &l.RouteDay, &l.Date, &l.Time,
			&l.Latitude, &l.Longitude, &l.DistanceToRouteM, &l.Result, &l.DeviceRef, &l.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("could not scan resumption log row: %w", err)
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return logs, nil
}

func (db *RealDB) GetOutletVisits(salesAssociateID string, from, to time.Time) ([]models.OutletVisit, error) {
	query := `
		SELECT sales_associate_id, outlet_id, route_day, visited_at,
		       latitude, longitude, distance_from_outlet_m, geofence_status, created_at
		FROM outlet_visits
		WHERE visited_at >= $1 AND visited_at < $2
	`
	args := []interface{}{from, to}
	if salesAssociateID != "" {
		query += ` AND sales_associate_id = $3`
		args = append(args, salesAssociateID)
	}
	query += ` ORDER BY visited_at DESC;`

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not fetch outlet visits: %w", err)
	}
	defer rows.Close()

	visits := make([]models.OutletVisit, 0)
	for rows.Next() {
		var v models.OutletVisit
		if err := rows.Scan(
			&v.SalesAssociateID, &v.OutletID, &v.RouteDay, &v.VisitedAt,
			&v.Latitude, &v.Longitude, &v.DistanceFromOutletM, &v.GeofenceStatus, &v.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("could not scan outlet visit row: %w", err)
		}
		visits = append(visits, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return visits, nil
}

func (db *RealDB) MarkOverdueInvoices() ([]string, error) {
	rows, err := db.DB.Query(`
		UPDATE invoices
		SET status = 'overdue', updated_at = CURRENT_TIMESTAMP
		WHERE status = 'open' AND due_at < CURRENT_TIMESTAMP
		RETURNING invoice_id;
	`)
	if err != nil {
		return nil, fmt.Errorf("could not mark overdue invoices: %w", err)
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("could not scan overdue invoice id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return ids, nil
}
