package application

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/fraud/internal/domain"
)

type memProfileRepo struct {
	profiles map[uuid.UUID]domain.UserProfile
}

func newMemProfileRepo() *memProfileRepo {
	return &memProfileRepo{profiles: map[uuid.UUID]domain.UserProfile{}}
}

func (r *memProfileRepo) GetProfile(_ context.Context, userID uuid.UUID) (domain.UserProfile, error) {
	p, ok := r.profiles[userID]
	if !ok {
		return domain.UserProfile{}, apperrors.New(apperrors.CodeNotFound, "not found")
	}
	return p, nil
}

func (r *memProfileRepo) UpdateProfile(_ context.Context, p domain.UserProfile) error {
	r.profiles[p.UserID] = p
	return nil
}

func reqp(userID uuid.UUID, amountCents int64) *domain.EvaluationRequest {
	return &domain.EvaluationRequest{
		TransactionID: uuid.New(),
		UserID:        userID,
		WalletID:      uuid.New(),
		AmountCents:   amountCents,
		OccurredAt:    time.Now().UTC(),
	}
}

func TestFraudService_Evaluate_Approved(t *testing.T) {
	repo := newMemProfileRepo()
	svc := NewFraudService(repo)
	ctx := context.Background()

	result, err := svc.Evaluate(ctx, reqp(uuid.New(), 10000))
	require.NoError(t, err)
	assert.Equal(t, domain.DecisionApproved, result.Decision)
}

func TestFraudService_Evaluate_DailyLimitExceeded(t *testing.T) {
	repo := newMemProfileRepo()
	svc := NewFraudService(repo)
	ctx := context.Background()

	userID := uuid.New()
	repo.profiles[userID] = domain.UserProfile{
		UserID:          userID,
		DailyTotalCents: 9_500_000,
	}

	result, err := svc.Evaluate(ctx, reqp(userID, 600_000))
	require.NoError(t, err)
	assert.Equal(t, domain.DecisionRejected, result.Decision)
	assert.Contains(t, result.Reason, "daily limit")
}

func TestFraudService_Evaluate_VelocityExceeded(t *testing.T) {
	repo := newMemProfileRepo()
	svc := NewFraudService(repo)
	ctx := context.Background()

	userID := uuid.New()
	repo.profiles[userID] = domain.UserProfile{
		UserID:            userID,
		TxCountLastMinute: 10,
	}

	result, err := svc.Evaluate(ctx, reqp(userID, 100))
	require.NoError(t, err)
	assert.Equal(t, domain.DecisionRejected, result.Decision)
	assert.Contains(t, result.Reason, "velocity")
}

func TestFraudService_Evaluate_LargeTransactionReview(t *testing.T) {
	repo := newMemProfileRepo()
	svc := NewFraudService(repo)
	ctx := context.Background()

	result, err := svc.Evaluate(ctx, reqp(uuid.New(), 1_000_000))
	require.NoError(t, err)
	assert.Equal(t, domain.DecisionReview, result.Decision)
	assert.Contains(t, result.Reason, "large transaction")
}

func TestFraudService_Evaluate_NoProfile_Defaults_Approved(t *testing.T) {
	repo := newMemProfileRepo()
	svc := NewFraudService(repo)
	ctx := context.Background()

	result, err := svc.Evaluate(ctx, reqp(uuid.New(), 500))
	require.NoError(t, err)
	assert.Equal(t, domain.DecisionApproved, result.Decision)
}
