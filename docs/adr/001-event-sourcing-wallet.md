# ADR 001 — Event Sourcing for Wallet Service

**Status:** Accepted  
**Date:** 2026-01-15

## Context

Wallet balances are the core financial asset in the platform. Any mutation (deposit, withdrawal, reservation) must be auditable, reproducible, and protected against concurrent write corruption.

Standard CRUD stores only the latest state. A bug or bad deployment could silently corrupt balances with no recovery path beyond backups.

## Decision

The Wallet service uses **Event Sourcing**: state is never stored directly. Every mutation appends an immutable `WalletEvent` row to `wallet_events`. Balance is reconstructed by replaying events in sequence. Snapshots are taken every 50 events to bound replay cost.

Optimistic concurrency is enforced via an expected-version check on `AppendEvents`, preventing split-brain writes under concurrent requests.

## Consequences

**Benefits:**
- Complete, tamper-evident audit trail of every cent moved
- Time-travel debugging: replay up to any point in time
- Trivial compensation: append a reversal event instead of mutating state
- No lost updates under concurrent writes (optimistic lock raises a conflict error)

**Trade-offs:**
- Reads require event replay (mitigated by snapshots and the CQRS Query Service)
- Schema evolution of event payloads requires versioned migration strategy
- Operational complexity: event store must be append-only; `DELETE` must be disabled at the DB level
