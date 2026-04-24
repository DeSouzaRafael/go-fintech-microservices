package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/logger"
	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/middleware"
	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/server"
	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/tracing"
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
		ServiceName:    "transaction",
		ServiceVersion: "0.1.0",
		OTLPEndpoint:   getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		SampleRate:     1.0,
	})
	if err != nil {
		return fmt.Errorf("tracing setup: %w", err)
	}
	defer func() { _ = shutdown(ctx) }()

	chain := middleware.ChainUnary(
		middleware.UnaryRecovery(log),
		middleware.UnaryTracing(),
		middleware.UnaryLogging(log),
		middleware.UnaryIDempotency(),
	)

	srv := server.NewGRPC(
		server.Config{
			Port:            getEnvInt("PORT", 50051),
			ShutdownTimeout: 30 * time.Second,
		},
		log,
		middleware.WithUnaryInterceptor(chain),
	)

	log.Info("starting server", zap.String("service", "transaction"), zap.Int("port", getEnvInt("PORT", 50051)))

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
