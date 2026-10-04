# Asynchronous LLM Processing & Inference Engine

## Executive Summary
This project implements a serverless, decoupled Go backend microservice designed to process high-throughput LLM inference jobs asynchronously. By placing AWS SQS FIFO in front of Amazon Bedrock (Nova Micro), the system eliminates synchronous HTTP gateway blocking, enforces concurrency backpressure, and guarantees zero-idle infrastructure costs.

---

## 1. High-Level Architecture Diagram

```text
 ┌─────────────────┐       1. Send Prompt        ┌──────────────────────────────┐
 │  cmd/producer   │ ──────────────────────────► │ AWS SQS FIFO Queue           │
 │ (Go CLI/Client) │                             │                              │
 └─────────────────┘                             └──────────────┬───────────────┘
                                                                │
                                                                │ 2. Long Poll
                                                                │    WaitTime: 20s
                                                                ▼
┌───────────────────────────────────────────────────────────────────────────────┐
│  cmd/worker (Go Microservice Engine)                                         │
│                                                                               │
│   ┌──────────────────────────────────────────────────────────────────────┐    │
│   │  SQS Receiver Loop                                                   │    │
│   └──────────────────────────────────┬───────────────────────────────────┘    │
│                                      │                                        │
│                                      │ 3. Push to Channel                     │
│                                      ▼                                        │
│   ┌──────────────────────────────────────────────────────────────────────┐    │
│   │  jobs channel (chan types.Message) [Buffer Size = 6]                 │    │
│   └──────┬───────────────────────────┬───────────────────────────┬───────┘    │
│          │                           │                           │            │
│          │ 4. Compete to Pull        │                           │            │
│          ▼                           ▼                           ▼            │
│   ┌──────────────┐            ┌──────────────┐            ┌──────────────┐    │
│   │ Worker Gorut.│            │ Worker Gorut.│            │ Worker Gorut.│    │
│   │   [ID: 1]    │            │   [ID: 2]    │            │   [ID: 3]    │    │
│   └──────┬───────┘            └──────┬───────┘            └──────┬───────┘    │
└──────────┼───────────────────────────┼───────────────────────────┼────────────┘
           │                           │                           │
           └───────────────────────────┼───────────────────────────┘
                                       │ 5. Invoke Bedrock API
                                       ▼
                         ┌──────────────────────────────┐
                         │ Amazon Bedrock Runtime       │
                         │ (Amazon Nova / Claude)       │
                         └──────────────┬───────────────┘
                                        │
                                        │ 6. Write Response (Phase 3 Target)
                                        ▼
                         ┌──────────────────────────────┐
                         │ Result Store                 │
                         │ (DynamoDB / S3)              │
                         └──────────────────────────────┘