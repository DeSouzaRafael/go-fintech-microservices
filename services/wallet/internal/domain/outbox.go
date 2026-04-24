package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type OutboxEventType string

const (
	OutboxFundsReserved  OutboxEventType = "FundsReserved"
	OutboxFundsReleased  OutboxEventType = "FundsReleased"
	OutboxFundsDeposited OutboxEventType = "FundsDeposited"
)

type WalletOutboxEvent struct {
	ID          uuid.UUID
	Topic       string
	Key         string
	Payload     []byte
	EventType   OutboxEventType
	PublishedAt *time.Time
	CreatedAt   time.Time
}

type WalletOutboxRepository interface {
	Save(ctx context.Context, event *WalletOutboxEvent) error
	FetchUnpublished(ctx context.Context, limit int) ([]WalletOutboxEvent, error)
	MarkPublished(ctx context.Context, id uuid.UUID) error
}

type ProcessedEventRepository interface {
	IsProcessed(ctx context.Context, eventID uuid.UUID) (bool, error)
	MarkProcessed(ctx context.Context, eventID uuid.UUID) error
}

type OutboxFundsReservedPayload struct {
	WalletID      uuid.UUID `json:"wallet_id"`
	TransactionID uuid.UUID `json:"transaction_id"`
	AmountCents   int64     `json:"amount_cents"`
}

type OutboxFundsDepositedPayload struct {
	WalletID      uuid.UUID `json:"wallet_id"`
	TransactionID uuid.UUID `json:"transaction_id"`
	AmountCents   int64     `json:"amount_cents"`
}
