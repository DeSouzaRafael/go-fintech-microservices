package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"

	"go.uber.org/zap"

	apperrors "github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/fraud/internal/domain"
)

type memProfileRepo struct {
	profiles map[uuid.UUID]domain.UserProfile
}

func newMemProfileRepo() *memProfileRepo {
	return &memProfileRepo{profiles: map[uuid.UUID]domain.UserProfile{}}
}

func (r *memProfileRepo) GetProfile(_ context.Context, id uuid.UUID) (domain.UserProfile, error) {
	p, ok := r.profiles[id]
	if !ok {
		return domain.UserProfile{}, apperrors.New(apperrors.CodeNotFound, "not found")
	}
	return p, nil
}

func (r *memProfileRepo) UpdateProfile(_ context.Context, p domain.UserProfile) error {
	r.profiles[p.UserID] = p
	return nil
}

func makeKafkaRecord(eventType string, payload any) *kgo.Record {
	data, _ := json.Marshal(payload)
	return &kgo.Record{
		Value: data,
		Headers: []kgo.RecordHeader{
			{Key: "event_type", Value: []byte(eventType)},
		},
	}
}

func newUpdater(repo *memProfileRepo) *ProfileUpdater {
	return &ProfileUpdater{profiles: repo, logger: zap.NewNop()}
}

func TestProfileUpdater_TransactionCompleted_UpdatesProfile(t *testing.T) {
	repo := newMemProfileRepo()
	u := newUpdater(repo)
	ctx := context.Background()
	userID := uuid.New()

	r := makeKafkaRecord("TransactionCompleted", map[string]any{
		"transaction_id": uuid.New(),
		"source_wallet_id": uuid.New(),
		"destination_wallet_id": uuid.New(),
		"amount_cents": int64(5000),
		"user_id":      userID,
	})

	u.handle(ctx, r)

	profile, err := repo.GetProfile(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, int64(5000), profile.DailyTotalCents)
	assert.Equal(t, 1, profile.TxCountLastMinute)
}

func TestProfileUpdater_TransactionCompleted_Accumulates(t *testing.T) {
	repo := newMemProfileRepo()
	u := newUpdater(repo)
	ctx := context.Background()
	userID := uuid.New()

	for i := 0; i < 3; i++ {
		r := makeKafkaRecord("TransactionCompleted", map[string]any{
			"amount_cents": int64(1000),
			"user_id":      userID,
		})
		u.handle(ctx, r)
	}

	profile, err := repo.GetProfile(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, int64(3000), profile.DailyTotalCents)
	assert.Equal(t, 3, profile.TxCountLastMinute)
}

func TestProfileUpdater_OtherEvent_Ignored(t *testing.T) {
	repo := newMemProfileRepo()
	u := newUpdater(repo)
	ctx := context.Background()

	r := makeKafkaRecord("TransactionInitiated", map[string]any{
		"amount_cents": int64(5000),
		"user_id":      uuid.New(),
	})
	u.handle(ctx, r)

	assert.Empty(t, repo.profiles)
}

func TestProfileUpdater_NilUserID_Ignored(t *testing.T) {
	repo := newMemProfileRepo()
	u := newUpdater(repo)
	ctx := context.Background()

	r := makeKafkaRecord("TransactionCompleted", map[string]any{
		"amount_cents": int64(1000),
		"user_id":      uuid.Nil,
	})
	u.handle(ctx, r)

	assert.Empty(t, repo.profiles)
}

func TestProfileUpdater_BadJSON_Ignored(t *testing.T) {
	repo := newMemProfileRepo()
	u := newUpdater(repo)
	ctx := context.Background()

	r := &kgo.Record{
		Value: []byte("not json"),
		Headers: []kgo.RecordHeader{
			{Key: "event_type", Value: []byte("TransactionCompleted")},
		},
	}
	u.handle(ctx, r)

	assert.Empty(t, repo.profiles)
}

func TestHeaderValue(t *testing.T) {
	r := &kgo.Record{
		Headers: []kgo.RecordHeader{
			{Key: "foo", Value: []byte("bar")},
		},
	}
	assert.Equal(t, "bar", headerValue(r, "foo"))
	assert.Equal(t, "", headerValue(r, "missing"))
}

var _ = time.Now
