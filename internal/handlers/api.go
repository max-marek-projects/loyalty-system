// Package handlers provides HTTP handlers for the loyalty system API.
package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/max-marek-projects/loyalty-system/internal/auth"
	"github.com/max-marek-projects/loyalty-system/internal/logger"
	"github.com/max-marek-projects/loyalty-system/internal/models"
	"github.com/max-marek-projects/loyalty-system/internal/service"
)

// RegisterUser handles user registration.
// Reads login/password from JSON, creates a user, and sets an auth cookie.
func (h *Handler) RegisterUser(w http.ResponseWriter, r *http.Request) {

	var requestData models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		logger.Log.Debug("cannot decode request JSON body", slog.Any("error", err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if requestData.Login == "" || requestData.Password == "" {
		http.Error(w, "Login and password are required", http.StatusBadRequest)
		return
	}
	userID, err := h.service.RegisterUser(r.Context(), requestData)
	if err != nil {
		if errors.Is(err, service.ErrLoginAlreadyTaken) {
			http.Error(w, "Login already taken", http.StatusConflict)
			return
		}
		logger.Log.Error("Failed to register user", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if err := auth.SetUserCookie(w, userID, h.secretKey); err != nil {
		logger.Log.Error("Failed to set auth cookie", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// LoginUser handles user login.
// Validates credentials and sets an auth cookie on success.
func (h *Handler) LoginUser(w http.ResponseWriter, r *http.Request) {
	var requestData models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		logger.Log.Debug("cannot decode request JSON body", slog.Any("error", err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if requestData.Login == "" || requestData.Password == "" {
		http.Error(w, "Login and password are required", http.StatusBadRequest)
		return
	}
	userID, err := h.service.LoginUser(r.Context(), requestData)
	if err != nil {
		if errors.Is(err, service.ErrWrongUsernamePassword) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		logger.Log.Error("Failed to register user", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if err := auth.SetUserCookie(w, userID, h.secretKey); err != nil {
		logger.Log.Error("Failed to set auth cookie", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// NewOrder adds a new order number for the authenticated user.
// Reads the order number from request body and validates it.
func (h *Handler) NewOrder(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		http.Error(w, "Empty number", http.StatusBadRequest)
		return
	}
	userID, err := auth.GetUserIDFromRequest(r, h.secretKey)
	if err != nil {
		logger.Log.Error("Failed to get user id from context", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	added, err := h.service.AddOrder(r.Context(), userID, string(body))
	if err != nil {
		if errors.Is(err, service.ErrOrdersConflict) {
			http.Error(w, "Order number already taken by other user", http.StatusConflict)
			return
		}
		if errors.Is(err, service.ErrNumberNotValid) {
			http.Error(w, "Order number not valid", http.StatusUnprocessableEntity)
			return
		}
		logger.Log.Error("Failed add order to storage", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if added {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GetUserOrders returns all orders belonging to the authenticated user.
// Responds with JSON array or 204 No Content if none exist.
func (h *Handler) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromRequest(r, h.secretKey)
	if err != nil {
		logger.Log.Error("Failed to get user id from context", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	orders, err := h.service.GetAllOrders(r.Context(), userID)
	if err != nil {
		logger.Log.Error("failed to get user URLs", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if len(orders) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	prettyJSON, err := json.MarshalIndent(orders, "", "  ")
	if err != nil {
		logger.Log.Error("failed to encode response", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(prettyJSON)
	if err != nil {
		logger.Log.Error("failed to write response", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

// GetBalance returns the current user's loyalty balance.
// Responds with JSON containing current balance and withdrawn amount.
func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromRequest(r, h.secretKey)
	if err != nil {
		logger.Log.Error("Failed to get user id from context", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	balance, err := h.service.GetBalance(r.Context(), userID)
	if err != nil {
		logger.Log.Error("failed to get user balance", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	prettyJSON, err := json.MarshalIndent(balance, "", "  ")
	if err != nil {
		logger.Log.Error("failed to encode response", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(prettyJSON)
	if err != nil {
		logger.Log.Error("failed to write response", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

// Withdraw handles a withdrawal request from the user's balance.
// Validates order number and sum, then processes the withdrawal.
func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	var withdrawData models.WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&withdrawData); err != nil {
		logger.Log.Debug("cannot decode request JSON body", slog.Any("error", err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if withdrawData.Order == "" {
		http.Error(w, "Order number is required", http.StatusBadRequest)
		return
	}
	if withdrawData.Sum < 0 {
		http.Error(w, "Sum must be positive", http.StatusBadRequest)
		return
	}
	userID, err := auth.GetUserIDFromRequest(r, h.secretKey)
	if err != nil {
		logger.Log.Error("Failed to get user id from context", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	err = h.service.Withdraw(r.Context(), userID, withdrawData.Order, withdrawData.Sum)
	if err != nil {
		if errors.Is(err, service.ErrNumberNotValid) {
			http.Error(w, "Order number not valid", http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, service.ErrInsufficientFunds) {
			http.Error(w, "User has insufficient funds", http.StatusPaymentRequired)
			return
		}
		logger.Log.Error("failed to get user balance", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GetUserWithdrawals returns all withdrawal transactions for the authenticated user.
// Responds with JSON array or 204 No Content if none exist.
func (h *Handler) GetUserWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromRequest(r, h.secretKey)
	if err != nil {
		logger.Log.Error("Failed to get user id from context", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	withdrawals, err := h.service.GetAllWithdrawals(r.Context(), userID)
	if err != nil {
		logger.Log.Error("failed to get user URLs", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if len(withdrawals) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	prettyJSON, err := json.MarshalIndent(withdrawals, "", "  ")
	if err != nil {
		logger.Log.Error("failed to encode response", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(prettyJSON)
	if err != nil {
		logger.Log.Error("failed to write response", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
