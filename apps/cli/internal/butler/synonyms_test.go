package butler

import (
	"testing"
)

func TestGetSynonyms(t *testing.T) {
	tests := []struct {
		term         string
		wantMinCount int
		shouldExist  bool
	}{
		{"get", 5, true},    // "get" has synonyms like fetch, retrieve, etc.
		{"create", 5, true}, // "create" has synonyms like new, make, etc.
		{"nonexistent", 0, false},
		{"GET", 5, true}, // case insensitive
		{"delete", 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.term, func(t *testing.T) {
			synonyms := GetSynonyms(tt.term)
			if tt.shouldExist && len(synonyms) < tt.wantMinCount {
				t.Errorf("GetSynonyms(%q) = %d synonyms, want at least %d", tt.term, len(synonyms), tt.wantMinCount)
			}
			if !tt.shouldExist && synonyms != nil {
				t.Errorf("GetSynonyms(%q) = %v, want nil", tt.term, synonyms)
			}
		})
	}
}

func TestExpandWithSynonyms(t *testing.T) {
	// Basic expansion
	tokens := []string{"get", "user"}
	expanded := expandWithSynonyms(tokens)

	// Should have more tokens than original
	if len(expanded) <= len(tokens) {
		t.Errorf("expandWithSynonyms should add synonyms, got %d tokens (original %d)", len(expanded), len(tokens))
	}

	// Original tokens should be included
	foundGet := false
	foundUser := false
	for _, tok := range expanded {
		if tok == "get" {
			foundGet = true
		}
		if tok == "user" {
			foundUser = true
		}
	}
	if !foundGet {
		t.Error("original token 'get' should be in expanded list")
	}
	if !foundUser {
		t.Error("original token 'user' should be in expanded list")
	}

	// Should include synonym for 'get'
	foundFetch := false
	for _, tok := range expanded {
		if tok == "fetch" {
			foundFetch = true
		}
	}
	if !foundFetch {
		t.Error("synonym 'fetch' should be in expanded list for 'get'")
	}
}

func TestExpandWithSynonymsEmpty(t *testing.T) {
	expanded := expandWithSynonyms([]string{})
	if len(expanded) != 0 {
		t.Errorf("expandWithSynonyms([]) should return empty, got %v", expanded)
	}
}

func TestExpandWithSynonymsNoDuplicates(t *testing.T) {
	// Test that duplicates are removed
	tokens := []string{"get", "fetch"} // fetch is a synonym of get
	expanded := expandWithSynonyms(tokens)

	// Count occurrences
	counts := make(map[string]int)
	for _, tok := range expanded {
		counts[tok]++
	}

	for tok, count := range counts {
		if count > 1 {
			t.Errorf("token %q appears %d times, should be 1", tok, count)
		}
	}
}

func TestProgrammingSynonymsMapExists(t *testing.T) {
	if ProgrammingSynonyms == nil {
		t.Fatal("ProgrammingSynonyms should not be nil")
	}

	// Check some expected entries
	expectedKeys := []string{"get", "set", "create", "delete", "update", "handler", "parse", "auth"}
	for _, key := range expectedKeys {
		if _, ok := ProgrammingSynonyms[key]; !ok {
			t.Errorf("expected key %q in ProgrammingSynonyms", key)
		}
	}
}
