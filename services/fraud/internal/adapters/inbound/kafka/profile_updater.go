package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"

	"github.com/DeSouzaRafael/go-fintech-microservices/services/fraud/internal/domain"
)

type transactionCompletedPayload struct {
	TransactionID       uuid.UUID `json:"transaction_id"`
	SourceWalletID      uuid.UUID `json:"source_wallet_id"`
	DestinationWalletID uuid.UUID `json:"destination_wallet_id"`
	AmountCents         int64     `json:"amount_cents"`
	UserID              uuid.UUID `json:"user_id"`
}

type ProfileUpdater struct {
	profiles domain.ProfileRepository
	client   *kgo.Client
	logger   *zap.Logger
}

func NewProfileUpdater(profiles domain.ProfileRepository, brokers []string, logger *zap.Logger) (*ProfileUpdater, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup("fraud-profile-updater"),
		kgo.ConsumeTopics("transactions"),
	)
	if err != nil {
		return nil, err
	}
	return &ProfileUpdater{profiles: profiles, client: client, logger: logger}, nil
}

func (u *ProfileUpdater) Run(ctx context.Context) {
	defer u.client.Close()

	for {
		fetches := u.client.PollFetches(ctx)
		if ctx.Err() != nil {
			return
		}
		fetches.EachError(func(_ string, _ int32, err error) {
			u.logger.Error("fetch error", zap.Error(err))
		})
		fetches.EachRecord(func(r *kgo.Record) {
			u.handle(ctx, r)
		})
		if err := u.client.CommitUncommittedOffsets(ctx); err != nil {
			u.logger.Error("commit offsets", zap.Error(err))
		}
	}
}

func (u *ProfileUpdater) handle(ctx context.Context, r *kgo.Record) {
	eventType := headerValue(r, "event_type")
	if eventType != "TransactionCompleted" {
		return
	}

	var payload transactionCompletedPayload
	if err := json.Unmarshal(r.Value, &payload); err != nil {
		u.logger.Error("unmarshal TransactionCompleted", zap.Error(err))
		return
	}

	if payload.UserID == uuid.Nil {
		return
	}

	profile, err := u.profiles.GetProfile(ctx, payload.UserID)
	if err != nil {
		profile = domain.UserProfile{UserID: payload.UserID}
	}

	profile.DailyTotalCents += payload.AmountCents
	profile.TxCountLastMinute++
	profile.UpdatedAt = time.Now().UTC()

	if err := u.profiles.UpdateProfile(ctx, profile); err != nil {
		u.logger.Error("update fraud profile",
			zap.String("user_id", payload.UserID.String()),
			zap.Error(err),
		)
	}
}

func headerValue(r *kgo.Record, key string) string {
	for _, h := range r.Headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}
