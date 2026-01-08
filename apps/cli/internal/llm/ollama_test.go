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

// TestOllamaCompletion tests successful completion requests.
func TestOllamaCompletion(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		opts     CompletionOptions
		response ollamaResponse
		wantText string
		wantErr  bool
	}{
		{
			name:   "basic completion",
			prompt: "Hello, world!",
			opts:   DefaultCompletionOptions(),
			response: ollamaResponse{
				Model:    "llama3.2",
				Response: "Hello! How can I help you today?",
				Done:     true,
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
			response: ollamaResponse{
				Model:    "llama3.2",
				Response: "The answer is 4.",
				Done:     true,
			},
			wantText: "The answer is 4.",
			wantErr:  false,
		},
		{
			name:   "completion with high temperature",
			prompt: "Tell me a story",
			opts: CompletionOptions{
				MaxTokens:   500,
				Temperature: 0.9,
			},
			response: ollamaResponse{
				Model:    "llama3.2",
				Response: "Once upon a time...",
				Done:     true,
			},
			wantText: "Once upon a time...",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and path
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/api/generate" {
					t.Errorf("Expected /api/generate path, got %s", r.URL.Path)
				}

				// Verify content type
				if ct := r.Header.Get("Content-Type"); ct != "application/json" {
					t.Errorf("Expected Content-Type: application/json, got %s", ct)
				}

				// Parse request body
				var req ollamaRequest
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("Failed to read request body: %v", err)
				}
				if err := json.Unmarshal(body, &req); err != nil {
					t.Fatalf("Failed to parse request: %v", err)
				}

				// Verify request fields
				if req.Prompt != tt.prompt {
					t.Errorf("Expected prompt %q, got %q", tt.prompt, req.Prompt)
				}
				if req.System != tt.opts.SystemPrompt {
					t.Errorf("Expected system %q, got %q", tt.opts.SystemPrompt, req.System)
				}
				if req.Stream != false {
					t.Errorf("Expected stream=false, got %v", req.Stream)
				}

				// Send mock response
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			// Create client with mock server URL
			client := NewOllamaClient(server.URL, "llama3.2")

			// Execute completion
			ctx := context.Background()
			result, err := client.Complete(ctx, tt.prompt, tt.opts)

			// Verify result
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

// TestOllamaCompletionJSON tests JSON mode completions.
func TestOllamaCompletionJSON(t *testing.T) {
	type TestResult struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name         string
		prompt       string
		response     string
		wantResult   TestResult
		wantErr      bool
		wantParseErr bool
	}{
		{
			name:   "valid json response",
			prompt: "Return a JSON object",
			response: `{
				"model": "llama3.2",
				"response": "{\"name\":\"test\",\"value\":42}",
				"done": true
			}`,
			wantResult: TestResult{Name: "test", Value: 42},
			wantErr:    false,
		},
		{
			name:   "json with markdown code block",
			prompt: "Return a JSON object",
			response: `{
				"model": "llama3.2",
				"response": "` + "```json\\n{\\\"name\\\":\\\"example\\\",\\\"value\\\":100}\\n```" + `",
				"done": true
			}`,
			wantResult: TestResult{Name: "example", Value: 100},
			wantErr:    false,
		},
		{
			name:   "invalid json response",
			prompt: "Return a JSON object",
			response: `{
				"model": "llama3.2",
				"response": "This is not JSON",
				"done": true
			}`,
			wantParseErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify JSON format was requested
				var req ollamaRequest
				json.NewDecoder(r.Body).Decode(&req)
				if req.Format != "json" {
					t.Errorf("Expected format=json, got %q", req.Format)
				}

				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewOllamaClient(server.URL, "llama3.2")

			var result TestResult
			err := client.CompleteJSON(context.Background(), tt.prompt, DefaultCompletionOptions(), &result)

			if tt.wantParseErr {
				if err == nil {
					t.Error("Expected parse error, got nil")
				}
				return
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("CompleteJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.wantResult {
				t.Errorf("CompleteJSON() result = %+v, want %+v", result, tt.wantResult)
			}
		})
	}
}

// TestOllamaErrors tests error handling scenarios.
func TestOllamaErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		wantErrMsg string
	}{
		{
			name:       "404 not found",
			statusCode: http.StatusNotFound,
			response:   `{"error": "model not found"}`,
			wantErrMsg: "ollama returned status 404",
		},
		{
			name:       "500 internal server error",
			statusCode: http.StatusInternalServerError,
			response:   `{"error": "internal server error"}`,
			wantErrMsg: "ollama returned status 500",
		},
		{
			name:       "503 service unavailable",
			statusCode: http.StatusServiceUnavailable,
			response:   `{"error": "service unavailable"}`,
			wantErrMsg: "ollama returned status 503",
		},
		{
			name:       "ollama error field",
			statusCode: http.StatusOK,
			response:   `{"model": "llama3.2", "error": "model loading failed", "done": true}`,
			wantErrMsg: "ollama error: model loading failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewOllamaClient(server.URL, "llama3.2")
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

// TestOllamaMalformedJSON tests handling of malformed JSON responses.
func TestOllamaMalformedJSON(t *testing.T) {
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
			response: `{"model": "llama3.2", "response":`,
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

			client := NewOllamaClient(server.URL, "llama3.2")
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

// TestOllamaTimeout tests timeout handling.
func TestOllamaTimeout(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		json.NewEncoder(w).Encode(ollamaResponse{
			Model:    "llama3.2",
			Response: "Delayed response",
			Done:     true,
		})
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	// Override timeout for testing
	client.client.Timeout = 50 * time.Millisecond

	ctx := context.Background()
	_, err := client.Complete(ctx, "test", DefaultCompletionOptions())

	if err == nil {
		t.Error("Expected timeout error, got nil")
		return
	}

	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// TestOllamaContextCancellation tests context cancellation handling.
func TestOllamaContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay to allow context cancellation
		time.Sleep(100 * time.Millisecond)
		json.NewEncoder(w).Encode(ollamaResponse{
			Model:    "llama3.2",
			Response: "Should not receive this",
			Done:     true,
		})
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")

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

// TestOllamaPing tests the Ping method.
func TestOllamaPing(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful ping",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/tags" {
					t.Errorf("Expected /api/tags, got %s", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"models": []map[string]string{},
					})
				}
			}))
			defer server.Close()

			client := NewOllamaClient(server.URL, "llama3.2")
			err := client.Ping(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("Ping() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestOllamaListModels tests the ListModels method.
func TestOllamaListModels(t *testing.T) {
	tests := []struct {
		name       string
		response   interface{}
		wantModels []string
		wantErr    bool
	}{
		{
			name: "multiple models",
			response: map[string]interface{}{
				"models": []map[string]string{
					{"name": "llama3.2"},
					{"name": "mistral"},
					{"name": "codellama"},
				},
			},
			wantModels: []string{"llama3.2", "mistral", "codellama"},
			wantErr:    false,
		},
		{
			name: "no models",
			response: map[string]interface{}{
				"models": []map[string]string{},
			},
			wantModels: nil,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client := NewOllamaClient(server.URL, "llama3.2")
			models, err := client.ListModels(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("ListModels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(models) != len(tt.wantModels) {
				t.Errorf("ListModels() returned %d models, want %d", len(models), len(tt.wantModels))
				return
			}

			for i, model := range models {
				if model != tt.wantModels[i] {
					t.Errorf("Model[%d] = %q, want %q", i, model, tt.wantModels[i])
				}
			}
		})
	}
}

// TestOllamaModel tests the Model method.
func TestOllamaModel(t *testing.T) {
	client := NewOllamaClient("http://localhost:11434", "llama3.2")
	if model := client.Model(); model != "llama3.2" {
		t.Errorf("Model() = %q, want %q", model, "llama3.2")
	}
}

// TestOllamaBackend tests the Backend method.
func TestOllamaBackend(t *testing.T) {
	client := NewOllamaClient("http://localhost:11434", "llama3.2")
	if backend := client.Backend(); backend != "ollama" {
		t.Errorf("Backend() = %q, want %q", backend, "ollama")
	}
}

// TestOllamaNetworkError tests handling of network errors.
func TestOllamaNetworkError(t *testing.T) {
	// Use an invalid URL to trigger network error
	client := NewOllamaClient("http://invalid-host-that-does-not-exist:99999", "llama3.2")
	client.client.Timeout = 1 * time.Second

	_, err := client.Complete(context.Background(), "test", DefaultCompletionOptions())

	if err == nil {
		t.Error("Expected network error, got nil")
		return
	}

	if !strings.Contains(err.Error(), "ollama request failed") {
		t.Errorf("Expected 'ollama request failed' in error, got: %v", err)
	}
}

// TestOllamaRequestVerification verifies that request parameters are correctly set.
func TestOllamaRequestVerification(t *testing.T) {
	var capturedRequest ollamaRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedRequest)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ollamaResponse{
			Model:    "llama3.2",
			Response: "test response",
			Done:     true,
		})
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "custom-model")
	opts := CompletionOptions{
		MaxTokens:    1000,
		Temperature:  0.7,
		SystemPrompt: "You are a helpful assistant",
		JSONMode:     true,
	}

	_, err := client.Complete(context.Background(), "test prompt", opts)
	if err != nil {
		t.Fatalf("Complete() failed: %v", err)
	}

	// Verify captured request
	if capturedRequest.Model != "custom-model" {
		t.Errorf("Model = %q, want %q", capturedRequest.Model, "custom-model")
	}
	if capturedRequest.Prompt != "test prompt" {
		t.Errorf("Prompt = %q, want %q", capturedRequest.Prompt, "test prompt")
	}
	if capturedRequest.System != opts.SystemPrompt {
		t.Errorf("System = %q, want %q", capturedRequest.System, opts.SystemPrompt)
	}
	if capturedRequest.Stream != false {
		t.Errorf("Stream = %v, want false", capturedRequest.Stream)
	}
	if capturedRequest.Format != "json" {
		t.Errorf("Format = %q, want %q", capturedRequest.Format, "json")
	}
	if capturedRequest.Options.NumPredict != opts.MaxTokens {
		t.Errorf("NumPredict = %d, want %d", capturedRequest.Options.NumPredict, opts.MaxTokens)
	}
	if capturedRequest.Options.Temperature != opts.Temperature {
		t.Errorf("Temperature = %f, want %f", capturedRequest.Options.Temperature, opts.Temperature)
	}
}
