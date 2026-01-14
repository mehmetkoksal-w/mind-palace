package butler

import (
	"sort"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// RouteRuleVersion is the version of the route derivation rules.
// Increment this when the derivation logic changes.
const RouteRuleVersion = "v1.0"

// RouteNodeKind represents the type of node in a route.
type RouteNodeKind string

const (
	RouteNodeKindRoom     RouteNodeKind = "room"
	RouteNodeKindDecision RouteNodeKind = "decision"
	RouteNodeKindLearning RouteNodeKind = "learning"
	RouteNodeKindFile     RouteNodeKind = "file"
)

// RouteNode represents a single node in the navigation route.
type RouteNode struct {
	Order    int           `json:"order"`
	Kind     RouteNodeKind `json:"kind"`
	ID       string        `json:"id"`
	Reason   string        `json:"reason"`
	FetchRef string        `json:"fetch_ref"`
}

// RouteResult contains the complete route derivation result.
type RouteResult struct {
	Nodes []RouteNode `json:"nodes"`
	Meta  RouteMeta   `json:"meta"`
}

// RouteMeta contains metadata about the route derivation.
type RouteMeta struct {
	RuleVersion string `json:"rule_version"`
	NodeCount   int    `json:"node_count"`
}

// RouteConfig configures the route derivation.
type RouteConfig struct {
	// MaxNodes is the maximum number of nodes to return (default: 10).
	MaxNodes int

	// MinLearningConfidence is the minimum confidence for including learnings (default: 0.7).
	MinLearningConfidence float64
}

// DefaultRouteConfig returns the default configuration.
func DefaultRouteConfig() *RouteConfig {
	return &RouteConfig{
		MaxNodes:              10,
		MinLearningConfidence: 0.7,
	}
}

// scoredNode wraps a RouteNode with a score for sorting.
type scoredNode struct {
	node  RouteNode
	score float64
}

// GetRoute derives a navigation route based on intent and scope.
// The derivation is deterministic: same inputs always produce same outputs.
//
// Derivation rules:
// 1. Match intent keywords to room names/summaries
// 2. Include relevant decisions from scope chain
// 3. Include high-confidence learnings (>=0.7)
// 4. Max 10 nodes, deterministic ordering
func (b *Butler) GetRoute(intent string, scope memory.Scope, scopePath string, cfg *RouteConfig) (*RouteResult, error) {
	if cfg == nil {
		cfg = DefaultRouteConfig()
	}

	var candidates []scoredNode
	intentLower := strings.ToLower(intent)
	intentWords := strings.Fields(intentLower)

	// Rule 1: Match rooms
	candidates = append(candidates, b.matchRooms(intentWords)...)

	// Rule 2: Include relevant decisions from scope chain
	if b.memory != nil {
		candidates = append(candidates, b.matchDecisions(scope, scopePath, intentWords)...)
	}

	// Rule 3: Include high-confidence learnings
	if b.memory != nil {
		candidates = append(candidates, b.matchLearnings(scope, scopePath, intentWords, cfg.MinLearningConfidence)...)
	}

	// Rule 4: Include scope file if provided
	if scope == memory.ScopeFile && scopePath != "" {
		candidates = append(candidates, scoredNode{
			node: RouteNode{
				Kind:     RouteNodeKindFile,
				ID:       scopePath,
				Reason:   "Specified scope file",
				FetchRef: "explore_file --file " + scopePath,
			},
			score: 0.5, // Mid-priority for the scope file
		})
	}

	// Sort by score descending (deterministic: ties broken by ID)
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].node.ID < candidates[j].node.ID
	})

	// Deduplicate by ID
	seen := make(map[string]bool)
	var unique []scoredNode
	for _, c := range candidates {
		key := string(c.node.Kind) + ":" + c.node.ID
		if !seen[key] {
			seen[key] = true
			unique = append(unique, c)
		}
	}

	// Limit to MaxNodes
	if len(unique) > cfg.MaxNodes {
		unique = unique[:cfg.MaxNodes]
	}

	// Build final nodes with order
	nodes := make([]RouteNode, len(unique))
	for i, c := range unique {
		nodes[i] = c.node
		nodes[i].Order = i + 1
	}

	return &RouteResult{
		Nodes: nodes,
		Meta: RouteMeta{
			RuleVersion: RouteRuleVersion,
			NodeCount:   len(nodes),
		},
	}, nil
}

// matchRooms matches intent keywords against room names and summaries.
func (b *Butler) matchRooms(intentWords []string) []scoredNode {
	var results []scoredNode

	for name, room := range b.rooms {
		nameLower := strings.ToLower(name)
		summaryLower := strings.ToLower(room.Summary)

		score := 0.0
		var reasons []string

		// Check name match
		for _, word := range intentWords {
			if strings.Contains(nameLower, word) {
				score += 1.0
				reasons = append(reasons, "Room name matches '"+word+"'")
			}
		}

		// Check summary match
		for _, word := range intentWords {
			if strings.Contains(summaryLower, word) {
				score += 0.5
				reasons = append(reasons, "Room summary matches '"+word+"'")
			}
		}

		if score > 0 {
			reason := "Room name matches intent"
			if len(reasons) > 0 {
				reason = reasons[0]
			}

			results = append(results, scoredNode{
				node: RouteNode{
					Kind:     RouteNodeKindRoom,
					ID:       name,
					Reason:   reason,
					FetchRef: "explore_rooms",
				},
				score: score,
			})

			// Also add entry points as file nodes
			for _, entry := range room.EntryPoints {
				results = append(results, scoredNode{
					node: RouteNode{
						Kind:     RouteNodeKindFile,
						ID:       entry,
						Reason:   "Room entry point",
						FetchRef: "explore_file --file " + entry,
					},
					score: score * 0.8, // Slightly lower than room
				})
			}
		}
	}

	return results
}

// matchDecisions matches authoritative decisions by content against intent.
func (b *Butler) matchDecisions(scope memory.Scope, scopePath string, intentWords []string) []scoredNode {
	var results []scoredNode

	// Query authoritative decisions across scope chain
	cfg := &memory.AuthoritativeQueryConfig{
		MaxDecisions:      20, // Query more, filter by relevance
		MaxLearnings:      0,
		MaxContentLen:     1000,
		AuthoritativeOnly: true,
	}

	state, err := b.memory.GetAuthoritativeState(scope, scopePath, b.resolveRoom, cfg)
	if err != nil {
		return results
	}

	for _, sd := range state.Decisions {
		contentLower := strings.ToLower(sd.Decision.Content)
		rationaleLower := strings.ToLower(sd.Decision.Rationale)

		score := 0.0
		var reasons []string

		// Check content match
		for _, word := range intentWords {
			if strings.Contains(contentLower, word) {
				score += 0.8
				reasons = append(reasons, "Decision content matches '"+word+"'")
			}
			if strings.Contains(rationaleLower, word) {
				score += 0.3
				reasons = append(reasons, "Decision rationale matches '"+word+"'")
			}
		}

		if score > 0 {
			reason := "Decision content matches intent"
			if len(reasons) > 0 {
				reason = reasons[0]
			}

			results = append(results, scoredNode{
				node: RouteNode{
					Kind:     RouteNodeKindDecision,
					ID:       sd.Decision.ID,
					Reason:   reason,
					FetchRef: "recall_decisions --id " + sd.Decision.ID,
				},
				score: score,
			})
		}
	}

	return results
}

// matchLearnings matches high-confidence authoritative learnings by content against intent.
func (b *Butler) matchLearnings(scope memory.Scope, scopePath string, intentWords []string, minConfidence float64) []scoredNode {
	var results []scoredNode

	// Query authoritative learnings across scope chain
	cfg := &memory.AuthoritativeQueryConfig{
		MaxDecisions:      0,
		MaxLearnings:      20, // Query more, filter by relevance
		MaxContentLen:     1000,
		AuthoritativeOnly: true,
	}

	state, err := b.memory.GetAuthoritativeState(scope, scopePath, b.resolveRoom, cfg)
	if err != nil {
		return results
	}

	for _, sl := range state.Learnings {
		// Filter by confidence threshold
		if sl.Learning.Confidence < minConfidence {
			continue
		}

		contentLower := strings.ToLower(sl.Learning.Content)

		score := 0.0
		var reasons []string

		// Check content match
		for _, word := range intentWords {
			if strings.Contains(contentLower, word) {
				score += 0.6
				reasons = append(reasons, "Learning content matches '"+word+"'")
			}
		}

		// Boost by confidence
		score += sl.Learning.Confidence * 0.4

		if score > 0 {
			confidencePercent := int(sl.Learning.Confidence * 100)
			reason := "High-confidence learning"
			if len(reasons) > 0 {
				reason = reasons[0]
			}
			reason += " (" + string(rune('0'+confidencePercent/10)) + string(rune('0'+confidencePercent%10)) + "%)"

			results = append(results, scoredNode{
				node: RouteNode{
					Kind:     RouteNodeKindLearning,
					ID:       sl.Learning.ID,
					Reason:   reason,
					FetchRef: "recall --id " + sl.Learning.ID,
				},
				score: score,
			})
		}
	}

	return results
}
