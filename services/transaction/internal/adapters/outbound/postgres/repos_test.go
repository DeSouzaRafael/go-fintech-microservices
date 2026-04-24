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

	pgadapter "github.com/DeSouzaRafael/go-fintech-microservices/services/transaction/internal/adapters/outbound/postgres"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/transaction/internal/domain"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func setupDB(t *testing.T) *sqlx.DB {
	t.Helper()
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("transaction_test"),
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

	schema, err := os.ReadFile("../../../../migrations/000001_create_tables.up.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(schema))
	require.NoError(t, err)

	return db
}

func TestTransactionRepository(t *testing.T) {
	db := setupDB(t)
	repo := pgadapter.NewTransactionRepository(db)
	ctx := context.Background()

	tx, err := domain.NewTransaction(uuid.New(), uuid.New(), 5000, "transfer", "idem-key-1")
	require.NoError(t, err)

	t.Run("saves and finds by id", func(t *testing.T) {
		require.NoError(t, repo.Save(ctx, tx))

		found, err := repo.FindByID(ctx, tx.ID)
		require.NoError(t, err)
		assert.Equal(t, tx.ID, found.ID)
		assert.Equal(t, domain.StatusPending, found.Status)
		assert.Equal(t, int64(5000), found.AmountCents)
	})

	t.Run("finds by idempotency key", func(t *testing.T) {
		found, err := repo.FindByIdempotencyKey(ctx, "idem-key-1")
		require.NoError(t, err)
		assert.Equal(t, tx.ID, found.ID)
	})

	t.Run("updates status", func(t *testing.T) {
		tx.Complete()
		require.NoError(t, repo.Update(ctx, tx))

		found, err := repo.FindByID(ctx, tx.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusCompleted, found.Status)
	})

	t.Run("returns not found for unknown id", func(t *testing.T) {
		_, err := repo.FindByID(ctx, uuid.New())
		require.Error(t, err)
	})
}

func TestOutboxRepository(t *testing.T) {
	db := setupDB(t)
	repo := pgadapter.NewOutboxRepository(db)
	ctx := context.Background()

	event := &domain.OutboxEvent{
		ID:        uuid.New(),
		Topic:     "transactions",
		Key:       uuid.New().String(),
		Payload:   []byte(`{"transaction_id":"abc"}`),
		EventType: domain.EventTransactionInitiated,
	}

	t.Run("saves event", func(t *testing.T) {
		require.NoError(t, repo.Save(ctx, event))
	})

	t.Run("fetches unpublished", func(t *testing.T) {
		events, err := repo.FetchUnpublished(ctx, 10)
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, event.ID, events[0].ID)
		assert.Nil(t, events[0].PublishedAt)
	})

	t.Run("marks published", func(t *testing.T) {
		require.NoError(t, repo.MarkPublished(ctx, event.ID))

		events, err := repo.FetchUnpublished(ctx, 10)
		require.NoError(t, err)
		assert.Empty(t, events)
	})
}
