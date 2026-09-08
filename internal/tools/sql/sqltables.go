package sqltools

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func (db *RealDB) SetupDatabase() error {
	// In a real implementation, this would set up the database connection.
	// For mockDB, we can just return nil to indicate success.
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:secret@localhost:5432/fameduel?sslmode=disable"
	}
	dbpointer, err := sql.Open("postgres", connStr)

	if err != nil {
		log.Fatal("Failed to connect to the database: ", err)
		return err
	}

	if err = dbpointer.Ping(); err != nil {
		log.Fatal("Failed to connect to the database: ", err)
		return err
	}

	db.DB = dbpointer

	// ── Core auth / users ──
	CreateUserTable(dbpointer)
	CreateLoggedInUserTable(dbpointer)
	CreateForgotPasswordTable(dbpointer)

	// ── SCS: pickups, products ──
	CreatePickupRequestTable(dbpointer)
	CreatePickupRequestItemsTable(dbpointer)
	CreateProductTable(dbpointer)

	// ── SCS: invoicing ──
	CreateInvoiceTable(dbpointer)
	CreateReceiptTable(dbpointer)

	// ── Master data: outlets, routes, distributors, regions, settings ──
	CreateOutletTable(dbpointer)
	CreateDistributorAssignmentTable(dbpointer)
	CreateRoutePlanTable(dbpointer)
	CreateRegionTable(dbpointer)
	CreateSettingTable(dbpointer)

	// ── Field activity: resumption, visits, sales ──
	CreateResumptionLogTable(dbpointer)
	CreateOutletVisitTable(dbpointer)
	CreateSaleTable(dbpointer)

	// ── Audit ──
	CreateAuditLogTable(dbpointer)

	return nil
}

// ============================================================================
// CORE AUTH / USERS
// ============================================================================

func CreateUserTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		user_id VARCHAR(50) NOT NULL UNIQUE CHECK (user_id <> ''),
		name VARCHAR(150) NOT NULL CHECK (name <> ''),
		email VARCHAR(100) NOT NULL UNIQUE CHECK (email <> ''),
		phone VARCHAR(15) NOT NULL UNIQUE CHECK (phone <> ''),
		password VARCHAR(100) NOT NULL CHECK (password <> ''),
		role VARCHAR(20) NOT NULL DEFAULT 'sales' CHECK (role IN ('admin', 'sales', 'distributor')),
		verified BOOL NOT NULL DEFAULT FALSE CHECK (verified IN (TRUE, FALSE)),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not create table: ", err)
		return err
	}
	return nil
}

func DeleteUserTable(db *sql.DB) error {
	query := `
	DROP TABLE IF EXISTS users;
	`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not delete table: ", err)
		return err
	}
	return nil
}

func AlterUsersTable(db *sql.DB) error {
	query := `
		ALTER TABLE users 
		ADD COLUMN IF NOT EXISTS rank INT NOT NULL DEFAULT 0 CHECK (rank >= 0);
	`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not alter table: ", err)
		return err
	}
	return nil
}

// AlterUsersTableAddPIN adds the 8-digit hashed PIN column used to
// authenticate Sales Associates on the APK. PIN identifies/authenticates
// only — it does not authorize stock release (distributor confirmation does).
func AlterUsersTableAddPIN(db *sql.DB) error {
	query := `
		ALTER TABLE users
		ADD COLUMN IF NOT EXISTS pin_hash VARCHAR(100);
	`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not alter table: ", err)
		return err
	}
	return nil
}

func CreateLoggedInUserTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS loggedin_users (
		id SERIAL PRIMARY KEY,
		user_id VARCHAR(50) NOT NULL UNIQUE CHECK (user_id <> ''),
		role VARCHAR(20) NOT NULL DEFAULT 'sales' CHECK (role IN ('admin', 'sales', 'distributor')),
		code VARCHAR(200),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not create table: ", err)
		return err
	}
	return nil
}

func CreateForgotPasswordTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS forgotpassword (
		id SERIAL PRIMARY KEY,
		email VARCHAR(50) NOT NULL UNIQUE CHECK (email <> ''),
		code VARCHAR(200),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not create table: ", err)
		return err
	}
	return nil
}

// ============================================================================
// SCS: PICKUPS, PRODUCTS
// ============================================================================

func CreatePickupRequestTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS pickup_requests (
		id SERIAL PRIMARY KEY,
		request_id VARCHAR(50) NOT NULL UNIQUE CHECK (request_id <> ''),
		sales_associate_id VARCHAR(50) NOT NULL CHECK (sales_associate_id <> ''),
		distributor_id VARCHAR(50) NOT NULL CHECK (distributor_id <> ''),
		confirmed BOOL NOT NULL DEFAULT FALSE CHECK (confirmed IN (TRUE, FALSE)),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not create table: ", err)
		return err
	}
	return nil
}

func CreatePickupRequestItemsTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS pickup_request_items (
		id SERIAL PRIMARY KEY,
		request_id VARCHAR(50) NOT NULL REFERENCES pickup_requests(request_id) ON DELETE CASCADE,
		sku VARCHAR(50) NOT NULL CHECK (sku <> ''),
		name VARCHAR(150) NOT NULL CHECK (name <> ''),
		quantity INT NOT NULL CHECK (quantity > 0),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not create table: ", err)
		return err
	}
	return nil
}

func CreateProductTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS products (
		id SERIAL PRIMARY KEY,
		sku VARCHAR(50) NOT NULL UNIQUE CHECK (sku <> ''),
		name VARCHAR(150) NOT NULL CHECK (name <> ''),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not create table: ", err)
		return err
	}
	return nil
}

// AlterProductTableAddPrice adds a unit price to products, needed for
// invoice total calculation. Kept as an ALTER (matching AlterUsersTable
// style) rather than baking into CreateProductTable, since products may
// already exist in production without it.
func AlterProductTableAddPrice(db *sql.DB) error {
	query := `
		ALTER TABLE products
		ADD COLUMN IF NOT EXISTS price NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (price >= 0);
	`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not alter table: ", err)
		return err
	}
	return nil
}

// ============================================================================
// SCS: INVOICING
// ============================================================================

// CreateInvoiceTable stores invoices generated on distributor confirmation.
// due_at is set at creation time (created_at + 7 days) so the overdue check
// is a simple comparison rather than a recalculation every run.
func CreateInvoiceTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS invoices (
		id SERIAL PRIMARY KEY,
		invoice_id VARCHAR(50) NOT NULL UNIQUE CHECK (invoice_id <> ''),
		request_id VARCHAR(50) NOT NULL REFERENCES pickup_requests(request_id),
		total_value NUMERIC(12,2) NOT NULL CHECK (total_value >= 0),
		outstanding_value NUMERIC(12,2) NOT NULL CHECK (outstanding_value >= 0),
		status VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'paid', 'overdue')),
		due_at TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not create table: ", err)
		return err
	}
	return nil
}

func CreateReceiptTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS receipts (
		id SERIAL PRIMARY KEY,
		receipt_id VARCHAR(50) NOT NULL UNIQUE CHECK (receipt_id <> ''),
		invoice_id VARCHAR(50) NOT NULL REFERENCES invoices(invoice_id),
		amount_paid NUMERIC(12,2) NOT NULL CHECK (amount_paid > 0),
		paid_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not create table: ", err)
		return err
	}
	return nil
}

// ============================================================================
// MASTER DATA: OUTLETS, ROUTES, DISTRIBUTORS, REGIONS, SETTINGS
// ============================================================================

// CreateOutletTable stores outlet master data. outlet_id must never be
// reused/deleted-and-recreated — historical sales stay attached to it via
// FK, so outlets are deactivated (active = FALSE) rather than dropped.
func CreateOutletTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS outlets (
		id SERIAL PRIMARY KEY,
		outlet_id VARCHAR(50) NOT NULL UNIQUE CHECK (outlet_id <> ''),
		name VARCHAR(150) NOT NULL CHECK (name <> ''),
		address VARCHAR(255),
		outlet_type VARCHAR(50),
		phone VARCHAR(20),
		latitude DOUBLE PRECISION NOT NULL,
		longitude DOUBLE PRECISION NOT NULL,
		area VARCHAR(100),
		zone VARCHAR(100),
		assigned_sales_associate_id VARCHAR(50) REFERENCES users(user_id),
		route_day VARCHAR(10) CHECK (route_day IN ('Monday','Tuesday','Wednesday','Thursday','Friday','Saturday')),
		priority VARCHAR(20),
		active BOOL NOT NULL DEFAULT TRUE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not create table: ", err)
		return err
	}
	return nil
}

// CreateDistributorAssignmentTable records which distributors a sales
// associate is "covered" by — needed for the outside-coverage % rule
// (pickups from unassigned distributors are allowed, but tracked).
func CreateDistributorAssignmentTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS distributor_assignments (
		id SERIAL PRIMARY KEY,
		sales_associate_id VARCHAR(50) NOT NULL REFERENCES users(user_id),
		distributor_id VARCHAR(50) NOT NULL REFERENCES users(user_id),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (sales_associate_id, distributor_id)
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not create table: ", err)
		return err
	}
	return nil
}

// CreateRoutePlanTable stores one row per associate/day/outlet with a
// sequence position. Admin approval happens per-plan via the approved flag.
func CreateRoutePlanTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS route_plans (
		id SERIAL PRIMARY KEY,
		sales_associate_id VARCHAR(50) NOT NULL REFERENCES users(user_id),
		outlet_id VARCHAR(50) NOT NULL REFERENCES outlets(outlet_id),
		route_day VARCHAR(10) NOT NULL CHECK (route_day IN ('Monday','Tuesday','Wednesday','Thursday','Friday','Saturday')),
		sequence INT NOT NULL CHECK (sequence > 0),
		approved BOOL NOT NULL DEFAULT FALSE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (sales_associate_id, route_day, outlet_id)
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not create table: ", err)
		return err
	}
	return nil
}

// CreateRegionTable seeds the region codes used to suffix transaction IDs
// (REQ-260816-000001-LG, etc.). Lagos (LG) is the initial region.
func CreateRegionTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS regions (
		code VARCHAR(5) PRIMARY KEY,
		name VARCHAR(100) NOT NULL
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not create table: ", err)
		return err
	}

	seed := `
	INSERT INTO regions (code, name) VALUES
		('LG', 'Lagos'),
		('SW', 'South West'),
		('SE', 'South East'),
		('SS', 'South South'),
		('NC', 'North Central'),
		('NE', 'North East'),
		('NW', 'North West')
	ON CONFLICT (code) DO NOTHING;
	`
	if _, err := db.Exec(seed); err != nil {
		log.Fatal("Could not seed regions table: ", err)
		return err
	}

	return nil
}

// CreateSettingTable seeds configurable thresholds (resumption radius,
// outlet geofence radius, outside-coverage %, overdue days) so they can be
// changed from the admin dashboard instead of redeploying hardcoded values.
func CreateSettingTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS settings (
		key VARCHAR(100) PRIMARY KEY,
		value VARCHAR(255) NOT NULL,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not create table: ", err)
		return err
	}

	seed := `
	INSERT INTO settings (key, value) VALUES
		('resumption_radius_km', '1'),
		('outlet_geofence_m', '250'),
		('outside_coverage_pct', '30'),
		('invoice_overdue_days', '7')
	ON CONFLICT (key) DO NOTHING;
	`
	if _, err := db.Exec(seed); err != nil {
		log.Fatal("Could not seed settings table: ", err)
		return err
	}

	return nil
}

// ============================================================================
// FIELD ACTIVITY: RESUMPTION, VISITS, SALES
// ============================================================================

// CreateResumptionLogTable stores the daily GPS resumption check result.
// One row per associate per date — a second resumption attempt on the same
// day updates, it doesn't duplicate (enforced at the query layer via the
// unique constraint below).
func CreateResumptionLogTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS resumption_log (
		id SERIAL PRIMARY KEY,
		sales_associate_id VARCHAR(50) NOT NULL REFERENCES users(user_id),
		route_day VARCHAR(10) NOT NULL,
		date DATE NOT NULL,
		time TIME NOT NULL,
		latitude DOUBLE PRECISION NOT NULL,
		longitude DOUBLE PRECISION NOT NULL,
		distance_to_route_m DOUBLE PRECISION NOT NULL,
		result VARCHAR(10) NOT NULL CHECK (result IN ('PASS', 'FAIL')),
		device_ref VARCHAR(100),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (sales_associate_id, date)
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not create table: ", err)
		return err
	}
	return nil
}

func CreateOutletVisitTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS outlet_visits (
		id SERIAL PRIMARY KEY,
		sales_associate_id VARCHAR(50) NOT NULL REFERENCES users(user_id),
		outlet_id VARCHAR(50) NOT NULL REFERENCES outlets(outlet_id),
		route_day VARCHAR(10) NOT NULL,
		visited_at TIMESTAMP WITH TIME ZONE NOT NULL,
		latitude DOUBLE PRECISION NOT NULL,
		longitude DOUBLE PRECISION NOT NULL,
		distance_from_outlet_m DOUBLE PRECISION NOT NULL,
		geofence_status VARCHAR(10) NOT NULL CHECK (geofence_status IN ('PASS', 'FAIL')),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not create table: ", err)
		return err
	}
	return nil
}

// CreateSaleTable stores outlet-level sales captured on the APK.
// transaction_id doubles as the idempotency key — a retried offline sync
// with the same transaction_id is rejected by the UNIQUE constraint rather
// than creating a duplicate sale.
func CreateSaleTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS sales (
		id SERIAL PRIMARY KEY,
		transaction_id VARCHAR(50) NOT NULL UNIQUE CHECK (transaction_id <> ''),
		sales_associate_id VARCHAR(50) NOT NULL REFERENCES users(user_id),
		outlet_id VARCHAR(50) NOT NULL REFERENCES outlets(outlet_id),
		sku VARCHAR(50) NOT NULL,
		quantity INT NOT NULL CHECK (quantity > 0),
		unit_value NUMERIC(12,2) NOT NULL CHECK (unit_value >= 0),
		total_value NUMERIC(12,2) NOT NULL CHECK (total_value >= 0),
		latitude DOUBLE PRECISION NOT NULL,
		longitude DOUBLE PRECISION NOT NULL,
		distance_from_outlet_m DOUBLE PRECISION NOT NULL,
		geofence_status VARCHAR(10) NOT NULL CHECK (geofence_status IN ('PASS', 'FAIL')),
		route_day VARCHAR(10),
		synced_at TIMESTAMP WITH TIME ZONE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not create table: ", err)
		return err
	}
	return nil
}

// ============================================================================
// AUDIT
// ============================================================================

func CreateAuditLogTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS audit_log (
		id SERIAL PRIMARY KEY,
		actor_id VARCHAR(50) NOT NULL REFERENCES users(user_id),
		action VARCHAR(100) NOT NULL,
		target VARCHAR(100),
		details TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Could not create table: ", err)
		return err
	}
	return nil
}
