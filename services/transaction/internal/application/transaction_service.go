package application

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/transaction/internal/domain"
)

const topicTransactions = "transactions"

type FraudChecker interface {
	Evaluate(ctx context.Context, txID, userID, walletID uuid.UUID, amountCents int64) (string, error)
}

type TransactionService struct {
	txRepo domain.TransactionRepository
	outbox domain.OutboxRepository
	fraud  FraudChecker
}

func NewTransactionService(txRepo domain.TransactionRepository, outboxRepo domain.OutboxRepository, fraud FraudChecker) *TransactionService {
	return &TransactionService{txRepo: txRepo, outbox: outboxRepo, fraud: fraud}
}

type InitiateResult struct {
	TransactionID uuid.UUID
	Status        domain.Status
	CreatedAt     time.Time
}

func (s *TransactionService) InitiateTransfer(ctx context.Context, sourceWalletID, destWalletID uuid.UUID, amountCents int64, description, idempotencyKey string) (InitiateResult, error) {
	if idempotencyKey != "" {
		existing, err := s.txRepo.FindByIdempotencyKey(ctx, idempotencyKey)
		if err == nil {
			return InitiateResult{
				TransactionID: existing.ID,
				Status:        existing.Status,
				CreatedAt:     existing.CreatedAt,
			}, nil
		}
	}

	tx, err := domain.NewTransaction(sourceWalletID, destWalletID, amountCents, description, idempotencyKey)
	if err != nil {
		return InitiateResult{}, err
	}

	if s.fraud != nil {
		decision, err := s.fraud.Evaluate(ctx, tx.ID, uuid.Nil, sourceWalletID, amountCents)
		if err != nil {
			return InitiateResult{}, errors.Wrap(errors.CodeInternal, "fraud check", err)
		}
		if decision == "REJECTED" {
			return InitiateResult{}, errors.New(errors.CodePermissionDenied, "transaction rejected by fraud check")
		}
	}

	payload, err := json.Marshal(domain.TransactionInitiatedPayload{
		TransactionID:       tx.ID,
		SourceWalletID:      tx.SourceWalletID,
		DestinationWalletID: tx.DestinationWalletID,
		AmountCents:         tx.AmountCents,
		Description:         tx.Description,
	})
	if err != nil {
		return InitiateResult{}, errors.Wrap(errors.CodeInternal, "marshal payload", err)
	}

	outboxEvent := &domain.OutboxEvent{
		ID:        uuid.New(),
		Topic:     topicTransactions,
		Key:       tx.SourceWalletID.String(),
		Payload:   payload,
		EventType: domain.EventTransactionInitiated,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.txRepo.Save(ctx, tx); err != nil {
		return InitiateResult{}, err
	}

	if err := s.outbox.Save(ctx, outboxEvent); err != nil {
		return InitiateResult{}, err
	}

	return InitiateResult{
		TransactionID: tx.ID,
		Status:        tx.Status,
		CreatedAt:     tx.CreatedAt,
	}, nil
}

func (s *TransactionService) CompleteTransaction(ctx context.Context, txID uuid.UUID) error {
	tx, err := s.txRepo.FindByID(ctx, txID)
	if err != nil {
		return err
	}
	if tx.Status != domain.StatusPending {
		return errors.New(errors.CodeConflict, "transaction not in pending state")
	}

	tx.Complete()

	payload, _ := json.Marshal(domain.TransactionCompletedPayload{
		TransactionID:       tx.ID,
		SourceWalletID:      tx.SourceWalletID,
		DestinationWalletID: tx.DestinationWalletID,
		AmountCents:         tx.AmountCents,
	})

	outboxEvent := &domain.OutboxEvent{
		ID:        uuid.New(),
		Topic:     topicTransactions,
		Key:       tx.SourceWalletID.String(),
		Payload:   payload,
		EventType: domain.EventTransactionCompleted,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.txRepo.Update(ctx, tx); err != nil {
		return err
	}

	return s.outbox.Save(ctx, outboxEvent)
}

func (s *TransactionService) FailTransaction(ctx context.Context, txID uuid.UUID, reason string) error {
	tx, err := s.txRepo.FindByID(ctx, txID)
	if err != nil {
		return err
	}
	if tx.Status != domain.StatusPending {
		return errors.New(errors.CodeConflict, "transaction not in pending state")
	}

	tx.Fail(reason)

	payload, _ := json.Marshal(domain.TransactionFailedPayload{
		TransactionID:  tx.ID,
		SourceWalletID: tx.SourceWalletID,
		Reason:         reason,
	})

	outboxEvent := &domain.OutboxEvent{
		ID:        uuid.New(),
		Topic:     topicTransactions,
		Key:       tx.SourceWalletID.String(),
		Payload:   payload,
		EventType: domain.EventTransactionFailed,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.txRepo.Update(ctx, tx); err != nil {
		return err
	}

	return s.outbox.Save(ctx, outboxEvent)
}

func (s *TransactionService) GetTransaction(ctx context.Context, txID uuid.UUID) (*domain.Transaction, error) {
	return s.txRepo.FindByID(ctx, txID)
}
