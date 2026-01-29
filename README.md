# Payment Orchestration System

A backend system for **correct, idempotent, and failure-tolerant payment processing**.

This project focuses on **system correctness under retries, crashes, and partial failures**, rather than throughput or UI concerns.  
The core goal is to model payments as **state machines driven by durable facts**, with strict separation between intent creation, execution, and reconciliation.

> ⚠️ **Status: In Progress**  
> The system currently implements **payment intent creation with idempotency guarantees**.  
> Execution, reconciliation, and projection components are under active development.

## ⚙️ Architecture
**Execution flow:**
<p align="center">
  <img src="docs/architecture.png" alt="Payment Orchestration System Architecture" width="900">
</p>