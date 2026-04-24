package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/identity/internal/domain"
)

type Config struct {
	JWTSecret          string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
}

type AuthService struct {
	users  domain.UserRepository
	tokens domain.TokenRepository
	cfg    Config
}

func NewAuthService(users domain.UserRepository, tokens domain.TokenRepository, cfg Config) *AuthService {
	return &AuthService{users: users, tokens: tokens, cfg: cfg}
}

type RegisterResult struct {
	UserID    uuid.UUID
	CreatedAt time.Time
}

func (s *AuthService) Register(ctx context.Context, email, password, fullName string) (RegisterResult, error) {
	if _, err := s.users.FindByEmail(ctx, email); err == nil {
		return RegisterResult{}, errors.New(errors.CodeAlreadyExists, "email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return RegisterResult{}, errors.Wrap(errors.CodeInternal, "hash password", err)
	}

	user := domain.NewUser(email, string(hash), fullName)
	if err := s.users.Save(ctx, &user); err != nil {
		return RegisterResult{}, err
	}

	return RegisterResult{UserID: user.ID, CreatedAt: user.CreatedAt}, nil
}

type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

func (s *AuthService) Login(ctx context.Context, email, password string) (LoginResult, error) {
	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return LoginResult{}, errors.New(errors.CodeUnauthenticated, "invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return LoginResult{}, errors.New(errors.CodeUnauthenticated, "invalid credentials")
	}

	return s.issueTokens(ctx, user.ID)
}

type RefreshResult struct {
	AccessToken string
	ExpiresAt   time.Time
}

func (s *AuthService) Refresh(ctx context.Context, rawToken string) (RefreshResult, error) {
	stored, err := s.tokens.FindByToken(ctx, rawToken)
	if err != nil {
		return RefreshResult{}, errors.New(errors.CodeUnauthenticated, "invalid refresh token")
	}

	if stored.IsExpired() {
		_ = s.tokens.Delete(ctx, rawToken)
		return RefreshResult{}, errors.New(errors.CodeUnauthenticated, "refresh token expired")
	}

	accessToken, expiresAt, err := s.signAccessToken(stored.UserID)
	if err != nil {
		return RefreshResult{}, err
	}

	return RefreshResult{AccessToken: accessToken, ExpiresAt: expiresAt}, nil
}

type ValidateResult struct {
	UserID uuid.UUID
	Email  string
}

func (s *AuthService) ValidateToken(ctx context.Context, rawToken string) (ValidateResult, error) {
	claims, err := s.parseToken(rawToken)
	if err != nil {
		return ValidateResult{}, errors.New(errors.CodeUnauthenticated, "invalid token")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return ValidateResult{}, errors.New(errors.CodeUnauthenticated, "invalid token subject")
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return ValidateResult{}, errors.New(errors.CodeUnauthenticated, "user not found")
	}

	return ValidateResult{UserID: userID, Email: user.Email}, nil
}

func (s *AuthService) Logout(ctx context.Context, rawToken string) error {
	return s.tokens.Delete(ctx, rawToken)
}

func (s *AuthService) issueTokens(ctx context.Context, userID uuid.UUID) (LoginResult, error) {
	accessToken, expiresAt, err := s.signAccessToken(userID)
	if err != nil {
		return LoginResult{}, err
	}

	rawRefresh, err := generateSecureToken()
	if err != nil {
		return LoginResult{}, errors.Wrap(errors.CodeInternal, "generate refresh token", err)
	}

	refresh := domain.NewRefreshToken(userID, rawRefresh, s.cfg.RefreshTokenTTL)
	if err := s.tokens.Save(ctx, &refresh); err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresAt:    expiresAt,
	}, nil
}

func (s *AuthService) signAccessToken(userID uuid.UUID) (string, time.Time, error) {
	expiresAt := time.Now().UTC().Add(s.cfg.AccessTokenTTL)
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		Issuer:    "fintech-identity",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", time.Time{}, errors.Wrap(errors.CodeInternal, "sign token", err)
	}

	return signed, expiresAt, nil
}

func (s *AuthService) parseToken(raw string) (*jwt.RegisteredClaims, error) {
	token, err := jwt.ParseWithClaims(raw, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	return claims, nil
}

func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
