package butler

import (
	"testing"

	"github.com/koksalmehmet/mind-palace/internal/model"
)

func TestPreprocessQuery(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		mustContain   []string // Substrings the result must contain
		mustNotEqual  string   // The result must not equal this (for non-empty)
	}{
		{
			name:         "empty query",
			query:        "",
			mustContain:  nil,
			mustNotEqual: "something",
		},
		{
			name:         "whitespace only",
			query:        "   ",
			mustContain:  nil,
			mustNotEqual: "something",
		},
		{
			name:         "code-like with underscore - tokenized and expanded",
			query:        "func_name",
			mustContain:  []string{`"func"`, `"name"`, `"func_name"`}, // Now tokenized
			mustNotEqual: "",
		},
		{
			name:         "code-like with dot - exact match",
			query:        "Class.method",
			mustContain:  []string{`"Class.method"`}, // Exact phrase for code symbols
			mustNotEqual: "",
		},
		{
			name:         "code-like with double colon",
			query:        "pkg::path",
			mustContain:  []string{`"pkg::path"`},
			mustNotEqual: "",
		},
		{
			name:         "natural language single word with synonyms",
			query:        "authentication",
			mustContain:  []string{`"authentication"`}, // Original term included
			mustNotEqual: "",
		},
		{
			name:         "natural language multiple words with synonyms",
			query:        "where is auth",
			mustContain:  []string{`"where"`, `"auth"`, `"authentication"`}, // Synonym expansion
			mustNotEqual: "",
		},
		{
			name:         "short words filtered",
			query:        "a b search",
			mustContain:  []string{`"search"`},
			mustNotEqual: "",
		},
		{
			name:         "CamelCase tokenization",
			query:        "handleAuth function",
			mustContain:  []string{`"handle"`, `"auth"`, `"handleAuth"`, `"function"`}, // CamelCase split
			mustNotEqual: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := preprocessQuery(tt.query)

			// Check empty case
			if len(tt.mustContain) == 0 {
				if result != "" {
					t.Errorf("preprocessQuery(%q) = %q, expected empty", tt.query, result)
				}
				return
			}

			// Check that result contains expected substrings
			for _, substr := range tt.mustContain {
				if !contains(result, substr) {
					t.Errorf("preprocessQuery(%q) = %q, expected to contain %q", tt.query, result, substr)
				}
			}
		})
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
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

