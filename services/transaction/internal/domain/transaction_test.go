package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransaction(t *testing.T) {
	src := uuid.New()
	dst := uuid.New()

	t.Run("creates pending transaction", func(t *testing.T) {
		tx, err := NewTransaction(src, dst, 1000, "test", "key-1")
		require.NoError(t, err)
		assert.Equal(t, StatusPending, tx.Status)
		assert.Equal(t, int64(1000), tx.AmountCents)
		assert.Equal(t, src, tx.SourceWalletID)
		assert.Equal(t, dst, tx.DestinationWalletID)
	})

	t.Run("rejects zero amount", func(t *testing.T) {
		_, err := NewTransaction(src, dst, 0, "test", "")
		require.Error(t, err)
	})

	t.Run("rejects same source and destination", func(t *testing.T) {
		_, err := NewTransaction(src, src, 100, "test", "")
		require.Error(t, err)
	})
}

func TestTransaction_StateTransitions(t *testing.T) {
	src, dst := uuid.New(), uuid.New()

	t.Run("complete", func(t *testing.T) {
		tx, _ := NewTransaction(src, dst, 500, "", "")
		tx.Complete()
		assert.Equal(t, StatusCompleted, tx.Status)
	})

	t.Run("fail with reason", func(t *testing.T) {
		tx, _ := NewTransaction(src, dst, 500, "", "")
		tx.Fail("fraud detected")
		assert.Equal(t, StatusFailed, tx.Status)
		assert.Equal(t, "fraud detected", tx.FailureReason)
	})

	t.Run("compensate", func(t *testing.T) {
		tx, _ := NewTransaction(src, dst, 500, "", "")
		tx.Compensate()
		assert.Equal(t, StatusCompensated, tx.Status)
	})
}
