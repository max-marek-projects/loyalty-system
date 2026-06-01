package handlers

import (
	"github.com/max-marek-projects/loyalty-system/internal/service"
)

// Endpoints handler
type Handler struct {
	service            service.Service
	maxParallelWorkers int
	secretKey          string
}

// Get new endpoints handler
func NewHandler(service service.Service, maxParallelWorkers int, secretKey string) *Handler {
	return &Handler{service: service, maxParallelWorkers: maxParallelWorkers, secretKey: secretKey}
}
