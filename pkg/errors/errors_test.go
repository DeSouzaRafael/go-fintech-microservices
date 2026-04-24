package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNew(t *testing.T) {
	err := New(CodeNotFound, "resource missing")
	assert.Equal(t, CodeNotFound, err.Code)
	assert.Equal(t, "resource missing", err.Message)
	assert.Nil(t, err.Err)
	assert.Contains(t, err.Error(), "NOT_FOUND")
}

func TestWrap(t *testing.T) {
	cause := errors.New("db timeout")
	err := Wrap(CodeInternal, "query failed", cause)
	assert.Equal(t, CodeInternal, err.Code)
	assert.ErrorIs(t, err, cause)
	assert.Contains(t, err.Error(), "db timeout")
}

func TestToGRPC(t *testing.T) {
	cases := []struct {
		code     Code
		wantCode codes.Code
	}{
		{CodeNotFound, codes.NotFound},
		{CodeAlreadyExists, codes.AlreadyExists},
		{CodeInvalidArgument, codes.InvalidArgument},
		{CodePermissionDenied, codes.PermissionDenied},
		{CodeUnauthenticated, codes.Unauthenticated},
		{CodeInsufficientFunds, codes.FailedPrecondition},
		{CodeConflict, codes.AlreadyExists},
		{CodeInternal, codes.Internal},
	}

	for _, c := range cases {
		grpcErr := ToGRPC(New(c.code, "msg"))
		st, ok := status.FromError(grpcErr)
		require.True(t, ok)
		assert.Equal(t, c.wantCode, st.Code(), "code %s", c.code)
	}
}

func TestToGRPC_Nil(t *testing.T) {
	assert.Nil(t, ToGRPC(nil))
}

func TestToGRPC_NonDomainError(t *testing.T) {
	grpcErr := ToGRPC(errors.New("raw error"))
	st, _ := status.FromError(grpcErr)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestDomainError_Unwrap(t *testing.T) {
	cause := errors.New("root")
	err := Wrap(CodeInternal, "wrapped", cause)
	assert.ErrorIs(t, err, cause)
}
