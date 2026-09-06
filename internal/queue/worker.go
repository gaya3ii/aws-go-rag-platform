package queue

import (
	"context"
	"log"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/gaya3ii/aws-go-rag-platform/internal/bedrock"
)

type WorkerPool struct {
	sqsClient     *sqs.Client
	bedrockClient *bedrock.Client
	queueURL      string
	concurrency   int
}

func NewWorkerPool(sqsClient *sqs.Client, bedrockClient *bedrock.Client, queueURL string, concurrency int) *WorkerPool {
	return &WorkerPool{
		sqsClient:     sqsClient,
		bedrockClient: bedrockClient,
		queueURL:      queueURL,
		concurrency:   concurrency,
	}
}

func (w *WorkerPool) Start(ctx context.Context) {
	jobs := make(chan types.Message, w.concurrency*2)
	var wg sync.WaitGroup

	// Start concurrent worker goroutines
	for i := 1; i <= w.concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for msg := range jobs {
				w.processMessage(ctx, workerID, msg)
			}
		}(i)
	}

	log.Printf("Worker pool started with %d concurrent workers...", w.concurrency)

	// Long-polling SQS loop
	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down worker pool...")
			close(jobs)
			wg.Wait()
			return
		default:
			output, err := w.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(w.queueURL),
				MaxNumberOfMessages: 10,
				WaitTimeSeconds:     20, // Long Polling
			})
			if err != nil {
				// Prevent tight error-looping on temporary context cancellation
				if ctx.Err() != nil {
					continue
				}
				log.Printf("Error receiving SQS messages: %v", err)
				continue
			}

			for _, msg := range output.Messages {
				jobs <- msg
			}
		}
	}
}

func (w *WorkerPool) processMessage(ctx context.Context, workerID int, msg types.Message) {
	log.Printf("[Worker %d] Processing job %s with body: %s", workerID, *msg.MessageId, *msg.Body)

	// Invoke Bedrock
	response, err := w.bedrockClient.GenerateCompletion(ctx, *msg.Body)
	if err != nil {
		log.Printf("[Worker %d] Failed to process message %s: %v", workerID, *msg.MessageId, err)
		return
	}

	log.Printf("[Worker %d] Job Complete!\n--- Output ---\n%s\n--------------", workerID, response)

	// Delete message from SQS upon success
	_, err = w.sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(w.queueURL),
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		log.Printf("[Worker %d] Failed to delete message %s from SQS: %v", workerID, *msg.MessageId, err)
	}
}
