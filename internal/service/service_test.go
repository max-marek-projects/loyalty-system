package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"

	"github.com/max-marek-projects/loyalty-system/internal/models"
	"github.com/max-marek-projects/loyalty-system/internal/repository"
)

func TestRegisterUser(t *testing.T) {
	type want struct {
		userID int64
		err    error
	}
	tests := []struct {
		name       string
		request    models.RegisterRequest
		repoUserID int64
		repoErr    error
		expected   want
	}{
		{
			name:       "success",
			request:    models.RegisterRequest{Login: "user", Password: "pass"},
			repoUserID: 123,
			repoErr:    nil,
			expected:   want{userID: 123, err: nil},
		},
		{
			name:       "login already taken",
			request:    models.RegisterRequest{Login: "taken", Password: "pass"},
			repoUserID: 0,
			repoErr:    repository.ErrAlreadyInStorage,
			expected:   want{userID: 0, err: ErrLoginAlreadyTaken},
		},
		{
			name:       "repository error",
			request:    models.RegisterRequest{Login: "fail", Password: "pass"},
			repoUserID: 0,
			repoErr:    errors.New("db error"),
			expected:   want{userID: 0, err: errors.New("failed to register user in storage: db error")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := NewStorage(t)
			mockStorage.EXPECT().
				RegisterUser(mock.Anything, mock.MatchedBy(func(data models.UserData) bool {
					return data.Login == tt.request.Login && data.PasswordHash != ""
				})).
				Return(tt.repoUserID, tt.repoErr)
			svc := &endpointService{storage: mockStorage}
			userID, err := svc.RegisterUser(context.Background(), tt.request)
			if tt.expected.err != nil {
				assert.EqualError(t, err, tt.expected.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.userID, userID)
			}
		})
	}
}

func TestLoginUser(t *testing.T) {
	hashedPass, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	type want struct {
		userID int64
		err    error
	}
	tests := []struct {
		name           string
		request        models.RegisterRequest
		repoUserID     int64
		repoHashedPass string
		repoErr        error
		expected       want
	}{
		{
			name:           "success",
			request:        models.RegisterRequest{Login: "user", Password: "correct"},
			repoUserID:     123,
			repoHashedPass: string(hashedPass),
			repoErr:        nil,
			expected:       want{userID: 123, err: nil},
		},
		{
			name:           "wrong password",
			request:        models.RegisterRequest{Login: "user", Password: "wrong"},
			repoUserID:     123,
			repoHashedPass: string(hashedPass),
			repoErr:        nil,
			expected:       want{userID: 0, err: ErrWrongUsernamePassword},
		},
		{
			name:           "user not found",
			request:        models.RegisterRequest{Login: "unknown", Password: "pass"},
			repoUserID:     0,
			repoHashedPass: "",
			repoErr:        repository.ErrUserNotFound,
			expected:       want{userID: 0, err: ErrWrongUsernamePassword},
		},
		{
			name:           "repository error",
			request:        models.RegisterRequest{Login: "fail", Password: "pass"},
			repoUserID:     0,
			repoHashedPass: "",
			repoErr:        errors.New("db error"),
			expected:       want{userID: 0, err: errors.New("failed to check user in storage: db error")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := NewStorage(t)
			mockStorage.EXPECT().
				CheckUser(mock.Anything, tt.request.Login).
				Return(tt.repoUserID, tt.repoHashedPass, tt.repoErr)
			svc := &endpointService{storage: mockStorage}
			userID, err := svc.LoginUser(context.Background(), tt.request)
			if tt.expected.err != nil {
				assert.EqualError(t, err, tt.expected.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.userID, userID)
			}
		})
	}
}

func TestAddOrder(t *testing.T) {
	const validNumber = "9278923470"  // passes Luhn
	const invalidNumber = "123456789" // fails Luhn
	type want struct {
		added bool
		err   error
	}
	tests := []struct {
		name        string
		userID      int64
		orderNumber string
		repoErr     error
		expected    want
	}{
		{
			name:        "success new order",
			userID:      1,
			orderNumber: validNumber,
			repoErr:     nil,
			expected:    want{added: true, err: nil},
		},
		{
			name:        "order already exists for this user",
			userID:      1,
			orderNumber: validNumber,
			repoErr:     repository.ErrAlreadyInStorage,
			expected:    want{added: false, err: nil},
		},
		{
			name:        "order conflict with other user",
			userID:      1,
			orderNumber: validNumber,
			repoErr:     repository.ErrStorageConflict,
			expected:    want{added: false, err: ErrOrdersConflict},
		},
		{
			name:        "invalid order number",
			userID:      1,
			orderNumber: invalidNumber,
			repoErr:     nil,
			expected:    want{added: false, err: ErrNumberNotValid},
		},
		{
			name:        "repository error",
			userID:      1,
			orderNumber: validNumber,
			repoErr:     errors.New("db error"),
			expected:    want{added: false, err: errors.New("failed to check user in storage: db error")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := NewStorage(t)
			if tt.orderNumber == validNumber {
				mockStorage.EXPECT().
					AddOrder(mock.Anything, tt.userID, tt.orderNumber).
					Return(tt.repoErr)
			}
			svc := &endpointService{storage: mockStorage}
			added, err := svc.AddOrder(context.Background(), tt.userID, tt.orderNumber)
			if tt.expected.err != nil {
				assert.EqualError(t, err, tt.expected.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.added, added)
			}
		})
	}
}

func TestGetAllOrders(t *testing.T) {
	expectedOrders := []models.OrderData{
		{ID: 1, Number: "123", Status: models.StatusPROCESSED, Accrual: 100},
	}
	tests := []struct {
		name        string
		userID      int64
		repoOrders  []models.OrderData
		repoErr     error
		expectedErr error
	}{
		{
			name:        "success",
			userID:      1,
			repoOrders:  expectedOrders,
			repoErr:     nil,
			expectedErr: nil,
		},
		{
			name:        "repository error",
			userID:      1,
			repoOrders:  nil,
			repoErr:     errors.New("db error"),
			expectedErr: errors.New("failed to get orders from storage: db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := NewStorage(t)
			mockStorage.EXPECT().
				GetAllOrders(mock.Anything, tt.userID).
				Return(tt.repoOrders, tt.repoErr)
			svc := &endpointService{storage: mockStorage}
			orders, err := svc.GetAllOrders(context.Background(), tt.userID)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.repoOrders, orders)
			}
		})
	}
}

func TestGetBalance(t *testing.T) {
	balance := &models.BalanceData{Current: 500.5, Withdrawn: 42}
	tests := []struct {
		name        string
		userID      int64
		repoBalance *models.BalanceData
		repoErr     error
	}{
		{
			name:        "success",
			userID:      1,
			repoBalance: balance,
			repoErr:     nil,
		},
		{
			name:        "repository error",
			userID:      1,
			repoBalance: nil,
			repoErr:     errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := NewStorage(t)
			mockStorage.EXPECT().
				GetBalance(mock.Anything, tt.userID).
				Return(tt.repoBalance, tt.repoErr)
			svc := &endpointService{storage: mockStorage}
			b, err := svc.GetBalance(context.Background(), tt.userID)
			if tt.repoErr != nil {
				assert.True(t, strings.Contains(err.Error(), tt.repoErr.Error()))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.repoBalance, b)
			}
		})
	}
}

func TestWithdraw(t *testing.T) {
	const validNumber = "9278923470"
	const invalidNumber = "123456789"
	tests := []struct {
		name        string
		userID      int64
		orderNumber string
		sum         float64
		repoErr     error
		expectedErr error
	}{
		{
			name:        "success",
			userID:      1,
			orderNumber: validNumber,
			sum:         100.5,
			repoErr:     nil,
			expectedErr: nil,
		},
		{
			name:        "invalid order number",
			userID:      1,
			orderNumber: invalidNumber,
			sum:         100,
			repoErr:     nil,
			expectedErr: ErrNumberNotValid,
		},
		{
			name:        "insufficient funds",
			userID:      1,
			orderNumber: validNumber,
			sum:         100,
			repoErr:     repository.ErrInsufficientFunds,
			expectedErr: ErrInsufficientFunds,
		},
		{
			name:        "repository other error",
			userID:      1,
			orderNumber: validNumber,
			sum:         100,
			repoErr:     errors.New("db error"),
			expectedErr: errors.New("failed to withdraw in storage: db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := NewStorage(t)
			if tt.orderNumber == validNumber {
				mockStorage.EXPECT().
					Withdraw(mock.Anything, tt.userID, tt.orderNumber, tt.sum).
					Return(tt.repoErr)
			}
			svc := &endpointService{storage: mockStorage}
			err := svc.Withdraw(context.Background(), tt.userID, tt.orderNumber, tt.sum)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetAllWithdrawals(t *testing.T) {
	withdrawals := []models.WithdrawData{
		{Order: "123", Sum: 100, ProcessedAt: models.TimeRFC3339{}},
	}
	tests := []struct {
		name          string
		userID        int64
		repoWithdraws []models.WithdrawData
		repoErr       error
		expectedErr   error
	}{
		{
			name:          "success",
			userID:        1,
			repoWithdraws: withdrawals,
			repoErr:       nil,
			expectedErr:   nil,
		},
		{
			name:          "repository error",
			userID:        1,
			repoWithdraws: nil,
			repoErr:       errors.New("db error"),
			expectedErr:   errors.New("failed to get orders from storage: db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := NewStorage(t)
			mockStorage.EXPECT().
				GetAllWithdrawals(mock.Anything, tt.userID).
				Return(tt.repoWithdraws, tt.repoErr)
			svc := &endpointService{storage: mockStorage}
			w, err := svc.GetAllWithdrawals(context.Background(), tt.userID)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.repoWithdraws, w)
			}
		})
	}
}

func TestStartOrderProcessor_Shutdown(t *testing.T) {
	mockStorage := NewStorage(t)
	mockStorage.EXPECT().FindAndClaimNewOrders(mock.Anything).Return([]models.OrderData{}, nil).Maybe()

	svc := &endpointService{storage: mockStorage}
	ctx, cancel := context.WithCancel(context.Background())
	go svc.StartOrderProcessor(ctx, 1, 1, true)
	time.Sleep(10 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond)
	mockStorage.AssertExpectations(t)
}
