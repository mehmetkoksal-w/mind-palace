package butler

import (
	"testing"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
)

func TestPreprocessQuery(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		mustContain  []string // Substrings the result must contain
		mustNotEqual string   // The result must not equal this (for non-empty)
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

func TestPreprocessQueryWithFuzzy(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		mustContain []string
	}{
		{
			name:        "word with fuzzy variants",
			query:       "search",
			mustContain: []string{`"search"`},
		},
		{
			name:        "empty query",
			query:       "",
			mustContain: nil,
		},
		{
			name:        "CamelCase with fuzzy",
			query:       "ProcessData",
			mustContain: []string{`"ProcessData"`, `"process"`, `"data"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := preprocessQueryWithFuzzy(tt.query)

			if len(tt.mustContain) == 0 {
				if result != "" {
					t.Errorf("preprocessQueryWithFuzzy(%q) = %q, expected empty", tt.query, result)
				}
				return
			}

			for _, substr := range tt.mustContain {
				if !contains(result, substr) {
					t.Errorf("preprocessQueryWithFuzzy(%q) = %q, expected to contain %q", tt.query, result, substr)
				}
			}
		})
	}
}

func TestPreprocessQueryWithOptions(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		useSynonyms bool
		expectEmpty bool
	}{
		{
			name:        "with synonyms",
			query:       "auth",
			useSynonyms: true,
			expectEmpty: false,
		},
		{
			name:        "without synonyms",
			query:       "auth",
			useSynonyms: false,
			expectEmpty: false,
		},
		{
			name:        "empty query",
			query:       "",
			useSynonyms: true,
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := preprocessQueryWithOptions(tt.query, tt.useSynonyms)

			if tt.expectEmpty && result != "" {
				t.Errorf("preprocessQueryWithOptions(%q, %v) = %q, expected empty", tt.query, tt.useSynonyms, result)
			}
			if !tt.expectEmpty && result == "" {
				t.Errorf("preprocessQueryWithOptions(%q, %v) = empty, expected non-empty", tt.query, tt.useSynonyms)
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

func TestInferRoom(t *testing.T) {
	b := &Butler{
		rooms: map[string]model.Room{
			"auth": {
				Name:        "auth",
				EntryPoints: []string{"internal/auth/**"},
			},
			"api": {
				Name:        "api",
				EntryPoints: []string{"internal/api/**"},
			},
		},
		entryPoints: map[string]string{
			"internal/auth/handler.go": "auth",
			"internal/api/routes.go":   "api",
		},
	}

	tests := []struct {
		path     string
		expected string
	}{
		{"internal/auth/handler.go", "auth"},
		{"internal/api/routes.go", "api"},
		{"unknown/path.go", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := b.inferRoom(tt.path)
			if result != tt.expected {
				t.Errorf("inferRoom(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestListRooms(t *testing.T) {
	b := &Butler{
		rooms: map[string]model.Room{
			"auth": {Name: "auth", Summary: "Auth module"},
			"api":  {Name: "api", Summary: "API module"},
		},
	}

	rooms := b.ListRooms()

	if len(rooms) != 2 {
		t.Errorf("Expected 2 rooms, got %d", len(rooms))
	}

	// Check that rooms are sorted by name
	names := make([]string, len(rooms))
	for i, r := range rooms {
		names[i] = r.Name
	}

	for i := 0; i < len(names)-1; i++ {
		if names[i] > names[i+1] {
			t.Errorf("Rooms not sorted: %v", names)
			break
		}
	}
}

func TestReadRoom(t *testing.T) {
	b := &Butler{
		rooms: map[string]model.Room{
			"auth": {Name: "auth", Summary: "Auth module"},
		},
	}

	t.Run("existing room", func(t *testing.T) {
		room, err := b.ReadRoom("auth")
		if err != nil {
			t.Fatalf("ReadRoom failed: %v", err)
		}
		if room.Name != "auth" {
			t.Errorf("Expected room name 'auth', got %q", room.Name)
		}
	})

	t.Run("non-existing room", func(t *testing.T) {
		_, err := b.ReadRoom("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existing room")
		}
	})
}

func TestSearchResult(t *testing.T) {
	result := SearchResult{
		Path:       "src/auth/handler.go",
		Room:       "auth",
		ChunkIndex: 0,
		StartLine:  10,
		EndLine:    20,
		Snippet:    "func HandleAuth() { ... }",
		Score:      10.5,
		IsEntry:    true,
	}

	if result.Path != "src/auth/handler.go" {
		t.Error("Path not set correctly")
	}
	if result.Room != "auth" {
		t.Error("Room not set correctly")
	}
	if result.StartLine != 10 {
		t.Error("StartLine not set correctly")
	}
	if result.Score != 10.5 {
		t.Error("Score not set correctly")
	}
	if !result.IsEntry {
		t.Error("IsEntry not set correctly")
	}
}

func TestGroupedResults(t *testing.T) {
	grouped := GroupedResults{
		Room:    "auth",
		Summary: "Authentication module",
		Results: []SearchResult{
			{Path: "auth/handler.go", Score: 10.0},
			{Path: "auth/service.go", Score: 8.0},
		},
	}

	if grouped.Room != "auth" {
		t.Error("Room not set correctly")
	}
	if len(grouped.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(grouped.Results))
	}
}

func TestSearchOptions(t *testing.T) {
	opts := SearchOptions{
		Limit:      20,
		RoomFilter: "auth",
		FuzzyMatch: true,
	}

	if opts.Limit != 20 {
		t.Error("Limit not set correctly")
	}
	if opts.RoomFilter != "auth" {
		t.Error("RoomFilter not set correctly")
	}
	if !opts.FuzzyMatch {
		t.Error("FuzzyMatch not set correctly")
	}
}

func TestEnhancedContextOptions(t *testing.T) {
	opts := EnhancedContextOptions{
		Query:            "auth handler",
		Limit:            10,
		MaxTokens:        4096,
		IncludeTests:     true,
		IncludeLearnings: true,
		IncludeFileIntel: true,
		SessionID:        "session-123",
	}

	if opts.Query != "auth handler" {
		t.Error("Query not set correctly")
	}
	if opts.MaxTokens != 4096 {
		t.Error("MaxTokens not set correctly")
	}
	if !opts.IncludeTests {
		t.Error("IncludeTests not set correctly")
	}
	if opts.SessionID != "session-123" {
		t.Error("SessionID not set correctly")
	}
}

func TestSetJSONCDecoder(t *testing.T) {
	// Save original
	original := jsonCDecode

	// Set custom decoder
	customDecoder := func(path string, v interface{}) error {
		return nil
	}

	SetJSONCDecoder(customDecoder)

	// Verify it was set
	if jsonCDecode == nil {
		t.Error("jsonCDecode should not be nil after SetJSONCDecoder")
	}

	// Restore original
	jsonCDecode = original
}
