package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const openAIBaseURL = "https://api.openai.com/v1"

// OpenAIClient implements Client using the OpenAI API.
type OpenAIClient struct {
	apiKey string
	model  string
	client *http.Client
}

// NewOpenAIClient creates a new OpenAI client.
func NewOpenAIClient(apiKey, model string) *OpenAIClient {
	return &OpenAIClient{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// openAIRequest is the request body for OpenAI's chat completions endpoint.
type openAIRequest struct {
	Model          string          `json:"model"`
	Messages       []openAIMessage `json:"messages"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	Temperature    float64         `json:"temperature,omitempty"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

// openAIResponse is the response from OpenAI's chat completions endpoint.
type openAIResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// Complete generates a text completion using OpenAI.
func (c *OpenAIClient) Complete(ctx context.Context, prompt string, opts CompletionOptions) (string, error) {
	messages := make([]openAIMessage, 0, 2)

	if opts.SystemPrompt != "" {
		messages = append(messages, openAIMessage{
			Role:    "system",
			Content: opts.SystemPrompt,
		})
	}

	messages = append(messages, openAIMessage{
		Role:    "user",
		Content: prompt,
	})

	reqBody := openAIRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   opts.MaxTokens,
		Temperature: opts.Temperature,
	}

	if opts.JSONMode {
		reqBody.ResponseFormat = &responseFormat{Type: "json_object"}
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openAIBaseURL+"/chat/completions", bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", fmt.Errorf("parse response: %w (body: %s)", err, truncate(string(body), 200))
	}

	if openAIResp.Error != nil {
		return "", fmt.Errorf("openai error: %s (%s)", openAIResp.Error.Message, openAIResp.Error.Type)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

// CompleteJSON generates a completion and parses the result as JSON.
func (c *OpenAIClient) CompleteJSON(ctx context.Context, prompt string, opts CompletionOptions, result interface{}) error {
	opts.JSONMode = true
	response, err := c.Complete(ctx, prompt, opts)
	if err != nil {
		return err
	}
	return parseJSONResponse(response, result)
}

// Model returns the model identifier.
func (c *OpenAIClient) Model() string {
	return c.model
}

// Backend returns "openai".
func (c *OpenAIClient) Backend() string {
	return "openai"
}
