package postgres

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/identity/internal/domain"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Save(ctx context.Context, user *domain.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, full_name, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		user.ID, user.Email, user.PasswordHash, user.FullName, user.CreatedAt,
	)
	if err != nil {
		return errors.Wrap(errors.CodeInternal, "save user", err)
	}
	return nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	var u domain.User
	err := r.db.QueryRowxContext(ctx,
		`SELECT id, email, password_hash, full_name, created_at FROM users WHERE email = $1`,
		email,
	).StructScan(&u)
	if err == sql.ErrNoRows {
		return domain.User{}, errors.New(errors.CodeNotFound, "user not found")
	}
	if err != nil {
		return domain.User{}, errors.Wrap(errors.CodeInternal, "find user by email", err)
	}
	return u, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id domain.UserID) (domain.User, error) {
	var u domain.User
	err := r.db.QueryRowxContext(ctx,
		`SELECT id, email, password_hash, full_name, created_at FROM users WHERE id = $1`,
		id,
	).StructScan(&u)
	if err == sql.ErrNoRows {
		return domain.User{}, errors.New(errors.CodeNotFound, "user not found")
	}
	if err != nil {
		return domain.User{}, errors.Wrap(errors.CodeInternal, "find user by id", err)
	}
	return u, nil
}
