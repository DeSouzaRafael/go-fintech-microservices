package domain

import (
	"context"

	"github.com/google/uuid"
)

type TransactionRepository interface {
	Save(ctx context.Context, tx *Transaction) error
	Update(ctx context.Context, tx *Transaction) error
	FindByID(ctx context.Context, id uuid.UUID) (*Transaction, error)
	FindByIdempotencyKey(ctx context.Context, key string) (*Transaction, error)
}

type OutboxRepository interface {
	Save(ctx context.Context, event *OutboxEvent) error
	FetchUnpublished(ctx context.Context, limit int) ([]OutboxEvent, error)
	MarkPublished(ctx context.Context, id uuid.UUID) error
}
