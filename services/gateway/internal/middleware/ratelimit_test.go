package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Skip("redis not available:", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func injectUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

func TestRateLimitMiddleware_NoUserID_PassThrough(t *testing.T) {
	client := newTestRedis(t)
	m := NewRateLimitMiddleware(client)

	req := httptest.NewRequest(http.MethodGet, "/v1/wallets/x", http.NoBody)
	rr := httptest.NewRecorder()

	m.Handler(okHandler()).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRateLimitMiddleware_WithUserID_AllowsUnderLimit(t *testing.T) {
	client := newTestRedis(t)
	userID := "rl-allow-" + time.Now().Format("150405.000")
	client.Del(context.Background(), "ratelimit:"+userID)
	t.Cleanup(func() { client.Del(context.Background(), "ratelimit:"+userID) })

	m := NewRateLimitMiddleware(client)
	req := httptest.NewRequest(http.MethodGet, "/v1/wallets/x", http.NoBody)
	req = req.WithContext(injectUserID(req.Context(), userID))
	rr := httptest.NewRecorder()

	m.Handler(okHandler()).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRateLimitMiddleware_ExceedsLimit_Returns429(t *testing.T) {
	client := newTestRedis(t)
	userID := "rl-block-" + time.Now().Format("150405.000")
	ctx := context.Background()
	key := "ratelimit:" + userID

	client.Set(ctx, key, rateMaxCalls+1, rateWindow)
	t.Cleanup(func() { client.Del(ctx, key) })

	m := NewRateLimitMiddleware(client)
	req := httptest.NewRequest(http.MethodGet, "/v1/wallets/x", http.NoBody)
	req = req.WithContext(injectUserID(req.Context(), userID))
	rr := httptest.NewRecorder()

	m.Handler(okHandler()).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

func TestRateLimitMiddleware_Increment(t *testing.T) {
	client := newTestRedis(t)
	m := NewRateLimitMiddleware(client)
	ctx := context.Background()
	key := "ratelimit:incr-" + time.Now().Format("150405.000000")
	t.Cleanup(func() { client.Del(ctx, key) })

	count, err := m.increment(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	count, err = m.increment(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}
