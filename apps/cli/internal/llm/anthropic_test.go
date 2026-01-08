package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestAnthropicCompletion tests successful completion requests.
func TestAnthropicCompletion(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		opts     CompletionOptions
		response anthropicResponse
		wantText string
		wantErr  bool
	}{
		{
			name:   "basic completion",
			prompt: "Hello, world!",
			opts:   DefaultCompletionOptions(),
			response: anthropicResponse{
				ID:   "msg_123",
				Type: "message",
				Role: "assistant",
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{
					{Type: "text", Text: "Hello! How can I help you today?"},
				},
				StopReason: "end_turn",
			},
			wantText: "Hello! How can I help you today?",
			wantErr:  false,
		},
		{
			name:   "completion with system prompt",
			prompt: "What is 2+2?",
			opts: CompletionOptions{
				MaxTokens:    100,
				Temperature:  0.0,
				SystemPrompt: "You are a math teacher.",
			},
			response: anthropicResponse{
				ID:   "msg_456",
				Type: "message",
				Role: "assistant",
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{
					{Type: "text", Text: "The answer is 4."},
				},
				StopReason: "end_turn",
			},
			wantText: "The answer is 4.",
			wantErr:  false,
		},
		{
			name:   "multiple content blocks",
			prompt: "Explain something",
			opts:   DefaultCompletionOptions(),
			response: anthropicResponse{
				ID:   "msg_789",
				Type: "message",
				Role: "assistant",
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{
					{Type: "text", Text: "First part. "},
					{Type: "text", Text: "Second part."},
				},
				StopReason: "end_turn",
			},
			wantText: "First part. Second part.",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and path
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/v1/messages" {
					t.Errorf("Expected /v1/messages path, got %s", r.URL.Path)
				}

				// Verify headers
				if ct := r.Header.Get("Content-Type"); ct != "application/json" {
					t.Errorf("Expected Content-Type: application/json, got %s", ct)
				}
				if apiKey := r.Header.Get("x-api-key"); apiKey != "test-api-key" {
					t.Errorf("Expected x-api-key header, got %s", apiKey)
				}
				if version := r.Header.Get("anthropic-version"); version != anthropicVersion {
					t.Errorf("Expected anthropic-version: %s, got %s", anthropicVersion, version)
				}

				// Parse request body
				var req anthropicRequest
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("Failed to read request body: %v", err)
				}
				if err := json.Unmarshal(body, &req); err != nil {
					t.Fatalf("Failed to parse request: %v", err)
				}

				// Verify messages
				if len(req.Messages) != 1 {
					t.Errorf("Expected 1 message, got %d", len(req.Messages))
				}
				if req.Messages[0].Role != "user" {
					t.Errorf("Expected user role, got %s", req.Messages[0].Role)
				}
				if req.Messages[0].Content != tt.prompt {
					t.Errorf("Expected content %q, got %q", tt.prompt, req.Messages[0].Content)
				}

				// Send mock response
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client := NewAnthropicClient("test-api-key", "claude-3-haiku-20240307")
			client.client = &http.Client{
				Transport: &replaceURLTransport{
					base:   server.URL,
					target: anthropicBaseURL,
					rt:     http.DefaultTransport,
				},
			}

			ctx := context.Background()
			result, err := client.Complete(ctx, tt.prompt, tt.opts)

			if (err != nil) != tt.wantErr {
				t.Errorf("Complete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.wantText {
				t.Errorf("Complete() = %q, want %q", result, tt.wantText)
			}
		})
	}
}

// TestAnthropicCompletionJSON tests JSON completions.
func TestAnthropicCompletionJSON(t *testing.T) {
	type TestResult struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := anthropicResponse{
			ID:   "msg_json",
			Type: "message",
			Role: "assistant",
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				{Type: "text", Text: `{"name":"test","value":42}`},
			},
			StopReason: "end_turn",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewAnthropicClient("test-api-key", "claude-3-haiku-20240307")
	client.client = &http.Client{
		Transport: &replaceURLTransport{
			base:   server.URL,
			target: anthropicBaseURL,
			rt:     http.DefaultTransport,
		},
	}

	var result TestResult
	err := client.CompleteJSON(context.Background(), "Return JSON", DefaultCompletionOptions(), &result)

	if err != nil {
		t.Fatalf("CompleteJSON() failed: %v", err)
	}

	if result.Name != "test" || result.Value != 42 {
		t.Errorf("CompleteJSON() result = %+v, want {Name:test Value:42}", result)
	}
}

// TestAnthropicErrors tests error handling scenarios.
func TestAnthropicErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   anthropicResponse
		wantErrMsg string
	}{
		{
			name:       "invalid API key (401)",
			statusCode: http.StatusUnauthorized,
			response: anthropicResponse{
				Error: &struct {
					Type    string `json:"type"`
					Message string `json:"message"`
				}{
					Type:    "authentication_error",
					Message: "Invalid API key",
				},
			},
			wantErrMsg: "Invalid API key",
		},
		{
			name:       "rate limit exceeded (429)",
			statusCode: http.StatusTooManyRequests,
			response: anthropicResponse{
				Error: &struct {
					Type    string `json:"type"`
					Message string `json:"message"`
				}{
					Type:    "rate_limit_error",
					Message: "Rate limit exceeded",
				},
			},
			wantErrMsg: "Rate limit exceeded",
		},
		{
			name:       "server error (500)",
			statusCode: http.StatusInternalServerError,
			response: anthropicResponse{
				Error: &struct {
					Type    string `json:"type"`
					Message string `json:"message"`
				}{
					Type:    "api_error",
					Message: "Internal server error",
				},
			},
			wantErrMsg: "Internal server error",
		},
		{
			name:       "no content blocks",
			statusCode: http.StatusOK,
			response: anthropicResponse{
				ID:   "msg_no_content",
				Type: "message",
				Role: "assistant",
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{},
				StopReason: "end_turn",
			},
			wantErrMsg: "returned no content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client := NewAnthropicClient("test-api-key", "claude-3-haiku-20240307")
			client.client = &http.Client{
				Transport: &replaceURLTransport{
					base:   server.URL,
					target: anthropicBaseURL,
					rt:     http.DefaultTransport,
				},
			}

			_, err := client.Complete(context.Background(), "test", DefaultCompletionOptions())

			if err == nil {
				t.Error("Expected error, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("Error message %q does not contain %q", err.Error(), tt.wantErrMsg)
			}
		})
	}
}

// TestAnthropicMalformedJSON tests handling of malformed JSON responses.
func TestAnthropicMalformedJSON(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantErr  string
	}{
		{
			name:     "invalid json",
			response: `{invalid json}`,
			wantErr:  "parse response",
		},
		{
			name:     "truncated json",
			response: `{"id": "msg_123", "type": "message",`,
			wantErr:  "parse response",
		},
		{
			name:     "empty response",
			response: ``,
			wantErr:  "parse response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewAnthropicClient("test-api-key", "claude-3-haiku-20240307")
			client.client = &http.Client{
				Transport: &replaceURLTransport{
					base:   server.URL,
					target: anthropicBaseURL,
					rt:     http.DefaultTransport,
				},
			}

			_, err := client.Complete(context.Background(), "test", DefaultCompletionOptions())

			if err == nil {
				t.Error("Expected error, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestAnthropicTimeout tests timeout handling.
func TestAnthropicTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		json.NewEncoder(w).Encode(anthropicResponse{})
	}))
	defer server.Close()

	client := NewAnthropicClient("test-api-key", "claude-3-haiku-20240307")
	client.client = &http.Client{
		Timeout: 50 * time.Millisecond,
		Transport: &replaceURLTransport{
			base:   server.URL,
			target: anthropicBaseURL,
			rt:     http.DefaultTransport,
		},
	}

	_, err := client.Complete(context.Background(), "test", DefaultCompletionOptions())

	if err == nil {
		t.Error("Expected timeout error, got nil")
		return
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "timeout") && !strings.Contains(errMsg, "deadline") && !strings.Contains(errMsg, "Client.Timeout exceeded") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// TestAnthropicContextCancellation tests context cancellation handling.
func TestAnthropicContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		json.NewEncoder(w).Encode(anthropicResponse{})
	}))
	defer server.Close()

	client := NewAnthropicClient("test-api-key", "claude-3-haiku-20240307")
	client.client = &http.Client{
		Transport: &replaceURLTransport{
			base:   server.URL,
			target: anthropicBaseURL,
			rt:     http.DefaultTransport,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := client.Complete(ctx, "test", DefaultCompletionOptions())

	if err == nil {
		t.Error("Expected context cancellation error, got nil")
		return
	}

	if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "deadline") {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}
}

// TestAnthropicModel tests the Model method.
func TestAnthropicModel(t *testing.T) {
	client := NewAnthropicClient("test-api-key", "claude-3-opus-20240229")
	if model := client.Model(); model != "claude-3-opus-20240229" {
		t.Errorf("Model() = %q, want %q", model, "claude-3-opus-20240229")
	}
}

// TestAnthropicBackend tests the Backend method.
func TestAnthropicBackend(t *testing.T) {
	client := NewAnthropicClient("test-api-key", "claude-3-haiku-20240307")
	if backend := client.Backend(); backend != "anthropic" {
		t.Errorf("Backend() = %q, want %q", backend, "anthropic")
	}
}

// TestAnthropicNetworkError tests handling of network errors.
func TestAnthropicNetworkError(t *testing.T) {
	client := NewAnthropicClient("test-api-key", "claude-3-haiku-20240307")
	client.client = &http.Client{
		Timeout: 1 * time.Second,
		Transport: &replaceURLTransport{
			base:   "http://invalid-host-that-does-not-exist:99999",
			target: anthropicBaseURL,
			rt:     http.DefaultTransport,
		},
	}

	_, err := client.Complete(context.Background(), "test", DefaultCompletionOptions())

	if err == nil {
		t.Error("Expected network error, got nil")
		return
	}

	if !strings.Contains(err.Error(), "anthropic request failed") {
		t.Errorf("Expected 'anthropic request failed' in error, got: %v", err)
	}
}

// TestAnthropicRequestVerification verifies that request parameters are correctly set.
func TestAnthropicRequestVerification(t *testing.T) {
	var capturedRequest anthropicRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedRequest)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(anthropicResponse{
			ID:   "msg_verify",
			Type: "message",
			Role: "assistant",
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				{Type: "text", Text: "test response"},
			},
			StopReason: "end_turn",
		})
	}))
	defer server.Close()

	client := NewAnthropicClient("test-api-key-123", "claude-3-opus-20240229")
	client.client = &http.Client{
		Transport: &replaceURLTransport{
			base:   server.URL,
			target: anthropicBaseURL,
			rt:     http.DefaultTransport,
		},
	}

	opts := CompletionOptions{
		MaxTokens:    3000,
		Temperature:  0.5,
		SystemPrompt: "You are a helpful assistant",
	}

	_, err := client.Complete(context.Background(), "test prompt", opts)
	if err != nil {
		t.Fatalf("Complete() failed: %v", err)
	}

	// Verify captured request
	if capturedRequest.Model != "claude-3-opus-20240229" {
		t.Errorf("Model = %q, want %q", capturedRequest.Model, "claude-3-opus-20240229")
	}
	if capturedRequest.MaxTokens != opts.MaxTokens {
		t.Errorf("MaxTokens = %d, want %d", capturedRequest.MaxTokens, opts.MaxTokens)
	}
	if capturedRequest.System != opts.SystemPrompt {
		t.Errorf("System = %q, want %q", capturedRequest.System, opts.SystemPrompt)
	}

	// Verify messages
	if len(capturedRequest.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(capturedRequest.Messages))
	}
	if capturedRequest.Messages[0].Role != "user" {
		t.Errorf("Message role = %q, want user", capturedRequest.Messages[0].Role)
	}
	if capturedRequest.Messages[0].Content != "test prompt" {
		t.Errorf("Message content = %q, want %q", capturedRequest.Messages[0].Content, "test prompt")
	}
}

// TestAnthropicMultipleModels tests different model configurations.
func TestAnthropicMultipleModels(t *testing.T) {
	models := []string{
		"claude-3-haiku-20240307",
		"claude-3-sonnet-20240229",
		"claude-3-opus-20240229",
	}

	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			var capturedModel string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req anthropicRequest
				json.NewDecoder(r.Body).Decode(&req)
				capturedModel = req.Model

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(anthropicResponse{
					ID:   "msg_model_test",
					Type: "message",
					Role: "assistant",
					Content: []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					}{
						{Type: "text", Text: "response"},
					},
					StopReason: "end_turn",
				})
			}))
			defer server.Close()

			client := NewAnthropicClient("test-api-key", model)
			client.client = &http.Client{
				Transport: &replaceURLTransport{
					base:   server.URL,
					target: anthropicBaseURL,
					rt:     http.DefaultTransport,
				},
			}

			_, err := client.Complete(context.Background(), "test", DefaultCompletionOptions())
			if err != nil {
				t.Fatalf("Complete() failed: %v", err)
			}

			if capturedModel != model {
				t.Errorf("Request model = %q, want %q", capturedModel, model)
			}

			if client.Model() != model {
				t.Errorf("Client.Model() = %q, want %q", client.Model(), model)
			}
		})
	}
}

// TestAnthropicMaxTokensDefault tests that MaxTokens defaults to 2048 when not set.
func TestAnthropicMaxTokensDefault(t *testing.T) {
	var capturedMaxTokens int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req anthropicRequest
		json.NewDecoder(r.Body).Decode(&req)
		capturedMaxTokens = req.MaxTokens

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(anthropicResponse{
			ID:   "msg_default",
			Type: "message",
			Role: "assistant",
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				{Type: "text", Text: "response"},
			},
			StopReason: "end_turn",
		})
	}))
	defer server.Close()

	client := NewAnthropicClient("test-api-key", "claude-3-haiku-20240307")
	client.client = &http.Client{
		Transport: &replaceURLTransport{
			base:   server.URL,
			target: anthropicBaseURL,
			rt:     http.DefaultTransport,
		},
	}

	// Use options with MaxTokens = 0 (should default to 2048)
	opts := CompletionOptions{
		MaxTokens: 0,
	}

	_, err := client.Complete(context.Background(), "test", opts)
	if err != nil {
		t.Fatalf("Complete() failed: %v", err)
	}

	if capturedMaxTokens != 2048 {
		t.Errorf("MaxTokens = %d, want 2048 (default)", capturedMaxTokens)
	}
}
