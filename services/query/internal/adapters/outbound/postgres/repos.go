package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/query/internal/domain"
)

type balanceRow struct {
	WalletID     uuid.UUID `db:"wallet_id"`
	UserID       uuid.UUID `db:"user_id"`
	Currency     string    `db:"currency"`
	BalanceCents int64     `db:"balance_cents"`
	Version      int64     `db:"version"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type BalanceRepository struct{ db *sqlx.DB }

func NewBalanceRepository(db *sqlx.DB) *BalanceRepository { return &BalanceRepository{db: db} }

func (r *BalanceRepository) Upsert(ctx context.Context, p *domain.BalanceProjection) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO balance_projections (wallet_id, user_id, currency, balance_cents, version, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (wallet_id) DO UPDATE
		 SET balance_cents=EXCLUDED.balance_cents, version=EXCLUDED.version, updated_at=EXCLUDED.updated_at`,
		p.WalletID, p.UserID, p.Currency, p.BalanceCents, p.Version, p.UpdatedAt,
	)
	if err != nil {
		return errors.Wrap(errors.CodeInternal, "upsert balance", err)
	}
	return nil
}

func (r *BalanceRepository) FindByWalletID(ctx context.Context, walletID uuid.UUID) (*domain.BalanceProjection, error) {
	var row balanceRow
	err := r.db.QueryRowxContext(ctx,
		`SELECT wallet_id, user_id, currency, balance_cents, version, updated_at FROM balance_projections WHERE wallet_id=$1`,
		walletID,
	).StructScan(&row)
	if err == sql.ErrNoRows {
		return nil, errors.New(errors.CodeNotFound, "balance not found")
	}
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternal, "find balance", err)
	}
	return &domain.BalanceProjection{WalletID: row.WalletID, UserID: row.UserID, Currency: row.Currency, BalanceCents: row.BalanceCents, Version: row.Version, UpdatedAt: row.UpdatedAt}, nil
}

func (r *BalanceRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]domain.BalanceProjection, error) {
	rows, err := r.db.QueryxContext(ctx,
		`SELECT wallet_id, user_id, currency, balance_cents, version, updated_at FROM balance_projections WHERE user_id=$1`,
		userID,
	)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternal, "find balances by user", err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.BalanceProjection
	for rows.Next() {
		var row balanceRow
		if err := rows.StructScan(&row); err != nil {
			return nil, errors.Wrap(errors.CodeInternal, "scan balance", err)
		}
		result = append(result, domain.BalanceProjection{WalletID: row.WalletID, UserID: row.UserID, Currency: row.Currency, BalanceCents: row.BalanceCents, Version: row.Version, UpdatedAt: row.UpdatedAt})
	}
	return result, rows.Err()
}

type statementRow struct {
	EventID     uuid.UUID `db:"event_id"`
	WalletID    uuid.UUID `db:"wallet_id"`
	Type        string    `db:"type"`
	AmountCents int64     `db:"amount_cents"`
	Description string    `db:"description"`
	OccurredAt  time.Time `db:"occurred_at"`
}

type StatementRepository struct{ db *sqlx.DB }

func NewStatementRepository(db *sqlx.DB) *StatementRepository { return &StatementRepository{db: db} }

func (r *StatementRepository) Append(ctx context.Context, e *domain.StatementEntry) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO statement_entries (event_id, wallet_id, type, amount_cents, description, occurred_at)
		 VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`,
		e.EventID, e.WalletID, e.Type, e.AmountCents, e.Description, e.OccurredAt,
	)
	if err != nil {
		return errors.Wrap(errors.CodeInternal, "append statement", err)
	}
	return nil
}

func (r *StatementRepository) FindByWalletID(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]domain.StatementEntry, error) {
	rows, err := r.db.QueryxContext(ctx,
		`SELECT event_id, wallet_id, type, amount_cents, description, occurred_at
		 FROM statement_entries WHERE wallet_id=$1 ORDER BY occurred_at DESC LIMIT $2 OFFSET $3`,
		walletID, limit, offset,
	)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternal, "find statement", err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.StatementEntry
	for rows.Next() {
		var row statementRow
		if err := rows.StructScan(&row); err != nil {
			return nil, errors.Wrap(errors.CodeInternal, "scan statement", err)
		}
		result = append(result, domain.StatementEntry{EventID: row.EventID, WalletID: row.WalletID, Type: row.Type, AmountCents: row.AmountCents, Description: row.Description, OccurredAt: row.OccurredAt})
	}
	return result, rows.Err()
}

type statsRow struct {
	UserID             uuid.UUID `db:"user_id"`
	TotalTransactions  int64     `db:"total_transactions"`
	TotalDepositCents  int64     `db:"total_deposit_cents"`
	TotalWithdrawCents int64     `db:"total_withdraw_cents"`
	UpdatedAt          time.Time `db:"updated_at"`
}

type StatsRepository struct{ db *sqlx.DB }

func NewStatsRepository(db *sqlx.DB) *StatsRepository { return &StatsRepository{db: db} }

func (r *StatsRepository) Upsert(ctx context.Context, s *domain.UserStats) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_stats (user_id, total_transactions, total_deposit_cents, total_withdraw_cents, updated_at)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (user_id) DO UPDATE
		 SET total_transactions=EXCLUDED.total_transactions,
		     total_deposit_cents=EXCLUDED.total_deposit_cents,
		     total_withdraw_cents=EXCLUDED.total_withdraw_cents,
		     updated_at=EXCLUDED.updated_at`,
		s.UserID, s.TotalTransactions, s.TotalDepositCents, s.TotalWithdrawCents, s.UpdatedAt,
	)
	if err != nil {
		return errors.Wrap(errors.CodeInternal, "upsert stats", err)
	}
	return nil
}

func (r *StatsRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*domain.UserStats, error) {
	var row statsRow
	err := r.db.QueryRowxContext(ctx,
		`SELECT user_id, total_transactions, total_deposit_cents, total_withdraw_cents, updated_at FROM user_stats WHERE user_id=$1`,
		userID,
	).StructScan(&row)
	if err == sql.ErrNoRows {
		return nil, errors.New(errors.CodeNotFound, "stats not found")
	}
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternal, "find stats", err)
	}
	return &domain.UserStats{UserID: row.UserID, TotalTransactions: row.TotalTransactions, TotalDepositCents: row.TotalDepositCents, TotalWithdrawCents: row.TotalWithdrawCents, UpdatedAt: row.UpdatedAt}, nil
}

type ProcessedEventRepository struct{ db *sqlx.DB }

func NewProcessedEventRepository(db *sqlx.DB) *ProcessedEventRepository {
	return &ProcessedEventRepository{db: db}
}

func (r *ProcessedEventRepository) IsProcessed(ctx context.Context, eventID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM query_processed_events WHERE event_id=$1)`, eventID,
	).Scan(&exists)
	if err != nil {
		return false, errors.Wrap(errors.CodeInternal, "check processed", err)
	}
	return exists, nil
}

func (r *ProcessedEventRepository) MarkProcessed(ctx context.Context, eventID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO query_processed_events (event_id) VALUES ($1) ON CONFLICT DO NOTHING`, eventID,
	)
	if err != nil {
		return errors.Wrap(errors.CodeInternal, "mark processed", err)
	}
	return nil
}
