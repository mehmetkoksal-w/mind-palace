package memory

import "strings"

// Authority represents the authoritative state of a record.
// This is the single source of truth for authority resolution.
type Authority string

const (
	// AuthorityProposed indicates a record proposed by an agent, not yet approved.
	AuthorityProposed Authority = "proposed"
	// AuthorityApproved indicates a record approved by a human.
	AuthorityApproved Authority = "approved"
	// AuthorityLegacyApproved indicates a pre-governance record, treated as approved.
	AuthorityLegacyApproved Authority = "legacy_approved"
)

// ValidAuthorities is the complete set of valid authority values.
var ValidAuthorities = []Authority{
	AuthorityProposed,
	AuthorityApproved,
	AuthorityLegacyApproved,
}

// IsAuthoritative returns true if the authority value represents
// trusted, human-approved state. ALL queries filtering for authoritative
// records must use this helper.
func IsAuthoritative(auth Authority) bool {
	return auth == AuthorityApproved || auth == AuthorityLegacyApproved
}

// AuthoritativeValues returns the list of authority values that represent
// authoritative state. Use this for SQL IN clauses.
// Queries MUST call this, not hard-code values.
func AuthoritativeValues() []Authority {
	return []Authority{AuthorityApproved, AuthorityLegacyApproved}
}

// AuthoritativeValuesStrings returns AuthoritativeValues as strings for SQL queries.
func AuthoritativeValuesStrings() []string {
	vals := AuthoritativeValues()
	result := make([]string, len(vals))
	for i, v := range vals {
		result[i] = string(v)
	}
	return result
}

// SQLPlaceholders returns a string of SQL placeholders for the given count.
// Example: SQLPlaceholders(3) returns "?, ?, ?"
func SQLPlaceholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.Repeat("?, ", count-1) + "?"
}
