package application

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"

	"github.com/DeSouzaRafael/go-fintech-microservices/services/notification/internal/domain"
)

type NotificationService struct {
	processed domain.ProcessedEventRepository
	sender    domain.NotificationSender
	logger    *zap.Logger
}

func NewNotificationService(
	processed domain.ProcessedEventRepository,
	sender domain.NotificationSender,
	logger *zap.Logger,
) *NotificationService {
	return &NotificationService{processed: processed, sender: sender, logger: logger}
}

func (s *NotificationService) Handle(ctx context.Context, record *kgo.Record) error {
	eventType := headerValue(record.Headers, "event_type")
	rawEventID := headerValue(record.Headers, "event_id")

	eventID, err := uuid.Parse(rawEventID)
	if err != nil {
		s.logger.Warn("invalid event_id", zap.String("raw", rawEventID))
		return nil
	}

	already, err := s.processed.IsProcessed(ctx, eventID)
	if err != nil {
		return err
	}
	if already {
		return nil
	}

	var payload map[string]any
	if err := json.Unmarshal(record.Value, &payload); err != nil {
		return err
	}

	notif := domain.Notification{
		ID:        uuid.New(),
		EventType: domain.EventType(eventType),
		SentAt:    time.Now().UTC(),
		Payload:   flattenPayload(payload),
	}

	if err := s.sender.Send(ctx, &notif); err != nil {
		s.logger.Error("send notification",
			zap.String("event_type", eventType),
			zap.Error(err),
		)
		return err
	}

	return s.processed.MarkProcessed(ctx, eventID)
}

func flattenPayload(m map[string]any) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case string:
			result[k] = val
		default:
			data, _ := json.Marshal(v)
			result[k] = string(data)
		}
	}
	return result
}

func headerValue(headers []kgo.RecordHeader, key string) string {
	for _, h := range headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}
