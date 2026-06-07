package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret"

func TestSetAndGetUserID(t *testing.T) {
	var userID int64 = 12345
	w := httptest.NewRecorder()
	err := SetUserCookie(w, userID, testSecret)
	require.NoError(t, err)

	cookie := w.Result().Cookies()[0]
	assert.Equal(t, cookieName, cookie.Name)
	assert.Equal(t, "/", cookie.Path)
	assert.True(t, cookie.HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)

	gotID, err := extractUserIDFromToken(cookie.Value, testSecret)
	assert.NoError(t, err)
	assert.Equal(t, userID, gotID)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)
	gotID, err = GetUserIDFromRequest(req, testSecret)
	assert.NoError(t, err)
	assert.Equal(t, userID, gotID)
}

func TestGetUserIDFromRequest_NoCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	_, err := GetUserIDFromRequest(req, testSecret)
	assert.Error(t, err)
	assert.Equal(t, http.ErrNoCookie, err)
}

func TestExtractUserIDFromToken_InvalidFormat(t *testing.T) {
	_, err := extractUserIDFromToken("not-a-jwt-token", testSecret)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to parse jwt")
}

func TestExtractUserIDFromToken_WrongSignature(t *testing.T) {
	// Создаём токен с одним секретом
	validToken := createTestToken(12345, testSecret)
	// Пытаемся распарсить другим секретом
	_, err := extractUserIDFromToken(validToken, "wrong-secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to parse jwt")
}

func TestExtractUserIDFromToken_EmptyToken(t *testing.T) {
	_, err := extractUserIDFromToken("", testSecret)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to parse jwt")
}

func createTestToken(userID int64, secret string) string {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
		UserID: userID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}
