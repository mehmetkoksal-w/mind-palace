package corridor

import (
	"testing"
)

func TestNamespacePath(t *testing.T) {
	tests := []struct {
		neighbor string
		path     string
		expected string
	}{
		{"backend", "src/api.ts", "corridor://backend/src/api.ts"},
		{"backend", "/src/api.ts", "corridor://backend/src/api.ts"},
		{"frontend", "components/Button.tsx", "corridor://frontend/components/Button.tsx"},
		{"core", "", "corridor://core/"},
	}

	for _, tt := range tests {
		result := NamespacePath(tt.neighbor, tt.path)
		if result != tt.expected {
			t.Errorf("NamespacePath(%q, %q) = %q, want %q", tt.neighbor, tt.path, result, tt.expected)
		}
	}
}

func TestParseNamespacedPath(t *testing.T) {
	tests := []struct {
		path           string
		wantNeighbor   string
		wantRelative   string
		wantIsCorridor bool
	}{
		{"corridor://backend/src/api.ts", "backend", "src/api.ts", true},
		{"corridor://frontend/components/Button.tsx", "frontend", "components/Button.tsx", true},
		{"corridor://core/", "core", "", true},
		{"corridor://core", "core", "", true},
		{"src/local.go", "", "src/local.go", false},
		{"/absolute/path.go", "", "/absolute/path.go", false},
	}

	for _, tt := range tests {
		neighbor, relative, isCorridor := ParseNamespacedPath(tt.path)
		if neighbor != tt.wantNeighbor || relative != tt.wantRelative || isCorridor != tt.wantIsCorridor {
			t.Errorf("ParseNamespacedPath(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.path, neighbor, relative, isCorridor,
				tt.wantNeighbor, tt.wantRelative, tt.wantIsCorridor)
		}
	}
}

func TestParseTTL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "24h0m0s"},
		{"24h", "24h0m0s"},
		{"1h", "1h0m0s"},
		{"30m", "30m0s"},
		{"invalid", "24h0m0s"},
	}

	for _, tt := range tests {
		result := parseTTL(tt.input)
		if result.String() != tt.expected {
			t.Errorf("parseTTL(%q) = %s, want %s", tt.input, result.String(), tt.expected)
		}
	}
}

func TestExpandEnv(t *testing.T) {
	t.Setenv("TEST_TOKEN", "secret123")

	tests := []struct {
		input    string
		expected string
	}{
		{"$TEST_TOKEN", "secret123"},
		{"$NONEXISTENT", ""},
		{"literal", "literal"},
		{"", ""},
	}

	for _, tt := range tests {
		result := expandEnv(tt.input)
		if result != tt.expected {
			t.Errorf("expandEnv(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestStripJSONComments(t *testing.T) {
	input := `{
  // This is a comment
  "name": "test",
  "value": 123 // inline comment
}`

	result := stripJSONComments([]byte(input))

	expected := `{
  "name": "test",
  "value": 123 
}`

	if string(result) != expected {
		t.Errorf("stripJSONComments result differs\nGot:\n%s\nWant:\n%s", result, expected)
	}
}
