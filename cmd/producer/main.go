package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type InferenceJob struct {
	JobID  string `json:"job_id"`
	UserID string `json:"user_id"`
	Prompt string `json:"prompt"`
}

func main() {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		log.Fatalf("failed to load SDK config: %v", err)
	}

	sqsClient := sqs.NewFromConfig(cfg)
	queueURL := os.Getenv("SQS_QUEUE_URL")
	if queueURL == "" {
		log.Fatal("SQS_QUEUE_URL environment variable is required")
	}

	// 1. Construct the job struct
	jobID := fmt.Sprintf("job-%d", time.Now().Unix())
	job := InferenceJob{
		JobID:  jobID,
		UserID: "usr-101",
		Prompt: "Explain how Amazon Bedrock Nova processes asynchronous prompt requests in 2 concise sentences.",
	}

	// 2. Marshal to structured JSON
	bodyBytes, err := json.Marshal(job)
	if err != nil {
		log.Fatalf("failed to marshal job payload: %v", err)
	}

	groupID := "inference-group"
	dedupID := fmt.Sprintf("%s-%d", jobID, time.Now().UnixNano())

	// 3. Publish JSON string to SQS FIFO
	log.Printf("Publishing JSON job %s to SQS...", jobID)
	out, err := sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:               aws.String(queueURL),
		MessageBody:            aws.String(string(bodyBytes)),
		MessageGroupId:         aws.String(groupID),
		MessageDeduplicationId: aws.String(dedupID),
	})
	if err != nil {
		log.Fatalf("failed to publish message to SQS: %v", err)
	}

	log.Printf("Successfully published job %s! MessageId: %s", jobID, *out.MessageId)
}
