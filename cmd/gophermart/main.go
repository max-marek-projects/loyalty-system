package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/max-marek-projects/loyalty-system/internal/config"
	"github.com/max-marek-projects/loyalty-system/internal/handlers"
	"github.com/max-marek-projects/loyalty-system/internal/logger"
	"github.com/max-marek-projects/loyalty-system/internal/repository"
	"github.com/max-marek-projects/loyalty-system/internal/server"
	"github.com/max-marek-projects/loyalty-system/internal/service"
)

// entry point
func main() {
	configData := config.LoadConfig()
	err := logger.Initialize(configData.LoggerLevel)
	if err != nil {
		logger.Log.Error("Unable to initialize logger", slog.Any("error", err))
		os.Exit(1)
	}
	store, err := repository.NewDBStorage(configData.DatabaseURI)
	if err != nil {
		logger.Log.Error("Unable to create storage", slog.Any("error", err))
		os.Exit(1)
	}
	service, err := service.NewEndpointService(store, configData.AccrualSystemAddress)
	if err != nil {
		logger.Log.Error("Unable to create service", slog.Any("error", err))
		os.Exit(1)
	}
	// create context with cancel for orders processing
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.StartOrderProcessor(ctx, configData.MaxParallelWorkers, configData.PollInterval, configData.MockExternalService)
	handler := handlers.NewHandler(service, configData.MaxParallelWorkers, configData.CookieSecret)
	srv := server.NewServer(configData.RunAddr, handler, configData.ReadTimeout, configData.WriteTimeout, configData.CookieSecret)

	// create separate goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	select {
	case sig := <-stop:
		logger.Log.Info("Shutdown signal received",
			slog.String("signal", sig.String()),
		)
		cancel()

		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelShutdown()
		if err := srv.Shutdown(ctxShutdown); err != nil {
			logger.Log.Error("Graceful shutdown failed", slog.Any("error", err))
		} else {
			logger.Log.Info("Server stopped gracefully")
		}
		// additional time for workers to shut down
		time.Sleep(2 * time.Second)

	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error("Server stopped with error", slog.Any("error", err))
			os.Exit(1)
		}
	}
}
