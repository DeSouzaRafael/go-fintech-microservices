package kafka

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/sony/gobreaker"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"

	pkgbreaker "github.com/DeSouzaRafael/go-fintech-microservices/pkg/breaker"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/wallet/internal/domain"
)

type OutboxWorker struct {
	outbox  domain.WalletOutboxRepository
	client  *kgo.Client
	logger  *zap.Logger
	ticker  *time.Ticker
	breaker *gobreaker.CircuitBreaker
}

func NewOutboxWorker(outbox domain.WalletOutboxRepository, brokers []string, logger *zap.Logger) (*OutboxWorker, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.RequiredAcks(kgo.AllISRAcks()),
	)
	if err != nil {
		return nil, err
	}

	return &OutboxWorker{
		outbox:  outbox,
		client:  client,
		logger:  logger,
		ticker:  time.NewTicker(500 * time.Millisecond),
		breaker: pkgbreaker.New("wallet-kafka-publish"),
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

		if err := w.publishWithRetry(ctx, record); err != nil {
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

func (w *OutboxWorker) publishWithRetry(ctx context.Context, record *kgo.Record) error {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 100 * time.Millisecond
	bo.MaxInterval = 5 * time.Second
	bo.MaxElapsedTime = 30 * time.Second

	return backoff.Retry(func() error {
		_, err := w.breaker.Execute(func() (any, error) {
			return nil, w.client.ProduceSync(ctx, record).FirstErr()
		})
		return err
	}, backoff.WithContext(bo, ctx))
}
