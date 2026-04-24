package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/wallet/internal/domain"
)

type EventStore struct {
	db *sqlx.DB
}

func NewEventStore(db *sqlx.DB) *EventStore {
	return &EventStore{db: db}
}

func (s *EventStore) AppendEvents(ctx context.Context, walletID uuid.UUID, events []domain.Event, expectedVersion int64) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(errors.CodeInternal, "begin transaction", err)
	}
	defer func() { _ = tx.Rollback() }()

	var currentVersion int64
	err = tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(version), 0) FROM wallet_events WHERE wallet_id = $1`,
		walletID,
	).Scan(&currentVersion)
	if err != nil {
		return errors.Wrap(errors.CodeInternal, "read current version", err)
	}

	if currentVersion != expectedVersion {
		return errors.New(errors.CodeConflict, fmt.Sprintf("optimistic concurrency conflict: expected version %d, got %d", expectedVersion, currentVersion))
	}

	for _, e := range events {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO wallet_events (id, wallet_id, type, payload, version, occured_at)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			e.ID, e.WalletID, string(e.Type), e.Payload, e.Version, e.OccuredAt,
		)
		if err != nil {
			return errors.Wrap(errors.CodeInternal, "append event", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(errors.CodeInternal, "commit transaction", err)
	}
	return nil
}

func (s *EventStore) LoadEvents(ctx context.Context, walletID uuid.UUID, afterVersion int64) ([]domain.Event, error) {
	rows, err := s.db.QueryxContext(ctx,
		`SELECT id, wallet_id, type, payload, version, occured_at
		 FROM wallet_events
		 WHERE wallet_id = $1 AND version > $2
		 ORDER BY version ASC`,
		walletID, afterVersion,
	)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternal, "load events", err)
	}
	defer func() { _ = rows.Close() }()

	var events []domain.Event
	for rows.Next() {
		var e domain.Event
		var eventType string
		if err := rows.Scan(&e.ID, &e.WalletID, &eventType, &e.Payload, &e.Version, &e.OccuredAt); err != nil {
			return nil, errors.Wrap(errors.CodeInternal, "scan event", err)
		}
		e.Type = domain.EventType(eventType)
		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(errors.CodeInternal, "iterate events", err)
	}

	return events, nil
}

type SnapshotStore struct {
	db *sqlx.DB
}

func NewSnapshotStore(db *sqlx.DB) *SnapshotStore {
	return &SnapshotStore{db: db}
}

func (s *SnapshotStore) SaveSnapshot(ctx context.Context, w *domain.Wallet) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO wallet_snapshots (wallet_id, balance_cents, reserved, currency, user_id, version, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())
		 ON CONFLICT (wallet_id) DO UPDATE
		 SET balance_cents = EXCLUDED.balance_cents,
		     reserved = EXCLUDED.reserved,
		     version = EXCLUDED.version,
		     created_at = EXCLUDED.created_at`,
		w.ID, w.BalanceCents, w.Reserved, w.Currency, w.UserID, w.Version,
	)
	if err != nil {
		return errors.Wrap(errors.CodeInternal, "save snapshot", err)
	}
	return nil
}

func (s *SnapshotStore) LoadSnapshot(ctx context.Context, walletID uuid.UUID) (*domain.Wallet, int64, error) {
	row := s.db.QueryRowxContext(ctx,
		`SELECT wallet_id, balance_cents, reserved, currency, user_id, version
		 FROM wallet_snapshots WHERE wallet_id = $1`,
		walletID,
	)

	var w domain.Wallet
	err := row.Scan(&w.ID, &w.BalanceCents, &w.Reserved, &w.Currency, &w.UserID, &w.Version)
	if err == sql.ErrNoRows {
		return nil, 0, errors.New(errors.CodeNotFound, "no snapshot")
	}
	if err != nil {
		return nil, 0, errors.Wrap(errors.CodeInternal, "load snapshot", err)
	}

	return &w, w.Version, nil
}
