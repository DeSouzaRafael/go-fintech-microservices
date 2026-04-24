# ADR 004 — Transactional Outbox for Reliable Event Publishing

**Status:** Accepted  
**Date:** 2026-01-15

## Context

Publishing a Kafka message and committing a DB transaction are two separate I/O operations. A service crash between them produces either a lost event (commit succeeded, publish failed) or a duplicate event (publish succeeded, commit failed). Both break the saga.

## Decision

All services that produce domain events use the **Transactional Outbox** pattern:

1. Within a single DB transaction, write the domain state change **and** insert a row into `*_outbox_events`.
2. A background `OutboxWorker` polls `FetchUnpublished`, calls `ProduceSync` (all-ISR acks), then calls `MarkPublished`.
3. Consumers deduplicate using `processed_events` keyed on `event_id`.

The outbox worker uses **exponential backoff** (100ms → 5s, max 30s) and a **circuit breaker** (opens after 5 consecutive failures, resets after 30s) to handle transient Kafka unavailability without hammering a down broker.

## Consequences

**Benefits:**
- Events are never lost: the outbox survives service crashes
- At-least-once delivery is guaranteed; exactly-once is achieved via consumer idempotency
- Circuit breaker prevents Kafka outage from cascading into DB saturation

**Trade-offs:**
- Outbox table grows until the worker clears it; requires a periodic cleanup job for old published rows
- Worker polling interval (500ms) introduces up to 500ms latency on the publish side
- Adding a new event type requires modifying both the domain and the worker topic routing
