package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"math/rand/v2"

	"github.com/max-marek-projects/loyalty-system/internal/logger"
	"github.com/max-marek-projects/loyalty-system/internal/models"
	"github.com/max-marek-projects/loyalty-system/internal/repository"
	"github.com/max-marek-projects/loyalty-system/internal/utils"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

//go:generate mockery --name=Service --output=../handlers/mocks --filename=service_mock.go --with-expecter
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

// register user in service
func (service *endpointService) RegisterUser(ctx context.Context, userData models.RegisterRequest) (int64, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(userData.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %v", err)
	}
	hashedUserData := models.UserData{Login: userData.Login, PasswordHash: string(hashed)}
	userID, err := service.storage.RegisterUser(ctx, hashedUserData)
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyInStorage) {
			return 0, ErrorLoginAlreadyTaken
		}
		return 0, fmt.Errorf("failed to register user in storage: %v", err)
	}
	return userID, nil
}

// login user in service
func (service *endpointService) LoginUser(ctx context.Context, userData models.RegisterRequest) (int64, error) {
	userID, hashedPassword, err := service.storage.CheckUser(ctx, userData.Login)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return 0, ErrorWrongUsernamePassword
		}
		return 0, fmt.Errorf("failed to check user in storage: %v", err)
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
		if errors.Is(err, repository.ErrAlreadyInStorage) {
			return false, nil
		}
		if errors.Is(err, repository.ErrStorageConflict) {
			return false, ErrorOrdersConflict
		}
		return false, fmt.Errorf("failed to check user in storage: %v", err)
	}
	return true, nil
}

// get all orders
func (service *endpointService) GetAllOrders(ctx context.Context, userID int64) ([]models.OrderData, error) {
	orders, err := service.storage.GetAllOrders(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders from storage: %v", err)
	}
	return orders, nil
}

// get balance by user id
func (service *endpointService) GetBalance(ctx context.Context, userID int64) (*models.BalanceData, error) {
	balance, err := service.storage.GetBalance(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance from storage: %v", err)
	}
	return balance, nil
}

// get balance by user id
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
		return fmt.Errorf("failed to withdraw in storage: %v", err)
	}
	return nil
}

// get all user withdrawals
func (service *endpointService) GetAllWithdrawals(ctx context.Context, userID int64) ([]models.WithdrawData, error) {
	withdrawals, err := service.storage.GetAllWithdrawals(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders from storage: %v", err)
	}
	return withdrawals, nil
}

// poll storage, find new orders and start jobs to handle them
func (service *endpointService) StartOrderProcessor(ctx context.Context, maxParallelWorkers int, pollInterval int, mockExternalService bool) {
	// Channel for orders to be processed
	orderCh := make(chan models.OrderData, maxParallelWorkers*2)
	// wait group for all workers to properly finish before exit
	var wg sync.WaitGroup
	for i := 0; i < maxParallelWorkers; i++ {
		wg.Add(1)
		go service.worker(ctx, &wg, orderCh, pollInterval, mockExternalService)
	}
	// ticker to parse new orders
	ticker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	defer ticker.Stop()
	logger.Log.Info("Order processor started", zap.Int("parallelWorkers", maxParallelWorkers), zap.String("accrualSystem", service.accrualSystemAddress))
	for {
		select {
		case <-ctx.Done():
			log.Println("Order processor shutting down...")
			close(orderCh)
			wg.Wait()
			logger.Log.Info("Order processor stopped.")
			return
		case <-ticker.C:
			orders, err := service.storage.FindAndClaimNewOrders(ctx)
			if err != nil {
				logger.Log.Error("Failed to fetch new orders", zap.Error(err))
				continue
			}
			if len(orders) == 0 {
				continue
			}
			for _, order := range orders {
				select {
				case orderCh <- order:
				case <-ctx.Done():
					close(orderCh)
					wg.Wait()
					return
				}
			}
		}
	}
}

// worker processes orders from the channel by calling the accrual system.
func (service *endpointService) worker(
	ctx context.Context,
	wg *sync.WaitGroup,
	orderCh chan models.OrderData,
	pollInterval int,
	mockExternalService bool,
) {
	defer wg.Done()
	for orderQueueItem := range orderCh {
		result, err := service.parseAccrualSystem(ctx, orderQueueItem, orderCh, pollInterval, mockExternalService)
		if errors.Is(err, ErrorOrderNotYetProcessed) {
			continue
		}
		if err != nil {
			log.Printf("Failed to process order %d: %v", orderQueueItem.ID, err)
			// Update order status to "failed" (optional, with error message)
			if updateErr := service.storage.UpdateOrderStatus(ctx, orderQueueItem.ID, models.StatusINVALID); updateErr != nil {
				log.Printf("Failed to update order %d status: %v", orderQueueItem.ID, updateErr)
			}
			continue
		}
		if result.Status == models.ExternalStatusINVALID {
			if updateErr := service.storage.UpdateOrderStatus(ctx, orderQueueItem.ID, models.StatusINVALID); updateErr != nil {
				log.Printf("Failed to update order %d status: %v", orderQueueItem.ID, updateErr)
			}
			continue
		}
		if result.Status == models.ExternalStatusPROCESSED {
			if updateErr := service.storage.ProcessOrderAccrual(ctx, orderQueueItem.ID, result.Accrual); updateErr != nil {
				log.Printf("Failed to update order %d status: %v", orderQueueItem.ID, updateErr)
			}
			continue
		}
		logger.Log.Error("Unknown situation occurred. Wrong status", zap.String("status", string(result.Status)))
	}
}

// receive response from accrual system
func (service *endpointService) parseAccrualSystem(
	ctx context.Context,
	order models.OrderData,
	orderCh chan models.OrderData,
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
			go func(o models.OrderData, delay time.Duration) {
				time.Sleep(delay)
				select {
				case orderCh <- o:
				case <-ctx.Done():
				}
			}(order, retryAfter)
			logger.Log.Debug("timeout received", zap.Duration("next attempt in", retryAfter))
			return nil, ErrorOrderNotYetProcessed
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
			go func(o models.OrderData, delay time.Duration) {
				time.Sleep(delay)
				select {
				case orderCh <- o:
				case <-ctx.Done():
				}
			}(order, retryAfter)
			logger.Log.Debug("status shows not yet processed", zap.String("status", string(result.Status)), zap.Duration("next attempt in", retryAfter))
			return nil, ErrorOrderNotYetProcessed
		}
		return &result, nil
	case http.StatusNoContent: // 204
		retryAfter := time.Duration(pollInterval) * time.Second
		go func(o models.OrderData, delay time.Duration) {
			time.Sleep(delay)
			select {
			case orderCh <- o:
			case <-ctx.Done():
			}
		}(order, retryAfter)
		logger.Log.Debug("status code shows not yet registered", zap.Int("status code", resp.StatusCode), zap.Duration("next attempt in", retryAfter))
		return nil, ErrorOrderNotYetProcessed

	case http.StatusTooManyRequests: // 429
		retryAfterStr := resp.Header.Get("Retry-After")
		var retryAfter time.Duration
		if retryAfterStr != "" {
			retryAfter = utils.ParseRetryAfter(retryAfterStr)
		} else {
			retryAfter = time.Duration(pollInterval) * time.Second
		}
		go func(o models.OrderData, delay time.Duration) {
			time.Sleep(delay)
			select {
			case orderCh <- o:
			case <-ctx.Done():
			}
		}(order, retryAfter)
		logger.Log.Debug("status code shows too many requests", zap.Int("status code", resp.StatusCode), zap.Duration("next attempt in", retryAfter))
		return nil, ErrorOrderNotYetProcessed

	case http.StatusInternalServerError: // 500
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("internal server error: %s", string(body))

	default:
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: code = %d, body = %s", resp.StatusCode, body)
	}
}
