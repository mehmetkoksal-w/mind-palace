package llm

import (
	"os"
	"strings"
	"testing"
)

// TestNewClient tests the client factory function.
func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		wantBackend string
		wantModel   string
		wantErr     bool
		errContains string
	}{
		{
			name: "ollama client",
			config: Config{
				Backend: "ollama",
				Model:   "llama3.2",
				URL:     "http://localhost:11434",
			},
			wantBackend: "ollama",
			wantModel:   "llama3.2",
			wantErr:     false,
		},
		{
			name: "ollama with defaults",
			config: Config{
				Backend: "ollama",
			},
			wantBackend: "ollama",
			wantModel:   "llama3.2",
			wantErr:     false,
		},
		{
			name: "openai client",
			config: Config{
				Backend: "openai",
				Model:   "gpt-4o-mini",
				APIKey:  "test-key",
			},
			wantBackend: "openai",
			wantModel:   "gpt-4o-mini",
			wantErr:     false,
		},
		{
			name: "openai missing API key",
			config: Config{
				Backend: "openai",
				Model:   "gpt-4o-mini",
			},
			wantErr:     true,
			errContains: "API key required",
		},
		{
			name: "anthropic client",
			config: Config{
				Backend: "anthropic",
				Model:   "claude-3-haiku-20240307",
				APIKey:  "test-key",
			},
			wantBackend: "anthropic",
			wantModel:   "claude-3-haiku-20240307",
			wantErr:     false,
		},
		{
			name: "anthropic missing API key",
			config: Config{
				Backend: "anthropic",
			},
			wantErr:     true,
			errContains: "API key required",
		},
		{
			name: "disabled backend",
			config: Config{
				Backend: "disabled",
			},
			wantErr:     true,
			errContains: "not configured",
		},
		{
			name: "empty backend",
			config: Config{
				Backend: "",
			},
			wantErr:     true,
			errContains: "not configured",
		},
		{
			name: "unsupported backend",
			config: Config{
				Backend: "unsupported",
			},
			wantErr:     true,
			errContains: "unsupported backend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err != nil && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if client == nil {
				t.Error("NewClient() returned nil client")
				return
			}

			if client.Backend() != tt.wantBackend {
				t.Errorf("Backend() = %q, want %q", client.Backend(), tt.wantBackend)
			}

			if client.Model() != tt.wantModel {
				t.Errorf("Model() = %q, want %q", client.Model(), tt.wantModel)
			}
		})
	}
}

// TestNewClientWithEnvVars tests client creation with environment variables.
func TestNewClientWithEnvVars(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		envVars     map[string]string
		wantBackend string
		wantErr     bool
	}{
		{
			name: "openai with env var",
			config: Config{
				Backend: "openai",
				Model:   "gpt-4o-mini",
			},
			envVars: map[string]string{
				"OPENAI_API_KEY": "env-test-key",
			},
			wantBackend: "openai",
			wantErr:     false,
		},
		{
			name: "anthropic with env var",
			config: Config{
				Backend: "anthropic",
				Model:   "claude-3-haiku-20240307",
			},
			envVars: map[string]string{
				"ANTHROPIC_API_KEY": "env-test-key",
			},
			wantBackend: "anthropic",
			wantErr:     false,
		},
		{
			name: "config API key takes precedence",
			config: Config{
				Backend: "openai",
				Model:   "gpt-4o-mini",
				APIKey:  "config-key",
			},
			envVars: map[string]string{
				"OPENAI_API_KEY": "env-key",
			},
			wantBackend: "openai",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				val := k // capture loop variable
				t.Cleanup(func() {
					os.Unsetenv(val)
				})
			}

			client, err := NewClient(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && client.Backend() != tt.wantBackend {
				t.Errorf("Backend() = %q, want %q", client.Backend(), tt.wantBackend)
			}
		})
	}
}

// TestDefaultConfig tests the default configuration.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Backend != "ollama" {
		t.Errorf("DefaultConfig().Backend = %q, want ollama", cfg.Backend)
	}
	if cfg.Model != "llama3.2" {
		t.Errorf("DefaultConfig().Model = %q, want llama3.2", cfg.Model)
	}
	if cfg.URL != "http://localhost:11434" {
		t.Errorf("DefaultConfig().URL = %q, want http://localhost:11434", cfg.URL)
	}
}

// TestDefaultCompletionOptions tests the default completion options.
func TestDefaultCompletionOptions(t *testing.T) {
	opts := DefaultCompletionOptions()

	if opts.MaxTokens != 2048 {
		t.Errorf("DefaultCompletionOptions().MaxTokens = %d, want 2048", opts.MaxTokens)
	}
	if opts.Temperature != 0.3 {
		t.Errorf("DefaultCompletionOptions().Temperature = %f, want 0.3", opts.Temperature)
	}
	if opts.JSONMode != false {
		t.Errorf("DefaultCompletionOptions().JSONMode = %v, want false", opts.JSONMode)
	}
}

// TestExtractJSON tests JSON extraction from various formats.
func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantJSON string
	}{
		{
			name:     "plain JSON",
			input:    `{"name":"test","value":42}`,
			wantJSON: `{"name":"test","value":42}`,
		},
		{
			name:     "JSON in markdown code block",
			input:    "```json\n{\"name\":\"test\",\"value\":42}\n```",
			wantJSON: `{"name":"test","value":42}`,
		},
		{
			name:     "JSON in generic code block",
			input:    "```\n{\"name\":\"test\",\"value\":42}\n```",
			wantJSON: `{"name":"test","value":42}`,
		},
		{
			name:     "JSON with prefix text",
			input:    "Here is the result:\n{\"name\":\"test\",\"value\":42}",
			wantJSON: `{"name":"test","value":42}`,
		},
		{
			name:     "array JSON",
			input:    `[{"id":1},{"id":2}]`,
			wantJSON: `[{"id":1},{"id":2}]`,
		},
		{
			name:     "nested JSON",
			input:    `{"outer":{"inner":{"deep":"value"}}}`,
			wantJSON: `{"outer":{"inner":{"deep":"value"}}}`,
		},
		{
			name:     "JSON with escaped quotes",
			input:    `{"message":"He said \"hello\""}`,
			wantJSON: `{"message":"He said \"hello\""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			if result != tt.wantJSON {
				t.Errorf("extractJSON() = %q, want %q", result, tt.wantJSON)
			}
		})
	}
}

// TestParseJSONResponse tests JSON parsing from LLM responses.
func TestParseJSONResponse(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name     string
		response string
		want     TestStruct
		wantErr  bool
	}{
		{
			name:     "valid JSON",
			response: `{"name":"test","value":42}`,
			want:     TestStruct{Name: "test", Value: 42},
			wantErr:  false,
		},
		{
			name:     "JSON in code block",
			response: "```json\n{\"name\":\"example\",\"value\":100}\n```",
			want:     TestStruct{Name: "example", Value: 100},
			wantErr:  false,
		},
		{
			name:     "invalid JSON",
			response: `{invalid}`,
			wantErr:  true,
		},
		{
			name:     "empty response",
			response: ``,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result TestStruct
			err := parseJSONResponse(tt.response, &result)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseJSONResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.want {
				t.Errorf("parseJSONResponse() result = %+v, want %+v", result, tt.want)
			}
		})
	}
}

// TestFindJSONStart tests JSON start position detection.
func TestFindJSONStart(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "starts with object",
			input: `{"key":"value"}`,
			want:  0,
		},
		{
			name:  "starts with array",
			input: `[1,2,3]`,
			want:  0,
		},
		{
			name:  "json code block",
			input: "```json\n{\"key\":\"value\"}",
			want:  8, // After ```json\n
		},
		{
			name:  "generic code block",
			input: "```\n{\"key\":\"value\"}",
			want:  4, // At the {
		},
		{
			name:  "text before JSON",
			input: "Here is the result: {\"key\":\"value\"}",
			want:  20, // At the {
		},
		{
			name:  "no JSON",
			input: "No JSON here",
			want:  -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findJSONStart(tt.input)
			if result != tt.want {
				t.Errorf("findJSONStart() = %d, want %d", result, tt.want)
			}
		})
	}
}

// TestFindJSONEnd tests JSON end position detection.
func TestFindJSONEnd(t *testing.T) {
	tests := []struct {
		name  string
		input string
		start int
		want  int
	}{
		{
			name:  "simple object",
			input: `{"key":"value"}`,
			start: 0,
			want:  15,
		},
		{
			name:  "nested objects",
			input: `{"outer":{"inner":"value"}}`,
			start: 0,
			want:  27,
		},
		{
			name:  "array",
			input: `[1,2,3]`,
			start: 0,
			want:  7,
		},
		{
			name:  "escaped quotes in string",
			input: `{"message":"He said \"hello\""}`,
			start: 0,
			want:  31,
		},
		{
			name:  "multiple objects, find first",
			input: `{"first":"obj"}{"second":"obj"}`,
			start: 0,
			want:  15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findJSONEnd(tt.input, tt.start)
			if result != tt.want {
				t.Errorf("findJSONEnd() = %d, want %d", result, tt.want)
			}
		})
	}
}

// TestIndexOf tests the indexOf helper function.
func TestIndexOf(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   int
	}{
		{
			name:   "found at start",
			s:      "hello world",
			substr: "hello",
			want:   0,
		},
		{
			name:   "found in middle",
			s:      "hello world",
			substr: "world",
			want:   6,
		},
		{
			name:   "not found",
			s:      "hello world",
			substr: "foo",
			want:   -1,
		},
		{
			name:   "empty substring",
			s:      "hello",
			substr: "",
			want:   0,
		},
		{
			name:   "substring longer than string",
			s:      "hi",
			substr: "hello",
			want:   -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexOf(tt.s, tt.substr)
			if result != tt.want {
				t.Errorf("indexOf() = %d, want %d", result, tt.want)
			}
		})
	}
}

// TestTruncate tests the truncate helper function.
func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "no truncation needed",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact length",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "truncation needed",
			input:  "hello world",
			maxLen: 5,
			want:   "hello...",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.want {
				t.Errorf("truncate() = %q, want %q", result, tt.want)
			}
		})
	}
}

// TestClientInterfaceCompliance verifies all clients implement the Client interface.
func TestClientInterfaceCompliance(t *testing.T) {
	var _ Client = (*OllamaClient)(nil)
	var _ Client = (*OpenAIClient)(nil)
	var _ Client = (*AnthropicClient)(nil)
}

// TestProviderSwitching tests switching between different providers.
func TestProviderSwitching(t *testing.T) {
	configs := []Config{
		{Backend: "ollama", Model: "llama3.2", URL: "http://localhost:11434"},
		{Backend: "openai", Model: "gpt-4o-mini", APIKey: "test-key-1"},
		{Backend: "anthropic", Model: "claude-3-haiku-20240307", APIKey: "test-key-2"},
	}

	expectedBackends := []string{"ollama", "openai", "anthropic"}
	expectedModels := []string{"llama3.2", "gpt-4o-mini", "claude-3-haiku-20240307"}

	for i, cfg := range configs {
		t.Run(cfg.Backend, func(t *testing.T) {
			client, err := NewClient(cfg)
			if err != nil {
				t.Fatalf("NewClient() failed: %v", err)
			}

			if client.Backend() != expectedBackends[i] {
				t.Errorf("Backend() = %q, want %q", client.Backend(), expectedBackends[i])
			}

			if client.Model() != expectedModels[i] {
				t.Errorf("Model() = %q, want %q", client.Model(), expectedModels[i])
			}
		})
	}
}

// TestErrorConstants tests that error constants are properly defined.
func TestErrorConstants(t *testing.T) {
	if ErrNotConfigured == nil {
		t.Error("ErrNotConfigured is nil")
	}
	if ErrUnsupportedBackend == nil {
		t.Error("ErrUnsupportedBackend is nil")
	}

	if ErrNotConfigured.Error() != "llm: backend not configured" {
		t.Errorf("ErrNotConfigured.Error() = %q, want 'llm: backend not configured'", ErrNotConfigured.Error())
	}
	if ErrUnsupportedBackend.Error() != "llm: unsupported backend" {
		t.Errorf("ErrUnsupportedBackend.Error() = %q, want 'llm: unsupported backend'", ErrUnsupportedBackend.Error())
	}
}

// TestConcurrentRequests tests that clients can handle concurrent requests.
func TestConcurrentRequests(t *testing.T) {
	// This test verifies thread-safety of client instances
	client := NewOllamaClient("http://localhost:11434", "llama3.2")

	// Running multiple goroutines to check for race conditions
	done := make(chan bool, 3)

	for i := 0; i < 3; i++ {
		go func() {
			// Just verify the methods can be called concurrently
			_ = client.Model()
			_ = client.Backend()
			done <- true
		}()
	}

	for i := 0; i < 3; i++ {
		<-done
	}

	// If we get here without panicking or deadlocking, test passes
}

// BenchmarkExtractJSON benchmarks the JSON extraction function.
func BenchmarkExtractJSON(b *testing.B) {
	input := "```json\n{\"name\":\"test\",\"value\":42,\"nested\":{\"deep\":\"value\"}}\n```"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = extractJSON(input)
	}
}

// BenchmarkParseJSONResponse benchmarks JSON parsing.
func BenchmarkParseJSONResponse(b *testing.B) {
	type TestStruct struct {
		Name   string `json:"name"`
		Value  int    `json:"value"`
		Nested struct {
			Deep string `json:"deep"`
		} `json:"nested"`
	}

	response := `{"name":"test","value":42,"nested":{"deep":"value"}}`
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var result TestStruct
		_ = parseJSONResponse(response, &result)
	}
}
