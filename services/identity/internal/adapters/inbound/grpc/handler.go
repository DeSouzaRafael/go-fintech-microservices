package grpc

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	apperrors "github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/identity/internal/application"
)

type Handler struct {
	auth *application.AuthService
}

func NewHandler(auth *application.AuthService) *Handler {
	return &Handler{auth: auth}
}

type RegisterRequest struct {
	Email    string
	Password string
	FullName string
}

type RegisterResponse struct {
	UserID    string
	CreatedAt *timestamppb.Timestamp
}

func (h *Handler) Register(ctx context.Context, req RegisterRequest) (RegisterResponse, error) {
	if req.Email == "" || req.Password == "" {
		return RegisterResponse{}, apperrors.ToGRPC(apperrors.New(apperrors.CodeInvalidArgument, "email and password required"))
	}

	result, err := h.auth.Register(ctx, req.Email, req.Password, req.FullName)
	if err != nil {
		return RegisterResponse{}, apperrors.ToGRPC(err)
	}

	return RegisterResponse{
		UserID:    result.UserID.String(),
		CreatedAt: timestamppb.New(result.CreatedAt),
	}, nil
}

type LoginRequest struct {
	Email    string
	Password string
}

type LoginResponse struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    *timestamppb.Timestamp
}

func (h *Handler) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return LoginResponse{}, apperrors.ToGRPC(apperrors.New(apperrors.CodeInvalidArgument, "email and password required"))
	}

	result, err := h.auth.Login(ctx, req.Email, req.Password)
	if err != nil {
		return LoginResponse{}, apperrors.ToGRPC(err)
	}

	return LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    timestamppb.New(result.ExpiresAt),
	}, nil
}

type RefreshRequest struct {
	RefreshToken string
}

type RefreshResponse struct {
	AccessToken string
	ExpiresAt   *timestamppb.Timestamp
}

func (h *Handler) RefreshToken(ctx context.Context, req RefreshRequest) (RefreshResponse, error) {
	if req.RefreshToken == "" {
		return RefreshResponse{}, apperrors.ToGRPC(apperrors.New(apperrors.CodeInvalidArgument, "refresh token required"))
	}

	result, err := h.auth.Refresh(ctx, req.RefreshToken)
	if err != nil {
		return RefreshResponse{}, apperrors.ToGRPC(err)
	}

	return RefreshResponse{
		AccessToken: result.AccessToken,
		ExpiresAt:   timestamppb.New(result.ExpiresAt),
	}, nil
}

type ValidateRequest struct {
	Token string
}

type ValidateResponse struct {
	UserID string
	Email  string
	Valid  bool
}

func (h *Handler) ValidateToken(ctx context.Context, req ValidateRequest) (ValidateResponse, error) {
	result, err := h.auth.ValidateToken(ctx, req.Token)
	if err != nil {
		return ValidateResponse{Valid: false}, nil
	}

	return ValidateResponse{
		UserID: result.UserID.String(),
		Email:  result.Email,
		Valid:  true,
	}, nil
}

type LogoutRequest struct {
	RefreshToken string
}

func (h *Handler) Logout(ctx context.Context, req LogoutRequest) error {
	return apperrors.ToGRPC(h.auth.Logout(ctx, req.RefreshToken))
}
