package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventTransactionCompleted EventType = "TransactionCompleted"
	EventTransactionFailed    EventType = "TransactionFailed"
	EventFraudDetected        EventType = "FraudDetected"
)

type Notification struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	EventType EventType
	Payload   map[string]string
	SentAt    time.Time
}

type ProcessedEventRepository interface {
	IsProcessed(ctx context.Context, eventID uuid.UUID) (bool, error)
	MarkProcessed(ctx context.Context, eventID uuid.UUID) error
}

type NotificationSender interface {
	Send(ctx context.Context, n *Notification) error
}
