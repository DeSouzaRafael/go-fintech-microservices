package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/transaction/internal/domain"
)

type memTxRepo struct {
	byID             map[uuid.UUID]*domain.Transaction
	byIdempotencyKey map[string]*domain.Transaction
}

func newMemTxRepo() *memTxRepo {
	return &memTxRepo{
		byID:             map[uuid.UUID]*domain.Transaction{},
		byIdempotencyKey: map[string]*domain.Transaction{},
	}
}

func (r *memTxRepo) Save(_ context.Context, tx *domain.Transaction) error {
	r.byID[tx.ID] = tx
	if tx.IdempotencyKey != "" {
		r.byIdempotencyKey[tx.IdempotencyKey] = tx
	}
	return nil
}

func (r *memTxRepo) Update(_ context.Context, tx *domain.Transaction) error {
	r.byID[tx.ID] = tx
	return nil
}

func (r *memTxRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Transaction, error) {
	tx, ok := r.byID[id]
	if !ok {
		return nil, apperrors.New(apperrors.CodeNotFound, "not found")
	}
	return tx, nil
}

func (r *memTxRepo) FindByIdempotencyKey(_ context.Context, key string) (*domain.Transaction, error) {
	tx, ok := r.byIdempotencyKey[key]
	if !ok {
		return nil, apperrors.New(apperrors.CodeNotFound, "not found")
	}
	return tx, nil
}

type memOutboxRepo struct {
	events []domain.OutboxEvent
}

func (r *memOutboxRepo) Save(_ context.Context, e *domain.OutboxEvent) error {
	r.events = append(r.events, *e)
	return nil
}

func (r *memOutboxRepo) FetchUnpublished(_ context.Context, _ int) ([]domain.OutboxEvent, error) {
	return r.events, nil
}

func (r *memOutboxRepo) MarkPublished(_ context.Context, _ uuid.UUID) error {
	return nil
}

type stubFraud struct{ decision string }

func (f *stubFraud) Evaluate(_ context.Context, _, _, _ uuid.UUID, _ int64) (string, error) {
	return f.decision, nil
}

func newTestService() *TransactionService {
	return NewTransactionService(newMemTxRepo(), &memOutboxRepo{}, &stubFraud{decision: "APPROVED"})
}

func TestTransactionService_InitiateTransfer(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()
	src, dst := uuid.New(), uuid.New()

	t.Run("creates pending transaction and outbox event", func(t *testing.T) {
		result, err := svc.InitiateTransfer(ctx, src, dst, 5000, "payment", "key-1")
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, result.TransactionID)
		assert.Equal(t, domain.StatusPending, result.Status)
	})

	t.Run("idempotency returns same transaction", func(t *testing.T) {
		r1, _ := svc.InitiateTransfer(ctx, src, dst, 1000, "pay", "idem-42")
		r2, err := svc.InitiateTransfer(ctx, src, dst, 1000, "pay", "idem-42")
		require.NoError(t, err)
		assert.Equal(t, r1.TransactionID, r2.TransactionID)
	})

	t.Run("rejects invalid amount", func(t *testing.T) {
		_, err := svc.InitiateTransfer(ctx, src, dst, 0, "", "")
		require.Error(t, err)
	})

	t.Run("fraud rejection blocks transfer", func(t *testing.T) {
		svcBlocked := NewTransactionService(newMemTxRepo(), &memOutboxRepo{}, &stubFraud{decision: "REJECTED"})
		_, err := svcBlocked.InitiateTransfer(ctx, src, dst, 5000, "blocked", "key-fraud")
		require.Error(t, err)
		var de *apperrors.DomainError
		require.ErrorAs(t, err, &de)
		assert.Equal(t, apperrors.CodePermissionDenied, de.Code)
	})
}

func TestTransactionService_CompleteTransaction(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	result, _ := svc.InitiateTransfer(ctx, uuid.New(), uuid.New(), 1000, "", "")

	t.Run("completes pending transaction", func(t *testing.T) {
		err := svc.CompleteTransaction(ctx, result.TransactionID)
		require.NoError(t, err)

		tx, _ := svc.GetTransaction(ctx, result.TransactionID)
		assert.Equal(t, domain.StatusCompleted, tx.Status)
	})

	t.Run("rejects double completion", func(t *testing.T) {
		err := svc.CompleteTransaction(ctx, result.TransactionID)
		require.Error(t, err)
	})
}

func TestTransactionService_FailTransaction(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	result, _ := svc.InitiateTransfer(ctx, uuid.New(), uuid.New(), 2000, "", "")

	t.Run("fails with reason", func(t *testing.T) {
		err := svc.FailTransaction(ctx, result.TransactionID, "fraud detected")
		require.NoError(t, err)

		tx, _ := svc.GetTransaction(ctx, result.TransactionID)
		assert.Equal(t, domain.StatusFailed, tx.Status)
		assert.Equal(t, "fraud detected", tx.FailureReason)
	})
}

func TestTransactionService_GetTransaction(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	t.Run("returns not found for unknown id", func(t *testing.T) {
		_, err := svc.GetTransaction(ctx, uuid.New())
		require.Error(t, err)
		var de *apperrors.DomainError
		require.ErrorAs(t, err, &de)
		assert.Equal(t, apperrors.CodeNotFound, de.Code)
	})
}
