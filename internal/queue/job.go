package queue

import (
	"encoding/json"
	"fmt"
)

// InferenceJob represents the structured JSON payload sent via SQS.
type InferenceJob struct {
	JobID       string  `json:"job_id"`
	UserID      string  `json:"user_id"`
	Prompt      string  `json:"prompt"`
	ModelID     string  `json:"model_id,omitempty"`
	Temperature float32 `json:"temperature,omitempty"`
	MaxTokens   int32   `json:"max_tokens,omitempty"`
}

// Marshal converts the InferenceJob struct to a JSON string.
func (j *InferenceJob) Marshal() (string, error) {
	bytes, err := json.Marshal(j)
	if err != nil {
		return "", fmt.Errorf("failed to marshal inference job: %w", err)
	}
	return string(bytes), nil
}

// UnmarshalJob parses a raw JSON string from SQS into an InferenceJob.
func UnmarshalJob(data string) (*InferenceJob, error) {
	var job InferenceJob
	err := json.Unmarshal([]byte(data), &job)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal inference job: %w", err)
	}
	return &job, nil
}
