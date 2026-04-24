package domain

import "context"

type UserRepository interface {
	Save(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (User, error)
	FindByID(ctx context.Context, id UserID) (User, error)
}

type TokenRepository interface {
	Save(ctx context.Context, token *RefreshToken) error
	FindByToken(ctx context.Context, token string) (RefreshToken, error)
	Delete(ctx context.Context, token string) error
}
