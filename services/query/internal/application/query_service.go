package application

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"

	"github.com/DeSouzaRafael/go-fintech-microservices/services/query/internal/domain"
)

type QueryService struct {
	balances  domain.BalanceRepository
	statement domain.StatementRepository
	stats     domain.StatsRepository
	processed domain.ProcessedEventRepository
	logger    *zap.Logger
}

func NewQueryService(
	balances domain.BalanceRepository,
	statement domain.StatementRepository,
	stats domain.StatsRepository,
	processed domain.ProcessedEventRepository,
	logger *zap.Logger,
) *QueryService {
	return &QueryService{
		balances:  balances,
		statement: statement,
		stats:     stats,
		processed: processed,
		logger:    logger,
	}
}

func (s *QueryService) GetBalance(ctx context.Context, walletID uuid.UUID) (*domain.BalanceProjection, error) {
	return s.balances.FindByWalletID(ctx, walletID)
}

func (s *QueryService) GetStatement(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]domain.StatementEntry, error) {
	return s.statement.FindByWalletID(ctx, walletID, limit, offset)
}

func (s *QueryService) GetStats(ctx context.Context, userID uuid.UUID) (*domain.UserStats, error) {
	return s.stats.FindByUserID(ctx, userID)
}

func (s *QueryService) HandleEvent(ctx context.Context, record *kgo.Record) error {
	eventType := headerValue(record.Headers, "event_type")
	rawEventID := headerValue(record.Headers, "event_id")

	eventID, err := uuid.Parse(rawEventID)
	if err != nil {
		return nil
	}

	already, err := s.processed.IsProcessed(ctx, eventID)
	if err != nil {
		return err
	}
	if already {
		return nil
	}

	switch eventType {
	case "WalletCreated":
		err = s.handleWalletCreated(ctx, record.Value)
	case "FundsDeposited":
		err = s.handleFundsDeposited(ctx, record.Value)
	case "FundsWithdrawn":
		err = s.handleFundsWithdrawn(ctx, record.Value)
	case "TransactionCompleted":
		err = s.handleTransactionCompleted(ctx, record.Value)
	default:
		return nil
	}

	if err != nil {
		s.logger.Error("handle query event", zap.String("type", eventType), zap.Error(err))
		return err
	}

	return s.processed.MarkProcessed(ctx, eventID)
}

func (s *QueryService) handleWalletCreated(ctx context.Context, data []byte) error {
	var p struct {
		WalletID uuid.UUID `json:"wallet_id"`
		UserID   uuid.UUID `json:"user_id"`
		Currency string    `json:"currency"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}

	return s.balances.Upsert(ctx, &domain.BalanceProjection{
		WalletID:     p.WalletID,
		UserID:       p.UserID,
		Currency:     p.Currency,
		BalanceCents: 0,
		UpdatedAt:    time.Now().UTC(),
	})
}

func (s *QueryService) handleFundsDeposited(ctx context.Context, data []byte) error {
	var p struct {
		WalletID    uuid.UUID `json:"wallet_id"`
		AmountCents int64     `json:"amount_cents"`
		Description string    `json:"description"`
		Version     int64     `json:"version"`
		EventID     uuid.UUID `json:"event_id"`
		OccurredAt  time.Time `json:"occurred_at"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}

	proj, err := s.balances.FindByWalletID(ctx, p.WalletID)
	if err != nil {
		return err
	}

	proj.BalanceCents += p.AmountCents
	proj.Version = p.Version
	proj.UpdatedAt = time.Now().UTC()

	if err := s.balances.Upsert(ctx, proj); err != nil {
		return err
	}

	return s.statement.Append(ctx, &domain.StatementEntry{
		EventID:     p.EventID,
		WalletID:    p.WalletID,
		Type:        "DEPOSIT",
		AmountCents: p.AmountCents,
		Description: p.Description,
		OccurredAt:  p.OccurredAt,
	})
}

func (s *QueryService) handleFundsWithdrawn(ctx context.Context, data []byte) error {
	var p struct {
		WalletID    uuid.UUID `json:"wallet_id"`
		AmountCents int64     `json:"amount_cents"`
		Description string    `json:"description"`
		Version     int64     `json:"version"`
		EventID     uuid.UUID `json:"event_id"`
		OccurredAt  time.Time `json:"occurred_at"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}

	proj, err := s.balances.FindByWalletID(ctx, p.WalletID)
	if err != nil {
		return err
	}

	proj.BalanceCents -= p.AmountCents
	proj.Version = p.Version
	proj.UpdatedAt = time.Now().UTC()

	if err := s.balances.Upsert(ctx, proj); err != nil {
		return err
	}

	return s.statement.Append(ctx, &domain.StatementEntry{
		EventID:     p.EventID,
		WalletID:    p.WalletID,
		Type:        "WITHDRAWAL",
		AmountCents: p.AmountCents,
		Description: p.Description,
		OccurredAt:  p.OccurredAt,
	})
}

func (s *QueryService) handleTransactionCompleted(ctx context.Context, data []byte) error {
	var p struct {
		UserID      uuid.UUID `json:"user_id"`
		AmountCents int64     `json:"amount_cents"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}

	stats, err := s.stats.FindByUserID(ctx, p.UserID)
	if err != nil {
		stats = &domain.UserStats{UserID: p.UserID}
	}

	stats.TotalTransactions++
	stats.TotalDepositCents += p.AmountCents
	stats.UpdatedAt = time.Now().UTC()

	return s.stats.Upsert(ctx, stats)
}

func headerValue(headers []kgo.RecordHeader, key string) string {
	for _, h := range headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}
