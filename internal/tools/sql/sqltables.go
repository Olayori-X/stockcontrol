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

	CreateUserTable(dbpointer)
	CreateLoggedInUserTable(dbpointer)
	CreateForgotPasswordTable(dbpointer)
	CreatePickupRequestTable(dbpointer)
	CreatePickupRequestItemsTable(dbpointer)
	CreateProductTable(dbpointer)
	// AlterUsersTable(dbpointer)
	// DeleteUserTable(dbpointer)
	return nil
}

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
