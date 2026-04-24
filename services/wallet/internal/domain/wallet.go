package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
)

const SnapshotInterval = 50

type Wallet struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Currency     string
	BalanceCents int64
	Reserved     int64
	Version      int64
	changes      []Event
}

func NewWallet(userID uuid.UUID, currency string) (*Wallet, error) {
	if currency == "" {
		return nil, errors.New(errors.CodeInvalidArgument, "currency required")
	}

	w := &Wallet{}
	e := Event{
		ID:        uuid.New(),
		WalletID:  uuid.New(),
		Type:      EventWalletCreated,
		Version:   1,
		OccuredAt: time.Now().UTC(),
	}
	e.Payload, _ = json.Marshal(WalletCreatedPayload{UserID: userID, Currency: currency})
	w.apply(&e)
	return w, nil
}

func (w *Wallet) Deposit(amountCents int64, description string) error {
	if amountCents <= 0 {
		return errors.New(errors.CodeInvalidArgument, "deposit amount must be positive")
	}

	e := Event{
		ID:        uuid.New(),
		WalletID:  w.ID,
		Type:      EventFundsDeposited,
		Version:   w.Version + 1,
		OccuredAt: time.Now().UTC(),
	}
	e.Payload, _ = json.Marshal(FundsDepositedPayload{AmountCents: amountCents, Description: description})
	w.apply(&e)
	return nil
}

func (w *Wallet) Withdraw(amountCents int64, description string) error {
	if amountCents <= 0 {
		return errors.New(errors.CodeInvalidArgument, "withdrawal amount must be positive")
	}
	if w.AvailableBalance() < amountCents {
		return errors.New(errors.CodeInsufficientFunds, "insufficient funds")
	}

	e := Event{
		ID:        uuid.New(),
		WalletID:  w.ID,
		Type:      EventFundsWithdrawn,
		Version:   w.Version + 1,
		OccuredAt: time.Now().UTC(),
	}
	e.Payload, _ = json.Marshal(FundsWithdrawnPayload{AmountCents: amountCents, Description: description})
	w.apply(&e)
	return nil
}

func (w *Wallet) Reserve(amountCents int64, transactionID uuid.UUID) error {
	if w.AvailableBalance() < amountCents {
		return errors.New(errors.CodeInsufficientFunds, "insufficient funds to reserve")
	}

	e := Event{
		ID:        uuid.New(),
		WalletID:  w.ID,
		Type:      EventFundsReserved,
		Version:   w.Version + 1,
		OccuredAt: time.Now().UTC(),
	}
	e.Payload, _ = json.Marshal(FundsReservedPayload{AmountCents: amountCents, TransactionID: transactionID})
	w.apply(&e)
	return nil
}

func (w *Wallet) Release(amountCents int64, transactionID uuid.UUID) {
	e := Event{
		ID:        uuid.New(),
		WalletID:  w.ID,
		Type:      EventFundsReleased,
		Version:   w.Version + 1,
		OccuredAt: time.Now().UTC(),
	}
	e.Payload, _ = json.Marshal(FundsReleasedPayload{AmountCents: amountCents, TransactionID: transactionID})
	w.apply(&e)
}

func (w *Wallet) AvailableBalance() int64 {
	return w.BalanceCents - w.Reserved
}

func (w *Wallet) Changes() []Event {
	return w.changes
}

func (w *Wallet) ClearChanges() {
	w.changes = nil
}

func (w *Wallet) Reconstitute(events []Event) {
	for i := range events {
		w.applyEvent(&events[i])
		w.Version = events[i].Version
	}
}

func (w *Wallet) apply(e *Event) {
	w.applyEvent(e)
	w.Version = e.Version
	w.changes = append(w.changes, *e)
}

func (w *Wallet) applyEvent(e *Event) {
	switch e.Type {
	case EventWalletCreated:
		var p WalletCreatedPayload
		_ = json.Unmarshal(e.Payload, &p)
		w.ID = e.WalletID
		w.UserID = p.UserID
		w.Currency = p.Currency

	case EventFundsDeposited:
		var p FundsDepositedPayload
		_ = json.Unmarshal(e.Payload, &p)
		w.BalanceCents += p.AmountCents

	case EventFundsWithdrawn:
		var p FundsWithdrawnPayload
		_ = json.Unmarshal(e.Payload, &p)
		w.BalanceCents -= p.AmountCents

	case EventFundsReserved:
		var p FundsReservedPayload
		_ = json.Unmarshal(e.Payload, &p)
		w.Reserved += p.AmountCents

	case EventFundsReleased:
		var p FundsReleasedPayload
		_ = json.Unmarshal(e.Payload, &p)
		w.Reserved -= p.AmountCents
	}
}
