package handlers

import (
	"github.com/max-marek-projects/loyalty-system/internal/service"
)

// Handler holds dependencies for HTTP handlers: service, worker limits, and cookie secret.
type Handler struct {
	service            service.Service
	maxParallelWorkers int
	secretKey          string
}

// NewHandler creates a new Handler instance with the given service, worker limit, and secret key.
// Parameters:
//   - service: business logic layer implementation.
//   - maxParallelWorkers: limit for concurrent background workers.
//   - secretKey: key used for signing auth cookies.
//
// Returns a pointer to the initialized Handler.
func NewHandler(service service.Service, maxParallelWorkers int, secretKey string) *Handler {
	return &Handler{service: service, maxParallelWorkers: maxParallelWorkers, secretKey: secretKey}
}
