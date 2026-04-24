package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type BalanceProjection struct {
	WalletID     uuid.UUID
	UserID       uuid.UUID
	Currency     string
	BalanceCents int64
	Version      int64
	UpdatedAt    time.Time
}

type StatementEntry struct {
	EventID     uuid.UUID
	WalletID    uuid.UUID
	Type        string
	AmountCents int64
	Description string
	OccurredAt  time.Time
}

type UserStats struct {
	UserID            uuid.UUID
	TotalTransactions int64
	TotalDepositCents int64
	TotalWithdrawCents int64
	UpdatedAt         time.Time
}

type BalanceRepository interface {
	Upsert(ctx context.Context, p *BalanceProjection) error
	FindByWalletID(ctx context.Context, walletID uuid.UUID) (*BalanceProjection, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]BalanceProjection, error)
}

type StatementRepository interface {
	Append(ctx context.Context, entry *StatementEntry) error
	FindByWalletID(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]StatementEntry, error)
}

type StatsRepository interface {
	Upsert(ctx context.Context, s *UserStats) error
	FindByUserID(ctx context.Context, userID uuid.UUID) (*UserStats, error)
}

type ProcessedEventRepository interface {
	IsProcessed(ctx context.Context, eventID uuid.UUID) (bool, error)
	MarkProcessed(ctx context.Context, eventID uuid.UUID) error
}
