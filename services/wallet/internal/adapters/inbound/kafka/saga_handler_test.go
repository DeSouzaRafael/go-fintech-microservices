package kafka_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	sagahandler "github.com/DeSouzaRafael/go-fintech-microservices/services/wallet/internal/adapters/inbound/kafka"
	pgadapter "github.com/DeSouzaRafael/go-fintech-microservices/services/wallet/internal/adapters/outbound/postgres"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/wallet/internal/application"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/wallet/internal/domain"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func setupDB(t *testing.T) *sqlx.DB {
	t.Helper()
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("wallet_saga_test"),
		tcpostgres.WithUsername("fintech"),
		tcpostgres.WithPassword("fintech"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := sqlx.Connect("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	schema, err := os.ReadFile("../../../../migrations/001_create_tables.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(schema))
	require.NoError(t, err)

	return db
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

func setupHandler(t *testing.T, db *sqlx.DB) (*sagahandler.SagaHandler, *application.WalletService) {
	t.Helper()

	eventStore := pgadapter.NewEventStore(db)
	snapStore := pgadapter.NewSnapshotStore(db)
	outboxRepo := pgadapter.NewWalletOutboxRepository(db)
	processedRepo := pgadapter.NewProcessedEventRepository(db)

	walletSvc := application.NewWalletService(eventStore, snapStore)
	logger := zap.NewNop()

	handler := sagahandler.NewSagaHandler(walletSvc, processedRepo, outboxRepo, logger)
	return handler, walletSvc
}

func TestSagaHandler_TransactionInitiated_ReservesFunds(t *testing.T) {
	db := setupDB(t)
	handler, walletSvc := setupHandler(t, db)
	ctx := context.Background()

	userID := uuid.New()
	result, err := walletSvc.CreateWallet(ctx, userID, "BRL")
	require.NoError(t, err)

	_, err = walletSvc.Deposit(ctx, result.WalletID, 10000, "initial")
	require.NoError(t, err)

	txID := uuid.New()
	eventID := uuid.New()

	record := makeRecord("TransactionInitiated", eventID.String(), map[string]any{
		"transaction_id":        txID.String(),
		"source_wallet_id":      result.WalletID.String(),
		"destination_wallet_id": uuid.New().String(),
		"amount_cents":          int64(3000),
	})

	err = handler.Handle(ctx, record)
	require.NoError(t, err)

	balance, err := walletSvc.GetBalance(ctx, result.WalletID)
	require.NoError(t, err)
	assert.Equal(t, int64(10000), balance.BalanceCents)
	assert.Equal(t, int64(7000), balance.BalanceCents-3000)

	outbox := pgadapter.NewWalletOutboxRepository(db)
	events, err := outbox.FetchUnpublished(ctx, 10)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, domain.OutboxFundsReserved, events[0].EventType)
}

func TestSagaHandler_TransactionCompleted_CreditsDestination(t *testing.T) {
	db := setupDB(t)
	handler, walletSvc := setupHandler(t, db)
	ctx := context.Background()

	destResult, err := walletSvc.CreateWallet(ctx, uuid.New(), "BRL")
	require.NoError(t, err)

	txID := uuid.New()
	eventID := uuid.New()

	record := makeRecord("TransactionCompleted", eventID.String(), map[string]any{
		"transaction_id":        txID.String(),
		"source_wallet_id":      uuid.New().String(),
		"destination_wallet_id": destResult.WalletID.String(),
		"amount_cents":          int64(5000),
	})

	err = handler.Handle(ctx, record)
	require.NoError(t, err)

	balance, err := walletSvc.GetBalance(ctx, destResult.WalletID)
	require.NoError(t, err)
	assert.Equal(t, int64(5000), balance.BalanceCents)

	outbox := pgadapter.NewWalletOutboxRepository(db)
	events, err := outbox.FetchUnpublished(ctx, 10)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, domain.OutboxFundsDeposited, events[0].EventType)
}

func TestSagaHandler_Idempotency(t *testing.T) {
	db := setupDB(t)
	handler, walletSvc := setupHandler(t, db)
	ctx := context.Background()

	destResult, err := walletSvc.CreateWallet(ctx, uuid.New(), "BRL")
	require.NoError(t, err)

	eventID := uuid.New().String()
	record := makeRecord("TransactionCompleted", eventID, map[string]any{
		"transaction_id":        uuid.New().String(),
		"source_wallet_id":      uuid.New().String(),
		"destination_wallet_id": destResult.WalletID.String(),
		"amount_cents":          int64(1000),
	})

	require.NoError(t, handler.Handle(ctx, record))
	require.NoError(t, handler.Handle(ctx, record))

	balance, err := walletSvc.GetBalance(ctx, destResult.WalletID)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), balance.BalanceCents)
}

func TestSagaHandler_InsufficientFunds_ReturnsError(t *testing.T) {
	db := setupDB(t)
	handler, walletSvc := setupHandler(t, db)
	ctx := context.Background()

	result, err := walletSvc.CreateWallet(ctx, uuid.New(), "BRL")
	require.NoError(t, err)

	record := makeRecord("TransactionInitiated", uuid.New().String(), map[string]any{
		"transaction_id":        uuid.New().String(),
		"source_wallet_id":      result.WalletID.String(),
		"destination_wallet_id": uuid.New().String(),
		"amount_cents":          int64(99999),
	})

	err = handler.Handle(ctx, record)
	require.Error(t, err)
}
