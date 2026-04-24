package postgres

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/identity/internal/domain"
)

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
	var t domain.RefreshToken
	err := r.db.QueryRowxContext(ctx,
		`SELECT id, user_id, token, expires_at, created_at FROM refresh_tokens WHERE token = $1`,
		token,
	).StructScan(&t)
	if err == sql.ErrNoRows {
		return domain.RefreshToken{}, errors.New(errors.CodeNotFound, "token not found")
	}
	if err != nil {
		return domain.RefreshToken{}, errors.Wrap(errors.CodeInternal, "find refresh token", err)
	}
	return t, nil
}

func (r *TokenRepository) Delete(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE token = $1`, token)
	if err != nil {
		return errors.Wrap(errors.CodeInternal, "delete refresh token", err)
	}
	return nil
}
