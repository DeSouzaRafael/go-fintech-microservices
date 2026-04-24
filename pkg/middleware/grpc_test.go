package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var testInfo = &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

func okHandler(_ context.Context, req any) (any, error) { return req, nil }
func errHandler(_ context.Context, _ any) (any, error)  { return nil, errors.New("boom") }
func panicHandler(_ context.Context, _ any) (any, error) { panic("test panic") }

func TestUnaryLogging_Success(t *testing.T) {
	i := UnaryLogging(zap.NewNop())
	resp, err := i(context.Background(), "req", testInfo, okHandler)
	require.NoError(t, err)
	assert.Equal(t, "req", resp)
}

func TestUnaryLogging_Error(t *testing.T) {
	i := UnaryLogging(zap.NewNop())
	_, err := i(context.Background(), "req", testInfo, errHandler)
	require.Error(t, err)
}

func TestUnaryTracing_Success(t *testing.T) {
	i := UnaryTracing()
	resp, err := i(context.Background(), "req", testInfo, okHandler)
	require.NoError(t, err)
	assert.Equal(t, "req", resp)
}

func TestUnaryTracing_Error(t *testing.T) {
	i := UnaryTracing()
	_, err := i(context.Background(), "req", testInfo, errHandler)
	require.Error(t, err)
}

func TestUnaryRecovery_NoPanic(t *testing.T) {
	i := UnaryRecovery(zap.NewNop())
	resp, err := i(context.Background(), "req", testInfo, okHandler)
	require.NoError(t, err)
	assert.Equal(t, "req", resp)
}

func TestUnaryRecovery_Panic(t *testing.T) {
	i := UnaryRecovery(zap.NewNop())
	resp, err := i(context.Background(), "req", testInfo, panicHandler)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "internal server error")
}

func TestUnaryIDempotency_WithKey(t *testing.T) {
	i := UnaryIDempotency()
	md := metadata.New(map[string]string{"idempotency-key": "abc-123"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	var capturedKey string
	handler := func(ctx context.Context, req any) (any, error) {
		capturedKey = IdempotencyKeyFromContext(ctx)
		return req, nil
	}

	_, err := i(ctx, "req", testInfo, handler)
	require.NoError(t, err)
	assert.Equal(t, "abc-123", capturedKey)
}

func TestUnaryIDempotency_WithoutKey(t *testing.T) {
	i := UnaryIDempotency()
	var capturedKey string
	handler := func(ctx context.Context, req any) (any, error) {
		capturedKey = IdempotencyKeyFromContext(ctx)
		return req, nil
	}
	_, err := i(context.Background(), "req", testInfo, handler)
	require.NoError(t, err)
	assert.Equal(t, "", capturedKey)
}

func TestIdempotencyKeyFromContext_Empty(t *testing.T) {
	assert.Equal(t, "", IdempotencyKeyFromContext(context.Background()))
}

func TestChainUnary(t *testing.T) {
	order := []int{}
	make := func(n int) grpc.UnaryServerInterceptor {
		return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, h grpc.UnaryHandler) (any, error) {
			order = append(order, n)
			return h(ctx, req)
		}
	}
	chain := ChainUnary(make(1), make(2), make(3))
	_, err := chain(context.Background(), nil, testInfo, okHandler)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, order)
}

func TestWithUnaryInterceptor(t *testing.T) {
	i := UnaryLogging(zap.NewNop())
	opt := WithUnaryInterceptor(i)
	assert.NotNil(t, opt)
}
