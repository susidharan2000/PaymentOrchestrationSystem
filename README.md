# Payment Orchestration System

> A payment system that **never produces incorrect financial state**, even under retries, crashes, and concurrent operations.

Most systems optimize for success cases.  
This system is designed for **failure as the default condition**.
> _**Tech Stack**: Go, PostgreSQL, Event-Driven Architecture, Background Workers_

---

## Overview

> Payment Orchestration System is a backend system designed to process payments and refunds **without creating incorrect financial state under failure**.
>
> The system is intentionally built around correctness:
> - All financial actions are persisted as immutable events  
> - Execution is idempotent across all layers  
> - Failures are expected and handled deterministically  
>
> Unlike typical systems that mutate state directly, this system treats financial operations as immutable facts and derives state from them.
---

## Problem Statement

> Payment systems fail in non-obvious ways:
>
> - Duplicate requests → multiple captures  
> - Partial failures → inconsistent state  
> - Concurrent refunds → over-refund  
> - Missed webhooks → lost financial updates  
>
> Preventing retries is impossible.  
> The real problem is:
>
> 👉 **Ensuring financial correctness despite retries, failures, and concurrency**

---

## Why naive systems fail

A naive system directly updates payment state during request handling.

This breaks under real-world conditions:

- Retries → duplicate charges or refunds  
- Crashes → partial state updates  
- Concurrency → race conditions and over-refund  

This system avoids these issues by:

- Recording only **confirmed financial events**
- Separating **intent from execution**
- Deriving state instead of mutating it directly

---

## Core Design Principle

> The system is built on a single invariant:
>
> 👉 **Financial state must always be correct and derivable from history**
>
> This is achieved by:
>
> - Using an **append-only ledger** as the source of truth  
> - Deriving state asynchronously via projection  
> - Enforcing **idempotency at every layer**  
> - Using a **reservation model** to prevent over-refund  
>
> Correctness is enforced by design, not by retry logic.

---

## ⚙️ Architecture

**Execution flow:**
<p align="center">
  <img src="docs/architectureV1.png" alt="Payment Orchestration System Architecture" width="900">
</p>

---

## End-to-End Flow

### PAYMENT flow:

1. Client initiates payment → payment_intent created  
2. External PSP processes payment  
3. Webhook / reconciliation confirms outcome  
4. PAYMENT event written to ledger  
5. Projector updates derived state  

### Refund flow:

1. Client requests refund  
2. System validates using reservation model  
3. refund_record created (PENDING)  
4. Worker executes refund via PSP  
5. Webhook / reconciliation confirms  
6. REFUND event written to ledger  
7. Projector updates derived state  

---

## Core Concepts

### Ledger (Source of Truth)
All confirmed financial actions are recorded as immutable events.
The ledger is append-only and guarantees auditability and replayability.

### Projection (Derived State)
The current payment state is derived asynchronously from the ledger.
This separates write correctness from read performance.

### Idempotency
Idempotency is enforced across all layers:
- API layer (request hash / idempotency key)
- Database layer (unique constraints)
- Ledger (external reference uniqueness)
- Projector (event sequence tracking)

This ensures retries do not create duplicate financial effects.

### Concurrency Control
Refunds use row-level locking and a reservation model:
- Prevents race conditions
- Prevents over-refund
- Ensures deterministic allocation under concurrency

---

## Design Philosophy

The system prioritizes:
- Correctness over performance  
- Determinism over convenience  
- Failure recovery over success-path optimization  

---

## System Guarantees

This system guarantees:

- **No duplicate financial effects**  
- **No over-refund under concurrency**  
- **Deterministic recovery via ledger replay**  
- **No partial state visibility (atomic operations)**  
- **Eventual consistency for reads**  
- **At-least-once processing with idempotency**

Duplicate processing is allowed by design but does not affect correctness.

---

## Performance Characteristics

System performance depends on workload characteristics:

| Component | Bottleneck |
|----------|-----------|
| Writes | Database transactions |
| Workers | External PSP latency |
| Reads | Projection lag |

Under typical workload:
- Write latency is dominated by DB commit  
- Processing latency depends on external providers  
- Read latency is near real-time but eventually consistent  

The system prioritizes **correctness over throughput**.

---

## Observability

The system exposes operational visibility via:

- payment state transitions  
- refund lifecycle tracking  
- webhook ingestion status  
- retry and failure counts  
- reconciliation activity  

These signals help verify correctness under failure scenarios.

---

## Failure Model

The system assumes failures are normal.

| Failure Scenario | System Behavior |
|------------------|-----------------|
| API retry | Idempotent request handling prevents duplication |
| Worker crash | Job retried safely via idempotent execution |
| Missed webhook | Reconciliation recovers state |
| Duplicate events | Deduplicated at ledger level |
| Partial failure | Transaction rollback prevents corruption |

Failure handling is **deterministic and repeatable**.

---

## Non-Goals

The system explicitly does **NOT** attempt to solve:

- **Exactly-once processing**  
- **Strong consistency for reads**  
- **Distributed multi-region coordination**  
- **High-throughput streaming systems**  

These trade-offs simplify correctness and failure handling.

---

## Design Details

**Full design rationale, data model, and failure scenarios:**

👉 **[DESIGN.md](./DESIGN.md)**

---
