package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/max-marek-projects/loyalty-system/internal/handlers"
	"github.com/max-marek-projects/loyalty-system/internal/logger"
	"github.com/max-marek-projects/loyalty-system/internal/middlewares"
	"go.uber.org/zap"
)

type Server struct {
	http.Server
}

func NewServer(addr string, h *handlers.Handler, readTimeout, writeTimeout time.Duration, cookieSecret string) *Server {
	r := chi.NewRouter()

	//middlewares
	r.Use(middleware.Recoverer)
	r.Use(middlewares.RequestsLogger)

	r.Route("/api/user", func(api chi.Router) {
		// public endpoints
		api.Post("/register", h.RegisterUser)
		api.Post("/login", h.LoginUser)
		// protected endpoints
		api.Group(func(protected chi.Router) {
			protected.Use(middlewares.AuthMiddleware(cookieSecret))
			protected.Get("/orders", h.GetUserOrders)
			protected.Post("/orders", h.NewOrder)
			protected.Get("/balance", h.GetBalance)
			protected.Post("/balance/withdraw", h.Withdraw)
			protected.Get("/withdrawals", h.GetUserWithdrawals)
		})
	})

	return &Server{
		Server: http.Server{
			Addr:         addr,
			Handler:      r,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
		},
	}
}

func (s *Server) ListenAndServe() error {
	logger.Log.Info("Starting server", zap.String("address", s.Addr))
	return s.Server.ListenAndServe()
}
