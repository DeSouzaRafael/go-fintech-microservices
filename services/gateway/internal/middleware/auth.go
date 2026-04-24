package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDKey contextKey = "user_id"

type AuthMiddleware struct {
	secret string
}

func NewAuthMiddleware(secret string) *AuthMiddleware {
	return &AuthMiddleware{secret: secret}
}

func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		raw := r.Header.Get("Authorization")
		if raw == "" || !strings.HasPrefix(raw, "Bearer ") {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(raw, "Bearer ")
		claims, err := m.parseToken(tokenStr)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, claims.Subject)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) parseToken(raw string) (*jwt.RegisteredClaims, error) {
	token, err := jwt.ParseWithClaims(raw, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(m.secret), nil
	}, jwt.WithExpirationRequired(), jwt.WithTimeFunc(time.Now))
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}
	return claims, nil
}

func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

func isPublicPath(path string) bool {
	public := []string{"/v1/auth/login", "/v1/auth/register", "/v1/auth/refresh", "/healthz"}
	for _, p := range public {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
