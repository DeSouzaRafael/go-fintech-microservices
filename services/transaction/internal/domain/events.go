package domain

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventTransactionInitiated  EventType = "TransactionInitiated"
	EventTransactionCompleted  EventType = "TransactionCompleted"
	EventTransactionFailed     EventType = "TransactionFailed"
	EventTransactionCompensated EventType = "TransactionCompensated"
)

type OutboxEvent struct {
	ID          uuid.UUID
	Topic       string
	Key         string
	Payload     []byte
	EventType   EventType
	PublishedAt *time.Time
	CreatedAt   time.Time
}

type TransactionInitiatedPayload struct {
	TransactionID       uuid.UUID `json:"transaction_id"`
	SourceWalletID      uuid.UUID `json:"source_wallet_id"`
	DestinationWalletID uuid.UUID `json:"destination_wallet_id"`
	AmountCents         int64     `json:"amount_cents"`
	Description         string    `json:"description"`
}

type TransactionCompletedPayload struct {
	TransactionID       uuid.UUID `json:"transaction_id"`
	SourceWalletID      uuid.UUID `json:"source_wallet_id"`
	DestinationWalletID uuid.UUID `json:"destination_wallet_id"`
	AmountCents         int64     `json:"amount_cents"`
}

type TransactionFailedPayload struct {
	TransactionID uuid.UUID `json:"transaction_id"`
	SourceWalletID uuid.UUID `json:"source_wallet_id"`
	Reason        string    `json:"reason"`
}
