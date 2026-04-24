package application

import (
	"context"
	"fmt"

	"github.com/DeSouzaRafael/go-fintech-microservices/services/fraud/internal/domain"
)

const (
	dailyLimitCents       = 10_000_000
	velocityMaxPerMin     = 10
	largeTransactionCents = 1_000_000
)

type FraudService struct {
	profiles domain.ProfileRepository
}

func NewFraudService(profiles domain.ProfileRepository) *FraudService {
	return &FraudService{profiles: profiles}
}

func (s *FraudService) Evaluate(ctx context.Context, req *domain.EvaluationRequest) (domain.EvaluationResult, error) {
	profile, err := s.profiles.GetProfile(ctx, req.UserID)
	if err != nil {
		profile = domain.UserProfile{UserID: req.UserID}
	}

	if result, failed := s.checkDailyLimit(req, &profile); failed {
		return result, nil
	}

	if result, failed := s.checkVelocity(req, &profile); failed {
		return result, nil
	}

	if result, review := s.checkLargeTransaction(req); review {
		return result, nil
	}

	return domain.EvaluationResult{
		TransactionID: req.TransactionID,
		Decision:      domain.DecisionApproved,
	}, nil
}

func (s *FraudService) checkDailyLimit(req *domain.EvaluationRequest, profile *domain.UserProfile) (domain.EvaluationResult, bool) {
	if profile.DailyTotalCents+req.AmountCents > dailyLimitCents {
		return domain.EvaluationResult{
			TransactionID: req.TransactionID,
			Decision:      domain.DecisionRejected,
			Reason:        fmt.Sprintf("daily limit exceeded: %d + %d > %d", profile.DailyTotalCents, req.AmountCents, dailyLimitCents),
		}, true
	}
	return domain.EvaluationResult{}, false
}

func (s *FraudService) checkVelocity(req *domain.EvaluationRequest, profile *domain.UserProfile) (domain.EvaluationResult, bool) {
	if profile.TxCountLastMinute >= velocityMaxPerMin {
		return domain.EvaluationResult{
			TransactionID: req.TransactionID,
			Decision:      domain.DecisionRejected,
			Reason:        fmt.Sprintf("velocity limit exceeded: %d txns in last minute", profile.TxCountLastMinute),
		}, true
	}
	return domain.EvaluationResult{}, false
}

func (s *FraudService) checkLargeTransaction(req *domain.EvaluationRequest) (domain.EvaluationResult, bool) {
	if req.AmountCents >= largeTransactionCents {
		return domain.EvaluationResult{
			TransactionID: req.TransactionID,
			Decision:      domain.DecisionReview,
			Reason:        fmt.Sprintf("large transaction: %d cents requires review", req.AmountCents),
		}, true
	}
	return domain.EvaluationResult{}, false
}
