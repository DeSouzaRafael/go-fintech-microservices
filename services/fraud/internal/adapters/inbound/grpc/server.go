package grpc

import (
	"context"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	fraudv1 "github.com/DeSouzaRafael/go-fintech-microservices/api/proto/fraud/v1"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/fraud/internal/domain"
)

type Evaluator interface {
	Evaluate(ctx context.Context, req *domain.EvaluationRequest) (domain.EvaluationResult, error)
}

type Server struct {
	fraudv1.UnimplementedFraudServiceServer
	svc Evaluator
}

func NewServer(svc Evaluator) *Server {
	return &Server{svc: svc}
}

func (s *Server) Evaluate(ctx context.Context, req *fraudv1.EvaluateRequest) (*fraudv1.EvaluateResponse, error) {
	txID, err := uuid.Parse(req.TransactionId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid transaction_id: %v", err)
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}
	walletID, err := uuid.Parse(req.WalletId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid wallet_id: %v", err)
	}

	occurredAt := time.Now()
	if req.OccurredAt != nil {
		occurredAt = req.OccurredAt.AsTime()
	}

	result, err := s.svc.Evaluate(ctx, &domain.EvaluationRequest{
		TransactionID: txID,
		UserID:        userID,
		WalletID:      walletID,
		AmountCents:   req.AmountCents,
		OccurredAt:    occurredAt,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "evaluate: %v", err)
	}

	return &fraudv1.EvaluateResponse{
		TransactionId: result.TransactionID.String(),
		Decision:      toProtoDecision(result.Decision),
		Reason:        result.Reason,
	}, nil
}

func toProtoDecision(d domain.Decision) fraudv1.Decision {
	switch d {
	case domain.DecisionApproved:
		return fraudv1.Decision_DECISION_APPROVED
	case domain.DecisionRejected:
		return fraudv1.Decision_DECISION_REJECTED
	case domain.DecisionReview:
		return fraudv1.Decision_DECISION_REVIEW
	default:
		return fraudv1.Decision_DECISION_UNSPECIFIED
	}
}

var _ = (*timestamppb.Timestamp)(nil)
