package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/max-marek-projects/loyalty-system/internal/handlers/mocks"
	"github.com/max-marek-projects/loyalty-system/internal/models"
	"github.com/max-marek-projects/loyalty-system/internal/service"
)

// create new router for handlers testing
func newAPITestRouter(service *mocks.Service) http.Handler {
	h := NewHandler(service, 100, "12345")

	r := chi.NewRouter()
	r.Route("/api", func(api chi.Router) {
		// public endpoints
		api.Post("/user/register", h.RegisterUser)
		api.Post("/user/login", h.LoginUser)
		// protected endpoints
		api.Get("/user/orders", h.GetUserOrders)
		api.Post("/user/orders", h.NewOrder)
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
	mockService := mocks.NewService(t)
	mockService.EXPECT().RegisterUser(mock.Anything, validUserData).Return(registeredUserID, nil)
	mockService.EXPECT().RegisterUser(mock.Anything, alreadyTakenUserData).Return(0, service.ErrorLoginAlreadyTaken)
	mockService.EXPECT().RegisterUser(mock.Anything, brokenUserData).Return(0, fmt.Errorf("some broken data"))
	mockService.EXPECT().RegisterUser(mock.Anything, validUserCookieErrorData).Return(userIDCookieError, nil)

	ts := httptest.NewServer(newAPITestRouter(mockService))
	defer ts.Close()

	type want struct {
		code         int
		errorMessage string
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
			resp, _ := testRequest(t, ts, test.method, "/api/user/register", test.request)
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
	mockService := mocks.NewService(t)
	mockService.EXPECT().LoginUser(mock.Anything, validUserData).Return(registeredUserID, nil)
	mockService.EXPECT().LoginUser(mock.Anything, wrongUserData).Return(0, service.ErrorWrongUsernamePassword)
	mockService.EXPECT().LoginUser(mock.Anything, brokenUserData).Return(0, fmt.Errorf("some broken data"))
	mockService.EXPECT().LoginUser(mock.Anything, validUserCookieErrorData).Return(userIDCookieError, nil)

	ts := httptest.NewServer(newAPITestRouter(mockService))
	defer ts.Close()

	type want struct {
		code         int
		errorMessage string
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
			resp, _ := testRequest(t, ts, test.method, "/api/user/login", test.request)
			defer resp.Body.Close()
			assert.Equal(t, test.want.code, resp.StatusCode)
		})
	}
}
