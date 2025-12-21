package butler

import (
	"testing"

	"github.com/koksalmehmet/mind-palace/internal/model"
)

func TestPreprocessQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "empty query",
			query:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			query:    "   ",
			expected: "",
		},
		{
			name:     "code-like with underscore",
			query:    "func_name",
			expected: `"func_name"`,
		},
		{
			name:     "code-like with dot",
			query:    "Class.method",
			expected: `"Class.method"`,
		},
		{
			name:     "code-like with double colon",
			query:    "pkg::path",
			expected: `"pkg::path"`,
		},
		{
			name:     "natural language single word",
			query:    "authentication",
			expected: `"authentication"*`,
		},
		{
			name:     "natural language multiple words",
			query:    "where is auth",
			expected: `"where"* OR "is"* OR "auth"*`,
		},
		{
			name:     "short words filtered",
			query:    "a b search",
			expected: `"search"*`,
		},
		{
			name:     "mixed natural and code",
			query:    "handleAuth function",
			expected: `"handleAuth"* OR "function"*`, // No code symbols, so it's treated as natural language
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := preprocessQuery(tt.query)
			if result != tt.expected {
				t.Errorf("preprocessQuery(%q) = %q, want %q", tt.query, result, tt.expected)
			}
		})
	}
}

func TestCalculateScore(t *testing.T) {
	b := &Butler{
		entryPoints: map[string]string{
			"README.md": "project-overview",
		},
	}

	tests := []struct {
		name        string
		baseScore   float64
		path        string
		query       string
		expectBoost bool
	}{
		{
			name:        "entry point boost",
			baseScore:   -1.0,
			path:        "README.md",
			query:       "test",
			expectBoost: true,
		},
		{
			name:        "path match boost",
			baseScore:   -1.0,
			path:        "internal/auth/handler.go",
			query:       "auth",
			expectBoost: true,
		},
		{
			name:        "no boost",
			baseScore:   -1.0,
			path:        "config.json",
			query:       "test",
			expectBoost: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := b.calculateScore(tt.baseScore, tt.path, tt.query)

			basePositive := 1.0

			if tt.expectBoost {
				if score <= basePositive*1.2 { // Minimum boost is 1.2 for code files
					t.Errorf("Expected boost for %s with query %q, got score %.2f", tt.path, tt.query, score)
				}
			}
		})
	}
}

func TestGroupByRoom(t *testing.T) {
	b := &Butler{
		rooms: map[string]model.Room{
			"auth": {Name: "auth", Summary: "Authentication module"},
		},
		entryPoints: map[string]string{},
	}

	results := []SearchResult{
		{Path: "auth/handler.go", Room: "auth", Score: 10.0},
		{Path: "auth/service.go", Room: "auth", Score: 8.0},
		{Path: "main.go", Room: "", Score: 5.0},
	}

	grouped := b.groupByRoom(results)

	if len(grouped) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(grouped))
	}

	if grouped[0].Room != "auth" {
		t.Errorf("Expected first group to be 'auth', got %q", grouped[0].Room)
	}

	if len(grouped[0].Results) != 2 {
		t.Errorf("Expected 2 results in 'auth' group, got %d", len(grouped[0].Results))
	}
}

