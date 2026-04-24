package application

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
	"github.com/DeSouzaRafael/go-fintech-microservices/services/query/internal/domain"
)

type memBalanceRepo struct{ m map[uuid.UUID]*domain.BalanceProjection }

func newMemBalanceRepo() *memBalanceRepo { return &memBalanceRepo{m: map[uuid.UUID]*domain.BalanceProjection{}} }
func (r *memBalanceRepo) Upsert(_ context.Context, p *domain.BalanceProjection) error {
	r.m[p.WalletID] = p; return nil
}
func (r *memBalanceRepo) FindByWalletID(_ context.Context, id uuid.UUID) (*domain.BalanceProjection, error) {
	if p, ok := r.m[id]; ok { return p, nil }
	return nil, apperrors.New(apperrors.CodeNotFound, "not found")
}
func (r *memBalanceRepo) FindByUserID(_ context.Context, uid uuid.UUID) ([]domain.BalanceProjection, error) {
	var out []domain.BalanceProjection
	for _, p := range r.m { if p.UserID == uid { out = append(out, *p) } }
	return out, nil
}

type memStatementRepo struct{ entries []domain.StatementEntry }

func (r *memStatementRepo) Append(_ context.Context, e *domain.StatementEntry) error {
	r.entries = append(r.entries, *e); return nil
}
func (r *memStatementRepo) FindByWalletID(_ context.Context, id uuid.UUID, limit, offset int) ([]domain.StatementEntry, error) {
	var out []domain.StatementEntry
	for _, e := range r.entries { if e.WalletID == id { out = append(out, e) } }
	if offset >= len(out) { return nil, nil }
	end := offset + limit
	if end > len(out) { end = len(out) }
	return out[offset:end], nil
}

type memStatsRepo struct{ m map[uuid.UUID]*domain.UserStats }

func newMemStatsRepo() *memStatsRepo { return &memStatsRepo{m: map[uuid.UUID]*domain.UserStats{}} }
func (r *memStatsRepo) Upsert(_ context.Context, s *domain.UserStats) error { r.m[s.UserID] = s; return nil }
func (r *memStatsRepo) FindByUserID(_ context.Context, id uuid.UUID) (*domain.UserStats, error) {
	if s, ok := r.m[id]; ok { return s, nil }
	return nil, apperrors.New(apperrors.CodeNotFound, "not found")
}

type memProcessedRepo struct{ ids map[uuid.UUID]bool }

func newMemProcessedRepo() *memProcessedRepo { return &memProcessedRepo{ids: map[uuid.UUID]bool{}} }
func (r *memProcessedRepo) IsProcessed(_ context.Context, id uuid.UUID) (bool, error) { return r.ids[id], nil }
func (r *memProcessedRepo) MarkProcessed(_ context.Context, id uuid.UUID) error { r.ids[id] = true; return nil }

func makeRecord(eventType, eventID string, payload any) *kgo.Record {
	data, _ := json.Marshal(payload)
	return &kgo.Record{Value: data, Headers: []kgo.RecordHeader{
		{Key: "event_type", Value: []byte(eventType)},
		{Key: "event_id", Value: []byte(eventID)},
	}}
}

func newSvc() *QueryService {
	return NewQueryService(newMemBalanceRepo(), &memStatementRepo{}, newMemStatsRepo(), newMemProcessedRepo(), zap.NewNop())
}

func TestQueryService_WalletCreated(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	walletID, userID := uuid.New(), uuid.New()

	r := makeRecord("WalletCreated", uuid.New().String(), map[string]any{
		"wallet_id": walletID, "user_id": userID, "currency": "BRL",
	})
	require.NoError(t, svc.HandleEvent(ctx, r))

	bal, err := svc.GetBalance(ctx, walletID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), bal.BalanceCents)
	assert.Equal(t, "BRL", bal.Currency)
}

func TestQueryService_FundsDeposited(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	walletID, userID := uuid.New(), uuid.New()

	_ = svc.HandleEvent(ctx, makeRecord("WalletCreated", uuid.New().String(), map[string]any{
		"wallet_id": walletID, "user_id": userID, "currency": "BRL",
	}))

	r := makeRecord("FundsDeposited", uuid.New().String(), map[string]any{
		"wallet_id": walletID, "amount_cents": int64(5000), "description": "dep",
		"version": int64(2), "event_id": uuid.New(), "occurred_at": time.Now(),
	})
	require.NoError(t, svc.HandleEvent(ctx, r))

	bal, _ := svc.GetBalance(ctx, walletID)
	assert.Equal(t, int64(5000), bal.BalanceCents)

	stmt, _ := svc.GetStatement(ctx, walletID, 10, 0)
	assert.Len(t, stmt, 1)
	assert.Equal(t, "DEPOSIT", stmt[0].Type)
}

func TestQueryService_FundsWithdrawn(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	walletID, userID := uuid.New(), uuid.New()

	_ = svc.HandleEvent(ctx, makeRecord("WalletCreated", uuid.New().String(), map[string]any{"wallet_id": walletID, "user_id": userID, "currency": "BRL"}))
	_ = svc.HandleEvent(ctx, makeRecord("FundsDeposited", uuid.New().String(), map[string]any{"wallet_id": walletID, "amount_cents": int64(8000), "description": "", "version": int64(2), "event_id": uuid.New(), "occurred_at": time.Now()}))
	_ = svc.HandleEvent(ctx, makeRecord("FundsWithdrawn", uuid.New().String(), map[string]any{"wallet_id": walletID, "amount_cents": int64(3000), "description": "spend", "version": int64(3), "event_id": uuid.New(), "occurred_at": time.Now()}))

	bal, _ := svc.GetBalance(ctx, walletID)
	assert.Equal(t, int64(5000), bal.BalanceCents)

	stmt, _ := svc.GetStatement(ctx, walletID, 10, 0)
	assert.Len(t, stmt, 2)
}

func TestQueryService_Idempotency(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	walletID, userID := uuid.New(), uuid.New()
	eventID := uuid.New().String()

	_ = svc.HandleEvent(ctx, makeRecord("WalletCreated", uuid.New().String(), map[string]any{"wallet_id": walletID, "user_id": userID, "currency": "BRL"}))

	r := makeRecord("FundsDeposited", eventID, map[string]any{"wallet_id": walletID, "amount_cents": int64(1000), "description": "", "version": int64(2), "event_id": uuid.New(), "occurred_at": time.Now()})
	require.NoError(t, svc.HandleEvent(ctx, r))
	require.NoError(t, svc.HandleEvent(ctx, r))

	bal, _ := svc.GetBalance(ctx, walletID)
	assert.Equal(t, int64(1000), bal.BalanceCents)
}

func TestQueryService_UnknownEvent_Ignored(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	r := makeRecord("SomeOtherEvent", uuid.New().String(), map[string]any{})
	require.NoError(t, svc.HandleEvent(ctx, r))
}
