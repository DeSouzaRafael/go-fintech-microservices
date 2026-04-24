package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
)

type ProcessedEventRepository struct {
	db *sqlx.DB
}

func NewProcessedEventRepository(db *sqlx.DB) *ProcessedEventRepository {
	return &ProcessedEventRepository{db: db}
}

func (r *ProcessedEventRepository) IsProcessed(ctx context.Context, eventID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM notification_processed_events WHERE event_id = $1)`, eventID,
	).Scan(&exists)
	if err != nil {
		return false, errors.Wrap(errors.CodeInternal, "check processed event", err)
	}
	return exists, nil
}

func (r *ProcessedEventRepository) MarkProcessed(ctx context.Context, eventID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO notification_processed_events (event_id, processed_at) VALUES ($1, NOW()) ON CONFLICT DO NOTHING`,
		eventID,
	)
	if err != nil {
		return errors.Wrap(errors.CodeInternal, "mark processed", err)
	}
	return nil
}
