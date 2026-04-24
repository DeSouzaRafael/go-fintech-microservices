package logger

import (
	"context"

	"go.uber.org/zap"

	"github.com/DeSouzaRafael/go-fintech-microservices/services/notification/internal/domain"
)

type LogSender struct {
	logger *zap.Logger
}

func NewLogSender(logger *zap.Logger) *LogSender {
	return &LogSender{logger: logger}
}

func (s *LogSender) Send(_ context.Context, n *domain.Notification) error {
	s.logger.Info("notification dispatched",
		zap.String("notification_id", n.ID.String()),
		zap.String("event_type", string(n.EventType)),
		zap.Any("payload", n.Payload),
	)
	return nil
}
