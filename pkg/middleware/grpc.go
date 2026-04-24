package middleware

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/logger"
)

func UnaryLogging(l *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		log := logger.WithTrace(ctx, l)
		if err != nil {
			st, _ := status.FromError(err)
			log.Error("grpc request failed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.String("code", st.Code().String()),
				zap.String("error", st.Message()),
			)
		} else {
			log.Info("grpc request",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
			)
		}
		return resp, err
	}
}

func UnaryTracing() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		tracer := otel.Tracer("grpc-server")
		ctx, span := tracer.Start(ctx, info.FullMethod)
		defer span.End()

		span.SetAttributes(attribute.String("rpc.method", info.FullMethod))

		resp, err := handler(ctx, req)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return resp, err
	}
}

func UnaryRecovery(l *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				l.Error("panic recovered", zap.String("method", info.FullMethod), zap.Any("panic", r))
				err = status.Errorf(500, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

func UnaryIDempotency() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if keys := md.Get("idempotency-key"); len(keys) > 0 {
				ctx = context.WithValue(ctx, idempotencyKey{}, keys[0])
			}
		}
		return handler(ctx, req)
	}
}

type idempotencyKey struct{}

func IdempotencyKeyFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(idempotencyKey{}).(string); ok {
		return v
	}
	return ""
}

func ChainUnary(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			interceptor := interceptors[i]
			next := chain
			chain = func(ctx context.Context, req any) (any, error) {
				return interceptor(ctx, req, info, next)
			}
		}
		return chain(ctx, req)
	}
}

func WithUnaryInterceptor(i grpc.UnaryServerInterceptor) grpc.ServerOption {
	return grpc.UnaryInterceptor(i)
}
