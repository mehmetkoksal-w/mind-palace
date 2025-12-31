// Package llm provides interfaces and implementations for LLM text generation.
// This is used for conversation extraction and contradiction detection features.
package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// ErrNotConfigured is returned when LLM backend is not configured.
var ErrNotConfigured = errors.New("llm: backend not configured")

// ErrUnsupportedBackend is returned when an unknown backend is specified.
var ErrUnsupportedBackend = errors.New("llm: unsupported backend")

// Client defines the interface for LLM text generation.
type Client interface {
	// Complete generates a text completion for the given prompt.
	Complete(ctx context.Context, prompt string, opts CompletionOptions) (string, error)

	// CompleteJSON generates a completion and parses the result as JSON into result.
	CompleteJSON(ctx context.Context, prompt string, opts CompletionOptions, result interface{}) error

	// Model returns the model identifier being used.
	Model() string

	// Backend returns the backend type (e.g., "ollama", "openai", "anthropic").
	Backend() string
}

// CompletionOptions configures completion behavior.
type CompletionOptions struct {
	// MaxTokens is the maximum number of tokens to generate.
	// Default: 2048 for most backends.
	MaxTokens int

	// Temperature controls randomness. 0.0 is deterministic, higher is more random.
	// Default: 0.3 for extraction/analysis tasks.
	Temperature float64

	// SystemPrompt is an optional system-level instruction.
	SystemPrompt string

	// JSONMode requests JSON output from supported backends.
	JSONMode bool
}

// DefaultCompletionOptions returns sensible defaults for extraction tasks.
func DefaultCompletionOptions() CompletionOptions {
	return CompletionOptions{
		MaxTokens:   2048,
		Temperature: 0.3,
		JSONMode:    false,
	}
}

// Config holds LLM provider configuration.
type Config struct {
	// Backend is the LLM provider: "ollama", "openai", "anthropic", or "disabled".
	Backend string

	// Model is the model identifier, e.g., "llama3.2", "gpt-4o-mini", "claude-3-haiku-20240307".
	Model string

	// URL is the base URL for the API (primarily for Ollama).
	// Default for Ollama: "http://localhost:11434"
	URL string

	// APIKey is the API key for cloud providers (OpenAI, Anthropic).
	// Falls back to environment variables if not set.
	APIKey string
}

// DefaultConfig returns the default configuration using Ollama.
func DefaultConfig() Config {
	return Config{
		Backend: "ollama",
		Model:   "llama3.2",
		URL:     "http://localhost:11434",
	}
}

// NewClient creates an LLM client based on the configuration.
func NewClient(cfg Config) (Client, error) {
	switch cfg.Backend {
	case "", "disabled":
		return nil, ErrNotConfigured

	case "ollama":
		url := cfg.URL
		if url == "" {
			url = "http://localhost:11434"
		}
		model := cfg.Model
		if model == "" {
			model = "llama3.2"
		}
		return NewOllamaClient(url, model), nil

	case "openai":
		apiKey := cfg.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("llm: OpenAI API key required (set llmApiKey or OPENAI_API_KEY)")
		}
		model := cfg.Model
		if model == "" {
			model = "gpt-4o-mini"
		}
		return NewOpenAIClient(apiKey, model), nil

	case "anthropic":
		apiKey := cfg.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("llm: Anthropic API key required (set llmApiKey or ANTHROPIC_API_KEY)")
		}
		model := cfg.Model
		if model == "" {
			model = "claude-3-haiku-20240307"
		}
		return NewAnthropicClient(apiKey, model), nil

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedBackend, cfg.Backend)
	}
}

// extractJSON attempts to extract JSON from a response that may contain markdown code blocks.
func extractJSON(response string) string {
	// Try to find JSON in code blocks first
	if start := findJSONStart(response); start >= 0 {
		if end := findJSONEnd(response, start); end > start {
			return response[start:end]
		}
	}
	// Return as-is if no code blocks found
	return response
}

func findJSONStart(s string) int {
	// Look for ```json or ``` followed by {
	patterns := []string{"```json\n", "```json\r\n", "```\n{", "```\r\n{"}
	for _, p := range patterns {
		if idx := indexOf(s, p); idx >= 0 {
			if p == "```json\n" || p == "```json\r\n" {
				return idx + len(p)
			}
			// For ```\n{ patterns, include the {
			return idx + len(p) - 1
		}
	}
	// No code block, look for first {
	for i, c := range s {
		if c == '{' || c == '[' {
			return i
		}
	}
	return -1
}

func findJSONEnd(s string, start int) int {
	// Count braces/brackets to find matching end
	depth := 0
	inString := false
	escape := false

	for i := start; i < len(s); i++ {
		c := s[i]

		if escape {
			escape = false
			continue
		}

		if c == '\\' && inString {
			escape = true
			continue
		}

		if c == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch c {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}

	return len(s)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// parseJSONResponse extracts and parses JSON from an LLM response.
func parseJSONResponse(response string, result interface{}) error {
	jsonStr := extractJSON(response)
	if err := json.Unmarshal([]byte(jsonStr), result); err != nil {
		return fmt.Errorf("parse JSON response: %w (raw: %s)", err, truncate(response, 200))
	}
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
