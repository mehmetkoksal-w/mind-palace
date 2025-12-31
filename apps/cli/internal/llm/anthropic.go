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

const anthropicBaseURL = "https://api.anthropic.com/v1"
const anthropicVersion = "2023-06-01"

// AnthropicClient implements Client using the Anthropic API.
type AnthropicClient struct {
	apiKey string
	model  string
	client *http.Client
}

// NewAnthropicClient creates a new Anthropic client.
func NewAnthropicClient(apiKey, model string) *AnthropicClient {
	return &AnthropicClient{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// anthropicRequest is the request body for Anthropic's messages endpoint.
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse is the response from Anthropic's messages endpoint.
type anthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Error      *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Complete generates a text completion using Anthropic.
func (c *AnthropicClient) Complete(ctx context.Context, prompt string, opts CompletionOptions) (string, error) {
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2048
	}

	reqBody := anthropicRequest{
		Model:     c.model,
		MaxTokens: maxTokens,
		System:    opts.SystemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", anthropicBaseURL+"/messages", bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return "", fmt.Errorf("parse response: %w (body: %s)", err, truncate(string(body), 200))
	}

	if anthropicResp.Error != nil {
		return "", fmt.Errorf("anthropic error: %s (%s)", anthropicResp.Error.Message, anthropicResp.Error.Type)
	}

	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("anthropic returned no content")
	}

	// Concatenate all text blocks
	var result string
	for _, block := range anthropicResp.Content {
		if block.Type == "text" {
			result += block.Text
		}
	}

	return result, nil
}

// CompleteJSON generates a completion and parses the result as JSON.
func (c *AnthropicClient) CompleteJSON(ctx context.Context, prompt string, opts CompletionOptions, result interface{}) error {
	// Anthropic doesn't have native JSON mode, so we instruct in the prompt
	jsonPrompt := prompt + "\n\nRespond with valid JSON only, no additional text or markdown."
	response, err := c.Complete(ctx, jsonPrompt, opts)
	if err != nil {
		return err
	}
	return parseJSONResponse(response, result)
}

// Model returns the model identifier.
func (c *AnthropicClient) Model() string {
	return c.model
}

// Backend returns "anthropic".
func (c *AnthropicClient) Backend() string {
	return "anthropic"
}
