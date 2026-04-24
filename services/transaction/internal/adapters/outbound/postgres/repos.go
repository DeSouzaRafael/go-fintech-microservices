package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/transaction/internal/domain"
)

type txRow struct {
	ID                  uuid.UUID  `db:"id"`
	SourceWalletID      uuid.UUID  `db:"source_wallet_id"`
	DestinationWalletID uuid.UUID  `db:"destination_wallet_id"`
	AmountCents         int64      `db:"amount_cents"`
	Description         string     `db:"description"`
	Status              string     `db:"status"`
	FailureReason       string     `db:"failure_reason"`
	IdempotencyKey      string     `db:"idempotency_key"`
	CreatedAt           time.Time  `db:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at"`
}

func (r *txRow) toDomain() *domain.Transaction {
	return &domain.Transaction{
		ID:                  r.ID,
		SourceWalletID:      r.SourceWalletID,
		DestinationWalletID: r.DestinationWalletID,
		AmountCents:         r.AmountCents,
		Description:         r.Description,
		Status:              domain.Status(r.Status),
		FailureReason:       r.FailureReason,
		IdempotencyKey:      r.IdempotencyKey,
		CreatedAt:           r.CreatedAt,
		UpdatedAt:           r.UpdatedAt,
	}
}

type TransactionRepository struct {
	db *sqlx.DB
}

func NewTransactionRepository(db *sqlx.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Save(ctx context.Context, tx *domain.Transaction) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO transactions (id, source_wallet_id, destination_wallet_id, amount_cents, description, status, failure_reason, idempotency_key, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		tx.ID, tx.SourceWalletID, tx.DestinationWalletID, tx.AmountCents, tx.Description,
		string(tx.Status), tx.FailureReason, tx.IdempotencyKey, tx.CreatedAt, tx.UpdatedAt,
	)
	if err != nil {
		return errors.Wrap(errors.CodeInternal, "save transaction", err)
	}
	return nil
}

func (r *TransactionRepository) Update(ctx context.Context, tx *domain.Transaction) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE transactions SET status = $1, failure_reason = $2, updated_at = $3 WHERE id = $4`,
		string(tx.Status), tx.FailureReason, tx.UpdatedAt, tx.ID,
	)
	if err != nil {
		return errors.Wrap(errors.CodeInternal, "update transaction", err)
	}
	return nil
}

func (r *TransactionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	var row txRow
	err := r.db.QueryRowxContext(ctx,
		`SELECT id, source_wallet_id, destination_wallet_id, amount_cents, description, status, failure_reason, idempotency_key, created_at, updated_at
		 FROM transactions WHERE id = $1`, id,
	).StructScan(&row)
	if err == sql.ErrNoRows {
		return nil, errors.New(errors.CodeNotFound, "transaction not found")
	}
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternal, "find transaction", err)
	}
	return row.toDomain(), nil
}

func (r *TransactionRepository) FindByIdempotencyKey(ctx context.Context, key string) (*domain.Transaction, error) {
	var row txRow
	err := r.db.QueryRowxContext(ctx,
		`SELECT id, source_wallet_id, destination_wallet_id, amount_cents, description, status, failure_reason, idempotency_key, created_at, updated_at
		 FROM transactions WHERE idempotency_key = $1`, key,
	).StructScan(&row)
	if err == sql.ErrNoRows {
		return nil, errors.New(errors.CodeNotFound, "transaction not found")
	}
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternal, "find transaction by idempotency key", err)
	}
	return row.toDomain(), nil
}

type outboxRow struct {
	ID          uuid.UUID  `db:"id"`
	Topic       string     `db:"topic"`
	Key         string     `db:"key"`
	Payload     []byte     `db:"payload"`
	EventType   string     `db:"event_type"`
	PublishedAt *time.Time `db:"published_at"`
	CreatedAt   time.Time  `db:"created_at"`
}

type OutboxRepository struct {
	db *sqlx.DB
}

func NewOutboxRepository(db *sqlx.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) Save(ctx context.Context, event *domain.OutboxEvent) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO outbox_events (id, topic, key, payload, event_type, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		event.ID, event.Topic, event.Key, event.Payload, string(event.EventType), event.CreatedAt,
	)
	if err != nil {
		return errors.Wrap(errors.CodeInternal, "save outbox event", err)
	}
	return nil
}

func (r *OutboxRepository) FetchUnpublished(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	rows, err := r.db.QueryxContext(ctx,
		`SELECT id, topic, key, payload, event_type, published_at, created_at
		 FROM outbox_events WHERE published_at IS NULL ORDER BY created_at ASC LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternal, "fetch outbox events", err)
	}
	defer func() { _ = rows.Close() }()

	var events []domain.OutboxEvent
	for rows.Next() {
		var row outboxRow
		if err := rows.StructScan(&row); err != nil {
			return nil, errors.Wrap(errors.CodeInternal, "scan outbox event", err)
		}
		events = append(events, domain.OutboxEvent{
			ID:          row.ID,
			Topic:       row.Topic,
			Key:         row.Key,
			Payload:     row.Payload,
			EventType:   domain.EventType(row.EventType),
			PublishedAt: row.PublishedAt,
			CreatedAt:   row.CreatedAt,
		})
	}
	return events, rows.Err()
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE outbox_events SET published_at = NOW() WHERE id = $1`, id,
	)
	if err != nil {
		return errors.Wrap(errors.CodeInternal, "mark outbox event published", err)
	}
	return nil
}
