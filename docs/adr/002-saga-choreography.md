# ADR 002 — Saga Choreography via Kafka for Distributed Transactions

**Status:** Accepted  
**Date:** 2026-01-15

## Context

A fund transfer spans three services: Transaction (initiator), Wallet-source (debit), and Wallet-destination (credit). Each owns its own database. A distributed transaction across them would require 2PC, which introduces tight coupling and is unavailable when any participant is down.

## Decision

We use **Saga choreography** over Kafka. No central orchestrator exists. Each service reacts to domain events and emits its own:

```
TransactionInitiated → (Wallet) FundsReserved → (Transaction) → (Wallet) FundsDeposited → TransactionCompleted
                                               → FundsReleased (on any failure path)
```

The **Transactional Outbox** pattern guarantees exactly-once publishing: each service writes events to an `outbox_events` table in the same DB transaction as its state change. A background worker polls and publishes. Consumers deduplicate via `processed_events`.

## Consequences

**Benefits:**
- Services are loosely coupled and deployable independently
- No single point of failure in the happy path
- Each service can replay events to recover from downtime without re-triggering side effects (idempotency)
- Compensation is a natural domain event (`FundsReleased`), not a rollback

**Trade-offs:**
- No global transaction visibility without the Query Service
- Debugging a failed saga requires correlating events across multiple services by `transaction_id`
- Schema changes to event payloads require backward-compatible rollouts
- Velocity of the saga is bounded by Kafka end-to-end latency (~5–15ms per hop)
