package butler

import (
	"strings"
	"unicode"
)

// LevenshteinDistance calculates the edit distance between two strings
func LevenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Normalize to lowercase for comparison
	a = strings.ToLower(a)
	b = strings.ToLower(b)

	// Use runes for proper Unicode handling
	aRunes := []rune(a)
	bRunes := []rune(b)

	lenA := len(aRunes)
	lenB := len(bRunes)

	// Create distance matrix
	dist := make([][]int, lenA+1)
	for i := range dist {
		dist[i] = make([]int, lenB+1)
		dist[i][0] = i
	}
	for j := 0; j <= lenB; j++ {
		dist[0][j] = j
	}

	// Fill in the rest
	for i := 1; i <= lenA; i++ {
		for j := 1; j <= lenB; j++ {
			cost := 0
			if aRunes[i-1] != bRunes[j-1] {
				cost = 1
			}
			dist[i][j] = min(
				dist[i-1][j]+1,      // deletion
				dist[i][j-1]+1,      // insertion
				dist[i-1][j-1]+cost, // substitution
			)
		}
	}

	return dist[lenA][lenB]
}

// FuzzyMatch checks if two strings are a fuzzy match within a given distance
func FuzzyMatch(query, target string, maxDistance int) bool {
	dist := LevenshteinDistance(query, target)
	return dist <= maxDistance
}

// FuzzyMatchScore returns a normalized similarity score (0-1, 1 = identical)
func FuzzyMatchScore(query, target string) float64 {
	dist := LevenshteinDistance(query, target)
	maxLen := max(len(query), len(target))
	if maxLen == 0 {
		return 1.0
	}
	return 1.0 - float64(dist)/float64(maxLen)
}

// SuggestFuzzyMatches finds terms in candidates that fuzzy-match the query
func SuggestFuzzyMatches(query string, candidates []string, maxDistance int) []FuzzyResult {
	var results []FuzzyResult

	queryLower := strings.ToLower(query)

	for _, candidate := range candidates {
		candidateLower := strings.ToLower(candidate)

		// Calculate distance
		dist := LevenshteinDistance(queryLower, candidateLower)

		if dist <= maxDistance {
			results = append(results, FuzzyResult{
				Term:     candidate,
				Distance: dist,
				Score:    FuzzyMatchScore(queryLower, candidateLower),
			})
		}
	}

	// Sort by distance (closest first)
	sortFuzzyResults(results)
	return results
}

// FuzzyResult represents a fuzzy match result
type FuzzyResult struct {
	Term     string
	Distance int
	Score    float64
}

// sortFuzzyResults sorts by distance ascending, then by score descending
func sortFuzzyResults(results []FuzzyResult) {
	for i := 1; i < len(results); i++ {
		j := i
		for j > 0 {
			if results[j].Distance < results[j-1].Distance ||
				(results[j].Distance == results[j-1].Distance && results[j].Score > results[j-1].Score) {
				results[j], results[j-1] = results[j-1], results[j]
				j--
			} else {
				break
			}
		}
	}
}

// GetMaxFuzzyDistance returns the recommended max distance for a word of given length
// Shorter words need exact matches, longer words can tolerate more typos
func GetMaxFuzzyDistance(wordLength int) int {
	switch {
	case wordLength <= 3:
		return 0 // No typos for very short words
	case wordLength <= 5:
		return 1 // 1 typo for short words
	case wordLength <= 8:
		return 2 // 2 typos for medium words
	default:
		return 3 // 3 typos for long words
	}
}

// ExpandWithFuzzyVariants generates potential fuzzy variants of a query term
// This is useful for generating FTS5 OR queries
func ExpandWithFuzzyVariants(term string, commonTerms []string) []string {
	maxDist := GetMaxFuzzyDistance(len(term))
	if maxDist == 0 {
		return []string{term}
	}

	results := []string{term}

	// Find fuzzy matches in common terms
	matches := SuggestFuzzyMatches(term, commonTerms, maxDist)
	for _, match := range matches {
		if match.Term != term {
			results = append(results, match.Term)
		}
	}

	return results
}

// CommonProgrammingTerms is a list of common programming terms for fuzzy matching
var CommonProgrammingTerms = []string{
	// Common keywords
	"function", "method", "class", "interface", "struct", "type",
	"const", "constant", "variable", "var", "let", "public", "private",
	"protected", "static", "async", "await", "return", "import", "export",
	// Common concepts
	"handler", "service", "controller", "model", "view", "component",
	"provider", "factory", "builder", "manager", "helper", "util", "utils",
	"config", "configuration", "settings", "options", "params", "args",
	"request", "response", "client", "server", "connection", "socket",
	"database", "query", "result", "record", "entity", "schema",
	"error", "exception", "message", "event", "callback", "promise",
	"user", "auth", "authentication", "authorization", "session", "token",
	"file", "path", "directory", "buffer", "stream", "reader", "writer",
	"parse", "parser", "format", "formatter", "encode", "decode", "serialize",
	"cache", "store", "storage", "memory", "state", "context",
	"test", "spec", "mock", "stub", "fixture", "assert", "expect",
	"logger", "logging", "debug", "info", "warn", "error",
	"create", "read", "update", "delete", "get", "set", "add", "remove",
	"init", "initialize", "start", "stop", "run", "execute", "process",
	"validate", "check", "verify", "compare", "match", "filter", "search",
	// Common abbreviations
	"http", "https", "api", "url", "uri", "json", "xml", "html", "css",
	"sql", "db", "id", "uuid", "guid", "ref", "ptr", "fn", "cb",
}

// NormalizeForFuzzy normalizes a string for fuzzy comparison
func NormalizeForFuzzy(s string) string {
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(unicode.ToLower(r))
		}
	}
	return result.String()
}

func min(values ...int) int {
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
