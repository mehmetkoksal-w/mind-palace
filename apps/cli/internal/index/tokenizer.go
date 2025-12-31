package index

import (
	"sort"
	"strings"
	"unicode"
)

// TokenEstimate represents a token count estimate for text
type TokenEstimate struct {
	Text       string
	TokenCount int
}

// EstimateTokens returns an approximate token count for the given text.
// Uses a conservative estimate of ~4 characters per token for code,
// which accounts for whitespace compression and common token patterns.
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	// For code, we use a more nuanced approach:
	// - Whitespace-only runs compress well
	// - Identifiers and keywords are often single tokens
	// - Punctuation is usually individual tokens

	tokens := 0
	inWord := false
	wordLen := 0

	for _, r := range text {
		if unicode.IsSpace(r) {
			if inWord {
				// End of word - estimate tokens for it
				tokens += estimateWordTokens(wordLen)
				inWord = false
				wordLen = 0
			}
			// Whitespace: multiple spaces often compress to 1-2 tokens
			// We'll count significant whitespace
			if r == '\n' {
				tokens++ // Newlines are usually separate tokens
			}
		} else if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			if inWord {
				tokens += estimateWordTokens(wordLen)
				inWord = false
				wordLen = 0
			}
			tokens++ // Punctuation/symbols are usually individual tokens
		} else {
			inWord = true
			wordLen++
		}
	}

	// Handle trailing word
	if inWord {
		tokens += estimateWordTokens(wordLen)
	}

	return tokens
}

// estimateWordTokens estimates tokens for a word of given length
func estimateWordTokens(length int) int {
	if length == 0 {
		return 0
	}
	if length <= 4 {
		return 1 // Short words are usually single tokens
	}
	if length <= 8 {
		return 2 // Medium words might be 1-2 tokens
	}
	// Longer words: ~4 chars per token
	return (length + 3) / 4
}

// EstimateTokensSimple uses a simple character-based estimate
// (~4 characters per token, which is conservative for code)
func EstimateTokensSimple(text string) int {
	if text == "" {
		return 0
	}
	return (len(text) + 3) / 4
}

// BudgetedItem represents an item with its token cost
type BudgetedItem struct {
	Item       interface{}
	TokenCount int
	Priority   float64 // Higher = more important
}

// TruncateToTokenBudget takes a slice of items with token counts and priorities,
// and returns items that fit within the budget, prioritizing higher-priority items.
func TruncateToTokenBudget(items []BudgetedItem, budget int) []BudgetedItem {
	if budget <= 0 {
		return nil
	}

	// Sort by priority (descending)
	sorted := make([]BudgetedItem, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority > sorted[j].Priority
	})

	var result []BudgetedItem
	usedTokens := 0

	for _, item := range sorted {
		if usedTokens+item.TokenCount <= budget {
			result = append(result, item)
			usedTokens += item.TokenCount
		}
	}

	return result
}

// TruncateSymbols truncates symbols to fit within a token budget
func TruncateSymbols(symbols []SymbolInfo, budget int) []SymbolInfo {
	if budget <= 0 || len(symbols) == 0 {
		return symbols
	}

	budgeted := make([]BudgetedItem, len(symbols))
	for i, sym := range symbols {
		// Estimate tokens for symbol representation
		symText := sym.Name + " " + sym.Kind + " " + sym.FilePath
		if sym.Signature != "" {
			symText += " " + sym.Signature
		}
		tokenCount := EstimateTokens(symText)

		// Priority based on symbol kind (functions/classes are more important)
		priority := 1.0
		switch strings.ToLower(sym.Kind) {
		case "function", "method":
			priority = 3.0
		case "class", "struct", "interface":
			priority = 2.5
		case "type":
			priority = 2.0
		case "constant", "variable":
			priority = 1.5
		}

		budgeted[i] = BudgetedItem{
			Item:       sym,
			TokenCount: tokenCount,
			Priority:   priority,
		}
	}

	truncated := TruncateToTokenBudget(budgeted, budget)

	result := make([]SymbolInfo, len(truncated))
	for i, b := range truncated {
		result[i] = b.Item.(SymbolInfo)
	}

	return result
}

// TokenBudget tracks token usage across multiple categories
type TokenBudget struct {
	Total    int
	Used     int
	Symbols  int
	Chunks   int
	Metadata int
}

// NewTokenBudget creates a new token budget with the given total
func NewTokenBudget(total int) *TokenBudget {
	return &TokenBudget{Total: total}
}

// Remaining returns the remaining token budget
func (tb *TokenBudget) Remaining() int {
	return tb.Total - tb.Used
}

// CanFit returns true if the given token count fits in the budget
func (tb *TokenBudget) CanFit(tokens int) bool {
	return tb.Used+tokens <= tb.Total
}

// Allocate tries to allocate tokens and returns true if successful
func (tb *TokenBudget) Allocate(tokens int, category string) bool {
	if !tb.CanFit(tokens) {
		return false
	}
	tb.Used += tokens
	switch category {
	case "symbols":
		tb.Symbols += tokens
	case "chunks":
		tb.Chunks += tokens
	case "metadata":
		tb.Metadata += tokens
	}
	return true
}

// Summary returns a summary of token usage
func (tb *TokenBudget) Summary() string {
	return strings.Join([]string{
		"Token Budget:",
		"  Total:    " + formatTokenCount(tb.Total),
		"  Used:     " + formatTokenCount(tb.Used),
		"  Symbols:  " + formatTokenCount(tb.Symbols),
		"  Chunks:   " + formatTokenCount(tb.Chunks),
		"  Metadata: " + formatTokenCount(tb.Metadata),
		"  Remaining:" + formatTokenCount(tb.Remaining()),
	}, "\n")
}

func formatTokenCount(n int) string {
	if n >= 1000 {
		return strings.TrimRight(strings.TrimRight(
			strings.Replace(
				string(rune('0'+n/1000))+"k",
				"0k", "", 1,
			), "0"), ".") + "k"
	}
	return string(rune('0'+n/100)) + string(rune('0'+n/10%10)) + string(rune('0'+n%10))
}
