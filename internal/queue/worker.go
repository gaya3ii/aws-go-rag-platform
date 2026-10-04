package queue

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/gaya3ii/aws-go-rag-platform/internal/store"
)

type WorkerPool struct {
	sqsClient     *sqs.Client
	bedrockClient *bedrockruntime.Client
	dynamoStore   *store.DynamoStore
	queueURL      string
	workerCount   int
	jobs          chan types.Message
	wg            sync.WaitGroup
}

func NewWorkerPool(
	sqsClient *sqs.Client,
	bedrockClient *bedrockruntime.Client,
	dynamoStore *store.DynamoStore,
	queueURL string,
	workerCount int,
) *WorkerPool {
	return &WorkerPool{
		sqsClient:     sqsClient,
		bedrockClient: bedrockClient,
		dynamoStore:   dynamoStore,
		queueURL:      queueURL,
		workerCount:   workerCount,
		jobs:          make(chan types.Message, workerCount*2),
	}
}

// Start launches the worker goroutines and begins long polling SQS.
func (w *WorkerPool) Start(ctx context.Context) {
	log.Printf("Starting worker pool with %d concurrent workers...", w.workerCount)

	// Launch worker goroutines
	for i := 1; i <= w.workerCount; i++ {
		w.wg.Add(1)
		go w.worker(ctx, i)
	}

	// Long poll SQS loop
	w.pollQueue(ctx)

	// Wait for active workers to shut down gracefully on context cancellation
	close(w.jobs)
	w.wg.Wait()
	log.Println("Worker pool stopped gracefully.")
}

func (w *WorkerPool) worker(ctx context.Context, workerID int) {
	defer w.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-w.jobs:
			if !ok {
				return
			}
			w.processMessage(ctx, workerID, msg)
		}
	}
}

func (w *WorkerPool) pollQueue(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			output, err := w.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(w.queueURL),
				MaxNumberOfMessages: 10,
				WaitTimeSeconds:     20, // SQS Long Polling
			})
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("Error receiving SQS messages: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}

			for _, msg := range output.Messages {
				select {
				case <-ctx.Done():
					return
				case w.jobs <- msg:
				}
			}
		}
	}
}

func (w *WorkerPool) processMessage(ctx context.Context, workerID int, msg types.Message) {
	if msg.Body == nil {
		return
	}

	var job InferenceJob
	if err := json.Unmarshal([]byte(*msg.Body), &job); err != nil {
		log.Printf("[Worker %d] Malformed JSON payload: %v", workerID, err)
		return
	}

	// 1. Transition state to PROCESSING
	_ = w.dynamoStore.UpdateJobStatus(ctx, job.JobID, "PROCESSING", "", "")
	log.Printf("[Worker %d] Processing Job %s for User %s", workerID, job.JobID, job.UserID)

	// 2. Invoke Bedrock LLM (or mock for local testing)
	response, err := w.invokeBedrock(ctx, job.Prompt)
	if err != nil {
		log.Printf("[Worker %d] Bedrock execution failed for Job %s: %v", workerID, job.JobID, err)
		_ = w.dynamoStore.UpdateJobStatus(ctx, job.JobID, "FAILED", "", err.Error())
		return
	}

	// 3. Save output & update state to COMPLETED
	err = w.dynamoStore.UpdateJobStatus(ctx, job.JobID, "COMPLETED", response, "")
	if err != nil {
		log.Printf("[Worker %d] Failed to write status to DynamoDB for Job %s: %v", workerID, job.JobID, err)
		return
	}

	// 4. Delete message from SQS queue
	w.deleteMessage(ctx, msg.ReceiptHandle)
	log.Printf("[Worker %d] Job %s successfully processed and persisted!", workerID, job.JobID)
}

func (w *WorkerPool) invokeBedrock(ctx context.Context, prompt string) (string, error) {
	// Stub response for local testing; replace with actual Bedrock InvokeModel API call
	return "Bedrock response generated for: " + prompt, nil
}

func (w *WorkerPool) deleteMessage(ctx context.Context, receiptHandle *string) {
	_, _ = w.sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(w.queueURL),
		ReceiptHandle: receiptHandle,
	})
}
