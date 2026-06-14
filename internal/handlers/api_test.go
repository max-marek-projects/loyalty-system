package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/max-marek-projects/loyalty-system/internal/models"
	"github.com/max-marek-projects/loyalty-system/internal/service"
)

// create new router for handlers testing
func newAPITestRouter(service *Service) http.Handler {
	h := NewHandler(service, 100, secretKey)

	r := chi.NewRouter()
	r.Route("/api/user", func(api chi.Router) {
		// public endpoints
		api.Post("/register", h.RegisterUser)
		api.Post("/login", h.LoginUser)
		// protected endpoints
		api.Get("/orders", h.GetUserOrders)
		api.Post("/orders", h.NewOrder)
		api.Get("/balance", h.GetBalance)
		api.Post("/balance/withdraw", h.Withdraw)
		api.Get("/withdrawals", h.GetUserWithdrawals)
	})
	return r
}

// test user register
func TestRegisterUser(t *testing.T) {
	validUserData := models.RegisterRequest{Login: "existing", Password: "example"}
	validUserEncodedData, err := json.Marshal(validUserData)
	assert.NoError(t, err)
	alreadyTakenUserData := models.RegisterRequest{Login: "taken", Password: "example"}
	alreadyTakenUserEncodedData, err := json.Marshal(alreadyTakenUserData)
	assert.NoError(t, err)
	brokenUserData := models.RegisterRequest{Login: "broken", Password: "example"}
	brokenUserEncodedData, err := json.Marshal(brokenUserData)
	assert.NoError(t, err)
	validUserCookieErrorData := models.RegisterRequest{Login: "broken", Password: "example"}
	validUserCookieErrorEncodedData, err := json.Marshal(validUserCookieErrorData)
	assert.NoError(t, err)
	var registeredUserID int64 = 123
	var userIDCookieError int64 = 314
	mockService := NewService(t)
	mockService.EXPECT().RegisterUser(mock.Anything, validUserData).Return(registeredUserID, nil)
	mockService.EXPECT().RegisterUser(mock.Anything, alreadyTakenUserData).Return(0, service.ErrLoginAlreadyTaken)
	mockService.EXPECT().RegisterUser(mock.Anything, brokenUserData).Return(0, fmt.Errorf("some broken data"))
	mockService.EXPECT().RegisterUser(mock.Anything, validUserCookieErrorData).Return(userIDCookieError, nil)

	ts := httptest.NewServer(newAPITestRouter(mockService))
	defer ts.Close()

	type want struct {
		code int
	}
	tests := []struct {
		name    string
		method  string
		request string
		want    want
	}{
		{
			name:    "positive test",
			method:  http.MethodPost,
			request: string(validUserEncodedData),
			want: want{
				code: http.StatusOK,
			},
		},
		{
			name:    "wrong method test",
			method:  http.MethodGet,
			request: string(validUserEncodedData),
			want: want{
				code: http.StatusMethodNotAllowed,
			},
		},
		{
			name:    "invalid json",
			method:  http.MethodPost,
			request: `{"login":"existing" "password":"example"}`,
			want: want{
				code: http.StatusBadRequest,
			},
		},
		{
			name:    "empty login",
			method:  http.MethodPost,
			request: `{"login":"", "password":"example"}`,
			want: want{
				code: http.StatusBadRequest,
			},
		},
		{
			name:    "empty password",
			method:  http.MethodPost,
			request: `{"login":"valid", "password":""}`,
			want: want{
				code: http.StatusBadRequest,
			},
		},
		{
			name:    "already taken data",
			method:  http.MethodPost,
			request: string(alreadyTakenUserEncodedData),
			want: want{
				code: http.StatusConflict,
			},
		},
		{
			name:    "broken user data",
			method:  http.MethodPost,
			request: string(brokenUserEncodedData),
			want: want{
				code: http.StatusInternalServerError,
			},
		},
		{
			name:    "unable to set cookies",
			method:  http.MethodPost,
			request: string(validUserCookieErrorEncodedData),
			want: want{
				code: http.StatusInternalServerError,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, _ := testRequest(t, ts, test.method, "/api/user/register", test.request, false)
			defer resp.Body.Close()
			assert.Equal(t, test.want.code, resp.StatusCode)
		})
	}
}

// test user login
func TestLoginUser(t *testing.T) {
	validUserData := models.RegisterRequest{Login: "existing", Password: "example"}
	validUserEncodedData, err := json.Marshal(validUserData)
	assert.NoError(t, err)
	wrongUserData := models.RegisterRequest{Login: "taken", Password: "example"}
	wrongUserEncodedData, err := json.Marshal(wrongUserData)
	assert.NoError(t, err)
	brokenUserData := models.RegisterRequest{Login: "broken", Password: "example"}
	brokenUserEncodedData, err := json.Marshal(brokenUserData)
	assert.NoError(t, err)
	validUserCookieErrorData := models.RegisterRequest{Login: "broken", Password: "example"}
	validUserCookieErrorEncodedData, err := json.Marshal(validUserCookieErrorData)
	assert.NoError(t, err)
	var registeredUserID int64 = 123
	var userIDCookieError int64 = 314
	mockService := NewService(t)
	mockService.EXPECT().LoginUser(mock.Anything, validUserData).Return(registeredUserID, nil)
	mockService.EXPECT().LoginUser(mock.Anything, wrongUserData).Return(0, service.ErrWrongUsernamePassword)
	mockService.EXPECT().LoginUser(mock.Anything, brokenUserData).Return(0, fmt.Errorf("some broken data"))
	mockService.EXPECT().LoginUser(mock.Anything, validUserCookieErrorData).Return(userIDCookieError, nil)

	ts := httptest.NewServer(newAPITestRouter(mockService))
	defer ts.Close()

	type want struct {
		code int
	}
	tests := []struct {
		name    string
		method  string
		request string
		want    want
	}{
		{
			name:    "positive test",
			method:  http.MethodPost,
			request: string(validUserEncodedData),
			want: want{
				code: http.StatusOK,
			},
		},
		{
			name:    "wrong method test",
			method:  http.MethodGet,
			request: string(validUserEncodedData),
			want: want{
				code: http.StatusMethodNotAllowed,
			},
		},
		{
			name:    "invalid json",
			method:  http.MethodPost,
			request: `{"login":"existing" "password":"example"}`,
			want: want{
				code: http.StatusBadRequest,
			},
		},
		{
			name:    "empty login",
			method:  http.MethodPost,
			request: `{"login":"", "password":"example"}`,
			want: want{
				code: http.StatusBadRequest,
			},
		},
		{
			name:    "empty password",
			method:  http.MethodPost,
			request: `{"login":"valid", "password":""}`,
			want: want{
				code: http.StatusBadRequest,
			},
		},
		{
			name:    "wrong username password",
			method:  http.MethodPost,
			request: string(wrongUserEncodedData),
			want: want{
				code: http.StatusUnauthorized,
			},
		},
		{
			name:    "broken user data",
			method:  http.MethodPost,
			request: string(brokenUserEncodedData),
			want: want{
				code: http.StatusInternalServerError,
			},
		},
		{
			name:    "unable to set cookies",
			method:  http.MethodPost,
			request: string(validUserCookieErrorEncodedData),
			want: want{
				code: http.StatusInternalServerError,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, _ := testRequest(t, ts, test.method, "/api/user/login", test.request, false)
			defer resp.Body.Close()
			assert.Equal(t, test.want.code, resp.StatusCode)
		})
	}
}

// test new order creation
func TestNewOrder(t *testing.T) {
	validOrderNumber := "12345678903"
	notValidNumber := "123456789"
	existingOrderNumber := "12345"
	mockService := NewService(t)
	mockService.EXPECT().AddOrder(mock.Anything, testUserID, validOrderNumber).Return(true, nil)
	mockService.EXPECT().AddOrder(mock.Anything, testUserID, existingOrderNumber).Return(false, nil)
	mockService.EXPECT().AddOrder(mock.Anything, testUserID, notValidNumber).Return(false, service.ErrNumberNotValid)

	ts := httptest.NewServer(newAPITestRouter(mockService))
	defer ts.Close()

	type want struct {
		code int
	}
	tests := []struct {
		name      string
		method    string
		request   string
		authorize bool
		want      want
	}{
		{
			name:      "positive test",
			method:    http.MethodPost,
			request:   validOrderNumber,
			authorize: true,
			want: want{
				code: http.StatusAccepted,
			},
		},
		{
			name:      "wrong method test",
			method:    http.MethodDelete,
			request:   validOrderNumber,
			authorize: true,
			want: want{
				code: http.StatusMethodNotAllowed,
			},
		},
		{
			name:      "empty body",
			method:    http.MethodPost,
			request:   "",
			authorize: true,
			want: want{
				code: http.StatusBadRequest,
			},
		},
		{
			name:      "unauthorized",
			method:    http.MethodPost,
			request:   validOrderNumber,
			authorize: false,
			want: want{
				code: http.StatusInternalServerError,
			},
		},
		{
			name:      "already existing",
			method:    http.MethodPost,
			request:   existingOrderNumber,
			authorize: true,
			want: want{
				code: http.StatusOK,
			},
		},
		{
			name:      "not valid luhna",
			method:    http.MethodPost,
			request:   notValidNumber,
			authorize: true,
			want: want{
				code: http.StatusUnprocessableEntity,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, _ := testRequest(t, ts, test.method, "/api/user/orders", test.request, test.authorize)
			defer resp.Body.Close()
			assert.Equal(t, test.want.code, resp.StatusCode)
		})
	}
}

// test all orders receiving
func TestAllOrders(t *testing.T) {
	mockService := NewService(t)
	successOrders := []models.OrderData{{ID: 12345, Number: "12345", Status: models.StatusPROCESSED, Accrual: 0, UploadedAt: models.TimeRFC3339{}}}
	mockService.EXPECT().GetAllOrders(mock.Anything, testUserID).Return(successOrders, nil)

	ts := httptest.NewServer(newAPITestRouter(mockService))
	defer ts.Close()

	successOrdersString, err := json.MarshalIndent(successOrders, "", "    ")
	require.NoError(t, err)

	type want struct {
		code     int
		response string
	}
	tests := []struct {
		name      string
		method    string
		authorize bool
		want      want
	}{
		{
			name:      "positive test",
			method:    http.MethodGet,
			authorize: true,
			want: want{
				code:     http.StatusOK,
				response: string(successOrdersString),
			},
		},
		{
			name:      "wrong method test",
			method:    http.MethodDelete,
			authorize: true,
			want: want{
				code:     http.StatusMethodNotAllowed,
				response: "",
			},
		},
		{
			name:      "unauthorized",
			method:    http.MethodGet,
			authorize: false,
			want: want{
				code:     http.StatusInternalServerError,
				response: "",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, body := testRequest(t, ts, test.method, "/api/user/orders", "", test.authorize)
			defer resp.Body.Close()
			assert.Equal(t, test.want.code, resp.StatusCode)
			if test.want.response != "" {
				assert.JSONEq(t, body, test.want.response)
			}

		})
	}
}

// test getting user balance
func TestGetBalance(t *testing.T) {
	balanceData := &models.BalanceData{Current: 500.5, Withdrawn: 42}
	balanceJSON, err := json.MarshalIndent(balanceData, "", "  ")
	require.NoError(t, err)

	mockService := NewService(t)
	mockService.EXPECT().GetBalance(mock.Anything, testUserID).Return(balanceData, nil).Once()
	mockService.EXPECT().GetBalance(mock.Anything, testUserID).Return(nil, errors.New("db error")).Once()

	ts := httptest.NewServer(newAPITestRouter(mockService))
	defer ts.Close()

	type want struct {
		code     int
		response string
	}
	tests := []struct {
		name      string
		method    string
		authorize bool
		want      want
	}{
		{
			name:      "success",
			method:    http.MethodGet,
			authorize: true,
			want: want{
				code:     http.StatusOK,
				response: string(balanceJSON),
			},
		},
		{
			name:      "service error",
			method:    http.MethodGet,
			authorize: true,
			want: want{
				code:     http.StatusInternalServerError,
				response: "",
			},
		},
		{
			name:      "unauthorized",
			method:    http.MethodGet,
			authorize: false,
			want: want{
				code:     http.StatusInternalServerError,
				response: "",
			},
		},
		{
			name:      "wrong method",
			method:    http.MethodPost,
			authorize: true,
			want: want{
				code:     http.StatusMethodNotAllowed,
				response: "",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, body := testRequest(t, ts, test.method, "/api/user/balance", "", test.authorize)
			defer resp.Body.Close()
			assert.Equal(t, test.want.code, resp.StatusCode)
			if test.want.response != "" {
				assert.JSONEq(t, test.want.response, body)
			}
		})
	}
}

// test withdraw loyalty points
func TestWithdraw(t *testing.T) {
	validOrder := "12345678903"
	invalidOrder := "123456789"
	insufficientOrder := "9278923470"

	mockService := NewService(t)
	mockService.EXPECT().Withdraw(mock.Anything, testUserID, validOrder, 100.0).Return(nil).Once()
	mockService.EXPECT().Withdraw(mock.Anything, testUserID, insufficientOrder, 1000.0).Return(service.ErrInsufficientFunds).Once()
	mockService.EXPECT().Withdraw(mock.Anything, testUserID, invalidOrder, 100.0).Return(service.ErrNumberNotValid).Once()

	ts := httptest.NewServer(newAPITestRouter(mockService))
	defer ts.Close()

	type request struct {
		Order string  `json:"order"`
		Sum   float64 `json:"sum"`
	}
	type want struct {
		code int
	}
	tests := []struct {
		name      string
		method    string
		request   request
		authorize bool
		want      want
	}{
		{
			name:      "success",
			method:    http.MethodPost,
			request:   request{Order: validOrder, Sum: 100},
			authorize: true,
			want:      want{code: http.StatusOK},
		},
		{
			name:      "invalid order number (Luhn)",
			method:    http.MethodPost,
			request:   request{Order: invalidOrder, Sum: 100},
			authorize: true,
			want:      want{code: http.StatusUnprocessableEntity},
		},
		{
			name:      "insufficient funds",
			method:    http.MethodPost,
			request:   request{Order: insufficientOrder, Sum: 1000},
			authorize: true,
			want:      want{code: http.StatusPaymentRequired},
		},
		{
			name:      "unauthorized",
			method:    http.MethodPost,
			request:   request{Order: validOrder, Sum: 100},
			authorize: false,
			want:      want{code: http.StatusInternalServerError},
		},
		{
			name:      "empty order",
			method:    http.MethodPost,
			request:   request{Order: "", Sum: 100},
			authorize: true,
			want:      want{code: http.StatusBadRequest},
		},
		{
			name:      "negative sum",
			method:    http.MethodPost,
			request:   request{Order: validOrder, Sum: -10},
			authorize: true,
			want:      want{code: http.StatusBadRequest},
		},
		{
			name:      "wrong method",
			method:    http.MethodGet,
			request:   request{Order: validOrder, Sum: 100},
			authorize: true,
			want:      want{code: http.StatusMethodNotAllowed},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(test.request)
			resp, _ := testRequest(t, ts, test.method, "/api/user/balance/withdraw", string(bodyBytes), test.authorize)
			defer resp.Body.Close()
			assert.Equal(t, test.want.code, resp.StatusCode)
		})
	}
}

// test getting all withdrawals for current user
func TestGetUserWithdrawals(t *testing.T) {
	withdrawalsData := []models.WithdrawData{
		{Order: "123", Sum: 100, ProcessedAt: models.TimeRFC3339{}},
	}
	withdrawalsJSON, err := json.MarshalIndent(withdrawalsData, "", "  ")
	require.NoError(t, err)

	mockService := NewService(t)
	mockService.EXPECT().GetAllWithdrawals(mock.Anything, testUserID).Return(withdrawalsData, nil).Once()
	mockService.EXPECT().GetAllWithdrawals(mock.Anything, testUserID).Return([]models.WithdrawData{}, nil).Once()
	mockService.EXPECT().GetAllWithdrawals(mock.Anything, testUserID).Return(nil, errors.New("db error")).Once()

	ts := httptest.NewServer(newAPITestRouter(mockService))
	defer ts.Close()

	type want struct {
		code     int
		response string
	}
	tests := []struct {
		name      string
		method    string
		authorize bool
		want      want
	}{
		{
			name:      "success with data",
			method:    http.MethodGet,
			authorize: true,
			want: want{
				code:     http.StatusOK,
				response: string(withdrawalsJSON),
			},
		},
		{
			name:      "no data",
			method:    http.MethodGet,
			authorize: true,
			want: want{
				code:     http.StatusNoContent,
				response: "",
			},
		},
		{
			name:      "service error",
			method:    http.MethodGet,
			authorize: true,
			want: want{
				code:     http.StatusInternalServerError,
				response: "",
			},
		},
		{
			name:      "unauthorized",
			method:    http.MethodGet,
			authorize: false,
			want: want{
				code:     http.StatusInternalServerError,
				response: "",
			},
		},
		{
			name:      "wrong method",
			method:    http.MethodPost,
			authorize: true,
			want: want{
				code:     http.StatusMethodNotAllowed,
				response: "",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, body := testRequest(t, ts, test.method, "/api/user/withdrawals", "", test.authorize)
			defer resp.Body.Close()
			assert.Equal(t, test.want.code, resp.StatusCode)
			if test.want.response != "" {
				assert.JSONEq(t, test.want.response, body)
			}
		})
	}
}
