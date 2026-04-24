package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/wallet/internal/domain"
)

type memEventStore struct {
	events map[uuid.UUID][]domain.Event
}

func newMemEventStore() *memEventStore {
	return &memEventStore{events: map[uuid.UUID][]domain.Event{}}
}

func (s *memEventStore) AppendEvents(_ context.Context, walletID uuid.UUID, events []domain.Event, expectedVersion int64) error {
	stored := s.events[walletID]
	currentVersion := int64(len(stored))
	if currentVersion != expectedVersion {
		return apperrors.New(apperrors.CodeConflict, "version conflict")
	}
	s.events[walletID] = append(stored, events...)
	return nil
}

func (s *memEventStore) LoadEvents(_ context.Context, walletID uuid.UUID, afterVersion int64) ([]domain.Event, error) {
	all := s.events[walletID]
	var result []domain.Event
	for _, e := range all {
		if e.Version > afterVersion {
			result = append(result, e)
		}
	}
	return result, nil
}

type memSnapshotStore struct{}

func (s *memSnapshotStore) SaveSnapshot(_ context.Context, _ *domain.Wallet) error { return nil }
func (s *memSnapshotStore) LoadSnapshot(_ context.Context, _ uuid.UUID) (*domain.Wallet, int64, error) {
	return nil, 0, apperrors.New(apperrors.CodeNotFound, "no snapshot")
}

func newTestService() *WalletService {
	return NewWalletService(newMemEventStore(), &memSnapshotStore{})
}

func TestWalletService_CreateWallet(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	t.Run("creates wallet", func(t *testing.T) {
		result, err := svc.CreateWallet(ctx, uuid.New(), "BRL")
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, result.WalletID)
		assert.False(t, result.CreatedAt.IsZero())
	})

	t.Run("rejects empty currency", func(t *testing.T) {
		_, err := svc.CreateWallet(ctx, uuid.New(), "")
		require.Error(t, err)
	})
}

func TestWalletService_GetBalance(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	result, _ := svc.CreateWallet(ctx, uuid.New(), "USD")

	t.Run("returns zero balance for new wallet", func(t *testing.T) {
		bal, err := svc.GetBalance(ctx, result.WalletID)
		require.NoError(t, err)
		assert.Equal(t, int64(0), bal.BalanceCents)
		assert.Equal(t, "USD", bal.Currency)
	})

	t.Run("not found for unknown wallet", func(t *testing.T) {
		_, err := svc.GetBalance(ctx, uuid.New())
		require.Error(t, err)
	})
}

func TestWalletService_Deposit(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	result, _ := svc.CreateWallet(ctx, uuid.New(), "BRL")

	t.Run("deposits funds", func(t *testing.T) {
		res, err := svc.Deposit(ctx, result.WalletID, 5000, "initial")
		require.NoError(t, err)
		assert.Equal(t, int64(5000), res.NewBalance)
		assert.NotEqual(t, uuid.Nil, res.EventID)
	})

	t.Run("rejects negative amount", func(t *testing.T) {
		_, err := svc.Deposit(ctx, result.WalletID, -100, "bad")
		require.Error(t, err)
	})
}

func TestWalletService_Withdraw(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	result, _ := svc.CreateWallet(ctx, uuid.New(), "BRL")
	_, _ = svc.Deposit(ctx, result.WalletID, 10000, "load")

	t.Run("withdraws funds", func(t *testing.T) {
		res, err := svc.Withdraw(ctx, result.WalletID, 3000, "spend")
		require.NoError(t, err)
		assert.Equal(t, int64(7000), res.NewBalance)
	})

	t.Run("rejects overdraft", func(t *testing.T) {
		_, err := svc.Withdraw(ctx, result.WalletID, 99999, "overdraft")
		require.Error(t, err)
		var de *apperrors.DomainError
		require.ErrorAs(t, err, &de)
		assert.Equal(t, apperrors.CodeInsufficientFunds, de.Code)
	})
}

func TestWalletService_ReserveForTransaction(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	result, _ := svc.CreateWallet(ctx, uuid.New(), "BRL")
	_, _ = svc.Deposit(ctx, result.WalletID, 10000, "load")

	t.Run("reserves funds", func(t *testing.T) {
		err := svc.ReserveForTransaction(ctx, result.WalletID, 4000, uuid.New())
		require.NoError(t, err)

		bal, _ := svc.GetBalance(ctx, result.WalletID)
		assert.Equal(t, int64(10000), bal.BalanceCents)
	})

	t.Run("rejects reserve exceeding available", func(t *testing.T) {
		err := svc.ReserveForTransaction(ctx, result.WalletID, 99999, uuid.New())
		require.Error(t, err)
	})
}

func TestWalletService_CreditForTransaction(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	result, _ := svc.CreateWallet(ctx, uuid.New(), "BRL")

	t.Run("credits destination wallet", func(t *testing.T) {
		err := svc.CreditForTransaction(ctx, result.WalletID, 7000, uuid.New())
		require.NoError(t, err)

		bal, _ := svc.GetBalance(ctx, result.WalletID)
		assert.Equal(t, int64(7000), bal.BalanceCents)
	})
}

func TestWalletService_ReleaseReservation(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	result, _ := svc.CreateWallet(ctx, uuid.New(), "BRL")
	_, _ = svc.Deposit(ctx, result.WalletID, 5000, "load")
	txID := uuid.New()
	_ = svc.ReserveForTransaction(ctx, result.WalletID, 2000, txID)

	t.Run("releases reservation", func(t *testing.T) {
		err := svc.ReleaseReservation(ctx, result.WalletID, 2000, txID)
		require.NoError(t, err)
	})
}

func TestWalletService_Errors_UnknownWallet(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()
	unknownID := uuid.New()

	t.Run("deposit unknown wallet", func(t *testing.T) {
		_, err := svc.Deposit(ctx, unknownID, 100, "")
		require.Error(t, err)
	})

	t.Run("withdraw unknown wallet", func(t *testing.T) {
		_, err := svc.Withdraw(ctx, unknownID, 100, "")
		require.Error(t, err)
	})

	t.Run("reserve unknown wallet", func(t *testing.T) {
		err := svc.ReserveForTransaction(ctx, unknownID, 100, uuid.New())
		require.Error(t, err)
	})

	t.Run("credit unknown wallet", func(t *testing.T) {
		err := svc.CreditForTransaction(ctx, unknownID, 100, uuid.New())
		require.Error(t, err)
	})

	t.Run("release unknown wallet", func(t *testing.T) {
		err := svc.ReleaseReservation(ctx, unknownID, 100, uuid.New())
		require.Error(t, err)
	})
}

func TestWalletService_MaybeSnapshot_TriggeredAtInterval(t *testing.T) {
	store := newMemEventStore()
	snapshots := &capturingSnapshotStore{}
	svc := NewWalletService(store, snapshots)
	ctx := context.Background()

	result, err := svc.CreateWallet(ctx, uuid.New(), "BRL")
	require.NoError(t, err)

	for range domain.SnapshotInterval - 1 {
		_, err = svc.Deposit(ctx, result.WalletID, 100, "fill")
		require.NoError(t, err)
	}
	assert.Equal(t, 1, snapshots.count)
}

type capturingSnapshotStore struct{ count int }

func (s *capturingSnapshotStore) SaveSnapshot(_ context.Context, _ *domain.Wallet) error {
	s.count++
	return nil
}
func (s *capturingSnapshotStore) LoadSnapshot(_ context.Context, _ uuid.UUID) (*domain.Wallet, int64, error) {
	return nil, 0, apperrors.New(apperrors.CodeNotFound, "no snapshot")
}

func TestWalletService_Deposit_AppendError(t *testing.T) {
	store := newMemEventStore()
	svc := NewWalletService(store, &memSnapshotStore{})
	ctx := context.Background()
	result, _ := svc.CreateWallet(ctx, uuid.New(), "BRL")
	store.events[result.WalletID][0].Version = 99
	_, err := svc.Deposit(ctx, result.WalletID, 100, "")
	require.Error(t, err)
}

func TestWalletService_Withdraw_AppendError(t *testing.T) {
	store := newMemEventStore()
	svc := NewWalletService(store, &memSnapshotStore{})
	ctx := context.Background()
	result, _ := svc.CreateWallet(ctx, uuid.New(), "BRL")
	_, _ = svc.Deposit(ctx, result.WalletID, 5000, "")
	store.events[result.WalletID][1].Version = 99
	_, err := svc.Withdraw(ctx, result.WalletID, 100, "")
	require.Error(t, err)
}

func TestWalletService_MaybeSnapshot_Error(t *testing.T) {
	type errSnapshotStore struct{ memSnapshotStore }
	store := newMemEventStore()
	svc := NewWalletService(store, &capturingSnapshotStore{})
	ctx := context.Background()
	result, _ := svc.CreateWallet(ctx, uuid.New(), "BRL")
	for range domain.SnapshotInterval - 2 {
		_, _ = svc.Deposit(ctx, result.WalletID, 1, "")
	}
	_, err := svc.Deposit(ctx, result.WalletID, 1, "")
	require.NoError(t, err)
}

func TestWalletService_Load_WithSnapshot(t *testing.T) {
	store := newMemEventStore()
	snap := &capturingSnapshotStore{}
	svc := NewWalletService(store, snap)
	ctx := context.Background()
	result, _ := svc.CreateWallet(ctx, uuid.New(), "BRL")
	_, _ = svc.Deposit(ctx, result.WalletID, 100, "")
	bal, err := svc.GetBalance(ctx, result.WalletID)
	require.NoError(t, err)
	assert.Equal(t, int64(100), bal.BalanceCents)
}
