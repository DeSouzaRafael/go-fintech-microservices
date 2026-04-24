package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	pgadapter "github.com/DeSouzaRafael/go-fintech-microservices/services/wallet/internal/adapters/outbound/postgres"
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
		tcpostgres.WithDatabase("wallet_test"),
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

func TestEventStore_AppendAndLoad(t *testing.T) {
	db := setupDB(t)
	store := pgadapter.NewEventStore(db)
	ctx := context.Background()

	wallet, err := domain.NewWallet(uuid.New(), "BRL")
	require.NoError(t, err)
	_ = wallet.Deposit(5000, "initial deposit")
	_ = wallet.Withdraw(1000, "withdrawal")
	events := wallet.Changes()

	t.Run("appends events successfully", func(t *testing.T) {
		err := store.AppendEvents(ctx, wallet.ID, events, 0)
		require.NoError(t, err)
	})

	t.Run("loads events in order", func(t *testing.T) {
		loaded, err := store.LoadEvents(ctx, wallet.ID, 0)
		require.NoError(t, err)
		assert.Len(t, loaded, len(events))
		assert.Equal(t, domain.EventWalletCreated, loaded[0].Type)
		assert.Equal(t, domain.EventFundsDeposited, loaded[1].Type)
		assert.Equal(t, domain.EventFundsWithdrawn, loaded[2].Type)
	})

	t.Run("loads events after version", func(t *testing.T) {
		loaded, err := store.LoadEvents(ctx, wallet.ID, 1)
		require.NoError(t, err)
		assert.Len(t, loaded, 2)
		assert.Equal(t, domain.EventFundsDeposited, loaded[0].Type)
	})

	t.Run("optimistic concurrency conflict", func(t *testing.T) {
		err := store.AppendEvents(ctx, wallet.ID, events[:1], 0)
		require.Error(t, err)
	})

	t.Run("empty result for unknown wallet", func(t *testing.T) {
		loaded, err := store.LoadEvents(ctx, uuid.New(), 0)
		require.NoError(t, err)
		assert.Empty(t, loaded)
	})
}

func TestSnapshotStore_SaveAndLoad(t *testing.T) {
	db := setupDB(t)
	snapStore := pgadapter.NewSnapshotStore(db)
	eventStore := pgadapter.NewEventStore(db)
	ctx := context.Background()

	wallet, _ := domain.NewWallet(uuid.New(), "USD")
	_ = wallet.Deposit(10000, "big deposit")
	require.NoError(t, eventStore.AppendEvents(ctx, wallet.ID, wallet.Changes(), 0))

	t.Run("saves snapshot", func(t *testing.T) {
		err := snapStore.SaveSnapshot(ctx, wallet)
		require.NoError(t, err)
	})

	t.Run("loads snapshot with correct state", func(t *testing.T) {
		loaded, version, err := snapStore.LoadSnapshot(ctx, wallet.ID)
		require.NoError(t, err)
		assert.Equal(t, wallet.ID, loaded.ID)
		assert.Equal(t, wallet.BalanceCents, loaded.BalanceCents)
		assert.Equal(t, wallet.Currency, loaded.Currency)
		assert.Equal(t, wallet.Version, version)
	})

	t.Run("upserts snapshot on second save", func(t *testing.T) {
		_ = wallet.Deposit(500, "extra")
		wallet.ClearChanges()
		require.NoError(t, snapStore.SaveSnapshot(ctx, wallet))

		loaded, version, err := snapStore.LoadSnapshot(ctx, wallet.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(10500), loaded.BalanceCents)
		assert.Equal(t, wallet.Version, version)
	})

	t.Run("returns not found for unknown wallet", func(t *testing.T) {
		_, _, err := snapStore.LoadSnapshot(ctx, uuid.New())
		require.Error(t, err)
	})
}

func TestEventSourcing_ReconstitutFromEvents(t *testing.T) {
	db := setupDB(t)
	store := pgadapter.NewEventStore(db)
	ctx := context.Background()

	original, _ := domain.NewWallet(uuid.New(), "BRL")
	_ = original.Deposit(8000, "dep1")
	_ = original.Deposit(2000, "dep2")
	_ = original.Withdraw(3000, "wd1")

	require.NoError(t, store.AppendEvents(ctx, original.ID, original.Changes(), 0))

	loaded, err := store.LoadEvents(ctx, original.ID, 0)
	require.NoError(t, err)

	rebuilt := &domain.Wallet{}
	rebuilt.Reconstitute(loaded)

	assert.Equal(t, original.ID, rebuilt.ID)
	assert.Equal(t, int64(7000), rebuilt.BalanceCents)
	assert.Equal(t, original.Version, rebuilt.Version)
}
