package middlewares

import (
	"errors"
	"net/http"

	"github.com/max-marek-projects/loyalty-system/internal/auth"
	"github.com/max-marek-projects/loyalty-system/internal/logger"
	"go.uber.org/zap"
)

func AuthMiddleware(secretKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// check if userID is exists and is valid
			_, err := auth.GetUserIDFromRequest(r, secretKey)
			if err != nil {
				if !errors.Is(err, http.ErrNoCookie) {
					// invalid cookie -> 401 Unauthorized
					logger.Log.Error("Received invalid cookie", zap.Error(err))
				}
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
