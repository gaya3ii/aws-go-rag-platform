package store

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type JobRecord struct {
	JobID     string `dynamodbav:"job_id" json:"job_id"`
	UserID    string `dynamodbav:"user_id" json:"user_id"`
	Status    string `dynamodbav:"status" json:"status"`
	Prompt    string `dynamodbav:"prompt" json:"prompt"`
	Response  string `dynamodbav:"response,omitempty" json:"response,omitempty"`
	Error     string `dynamodbav:"error,omitempty" json:"error,omitempty"`
	CreatedAt string `dynamodbav:"created_at" json:"created_at"`
	UpdatedAt string `dynamodbav:"updated_at" json:"updated_at"`
}

type DynamoStore struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamoStore(client *dynamodb.Client, tableName string) *DynamoStore {
	return &DynamoStore{
		client:    client,
		tableName: tableName,
	}
}

func (s *DynamoStore) CreateJob(ctx context.Context, job JobRecord) error {
	item, err := attributevalue.MarshalMap(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job item: %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	return err
}

func (s *DynamoStore) GetJob(ctx context.Context, jobID string) (*JobRecord, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"job_id": &types.AttributeValueMemberS{Value: jobID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get item from DynamoDB: %w", err)
	}
	if result.Item == nil {
		return nil, nil
	}

	var record JobRecord
	if err := attributevalue.UnmarshalMap(result.Item, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job item: %w", err)
	}

	return &record, nil
}

func (s *DynamoStore) UpdateJobStatus(ctx context.Context, jobID, status, response, errMsg string) error {
	now := time.Now().Format(time.RFC3339)

	updateExpr := "SET #s = :status, updated_at = :updated_at"
	exprNames := map[string]string{
		"#s": "status",
	}
	exprValues := map[string]types.AttributeValue{
		":status":     &types.AttributeValueMemberS{Value: status},
		":updated_at": &types.AttributeValueMemberS{Value: now},
	}

	// Alias #resp to prevent DynamoDB "reserved keyword: response" error
	if response != "" {
		updateExpr += ", #resp = :response"
		exprNames["#resp"] = "response"
		exprValues[":response"] = &types.AttributeValueMemberS{Value: response}
	}

	if errMsg != "" {
		updateExpr += ", #err = :error"
		exprNames["#err"] = "error"
		exprValues[":error"] = &types.AttributeValueMemberS{Value: errMsg}
	}

	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"job_id": &types.AttributeValueMemberS{Value: jobID},
		},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprNames,
		ExpressionAttributeValues: exprValues,
	})
	return err
}
