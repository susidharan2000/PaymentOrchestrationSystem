# Payment Orchestration System

> A payment system that **guarantees financial correctness under retries, crashes, and concurrency**.

---

### 🚀 Key Highlights

- No duplicate financial effects under retries and crashes  
- No over-refund under concurrent requests  
- Crash-safe recovery via deterministic ledger replay  
- Idempotency enforced across API, database, and event layers  
- Designed for failure as the default condition  

---

Most systems optimize for success cases.  
This system is designed for **failure as the default condition**.

---

_**Tech Stack**: Go, PostgreSQL_  
_**System Scope**: Single-tenant, single-PSP (Stripe)_  
_**Extensibility**: Multi-PSP (adapter pattern), multi-tenant via scoped idempotency_

---

## Overview

Payment Orchestration System processes payments and refunds  
**without ever producing incorrect financial state**.

It is built on a simple principle:

> 👉 **Correctness must not depend on successful execution paths**

- All financial actions are immutable  
- State is derived, not mutated  
- Failures are expected and handled deterministically  

---

## Problem Statement

Payment systems fail in non-obvious ways:

- Duplicate requests → multiple captures  
- Partial failures → inconsistent state  
- Concurrent refunds → over-refund  
- Missed webhooks → lost financial updates  

Preventing retries is impossible.

> 👉 **The real problem is ensuring correctness despite failures**

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

> 👉 **Financial state must always be correct and derivable from history**

Achieved by:

- Append-only ledger as source of truth  
- Asynchronous state projection  
- Idempotency at every layer  
- Reservation model for safe refunds  

---

## ⚙️ Architecture

<p align="center">
  <img src="docs/architectureV1.png" width="900">
</p>

---

## End-to-End Flow

### Payment Flow

1. Client initiates payment → payment_intent created  
2. External PSP processes payment  
3. Webhook / reconciliation confirms outcome  
4. PAYMENT event written to ledger  
5. Projector updates derived state  

### Refund Flow

1. Client requests refund  
2. Validate using reservation model  
3. refund_record created (PENDING)  
4. Worker executes refund  
5. Webhook / reconciliation confirms  
6. REFUND event written to ledger  
7. Projector updates derived state  

---

## Core Concepts

### Ledger (Source of Truth)
Immutable record of all confirmed financial actions.  
Enables auditability and deterministic replay.

### Projection (Derived State)
State is asynchronously derived from the ledger.  
Separates correctness from read performance.

### Idempotency
Enforced across:
- API layer  
- Database constraints  
- Ledger uniqueness  
- Projector sequencing  

### Concurrency Control
- Row-level locking  
- Reservation model  
- Deterministic refund allocation  

---

## System Guarantees

- No duplicate financial effects  
- No over-refund under concurrency  
- Deterministic recovery via replay  
- Atomic financial operations  
- Eventual consistency for reads  
- At-least-once processing (safe by design)  

---

## Performance Characteristics

| Component | Bottleneck |
|----------|-----------|
| Writes | Database transactions |
| Workers | External PSP latency |
| Reads | Projection lag |

- Write latency → DB commit  
- Processing latency → PSP  
- Read latency → near real-time (eventual consistency)  

---

## Observability

- Payment state transitions  
- Refund lifecycle tracking  
- Webhook ingestion status  
- Retry and failure metrics  
- Reconciliation activity  

---

## Failure Model

| Failure Scenario | System Behavior |
|------------------|-----------------|
| API retry | Idempotent handling |
| Worker crash | Safe retry |
| Missed webhook | Reconciliation recovery |
| Duplicate events | Deduplicated |
| Partial failure | Transaction rollback |

---

## Non-Goals

- Exactly-once processing  
- Strong read consistency  
- Multi-region coordination  
- High-throughput streaming  

---

## Design Details

👉 [Full Design Document](docs/DESIGN.pdf)

---

## ⚙️ Configuration

Create a `.env` file:

```env
# Database
DATABASE_URL=postgres://payment_user:payment_pass@localhost:5432/payment_orchestration?sslmode=disable

# Stripe
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