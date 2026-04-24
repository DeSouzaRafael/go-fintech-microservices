# Fintech Microservices Platform

[![Go Report Card](https://goreportcard.com/badge/github.com/DeSouzaRafael/go-fintech-microservices)](https://goreportcard.com/report/github.com/DeSouzaRafael/go-fintech-microservices)
[![CI](https://github.com/DeSouzaRafael/go-fintech-microservices/actions/workflows/ci.yml/badge.svg)](https://github.com/DeSouzaRafael/go-fintech-microservices/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/DeSouzaRafael/go-fintech-microservices/branch/main/graph/badge.svg)](https://codecov.io/gh/DeSouzaRafael/go-fintech-microservices)
[![License](https://img.shields.io/github/license/DeSouzaRafael/go-fintech-microservices.svg)](https://github.com/DeSouzaRafael/go-fintech-microservices/blob/main/LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25.9-00ADD8?logo=go)](https://go.dev/)

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
- [x] Shared `.proto` contracts (`wallet.proto`, `transaction.proto`, `identity.proto`, `fraud.proto`, `query.proto`)
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
- [x] Fraud Service: daily limit rule
- [x] Fraud Service: velocity check (N txns in T seconds)
- [x] Fraud Service: user risk profile cache in Redis
- [x] Fraud Service: async profile update via Kafka consumer
- [x] Transaction Service: call Fraud before committing
- [x] Notification Service: consume `TransactionCompleted`
- [x] Notification Service: consume `TransactionFailed`
- [x] Notification Service: consume `FraudDetected`
- [x] Notification Service: idempotency via processed events table
- [x] API Gateway: rate limiting per user (Redis token bucket)

### 5 — CQRS & API Gateway
- [x] Query Service: consume wallet events, build balance projection
- [x] Query Service: consume transaction events, build statement projection
- [x] Query Service: paginated statement endpoint (gRPC + REST)
- [x] Query Service: per-user statistics projection
- [x] API Gateway: full grpc-gateway setup
- [x] API Gateway: JWT validation middleware
- [x] API Gateway: route all services
- [x] OpenAPI spec generated from proto annotations

### 6 — Observability, Resilience & Load
- [x] Grafana dashboards: RED metrics per service
- [x] Grafana dashboards: p50/p95/p99 latency panels
- [x] Grafana dashboards: Kafka consumer lag
- [x] Circuit breakers (`sony/gobreaker`) on inter-service calls
- [x] Retry with exponential backoff on Kafka publish
- [x] k6 load test: 1,000 TPS baseline
- [x] k6 load test: 10,000 TPS stress
- [x] Architecture Decision Records (ADRs)
- [ ] Distributed trace screenshots (Jaeger)
- [ ] Load test results in `docs/`

## Project Structure

```
go-fintech-microservices/
├── api/proto/                  # .proto sources + generated Go stubs (*/v1/)
├── third_party/googleapis/     # google/api protos for grpc-gateway annotations
├── pkg/
│   ├── breaker/                # sony/gobreaker circuit breaker wrapper
│   ├── errors/                 # domain error types + gRPC mapping
│   ├── kafka/                  # franz-go consumer wrapper
│   ├── logger/                 # zap structured logging
│   ├── metrics/                # OTel Prometheus exporter
│   ├── middleware/             # gRPC interceptors (auth, tracing, logging, recovery)
│   ├── server/                 # gRPC server with graceful shutdown
│   └── tracing/                # OTel SDK + OTLP exporter setup
├── services/
│   ├── identity/               # Auth, JWT, refresh tokens
│   ├── wallet/                 # Event Sourcing, saga consumer
│   ├── transaction/            # Saga orchestration, fraud check
│   ├── fraud/                  # Rules engine, Redis profile cache
│   ├── notification/           # Idempotent Kafka consumer
│   ├── query/                  # CQRS read model
│   └── gateway/                # grpc-gateway HTTP proxy, JWT + rate limit
├── deploy/
│   ├── docker-compose.yml      # Postgres ×6, Redis, Redpanda, Jaeger, Prometheus, Grafana
│   └── grafana/                # dashboards + provisioning
├── tests/
│   └── load/                   # k6 baseline (1k TPS) and stress (10k TPS) scripts
└── docs/
    ├── openapi/fintech.swagger.json
    └── adr/                    # Architecture Decision Records (001–004)
```

Each service: hexagonal architecture — `cmd/`, `internal/domain/`, `internal/application/`, `internal/adapters/`.

## Running Locally

```bash
# 1. Start infrastructure (Postgres, Redis, Redpanda, Jaeger, Prometheus, Grafana)
docker compose -f deploy/docker-compose.yml up -d

# 2. Run database migrations
make migrate-up

# 3. Build all services
make build

# 4. Run tests
make test

# Regenerate proto stubs + OpenAPI spec
make proto
```

| UI | URL |
|----|-----|
| Redpanda Console | http://localhost:18080 |
| Grafana | http://localhost:13000 (admin/admin) |
| Jaeger | http://localhost:16686 |
| Prometheus | http://localhost:9090 |
| OpenAPI spec | `docs/openapi/fintech.swagger.json` |

## Non-Functional Targets

| Metric | Target |
|--------|--------|
| Throughput | 10,000 TPS |
| p99 latency (sync ops) | < 200ms |
| Availability per service | 99.9% |
| Event store | Immutable, append-only |
