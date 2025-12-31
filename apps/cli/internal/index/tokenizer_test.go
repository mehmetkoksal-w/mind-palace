package index

import (
	"strings"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		text string
		min  int
		max  int
	}{
		{"hello", 1, 2},
		{"function calculateScore(baseScore float64, path, query string)", 5, 20},
		{"", 0, 0},
	}

	for _, tt := range tests {
		got := EstimateTokens(tt.text)
		if got < tt.min || got > tt.max {
			t.Errorf("EstimateTokens(%q) = %d, want [%d, %d]", tt.text, got, tt.min, tt.max)
		}
	}
}

func TestTruncateToTokenBudget(t *testing.T) {
	items := []BudgetedItem{
		{Item: "A", TokenCount: 100, Priority: 1.0},
		{Item: "B", TokenCount: 100, Priority: 0.5},
	}

	t.Run("full budget", func(t *testing.T) {
		res := TruncateToTokenBudget(items, 300)
		if len(res) != 2 {
			t.Errorf("Expected 2 items, got %d", len(res))
		}
	})

	t.Run("limited budget", func(t *testing.T) {
		res := TruncateToTokenBudget(items, 150)
		if len(res) != 1 {
			t.Errorf("Expected 1 item, got %d", len(res))
		}
		if res[0].Item.(string) != "A" {
			t.Errorf("Expected item A (higher priority), got %v", res[0].Item)
		}
	})
}

func TestTokenBudget(t *testing.T) {
	b := NewTokenBudget(100)
	if !b.CanFit(50) {
		t.Error("Should fit 50")
	}
	if !b.Allocate(50, "symbols") {
		t.Error("Should allocate 50")
	}
	if b.Remaining() != 50 {
		t.Errorf("Expected 50 remaining, got %d", b.Remaining())
	}
	if b.CanFit(60) {
		t.Error("Should not fit 60")
	}
}

func TestEstimateTokensSimple(t *testing.T) {
	if got := EstimateTokensSimple(""); got != 0 {
		t.Fatalf("EstimateTokensSimple(\"\") = %d, want 0", got)
	}
	if got := EstimateTokensSimple("abcd"); got != 1 {
		t.Fatalf("EstimateTokensSimple(\"abcd\") = %d, want 1", got)
	}
	if got := EstimateTokensSimple("abcdef"); got != 2 {
		t.Fatalf("EstimateTokensSimple(\"abcdef\") = %d, want 2", got)
	}
}

func TestTruncateSymbolsZeroBudget(t *testing.T) {
	symbols := []SymbolInfo{
		{Name: "A", Kind: "function", FilePath: "a.go"},
		{Name: "B", Kind: "type", FilePath: "b.go"},
	}
	truncated := TruncateSymbols(symbols, 0)
	if len(truncated) != len(symbols) {
		t.Fatalf("expected symbols to remain when budget <= 0")
	}
}

func TestTokenBudgetSummaryAndFormat(t *testing.T) {
	b := NewTokenBudget(1234)
	b.Allocate(10, "symbols")
	summary := b.Summary()
	if !strings.HasPrefix(summary, "Token Budget") {
		t.Fatalf("unexpected summary: %q", summary)
	}

	if got := formatTokenCount(5); got != "005" {
		t.Fatalf("formatTokenCount(5) = %q, want 005", got)
	}
	if got := formatTokenCount(1200); got != "1kk" {
		t.Fatalf("formatTokenCount(1200) = %q, want 1kk", got)
	}
}
