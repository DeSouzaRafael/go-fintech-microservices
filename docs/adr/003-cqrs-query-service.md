# ADR 003 — CQRS Read Model via Dedicated Query Service

**Status:** Accepted  
**Date:** 2026-01-15

## Context

The Wallet service stores state as an event stream. Answering "what is the current balance?" by replaying all events on every read request is unacceptable at scale. The Wallet service also serves write commands and should not be burdened with projection logic.

Account statements and per-user statistics are read-heavy workloads with different schema requirements from the write side.

## Decision

A dedicated **Query Service** implements the read side of CQRS. It subscribes to wallet and transaction Kafka events, materializes three projections into its own PostgreSQL schema:

| Projection | Table | Updated by |
|---|---|---|
| Balance | `balance_projections` | `WalletCreated`, `FundsDeposited`, `FundsWithdrawn` |
| Statement | `statement_entries` | `FundsDeposited`, `FundsWithdrawn` |
| User stats | `user_stats` | `TransactionCompleted` |

Idempotency is enforced via `query_processed_events`. The Query Service exposes gRPC endpoints with REST via grpc-gateway.

## Consequences

**Benefits:**
- Wallet service writes are never blocked by read workload
- Projections can be schema-optimised for query patterns (pagination, aggregation)
- Read model can be rebuilt by replaying Kafka from the beginning
- Independent scaling: Query Service can be scaled out for read-heavy load

**Trade-offs:**
- Eventual consistency: balance reads lag behind writes by Kafka round-trip (~5–15ms typical)
- Two schemas to maintain for the same domain concept
- Rebuild takes time for large event histories
