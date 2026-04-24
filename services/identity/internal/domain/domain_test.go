package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewUser(t *testing.T) {
	u := NewUser("a@b.com", "hash", "Full Name")
	assert.NotEqual(t, uuid.Nil, u.ID)
	assert.Equal(t, "a@b.com", u.Email)
	assert.Equal(t, "hash", u.PasswordHash)
	assert.Equal(t, "Full Name", u.FullName)
	assert.False(t, u.CreatedAt.IsZero())
}

func TestNewRefreshToken(t *testing.T) {
	userID := uuid.New()
	token := NewRefreshToken(userID, "raw-token", 24*time.Hour)
	assert.NotEqual(t, uuid.Nil, token.ID)
	assert.Equal(t, userID, token.UserID)
	assert.Equal(t, "raw-token", token.Token)
	assert.False(t, token.ExpiresAt.IsZero())
	assert.False(t, token.IsExpired())
}

func TestRefreshToken_IsExpired(t *testing.T) {
	t.Run("not expired", func(t *testing.T) {
		tok := NewRefreshToken(uuid.New(), "tok", time.Hour)
		assert.False(t, tok.IsExpired())
	})

	t.Run("expired", func(t *testing.T) {
		tok := RefreshToken{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Token:     "old",
			ExpiresAt: time.Now().UTC().Add(-time.Minute),
		}
		assert.True(t, tok.IsExpired())
	})
}
