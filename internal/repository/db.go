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
	config := db.NewDBConf(dbURL)
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
			return 0, ErrAlreadyInStorage
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
			return 0, "", ErrUserNotFound
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
	var recordUserID int64
	query = `--sql
		SELECT user_id FROM orders 
		WHERE number = $1;
	`
	err = dbs.storage.QueryRowContext(ctx, query, orderNumber).Scan(&recordUserID)
	if err != nil {
		return fmt.Errorf("Failed to select order data from storage: %w", err)
	}
	if recordUserID != userID {
		return ErrStorageConflict
	}
	return ErrAlreadyInStorage
}

// get all orders by user id
func (dbs *dbStorage) GetAllOrders(ctx context.Context, userID int64) ([]models.OrderData, error) {
	query := `SELECT id, number, status, accrual, uploaded_at FROM orders WHERE user_id = $1 ORDER BY uploaded_at DESC`
	rows, err := dbs.storage.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get all orders from database: %v", err)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("Failed to get all orders from database: %v", err)
	}
	defer rows.Close()
	var res []models.OrderData
	for rows.Next() {
		var orderData models.OrderData
		if err := rows.Scan(&orderData.ID, &orderData.Number, &orderData.Status, &orderData.Accrual, &orderData.UploadedAt); err != nil {
			return nil, err
		}
		res = append(res, orderData)
	}
	return res, nil
}

// get balance by user id
func (dbs *dbStorage) GetBalance(ctx context.Context, userID int64) (*models.BalanceData, error) {
	balanceData := &models.BalanceData{}
	query := `SELECT balance, withdrawn FROM users WHERE id = $1`
	err := dbs.storage.QueryRowContext(ctx, query, userID).Scan(&balanceData.Current, &balanceData.Withdrawn)
	if err != nil {
		return nil, fmt.Errorf("Failed to get all orders from database: %v", err)
	}
	return balanceData, nil
}

// withdraw sum by order number
func (dbs *dbStorage) Withdraw(ctx context.Context, userID int64, orderNumber string, sum float64) error {
	tx, err := dbs.storage.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start database transaction: %w", err)
	}
	defer tx.Rollback()
	var currentBalance float64
	err = tx.QueryRowContext(ctx,
		`SELECT balance FROM users WHERE id = $1 FOR UPDATE`,
		userID,
	).Scan(&currentBalance)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to get user balance: %w", err)
	}
	if currentBalance < sum {
		return ErrInsufficientFunds
	}
	_, err = tx.ExecContext(ctx,
		`UPDATE users 
         	SET balance = balance - $1,
            	withdrawn = withdrawn + $1
         	WHERE id = $2`,
		sum, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO withdrawals (user_id, order_number, sum, processed_at)
         	VALUES ($1, $2, $3, NOW())`,
		userID, orderNumber, sum,
	)
	if err != nil {
		return fmt.Errorf("failed to insert withdrawal: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// get all withdrawals by user id
func (dbs *dbStorage) GetAllWithdrawals(ctx context.Context, userID int64) ([]models.WithdrawData, error) {
	query := `SELECT order_number, sum, processed_at FROM withdrawals WHERE user_id = $1 ORDER BY processed_at DESC`
	rows, err := dbs.storage.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get all orders from database: %v", err)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("Failed to get all orders from database: %v", err)
	}
	defer rows.Close()
	var res []models.WithdrawData
	for rows.Next() {
		var withdrawal models.WithdrawData
		if err := rows.Scan(&withdrawal.Order, &withdrawal.Sum, &withdrawal.ProcessedAt); err != nil {
			return nil, err
		}
		res = append(res, withdrawal)
	}
	return res, nil
}

// get all orders by user id
func (dbs *dbStorage) MarkAllProcessingAsNew(ctx context.Context) error {
	query := `UPDATE orders
		SET status = $1
		WHERE status = $2;
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
		SET status = $1
		WHERE status = $2
		RETURNING id, number, status, accrual, uploaded_at;
	`
	rows, err := dbs.storage.QueryContext(ctx, query, models.StatusPROCESSING, models.StatusNEW)
	if err != nil {
		return nil, fmt.Errorf("Failed to get all orders from database: %v", err)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("Failed to get all orders from database: %v", err)
	}
	defer rows.Close()
	var res []models.OrderData
	for rows.Next() {
		var orderData models.OrderData
		if err := rows.Scan(&orderData.ID, &orderData.Number, &orderData.Status, &orderData.Accrual, &orderData.UploadedAt); err != nil {
			return nil, err
		}
		res = append(res, orderData)
	}
	return res, nil
}

// update order status
func (dbs *dbStorage) UpdateOrderStatus(ctx context.Context, orderID int64, status models.OrderStatus) error {
	query := `UPDATE orders
		SET status = $1
		WHERE id = $2;
	`
	result, err := dbs.storage.ExecContext(ctx, query, status, orderID)
	if err != nil {
		return fmt.Errorf("Failed to update status: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Could not get affected rows: %v", err)
	}
	logger.Log.Info("Successfully reset statuses.", zap.Int64("rowsAffected", rowsAffected))
	return nil
}

// ProcessOrderAccrual updates status and user balance
func (dbs *dbStorage) ProcessOrderAccrual(ctx context.Context, orderID int64, accrual float64) error {
	// start transaction
	tx, err := dbs.storage.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// rollback on error
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	// update order status
	var userID int64
	queryOrder := `UPDATE orders 
        SET status = $1, accrual = $2
        WHERE id = $3 RETURNING user_id`
	err = tx.QueryRowContext(ctx, queryOrder, models.StatusPROCESSED, accrual, orderID).Scan(&userID)
	if err != nil {
		return fmt.Errorf("update order failed: %w", err)
	}
	// update user balance
	queryUser := `UPDATE users 
        SET balance = balance + $1 
        WHERE id = $2`
	resUser, err := tx.ExecContext(ctx, queryUser, accrual, userID)
	if err != nil {
		return fmt.Errorf("update user balance failed: %w", err)
	}
	rowsAffectedUser, err := resUser.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get user rows affected: %w", err)
	}
	if rowsAffectedUser == 0 {
		return fmt.Errorf("user %d not found", userID)
	}
	// commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction failed: %w", err)
	}
	logger.Log.Info("Order accrual processed",
		zap.Int64("orderID", orderID),
		zap.Int64("userID", userID),
		zap.Float64("accrual", accrual))
	return nil
}
