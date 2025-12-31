package butler

import (
	"testing"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"abc", "adc", 1},
		{"abc", "Abc", 0}, // case insensitive
		{"kitten", "sitting", 3},
		{"hello", "hallo", 1},
		{"test", "toast", 2},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			got := LevenshteinDistance(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		query, target string
		maxDistance   int
		expected      bool
	}{
		{"test", "test", 0, true},
		{"test", "tost", 1, true},
		{"test", "toast", 1, false},
		{"test", "toast", 2, true},
		{"hello", "hallo", 2, true},
		{"abc", "xyz", 3, true},
		{"abc", "xyz", 2, false},
	}

	for _, tt := range tests {
		t.Run(tt.query+"_"+tt.target, func(t *testing.T) {
			got := FuzzyMatch(tt.query, tt.target, tt.maxDistance)
			if got != tt.expected {
				t.Errorf("FuzzyMatch(%q, %q, %d) = %v, want %v", tt.query, tt.target, tt.maxDistance, got, tt.expected)
			}
		})
	}
}

func TestFuzzyMatchScore(t *testing.T) {
	tests := []struct {
		query, target string
		minScore      float64
		maxScore      float64
	}{
		{"test", "test", 0.99, 1.01},     // identical = 1.0
		{"", "", 0.99, 1.01},             // both empty = 1.0
		{"test", "tost", 0.7, 0.8},       // 1 diff out of 4
		{"kitten", "sitting", 0.5, 0.65}, // 3 diffs out of 7
	}

	for _, tt := range tests {
		t.Run(tt.query+"_"+tt.target, func(t *testing.T) {
			got := FuzzyMatchScore(tt.query, tt.target)
			if got < tt.minScore || got > tt.maxScore {
				t.Errorf("FuzzyMatchScore(%q, %q) = %v, want between %v and %v", tt.query, tt.target, got, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestSuggestFuzzyMatches(t *testing.T) {
	candidates := []string{"function", "method", "handler", "helper", "handle"}

	// Should find "handle" and "handler" with typo "handel"
	results := SuggestFuzzyMatches("handel", candidates, 2)

	if len(results) == 0 {
		t.Error("expected to find fuzzy matches")
	}

	// Check that results are sorted by distance
	for i := 1; i < len(results); i++ {
		if results[i].Distance < results[i-1].Distance {
			t.Errorf("results not sorted by distance: %d before %d", results[i-1].Distance, results[i].Distance)
		}
	}
}

func TestGetMaxFuzzyDistance(t *testing.T) {
	tests := []struct {
		length   int
		expected int
	}{
		{1, 0},
		{2, 0},
		{3, 0},
		{4, 1},
		{5, 1},
		{6, 2},
		{7, 2},
		{8, 2},
		{9, 3},
		{20, 3},
	}

	for _, tt := range tests {
		got := GetMaxFuzzyDistance(tt.length)
		if got != tt.expected {
			t.Errorf("GetMaxFuzzyDistance(%d) = %d, want %d", tt.length, got, tt.expected)
		}
	}
}

func TestExpandWithFuzzyVariants(t *testing.T) {
	// Short terms should not expand (distance 0)
	result := ExpandWithFuzzyVariants("get", CommonProgrammingTerms)
	if len(result) != 1 || result[0] != "get" {
		t.Errorf("short term should not expand, got %v", result)
	}

	// Longer terms with typos might expand
	result = ExpandWithFuzzyVariants("functio", CommonProgrammingTerms)
	if len(result) < 1 {
		t.Error("expected at least the original term")
	}
}

func TestNormalizeForFuzzy(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello_World", "helloworld"},
		{"get-user-name", "getusername"},
		{"Test123", "test123"},
		{"  spaces  ", "spaces"},
		{"ALL_CAPS", "allcaps"},
		{"", ""},
	}

	for _, tt := range tests {
		got := NormalizeForFuzzy(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeForFuzzy(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMinMax(t *testing.T) {
	if got := min(3, 1, 2); got != 1 {
		t.Errorf("min(3,1,2) = %d, want 1", got)
	}

	if got := max(1, 2); got != 2 {
		t.Errorf("max(1,2) = %d, want 2", got)
	}

	if got := max(5, 3); got != 5 {
		t.Errorf("max(5,3) = %d, want 5", got)
	}
}

func TestSortFuzzyResults(t *testing.T) {
	results := []FuzzyResult{
		{Term: "far", Distance: 3, Score: 0.5},
		{Term: "close", Distance: 1, Score: 0.9},
		{Term: "medium", Distance: 2, Score: 0.7},
	}

	sortFuzzyResults(results)

	if results[0].Term != "close" {
		t.Errorf("first should be 'close', got %s", results[0].Term)
	}
	if results[1].Term != "medium" {
		t.Errorf("second should be 'medium', got %s", results[1].Term)
	}
	if results[2].Term != "far" {
		t.Errorf("third should be 'far', got %s", results[2].Term)
	}
}
