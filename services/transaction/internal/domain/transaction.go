package domain

import (
	"time"

	"github.com/google/uuid"

	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
)

type Status string

const (
	StatusPending     Status = "PENDING"
	StatusCompleted   Status = "COMPLETED"
	StatusFailed      Status = "FAILED"
	StatusCompensated Status = "COMPENSATED"
)

type Transaction struct {
	ID                  uuid.UUID
	SourceWalletID      uuid.UUID
	DestinationWalletID uuid.UUID
	AmountCents         int64
	Description         string
	Status              Status
	FailureReason       string
	IdempotencyKey      string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func NewTransaction(sourceWalletID, destWalletID uuid.UUID, amountCents int64, description, idempotencyKey string) (*Transaction, error) {
	if amountCents <= 0 {
		return nil, errors.New(errors.CodeInvalidArgument, "amount must be positive")
	}
	if sourceWalletID == destWalletID {
		return nil, errors.New(errors.CodeInvalidArgument, "source and destination wallets must differ")
	}

	now := time.Now().UTC()
	return &Transaction{
		ID:                  uuid.New(),
		SourceWalletID:      sourceWalletID,
		DestinationWalletID: destWalletID,
		AmountCents:         amountCents,
		Description:         description,
		Status:              StatusPending,
		IdempotencyKey:      idempotencyKey,
		CreatedAt:           now,
		UpdatedAt:           now,
	}, nil
}

func (t *Transaction) Complete() {
	t.Status = StatusCompleted
	t.UpdatedAt = time.Now().UTC()
}

func (t *Transaction) Fail(reason string) {
	t.Status = StatusFailed
	t.FailureReason = reason
	t.UpdatedAt = time.Now().UTC()
}

func (t *Transaction) Compensate() {
	t.Status = StatusCompensated
	t.UpdatedAt = time.Now().UTC()
}
