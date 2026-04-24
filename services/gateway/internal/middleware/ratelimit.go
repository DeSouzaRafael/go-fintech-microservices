package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	rateWindow   = time.Minute
	rateMaxCalls = 100
)

type RateLimitMiddleware struct {
	client *redis.Client
}

func NewRateLimitMiddleware(client *redis.Client) *RateLimitMiddleware {
	return &RateLimitMiddleware{client: client}
}

func (m *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := UserIDFromContext(r.Context())
		if userID == "" {
			next.ServeHTTP(w, r)
			return
		}

		key := fmt.Sprintf("ratelimit:%s", userID)
		count, err := m.increment(r.Context(), key)
		if err != nil || count > rateMaxCalls {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *RateLimitMiddleware) increment(ctx context.Context, key string) (int64, error) {
	pipe := m.client.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, rateWindow)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return incr.Val(), nil
}
