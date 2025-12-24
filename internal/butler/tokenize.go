package butler

import (
	"strings"
	"unicode"
)

// splitIdentifier splits a code identifier into constituent words.
// Examples:
//   - "getUserName" -> ["get", "User", "Name", "getUserName"]
//   - "get_user_name" -> ["get", "user", "name", "get_user_name"]
//   - "HTTPServer" -> ["HTTP", "Server", "HTTPServer"]
//   - "parseJSON" -> ["parse", "JSON", "parseJSON"]
func splitIdentifier(s string) []string {
	if s == "" {
		return nil
	}

	var parts []string

	// Handle snake_case and kebab-case
	if strings.Contains(s, "_") || strings.Contains(s, "-") {
		for _, part := range strings.FieldsFunc(s, func(r rune) bool {
			return r == '_' || r == '-'
		}) {
			if part != "" {
				parts = append(parts, strings.ToLower(part))
			}
		}
		// Add original as well
		parts = append(parts, s)
		return dedup(parts)
	}

	// Handle camelCase and PascalCase
	var current strings.Builder
	var prevUpper, prevLower bool

	for i, r := range s {
		isUpper := unicode.IsUpper(r)
		isLower := unicode.IsLower(r)
		isDigit := unicode.IsDigit(r)

		// Start new word on transitions
		shouldSplit := false

		if i > 0 {
			// Split on lowercase -> uppercase (camelCase boundary)
			if prevLower && isUpper {
				shouldSplit = true
			}
			// Split on uppercase -> lowercase when previous was uppercase (for "HTTPServer" -> "HTTP" + "Server")
			if prevUpper && isLower && current.Len() > 1 {
				// Save all but last char of current, start new with last char + current
				word := current.String()
				if len(word) > 1 {
					parts = append(parts, strings.ToLower(word[:len(word)-1]))
					current.Reset()
					current.WriteString(word[len(word)-1:])
				}
			}
		}

		if shouldSplit && current.Len() > 0 {
			parts = append(parts, strings.ToLower(current.String()))
			current.Reset()
		}

		current.WriteRune(r)

		prevUpper = isUpper
		prevLower = isLower || isDigit
	}

	// Add final part
	if current.Len() > 0 {
		parts = append(parts, strings.ToLower(current.String()))
	}

	// Add original if we found multiple parts
	if len(parts) > 1 {
		parts = append(parts, s)
	}

	return dedup(parts)
}

// isCodeIdentifier checks if the string looks like a code identifier
func isCodeIdentifier(s string) bool {
	if s == "" || len(s) < 2 {
		return false
	}

	// Must start with letter or underscore
	first := rune(s[0])
	if !unicode.IsLetter(first) && first != '_' {
		return false
	}

	hasUnderscore := strings.Contains(s, "_")
	hasHyphen := strings.Contains(s, "-")
	hasMixedCase := false

	var hasUpper, hasLower bool
	for _, r := range s {
		if unicode.IsUpper(r) {
			hasUpper = true
		}
		if unicode.IsLower(r) {
			hasLower = true
		}
		if hasUpper && hasLower {
			hasMixedCase = true
			break
		}
	}

	// It's a code identifier if it has underscores, hyphens, or mixed case
	return hasUnderscore || hasHyphen || hasMixedCase
}

// expandQueryTokens takes a query and expands code identifiers into their parts
func expandQueryTokens(query string) []string {
	words := strings.Fields(query)
	var result []string

	for _, word := range words {
		if isCodeIdentifier(word) {
			// Split the identifier and add all parts
			parts := splitIdentifier(word)
			result = append(result, parts...)
		} else {
			result = append(result, word)
		}
	}

	return dedup(result)
}

// dedup removes duplicates while preserving order
func dedup(items []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range items {
		lower := strings.ToLower(item)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, item)
		}
	}
	return result
}
