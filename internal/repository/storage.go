package repository

import (
	"context"

	"github.com/max-marek-projects/loyalty-system/internal/models"
)

// Common storage interface
//
//go:generate mockery --name=Storage --output=../service/mocks --filename=storage_mock.go --with-expecter
type Storage interface {
	RegisterUser(ctx context.Context, userData models.UserData) (int64, error)
	CheckUser(ctx context.Context, username string) (int64, string, error)
	AddOrder(ctx context.Context, userID int64, orderNumber string) error
	GetAllOrders(ctx context.Context, userID int64) ([]models.OrderData, error)
}
