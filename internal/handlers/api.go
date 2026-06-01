package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/max-marek-projects/loyalty-system/internal/auth"
	"github.com/max-marek-projects/loyalty-system/internal/logger"
	"github.com/max-marek-projects/loyalty-system/internal/models"
	"github.com/max-marek-projects/loyalty-system/internal/service"
	"go.uber.org/zap"
)

// register user in service
func (h *Handler) RegisterUser(w http.ResponseWriter, r *http.Request) {

	var requestData models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if requestData.Login == "" || requestData.Password == "" {
		http.Error(w, "Login and password are required", http.StatusBadRequest)
		return
	}
	userID, err := h.service.RegisterUser(r.Context(), requestData)
	if err != nil {
		if errors.Is(err, service.ErrorLoginAlreadyTaken) {
			http.Error(w, "Login already taken", http.StatusConflict)
			return
		}
		logger.Log.Error("Failed to register user", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if err := auth.SetUserCookie(w, userID, h.secretKey); err != nil {
		logger.Log.Error("Failed to set auth cookie", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// login user in service
func (h *Handler) LoginUser(w http.ResponseWriter, r *http.Request) {
	var requestData models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if requestData.Login == "" || requestData.Password == "" {
		http.Error(w, "Login and password are required", http.StatusBadRequest)
		return
	}
	userID, err := h.service.LoginUser(r.Context(), requestData)
	if err != nil {
		if errors.Is(err, service.ErrorWrongUsernamePassword) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		logger.Log.Error("Failed to register user", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if err := auth.SetUserCookie(w, userID, h.secretKey); err != nil {
		logger.Log.Error("Failed to set auth cookie", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// add new order number
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
		logger.Log.Error("Failed to get user id from context", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	added, err := h.service.AddOrder(r.Context(), userID, string(body))
	if err != nil {
		if errors.Is(err, service.ErrorOrdersConflict) {
			http.Error(w, "Order number already taken by other user", http.StatusConflict)
			return
		}
		if errors.Is(err, service.ErrorNumberNotValid) {
			http.Error(w, "Order number not valid", http.StatusUnprocessableEntity)
			return
		}
		logger.Log.Error("Failed add order to storage", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if added {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromRequest(r, h.secretKey)
	if err != nil {
		logger.Log.Error("Failed to get user id from context", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	orders, err := h.service.GetAllOrders(r.Context(), userID)
	if err != nil {
		logger.Log.Error("failed to get user URLs", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	prettyJSON, err := json.MarshalIndent(orders, "", "  ")
	if err != nil {
		logger.Log.Error("failed to encode response", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(prettyJSON)
	if err != nil {
		logger.Log.Error("failed to write response", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
