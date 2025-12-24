package cli

import (
	"testing"
)

func TestBoolFlag(t *testing.T) {
	var f boolFlag

	// Test default state
	if f.value != false || f.set != false {
		t.Fatalf("expected default false/unset, got value=%v set=%v", f.value, f.set)
	}

	// Test setting to true
	if err := f.Set("true"); err != nil {
		t.Fatalf("unexpected error setting true: %v", err)
	}
	if !f.value || !f.set {
		t.Fatalf("expected true/set after Set(true), got value=%v set=%v", f.value, f.set)
	}

	// Test setting to false
	f = boolFlag{}
	if err := f.Set("false"); err != nil {
		t.Fatalf("unexpected error setting false: %v", err)
	}
	if f.value || !f.set {
		t.Fatalf("expected false/set after Set(false), got value=%v set=%v", f.value, f.set)
	}

	// Test empty string (boolean flag without value)
	f = boolFlag{}
	if err := f.Set(""); err != nil {
		t.Fatalf("unexpected error setting empty: %v", err)
	}
	if !f.value || !f.set {
		t.Fatalf("expected true/set after Set(\"\"), got value=%v set=%v", f.value, f.set)
	}

	// Test invalid value
	f = boolFlag{}
	if err := f.Set("invalid"); err == nil {
		t.Fatal("expected error for invalid value")
	}

	// Test String()
	f = boolFlag{value: true}
	if f.String() != "true" {
		t.Fatalf("expected String()=\"true\", got %q", f.String())
	}
	f = boolFlag{value: false}
	if f.String() != "false" {
		t.Fatalf("expected String()=\"false\", got %q", f.String())
	}

	// Test IsBoolFlag
	if !f.IsBoolFlag() {
		t.Fatal("expected IsBoolFlag() to return true")
	}
}

func TestTruncateLine(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a longer line", 10, "this is a ..."},
		{"", 5, ""},
	}

	for _, tt := range tests {
		got := truncateLine(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateLine(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}
