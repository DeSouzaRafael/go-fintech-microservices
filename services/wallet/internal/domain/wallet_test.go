package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWallet(t *testing.T) {
	userID := uuid.New()

	t.Run("creates wallet with correct state", func(t *testing.T) {
		w, err := NewWallet(userID, "BRL")
		require.NoError(t, err)
		assert.Equal(t, userID, w.UserID)
		assert.Equal(t, "BRL", w.Currency)
		assert.Equal(t, int64(0), w.BalanceCents)
		assert.Equal(t, int64(1), w.Version)
		assert.Len(t, w.Changes(), 1)
		assert.Equal(t, EventWalletCreated, w.Changes()[0].Type)
	})

	t.Run("rejects empty currency", func(t *testing.T) {
		_, err := NewWallet(userID, "")
		require.Error(t, err)
	})
}

func TestWallet_Deposit(t *testing.T) {
	w, _ := NewWallet(uuid.New(), "BRL")
	w.ClearChanges()

	t.Run("increases balance", func(t *testing.T) {
		err := w.Deposit(1000, "test deposit")
		require.NoError(t, err)
		assert.Equal(t, int64(1000), w.BalanceCents)
		assert.Equal(t, int64(1000), w.AvailableBalance())
		assert.Len(t, w.Changes(), 1)
		assert.Equal(t, EventFundsDeposited, w.Changes()[0].Type)
	})

	t.Run("rejects zero amount", func(t *testing.T) {
		err := w.Deposit(0, "zero")
		require.Error(t, err)
	})

	t.Run("rejects negative amount", func(t *testing.T) {
		err := w.Deposit(-500, "negative")
		require.Error(t, err)
	})
}

func TestWallet_Withdraw(t *testing.T) {
	w, _ := NewWallet(uuid.New(), "BRL")
	_ = w.Deposit(5000, "initial")
	w.ClearChanges()

	t.Run("decreases balance", func(t *testing.T) {
		err := w.Withdraw(2000, "withdrawal")
		require.NoError(t, err)
		assert.Equal(t, int64(3000), w.BalanceCents)
		assert.Len(t, w.Changes(), 1)
		assert.Equal(t, EventFundsWithdrawn, w.Changes()[0].Type)
	})

	t.Run("rejects insufficient funds", func(t *testing.T) {
		err := w.Withdraw(999999, "too much")
		require.Error(t, err)
	})

	t.Run("rejects zero amount", func(t *testing.T) {
		err := w.Withdraw(0, "zero")
		require.Error(t, err)
	})
}

func TestWallet_Reserve(t *testing.T) {
	w, _ := NewWallet(uuid.New(), "BRL")
	_ = w.Deposit(10000, "initial")
	w.ClearChanges()

	txID := uuid.New()

	t.Run("reserves funds reducing available balance", func(t *testing.T) {
		err := w.Reserve(3000, txID)
		require.NoError(t, err)
		assert.Equal(t, int64(10000), w.BalanceCents)
		assert.Equal(t, int64(3000), w.Reserved)
		assert.Equal(t, int64(7000), w.AvailableBalance())
	})

	t.Run("rejects reserve exceeding available balance", func(t *testing.T) {
		err := w.Reserve(8000, uuid.New())
		require.Error(t, err)
	})
}

func TestWallet_Release(t *testing.T) {
	w, _ := NewWallet(uuid.New(), "BRL")
	_ = w.Deposit(10000, "initial")
	txID := uuid.New()
	_ = w.Reserve(3000, txID)
	w.ClearChanges()

	t.Run("releases reserved funds", func(t *testing.T) {
		w.Release(3000, txID)
		assert.Equal(t, int64(0), w.Reserved)
		assert.Equal(t, int64(10000), w.AvailableBalance())
		assert.Len(t, w.Changes(), 1)
		assert.Equal(t, EventFundsReleased, w.Changes()[0].Type)
	})
}

func TestWallet_Reconstitute(t *testing.T) {
	t.Run("rebuilds state from events", func(t *testing.T) {
		original, _ := NewWallet(uuid.New(), "USD")
		_ = original.Deposit(5000, "dep1")
		_ = original.Deposit(3000, "dep2")
		_ = original.Withdraw(1000, "wd1")
		allEvents := original.Changes()

		rebuilt := &Wallet{}
		rebuilt.Reconstitute(allEvents)

		assert.Equal(t, original.ID, rebuilt.ID)
		assert.Equal(t, original.UserID, rebuilt.UserID)
		assert.Equal(t, original.Currency, rebuilt.Currency)
		assert.Equal(t, int64(7000), rebuilt.BalanceCents)
		assert.Equal(t, original.Version, rebuilt.Version)
	})
}

func TestWallet_SnapshotInterval(t *testing.T) {
	assert.Equal(t, 50, SnapshotInterval)
}
