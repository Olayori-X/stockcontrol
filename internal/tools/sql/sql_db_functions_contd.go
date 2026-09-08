package sqltools

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/Olayori-X/stock-control-backend/api"
	"github.com/Olayori-X/stock-control-backend/functions"
	"github.com/Olayori-X/stock-control-backend/models"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

// PlannedOutletLocation is a lightweight projection of route_plans + outlets
// used purely for the resumption geofence check — no need for the full
// Outlet model here.
type PlannedOutletLocation struct {
	OutletID  string
	Latitude  float64
	Longitude float64
}

// GetPlannedOutletsForDay returns every outlet assigned to a sales associate
// for a given route day (e.g. "Monday"), for resumption geofence checking.
func (db *RealDB) GetPlannedOutletsForDay(salesAssociateID, routeDay string) ([]PlannedOutletLocation, error) {
	query := `
	SELECT o.outlet_id, o.latitude, o.longitude
	FROM route_plans rp
	JOIN outlets o ON o.outlet_id = rp.outlet_id
	WHERE rp.sales_associate_id = $1 AND rp.route_day = $2 AND o.active = TRUE;
	`

	rows, err := db.DB.Query(query, salesAssociateID, routeDay)
	if err != nil {
		return nil, fmt.Errorf("could not fetch planned outlets: %w", err)
	}
	defer rows.Close()

	locations := make([]PlannedOutletLocation, 0)
	for rows.Next() {
		var loc PlannedOutletLocation
		if err := rows.Scan(&loc.OutletID, &loc.Latitude, &loc.Longitude); err != nil {
			return nil, fmt.Errorf("could not scan planned outlet row: %w", err)
		}
		locations = append(locations, loc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return locations, nil
}

// GetSettingFloat reads a numeric threshold from the settings table
// (e.g. resumption_radius_km) so it stays admin-configurable rather than
// hardcoded in the handler.
func (db *RealDB) GetSettingFloat(key string) (float64, error) {
	var value string
	err := db.DB.QueryRow(`SELECT value FROM settings WHERE key = $1`, key).Scan(&value)
	if err != nil {
		return 0, fmt.Errorf("could not fetch setting %q: %w", key, err)
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("setting %q is not a valid number: %w", key, err)
	}
	return parsed, nil
}

// RecordResumption upserts today's resumption result for a sales associate.
// ON CONFLICT lets a second login attempt on the same day overwrite the
// previous result rather than erroring on the UNIQUE(sales_associate_id, date).
func (db *RealDB) RecordResumption(salesAssociateID, routeDay string, lat, lon, distanceM float64, result, deviceRef string) error {
	query := `
	INSERT INTO resumption_log (sales_associate_id, route_day, date, time, latitude, longitude, distance_to_route_m, result, device_ref)
	VALUES ($1, $2, CURRENT_DATE, CURRENT_TIME, $3, $4, $5, $6, $7)
	ON CONFLICT (sales_associate_id, date)
	DO UPDATE SET
		route_day = EXCLUDED.route_day,
		time = EXCLUDED.time,
		latitude = EXCLUDED.latitude,
		longitude = EXCLUDED.longitude,
		distance_to_route_m = EXCLUDED.distance_to_route_m,
		result = EXCLUDED.result,
		device_ref = EXCLUDED.device_ref;
	`

	_, err := db.DB.Exec(query, salesAssociateID, routeDay, lat, lon, distanceM, result, deviceRef)
	if err != nil {
		return fmt.Errorf("could not record resumption: %w", err)
	}
	return nil
}

// ============================================================================
// ROUTE PLANNING
// ============================================================================

// SetRoutePlan replaces a sales associate's entire route plan for a given
// day: existing stops for that associate/day are deleted, then the new
// ordered list is inserted with sequence = position in the slice (1-indexed).
// Runs in a transaction so a partial replace never lands (e.g. delete
// succeeds but a later insert fails on a bad outlet_id).
//
// Replacing rather than diffing is deliberate: an admin's route-planner UI
// naturally submits the whole day's list on every save (drag-to-reorder,
// add, remove all mutate the same in-memory list), so there's no need for
// separate add/remove/reorder endpoints or diffing logic here.
func (db *RealDB) SetRoutePlan(salesAssociateID, routeDay string, outletIDs []string) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`DELETE FROM route_plans WHERE sales_associate_id = $1 AND route_day = $2`,
		salesAssociateID, routeDay,
	)
	if err != nil {
		return fmt.Errorf("could not clear existing route plan: %w", err)
	}

	insert := `
		INSERT INTO route_plans (sales_associate_id, outlet_id, route_day, sequence, approved)
		VALUES ($1, $2, $3, $4, FALSE);
	`
	for i, outletID := range outletIDs {
		_, err = tx.Exec(insert, salesAssociateID, outletID, routeDay, i+1)
		if err != nil {
			return fmt.Errorf("could not insert route plan stop (outlet=%s): %w", outletID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction: %w", err)
	}

	return nil
}

// GetRoutePlan fetches a sales associate's full plan for a route day,
// joined against outlets for display details, ordered by sequence.
func (db *RealDB) GetRoutePlan(salesAssociateID, routeDay string) (*models.RoutePlan, error) {
	query := `
		SELECT o.outlet_id, o.name, o.latitude, o.longitude, o.address, rp.sequence,
		       bool_and(rp.approved) OVER () AS all_approved,
		       max(rp.updated_at) OVER () AS latest_update
		FROM route_plans rp
		JOIN outlets o ON o.outlet_id = rp.outlet_id
		WHERE rp.sales_associate_id = $1 AND rp.route_day = $2
		ORDER BY rp.sequence;
	`

	rows, err := db.DB.Query(query, salesAssociateID, routeDay)
	if err != nil {
		return nil, fmt.Errorf("could not fetch route plan: %w", err)
	}
	defer rows.Close()

	plan := &models.RoutePlan{
		SalesAssociateID: salesAssociateID,
		RouteDay:         routeDay,
		Stops:            []models.RoutePlanStop{},
	}

	for rows.Next() {
		var stop models.RoutePlanStop
		var approved bool
		var updatedAt time.Time
		if err := rows.Scan(
			&stop.OutletID, &stop.OutletName, &stop.Latitude, &stop.Longitude,
			&stop.Address, &stop.Sequence, &approved, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("could not scan route plan stop: %w", err)
		}
		plan.Stops = append(plan.Stops, stop)
		plan.Approved = approved
		plan.UpdatedAt = updatedAt
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return plan, nil
}

// ApproveRoutePlan marks every stop in an associate's day plan as approved.
// Returns false (no error) if the plan doesn't exist / has no stops, so the
// handler can distinguish "nothing to approve" from a real DB failure.
func (db *RealDB) ApproveRoutePlan(salesAssociateID, routeDay string) (bool, error) {
	result, err := db.DB.Exec(
		`UPDATE route_plans SET approved = TRUE, updated_at = CURRENT_TIMESTAMP
		 WHERE sales_associate_id = $1 AND route_day = $2`,
		salesAssociateID, routeDay,
	)
	if err != nil {
		return false, fmt.Errorf("could not approve route plan: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("could not check rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

// ============================================================================
// OUTLETS
// ============================================================================

// AddOutlet generates a permanent outlet_id and inserts the outlet.
// outlet_id is never reused even after deactivation — historical sales,
// visits, and route plans stay attached to it via FK.
func (db *RealDB) AddOutlet(outlet *models.Outlet) error {
	outletID := "OUT_" + uuid.NewString()

	query := `
		INSERT INTO outlets (
			outlet_id, name, address, outlet_type, phone,
			latitude, longitude, area, zone,
			assigned_sales_associate_id, route_day, priority
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING outlet_id, active, created_at, updated_at;
	`

	err := db.DB.QueryRow(
		query,
		outletID, outlet.Name, outlet.Address, outlet.OutletType, outlet.Phone,
		outlet.Latitude, outlet.Longitude, outlet.Area, outlet.Zone,
		nullableString(outlet.AssignedSalesAssociateID), nullableString(outlet.RouteDay), outlet.Priority,
	).Scan(&outlet.OutletID, &outlet.Active, &outlet.CreatedAt, &outlet.UpdatedAt)
	if err != nil {
		return fmt.Errorf("could not create outlet: %w", err)
	}

	outlet.Name = outlet.Name // no-op, keeps the rest of the struct as passed in

	return nil
}

// GetOutlets lists outlets. includeInactive controls whether deactivated
// outlets are returned — the admin dashboard's management view needs to see
// them (to reactivate or audit), but route planning / sales-associate pickers
// should only ever see active ones.
func (db *RealDB) GetOutlets(includeInactive bool) ([]models.Outlet, error) {
	query := `
		SELECT outlet_id, name, address, outlet_type, phone,
		       latitude, longitude, area, zone,
		       COALESCE(assigned_sales_associate_id, ''), COALESCE(route_day, ''), priority,
		       active, created_at, updated_at
		FROM outlets
	`
	if !includeInactive {
		query += ` WHERE active = TRUE`
	}
	query += ` ORDER BY name;`

	rows, err := db.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("could not fetch outlets: %w", err)
	}
	defer rows.Close()

	outlets := make([]models.Outlet, 0)
	for rows.Next() {
		var o models.Outlet
		if err := rows.Scan(
			&o.OutletID, &o.Name, &o.Address, &o.OutletType, &o.Phone,
			&o.Latitude, &o.Longitude, &o.Area, &o.Zone,
			&o.AssignedSalesAssociateID, &o.RouteDay, &o.Priority,
			&o.Active, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("could not scan outlet row: %w", err)
		}
		outlets = append(outlets, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return outlets, nil
}

// GetOutletByID fetches a single outlet regardless of active status —
// callers that need "must be active" (e.g. route planning) check the
// Active field themselves.
func (db *RealDB) GetOutletByID(outletID string) (*models.Outlet, error) {
	query := `
		SELECT outlet_id, name, address, outlet_type, phone,
		       latitude, longitude, area, zone,
		       COALESCE(assigned_sales_associate_id, ''), COALESCE(route_day, ''), priority,
		       active, created_at, updated_at
		FROM outlets
		WHERE outlet_id = $1;
	`

	var o models.Outlet
	err := db.DB.QueryRow(query, outletID).Scan(
		&o.OutletID, &o.Name, &o.Address, &o.OutletType, &o.Phone,
		&o.Latitude, &o.Longitude, &o.Area, &o.Zone,
		&o.AssignedSalesAssociateID, &o.RouteDay, &o.Priority,
		&o.Active, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("could not fetch outlet: %w", err)
	}

	return &o, nil
}

// EditOutlet updates every editable field. outlet_id and active status are
// never changed here — active status has its own dedicated
// activate/deactivate operation so it can't be silently flipped by a
// general edit.
func (db *RealDB) EditOutlet(outlet *models.Outlet) (*models.Outlet, error) {
	query := `
		UPDATE outlets
		SET name = $2, address = $3, outlet_type = $4, phone = $5,
		    latitude = $6, longitude = $7, area = $8, zone = $9,
		    assigned_sales_associate_id = $10, route_day = $11, priority = $12,
		    updated_at = CURRENT_TIMESTAMP
		WHERE outlet_id = $1
		RETURNING outlet_id, name, address, outlet_type, phone,
		          latitude, longitude, area, zone,
		          COALESCE(assigned_sales_associate_id, ''), COALESCE(route_day, ''), priority,
		          active, created_at, updated_at;
	`

	var updated models.Outlet
	err := db.DB.QueryRow(
		query,
		outlet.OutletID, outlet.Name, outlet.Address, outlet.OutletType, outlet.Phone,
		outlet.Latitude, outlet.Longitude, outlet.Area, outlet.Zone,
		nullableString(outlet.AssignedSalesAssociateID), nullableString(outlet.RouteDay), outlet.Priority,
	).Scan(
		&updated.OutletID, &updated.Name, &updated.Address, &updated.OutletType, &updated.Phone,
		&updated.Latitude, &updated.Longitude, &updated.Area, &updated.Zone,
		&updated.AssignedSalesAssociateID, &updated.RouteDay, &updated.Priority,
		&updated.Active, &updated.CreatedAt, &updated.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("could not update outlet: %w", err)
	}

	return &updated, nil
}

// SetOutletActive activates or deactivates an outlet. Deactivation is the
// only supported form of "delete" — outlet_id must never disappear from the
// table, since sales/visits/route_plans reference it by FK and historical
// records must stay attached to the correct outlet.
func (db *RealDB) SetOutletActive(outletID string, active bool) (bool, error) {
	result, err := db.DB.Exec(
		`UPDATE outlets SET active = $2, updated_at = CURRENT_TIMESTAMP WHERE outlet_id = $1`,
		outletID, active,
	)
	if err != nil {
		return false, fmt.Errorf("could not update outlet status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("could not check rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

// nullableString converts an empty string to SQL NULL, so optional
// foreign-key/text columns (assigned_sales_associate_id, route_day) don't
// get stored as empty-string values that would violate the FK or just be
// semantically wrong for "unset".
func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// ============================================================================
// OUTLET GEOFENCE / VISITS
// ============================================================================

// RecordOutletVisit inserts a visit attempt — PASS or FAIL — so there's an
// audit trail of every geofence check, not just successful ones. The caller
// decides what to allow next based on the returned status; this function
// just records what happened.
func (db *RealDB) RecordOutletVisit(salesAssociateID, outletID, routeDay string, lat, lon, distanceM float64, status string) error {
	query := `
		INSERT INTO outlet_visits (
			sales_associate_id, outlet_id, route_day, visited_at,
			latitude, longitude, distance_from_outlet_m, geofence_status
		)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, $4, $5, $6, $7);
	`

	_, err := db.DB.Exec(query, salesAssociateID, outletID, routeDay, lat, lon, distanceM, status)
	if err != nil {
		return fmt.Errorf("could not record outlet visit: %w", err)
	}
	return nil
}

// SubmitSale attempts to record a sale. The geofence check happens inside
// this function, at the moment of submission — it never trusts a prior
// PASS from ConfirmOutletVisitHandler, since GPS can drift or the associate
// can walk away between confirming a visit and submitting a sale.
//
// Returns (sale, alreadyExisted, blocked, error):
//   - blocked=true, sale=nil: outside geofence, nothing was written except
//     the outlet_visits audit row.
//   - alreadyExisted=true: this transaction_id was already recorded (safe
//     retry) — the original sale is returned, nothing new was inserted.
//   - otherwise: a new sale was created and returned.
func (db *RealDB) SubmitSale(salesAssociateID string, input *api.SubmitSaleInput) (sale *models.Sale, alreadyExisted bool, blocked bool, err error) {
	outlet, err := db.GetOutletByID(input.OutletID)
	if err != nil {
		return nil, false, false, fmt.Errorf("could not fetch outlet: %w", err)
	}
	if outlet == nil || !outlet.Active {
		return nil, false, false, fmt.Errorf("outlet not found")
	}

	radiusM, err := db.GetSettingFloat("outlet_geofence_m")
	if err != nil {
		return nil, false, false, fmt.Errorf("could not read outlet_geofence_m setting: %w", err)
	}

	distance := functions.HaversineMeters(input.Latitude, input.Longitude, outlet.Latitude, outlet.Longitude)

	status := "FAIL"
	if distance <= radiusM {
		status = "PASS"
	}

	// Record the visit attempt regardless of outcome — same audit-trail
	// principle as ConfirmOutletVisitHandler.
	if visitErr := db.RecordOutletVisit(
		salesAssociateID, input.OutletID, input.RouteDay,
		input.Latitude, input.Longitude, distance, status,
	); visitErr != nil {
		log.Error("Failed to record outlet visit during sale submission: ", visitErr)
		// Non-fatal: the geofence decision below doesn't depend on this
		// write succeeding, so a sale isn't blocked purely because the
		// audit insert had a transient failure. It is logged either way.
	}

	if status == "FAIL" {
		return nil, false, true, nil
	}

	// Existing transaction_id → safe retry, return what's already there.
	existing, err := db.getSaleByTransactionID(input.TransactionID)
	if err != nil {
		return nil, false, false, fmt.Errorf("could not check for existing sale: %w", err)
	}
	if existing != nil {
		return existing, true, false, nil
	}

	product, err := db.GetProductBySKU(input.SKU)
	if err != nil {
		return nil, false, false, fmt.Errorf("could not fetch product: %w", err)
	}
	if product == nil {
		return nil, false, false, fmt.Errorf("product not found for sku %s", input.SKU)
	}

	totalValue := product.Price * float64(input.Quantity)

	insert := `
		INSERT INTO sales (
			transaction_id, sales_associate_id, outlet_id, sku, quantity,
			unit_value, total_value, latitude, longitude,
			distance_from_outlet_m, geofence_status, route_day, synced_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, CURRENT_TIMESTAMP)
		ON CONFLICT (transaction_id) DO NOTHING
		RETURNING transaction_id, sales_associate_id, outlet_id, sku, quantity,
		          unit_value, total_value, latitude, longitude,
		          distance_from_outlet_m, geofence_status, route_day, synced_at, created_at;
	`

	var s models.Sale
	err = db.DB.QueryRow(
		insert,
		input.TransactionID, salesAssociateID, input.OutletID, input.SKU, input.Quantity,
		product.Price, totalValue, input.Latitude, input.Longitude,
		distance, status, input.RouteDay,
	).Scan(
		&s.TransactionID, &s.SalesAssociateID, &s.OutletID, &s.SKU, &s.Quantity,
		&s.UnitValue, &s.TotalValue, &s.Latitude, &s.Longitude,
		&s.DistanceFromOutletM, &s.GeofenceStatus, &s.RouteDay, &s.SyncedAt, &s.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// Lost a race against a concurrent identical retry between our
			// existence check above and this insert — fetch what the other
			// request wrote and return that instead of erroring.
			existing, fetchErr := db.getSaleByTransactionID(input.TransactionID)
			if fetchErr != nil {
				return nil, false, false, fmt.Errorf("could not fetch sale after conflict: %w", fetchErr)
			}
			if existing != nil {
				return existing, true, false, nil
			}
		}
		return nil, false, false, fmt.Errorf("could not insert sale: %w", err)
	}

	return &s, false, false, nil
}

func (db *RealDB) getSaleByTransactionID(transactionID string) (*models.Sale, error) {
	query := `
		SELECT transaction_id, sales_associate_id, outlet_id, sku, quantity,
		       unit_value, total_value, latitude, longitude,
		       distance_from_outlet_m, geofence_status, route_day, synced_at, created_at
		FROM sales
		WHERE transaction_id = $1;
	`

	var s models.Sale
	err := db.DB.QueryRow(query, transactionID).Scan(
		&s.TransactionID, &s.SalesAssociateID, &s.OutletID, &s.SKU, &s.Quantity,
		&s.UnitValue, &s.TotalValue, &s.Latitude, &s.Longitude,
		&s.DistanceFromOutletM, &s.GeofenceStatus, &s.RouteDay, &s.SyncedAt, &s.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("could not fetch sale: %w", err)
	}
	return &s, nil
}
