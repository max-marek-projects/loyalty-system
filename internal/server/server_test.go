package server

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/max-marek-projects/loyalty-system/internal/handlers"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	mockService := NewService(t)
	handler := handlers.NewHandler(mockService, 5, "secret")
	srv := NewServer(":0", handler, time.Second, time.Second, "secret")
	assert.NotNil(t, srv)
	assert.Equal(t, ":0", srv.Addr)
	assert.NotNil(t, srv.Handler)
}

func TestServer_ListenAndServe(t *testing.T) {
	mockService := NewService(t)
	handler := handlers.NewHandler(mockService, 5, "secret")
	srv := NewServer(":0", handler, time.Second, time.Second, "secret")
	go func() {
		err := srv.ListenAndServe()
		assert.ErrorIs(t, err, http.ErrServerClosed)
	}()
	time.Sleep(10 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := srv.Shutdown(ctx)
	assert.NoError(t, err)
}
