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

// TestOpenAICompletion tests successful completion requests.
func TestOpenAICompletion(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		opts     CompletionOptions
		response openAIResponse
		wantText string
		wantErr  bool
	}{
		{
			name:   "basic completion",
			prompt: "Hello, world!",
			opts:   DefaultCompletionOptions(),
			response: openAIResponse{
				ID: "chatcmpl-123",
				Choices: []struct {
					Message struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					} `json:"message"`
					FinishReason string `json:"finish_reason"`
				}{
					{
						Message: struct {
							Role    string `json:"role"`
							Content string `json:"content"`
						}{
							Role:    "assistant",
							Content: "Hello! How can I help you today?",
						},
						FinishReason: "stop",
					},
				},
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
			response: openAIResponse{
				ID: "chatcmpl-456",
				Choices: []struct {
					Message struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					} `json:"message"`
					FinishReason string `json:"finish_reason"`
				}{
					{
						Message: struct {
							Role    string `json:"role"`
							Content string `json:"content"`
						}{
							Role:    "assistant",
							Content: "The answer is 4.",
						},
						FinishReason: "stop",
					},
				},
			},
			wantText: "The answer is 4.",
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
				if r.URL.Path != "/v1/chat/completions" {
					t.Errorf("Expected /v1/chat/completions path, got %s", r.URL.Path)
				}

				// Verify headers
				if ct := r.Header.Get("Content-Type"); ct != "application/json" {
					t.Errorf("Expected Content-Type: application/json, got %s", ct)
				}
				if auth := r.Header.Get("Authorization"); !strings.HasPrefix(auth, "Bearer ") {
					t.Errorf("Expected Authorization header with Bearer token, got %s", auth)
				}

				// Parse request body
				var req openAIRequest
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("Failed to read request body: %v", err)
				}
				if err := json.Unmarshal(body, &req); err != nil {
					t.Fatalf("Failed to parse request: %v", err)
				}

				// Verify messages
				expectedMsgCount := 1
				if tt.opts.SystemPrompt != "" {
					expectedMsgCount = 2
				}
				if len(req.Messages) != expectedMsgCount {
					t.Errorf("Expected %d messages, got %d", expectedMsgCount, len(req.Messages))
				}

				// Verify user message
				userMsg := req.Messages[len(req.Messages)-1]
				if userMsg.Role != "user" {
					t.Errorf("Expected user role, got %s", userMsg.Role)
				}
				if userMsg.Content != tt.prompt {
					t.Errorf("Expected content %q, got %q", tt.prompt, userMsg.Content)
				}

				// Send mock response
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			// Create client pointing to mock server
			client := NewOpenAIClient("test-api-key", "gpt-4o-mini")

			// Replace the client's http.Client to redirect requests to our test server
			// (openAIBaseURL is a package constant, so we use transport replacement)
			client.client = &http.Client{
				Transport: &replaceURLTransport{
					base:   server.URL,
					target: openAIBaseURL,
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

// replaceURLTransport is a custom RoundTripper that replaces the target URL with base URL.
type replaceURLTransport struct {
	base   string
	target string
	rt     http.RoundTripper
}

func (t *replaceURLTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request
	newReq := req.Clone(req.Context())
	// Replace the URL
	newReq.URL.Scheme = "http"
	newReq.URL.Host = strings.TrimPrefix(t.base, "http://")
	return t.rt.RoundTrip(newReq)
}

// TestOpenAICompletionJSON tests JSON mode completions.
func TestOpenAICompletionJSON(t *testing.T) {
	type TestResult struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify JSON mode was requested
		var req openAIRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.ResponseFormat == nil || req.ResponseFormat.Type != "json_object" {
			t.Errorf("Expected response_format.type=json_object")
		}

		response := openAIResponse{
			ID: "chatcmpl-789",
			Choices: []struct {
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: `{"name":"test","value":42}`,
					},
					FinishReason: "stop",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewOpenAIClient("test-api-key", "gpt-4o-mini")
	client.client = &http.Client{
		Transport: &replaceURLTransport{
			base:   server.URL,
			target: openAIBaseURL,
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

// TestOpenAIErrors tests error handling scenarios.
func TestOpenAIErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   openAIResponse
		wantErrMsg string
	}{
		{
			name:       "invalid API key (401)",
			statusCode: http.StatusUnauthorized,
			response: openAIResponse{
				Error: &struct {
					Message string `json:"message"`
					Type    string `json:"type"`
				}{
					Message: "Incorrect API key provided",
					Type:    "invalid_request_error",
				},
			},
			wantErrMsg: "Incorrect API key provided",
		},
		{
			name:       "rate limit exceeded (429)",
			statusCode: http.StatusTooManyRequests,
			response: openAIResponse{
				Error: &struct {
					Message string `json:"message"`
					Type    string `json:"type"`
				}{
					Message: "Rate limit exceeded",
					Type:    "rate_limit_error",
				},
			},
			wantErrMsg: "Rate limit exceeded",
		},
		{
			name:       "model not found (404)",
			statusCode: http.StatusNotFound,
			response: openAIResponse{
				Error: &struct {
					Message string `json:"message"`
					Type    string `json:"type"`
				}{
					Message: "The model does not exist",
					Type:    "invalid_request_error",
				},
			},
			wantErrMsg: "The model does not exist",
		},
		{
			name:       "quota exceeded",
			statusCode: http.StatusForbidden,
			response: openAIResponse{
				Error: &struct {
					Message string `json:"message"`
					Type    string `json:"type"`
				}{
					Message: "You exceeded your current quota",
					Type:    "insufficient_quota",
				},
			},
			wantErrMsg: "You exceeded your current quota",
		},
		{
			name:       "server error (500)",
			statusCode: http.StatusInternalServerError,
			response: openAIResponse{
				Error: &struct {
					Message string `json:"message"`
					Type    string `json:"type"`
				}{
					Message: "Internal server error",
					Type:    "server_error",
				},
			},
			wantErrMsg: "Internal server error",
		},
		{
			name:       "no choices in response",
			statusCode: http.StatusOK,
			response: openAIResponse{
				ID: "chatcmpl-no-choices",
				Choices: []struct {
					Message struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					} `json:"message"`
					FinishReason string `json:"finish_reason"`
				}{},
			},
			wantErrMsg: "returned no choices",
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

			client := NewOpenAIClient("test-api-key", "gpt-4o-mini")
			client.client = &http.Client{
				Transport: &replaceURLTransport{
					base:   server.URL,
					target: openAIBaseURL,
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

// TestOpenAIMalformedJSON tests handling of malformed JSON responses.
func TestOpenAIMalformedJSON(t *testing.T) {
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
			response: `{"id": "chatcmpl-123", "choices":`,
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

			client := NewOpenAIClient("test-api-key", "gpt-4o-mini")
			client.client = &http.Client{
				Transport: &replaceURLTransport{
					base:   server.URL,
					target: openAIBaseURL,
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

// TestOpenAITimeout tests timeout handling.
func TestOpenAITimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		json.NewEncoder(w).Encode(openAIResponse{
			ID: "chatcmpl-delayed",
			Choices: []struct {
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: "Delayed response",
					},
					FinishReason: "stop",
				},
			},
		})
	}))
	defer server.Close()

	client := NewOpenAIClient("test-api-key", "gpt-4o-mini")
	client.client = &http.Client{
		Timeout: 50 * time.Millisecond,
		Transport: &replaceURLTransport{
			base:   server.URL,
			target: openAIBaseURL,
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

// TestOpenAIContextCancellation tests context cancellation handling.
func TestOpenAIContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		json.NewEncoder(w).Encode(openAIResponse{})
	}))
	defer server.Close()

	client := NewOpenAIClient("test-api-key", "gpt-4o-mini")
	client.client = &http.Client{
		Transport: &replaceURLTransport{
			base:   server.URL,
			target: openAIBaseURL,
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

// TestOpenAIModel tests the Model method.
func TestOpenAIModel(t *testing.T) {
	client := NewOpenAIClient("test-api-key", "gpt-4o")
	if model := client.Model(); model != "gpt-4o" {
		t.Errorf("Model() = %q, want %q", model, "gpt-4o")
	}
}

// TestOpenAIBackend tests the Backend method.
func TestOpenAIBackend(t *testing.T) {
	client := NewOpenAIClient("test-api-key", "gpt-4o-mini")
	if backend := client.Backend(); backend != "openai" {
		t.Errorf("Backend() = %q, want %q", backend, "openai")
	}
}

// TestOpenAINetworkError tests handling of network errors.
func TestOpenAINetworkError(t *testing.T) {
	client := NewOpenAIClient("test-api-key", "gpt-4o-mini")
	client.client = &http.Client{
		Timeout: 1 * time.Second,
		Transport: &replaceURLTransport{
			base:   "http://invalid-host-that-does-not-exist:99999",
			target: openAIBaseURL,
			rt:     http.DefaultTransport,
		},
	}

	_, err := client.Complete(context.Background(), "test", DefaultCompletionOptions())

	if err == nil {
		t.Error("Expected network error, got nil")
		return
	}

	if !strings.Contains(err.Error(), "openai request failed") {
		t.Errorf("Expected 'openai request failed' in error, got: %v", err)
	}
}

// TestOpenAIRequestVerification verifies that request parameters are correctly set.
func TestOpenAIRequestVerification(t *testing.T) {
	var capturedRequest openAIRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedRequest)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAIResponse{
			ID: "chatcmpl-verify",
			Choices: []struct {
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: "test response",
					},
					FinishReason: "stop",
				},
			},
		})
	}))
	defer server.Close()

	client := NewOpenAIClient("test-api-key-123", "gpt-4")
	client.client = &http.Client{
		Transport: &replaceURLTransport{
			base:   server.URL,
			target: openAIBaseURL,
			rt:     http.DefaultTransport,
		},
	}

	opts := CompletionOptions{
		MaxTokens:    1500,
		Temperature:  0.8,
		SystemPrompt: "You are a coding assistant",
		JSONMode:     true,
	}

	_, err := client.Complete(context.Background(), "test prompt", opts)
	if err != nil {
		t.Fatalf("Complete() failed: %v", err)
	}

	// Verify captured request
	if capturedRequest.Model != "gpt-4" {
		t.Errorf("Model = %q, want %q", capturedRequest.Model, "gpt-4")
	}
	if capturedRequest.MaxTokens != opts.MaxTokens {
		t.Errorf("MaxTokens = %d, want %d", capturedRequest.MaxTokens, opts.MaxTokens)
	}
	if capturedRequest.Temperature != opts.Temperature {
		t.Errorf("Temperature = %f, want %f", capturedRequest.Temperature, opts.Temperature)
	}
	if capturedRequest.ResponseFormat == nil || capturedRequest.ResponseFormat.Type != "json_object" {
		t.Errorf("ResponseFormat not set correctly for JSON mode")
	}

	// Verify messages
	if len(capturedRequest.Messages) != 2 {
		t.Fatalf("Expected 2 messages (system + user), got %d", len(capturedRequest.Messages))
	}
	if capturedRequest.Messages[0].Role != "system" {
		t.Errorf("First message role = %q, want system", capturedRequest.Messages[0].Role)
	}
	if capturedRequest.Messages[0].Content != opts.SystemPrompt {
		t.Errorf("System message = %q, want %q", capturedRequest.Messages[0].Content, opts.SystemPrompt)
	}
	if capturedRequest.Messages[1].Role != "user" {
		t.Errorf("Second message role = %q, want user", capturedRequest.Messages[1].Role)
	}
	if capturedRequest.Messages[1].Content != "test prompt" {
		t.Errorf("User message = %q, want %q", capturedRequest.Messages[1].Content, "test prompt")
	}
}

// TestOpenAIMultipleModels tests different model configurations.
func TestOpenAIMultipleModels(t *testing.T) {
	models := []string{"gpt-3.5-turbo", "gpt-4", "gpt-4o", "gpt-4o-mini"}

	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			var capturedModel string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req openAIRequest
				json.NewDecoder(r.Body).Decode(&req)
				capturedModel = req.Model

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(openAIResponse{
					ID: "chatcmpl-model-test",
					Choices: []struct {
						Message struct {
							Role    string `json:"role"`
							Content string `json:"content"`
						} `json:"message"`
						FinishReason string `json:"finish_reason"`
					}{
						{
							Message: struct {
								Role    string `json:"role"`
								Content string `json:"content"`
							}{
								Role:    "assistant",
								Content: "response",
							},
							FinishReason: "stop",
						},
					},
				})
			}))
			defer server.Close()

			client := NewOpenAIClient("test-api-key", model)
			client.client = &http.Client{
				Transport: &replaceURLTransport{
					base:   server.URL,
					target: openAIBaseURL,
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
