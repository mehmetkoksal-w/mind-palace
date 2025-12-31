package butler

import (
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// SearchResult represents a single search hit.
type SearchResult struct {
	Path       string  `json:"path"`
	Room       string  `json:"room,omitempty"`
	ChunkIndex int     `json:"chunkIndex"`
	StartLine  int     `json:"startLine"`
	EndLine    int     `json:"endLine"`
	Snippet    string  `json:"snippet"`
	Score      float64 `json:"score"`
	IsEntry    bool    `json:"isEntry,omitempty"`
}

// GroupedResults groups search results by room.
type GroupedResults struct {
	Room    string         `json:"room"`
	Summary string         `json:"summary,omitempty"`
	Results []SearchResult `json:"results"`
}

// SearchOptions configures search behavior.
type SearchOptions struct {
	Limit      int    // Maximum results (default 20)
	RoomFilter string // Optional: filter to specific room
	FuzzyMatch bool   // Enable fuzzy matching for typo tolerance
}

// EnhancedContextOptions configures memory-aware context assembly.
type EnhancedContextOptions struct {
	Query            string `json:"query"`
	Limit            int    `json:"limit"`
	MaxTokens        int    `json:"maxTokens"`
	IncludeTests     bool   `json:"includeTests"`
	IncludeLearnings bool   `json:"includeLearnings"`
	IncludeFileIntel bool   `json:"includeFileIntel"`
	IncludeIdeas     bool   `json:"includeIdeas"`
	IncludeDecisions bool   `json:"includeDecisions"`
	SessionID        string `json:"sessionId,omitempty"`
}

// EnhancedContextResult includes code context plus memory data.
type EnhancedContextResult struct {
	*index.ContextResult

	// Memory-enhanced data
	Learnings         []memory.Learning            `json:"learnings,omitempty"`
	FileIntel         map[string]*memory.FileIntel `json:"fileIntel,omitempty"`
	Conflict          *memory.Conflict             `json:"conflict,omitempty"`
	BrainIdeas        []memory.Idea                `json:"brainIdeas,omitempty"`
	BrainDecisions    []memory.Decision            `json:"brainDecisions,omitempty"`
	RelatedLinks      []memory.Link                `json:"relatedLinks,omitempty"`
	DecisionConflicts []memory.DecisionConflict    `json:"decisionConflicts,omitempty"`
}

// BriefingResult contains a comprehensive briefing.
type BriefingResult struct {
	FilePath       string               `json:"filePath,omitempty"`
	ActiveAgents   []memory.ActiveAgent `json:"activeAgents,omitempty"`
	Conflict       *memory.Conflict     `json:"conflict,omitempty"`
	FileIntel      *memory.FileIntel    `json:"fileIntel,omitempty"`
	Learnings      []memory.Learning    `json:"learnings,omitempty"`
	Hotspots       []memory.FileIntel   `json:"hotspots,omitempty"`
	BrainIdeas     []memory.Idea        `json:"brainIdeas,omitempty"`
	BrainDecisions []memory.Decision    `json:"brainDecisions,omitempty"`
}

// IndexInfo contains information about the code index.
type IndexInfo struct {
	FileCount int       `json:"fileCount"`
	LastScan  time.Time `json:"lastScan"`
	Status    string    `json:"status"` // "fresh", "stale", "scanning"
}

// AutoInjectedContext contains context automatically assembled for AI agents.
type AutoInjectedContext struct {
	FilePath    string                `json:"filePath"`
	Room        string                `json:"room,omitempty"`
	Learnings   []PrioritizedLearning `json:"learnings,omitempty"`
	Decisions   []memory.Decision     `json:"decisions,omitempty"`
	Failures    []FileFailure         `json:"failures,omitempty"`
	Warnings    []ContextWarning      `json:"warnings,omitempty"`
	TotalTokens int                   `json:"totalTokens"`
	GeneratedAt time.Time             `json:"generatedAt"`
}

// PrioritizedLearning wraps a learning with priority and explanation.
type PrioritizedLearning struct {
	Learning memory.Learning `json:"learning"`
	Priority float64         `json:"priority"` // Computed from confidence + recency + scope match
	Reason   string          `json:"reason"`   // Why this was included
}

// FileFailure contains failure information for a file.
type FileFailure struct {
	Path         string `json:"path"`
	FailureCount int    `json:"failureCount"`
	LastFailure  string `json:"lastFailure,omitempty"`
	Severity     string `json:"severity"` // "low", "medium", "high"
}

// ContextWarning represents a warning to show in context.
type ContextWarning struct {
	Type    string `json:"type"`    // "contradiction", "decay_risk", "stale", "unreviewed_decision"
	Message string `json:"message"`
	ID      string `json:"id,omitempty"`      // Related record ID
	Details string `json:"details,omitempty"` // Additional details
}

// ScopeExplanation explains the scope inheritance for a file.
type ScopeExplanation struct {
	FilePath         string           `json:"filePath"`
	ResolvedRoom     string           `json:"resolvedRoom"`
	InheritanceChain []ScopeLevel     `json:"inheritanceChain"`
	TotalRecords     map[string]int   `json:"totalRecords"` // Per scope level
	EffectiveRules   []InheritanceRule `json:"effectiveRules"`
}

// ScopeLevel represents one level in the scope hierarchy.
type ScopeLevel struct {
	Scope       string `json:"scope"`       // "file", "room", "palace", "corridor"
	Path        string `json:"path"`        // Scope path
	RecordCount int    `json:"recordCount"`
	Active      bool   `json:"active"` // Whether inheritance is enabled
}

// InheritanceRule represents a scope inheritance rule.
type InheritanceRule struct {
	Pattern string `json:"pattern"` // e.g., "*.test.ts"
	Scope   string `json:"scope"`
	Action  string `json:"action"` // "include", "exclude"
}

// ScopeHierarchyView provides full hierarchy data.
type ScopeHierarchyView struct {
	Levels []ScopeLevelDetail `json:"levels"`
}

// ScopeLevelDetail contains records at a scope level.
type ScopeLevelDetail struct {
	Scope     string            `json:"scope"`
	Learnings []memory.Learning `json:"learnings,omitempty"`
	Decisions []memory.Decision `json:"decisions,omitempty"`
	Ideas     []memory.Idea     `json:"ideas,omitempty"`
}
