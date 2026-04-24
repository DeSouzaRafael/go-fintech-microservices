package errors

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Code string

const (
	CodeNotFound           Code = "NOT_FOUND"
	CodeAlreadyExists      Code = "ALREADY_EXISTS"
	CodeInvalidArgument    Code = "INVALID_ARGUMENT"
	CodePermissionDenied   Code = "PERMISSION_DENIED"
	CodeUnauthenticated    Code = "UNAUTHENTICATED"
	CodeInsufficientFunds  Code = "INSUFFICIENT_FUNDS"
	CodeConflict           Code = "CONFLICT"
	CodeInternal           Code = "INTERNAL"
)

type DomainError struct {
	Code    Code
	Message string
	Err     error
}

func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error { return e.Err }

func New(code Code, message string) *DomainError {
	return &DomainError{Code: code, Message: message}
}

func Wrap(code Code, message string, err error) *DomainError {
	return &DomainError{Code: code, Message: message, Err: err}
}

func ToGRPC(err error) error {
	if err == nil {
		return nil
	}
	var de *DomainError
	if !errors.As(err, &de) {
		return status.Error(codes.Internal, err.Error())
	}
	switch de.Code {
	case CodeNotFound:
		return status.Error(codes.NotFound, de.Message)
	case CodeAlreadyExists:
		return status.Error(codes.AlreadyExists, de.Message)
	case CodeInvalidArgument:
		return status.Error(codes.InvalidArgument, de.Message)
	case CodePermissionDenied:
		return status.Error(codes.PermissionDenied, de.Message)
	case CodeUnauthenticated:
		return status.Error(codes.Unauthenticated, de.Message)
	case CodeInsufficientFunds:
		return status.Error(codes.FailedPrecondition, de.Message)
	case CodeConflict:
		return status.Error(codes.AlreadyExists, de.Message)
	default:
		return status.Error(codes.Internal, de.Message)
	}
}

var (
	Is  = errors.Is
	As  = errors.As
)
