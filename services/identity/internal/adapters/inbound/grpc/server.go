package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	identityv1 "github.com/DeSouzaRafael/go-fintech-microservices/api/proto/identity/v1"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/identity/internal/application"
)

type Server struct {
	identityv1.UnimplementedIdentityServiceServer
	auth *application.AuthService
}

func NewServer(auth *application.AuthService) *Server {
	return &Server{auth: auth}
}

func (s *Server) Register(ctx context.Context, req *identityv1.RegisterRequest) (*identityv1.RegisterResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password required")
	}
	result, err := s.auth.Register(ctx, req.Email, req.Password, req.FullName)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &identityv1.RegisterResponse{
		UserId:    result.UserID.String(),
		CreatedAt: timestamppb.New(result.CreatedAt),
	}, nil
}

func (s *Server) Login(ctx context.Context, req *identityv1.LoginRequest) (*identityv1.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password required")
	}
	result, err := s.auth.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	return &identityv1.LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    timestamppb.New(result.ExpiresAt),
	}, nil
}

func (s *Server) RefreshToken(ctx context.Context, req *identityv1.RefreshTokenRequest) (*identityv1.RefreshTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token required")
	}
	result, err := s.auth.Refresh(ctx, req.RefreshToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}
	return &identityv1.RefreshTokenResponse{
		AccessToken: result.AccessToken,
		ExpiresAt:   timestamppb.New(result.ExpiresAt),
	}, nil
}

func (s *Server) ValidateToken(ctx context.Context, req *identityv1.ValidateTokenRequest) (*identityv1.ValidateTokenResponse, error) {
	result, err := s.auth.ValidateToken(ctx, req.Token)
	if err != nil {
		return &identityv1.ValidateTokenResponse{Valid: false}, nil
	}
	return &identityv1.ValidateTokenResponse{
		UserId: result.UserID.String(),
		Email:  result.Email,
		Valid:  true,
	}, nil
}

func (s *Server) Logout(ctx context.Context, req *identityv1.LogoutRequest) (*identityv1.LogoutResponse, error) {
	if err := s.auth.Logout(ctx, req.RefreshToken); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &identityv1.LogoutResponse{Success: true}, nil
}
