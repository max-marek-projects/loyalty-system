package auth

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/max-marek-projects/loyalty-system/internal/logger"
	"go.uber.org/zap"
)

const cookieName = "token"

type Claims struct {
	jwt.RegisteredClaims
	UserID int64
}

// create jwt and set it to cookie
func SetUserCookie(w http.ResponseWriter, userID int64, secretKey string) error {
	logger.Log.Info("Signing token with key", zap.String("key", secretKey))
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
		UserID: userID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func extractUserIDFromToken(token string, secretKey string) (int64, error) {
	claims := &Claims{}
	logger.Log.Info("Token and key", zap.String("token", token), zap.String("key", secretKey))
	tokenData, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})
	if err != nil {
		return 0, fmt.Errorf("Failed to parse jwt: %v", err)
	}
	if !tokenData.Valid {
		return 0, errors.New("invalid token")
	}
	return claims.UserID, nil
}

// get user id from request cookie
func GetUserIDFromRequest(r *http.Request, secretKey string) (int64, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return 0, err
	}
	return extractUserIDFromToken(cookie.Value, secretKey)
}
