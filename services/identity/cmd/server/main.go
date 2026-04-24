package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"

	identityv1 "github.com/DeSouzaRafael/go-fintech-microservices/api/proto/identity/v1"
	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/logger"
	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/middleware"
	pkgmetrics "github.com/DeSouzaRafael/go-fintech-microservices/pkg/metrics"
	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/server"
	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/tracing"
	identitygrpc "github.com/DeSouzaRafael/go-fintech-microservices/services/identity/internal/adapters/inbound/grpc"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/identity/internal/adapters/outbound/postgres"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/identity/internal/application"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	log := logger.MustNew(getEnv("LOG_LEVEL", "info"))
	defer func() { _ = log.Sync() }()

	ctx := context.Background()

	shutdown, err := tracing.Setup(ctx, tracing.Config{
		ServiceName:    "identity",
		ServiceVersion: "0.1.0",
		OTLPEndpoint:   getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		SampleRate:     1.0,
	})
	if err != nil {
		return fmt.Errorf("tracing setup: %w", err)
	}
	defer func() { _ = shutdown(ctx) }()

	metricsSrv, err := pkgmetrics.Setup(pkgmetrics.Config{
		ServiceName: "identity",
		Port:        getEnvInt("METRICS_PORT", 9102),
	})
	if err != nil {
		return fmt.Errorf("metrics setup: %w", err)
	}
	defer func() { _ = metricsSrv.Shutdown(ctx) }()

	db, err := sqlx.Connect("postgres", getEnv("DB_DSN", "postgres://fintech:fintech@localhost:15432/identity?sslmode=disable"))
	if err != nil {
		return fmt.Errorf("db connect: %w", err)
	}
	defer db.Close()

	userRepo := postgres.NewUserRepository(db)
	tokenRepo := postgres.NewTokenRepository(db)
	authSvc := application.NewAuthService(userRepo, tokenRepo, application.Config{
		JWTSecret:       getEnv("JWT_SECRET", "change-me"),
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	})
	grpcHandler := identitygrpc.NewServer(authSvc)

	chain := middleware.ChainUnary(
		middleware.UnaryRecovery(log),
		middleware.UnaryTracing(),
		middleware.UnaryLogging(log),
		middleware.UnaryIDempotency(),
	)

	srv := server.NewGRPC(
		server.Config{
			Port:            getEnvInt("PORT", 50052),
			ShutdownTimeout: 30 * time.Second,
		},
		log,
		middleware.WithUnaryInterceptor(chain),
	)

	identityv1.RegisterIdentityServiceServer(srv.Server(), grpcHandler)

	log.Info("starting server", zap.String("service", "identity"), zap.Int("port", getEnvInt("PORT", 50052)))

	return srv.Run(ctx)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
