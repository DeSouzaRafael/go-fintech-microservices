package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/wallet/internal/domain"
)

type outboxRow struct {
	ID          uuid.UUID  `db:"id"`
	Topic       string     `db:"topic"`
	Key         string     `db:"key"`
	Payload     []byte     `db:"payload"`
	EventType   string     `db:"event_type"`
	PublishedAt *time.Time `db:"published_at"`
	CreatedAt   time.Time  `db:"created_at"`
}

type WalletOutboxRepository struct {
	db *sqlx.DB
}

func NewWalletOutboxRepository(db *sqlx.DB) *WalletOutboxRepository {
	return &WalletOutboxRepository{db: db}
}

func (r *WalletOutboxRepository) Save(ctx context.Context, event *domain.WalletOutboxEvent) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO wallet_outbox_events (id, topic, key, payload, event_type, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		event.ID, event.Topic, event.Key, event.Payload, string(event.EventType), event.CreatedAt,
	)
	if err != nil {
		return errors.Wrap(errors.CodeInternal, "save wallet outbox event", err)
	}
	return nil
}

func (r *WalletOutboxRepository) FetchUnpublished(ctx context.Context, limit int) ([]domain.WalletOutboxEvent, error) {
	rows, err := r.db.QueryxContext(ctx,
		`SELECT id, topic, key, payload, event_type, published_at, created_at
		 FROM wallet_outbox_events WHERE published_at IS NULL ORDER BY created_at ASC LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternal, "fetch wallet outbox events", err)
	}
	defer func() { _ = rows.Close() }()

	var events []domain.WalletOutboxEvent
	for rows.Next() {
		var row outboxRow
		if err := rows.StructScan(&row); err != nil {
			return nil, errors.Wrap(errors.CodeInternal, "scan wallet outbox event", err)
		}
		events = append(events, domain.WalletOutboxEvent{
			ID:          row.ID,
			Topic:       row.Topic,
			Key:         row.Key,
			Payload:     row.Payload,
			EventType:   domain.OutboxEventType(row.EventType),
			PublishedAt: row.PublishedAt,
			CreatedAt:   row.CreatedAt,
		})
	}
	return events, rows.Err()
}

func (r *WalletOutboxRepository) MarkPublished(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE wallet_outbox_events SET published_at = NOW() WHERE id = $1`, id,
	)
	if err != nil {
		return errors.Wrap(errors.CodeInternal, "mark wallet outbox published", err)
	}
	return nil
}

type ProcessedEventRepository struct {
	db *sqlx.DB
}

func NewProcessedEventRepository(db *sqlx.DB) *ProcessedEventRepository {
	return &ProcessedEventRepository{db: db}
}

func (r *ProcessedEventRepository) IsProcessed(ctx context.Context, eventID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM processed_events WHERE event_id = $1)`, eventID,
	).Scan(&exists)
	if err != nil {
		return false, errors.Wrap(errors.CodeInternal, "check processed event", err)
	}
	return exists, nil
}

func (r *ProcessedEventRepository) MarkProcessed(ctx context.Context, eventID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO processed_events (event_id, processed_at) VALUES ($1, NOW()) ON CONFLICT DO NOTHING`,
		eventID,
	)
	if err != nil {
		return errors.Wrap(errors.CodeInternal, "mark event processed", err)
	}
	return nil
}
