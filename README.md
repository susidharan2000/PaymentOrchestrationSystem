# Payment Orchestration System

A payment system designed to guarantee financial correctness under retries, crashes, and concurrent execution.

Most payment systems break under failure:
- The same request executes twice → double charge  
- A crash occurs mid-operation → inconsistent state  
- Concurrent refunds exceed captured amount → financial corruption  

This system prevents these failures by modeling payments as:
- **Immutable financial events (ledger)**  
- **Deterministic state derived from history**  
- **Strict idempotency and reservation-based validation**  

The result is a system where:
- retries are safe  
- failures are recoverable  
- correctness is guaranteed by design  

---

## Why this exists

Real-world payment systems are unreliable:

- Requests are retried due to network failures  
- Systems crash during processing  
- External payment providers behave unpredictably  

Without careful design, this leads to:
- Duplicate charges or refunds  
- Inconsistent financial state  
- Race conditions under concurrency  

This project focuses on building a system that remains correct under all these conditions.

---

## Core Ideas

- Ledger as source of truth  
  All financial actions are stored as immutable events  

- Projection-based state  
  Current payment state is derived asynchronously  

- Idempotency at every layer  
  Safe retries across API, workers, and event ingestion  

- Reservation-based refunds  
  Prevents over-refund under concurrent requests  

- At-least-once processing  
  Duplicate processing is allowed but made safe via deduplication  

---

## Architecture Overview

![Architecture Diagram](./docs/architectureV1.png)

1. Incoming requests generate financial events (PAYMENT / REFUND)  
2. Events are stored in an append-only ledger  
3. A projector processes events and derives current state  
4. Reads are served from the derived state  

---

## Key Guarantees

- No duplicate financial effects  
- No over-refund under concurrency  
- Deterministic recovery via ledger replay  
- No partial state visibility (atomic operations)  
- Eventual consistency for reads  
- Safe handling of retries and duplicate events  

---

## Trade-offs

- Increased system complexity due to event-driven design  
- Eventual consistency for read models  
- Database used as coordination layer (limits scalability)  

---

## Limitations

- Database can become a bottleneck under high load  
- Worker polling introduces inefficiency  
- Projector throughput is limited by sequential processing  
- Single-region deployment (no cross-region fault tolerance)  

---

## Full Design Document

For detailed architecture, data model, failure scenarios, and trade-offs:

👉 [Design Document](./)

---

## Key Highlights

- Designed for correctness over performance  
- Handles failures as a first-class concern  
- Ensures financial safety under concurrency  
- Supports deterministic recovery via replay  

---

## Future Improvements

- Move to queue-based architecture (Kafka / message broker)  
- Partition projector for parallel processing  
- Introduce multi-region support  
- Optimize worker coordination and reduce polling  

---