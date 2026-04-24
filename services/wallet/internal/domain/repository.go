package domain

import (
	"context"

	"github.com/google/uuid"
)

type EventStore interface {
	AppendEvents(ctx context.Context, walletID uuid.UUID, events []Event, expectedVersion int64) error
	LoadEvents(ctx context.Context, walletID uuid.UUID, afterVersion int64) ([]Event, error)
}

type SnapshotStore interface {
	SaveSnapshot(ctx context.Context, w *Wallet) error
	LoadSnapshot(ctx context.Context, walletID uuid.UUID) (*Wallet, int64, error)
}
