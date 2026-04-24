package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type Config struct {
	Port            int
	ShutdownTimeout time.Duration
}

type GRPCServer struct {
	srv    *grpc.Server
	cfg    Config
	logger *zap.Logger
}

func NewGRPC(cfg Config, logger *zap.Logger, opts ...grpc.ServerOption) *GRPCServer {
	srv := grpc.NewServer(opts...)
	reflection.Register(srv)

	healthSvc := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthSvc)

	return &GRPCServer{srv: srv, cfg: cfg, logger: logger}
}

func (s *GRPCServer) Server() *grpc.Server {
	return s.srv
}

func (s *GRPCServer) Run(ctx context.Context) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.Port))
	if err != nil {
		return fmt.Errorf("listen on port %d: %w", s.cfg.Port, err)
	}

	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("gRPC server started", zap.Int("port", s.cfg.Port))
		if err := s.srv.Serve(lis); err != nil {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		s.logger.Info("shutdown signal received", zap.String("signal", sig.String()))
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	stopped := make(chan struct{})
	go func() {
		s.srv.GracefulStop()
		close(stopped)
	}()

	timeout := s.cfg.ShutdownTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	select {
	case <-stopped:
		s.logger.Info("server stopped gracefully")
	case <-time.After(timeout):
		s.srv.Stop()
		s.logger.Warn("server forced stop after timeout")
	}

	return nil
}
