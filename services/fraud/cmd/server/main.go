package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	fraudv1 "github.com/DeSouzaRafael/go-fintech-microservices/api/proto/fraud/v1"
	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/logger"
	pkgmetrics "github.com/DeSouzaRafael/go-fintech-microservices/pkg/metrics"
	pkgmw "github.com/DeSouzaRafael/go-fintech-microservices/pkg/middleware"
	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/server"
	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/tracing"
	fraudgrpc "github.com/DeSouzaRafael/go-fintech-microservices/services/fraud/internal/adapters/inbound/grpc"
	fraudkafka "github.com/DeSouzaRafael/go-fintech-microservices/services/fraud/internal/adapters/inbound/kafka"
	redisrepo "github.com/DeSouzaRafael/go-fintech-microservices/services/fraud/internal/adapters/outbound/redis"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/fraud/internal/application"
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
		ServiceName:    "fraud",
		ServiceVersion: "0.1.0",
		OTLPEndpoint:   getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		SampleRate:     1.0,
	})
	if err != nil {
		return fmt.Errorf("tracing setup: %w", err)
	}
	defer func() { _ = shutdown(ctx) }()

	redisClient := redis.NewClient(&redis.Options{
		Addr: getEnv("REDIS_ADDR", "localhost:6379"),
	})

	profileRepo := redisrepo.NewProfileRepository(redisClient)
	fraudSvc := application.NewFraudService(profileRepo)

	brokers := []string{getEnv("KAFKA_BROKERS", "localhost:9092")}
	updater, err := fraudkafka.NewProfileUpdater(profileRepo, brokers, log)
	if err != nil {
		return fmt.Errorf("kafka profile updater: %w", err)
	}
	go updater.Run(ctx)

	metricsSrv, err := pkgmetrics.Setup(pkgmetrics.Config{
		ServiceName: "fraud",
		Port:        getEnvInt("METRICS_PORT", 9104),
	})
	if err != nil {
		return fmt.Errorf("metrics setup: %w", err)
	}
	defer func() { _ = metricsSrv.Shutdown(ctx) }()

	chain := pkgmw.ChainUnary(
		pkgmw.UnaryRecovery(log),
		pkgmw.UnaryTracing(),
		pkgmw.UnaryLogging(log),
	)

	srv := server.NewGRPC(
		server.Config{
			Port:            getEnvInt("PORT", 50051),
			ShutdownTimeout: 30 * time.Second,
		},
		log,
		pkgmw.WithUnaryInterceptor(chain),
	)

	fraudv1.RegisterFraudServiceServer(srv.Server(), fraudgrpc.NewServer(fraudSvc))

	log.Info("starting server", zap.String("service", "fraud"), zap.Int("port", getEnvInt("PORT", 50051)))

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
