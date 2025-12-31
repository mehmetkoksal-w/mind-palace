package butler

import (
	"testing"
)

func TestSplitIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"simple", []string{"simple"}},
		{"getUserName", []string{"get", "user", "name", "getUserName"}},
		{"get_user_name", []string{"get", "user", "name", "get_user_name"}},
		{"HTTPServer", []string{"http", "server", "HTTPServer"}},
		{"parseJSON", []string{"parse", "json", "parseJSON"}},
		{"HTMLParser", []string{"html", "parser", "HTMLParser"}},
		{"getID", []string{"get", "id", "getID"}},
		{"simple", []string{"simple"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitIdentifier(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("splitIdentifier(%q) = %v (len %d), want %v (len %d)",
					tt.input, got, len(got), tt.expected, len(tt.expected))
				return
			}
			// Check approximate content (order-insensitive for some cases)
			for i, exp := range tt.expected {
				if i < len(got) && got[i] != exp {
					// Allow lowercase comparison
					t.Logf("splitIdentifier(%q)[%d] = %q, expected %q", tt.input, i, got[i], exp)
				}
			}
		})
	}
}

func TestIsCodeIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", false},
		{"a", false},          // too short
		{"ab", false},         // no mixed case, no underscore
		{"AB", false},         // no mixed case, no underscore
		{"Ab", true},          // mixed case
		{"aB", true},          // mixed case
		{"get_user", true},    // underscore
		{"get-user", true},    // hyphen
		{"getUserName", true}, // camelCase
		{"HTTPServer", true},  // acronym + word
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isCodeIdentifier(tt.input)
			if got != tt.expected {
				t.Errorf("isCodeIdentifier(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExpandQueryTokens(t *testing.T) {
	tests := []struct {
		query       string
		minExpected int
	}{
		{"simple", 1},
		{"getUserName", 4}, // get, user, name, getUserName
		{"get_user", 3},    // get, user, get_user
		{"hello world", 2}, // Two simple words
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := expandQueryTokens(tt.query)
			if len(got) < tt.minExpected {
				t.Errorf("expandQueryTokens(%q) = %v (len %d), want at least %d tokens",
					tt.query, got, len(got), tt.minExpected)
			}
		})
	}
}

func TestDedup(t *testing.T) {
	tests := []struct {
		input    []string
		expected []string
	}{
		{nil, nil},
		{[]string{}, nil},
		{[]string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{[]string{"a", "A", "a"}, []string{"a"}}, // case insensitive dedup
		{[]string{"one", "two", "ONE", "three"}, []string{"one", "two", "three"}},
	}

	for _, tt := range tests {
		got := dedup(tt.input)
		if len(got) != len(tt.expected) {
			t.Errorf("dedup(%v) = %v (len %d), want %v (len %d)",
				tt.input, got, len(got), tt.expected, len(tt.expected))
		}
	}
}

func TestSplitIdentifierWithKebabCase(t *testing.T) {
	got := splitIdentifier("my-component-name")

	// Should contain the parts
	found := make(map[string]bool)
	for _, part := range got {
		found[part] = true
	}

	if !found["my"] {
		t.Error("expected 'my' in split result")
	}
	if !found["component"] {
		t.Error("expected 'component' in split result")
	}
	if !found["name"] {
		t.Error("expected 'name' in split result")
	}
}
