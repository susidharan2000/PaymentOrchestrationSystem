# Payment Orchestration System

> A payment system that **guarantees financial correctness under retries, crashes, and concurrency**.

### 🚀 Key Highlights

> - Guarantees **no duplicate financial effects** under retries and crashes  
> - Prevents **over-refund under concurrent requests** using reservation model  
> - Uses **append-only ledger** as source of truth for deterministic replay  
> - Enforces **idempotency across API, DB, and event layers**  
> - Designed for **failure as default condition**, not success  

---

> Most systems optimize for success cases.  
> This system is designed for **failure as the default condition**.
>
--- 

_**Tech Stack**: Go, PostgreSQL_  
_**System Scope**: Single-tenant, single-PSP (Stripe)_  
_**Extensibility**: Multi-PSP (adapter pattern), multi-tenant via scoped idempotency_

> **What this system guarantees:**
> - No duplicate financial effects under retries  
> - No over-refund under concurrency  
> - Crash-safe recovery via deterministic replay  
> - Financial state derived entirely from immutable history  

---

## Overview

> Payment Orchestration System is a backend system designed to process payments and refunds **without creating incorrect financial state under failure**.
>
> The system is intentionally built around correctness:
> - All financial actions are persisted as immutable events  
> - Execution is idempotent across all layers  
> - Failures are expected and handled deterministically  

>
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

Naive systems directly mutate state during request handling.

Under real-world conditions:
- Retries → duplicate effects  
- Crashes → partial state  
- Concurrency → race conditions  

👉 Direct state mutation cannot guarantee correctness.

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

👉 **[Full Design Document](docs/DESIGN.pdf)**

---

## ⚙️ Configuration

Create a `.env` file in the project root:

```env
# Database
DATABASE_URL=postgres://payment_user:payment_pass@localhost:5432/payment_orchestration?sslmode=disable

# Stripe (use your own test keys)
STRIPE_SECRET_KEY=your_stripe_secret_key
STRIPE_WEBHOOK_SECRET=your_webhook_secret
STRIPE_PUBLISHABLE_KEY=your_publishable_key

# Workers
REFUND_WORKER_COUNT=10
WEBHOOK_WORKER_COUNT=10

# Payment Reconciliation
PAYMENT_RECONCILER_BATCH_SIZE=10
PAYMENT_RECONCILER_CONCURRENCY=10

# Refund Reconciliation
REFUND_RECONCILER_BATCH_SIZE=10
REFUND_RECONCILER_CONCURRENCY=10
```




