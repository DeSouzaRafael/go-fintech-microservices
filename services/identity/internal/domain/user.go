package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserID = uuid.UUID

type User struct {
	ID           UserID
	Email        string
	PasswordHash string
	FullName     string
	CreatedAt    time.Time
}

func NewUser(email, passwordHash, fullName string) User {
	return User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		FullName:     fullName,
		CreatedAt:    time.Now().UTC(),
	}
}
