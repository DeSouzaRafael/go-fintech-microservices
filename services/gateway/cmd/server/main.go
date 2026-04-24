package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/logger"
	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/tracing"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/gateway/internal/middleware"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/gateway/internal/router"
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
		ServiceName:    "gateway",
		ServiceVersion: "0.1.0",
		OTLPEndpoint:   getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		SampleRate:     1.0,
	})
	if err != nil {
		return fmt.Errorf("tracing setup: %w", err)
	}
	defer func() { _ = shutdown(ctx) }()

	gwMux, err := router.New(ctx, router.Config{
		IdentityAddr:    getEnv("IDENTITY_ADDR", "localhost:50052"),
		WalletAddr:      getEnv("WALLET_ADDR", "localhost:50053"),
		TransactionAddr: getEnv("TRANSACTION_ADDR", "localhost:50054"),
		QueryAddr:       getEnv("QUERY_ADDR", "localhost:50057"),
	})
	if err != nil {
		return fmt.Errorf("gateway router: %w", err)
	}

	jwtSecret := getEnv("JWT_SECRET", "change-me-in-production")
	authMw := middleware.NewAuthMiddleware(jwtSecret)
	handler := authMw.Handler(gwMux)

	addr := fmt.Sprintf(":%d", getEnvInt("PORT", 8080))
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Info("gateway listening", zap.String("addr", addr))

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
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
