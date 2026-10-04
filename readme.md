# Enterprise Asynchronous LLM Inference Platform in Go & AWS

A high-throughput, cloud-native asynchronous Large Language Model (LLM) inference engine built with **Go (Golang)**, **AWS SQS FIFO**, **AWS Bedrock**, and **Amazon DynamoDB**.

Designed to handle heavy AI inference workloads by decoupling client HTTP request cycles from long-running foundation model invocations using non-blocking API patterns, worker pools, long-polling, and persistent state machines.

---

## Core Design Principles & Key Features

* **Asynchronous Non-Blocking Engine:** Clients receive an immediate `202 Accepted` acknowledgement with a `job_id` and poll endpoint, preventing HTTP timeouts on long-running LLM completions.
* **Strict Message Ordering & Deduplication:** AWS SQS FIFO queue with `MessageGroupId` and content/ID-based deduplication ensures single-instance execution per request.
* **Controlled Concurrency & Cost Optimization:** Managed Go worker pool with `goroutines`, channels, and long polling (`WaitTimeSeconds: 20`) minimizes API polling overhead and stays within AWS Free Tier thresholds.
* **Persistent Lifecycle State Tracking:** DynamoDB maintains job states (`PENDING` $\rightarrow$ `PROCESSING` $\rightarrow$ `COMPLETED` / `FAILED`) using expression attribute aliasing (`#resp`, `#s`) to manage database keywords safely.
* **Graceful Worker Shutdown:** Intercepts `SIGINT`/`SIGTERM` OS signals via `sync.WaitGroup` to allow active Bedrock jobs to complete before shutting down worker processes.

---

## Technology Stack

| Layer | Technology | Key Details |
| :--- | :--- | :--- |
| **Language & Runtime** | Go 1.22+ | Standard `goroutines`, channels, `sync.WaitGroup` |
| **API Framework** | Gin Web Framework | High-performance routing & JSON binding |
| **Message Broker** | AWS SQS (FIFO) | Ordered message execution & backpressure isolation |
| **Persistence Layer** | Amazon DynamoDB | `PAY_PER_REQUEST` billing with lifecycle tracking |
| **AI Inference Engine** | Amazon Bedrock | Foundation model execution |
| **Cloud SDK** | AWS SDK for Go v2 | Modular, high-performance Go SDK |

---

## Directory Structure

```text
aws-go-rag-platform/
├── cmd/
│   ├── api/          # REST API HTTP Gateway (Gin)
│   │   └── main.go
│   └── worker/       # Asynchronous Worker Pool Consumer
│       └── main.go
├── internal/
│   ├── queue/        # SQS polling logic & worker engine
│   │   └── worker.go
│   └── store/        # DynamoDB persistence layer & state updates
│       └── dynamo.go
├── docs/             # Visual architecture diagrams and assets
├── go.mod
├── go.sum
└── README.md

```

## Getting Started with Docker

### Prerequisites

* [Docker Desktop](https://www.docker.com/products/docker-desktop/) installed.
* Provisioned SQS FIFO queue and DynamoDB table in AWS.

### 1. Configure Environment Variables

Create a `.env` file in the repository root:

```env
SQS_QUEUE_URL=[https://sqs.us-east-1.amazonaws.com/](https://sqs.us-east-1.amazonaws.com/)<YOUR_ACCOUNT_ID>/rag-jobs.fifo
AWS_REGION=us-east-1
AWS_PROFILE=default

```
### 2. Launch Containers

```text
docker compose up --build
```

### Option B: Running Natively with Go CLI
Provision AWS Resources (CLI)

```text
Create SQS FIFO Queue
aws sqs create-queue \
  --queue-name rag-jobs.fifo \
  --attributes FifoQueue=true,ContentBasedDeduplication=true \
  --region us-east-1

# Create DynamoDB State Table
aws dynamodb create-table \
  --table-name rag-jobs \
  --attribute-definitions AttributeName=job_id,AttributeType=S \
  --key-schema AttributeName=job_id,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --region us-east-1
```
Start Services

Set environment variables:
```text
export SQS_QUEUE_URL="[https://sqs.us-east-1.amazonaws.com/](https://sqs.us-east-1.amazonaws.com/)<YOUR_ACCOUNT_ID>/rag-jobs.fifo"
export AWS_REGION="us-east-1"
```
Start the worker engine (Terminal 1):

```text
go run cmd/worker/main.go
```
Start the REST API server (Terminal 2):

```text
 go run cmd/api/main.go
 ```
### API Reference
```
## 1. Submit Inference Request

POST /jobs — Submits a prompt for asynchronous LLM processing.
## Request Body
```text

{
  "user_id": "usr-101",
  "prompt": "Explain Go concurrency models in 2 sentences."
}
```
Response (202 Accepted)
```text

{
  "created_at": "2026-10-04T13:37:25Z",
  "job_id": "job-1791101245043374000",
  "poll_url": "/jobs/job-1791101245043374000",
  "status": "PENDING"
}
```
Example curl

```text
curl -X POST http://localhost:8081/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "usr-101",
    "prompt": "Explain Go concurrency models in 2 sentences."
  }'
```
### 2. Poll Job Status & Retrieval

GET /jobs/:job_id — Polls execution status and retrieves Bedrock LLM completion output.
Response (200 OK - Processing Complete)
```text

{
  "job_id": "job-1791101245043374000",
  "user_id": "usr-101",
  "status": "COMPLETED",
  "prompt": "Explain Go concurrency models in 2 sentences.",
  "response": "Go handles concurrency using goroutines, which are lightweight threads managed by the Go runtime rather than the operating system. Goroutines communicate safely using channels, avoiding shared-memory lock contention through CSP principles.",
  "created_at": "2026-10-04T13:37:25Z",
  "updated_at": "2026-10-04T13:42:33Z"
}

```
Example curl
```text

curl -X GET http://localhost:8081/jobs/job-1791101245043374000
```