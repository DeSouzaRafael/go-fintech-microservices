package kafka

import (
	"context"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"

	"github.com/DeSouzaRafael/go-fintech-microservices/services/transaction/internal/domain"
)

type OutboxWorker struct {
	outbox domain.OutboxRepository
	client *kgo.Client
	logger *zap.Logger
	ticker *time.Ticker
}

func NewOutboxWorker(outbox domain.OutboxRepository, brokers []string, logger *zap.Logger) (*OutboxWorker, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.RequiredAcks(kgo.AllISRAcks()),
	)
	if err != nil {
		return nil, err
	}

	return &OutboxWorker{
		outbox: outbox,
		client: client,
		logger: logger,
		ticker: time.NewTicker(500 * time.Millisecond),
	}, nil
}

func (w *OutboxWorker) Run(ctx context.Context) {
	defer w.client.Close()
	defer w.ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.ticker.C:
			w.flush(ctx)
		}
	}
}

func (w *OutboxWorker) flush(ctx context.Context) {
	events, err := w.outbox.FetchUnpublished(ctx, 100)
	if err != nil {
		w.logger.Error("fetch outbox events", zap.Error(err))
		return
	}

	for i := range events {
		e := &events[i]
		record := &kgo.Record{
			Topic: e.Topic,
			Key:   []byte(e.Key),
			Value: e.Payload,
			Headers: []kgo.RecordHeader{
				{Key: "event_type", Value: []byte(string(e.EventType))},
				{Key: "event_id", Value: []byte(e.ID.String())},
			},
		}

		if err := w.client.ProduceSync(ctx, record).FirstErr(); err != nil {
			w.logger.Error("publish outbox event",
				zap.String("event_id", e.ID.String()),
				zap.Error(err),
			)
			return
		}

		if err := w.outbox.MarkPublished(ctx, e.ID); err != nil {
			w.logger.Error("mark outbox published",
				zap.String("event_id", e.ID.String()),
				zap.Error(err),
			)
		}
	}
}
