package sqltools

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/Olayori-X/stock-control-backend/functions"
	"github.com/Olayori-X/stock-control-backend/models"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

// type mockDB struct{}

type RealDB struct {
	DB *sql.DB
}

func UserExists(db *RealDB, username string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	err := db.DB.QueryRow(query, username).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (db *RealDB) UserLoggedIn(userid string) *AuthenticatedUser {
	query := `
	SELECT user_id, code, role, created_at
	FROM loggedin_users
	WHERE user_id = $1;`

	var userID, code, role string
	var loginTime time.Time

	err := db.DB.QueryRow(query, userid).Scan(&userID, &code, &role, &loginTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return nil
	}

	return &AuthenticatedUser{
		UserID:    userID,
		Code:      code,
		Role:      role,
		LoginTime: loginTime,
	}
}

func (db *RealDB) UpsertLoggedInUser(userID string, code string, role string) error {
	query := `
	INSERT INTO loggedin_users (user_id, code, role)
	VALUES ($1, $2, $3)
	ON CONFLICT (user_id)
	DO UPDATE SET code = EXCLUDED.code, role = EXCLUDED.role, created_at = CURRENT_TIMESTAMP;
	`

	_, err := db.DB.Exec(query, userID, code, role)
	if err != nil {
		return err
	}
	return nil
}

func (db *RealDB) GetUserLoginDetails(email string) *LoginDetails {
	query := `
	SELECT user_id, password, role, verified
	FROM users
	WHERE email = $1;`

	var user_id, password, role string
	var verified bool

	err := db.DB.QueryRow(query, email).Scan(&user_id, &password, &role, &verified)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return nil
	}
	return &LoginDetails{
		UserID:   user_id,
		Password: password,
		Role:     role,
		Verified: verified,
	}
}

func (db *RealDB) GetUserDetails(userid string) *models.User {
	query := `
	SELECT user_id, name, email, phone, role, password,
	       created_at, updated_at
	FROM users
	WHERE user_id = $1;`

	var user models.User

	err := db.DB.QueryRow(query, userid).Scan(
		&user.UserID,
		&user.Name,
		&user.Email,
		&user.Phone,
		&user.Role,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return nil
	}
	return &user
}

func (db *RealDB) AddUser(user models.User) (string, error) {
	// Check if user already exists
	exists, err := UserExists(db, user.Email)
	if err != nil {
		return "", fmt.Errorf("error checking user: %w", err)
	}
	if exists {
		return "", fmt.Errorf("user '%s' already exists", user.Email)
	}

	query := `
	INSERT INTO users (user_id, name, email, phone, role, verified, password)
	VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING user_id;`

	userid, err := functions.GenerateUserID()
	if err != nil {
		// return fmt.Errorf("could not generate user ID: %w", err)
		return "", err
	}

	var pk string
	err = db.DB.QueryRow(query, userid, user.Name, user.Email, user.Phone, user.Role, true, user.Password).Scan(&pk)

	if err != nil {
		return "", err
	}

	return pk, nil
}

func (db *RealDB) UpdateUserCode(userID string, hashedCode string) error {
	_, err := db.DB.Exec(`
		UPDATE users
		SET code = $1, updated_at = NOW()
		WHERE user_id = $2
	`, hashedCode, userID)
	if err != nil {
		return fmt.Errorf("error updating user code: %w", err)
	}
	return nil
}

func (db *RealDB) UpdateUserProfile(user models.User) error {
	query := `
		UPDATE users
		SET name = $1,
			email = $2,
			phone = $3,
			updated_at = $4
		WHERE user_id = $5;
	`

	_, err := db.DB.Exec(query,
		user.Name,
		user.Email,
		user.Phone,
		time.Now(),
		user.UserID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user profile: %w", err)
	}

	return nil
}

func (db *RealDB) AddForgotPasswordRecord(email, code string) error {
	exists, err := UserExists(db, email)
	if err != nil {
		return fmt.Errorf("error checking user: %w", err)
	}
	if !exists {
		return fmt.Errorf("'%s' does not exist", email)
	}

	query := `
		INSERT INTO forgotpassword (user_id, code)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET
			code = EXCLUDED.code,
			created_at = CURRENT_TIMESTAMP;`

	_, err = db.DB.Exec(query, email, code)
	if err != nil {
		return fmt.Errorf("failed to add forgot password record: %w", err)
	}
	return nil
}

func (db *RealDB) ChangeUserPassword(email string, hashedPassword string) error {
	query := `
		UPDATE users
		SET password = $1
		WHERE email = $2;
	`

	_, err := db.DB.Exec(query, hashedPassword, email)
	if err != nil {
		return fmt.Errorf("could not change password: %w", err)
	}

	// Optionally clear the forgot password code
	clearCodeQuery := `
		UPDATE forgotpassword
		SET code = NULL
		WHERE email = $1;
	`
	_, err = db.DB.Exec(clearCodeQuery, email)
	// if err != nil {
	// 	log.Printf("Error clearing forgot password code for user %s: %v", email, err)
	// 	// Don't return this as critical — just log it.
	// }

	return nil
}

func (db *RealDB) SearchUsers(query string, excludeID string) ([]models.User, error) {
	users := make([]models.User, 0)

	rows, err := db.DB.Query(`
		SELECT
			user_id, name
		FROM users
		WHERE LOWER(name) LIKE LOWER($1)
		AND user_id != $2
		AND verified = TRUE
		LIMIT 20
	`, "%"+query+"%", excludeID)

	if err != nil {
		return nil, fmt.Errorf("error searching users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var u models.User
		if err := rows.Scan(
			&u.UserID, &u.Name,
		); err != nil {
			return nil, fmt.Errorf("error scanning user row: %w", err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return users, nil
}

func (db *RealDB) SearchDistributors(query string) ([]models.User, error) {
	users := make([]models.User, 0)

	rows, err := db.DB.Query(`
		SELECT
			user_id, name, email, phone, role
		FROM users
		WHERE LOWER(name) LIKE LOWER($1)
		AND verified = TRUE
		AND role = 'distributor'
		LIMIT 20
	`, "%"+query+"%")

	if err != nil {
		return nil, fmt.Errorf("error searching users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var u models.User
		if err := rows.Scan(
			&u.UserID, &u.Name, &u.Email, &u.Phone, &u.Role,
		); err != nil {
			return nil, fmt.Errorf("error scanning user row: %w", err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return users, nil
}

// CreatePickupRequest inserts a pickup request and its line items in a single transaction.
func (db *RealDB) CreatePickupRequest(req *models.PickupRequest) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback() // no-op if committed

	insertRequest := `
		INSERT INTO pickup_requests (request_id, sales_associate_id, distributor_id)
		VALUES ($1, $2, $3)
		RETURNING created_at, updated_at;
	`
	err = tx.QueryRow(
		insertRequest,
		req.RequestID,
		req.SalesAssociateID,
		req.DistributorID,
	).Scan(&req.CreatedAt, &req.UpdatedAt)
	if err != nil {
		return fmt.Errorf("could not create pickup request: %w", err)
	}

	insertItem := `
		INSERT INTO pickup_request_items (request_id, sku, name, quantity)
		VALUES ($1, $2, $3, $4);
	`
	for _, item := range req.Products {
		_, err = tx.Exec(insertItem, req.RequestID, item.SKU, item.Name, item.Quantity)
		if err != nil {
			return fmt.Errorf("could not insert product item (sku=%s): %w", item.SKU, err)
		}
	}

	// Fetch names while the tx is still open — cheap lookups, same connection.
	var salesAssociateName, distributorName string
	err = tx.QueryRow(`SELECT name FROM users WHERE user_id = $1`, req.SalesAssociateID).Scan(&salesAssociateName)
	if err != nil {
		return fmt.Errorf("could not fetch sales associate name: %w", err)
	}
	err = tx.QueryRow(`SELECT name FROM users WHERE user_id = $1`, req.DistributorID).Scan(&distributorName)
	if err != nil {
		return fmt.Errorf("could not fetch distributor name: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction: %w", err)
	}

	// DB write succeeded — the pickup request now exists regardless of what
	// happens below. A Sheets failure is logged, not returned, so it doesn't
	// turn a successful request into a 500.
	if err := appendPickupRequestToSheet(req, salesAssociateName, distributorName); err != nil {
		log.Error("failed to append pickup request to Google Sheet: ", err)
	}

	return nil
}

var (
	sheetsClient     *sheets.Service
	sheetsClientOnce sync.Once
	sheetsClientErr  error
)

// getSheetsClient returns a shared Sheets client, creating it once on first
// use and reusing it for every subsequent call — avoids reconnecting on
// every pickup request created or confirmed.
func getSheetsClient() (*sheets.Service, error) {
	sheetsClientOnce.Do(func() {
		credsJSON := os.Getenv("GOOGLE_SHEETS_CREDENTIALS_JSON")
		if credsJSON == "" {
			sheetsClientErr = fmt.Errorf("GOOGLE_SHEETS_CREDENTIALS_JSON is not set")
			return
		}

		ctx := context.Background()
		client, err := sheets.NewService(ctx, option.WithCredentialsJSON([]byte(credsJSON)))
		if err != nil {
			sheetsClientErr = fmt.Errorf("could not create sheets client: %w", err)
			return
		}

		sheetsClient = client
	})

	return sheetsClient, sheetsClientErr
}

func appendPickupRequestToSheet(req *models.PickupRequest, salesAssociateName, distributorName string) error {
	srv, err := getSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := os.Getenv("PICKUP_REQUESTS_SPREADSHEET_ID")
	if spreadsheetID == "" {
		return fmt.Errorf("PICKUP_REQUESTS_SPREADSHEET_ID is not set")
	}

	// One row per product so each SKU/quantity is visible in the sheet.
	var rows [][]interface{}
	for _, item := range req.Products {
		rows = append(rows, []interface{}{
			req.RequestID,
			salesAssociateName,
			distributorName,
			item.SKU,
			item.Name,
			strconv.Itoa(item.Quantity),
			req.Confirmed,
			req.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	valueRange := &sheets.ValueRange{Values: rows}

	_, err = srv.Spreadsheets.Values.Append(
		spreadsheetID, "Sheet1!A1",
		valueRange,
	).ValueInputOption("RAW").InsertDataOption("INSERT_ROWS").Do()
	if err != nil {
		return fmt.Errorf("could not append to sheet: %w", err)
	}

	return nil
}

func (db *RealDB) ConfirmPickupRequest(requestID, distributorID string) (bool, error) {
	query := `
		UPDATE pickup_requests
		SET confirmed = TRUE, updated_at = CURRENT_TIMESTAMP
		WHERE request_id = $1 AND distributor_id = $2 AND confirmed = FALSE;
	`

	result, err := db.DB.Exec(query, requestID, distributorID)
	if err != nil {
		return false, fmt.Errorf("could not confirm pickup request: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("could not check rows affected: %w", err)
	}

	confirmed := rowsAffected > 0

	// DB is source of truth and is already updated at this point.
	// A Sheets failure is logged, not returned, so it doesn't turn a
	// successful confirmation into an error response.
	if confirmed {
		if err := updateConfirmedInSheet(requestID); err != nil {
			log.Error("failed to update confirmed status in Google Sheet: ", err)
		}
	}

	return confirmed, nil
}

func updateConfirmedInSheet(requestID string) error {
	srv, err := getSheetsClient()
	if err != nil {
		return err
	}

	spreadsheetID := os.Getenv("PICKUP_REQUESTS_SPREADSHEET_ID")
	if spreadsheetID == "" {
		return fmt.Errorf("PICKUP_REQUESTS_SPREADSHEET_ID is not set")
	}

	// Column A holds request_id, column G holds confirmed — matches the
	// row layout written in appendPickupRequestToSheet.
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetID, "Sheet1!A:A").Do()
	if err != nil {
		return fmt.Errorf("could not read sheet to find request rows: %w", err)
	}

	var matchingRows []int
	for i, row := range resp.Values {
		if len(row) == 0 {
			continue
		}
		if cell, ok := row[0].(string); ok && cell == requestID {
			matchingRows = append(matchingRows, i+1) // sheet rows are 1-indexed
		}
	}

	if len(matchingRows) == 0 {
		return fmt.Errorf("no rows found in sheet for request_id %s", requestID)
	}

	var data []*sheets.ValueRange
	for _, rowNum := range matchingRows {
		data = append(data, &sheets.ValueRange{
			Range:  fmt.Sprintf("Sheet1!G%d", rowNum),
			Values: [][]interface{}{{true}},
		})
	}

	batchUpdate := &sheets.BatchUpdateValuesRequest{
		ValueInputOption: "RAW",
		Data:             data,
	}

	_, err = srv.Spreadsheets.Values.BatchUpdate(spreadsheetID, batchUpdate).Do()
	if err != nil {
		return fmt.Errorf("could not batch update confirmed status: %w", err)
	}

	return nil
}

func (db *RealDB) GetPendingPickupRequests(distributorID string) ([]models.PendingPickupRequest, error) {
	query := `
		SELECT pr.request_id, pr.sales_associate_id, pr.distributor_id,
		       pr.confirmed, pr.created_at, pr.updated_at, u.name
		FROM pickup_requests pr
		JOIN users u ON u.user_id = pr.sales_associate_id
		WHERE pr.distributor_id = $1 AND pr.confirmed = FALSE
		ORDER BY pr.created_at DESC;
	`

	rows, err := db.DB.Query(query, distributorID)
	if err != nil {
		return nil, fmt.Errorf("could not fetch pending pickup requests: %w", err)
	}
	defer rows.Close()

	requests := []models.PendingPickupRequest{}
	var requestIDs []string

	for rows.Next() {
		var req models.PendingPickupRequest
		if err := rows.Scan(
			&req.RequestID,
			&req.SalesAssociateID,
			&req.DistributorID,
			&req.Confirmed,
			&req.CreatedAt,
			&req.UpdatedAt,
			&req.SalesAssociateName,
		); err != nil {
			return nil, fmt.Errorf("could not scan pickup request row: %w", err)
		}
		requests = append(requests, req)
		requestIDs = append(requestIDs, req.RequestID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pickup request rows: %w", err)
	}

	if len(requests) == 0 {
		return requests, nil
	}

	// Fetch all line items for these requests in one batched query,
	// then group them back onto each request in memory.
	itemsQuery := `
		SELECT request_id, sku, name, quantity
		FROM pickup_request_items
		WHERE request_id = ANY($1);
	`

	itemRows, err := db.DB.Query(itemsQuery, pq.Array(requestIDs))
	if err != nil {
		return nil, fmt.Errorf("could not fetch pickup request items: %w", err)
	}
	defer itemRows.Close()

	itemsByRequest := make(map[string][]models.ProductItem)
	for itemRows.Next() {
		var requestID string
		var item models.ProductItem
		if err := itemRows.Scan(&requestID, &item.SKU, &item.Name, &item.Quantity); err != nil {
			return nil, fmt.Errorf("could not scan pickup request item row: %w", err)
		}
		itemsByRequest[requestID] = append(itemsByRequest[requestID], item)
	}
	if err := itemRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pickup request item rows: %w", err)
	}

	for i := range requests {
		requests[i].Products = itemsByRequest[requests[i].RequestID]
	}

	return requests, nil
}

func (db *RealDB) GetUnacceptedPickupRequests(salesAssociateID string) ([]models.PendingPickupRequest, error) {
	query := `
		SELECT pr.request_id, pr.sales_associate_id, pr.distributor_id,
		       pr.confirmed, pr.created_at, pr.updated_at, u.name
		FROM pickup_requests pr
		JOIN users u ON u.user_id = pr.distributor_id
		WHERE pr.sales_associate_id = $1 AND pr.confirmed = FALSE
		ORDER BY pr.created_at DESC;
	`

	rows, err := db.DB.Query(query, salesAssociateID)
	if err != nil {
		return nil, fmt.Errorf("could not fetch pending pickup requests: %w", err)
	}
	defer rows.Close()

	requests := []models.PendingPickupRequest{}
	var requestIDs []string

	for rows.Next() {
		var req models.PendingPickupRequest
		if err := rows.Scan(
			&req.RequestID,
			&req.SalesAssociateID,
			&req.DistributorID,
			&req.Confirmed,
			&req.CreatedAt,
			&req.UpdatedAt,
			&req.SalesAssociateName,
		); err != nil {
			return nil, fmt.Errorf("could not scan pickup request row: %w", err)
		}
		requests = append(requests, req)
		requestIDs = append(requestIDs, req.RequestID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pickup request rows: %w", err)
	}

	if len(requests) == 0 {
		return requests, nil
	}

	// Fetch all line items for these requests in one batched query,
	// then group them back onto each request in memory.
	itemsQuery := `
		SELECT request_id, sku, name, quantity
		FROM pickup_request_items
		WHERE request_id = ANY($1);
	`

	itemRows, err := db.DB.Query(itemsQuery, pq.Array(requestIDs))
	if err != nil {
		return nil, fmt.Errorf("could not fetch pickup request items: %w", err)
	}
	defer itemRows.Close()

	itemsByRequest := make(map[string][]models.ProductItem)
	for itemRows.Next() {
		var requestID string
		var item models.ProductItem
		if err := itemRows.Scan(&requestID, &item.SKU, &item.Name, &item.Quantity); err != nil {
			return nil, fmt.Errorf("could not scan pickup request item row: %w", err)
		}
		itemsByRequest[requestID] = append(itemsByRequest[requestID], item)
	}
	if err := itemRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pickup request item rows: %w", err)
	}

	for i := range requests {
		requests[i].Products = itemsByRequest[requests[i].RequestID]
	}

	return requests, nil
}

func (db *RealDB) AddProduct(product models.Products) error {
	query := `
		INSERT INTO products (sku, name, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`
	_, err := db.DB.Exec(query, product.SKU, product.Name)
	if err != nil {
		return fmt.Errorf("AddProduct: %w", err)
	}
	return nil
}

func (db *RealDB) GetProducts() ([]models.Products, error) {
	query := `
		SELECT sku, name, created_at, updated_at
		FROM products
		ORDER BY created_at DESC
	`
	rows, err := db.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("GetProducts: %w", err)
	}
	defer rows.Close()

	var products []models.Products

	for rows.Next() {
		var p models.Products
		if err := rows.Scan(&p.SKU, &p.Name, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("GetProducts scan: %w", err)
		}
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetProducts rows: %w", err)
	}

	return products, nil
}

func (db *RealDB) GetProductBySKU(sku string) (*models.Products, error) {
	query := `
		SELECT sku, name, created_at, updated_at
		FROM products
		WHERE sku = $1
	`
	var p models.Products
	err := db.DB.QueryRow(query, sku).Scan(&p.SKU, &p.Name, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetProductBySKU: %w", err)
	}
	return &p, nil
}

func (db *RealDB) EditProduct(sku string, updated models.Products) error {
	query := `
		UPDATE products
		SET name = $1, updated_at = NOW()
		WHERE sku = $2
	`
	result, err := db.DB.Exec(query, updated.Name, sku)
	if err != nil {
		return fmt.Errorf("EditProduct: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("EditProduct: no product found with SKU %s", sku)
	}
	return nil
}

func (db *RealDB) DeleteProduct(sku string) error {
	query := `DELETE FROM products WHERE sku = $1`
	result, err := db.DB.Exec(query, sku)
	if err != nil {
		return fmt.Errorf("DeleteProduct: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("DeleteProduct: no product found with SKU %s", sku)
	}
	return nil
}

func (db *RealDB) GetUsers() ([]models.User, error) {
	query := `
	SELECT 
		user_id, 
		name, 
		email, 
		phone,
		role,
		password,
		created_at, 
		updated_at
	FROM users;
	`

	rows, err := db.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		err := rows.Scan(
			&u.UserID,
			&u.Name,
			&u.Email,
			&u.Phone,
			&u.Role,
			&u.Password,
			&u.CreatedAt,
			&u.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
