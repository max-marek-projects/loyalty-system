package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"math/rand/v2"

	"github.com/max-marek-projects/loyalty-system/internal/logger"
	"github.com/max-marek-projects/loyalty-system/internal/models"
	"github.com/max-marek-projects/loyalty-system/internal/repository"
	"github.com/max-marek-projects/loyalty-system/internal/utils"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"
)

//go:generate mockery --name=Service --output=../handlers --outpkg=handlers --filename=service_mock_test.go --with-expecter
//go:generate mockery --name=Service --output=../server --outpkg=server --filename=service_mock_test.go --with-expecter

// Service defines the business logic interface for the loyalty system.
type Service interface {
	RegisterUser(ctx context.Context, userData models.RegisterRequest) (int64, error)
	LoginUser(ctx context.Context, userData models.RegisterRequest) (int64, error)
	AddOrder(ctx context.Context, userID int64, orderNumber string) (bool, error)
	GetAllOrders(ctx context.Context, userID int64) ([]models.OrderData, error)
	GetBalance(ctx context.Context, userID int64) (*models.BalanceData, error)
	Withdraw(ctx context.Context, userID int64, orderNumber string, sum float64) error
	GetAllWithdrawals(ctx context.Context, userID int64) ([]models.WithdrawData, error)
	StartOrderProcessor(ctx context.Context, maxParallelWorkers int, pollInterval int, mockExternalService bool)
}

// NewEndpointService creates a service instance with the given storage and accrual system address.
// Parameters:
//   - storage: repository implementation.
//   - accrualSystemAddress: base URL of the external accrual system.
//
// Returns the service or an error if resetting statuses fails.
func NewEndpointService(storage repository.Storage, accrualSystemAddress string) (Service, error) {
	service := &endpointService{storage: storage, accrualSystemAddress: accrualSystemAddress, httpClient: &http.Client{Timeout: 10 * time.Second}}
	err := service.storage.MarkAllProcessingAsNew(context.Background())
	if err != nil {
		return nil, err
	}
	return service, nil
}

type endpointService struct {
	storage              repository.Storage
	accrualSystemAddress string
	httpClient           *http.Client
}

// RegisterUser creates a new user with hashed password.
// Returns ErrorLoginAlreadyTaken if login already exists.
func (service *endpointService) RegisterUser(ctx context.Context, userData models.RegisterRequest) (int64, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(userData.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("Failed to hash password: %v", err)
	}
	hashedUserData := models.UserData{Login: userData.Login, PasswordHash: string(hashed)}
	userID, err := service.storage.RegisterUser(ctx, hashedUserData)
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyInStorage) {
			return 0, ErrorLoginAlreadyTaken
		}
		return 0, fmt.Errorf("Failed to register user in storage: %v", err)
	}
	return userID, nil
}

// LoginUser validates credentials and returns user ID.
// Returns ErrorWrongUsernamePassword if login or password is incorrect.
func (service *endpointService) LoginUser(ctx context.Context, userData models.RegisterRequest) (int64, error) {
	userID, hashedPassword, err := service.storage.CheckUser(ctx, userData.Login)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
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

// AddOrder validates order number via Luhn and stores it.
// Returns (true, nil) if new order added, (false, nil) if order already exists for this user,
// or error (ErrorNumberNotValid, ErrorOrdersConflict).
func (service *endpointService) AddOrder(ctx context.Context, userID int64, orderNumber string) (bool, error) {
	numberValid := utils.ValidateLuhnAlgorithm(orderNumber)
	if !numberValid {
		return false, ErrorNumberNotValid
	}
	err := service.storage.AddOrder(ctx, userID, orderNumber)
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyInStorage) {
			return false, nil
		}
		if errors.Is(err, repository.ErrStorageConflict) {
			return false, ErrorOrdersConflict
		}
		return false, fmt.Errorf("Failed to check user in storage: %v", err)
	}
	return true, nil
}

// GetAllOrders returns all orders belonging to a user.
func (service *endpointService) GetAllOrders(ctx context.Context, userID int64) ([]models.OrderData, error) {
	orders, err := service.storage.GetAllOrders(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get orders from storage: %v", err)
	}
	return orders, nil
}

// GetBalance returns the user's current balance and total withdrawn.
func (service *endpointService) GetBalance(ctx context.Context, userID int64) (*models.BalanceData, error) {
	balance, err := service.storage.GetBalance(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get balance from storage: %v", err)
	}
	return balance, nil
}

// Withdraw processes a withdrawal after Luhn validation.
// Returns ErrorNumberNotValid or ErrInsufficientFunds.
func (service *endpointService) Withdraw(ctx context.Context, userID int64, orderNumber string, sum float64) error {
	numberValid := utils.ValidateLuhnAlgorithm(orderNumber)
	if !numberValid {
		return ErrorNumberNotValid
	}
	err := service.storage.Withdraw(ctx, userID, orderNumber, sum)
	if err != nil {
		if errors.Is(err, repository.ErrInsufficientFunds) {
			return ErrInsufficientFunds
		}
		return fmt.Errorf("Failed to withdraw in storage: %v", err)
	}
	return nil
}

// GetAllWithdrawals returns all withdrawal records for a user.
func (service *endpointService) GetAllWithdrawals(ctx context.Context, userID int64) ([]models.WithdrawData, error) {
	withdrawals, err := service.storage.GetAllWithdrawals(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get orders from storage: %v", err)
	}
	return withdrawals, nil
}

// StartOrderProcessor launches background workers to poll the accrual system.
// Parameters:
//   - maxParallelWorkers: number of concurrent workers.
//   - pollInterval: seconds between scanning for new orders.
//   - mockExternalService: if true, uses mock accrual logic.
func (service *endpointService) StartOrderProcessor(ctx context.Context, maxParallelWorkers int, pollInterval int, mockExternalService bool) {
	// errorgroup for all workers to properly finish before exit
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxParallelWorkers)
	// prepared function to add order to queue
	var submit func(models.OrderData)
	submit = func(order models.OrderData) {
		g.Go(func() error {
			return service.worker(gctx, order, pollInterval, mockExternalService, submit)
		})
	}
	// ticker to parse new orders
	ticker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	defer ticker.Stop()
	logger.Log.Info("Order processor started", slog.Int("parallelWorkers", maxParallelWorkers), slog.String("accrualSystem", service.accrualSystemAddress))
	// main loop
	for {
		select {
		case <-gctx.Done():
			logger.Log.Info("Order processor shutting down...")
			if err := g.Wait(); err != nil {
				logger.Log.Error("Order processor finished with error", slog.Any("error", err))
			} else {
				logger.Log.Info("Order processor stopped.")
			}
			return
		case <-ticker.C:
			orders, err := service.storage.FindAndClaimNewOrders(gctx)
			if err != nil {
				logger.Log.Error("Failed to fetch new orders", slog.Any("error", err))
				continue
			}
			for _, order := range orders {
				orderCopy := order
				submit(orderCopy)
			}
		}
	}
}

// worker processes orders from the channel by calling the accrual system.
func (service *endpointService) worker(
	ctx context.Context,
	order models.OrderData,
	pollInterval int,
	mockExternalService bool,
	submit func(models.OrderData),
) error {
	result, err := service.parseAccrualSystem(ctx, order, pollInterval, mockExternalService)
	if err != nil {
		var errorWithRetry *ErrorOrderNotYetProcessed
		if errors.As(err, &errorWithRetry) {
			// temporary error, planning to retry later
			service.scheduleRetry(ctx, order, errorWithRetry.RetryAfter, submit)
			return nil
		}
		if err := service.storage.UpdateOrderStatus(ctx, order.ID, models.StatusINVALID); err != nil {
			logger.Log.Error("Failed to update order status", slog.Int64("orderID", order.ID), slog.Any("error", err))
		}
		return nil
	}

	switch result.Status {
	case models.ExternalStatusINVALID:
		if err := service.storage.UpdateOrderStatus(ctx, order.ID, models.StatusINVALID); err != nil {
			logger.Log.Error("Failed to update order status", slog.Int64("orderID", order.ID), slog.Any("error", err))
		}
		return nil
	case models.ExternalStatusPROCESSED:
		if err := service.storage.ProcessOrderAccrual(ctx, order.ID, result.Accrual); err != nil {
			logger.Log.Error("Failed to update order status", slog.Int64("orderID", order.ID), slog.Any("error", err))
		}
		return nil
	default:
		return fmt.Errorf("unknown status: %s", result.Status)
	}
}

// scheduleRetry runs goroutine that returns order in channel
func (service *endpointService) scheduleRetry(ctx context.Context, order models.OrderData, delay time.Duration, submit func(models.OrderData)) {
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
			select {
			case <-ctx.Done():
				return
			default:
				submit(order)
			}
		}
	}()
}

// parseAccrualSystem contacts external system or mock, returns accrual response.
// Returns ErrorOrderNotYetProcessed if order is not ready and should be retried.
func (service *endpointService) parseAccrualSystem(
	ctx context.Context,
	order models.OrderData,
	pollInterval int,
	mockExternalService bool,
) (*models.AccrualResponse, error) {
	if mockExternalService {
		//random accrual
		accrual := rand.Float64() * 500
		return &models.AccrualResponse{Order: order.Number, Status: models.ExternalStatusPROCESSED, Accrual: accrual}, nil
	}
	// create URL
	url := fmt.Sprintf("%s/api/orders/%s", service.accrualSystemAddress, order.Number)
	// create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	// execute request
	resp, err := service.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			retryAfter := time.Duration(pollInterval) * time.Second
			logger.Log.Debug("timeout received", slog.Duration("next attempt in", retryAfter))
			return nil, &ErrorOrderNotYetProcessed{fmt.Errorf("timeout received: %w", err), retryAfter}
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	// analyze status codes
	switch resp.StatusCode {
	case http.StatusOK: // 200
		var result models.AccrualResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		if result.Status == models.ExternalStatusREGISTERED || result.Status == models.ExternalStatusPROCESSING {
			retryAfter := time.Duration(pollInterval) * time.Second
			logger.Log.Debug("status shows not yet processed", slog.String("status", string(result.Status)), slog.Duration("next attempt in", retryAfter))
			return nil, &ErrorOrderNotYetProcessed{fmt.Errorf("status shows not yet processed"), retryAfter}
		}
		return &result, nil
	case http.StatusNoContent: // 204
		retryAfter := time.Duration(pollInterval) * time.Second
		logger.Log.Debug("status code shows not yet registered", slog.Int("status code", resp.StatusCode), slog.Duration("next attempt in", retryAfter))
		return nil, &ErrorOrderNotYetProcessed{fmt.Errorf("status code shows not yet registered"), retryAfter}

	case http.StatusTooManyRequests: // 429
		retryAfterStr := resp.Header.Get("Retry-After")
		var retryAfter time.Duration
		if retryAfterStr != "" {
			retryAfter = utils.ParseRetryAfter(retryAfterStr)
		} else {
			retryAfter = time.Duration(pollInterval) * time.Second
		}
		return nil, &ErrorOrderNotYetProcessed{fmt.Errorf("status code shows too many requests"), retryAfter}

	case http.StatusInternalServerError: // 500
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("internal server error: %s", body)

	default:
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d, body: %s", resp.StatusCode, body)
	}
}
