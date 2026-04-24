package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Decision string

const (
	DecisionApproved Decision = "APPROVED"
	DecisionRejected Decision = "REJECTED"
	DecisionReview   Decision = "REVIEW"
)

type EvaluationRequest struct {
	TransactionID  uuid.UUID
	UserID         uuid.UUID
	WalletID       uuid.UUID
	AmountCents    int64
	OccurredAt     time.Time
}

type EvaluationResult struct {
	TransactionID uuid.UUID
	Decision      Decision
	Reason        string
}

type UserProfile struct {
	UserID            uuid.UUID
	DailyTotalCents   int64
	TxCountLastMinute int
	UpdatedAt         time.Time
}

type ProfileRepository interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (UserProfile, error)
	UpdateProfile(ctx context.Context, profile UserProfile) error
}
