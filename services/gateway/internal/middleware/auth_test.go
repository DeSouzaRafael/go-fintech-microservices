package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key"

func makeToken(secret, subject string, expiry time.Duration) string {
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	m := NewAuthMiddleware(testSecret)
	token := makeToken(testSecret, "user-123", time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/v1/wallets/abc/balance", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	m.Handler(okHandler()).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	m := NewAuthMiddleware(testSecret)

	req := httptest.NewRequest(http.MethodGet, "/v1/wallets/abc/balance", http.NoBody)
	rr := httptest.NewRecorder()

	m.Handler(okHandler()).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	m := NewAuthMiddleware(testSecret)

	req := httptest.NewRequest(http.MethodGet, "/v1/wallets/abc/balance", http.NoBody)
	req.Header.Set("Authorization", "Bearer not.a.valid.token")
	rr := httptest.NewRecorder()

	m.Handler(okHandler()).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	m := NewAuthMiddleware(testSecret)
	token := makeToken(testSecret, "user-123", -time.Minute)

	req := httptest.NewRequest(http.MethodGet, "/v1/wallets/abc/balance", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	m.Handler(okHandler()).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthMiddleware_WrongSecret(t *testing.T) {
	m := NewAuthMiddleware(testSecret)
	token := makeToken("other-secret", "user-123", time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/v1/wallets/abc/balance", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	m.Handler(okHandler()).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthMiddleware_PublicPaths(t *testing.T) {
	m := NewAuthMiddleware(testSecret)
	public := []string{"/v1/auth/login", "/v1/auth/register", "/v1/auth/refresh", "/healthz"}

	for _, path := range public {
		req := httptest.NewRequest(http.MethodPost, path, http.NoBody)
		rr := httptest.NewRecorder()
		m.Handler(okHandler()).ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "path %s should be public", path)
	}
}

func TestUserIDFromContext(t *testing.T) {
	m := NewAuthMiddleware(testSecret)
	token := makeToken(testSecret, "user-abc", time.Hour)

	var capturedUserID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/wallets/x", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	m.Handler(handler).ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "user-abc", capturedUserID)
}
