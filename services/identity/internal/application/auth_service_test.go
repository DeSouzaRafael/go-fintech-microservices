package application

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/identity/internal/domain"
)

type memUserRepo struct {
	byEmail map[string]domain.User
	byID    map[uuid.UUID]domain.User
}

func newMemUserRepo() *memUserRepo {
	return &memUserRepo{byEmail: map[string]domain.User{}, byID: map[uuid.UUID]domain.User{}}
}

func (r *memUserRepo) Save(_ context.Context, u *domain.User) error {
	r.byEmail[u.Email] = *u
	r.byID[u.ID] = *u
	return nil
}

func (r *memUserRepo) FindByEmail(_ context.Context, email string) (domain.User, error) {
	u, ok := r.byEmail[email]
	if !ok {
		return domain.User{}, apperrors.New(apperrors.CodeNotFound, "not found")
	}
	return u, nil
}

func (r *memUserRepo) FindByID(_ context.Context, id uuid.UUID) (domain.User, error) {
	u, ok := r.byID[id]
	if !ok {
		return domain.User{}, apperrors.New(apperrors.CodeNotFound, "not found")
	}
	return u, nil
}

type memTokenRepo struct {
	tokens map[string]domain.RefreshToken
}

func newMemTokenRepo() *memTokenRepo {
	return &memTokenRepo{tokens: map[string]domain.RefreshToken{}}
}

func (r *memTokenRepo) Save(_ context.Context, t *domain.RefreshToken) error {
	r.tokens[t.Token] = *t
	return nil
}

func (r *memTokenRepo) FindByToken(_ context.Context, token string) (domain.RefreshToken, error) {
	t, ok := r.tokens[token]
	if !ok {
		return domain.RefreshToken{}, apperrors.New(apperrors.CodeNotFound, "not found")
	}
	return t, nil
}

func (r *memTokenRepo) Delete(_ context.Context, token string) error {
	delete(r.tokens, token)
	return nil
}

func newTestService() *AuthService {
	return NewAuthService(newMemUserRepo(), newMemTokenRepo(), Config{
		JWTSecret:       "test-secret-key",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	})
}

func TestAuthService_Register(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	t.Run("registers new user", func(t *testing.T) {
		result, err := svc.Register(ctx, "user@test.com", "password123", "Test User")
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, result.UserID)
		assert.False(t, result.CreatedAt.IsZero())
	})

	t.Run("rejects duplicate email", func(t *testing.T) {
		_, err := svc.Register(ctx, "user@test.com", "other", "Other")
		require.Error(t, err)
		var de *apperrors.DomainError
		require.ErrorAs(t, err, &de)
		assert.Equal(t, apperrors.CodeAlreadyExists, de.Code)
	})
}

func TestAuthService_Login(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.Register(ctx, "login@test.com", "secret123", "Login User")
	require.NoError(t, err)

	t.Run("login success", func(t *testing.T) {
		result, err := svc.Login(ctx, "login@test.com", "secret123")
		require.NoError(t, err)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
		assert.False(t, result.ExpiresAt.IsZero())
	})

	t.Run("wrong password", func(t *testing.T) {
		_, err := svc.Login(ctx, "login@test.com", "wrongpassword")
		require.Error(t, err)
		var de *apperrors.DomainError
		require.ErrorAs(t, err, &de)
		assert.Equal(t, apperrors.CodeUnauthenticated, de.Code)
	})

	t.Run("unknown email", func(t *testing.T) {
		_, err := svc.Login(ctx, "ghost@test.com", "any")
		require.Error(t, err)
	})
}

func TestAuthService_ValidateToken(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.Register(ctx, "validate@test.com", "pass", "Val User")
	require.NoError(t, err)
	login, err := svc.Login(ctx, "validate@test.com", "pass")
	require.NoError(t, err)

	t.Run("valid token returns user", func(t *testing.T) {
		result, err := svc.ValidateToken(ctx, login.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, "validate@test.com", result.Email)
		assert.NotEqual(t, uuid.Nil, result.UserID)
	})

	t.Run("invalid token returns error", func(t *testing.T) {
		_, err := svc.ValidateToken(ctx, "not.a.token")
		require.Error(t, err)
	})
}

func TestAuthService_Refresh(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.Register(ctx, "refresh@test.com", "pass", "Ref User")
	require.NoError(t, err)
	login, err := svc.Login(ctx, "refresh@test.com", "pass")
	require.NoError(t, err)

	t.Run("refresh returns new access token", func(t *testing.T) {
		result, err := svc.Refresh(ctx, login.RefreshToken)
		require.NoError(t, err)
		assert.NotEmpty(t, result.AccessToken)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		_, err := svc.Refresh(ctx, "bad-token")
		require.Error(t, err)
	})
}

func TestAuthService_Logout(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.Register(ctx, "logout@test.com", "pass", "Logout User")
	require.NoError(t, err)
	login, err := svc.Login(ctx, "logout@test.com", "pass")
	require.NoError(t, err)

	t.Run("logout invalidates refresh token", func(t *testing.T) {
		require.NoError(t, svc.Logout(ctx, login.RefreshToken))

		_, err := svc.Refresh(ctx, login.RefreshToken)
		require.Error(t, err)
	})
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()
	_, err := svc.Register(ctx, "dup@test.com", "pass", "User")
	require.NoError(t, err)
	_, err = svc.Register(ctx, "dup@test.com", "pass", "User")
	require.Error(t, err)
	var de *apperrors.DomainError
	require.ErrorAs(t, err, &de)
	assert.Equal(t, apperrors.CodeAlreadyExists, de.Code)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()
	_, _ = svc.Register(ctx, "wrongpw@test.com", "correct", "User")
	_, err := svc.Login(ctx, "wrongpw@test.com", "wrong")
	require.Error(t, err)
}

func TestAuthService_Login_UnknownEmail(t *testing.T) {
	svc := newTestService()
	_, err := svc.Login(context.Background(), "nobody@test.com", "pass")
	require.Error(t, err)
}

func TestAuthService_Refresh_ExpiredToken(t *testing.T) {
	svc := NewAuthService(newMemUserRepo(), newMemTokenRepo(), Config{
		JWTSecret:       "test-secret-key",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: -time.Second,
	})
	ctx := context.Background()
	_, _ = svc.Register(ctx, "exp@test.com", "pass", "User")
	login, err := svc.Login(ctx, "exp@test.com", "pass")
	require.NoError(t, err)
	_, err = svc.Refresh(ctx, login.RefreshToken)
	require.Error(t, err)
}

func TestAuthService_ValidateToken_InvalidSubject(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()
	_, _ = svc.Register(ctx, "sub@test.com", "pass", "User")
	login, _ := svc.Login(ctx, "sub@test.com", "pass")
	_, err := svc.ValidateToken(ctx, login.AccessToken+"tampered")
	require.Error(t, err)
}
