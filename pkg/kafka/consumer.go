package kafka

import (
	"context"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

type Handler func(ctx context.Context, record *kgo.Record) error

type Consumer struct {
	client  *kgo.Client
	handler Handler
	logger  *zap.Logger
}

func NewConsumer(brokers []string, group, topic string, logger *zap.Logger) (*Consumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(topic),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return nil, err
	}

	return &Consumer{client: client, logger: logger}, nil
}

func (c *Consumer) SetHandler(h Handler) {
	c.handler = h
}

func (c *Consumer) Run(ctx context.Context) {
	defer c.client.Close()

	for {
		fetches := c.client.PollFetches(ctx)
		if ctx.Err() != nil {
			return
		}

		fetches.EachError(func(t string, p int32, err error) {
			c.logger.Error("fetch error", zap.String("topic", t), zap.Int32("partition", p), zap.Error(err))
		})

		fetches.EachRecord(func(record *kgo.Record) {
			if err := c.handler(ctx, record); err != nil {
				c.logger.Error("handle record",
					zap.String("topic", record.Topic),
					zap.Int64("offset", record.Offset),
					zap.Error(err),
				)
				return
			}
			c.client.MarkCommitRecords(record)
		})

		if err := c.client.CommitMarkedOffsets(ctx); err != nil {
			c.logger.Error("commit offsets", zap.Error(err))
		}
	}
}
