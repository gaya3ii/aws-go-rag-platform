package bedrock

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

type Client struct {
	runtime *bedrockruntime.Client
	modelID string
}

func NewClient(runtime *bedrockruntime.Client, modelID string) *Client {
	return &Client{
		runtime: runtime,
		modelID: modelID,
	}
}

type NovaRequest struct {
	SchemaVersion   string `json:"schemaVersion,omitempty"`
	InferenceConfig struct {
		MaxTokens int `json:"maxTokens"`
	} `json:"inferenceConfig"`
	Messages []NovaMessage `json:"messages"`
}

type NovaMessage struct {
	Role    string        `json:"role"`
	Content []NovaContent `json:"content"`
}

type NovaContent struct {
	Text string `json:"text"`
}

type NovaResponse struct {
	Output struct {
		Message struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"message"`
	} `json:"output"`
}

func (c *Client) GenerateCompletion(ctx context.Context, prompt string) (string, error) {
	var reqPayload NovaRequest
	reqPayload.SchemaVersion = "messages-v1"
	reqPayload.InferenceConfig.MaxTokens = 300
	reqPayload.Messages = []NovaMessage{
		{
			Role: "user",
			Content: []NovaContent{
				{Text: prompt},
			},
		},
	}

	payloadBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	output, err := c.runtime.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(c.modelID),
		ContentType: aws.String("application/json"),
		Body:        payloadBytes,
	})
	if err != nil {
		return "", fmt.Errorf("bedrock invocation failed: %w", err)
	}

	var resp NovaResponse
	if err := json.Unmarshal(output.Body, &resp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(resp.Output.Message.Content) == 0 {
		return "", fmt.Errorf("empty response from bedrock")
	}

	return resp.Output.Message.Content[0].Text, nil
}
