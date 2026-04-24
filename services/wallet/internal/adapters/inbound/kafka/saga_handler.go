package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"

	"github.com/DeSouzaRafael/go-fintech-microservices/services/wallet/internal/application"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/wallet/internal/domain"
)

const topicWalletEvents = "wallet-events"

type transactionInitiatedPayload struct {
	TransactionID       uuid.UUID `json:"transaction_id"`
	SourceWalletID      uuid.UUID `json:"source_wallet_id"`
	DestinationWalletID uuid.UUID `json:"destination_wallet_id"`
	AmountCents         int64     `json:"amount_cents"`
}

type transactionCompletedPayload struct {
	TransactionID       uuid.UUID `json:"transaction_id"`
	SourceWalletID      uuid.UUID `json:"source_wallet_id"`
	DestinationWalletID uuid.UUID `json:"destination_wallet_id"`
	AmountCents         int64     `json:"amount_cents"`
}

type SagaHandler struct {
	walletSvc *application.WalletService
	processed domain.ProcessedEventRepository
	outbox    domain.WalletOutboxRepository
	logger    *zap.Logger
}

func NewSagaHandler(
	walletSvc *application.WalletService,
	processed domain.ProcessedEventRepository,
	outbox domain.WalletOutboxRepository,
	logger *zap.Logger,
) *SagaHandler {
	return &SagaHandler{walletSvc: walletSvc, processed: processed, outbox: outbox, logger: logger}
}

func (h *SagaHandler) Handle(ctx context.Context, record *kgo.Record) error {
	eventType := headerValue(record.Headers, "event_type")
	rawEventID := headerValue(record.Headers, "event_id")

	eventID, err := uuid.Parse(rawEventID)
	if err != nil {
		h.logger.Warn("invalid event_id header", zap.String("raw", rawEventID))
		return nil
	}

	already, err := h.processed.IsProcessed(ctx, eventID)
	if err != nil {
		return err
	}
	if already {
		return nil
	}

	switch eventType {
	case "TransactionInitiated":
		err = h.handleTransactionInitiated(ctx, record.Value)
	case "TransactionCompleted":
		err = h.handleTransactionCompleted(ctx, record.Value)
	default:
		return nil
	}

	if err != nil {
		return err
	}

	return h.processed.MarkProcessed(ctx, eventID)
}

func (h *SagaHandler) handleTransactionInitiated(ctx context.Context, data []byte) error {
	var p transactionInitiatedPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}

	if err := h.walletSvc.ReserveForTransaction(ctx, p.SourceWalletID, p.AmountCents, p.TransactionID); err != nil {
		h.logger.Warn("reserve failed, will emit funds released",
			zap.String("wallet_id", p.SourceWalletID.String()),
			zap.Error(err),
		)
		return err
	}

	payload, _ := json.Marshal(domain.OutboxFundsReservedPayload{
		WalletID:      p.SourceWalletID,
		TransactionID: p.TransactionID,
		AmountCents:   p.AmountCents,
	})

	return h.outbox.Save(ctx, &domain.WalletOutboxEvent{
		ID:        uuid.New(),
		Topic:     topicWalletEvents,
		Key:       p.SourceWalletID.String(),
		Payload:   payload,
		EventType: domain.OutboxFundsReserved,
		CreatedAt: time.Now().UTC(),
	})
}

func (h *SagaHandler) handleTransactionCompleted(ctx context.Context, data []byte) error {
	var p transactionCompletedPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}

	if err := h.walletSvc.CreditForTransaction(ctx, p.DestinationWalletID, p.AmountCents, p.TransactionID); err != nil {
		return err
	}

	payload, _ := json.Marshal(domain.OutboxFundsDepositedPayload{
		WalletID:      p.DestinationWalletID,
		TransactionID: p.TransactionID,
		AmountCents:   p.AmountCents,
	})

	return h.outbox.Save(ctx, &domain.WalletOutboxEvent{
		ID:        uuid.New(),
		Topic:     topicWalletEvents,
		Key:       p.DestinationWalletID.String(),
		Payload:   payload,
		EventType: domain.OutboxFundsDeposited,
		CreatedAt: time.Now().UTC(),
	})
}

func headerValue(headers []kgo.RecordHeader, key string) string {
	for _, h := range headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}
