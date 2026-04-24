package domain

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID
	UserID    UserID
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

func NewRefreshToken(userID UserID, token string, ttl time.Duration) RefreshToken {
	return RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().UTC().Add(ttl),
		CreatedAt: time.Now().UTC(),
	}
}

func (r *RefreshToken) IsExpired() bool {
	return time.Now().UTC().After(r.ExpiresAt)
}
