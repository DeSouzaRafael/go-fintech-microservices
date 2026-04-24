package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	fraudv1 "github.com/DeSouzaRafael/go-fintech-microservices/api/proto/fraud/v1"
)

type FraudClient struct {
	client fraudv1.FraudServiceClient
}

func NewFraudClient(addr string) (*FraudClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial fraud service: %w", err)
	}
	return &FraudClient{client: fraudv1.NewFraudServiceClient(conn)}, nil
}

func (c *FraudClient) Evaluate(ctx context.Context, txID, userID, walletID uuid.UUID, amountCents int64) (string, error) {
	resp, err := c.client.Evaluate(ctx, &fraudv1.EvaluateRequest{
		TransactionId: txID.String(),
		UserId:        userID.String(),
		WalletId:      walletID.String(),
		AmountCents:   amountCents,
	})
	if err != nil {
		return "", err
	}
	switch resp.Decision {
	case fraudv1.Decision_DECISION_REJECTED:
		return "REJECTED", nil
	case fraudv1.Decision_DECISION_REVIEW:
		return "REVIEW", nil
	case fraudv1.Decision_DECISION_APPROVED, fraudv1.Decision_DECISION_UNSPECIFIED:
		return "APPROVED", nil
	default:
		return "APPROVED", nil
	}
}
