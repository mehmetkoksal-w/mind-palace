package butler

import (
	"strings"
)

// preprocessQuery transforms a user query into an FTS5 query.
func preprocessQuery(query string) string {
	return preprocessQueryWithOptions(query, true) // Enable synonyms by default
}

// preprocessQueryWithFuzzy expands query with fuzzy variants.
func preprocessQueryWithFuzzy(query string) string {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return ""
	}

	// First expand tokens (CamelCase, snake_case)
	tokens := expandQueryTokens(trimmed)
	if len(tokens) == 0 {
		return "\"" + strings.ReplaceAll(trimmed, "\"", "\"\"") + "\""
	}

	// Expand with synonyms
	tokens = expandWithSynonyms(tokens)

	// Add fuzzy variants for each token
	var allTerms []string
	for _, token := range tokens {
		if len(token) < 2 {
			continue
		}

		// Add the original token
		allTerms = append(allTerms, "\""+strings.ReplaceAll(token, "\"", "\"\"")+"\"*")

		// Add fuzzy variants for longer words
		if len(token) >= 5 {
			fuzzyVariants := ExpandWithFuzzyVariants(token, CommonProgrammingTerms)
			for _, variant := range fuzzyVariants {
				if variant != token {
					allTerms = append(allTerms, "\""+strings.ReplaceAll(variant, "\"", "\"\"")+"\"*")
				}
			}
		}
	}

	if len(allTerms) == 0 {
		return "\"" + strings.ReplaceAll(trimmed, "\"", "\"\"") + "\""
	}

	return strings.Join(allTerms, " OR ")
}

// preprocessQueryWithOptions transforms a query with configurable options.
func preprocessQueryWithOptions(query string, useSynonyms bool) string {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return ""
	}

	// Check if query looks like code (exact match needed)
	isExactCodeQuery := strings.ContainsAny(trimmed, ".()[]{}") ||
		strings.Contains(trimmed, "::") ||
		strings.Contains(trimmed, "->")

	if isExactCodeQuery {
		// Exact phrase search with quote escaping
		return "\"" + strings.ReplaceAll(trimmed, "\"", "\"\"") + "\""
	}

	// Expand code identifiers (CamelCase, snake_case) into their parts
	expandedTokens := expandQueryTokens(trimmed)
	if len(expandedTokens) == 0 {
		return "\"" + strings.ReplaceAll(trimmed, "\"", "\"\"") + "\""
	}

	// Optionally expand with programming synonyms
	if useSynonyms {
		expandedTokens = expandWithSynonyms(expandedTokens)
	}

	var terms []string
	for _, word := range expandedTokens {
		if len(word) < 2 {
			continue
		}
		// Escape and add prefix operator
		terms = append(terms, "\""+strings.ReplaceAll(word, "\"", "\"\"")+"\"*")
	}

	if len(terms) == 0 {
		return "\"" + strings.ReplaceAll(trimmed, "\"", "\"\"") + "\""
	}

	return strings.Join(terms, " OR ")
}

// decodeJSONCFile decodes a JSONC file using the configured decoder.
func decodeJSONCFile(path string, v interface{}) error {
	return jsonCDecode(path, v)
}

var jsonCDecode func(path string, v interface{}) error

// SetJSONCDecoder sets the JSONC decoder function.
func SetJSONCDecoder(fn func(path string, v interface{}) error) {
	jsonCDecode = fn
}
