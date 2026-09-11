package sqltools

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Olayori-X/stock-control-backend/functions"
	"github.com/Olayori-X/stock-control-backend/models"
)

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

func nullableStringPtr(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

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

func (db *RealDB) GetPlannedVsActual(salesAssociateID string, date time.Time) (*models.PlannedVsActual, error) {
	routeDay := date.Weekday().String()

	report := &models.PlannedVsActual{
		SalesAssociateID: salesAssociateID,
		RouteDay:         routeDay,
		Date:             date,
	}

	plannedRows, err := db.DB.Query(`
		SELECT outlet_id FROM route_plans
		WHERE sales_associate_id = $1 AND route_day = $2;
	`, salesAssociateID, routeDay)
	if err != nil {
		return nil, fmt.Errorf("could not fetch planned outlets: %w", err)
	}
	planned := make(map[string]bool)
	for plannedRows.Next() {
		var id string
		if err := plannedRows.Scan(&id); err != nil {
			plannedRows.Close()
			return nil, fmt.Errorf("could not scan planned outlet: %w", err)
		}
		planned[id] = true
	}
	plannedRows.Close()
	if err := plannedRows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}
	report.OutletsPlanned = len(planned)

	// ── Visited outlets (PASS geofence attempts) for this date ──
	visitRows, err := db.DB.Query(`
		SELECT outlet_id, visited_at FROM outlet_visits
		WHERE sales_associate_id = $1
		  AND geofence_status = 'PASS'
		  AND visited_at >= $2 AND visited_at < $2::date + interval '1 day';
	`, salesAssociateID, date)
	if err != nil {
		return nil, fmt.Errorf("could not fetch outlet visits: %w", err)
	}
	visitedOutlets := make(map[string]bool)
	salesCalls := 0
	var lastActivity *time.Time
	for visitRows.Next() {
		var outletID string
		var visitedAt time.Time
		if err := visitRows.Scan(&outletID, &visitedAt); err != nil {
			visitRows.Close()
			return nil, fmt.Errorf("could not scan visit: %w", err)
		}
		visitedOutlets[outletID] = true
		salesCalls++
		if lastActivity == nil || visitedAt.After(*lastActivity) {
			t := visitedAt
			lastActivity = &t
		}
	}
	visitRows.Close()
	if err := visitRows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}
	report.OutletsVisited = len(visitedOutlets)
	report.SalesCalls = salesCalls

	var totalValue sql.NullFloat64
	productiveOutlets := make(map[string]bool)
	saleRows, err := db.DB.Query(`
		SELECT outlet_id, total_value, created_at FROM sales
		WHERE sales_associate_id = $1
		  AND created_at >= $2 AND created_at < $2::date + interval '1 day';
	`, salesAssociateID, date)
	if err != nil {
		return nil, fmt.Errorf("could not fetch sales: %w", err)
	}
	var sumValue float64
	for saleRows.Next() {
		var outletID string
		var value float64
		var createdAt time.Time
		if err := saleRows.Scan(&outletID, &value, &createdAt); err != nil {
			saleRows.Close()
			return nil, fmt.Errorf("could not scan sale: %w", err)
		}
		productiveOutlets[outletID] = true
		sumValue += value
		if lastActivity == nil || createdAt.After(*lastActivity) {
			t := createdAt
			lastActivity = &t
		}
	}
	saleRows.Close()
	if err := saleRows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}
	_ = totalValue
	report.ProductiveVisits = len(productiveOutlets)
	report.SalesValue = sumValue
	report.LastActivity = lastActivity

	missed := make([]string, 0)
	for outletID := range planned {
		if !visitedOutlets[outletID] {
			missed = append(missed, outletID)
		}
	}
	report.MissedOutlets = missed

	var resumptionTime, resumptionResult string
	err = db.DB.QueryRow(`
		SELECT time::text, result FROM resumption_log
		WHERE sales_associate_id = $1 AND date = $2::date;
	`, salesAssociateID, date).Scan(&resumptionTime, &resumptionResult)
	if err == nil {
		report.ResumptionTime = &resumptionTime
		report.ResumptionResult = &resumptionResult
	} else if err != sql.ErrNoRows {
		return nil, fmt.Errorf("could not fetch resumption record: %w", err)
	}

	if report.OutletsPlanned > 0 {
		visitedPlanned := 0
		for outletID := range visitedOutlets {
			if planned[outletID] {
				visitedPlanned++
			}
		}
		report.CoveragePct = (float64(visitedPlanned) / float64(report.OutletsPlanned)) * 100
	}

	if len(visitedOutlets) > 0 {
		onRoute := 0
		for outletID := range visitedOutlets {
			if planned[outletID] {
				onRoute++
			}
		}
		report.RouteAdherencePct = (float64(onRoute) / float64(len(visitedOutlets))) * 100
	}

	return report, nil
}

type routeStop struct {
	OutletID  string
	Latitude  float64
	Longitude float64
	Area      string
	Zone      string
	Sequence  int
}

func (db *RealDB) getRouteStopsForDay(salesAssociateID, routeDay string) ([]routeStop, error) {
	rows, err := db.DB.Query(`
		SELECT o.outlet_id, o.latitude, o.longitude, COALESCE(o.area, ''), COALESCE(o.zone, ''), rp.sequence
		FROM route_plans rp
		JOIN outlets o ON o.outlet_id = rp.outlet_id
		WHERE rp.sales_associate_id = $1 AND rp.route_day = $2 AND o.active = TRUE
		ORDER BY rp.sequence;
	`, salesAssociateID, routeDay)
	if err != nil {
		return nil, fmt.Errorf("could not fetch route stops: %w", err)
	}
	defer rows.Close()

	stops := make([]routeStop, 0)
	for rows.Next() {
		var s routeStop
		if err := rows.Scan(&s.OutletID, &s.Latitude, &s.Longitude, &s.Area, &s.Zone, &s.Sequence); err != nil {
			return nil, fmt.Errorf("could not scan route stop: %w", err)
		}
		stops = append(stops, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}
	return stops, nil
}

func (db *RealDB) GetRouteEfficiency(salesAssociateID, routeDay string) (*models.RouteEfficiency, error) {
	stops, err := db.getRouteStopsForDay(salesAssociateID, routeDay)
	if err != nil {
		return nil, err
	}

	report := &models.RouteEfficiency{
		SalesAssociateID: salesAssociateID,
		RouteDay:         routeDay,
		OutletCount:      len(stops),
		Outliers:         []string{},
		BacktrackLegs:    []string{},
	}

	if len(stops) == 0 {
		return report, nil
	}

	var totalDistance float64
	for i := 0; i < len(stops)-1; i++ {
		d := functions.HaversineMeters(stops[i].Latitude, stops[i].Longitude, stops[i+1].Latitude, stops[i+1].Longitude)
		totalDistance += d
	}
	report.PlannedDistanceM = totalDistance
	if len(stops) > 1 {
		report.AverageLegDistanceM = totalDistance / float64(len(stops)-1)
	}

	var sumLat, sumLon float64
	for _, s := range stops {
		sumLat += s.Latitude
		sumLon += s.Longitude
	}
	centroidLat := sumLat / float64(len(stops))
	centroidLon := sumLon / float64(len(stops))
	report.CentroidLatitude = centroidLat
	report.CentroidLongitude = centroidLon

	var sumDistFromCentroid float64
	distFromCentroid := make([]float64, len(stops))
	for i, s := range stops {
		d := functions.HaversineMeters(s.Latitude, s.Longitude, centroidLat, centroidLon)
		distFromCentroid[i] = d
		sumDistFromCentroid += d
	}
	dispersion := sumDistFromCentroid / float64(len(stops))
	report.DispersionM = dispersion

	for i, s := range stops {
		if dispersion > 0 && distFromCentroid[i] > dispersion*1.5 {
			report.Outliers = append(report.Outliers, s.OutletID)
		}
	}

	for i := 0; i < len(stops)-2; i++ {
		a, b, c := stops[i], stops[i+1], stops[i+2]
		direct := functions.HaversineMeters(a.Latitude, a.Longitude, c.Latitude, c.Longitude)
		viaB := functions.HaversineMeters(a.Latitude, a.Longitude, b.Latitude, b.Longitude) +
			functions.HaversineMeters(b.Latitude, b.Longitude, c.Latitude, c.Longitude)
		if direct > 0 && viaB > direct*1.5 {
			report.BacktrackCount++
			report.BacktrackLegs = append(report.BacktrackLegs, fmt.Sprintf("%s -> %s -> %s", a.OutletID, b.OutletID, c.OutletID))
		}
	}

	report.DominantArea, report.AreaAlignmentPct = dominantFieldPct(stops, func(s routeStop) string { return s.Area })
	report.DominantZone, report.ZoneAlignmentPct = dominantFieldPct(stops, func(s routeStop) string { return s.Zone })

	return report, nil
}

func dominantFieldPct(stops []routeStop, keyFn func(routeStop) string) (dominant string, pct float64) {
	counts := make(map[string]int)
	for _, s := range stops {
		key := keyFn(s)
		if key == "" {
			continue
		}
		counts[key]++
	}

	if len(counts) == 0 {
		return "", 0
	}

	var best string
	var bestCount int
	for key, count := range counts {
		if count > bestCount {
			best = key
			bestCount = count
		}
	}

	return best, (float64(bestCount) / float64(len(stops))) * 100
}

func (db *RealDB) GetAvailableStock(salesAssociateID, sku string) (int, error) {
	var pickedUp int
	err := db.DB.QueryRow(`
		SELECT COALESCE(SUM(pri.quantity), 0)
		FROM pickup_request_items pri
		JOIN pickup_requests pr ON pr.request_id = pri.request_id
		WHERE pr.sales_associate_id = $1 AND pr.confirmed = TRUE AND pri.sku = $2;
	`, salesAssociateID, sku).Scan(&pickedUp)
	if err != nil {
		return 0, fmt.Errorf("could not compute picked-up quantity: %w", err)
	}

	var sold int
	err = db.DB.QueryRow(`
		SELECT COALESCE(SUM(quantity), 0)
		FROM sales
		WHERE sales_associate_id = $1 AND sku = $2;
	`, salesAssociateID, sku).Scan(&sold)
	if err != nil {
		return 0, fmt.Errorf("could not compute sold quantity: %w", err)
	}

	return pickedUp - sold, nil
}
