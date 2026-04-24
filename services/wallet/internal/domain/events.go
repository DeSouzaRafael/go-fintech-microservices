package domain

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventWalletCreated  EventType = "WalletCreated"
	EventFundsDeposited EventType = "FundsDeposited"
	EventFundsWithdrawn EventType = "FundsWithdrawn"
	EventFundsReserved  EventType = "FundsReserved"
	EventFundsReleased  EventType = "FundsReleased"
)

type Event struct {
	ID        uuid.UUID
	WalletID  uuid.UUID
	Type      EventType
	Payload   []byte
	Version   int64
	OccuredAt time.Time
}

type WalletCreatedPayload struct {
	UserID   uuid.UUID `json:"user_id"`
	Currency string    `json:"currency"`
}

type FundsDepositedPayload struct {
	AmountCents int64  `json:"amount_cents"`
	Description string `json:"description"`
}

type FundsWithdrawnPayload struct {
	AmountCents int64  `json:"amount_cents"`
	Description string `json:"description"`
}

type FundsReservedPayload struct {
	AmountCents   int64     `json:"amount_cents"`
	TransactionID uuid.UUID `json:"transaction_id"`
}

type FundsReleasedPayload struct {
	AmountCents   int64     `json:"amount_cents"`
	TransactionID uuid.UUID `json:"transaction_id"`
}
