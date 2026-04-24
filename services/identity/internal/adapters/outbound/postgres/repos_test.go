package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	pgadapter "github.com/DeSouzaRafael/go-fintech-microservices/services/identity/internal/adapters/outbound/postgres"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/identity/internal/domain"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func setupDB(t *testing.T) *sqlx.DB {
	t.Helper()
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("identity_test"),
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

func TestUserRepository(t *testing.T) {
	db := setupDB(t)
	repo := pgadapter.NewUserRepository(db)
	ctx := context.Background()

	user := domain.NewUser("test@example.com", "hashed_password", "Test User")

	t.Run("saves and finds by email", func(t *testing.T) {
		err := repo.Save(ctx, &user)
		require.NoError(t, err)

		found, err := repo.FindByEmail(ctx, "test@example.com")
		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
		assert.Equal(t, user.Email, found.Email)
		assert.Equal(t, user.FullName, found.FullName)
	})

	t.Run("finds by id", func(t *testing.T) {
		found, err := repo.FindByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.Email, found.Email)
	})

	t.Run("returns not found for unknown email", func(t *testing.T) {
		_, err := repo.FindByEmail(ctx, "ghost@example.com")
		require.Error(t, err)
	})

	t.Run("returns not found for unknown id", func(t *testing.T) {
		_, err := repo.FindByID(ctx, uuid.New())
		require.Error(t, err)
	})
}

func TestTokenRepository(t *testing.T) {
	db := setupDB(t)
	userRepo := pgadapter.NewUserRepository(db)
	tokenRepo := pgadapter.NewTokenRepository(db)
	ctx := context.Background()

	user := domain.NewUser("token@example.com", "hash", "Token User")
	require.NoError(t, userRepo.Save(ctx, &user))

	token := domain.NewRefreshToken(user.ID, "secure-random-token-value", 24*time.Hour)

	t.Run("saves and finds token", func(t *testing.T) {
		err := tokenRepo.Save(ctx, &token)
		require.NoError(t, err)

		found, err := tokenRepo.FindByToken(ctx, token.Token)
		require.NoError(t, err)
		assert.Equal(t, token.ID, found.ID)
		assert.Equal(t, token.UserID, found.UserID)
		assert.False(t, found.IsExpired())
	})

	t.Run("deletes token", func(t *testing.T) {
		err := tokenRepo.Delete(ctx, token.Token)
		require.NoError(t, err)

		_, err = tokenRepo.FindByToken(ctx, token.Token)
		require.Error(t, err)
	})

	t.Run("returns not found for unknown token", func(t *testing.T) {
		_, err := tokenRepo.FindByToken(ctx, "does-not-exist")
		require.Error(t, err)
	})
}
