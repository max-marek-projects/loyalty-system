package repository

import (
	"context"

	"github.com/max-marek-projects/loyalty-system/internal/models"
)

// Storage defines the interface for data persistence operations.
//
//go:generate mockery --name=Storage --output=../service --outpkg=service --filename=storage_mock_test.go --with-expecter
type Storage interface {
	// User management
	RegisterUser(ctx context.Context, userData models.UserData) (int64, error)
	CheckUser(ctx context.Context, username string) (int64, string, error)

	// Order operations
	AddOrder(ctx context.Context, userID int64, orderNumber string) error
	GetAllOrders(ctx context.Context, userID int64) ([]models.OrderData, error)

	// Balance and withdrawals
	GetBalance(ctx context.Context, userID int64) (*models.BalanceData, error)
	Withdraw(ctx context.Context, userID int64, orderNumber string, sum float64) error
	GetAllWithdrawals(ctx context.Context, userID int64) ([]models.WithdrawData, error)

	// Background processing
	MarkAllProcessingAsNew(ctx context.Context) error
	FindAndClaimNewOrders(ctx context.Context) ([]models.OrderData, error)
	UpdateOrderStatus(ctx context.Context, orderID int64, status models.OrderStatus) error
	ProcessOrderAccrual(ctx context.Context, orderID int64, accrual float64) error
}
