package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fraudv1 "github.com/DeSouzaRafael/go-fintech-microservices/api/proto/fraud/v1"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/fraud/internal/domain"
)

type stubEvaluator struct {
	result domain.EvaluationResult
	err    error
}

func (s *stubEvaluator) Evaluate(_ context.Context, _ *domain.EvaluationRequest) (domain.EvaluationResult, error) {
	return s.result, s.err
}

func newServer(decision domain.Decision) *Server {
	return NewServer(&stubEvaluator{
		result: domain.EvaluationResult{TransactionID: uuid.New(), Decision: decision},
	})
}

func validReq() *fraudv1.EvaluateRequest {
	return &fraudv1.EvaluateRequest{
		TransactionId: uuid.New().String(),
		UserId:        uuid.New().String(),
		WalletId:      uuid.New().String(),
		AmountCents:   5000,
	}
}

func TestServer_Evaluate_Approved(t *testing.T) {
	resp, err := newServer(domain.DecisionApproved).Evaluate(context.Background(), validReq())
	require.NoError(t, err)
	assert.Equal(t, fraudv1.Decision_DECISION_APPROVED, resp.Decision)
}

func TestServer_Evaluate_Rejected(t *testing.T) {
	resp, err := newServer(domain.DecisionRejected).Evaluate(context.Background(), validReq())
	require.NoError(t, err)
	assert.Equal(t, fraudv1.Decision_DECISION_REJECTED, resp.Decision)
}

func TestServer_Evaluate_Review(t *testing.T) {
	resp, err := newServer(domain.DecisionReview).Evaluate(context.Background(), validReq())
	require.NoError(t, err)
	assert.Equal(t, fraudv1.Decision_DECISION_REVIEW, resp.Decision)
}

func TestServer_Evaluate_InvalidTransactionID(t *testing.T) {
	req := validReq()
	req.TransactionId = "not-a-uuid"
	_, err := newServer(domain.DecisionApproved).Evaluate(context.Background(), req)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestServer_Evaluate_InvalidUserID(t *testing.T) {
	req := validReq()
	req.UserId = "bad"
	_, err := newServer(domain.DecisionApproved).Evaluate(context.Background(), req)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestServer_Evaluate_InvalidWalletID(t *testing.T) {
	req := validReq()
	req.WalletId = "bad"
	_, err := newServer(domain.DecisionApproved).Evaluate(context.Background(), req)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestServer_Evaluate_ServiceError(t *testing.T) {
	srv := NewServer(&stubEvaluator{err: errors.New("internal error")})
	_, err := srv.Evaluate(context.Background(), validReq())
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestServer_Evaluate_NilOccurredAt(t *testing.T) {
	req := validReq()
	req.OccurredAt = nil
	resp, err := newServer(domain.DecisionApproved).Evaluate(context.Background(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.TransactionId)
}

func TestToProtoDecision(t *testing.T) {
	cases := []struct {
		in  domain.Decision
		out fraudv1.Decision
	}{
		{domain.DecisionApproved, fraudv1.Decision_DECISION_APPROVED},
		{domain.DecisionRejected, fraudv1.Decision_DECISION_REJECTED},
		{domain.DecisionReview, fraudv1.Decision_DECISION_REVIEW},
		{"unknown", fraudv1.Decision_DECISION_UNSPECIFIED},
	}
	for _, c := range cases {
		assert.Equal(t, c.out, toProtoDecision(c.in))
	}
}
