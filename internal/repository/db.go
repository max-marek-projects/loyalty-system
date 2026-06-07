package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/max-marek-projects/loyalty-system/internal/config/db"
	"github.com/max-marek-projects/loyalty-system/internal/logger"
	"github.com/max-marek-projects/loyalty-system/internal/models"
	"go.uber.org/zap"

	_ "github.com/golang-migrate/migrate/v4/database/postgres" // required for migrations
	_ "github.com/golang-migrate/migrate/v4/source/file"       // required for migrations
)

type dbStorage struct {
	storage *sql.DB
	config  *db.DBConf
}

func NewDBStorage(dbURL string) (*dbStorage, error) {
	config := db.NewDbConf(dbURL)
	storage, err := db.Connect(config)
	if err != nil {
		return nil, fmt.Errorf("Failed to create DB storage: %w", err)
	}
	dbs := &dbStorage{
		storage: storage,
		config:  config,
	}
	err = dbs.runMigrations()
	if err != nil {
		return nil, fmt.Errorf("Failed to create DB storage: %w", err)
	}
	return dbs, nil
}

// run database migrations
func (dbs *dbStorage) runMigrations() error {
	logger.Log.Info("Running migrations", zap.String("path", dbs.config.MigrationsPath))
	m, err := migrate.New(
		"file://"+dbs.config.MigrationsPath,
		dbs.config.URL,
	)
	if err != nil {
		return fmt.Errorf("Failed to run migrations: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("Failed to run migrations: %w", err)
	}
	return nil
}

// register user
func (dbs *dbStorage) RegisterUser(ctx context.Context, userData models.UserData) (int64, error) {
	var userID int64
	query := `--sql
        INSERT INTO users (username, password_hash)
		VALUES ($1, $2)
		ON CONFLICT (username) DO NOTHING
		RETURNING id;
	`
	err := dbs.storage.QueryRowContext(ctx, query, userData.Login, userData.PasswordHash).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrorAlreadyInStorage
		}
		return 0, fmt.Errorf("Failed to add user to storage: %w", err)
	}
	return userID, nil
}

// check user is present in storage
func (dbs *dbStorage) CheckUser(ctx context.Context, username string) (int64, string, error) {
	var userID int64
	var hashedPassword string
	query := `--sql
        SELECT id, password_hash FROM users
		WHERE username = $1;
	`
	err := dbs.storage.QueryRowContext(ctx, query, username).Scan(&userID, &hashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", ErrorUserNotFound
		}
		return 0, "", fmt.Errorf("Failed to get user from storage by id: %w", err)
	}
	return userID, hashedPassword, nil
}

// add new order to storage
func (dbs *dbStorage) AddOrder(ctx context.Context, userID int64, orderNumber string) error {
	var orderID int64
	query := `--sql
        INSERT INTO orders (number, user_id, status)
		VALUES ($1, $2, $3)
		ON CONFLICT (number) DO NOTHING
		RETURNING id;
	`
	err := dbs.storage.QueryRowContext(ctx, query, orderNumber, userID, models.StatusNEW).Scan(&orderID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("Failed to add order to storage: %w", err)
	}
	var recordUserId int64
	query = `--sql
		SELECT user_id FROM orders 
		WHERE number = $1;
	`
	err = dbs.storage.QueryRowContext(ctx, query, orderNumber).Scan(&recordUserId)
	if err != nil {
		return fmt.Errorf("Failed to select order data from storage: %w", err)
	}
	if recordUserId != userID {
		return ErrorStorageConflict
	}
	return ErrorAlreadyInStorage
}

// get all orders by user id
func (dbs *dbStorage) GetAllOrders(ctx context.Context, userID int64) ([]models.OrderData, error) {
	query := `SELECT id, number, status, accrual, uploaded_at FROM orders WHERE user_id = $1`
	rows, err := dbs.storage.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get all orders from database: %v", err)
	}
	defer rows.Close()
	var res []models.OrderData
	for rows.Next() {
		var orderData models.OrderData
		if err := rows.Scan(&orderData.Id, &orderData.Number, &orderData.Status, &orderData.Accrual, &orderData.UploadedAt); err != nil {
			return nil, err
		}
		res = append(res, orderData)
	}
	return res, nil
}

// get all orders by user id
func (dbs *dbStorage) MarkAllProcessingAsNew(ctx context.Context) error {
	query := `UPDATE orders
		SET status = $1,
			pending = FALSE
		WHERE status = $2 OR pending = TRUE;
	`
	result, err := dbs.storage.ExecContext(ctx, query, models.StatusNEW, models.StatusPROCESSING)
	if err != nil {
		return fmt.Errorf("Failed to reset all statuses: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Could not get affected rows: %v", err)
	}
	logger.Log.Info("Successfully reset statuses.", zap.Int64("rowsAffected", rowsAffected))
	return nil
}

// find new orders and mark them as pending
func (dbs *dbStorage) FindAndClaimNewOrders(ctx context.Context) ([]models.OrderData, error) {
	query := `UPDATE orders
		SET pending = TRUE
		WHERE status = $1 AND pending = FALSE
		RETURNING id, number, status, accrual, uploaded_at;
	`
	rows, err := dbs.storage.QueryContext(ctx, query, models.StatusNEW)
	if err != nil {
		return nil, fmt.Errorf("Failed to get all orders from database: %v", err)
	}
	defer rows.Close()
	var res []models.OrderData
	for rows.Next() {
		var orderData models.OrderData
		if err := rows.Scan(&orderData.Id, &orderData.Number, &orderData.Status, &orderData.Accrual, &orderData.UploadedAt); err != nil {
			return nil, err
		}
		res = append(res, orderData)
	}
	return res, nil
}
