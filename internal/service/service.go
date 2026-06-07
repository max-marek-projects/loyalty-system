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
	// StartOrderProcessor(ctx context.Context, maxParallelWorkers int)
}

func NewEndpointService(storage repository.Storage, accrualSystemAddress string) (Service, error) {
	service := &endpointService{storage: storage, accrualSystemAddress: accrualSystemAddress}
	err := service.storage.MarkAllProcessingAsNew(context.Background())
	if err != nil {
		return nil, err
	}
	return service, nil
}

type endpointService struct {
	storage              repository.Storage
	accrualSystemAddress string
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

// poll storage, find new orders and start jobs to handle them
// func (service *endpointService) StartOrderProcessor(ctx context.Context, maxParallelWorkers int) {
// 	// Channel for orders to be processed
// 	orderCh := make(chan models.OrderData, maxParallelWorkers)
// 	var wg sync.WaitGroup
// 	for i := 0; i < maxParallelWorkers; i++ {
// 		wg.Add(1)
// 		go service.worker(ctx, &wg, orderCh)
// 	}
// 	ticker := time.NewTicker(5 * time.Second)
// 	defer ticker.Stop()
// 	logger.Log.Info("Order processor started", zap.Int("parallelWorkers", maxParallelWorkers), zap.String("accrualSystem", service.accrualSystemAddress))
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			log.Println("Order processor shutting down...")
// 			close(orderCh)
// 			wg.Wait()
// 			logger.Log.Info("Order processor stopped.")
// 			return
// 		case <-ticker.C:
// 			orders, err := service.storage.FindAndClaimNewOrders(ctx)
// 			if err != nil {
// 				logger.Log.Error("Failed to fetch new orders", zap.Error(err))
// 				continue
// 			}
// 			if len(orders) == 0 {
// 				continue
// 			}
// 			for _, order := range orders {
// 				select {
// 				case orderCh <- order:
// 				case <-ctx.Done():
// 					close(orderCh)
// 					wg.Wait()
// 					return
// 				}
// 			}
// 		}
// 	}
// }

// worker processes orders from the channel by calling the accrual system.
// func (service *endpointService) worker(
// 	ctx context.Context,
// 	wg *sync.WaitGroup,
// 	orderCh <-chan models.OrderData,
// ) {
// 	defer wg.Done()

// 	for order := range orderCh {
// 		// Process the order (call accrual system, handle retries, etc.)
// 		response, err := service.parseAccrualSystem(ctx, order)
// 		if err != nil {
// 			log.Printf("Failed to process order %s: %v", order.Id, err)
// 			// Update order status to "failed" (optional, with error message)
// 			if updateErr := service.storage.UpdateOrderStatus(ctx, order.Id, "failed", err.Error()); updateErr != nil {
// 				log.Printf("Failed to update order %s status: %v", order.Id, updateErr)
// 			}
// 		} else {
// 			// Success: mark as processed
// 			if updateErr := service.storage.UpdateOrderStatus(ctx, order.Id, "processed", ""); updateErr != nil {
// 				log.Printf("Failed to update order %s status: %v", order.Id, updateErr)
// 			}
// 		}
// 	}
// }

// // receive response from accrual system
// func (service *endpointService) parseAccrualSystem(
// 	ctx context.Context,
// 	order models.OrderData,
// ) {
// 	// Формируем URL согласно спецификации
// 	url := fmt.Sprintf("%s/api/orders/%s", baseURL, orderNumber)

// 	// Создаём HTTP-запрос с контекстом
// 	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create request: %w", err)
// 	}

// 	// Выполняем запрос (можно настроить таймауты через контекст или клиент)
// 	client := &http.Client{
// 		Timeout: 10 * time.Second, // разумный таймаут на всё взаимодействие
// 	}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, fmt.Errorf("request failed: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	// Обрабатываем статус-коды
// 	switch resp.StatusCode {
// 	case http.StatusOK: // 200
// 		var result AccrualResponse
// 		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
// 			return nil, fmt.Errorf("failed to decode response: %w", err)
// 		}
// 		return &result, nil

// 	case http.StatusNoContent: // 204
// 		return nil, ErrNotFound

// 	case http.StatusTooManyRequests: // 429
// 		retryAfterStr := resp.Header.Get("Retry-After")
// 		var retryAfter int
// 		if retryAfterStr != "" {
// 			// Retry-After может быть числом секунд или датой.
// 			// В спецификации указано число секунд, поэтому пробуем парсить как int.
// 			if seconds, err := strconv.Atoi(retryAfterStr); err == nil {
// 				retryAfter = seconds
// 			} else {
// 				// Если не число – пробуем распарсить HTTP-дату (RFC 1123)
// 				if t, err := http.ParseTime(retryAfterStr); err == nil {
// 					retryAfter = int(time.Until(t).Seconds())
// 					if retryAfter < 0 {
// 						retryAfter = 0
// 					}
// 				}
// 			}
// 		}
// 		return nil, ErrRateLimit{RetryAfter: retryAfter}

// 	case http.StatusInternalServerError: // 500
// 		return nil, ErrInternalServer

// 	default:
// 		// Неожиданный код ответа
// 		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
// 	}
// }
