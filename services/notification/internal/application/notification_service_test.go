package application

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"

	apperrors "github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/notification/internal/domain"
)

type memProcessedRepo struct {
	ids map[uuid.UUID]bool
}

func newMemProcessedRepo() *memProcessedRepo {
	return &memProcessedRepo{ids: map[uuid.UUID]bool{}}
}

func (r *memProcessedRepo) IsProcessed(_ context.Context, id uuid.UUID) (bool, error) {
	return r.ids[id], nil
}

func (r *memProcessedRepo) MarkProcessed(_ context.Context, id uuid.UUID) error {
	r.ids[id] = true
	return nil
}

type captureSender struct {
	sent []domain.Notification
	err  error
}

func (s *captureSender) Send(_ context.Context, n *domain.Notification) error {
	if s.err != nil {
		return s.err
	}
	s.sent = append(s.sent, *n)
	return nil
}

func makeRecord(eventType, eventID string, payload any) *kgo.Record {
	data, _ := json.Marshal(payload)
	return &kgo.Record{
		Value: data,
		Headers: []kgo.RecordHeader{
			{Key: "event_type", Value: []byte(eventType)},
			{Key: "event_id", Value: []byte(eventID)},
		},
	}
}

func newTestService(sender domain.NotificationSender) *NotificationService {
	return NewNotificationService(newMemProcessedRepo(), sender, zap.NewNop())
}

func TestNotificationService_Handle_TransactionCompleted(t *testing.T) {
	sender := &captureSender{}
	svc := newTestService(sender)
	ctx := context.Background()

	record := makeRecord("TransactionCompleted", uuid.New().String(), map[string]any{
		"transaction_id": uuid.New().String(),
		"amount_cents":   5000,
	})

	err := svc.Handle(ctx, record)
	require.NoError(t, err)
	require.Len(t, sender.sent, 1)
	assert.Equal(t, domain.EventTransactionCompleted, sender.sent[0].EventType)
}

func TestNotificationService_Handle_Idempotency(t *testing.T) {
	sender := &captureSender{}
	svc := newTestService(sender)
	ctx := context.Background()

	eventID := uuid.New().String()
	record := makeRecord("TransactionFailed", eventID, map[string]any{"reason": "fraud"})

	require.NoError(t, svc.Handle(ctx, record))
	require.NoError(t, svc.Handle(ctx, record))

	assert.Len(t, sender.sent, 1)
}

func TestNotificationService_Handle_FraudDetected(t *testing.T) {
	sender := &captureSender{}
	svc := newTestService(sender)
	ctx := context.Background()

	record := makeRecord("FraudDetected", uuid.New().String(), map[string]any{
		"user_id": uuid.New().String(),
		"reason":  "velocity exceeded",
	})

	err := svc.Handle(ctx, record)
	require.NoError(t, err)
	assert.Equal(t, domain.EventFraudDetected, sender.sent[0].EventType)
}

func TestNotificationService_Handle_InvalidEventID(t *testing.T) {
	sender := &captureSender{}
	svc := newTestService(sender)
	ctx := context.Background()

	record := makeRecord("TransactionCompleted", "not-a-uuid", map[string]any{})
	err := svc.Handle(ctx, record)
	require.NoError(t, err)
	assert.Empty(t, sender.sent)
}

func TestNotificationService_Handle_SenderError(t *testing.T) {
	sender := &captureSender{err: apperrors.New(apperrors.CodeInternal, "smtp down")}
	svc := newTestService(sender)
	ctx := context.Background()

	record := makeRecord("TransactionCompleted", uuid.New().String(), map[string]any{})
	err := svc.Handle(ctx, record)
	require.Error(t, err)
}
