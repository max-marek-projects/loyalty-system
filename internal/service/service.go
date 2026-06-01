package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/max-marek-projects/loyalty-system/internal/models"
	"github.com/max-marek-projects/loyalty-system/internal/repository"
	"github.com/max-marek-projects/loyalty-system/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

//go:generate mockery --name=Service --output=../handlers/mocks --filename=service_mock.go --with-expecter
type Service interface {
	RegisterUser(ctx context.Context, userData models.RegisterRequest) (int64, error)
	LoginUser(ctx context.Context, userData models.RegisterRequest) (int64, error)
	AddOrder(ctx context.Context, userID int64, orderNumber string) (bool, error)
	GetAllOrders(ctx context.Context, userID int64) ([]models.OrderData, error)
}

func NewEndpointService(storage repository.Storage) Service {
	return &endpointService{storage: storage}
}

type endpointService struct {
	storage repository.Storage
}

// register user in service
func (service *endpointService) RegisterUser(ctx context.Context, userData models.RegisterRequest) (int64, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(userData.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("Failed to hash password: %v", err)
	}
	hashedUserData := models.UserData{Login: userData.Login, PasswordHash: string(hashed)}
	userID, err := service.storage.RegisterUser(ctx, hashedUserData)
	if err != nil {
		if errors.Is(err, repository.ErrorAlreadyInStorage) {
			return 0, ErrorLoginAlreadyTaken
		}
		return 0, fmt.Errorf("Failed to register user in storage: %v", err)
	}
	return userID, nil
}

// login user in service
func (service *endpointService) LoginUser(ctx context.Context, userData models.RegisterRequest) (int64, error) {
	userID, hashedPassword, err := service.storage.CheckUser(ctx, userData.Login)
	if err != nil {
		if errors.Is(err, repository.ErrorUserNotFound) {
			return 0, ErrorWrongUsernamePassword
		}
		return 0, fmt.Errorf("Failed to check user in storage: %v", err)
	}
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(userData.Password))
	if err != nil {
		return 0, ErrorWrongUsernamePassword
	}
	return userID, nil
}

// add new order
func (service *endpointService) AddOrder(ctx context.Context, userID int64, orderNumber string) (bool, error) {
	numberValid := utils.ValidateLuhnAlgorithm(orderNumber)
	if !numberValid {
		return false, ErrorNumberNotValid
	}
	err := service.storage.AddOrder(ctx, userID, orderNumber)
	if err != nil {
		if errors.Is(err, repository.ErrorAlreadyInStorage) {
			return false, nil
		}
		if errors.Is(err, repository.ErrorStorageConflict) {
			return false, ErrorOrdersConflict
		}
		return false, fmt.Errorf("Failed to check user in storage: %v", err)
	}
	return true, nil
}

// add new order
func (service *endpointService) GetAllOrders(ctx context.Context, userID int64) ([]models.OrderData, error) {
	orders, err := service.storage.GetAllOrders(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get orders from storage: %v", err)
	}
	return orders, nil
}
