package router

import (
	"context"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	identityv1 "github.com/DeSouzaRafael/go-fintech-microservices/api/proto/identity/v1"
	queryv1 "github.com/DeSouzaRafael/go-fintech-microservices/api/proto/query/v1"
	transactionv1 "github.com/DeSouzaRafael/go-fintech-microservices/api/proto/transaction/v1"
	walletv1 "github.com/DeSouzaRafael/go-fintech-microservices/api/proto/wallet/v1"
)

type Config struct {
	IdentityAddr    string
	WalletAddr      string
	TransactionAddr string
	QueryAddr       string
}

func New(ctx context.Context, cfg Config) (http.Handler, error) {
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	mux := runtime.NewServeMux()

	if err := identityv1.RegisterIdentityServiceHandlerFromEndpoint(ctx, mux, cfg.IdentityAddr, opts); err != nil {
		return nil, err
	}
	if err := walletv1.RegisterWalletServiceHandlerFromEndpoint(ctx, mux, cfg.WalletAddr, opts); err != nil {
		return nil, err
	}
	if err := transactionv1.RegisterTransactionServiceHandlerFromEndpoint(ctx, mux, cfg.TransactionAddr, opts); err != nil {
		return nil, err
	}
	if err := queryv1.RegisterQueryServiceHandlerFromEndpoint(ctx, mux, cfg.QueryAddr, opts); err != nil {
		return nil, err
	}

	return mux, nil
}
