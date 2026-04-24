package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/identity/internal/domain"
)

type tokenRow struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	Token     string    `db:"token"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

func (r *tokenRow) toDomain() domain.RefreshToken {
	return domain.RefreshToken{
		ID:        r.ID,
		UserID:    r.UserID,
		Token:     r.Token,
		ExpiresAt: r.ExpiresAt,
		CreatedAt: r.CreatedAt,
	}
}

type TokenRepository struct {
	db *sqlx.DB
}

func NewTokenRepository(db *sqlx.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

func (r *TokenRepository) Save(ctx context.Context, t *domain.RefreshToken) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		t.ID, t.UserID, t.Token, t.ExpiresAt, t.CreatedAt,
	)
	if err != nil {
		return errors.Wrap(errors.CodeInternal, "save refresh token", err)
	}
	return nil
}

func (r *TokenRepository) FindByToken(ctx context.Context, token string) (domain.RefreshToken, error) {
	var row tokenRow
	err := r.db.QueryRowxContext(ctx,
		`SELECT id, user_id, token, expires_at, created_at FROM refresh_tokens WHERE token = $1`,
		token,
	).StructScan(&row)
	if err == sql.ErrNoRows {
		return domain.RefreshToken{}, errors.New(errors.CodeNotFound, "token not found")
	}
	if err != nil {
		return domain.RefreshToken{}, errors.Wrap(errors.CodeInternal, "find refresh token", err)
	}
	return row.toDomain(), nil
}

func (r *TokenRepository) Delete(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE token = $1`, token)
	if err != nil {
		return errors.Wrap(errors.CodeInternal, "delete refresh token", err)
	}
	return nil
}
