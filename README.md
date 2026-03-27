# Payment Orchestration System

> **A correctness-first payment processing system**  
> _Designed to guarantee financial safety under retries, crashes, and concurrency_
>
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

## Guarantees

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

## Build & Run

### Prerequisites

- **Go 1.20+**
- **PostgreSQL**

---

### Configuration

Environment variables:

```bash
DB_URL=postgres://user:password@localhost:5432/payments
WORKER_COUNT=10