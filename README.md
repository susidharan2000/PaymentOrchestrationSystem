# Payment Orchestration System

A payment system that **guarantees financial correctness under retries, crashes, and concurrency**.

---

### 🚀 Key Highlights

- No duplicate financial effects under retries and crashes  
- No over-refund under concurrent requests  
- Crash-safe recovery via deterministic ledger replay  
- Idempotency enforced across API, database, and event layers  
- Designed for failure as the default condition  

--- 

_**Tech Stack**: Go, PostgreSQL_  
_**System Scope**: Single-tenant, single-PSP (Stripe)_  
_**Extensibility**: Multi-PSP (adapter pattern), multi-tenant via scoped idempotency_

---

## Overview

Payment Orchestration System is a backend system designed to process payments and refunds **without creating incorrect financial state under failure**.

The system is intentionally built around correctness:
- All financial actions are persisted as immutable events  
- Execution is idempotent across all layers  
- Failures are expected and handled deterministically  


---

## Problem Statement

Payment systems fail in non-obvious ways:

- Duplicate requests → multiple captures  
- Partial failures → inconsistent state  
- Concurrent refunds → over-refund  
- Missed webhooks → lost financial updates  

Preventing retries is impossible.  
The real problem is:

  👉 **Ensuring financial correctness despite retries, failures, and concurrency**

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

The system is built on a single invariant:

👉 **Financial state must always be correct and derivable from history**

This is achieved by:

- Using an **append-only ledger** as the source of truth  
- Deriving state asynchronously via projection  
- Enforcing **idempotency at every layer**  
- Using a **reservation model** to prevent over-refund  

👉 External PSPs are treated as **unreliable, non-transactional systems**.  
The system never trusts external state without **verification via webhook or reconciliation**.

Correctness is enforced by design, not by retry logic.

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
2. Client confirms payment (Stripe SDK) 
3. External PSP processes payment  
4. Webhook / reconciliation confirms outcome  
5. PAYMENT event written to ledger  
6. Projector updates derived state  

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
- **At-least-once processing with logical exactly-once effects (via idempotency and deduplication)**  

Duplicate processing is allowed by design but does not affect correctness.

---

## Failure Handling

The system is designed with failures as a **first-class concern**.

| Scenario | Handling |
|----------|----------|
| API retry | Idempotent request handling prevents duplicate effects |
| Duplicate webhook delivery | Deduplicated using unique (event_id, psp_name) constraint |
| Concurrent refunds | Prevented using row-level locking + reservation model |
| Worker crash | Safe retry due to idempotent execution |
| Missing webhook | Reconciliation ensures eventual correctness |
| Partial failure | Transaction rollback prevents inconsistent state |

### Key Properties

- All operations are **idempotent**
- Financial effects are applied **exactly once logically**
- System guarantees **deterministic recovery**
- Failures do not lead to **incorrect financial state**

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

## ▶️ Running the System

Follow the steps below to set up and run the system locally.

---

### 1. Prerequisites

Ensure you have the following installed:

- Go (1.20+)
- PostgreSQL
- Stripe account (test keys)
- Stripe CLI (for webhook forwarding)

---

### 2. Create Database

```bash
createdb payment_orchestration
```

---

### 3. Configuration

Create a `.env` file in the project root:

```env
DATABASE_URL=postgres://payment_user:payment_pass@localhost:5432/payment_orchestration?sslmode=disable

STRIPE_SECRET_KEY=your_stripe_secret_key
STRIPE_WEBHOOK_SECRET=your_webhook_secret
STRIPE_PUBLISHABLE_KEY=your_publishable_key

REFUND_WORKER_COUNT=10
WEBHOOK_WORKER_COUNT=10

PAYMENT_RECONCILER_BATCH_SIZE=10
PAYMENT_RECONCILER_CONCURRENCY=10

REFUND_RECONCILER_BATCH_SIZE=10
REFUND_RECONCILER_CONCURRENCY=10
```

---

### 4. Run Migrations

```bash
psql "$DATABASE_URL" -f migrations/000_init.sql
```

---

### 5. Start the Server

```bash
go build -o bin/server ./cmd/server
./bin/server
```

Server runs at:

```bash
http://localhost:8080
```

---

### 6. Install and Setup Stripe CLI

```bash
brew install stripe/stripe-cli/stripe
stripe login
```

---

### 7. Start Stripe Webhooks

```bash
stripe listen --forward-to localhost:8080/webhooks/psp/stripe
```

You will receive:

```bash
Ready! Your webhook signing secret is: whsec_xxx
```

Update your `.env`:

```env
STRIPE_WEBHOOK_SECRET=whsec_xxx
```

---

### Notes

- Use Stripe test mode only  
- Webhooks are required for payment confirmation  

--- 


## 🧪 Testing the System

This system requires **client-side confirmation** to complete payments.

👉 This design intentionally separates payment creation from payment confirmation,
mirroring real-world PSP flows (e.g., Stripe Payment Intents).

---

### ⚠️ Important

Creating a payment via API does **NOT** move money.

- Payment remains in `PROCESSING`
- No webhook is triggered
- No ledger entry is created

👉 Payment is only completed after **client confirmation**

---

### How to Complete a Payment

1. Create a payment:

```bash
POST /payments
```

2. Copy `client_secret` from response

3. Run `view/checkout.html` in browser

4. Enter the card details and confirm the payment

5. Use Stripe test card:

```bash
4242 4242 4242 4242
```

---

### What happens after confirmation

- Stripe processes payment  
- Webhook is triggered  
- System writes PAYMENT event to ledger  
- State updates → `CAPTURED`  

---

### Refund Testing

Refunds can be triggered via API:

```bash
POST /payments/{payment_id}/refund
```

Or via curl for quick testing.

---

### Note

- `checkout.html` is a minimal test UI  
- In production, this is replaced by frontend (React / mobile app)  
- Uses Stripe SDK for secure confirmation

---

## API Reference

Base URL:

http://localhost:8080

---

### 💳 Create Payment

Creates a new payment intent and initializes processing with PSP.

> ⚠️ Payment remains in PROCESSING until client-side confirmation and webhook reconciliation

> The system guarantees that retries with the same idempotency key will never create duplicate financial effects.

**Endpoint**

POST /payments

**Request**

```json
{
  "amount": 1000,
  "currency": "INR",
  "psp_name": "stripe",
  "idempotency_key": "new_idempotency_key_3005"
}
```

**Response**

```json
{
  "payment_id": "0825a8b4-45f1-43e2-b2f4-d6d075ce438a",
  "amount": 1000,
  "currency": "INR",
  "status": "PROCESSING",
  "psp_name": "stripe",
  "client_secret": "pi_xxx_secret_xxx",
  "publishable_key": "pk_test_xxx"
}
```

---

### 🔍 Get Payment

Fetch current state of a payment.

**Endpoint**

GET /payments/{payment_id}

**Response**

```json
{
  "amount": 500000,
  "currency": "INR",
  "status": "CAPTURED",
  "psp_name": "stripe"
}
```

---

### 💸 Create Refund

Initiates a refund for a given payment.

**Endpoint**

POST /payments/{payment_id}/refund

**Request**

```json
{
  "amount": 100000,
  "idempotency_key": "sample_idempotency_key_02"
}
```

**Response**

```json
{
  "refund_id": "cba16fd9-ff39-4764-89ea-87566c08e55a",
  "payment_id": "5c0e342c-01b1-418e-9e6a-99ae067286a8",
  "amount": 100000,
  "currency": "INR",
  "status": "CREATED",
  "created_at": "2026-03-28T11:03:08.389037+05:30"
}
```

---

### 🔔 PSP Webhook

Receives asynchronous events from external payment service providers.

**Endpoint**

POST /webhooks/psp/{psp_name}

**Example**

POST /webhooks/psp/stripe


  

