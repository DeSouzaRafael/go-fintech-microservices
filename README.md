# Fintech Microservices Platform

[![Go Report Card](https://goreportcard.com/badge/github.com/DeSouzaRafael/go-fintech-microservices)](https://goreportcard.com/report/github.com/DeSouzaRafael/go-fintech-microservices)
[![CI](https://github.com/DeSouzaRafael/go-fintech-microservices/actions/workflows/ci.yml/badge.svg)](https://github.com/DeSouzaRafael/go-fintech-microservices/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/DeSouzaRafael/go-fintech-microservices/branch/main/graph/badge.svg)](https://codecov.io/gh/DeSouzaRafael/go-fintech-microservices)
[![License](https://img.shields.io/github/license/DeSouzaRafael/go-fintech-microservices.svg)](https://github.com/DeSouzaRafael/go-fintech-microservices/blob/main/LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25+-00ADD8?logo=go)](https://go.dev/)

High-scale digital wallet platform built with Go microservices, demonstrating advanced distributed systems patterns: CQRS, Event Sourcing, Saga, Transactional Outbox, and full observability.

## Architecture

```
                         ┌─────────────────┐
                         │   API Gateway   │
                         │  REST → gRPC    │
                         └────────┬────────┘
                                  │ gRPC
          ┌──────────┬────────────┼─────────────┬──────────┐
          │          │            │             │          │
   ┌──────▼──┐ ┌─────▼──────┐ ┌───▼────┐ ┌──────▼──┐ ┌─────▼──────┐
   │Identity │ │  Wallet    │ │  Txn   │ │  Fraud  │ │   Query    │
   │Service  │ │ (Event     │ │Service │ │Detection│ │  Service   │
   │         │ │  Sourced)  │ │ (Saga) │ │         │ │(Read Model)│
   └──────┬──┘ └─────┬──────┘ └───┬────┘ └──────┬──┘ └─────┬──────┘
          │          │            │             │          │
          └──────────┴────────────┴─────────────┴──────────┘
                                  │
                            ┌─────▼─────┐
                            │   Kafka   │
                            └─────┬─────┘
                                  │
                         ┌────────▼────────┐
                         │  Notification   │
                         │    Service      │
                         └─────────────────┘
```

Each service owns its database (PostgreSQL). Redis for caching and rate limiting.

## Tech Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.25+ |
| RPC | gRPC + Protocol Buffers |
| REST exposure | grpc-gateway |
| Messaging | Apache Kafka (Redpanda in dev) |
| Databases | PostgreSQL 16 (per service) |
| Cache | Redis 7 |
| Observability | OpenTelemetry → Jaeger + Prometheus + Grafana |
| Testing | testify + testcontainers-go + k6 |
| CI/CD | GitHub Actions |

## Services

| Service | Responsibility | Pattern |
|---------|----------------|---------|
| `identity` | Auth, JWT, KYC | Standard |
| `wallet` | Balances, accounts | **Event Sourcing** |
| `transaction` | Transfer orchestration | **Saga (choreography)** |
| `fraud` | Risk evaluation | Rules engine |
| `notification` | Async notifications | Idempotent consumer |
| `query` | Read projections | **CQRS Read Model** |
| `gateway` | Ingress, JWT auth, rate limit | Reverse proxy |

## Implementation Roadmap

### 1 — Foundation
- [x] Monorepo structure with all service skeletons
- [x] Shared `.proto` contracts (`wallet.proto`, `transaction.proto`, `identity.proto`)
- [x] `pkg/logger` — structured JSON logging (zap)
- [x] `pkg/tracing` — OpenTelemetry SDK setup
- [x] `pkg/errors` — standardized error types
- [x] `pkg/middleware` — gRPC interceptors (auth, tracing, logging, recovery)
- [x] Docker Compose: PostgreSQL, Redis, Redpanda, Jaeger, Prometheus, Grafana
- [x] Service skeleton with gRPC + OTel + graceful shutdown
- [x] GitHub Actions: lint + build pipeline

### 2 — Identity & Wallet
- [x] Identity Service: signup, login, JWT issuance
- [x] Identity Service: refresh tokens, logout, JWT validation endpoint
- [x] Wallet Service: event store schema in PostgreSQL
- [x] Wallet Service: `Wallet` aggregate with domain events
- [x] Wallet Service: `CreateWallet`, `Deposit`, `Withdraw` commands
- [x] Wallet Service: state reconstruction via event replay
- [x] Wallet Service: snapshots every N events for performance
- [x] Unit tests for wallet domain (pure Go, no I/O)
- [x] Integration tests with testcontainers-go

### 3 — Transactions & Saga
- [x] Transaction Service: `TransactionInitiated` → Kafka
- [x] Wallet Service: consume `TransactionInitiated`, reserve funds → `FundsReserved`
- [x] Transaction Service: consume `FundsReserved`, trigger credit
- [x] Wallet Service: credit destination → `FundsDeposited`
- [x] Transaction Service: `TransactionCompleted`
- [x] Compensation flow: `FundsReleased` on failure
- [x] Transactional Outbox in Wallet and Transaction services
- [x] Outbox worker: poll → publish → mark sent
- [x] Idempotency via `event_id` deduplication table
- [x] Integration tests: happy path + simulated failures

### 4 — Fraud & Notifications
- [ ] Fraud Service: daily limit rule
- [ ] Fraud Service: velocity check (N txns in T seconds)
- [ ] Fraud Service: user risk profile cache in Redis
- [ ] Fraud Service: async profile update via Kafka consumer
- [ ] Transaction Service: call Fraud before committing
- [ ] Notification Service: consume `TransactionCompleted`
- [ ] Notification Service: consume `TransactionFailed`
- [ ] Notification Service: consume `FraudDetected`
- [ ] Notification Service: idempotency via processed events table
- [ ] API Gateway: rate limiting per user (Redis token bucket)

### 5 — CQRS & API Gateway
- [ ] Query Service: consume wallet events, build balance projection
- [ ] Query Service: consume transaction events, build statement projection
- [ ] Query Service: paginated statement endpoint (gRPC + REST)
- [ ] Query Service: per-user statistics projection
- [ ] API Gateway: full grpc-gateway setup
- [ ] API Gateway: JWT validation middleware
- [ ] API Gateway: route all services
- [ ] OpenAPI spec generated from proto annotations

### 6 — Observability, Resilience & Load
- [ ] Grafana dashboards: RED metrics per service
- [ ] Grafana dashboards: p50/p95/p99 latency panels
- [ ] Grafana dashboards: Kafka consumer lag
- [ ] Circuit breakers (`sony/gobreaker`) on inter-service calls
- [ ] Retry with exponential backoff on Kafka publish
- [ ] k6 load test: 1,000 TPS baseline
- [ ] k6 load test: 10,000 TPS stress
- [ ] Architecture Decision Records (ADRs)
- [ ] Distributed trace screenshots (Jaeger)
- [ ] Load test results in `docs/`

## Project Structure

```
fintech-platform/
├── api/proto/
├── pkg/
│   ├── logger/
│   ├── tracing/
│   ├── errors/
│   ├── middleware/
│   └── server/
├── services/
│   ├── identity/
│   ├── wallet/
│   ├── transaction/
│   ├── fraud/
│   ├── notification/
│   ├── query/
│   └── gateway/
├── deploy/
│   ├── docker-compose.yml
│   └── grafana/
├── scripts/
├── tests/
│   ├── integration/
│   └── load/
└── docs/
    ├── architecture.md
    └── adr/
```

Each service: hexagonal architecture — `cmd/`, `internal/domain/`, `internal/application/`, `internal/adapters/`.

## Running Locally

```bash
docker compose -f deploy/docker-compose.yml up -d

make build
make test
make test-integration
```

## Non-Functional Targets

| Metric | Target |
|--------|--------|
| Throughput | 10,000 TPS |
| p99 latency (sync ops) | < 200ms |
| Availability per service | 99.9% |
| Event store | Immutable, append-only |
