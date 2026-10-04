# Asynchronous LLM Processing & Inference Engine

## Executive Summary
This project implements a serverless, decoupled Go backend microservice designed to process high-throughput LLM inference jobs asynchronously. By placing AWS SQS FIFO in front of Amazon Bedrock (Nova Micro), the system eliminates synchronous HTTP gateway blocking, enforces concurrency backpressure, and guarantees zero-idle infrastructure costs.

---

## 1. High-Level Architecture Diagram

```text
[ Client Application ]
          │
          ├─► POST /jobs ───────────► [ REST API Gateway ] ────► [ DynamoDB: PENDING ]
          │                           (cmd/api - Gin)                  │
          │                                 │                          │
          │                                 ▼                          │
          │                           [ SQS FIFO Queue ] ──────────────┘
          │                           (rag-jobs.fifo)
          │                                 │
          │                                 ▼
          │                           [ Concurrent Worker Pool ]
          │                           (cmd/worker - Go Goroutines)
          │                                 │
          │                        ┌────────┴────────┐
          │                        ▼                 ▼
          │               [ Amazon Bedrock ]  [ DynamoDB Store ]
          │               (Claude/Nova model) (PROCESSING -> COMPLETED)
          │                                          ▲
          └─► GET /jobs/:job_id ─────────────────────┘