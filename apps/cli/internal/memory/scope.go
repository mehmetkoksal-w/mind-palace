package memory

// Scope represents a scope level in the hierarchy.
type Scope string

const (
	// ScopeFile represents file-level scope (most specific).
	ScopeFile Scope = "file"
	// ScopeRoom represents room-level scope.
	ScopeRoom Scope = "room"
	// ScopePalace represents palace-level scope (most general).
	ScopePalace Scope = "palace"
)

// ValidScopes is the ordered list of scopes from most specific to most general.
var ValidScopes = []Scope{ScopeFile, ScopeRoom, ScopePalace}

// ScopeLevel represents a single level in the scope hierarchy.
type ScopeLevel struct {
	Scope    Scope  // The scope type
	Path     string // The scope path (file path, room name, or empty for palace)
	Priority int    // Lower is higher priority (file=0, room=1, palace=2)
}

// ExpandScope expands a starting scope into a deterministic inheritance chain.
// The chain always goes: file -> room -> palace (when applicable).
// This is the single source of truth for scope expansion logic.
//
// Parameters:
//   - scope: The starting scope type ("file", "room", or "palace")
//   - scopePath: The path for the scope (file path, room name, or empty)
//   - roomResolver: Optional function to resolve a file path to a room name.
//     If nil, room inheritance is skipped for file scopes.
//
// Returns an ordered slice of ScopeLevel from most specific to most general.
// The chain is deterministic: same inputs always produce same outputs.
func ExpandScope(scope Scope, scopePath string, roomResolver func(string) string) []ScopeLevel {
	var chain []ScopeLevel

	switch scope {
	case ScopeFile:
		// File scope: file -> room (if resolvable) -> palace
		chain = append(chain, ScopeLevel{
			Scope:    ScopeFile,
			Path:     scopePath,
			Priority: 0,
		})

		// Try to resolve room from file path
		if roomResolver != nil && scopePath != "" {
			room := roomResolver(scopePath)
			if room != "" {
				chain = append(chain, ScopeLevel{
					Scope:    ScopeRoom,
					Path:     room,
					Priority: 1,
				})
			}
		}

		// Always include palace
		chain = append(chain, ScopeLevel{
			Scope:    ScopePalace,
			Path:     "",
			Priority: 2,
		})

	case ScopeRoom:
		// Room scope: room -> palace
		chain = append(chain, ScopeLevel{
			Scope:    ScopeRoom,
			Path:     scopePath,
			Priority: 1,
		})
		chain = append(chain, ScopeLevel{
			Scope:    ScopePalace,
			Path:     "",
			Priority: 2,
		})

	case ScopePalace:
		// Palace scope: just palace
		chain = append(chain, ScopeLevel{
			Scope:    ScopePalace,
			Path:     "",
			Priority: 2,
		})

	default:
		// Unknown scope: treat as palace
		chain = append(chain, ScopeLevel{
			Scope:    ScopePalace,
			Path:     "",
			Priority: 2,
		})
	}

	return chain
}

// AuthoritativeQueryConfig configures bounded queries for authoritative state.
// This ensures deterministic, bounded context assembly.
type AuthoritativeQueryConfig struct {
	// MaxDecisions is the maximum number of decisions to return.
	// Default: 10
	MaxDecisions int

	// MaxLearnings is the maximum number of learnings to return.
	// Default: 10
	MaxLearnings int

	// MaxContentLen is the maximum characters per content item.
	// Content exceeding this limit is truncated with "...".
	// Default: 500
	MaxContentLen int

	// AuthoritativeOnly when true, only returns approved/legacy_approved records.
	// Default: true
	AuthoritativeOnly bool
}

// DefaultAuthoritativeQueryConfig returns the default configuration.
func DefaultAuthoritativeQueryConfig() *AuthoritativeQueryConfig {
	return &AuthoritativeQueryConfig{
		MaxDecisions:      10,
		MaxLearnings:      10,
		MaxContentLen:     500,
		AuthoritativeOnly: true,
	}
}

// TruncateContent truncates content to the configured maximum length.
// If truncated, appends "..." to indicate truncation.
// This is deterministic: same input always produces same output.
func (c *AuthoritativeQueryConfig) TruncateContent(content string) string {
	if c.MaxContentLen <= 0 || len(content) <= c.MaxContentLen {
		return content
	}

	// Truncate and add ellipsis
	// Account for ellipsis length to stay within limit
	if c.MaxContentLen <= 3 {
		return "..."
	}
	return content[:c.MaxContentLen-3] + "..."
}

// ScopedQueryResult contains the result of a scoped authoritative query.
type ScopedQueryResult struct {
	// ScopeChain is the scope hierarchy that was queried.
	ScopeChain []ScopeLevel

	// Decisions grouped by scope level.
	Decisions []ScopedDecision

	// Learnings grouped by scope level.
	Learnings []ScopedLearning

	// TotalDecisions is the count before limiting.
	TotalDecisions int

	// TotalLearnings is the count before limiting.
	TotalLearnings int

	// Truncated indicates whether any content was truncated.
	Truncated bool
}

// ScopedDecision is a decision with its source scope.
type ScopedDecision struct {
	Decision    Decision
	SourceScope ScopeLevel
}

// ScopedLearning is a learning with its source scope.
type ScopedLearning struct {
	Learning    Learning
	SourceScope ScopeLevel
}

// GetAuthoritativeState queries for authoritative decisions and learnings
// across the scope chain, respecting the configuration bounds.
func (m *Memory) GetAuthoritativeState(
	startScope Scope,
	scopePath string,
	roomResolver func(string) string,
	cfg *AuthoritativeQueryConfig,
) (*ScopedQueryResult, error) {
	if cfg == nil {
		cfg = DefaultAuthoritativeQueryConfig()
	}

	// Expand scope into inheritance chain
	chain := ExpandScope(startScope, scopePath, roomResolver)

	result := &ScopedQueryResult{
		ScopeChain: chain,
	}

	// Collect decisions across scope chain
	seenDecisions := make(map[string]bool)
	for _, level := range chain {
		if len(result.Decisions) >= cfg.MaxDecisions {
			break
		}

		remaining := cfg.MaxDecisions - len(result.Decisions)
		decisions, err := m.GetDecisionsWithAuthority(
			"active", "", string(level.Scope), level.Path, remaining, cfg.AuthoritativeOnly,
		)
		if err != nil {
			continue // Skip scope on error, don't fail entire query
		}

		for _, d := range decisions {
			if seenDecisions[d.ID] {
				continue
			}
			seenDecisions[d.ID] = true

			// Apply content truncation
			originalLen := len(d.Content)
			d.Content = cfg.TruncateContent(d.Content)
			if len(d.Content) < originalLen {
				result.Truncated = true
			}

			result.Decisions = append(result.Decisions, ScopedDecision{
				Decision:    d,
				SourceScope: level,
			})

			if len(result.Decisions) >= cfg.MaxDecisions {
				break
			}
		}
	}
	result.TotalDecisions = len(seenDecisions)

	// Collect learnings across scope chain
	seenLearnings := make(map[string]bool)
	for _, level := range chain {
		if len(result.Learnings) >= cfg.MaxLearnings {
			break
		}

		remaining := cfg.MaxLearnings - len(result.Learnings)
		learnings, err := m.GetLearningsWithAuthority(
			string(level.Scope), level.Path, remaining, cfg.AuthoritativeOnly,
		)
		if err != nil {
			continue // Skip scope on error, don't fail entire query
		}

		for _, l := range learnings {
			if seenLearnings[l.ID] {
				continue
			}
			seenLearnings[l.ID] = true

			// Apply content truncation
			originalLen := len(l.Content)
			l.Content = cfg.TruncateContent(l.Content)
			if len(l.Content) < originalLen {
				result.Truncated = true
			}

			result.Learnings = append(result.Learnings, ScopedLearning{
				Learning:    l,
				SourceScope: level,
			})

			if len(result.Learnings) >= cfg.MaxLearnings {
				break
			}
		}
	}
	result.TotalLearnings = len(seenLearnings)

	return result, nil
}
