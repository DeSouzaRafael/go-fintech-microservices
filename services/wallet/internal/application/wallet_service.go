package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/wallet/internal/domain"
)

type WalletService struct {
	events    domain.EventStore
	snapshots domain.SnapshotStore
}

func NewWalletService(events domain.EventStore, snapshots domain.SnapshotStore) *WalletService {
	return &WalletService{events: events, snapshots: snapshots}
}

type CreateWalletResult struct {
	WalletID  uuid.UUID
	CreatedAt time.Time
}

func (s *WalletService) CreateWallet(ctx context.Context, userID uuid.UUID, currency string) (CreateWalletResult, error) {
	wallet, err := domain.NewWallet(userID, currency)
	if err != nil {
		return CreateWalletResult{}, err
	}

	if err := s.events.AppendEvents(ctx, wallet.ID, wallet.Changes(), 0); err != nil {
		return CreateWalletResult{}, err
	}

	return CreateWalletResult{WalletID: wallet.ID, CreatedAt: time.Now().UTC()}, nil
}

type BalanceResult struct {
	WalletID     uuid.UUID
	BalanceCents int64
	Currency     string
	Version      int64
}

func (s *WalletService) GetBalance(ctx context.Context, walletID uuid.UUID) (BalanceResult, error) {
	wallet, err := s.load(ctx, walletID)
	if err != nil {
		return BalanceResult{}, err
	}

	return BalanceResult{
		WalletID:     wallet.ID,
		BalanceCents: wallet.BalanceCents,
		Currency:     wallet.Currency,
		Version:      wallet.Version,
	}, nil
}

type DepositResult struct {
	EventID      uuid.UUID
	NewBalance   int64
	Version      int64
}

func (s *WalletService) Deposit(ctx context.Context, walletID uuid.UUID, amountCents int64, description string) (DepositResult, error) {
	wallet, err := s.load(ctx, walletID)
	if err != nil {
		return DepositResult{}, err
	}

	prevVersion := wallet.Version
	if err := wallet.Deposit(amountCents, description); err != nil {
		return DepositResult{}, err
	}

	changes := wallet.Changes()
	if err := s.events.AppendEvents(ctx, walletID, changes, prevVersion); err != nil {
		return DepositResult{}, err
	}

	if err := s.maybeSnapshot(ctx, wallet); err != nil {
		return DepositResult{}, err
	}

	return DepositResult{
		EventID:    changes[0].ID,
		NewBalance: wallet.BalanceCents,
		Version:    wallet.Version,
	}, nil
}

type WithdrawResult struct {
	EventID    uuid.UUID
	NewBalance int64
	Version    int64
}

func (s *WalletService) Withdraw(ctx context.Context, walletID uuid.UUID, amountCents int64, description string) (WithdrawResult, error) {
	wallet, err := s.load(ctx, walletID)
	if err != nil {
		return WithdrawResult{}, err
	}

	prevVersion := wallet.Version
	if err := wallet.Withdraw(amountCents, description); err != nil {
		return WithdrawResult{}, err
	}

	changes := wallet.Changes()
	if err := s.events.AppendEvents(ctx, walletID, changes, prevVersion); err != nil {
		return WithdrawResult{}, err
	}

	if err := s.maybeSnapshot(ctx, wallet); err != nil {
		return WithdrawResult{}, err
	}

	return WithdrawResult{
		EventID:    changes[0].ID,
		NewBalance: wallet.BalanceCents,
		Version:    wallet.Version,
	}, nil
}

func (s *WalletService) load(ctx context.Context, walletID uuid.UUID) (*domain.Wallet, error) {
	wallet := &domain.Wallet{}
	snapshotVersion := int64(0)

	snapshot, version, err := s.snapshots.LoadSnapshot(ctx, walletID)
	if err == nil && snapshot != nil {
		wallet = snapshot
		snapshotVersion = version
	}

	events, err := s.events.LoadEvents(ctx, walletID, snapshotVersion)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 && snapshotVersion == 0 {
		return nil, errors.New(errors.CodeNotFound, "wallet not found")
	}

	wallet.Reconstitute(events)
	return wallet, nil
}

func (s *WalletService) ReserveForTransaction(ctx context.Context, walletID uuid.UUID, amountCents int64, transactionID uuid.UUID) error {
	wallet, err := s.load(ctx, walletID)
	if err != nil {
		return err
	}

	prevVersion := wallet.Version
	if err := wallet.Reserve(amountCents, transactionID); err != nil {
		return err
	}

	if err := s.events.AppendEvents(ctx, walletID, wallet.Changes(), prevVersion); err != nil {
		return err
	}

	return s.maybeSnapshot(ctx, wallet)
}

func (s *WalletService) CreditForTransaction(ctx context.Context, walletID uuid.UUID, amountCents int64, transactionID uuid.UUID) error {
	wallet, err := s.load(ctx, walletID)
	if err != nil {
		return err
	}

	prevVersion := wallet.Version
	if err := wallet.Deposit(amountCents, "saga credit"); err != nil {
		return err
	}
	_ = transactionID

	if err := s.events.AppendEvents(ctx, walletID, wallet.Changes(), prevVersion); err != nil {
		return err
	}

	return s.maybeSnapshot(ctx, wallet)
}

func (s *WalletService) ReleaseReservation(ctx context.Context, walletID uuid.UUID, amountCents int64, transactionID uuid.UUID) error {
	wallet, err := s.load(ctx, walletID)
	if err != nil {
		return err
	}

	prevVersion := wallet.Version
	wallet.Release(amountCents, transactionID)

	if err := s.events.AppendEvents(ctx, walletID, wallet.Changes(), prevVersion); err != nil {
		return err
	}

	return s.maybeSnapshot(ctx, wallet)
}

func (s *WalletService) maybeSnapshot(ctx context.Context, w *domain.Wallet) error {
	if w.Version%domain.SnapshotInterval == 0 {
		return s.snapshots.SaveSnapshot(ctx, w)
	}
	return nil
}
