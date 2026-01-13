package butler

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// GetContextForTask returns complete context for a task - the Oracle query.
func (b *Butler) GetContextForTask(query string, limit int) (*index.ContextResult, error) {
	return index.GetContextForTask(b.db, query, limit)
}

// GetContextForTaskWithOptions returns context with custom options including token budgeting.
func (b *Butler) GetContextForTaskWithOptions(query string, limit, maxTokens int, includeTests bool) (*index.ContextResult, error) {
	opts := &index.ContextOptions{
		MaxTokens:    maxTokens,
		IncludeTests: includeTests,
	}
	return index.GetContextForTaskWithOptions(b.db, query, limit, opts)
}

// GetEnhancedContext returns context enriched with learnings and file intel.
func (b *Butler) GetEnhancedContext(opts EnhancedContextOptions) (*EnhancedContextResult, error) {
	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	// Get base code context
	codeOpts := &index.ContextOptions{
		MaxTokens:    opts.MaxTokens,
		IncludeTests: opts.IncludeTests,
	}
	codeContext, err := index.GetContextForTaskWithOptions(b.db, opts.Query, opts.Limit, codeOpts)
	if err != nil {
		return nil, fmt.Errorf("get code context: %w", err)
	}

	result := &EnhancedContextResult{
		ContextResult: codeContext,
	}

	// If no memory available, return code context only
	if b.memory == nil {
		return result, nil
	}

	// Get relevant learnings
	if opts.IncludeLearnings {
		learnings, err := b.memory.GetRelevantLearnings("", opts.Query, 10)
		if err == nil && len(learnings) > 0 {
			result.Learnings = learnings
		}
	}

	// Get file intel for files in context
	if opts.IncludeFileIntel && len(codeContext.Files) > 0 {
		result.FileIntel = make(map[string]*memory.FileIntel)
		for _, f := range codeContext.Files {
			intel, err := b.memory.GetFileIntel(f.Path)
			if err == nil && intel != nil && intel.EditCount > 0 {
				result.FileIntel[f.Path] = intel
			}
		}
	}

	// Get relevant ideas from brain
	if opts.IncludeIdeas {
		ideas, err := b.memory.SearchIdeas(opts.Query, 5)
		if err == nil && len(ideas) > 0 {
			result.BrainIdeas = ideas
		}
	}

	// Get relevant decisions from brain
	if opts.IncludeDecisions {
		decisions, err := b.memory.SearchDecisions(opts.Query, 5)
		if err == nil && len(decisions) > 0 {
			result.BrainDecisions = decisions
		}
	}

	// Get related links for brain ideas and decisions
	if len(result.BrainIdeas) > 0 || len(result.BrainDecisions) > 0 {
		linkMap := make(map[string]bool) // dedupe links
		var allLinks []memory.Link

		// Get links for ideas
		for i := range result.BrainIdeas {
			idea := &result.BrainIdeas[i]
			links, err := b.memory.GetAllLinksFor(idea.ID)
			if err == nil {
				for j := range links {
					link := &links[j]
					if !linkMap[link.ID] {
						linkMap[link.ID] = true
						allLinks = append(allLinks, *link)
					}
				}
			}
		}

		// Get links for decisions
		for i := range result.BrainDecisions {
			decision := &result.BrainDecisions[i]
			links, err := b.memory.GetAllLinksFor(decision.ID)
			if err == nil {
				for j := range links {
					link := &links[j]
					if !linkMap[link.ID] {
						linkMap[link.ID] = true
						allLinks = append(allLinks, *link)
					}
				}
			}
		}

		result.RelatedLinks = allLinks
	}

	// Check for decision conflicts
	if len(result.BrainDecisions) > 0 {
		conflictMap := make(map[string]bool) // dedupe conflicts
		var allConflicts []memory.DecisionConflict

		for i := range result.BrainDecisions {
			decision := &result.BrainDecisions[i]
			conflicts, err := b.memory.CheckDecisionConflicts(decision.ID)
			if err == nil {
				for j := range conflicts {
					conflict := &conflicts[j]
					if !conflictMap[conflict.ConflictingID] {
						conflictMap[conflict.ConflictingID] = true
						allConflicts = append(allConflicts, *conflict)
					}
				}
			}
		}

		result.DecisionConflicts = allConflicts
	}

	return result, nil
}

// GetAutoInjectionContext returns context automatically assembled for AI agents.
// This is designed to be called when an agent focuses on a file.
func (b *Butler) GetAutoInjectionContext(filePath string, cfg *config.AutoInjectionConfig) (*AutoInjectedContext, error) {
	if cfg == nil {
		cfg = config.DefaultAutoInjectionConfig()
	}

	result := &AutoInjectedContext{
		FilePath:    filePath,
		GeneratedAt: time.Now(),
	}

	// Resolve room from file path
	room := b.resolveRoom(filePath)
	result.Room = room

	if b.memory == nil {
		return result, nil
	}

	// Gather learnings from all applicable scopes
	if cfg.IncludeLearnings {
		learnings := b.gatherPrioritizedLearnings(filePath, room, cfg)
		result.Learnings = learnings
	}

	// Gather relevant decisions
	if cfg.IncludeDecisions {
		decisions := b.gatherRelevantDecisions(filePath, room, cfg)
		result.Decisions = decisions

		// Add warnings for unreviewed decisions
		for i := range decisions {
			d := &decisions[i]
			if d.Outcome == memory.DecisionOutcomeUnknown {
				daysSince := int(time.Since(d.CreatedAt).Hours() / 24)
				if daysSince > 14 {
					result.Warnings = append(result.Warnings, ContextWarning{
						Type:    "unreviewed_decision",
						Message: fmt.Sprintf("Decision from %d days ago needs review", daysSince),
						ID:      d.ID,
						Details: d.Content,
					})
				}
			}
		}
	}

	// Gather failure information
	if cfg.IncludeFailures {
		failures := b.gatherFailures(filePath, room)
		result.Failures = failures

		// Add warnings for high-failure files
		for i := range failures {
			f := &failures[i]
			if f.Severity == "high" {
				result.Warnings = append(result.Warnings, ContextWarning{
					Type:    "fragile_file",
					Message: fmt.Sprintf("File has %d recorded failures", f.FailureCount),
					Details: f.Path,
				})
			}
		}
	}

	// Check for contradictions involving relevant learnings
	for i := range result.Learnings {
		pl := &result.Learnings[i]
		contradictions, err := b.memory.GetContradictingRecords(pl.Learning.ID)
		if err == nil && len(contradictions) > 0 {
			result.Warnings = append(result.Warnings, ContextWarning{
				Type:    "contradiction",
				Message: fmt.Sprintf("Learning has %d contradictions", len(contradictions)),
				ID:      pl.Learning.ID,
				Details: pl.Learning.Content,
			})
		}
	}

	// Estimate token count (rough approximation)
	result.TotalTokens = b.estimateTokens(result)

	return result, nil
}

// resolveRoom determines which room a file belongs to.
func (b *Butler) resolveRoom(filePath string) string {
	// Check if file is in a room's entry points
	for roomName := range b.rooms {
		room := b.rooms[roomName]
		for _, entry := range room.EntryPoints {
			if strings.HasPrefix(filePath, entry) || strings.Contains(filePath, entry) {
				return roomName
			}
		}
	}

	// Fall back to first directory component
	parts := strings.Split(filePath, "/")
	if len(parts) > 1 {
		return parts[0]
	}

	return ""
}

// gatherPrioritizedLearnings collects and prioritizes learnings across scopes.
func (b *Butler) gatherPrioritizedLearnings(filePath, room string, cfg *config.AutoInjectionConfig) []PrioritizedLearning {
	var all []PrioritizedLearning
	seen := make(map[string]bool)

	// Helper to add learning with deduplication
	addLearning := func(l memory.Learning, scopeType string, priority float64) {
		if seen[l.ID] {
			return
		}
		if l.Confidence < cfg.MinConfidence {
			return
		}
		seen[l.ID] = true

		// Compute priority: base + confidence boost + recency boost
		finalPriority := priority
		finalPriority += l.Confidence * 0.3 // Higher confidence = higher priority

		if cfg.PrioritizeRecent {
			daysSinceUsed := time.Since(l.LastUsed).Hours() / 24
			if daysSinceUsed < 7 {
				finalPriority += 0.2
			} else if daysSinceUsed < 30 {
				finalPriority += 0.1
			}
		}

		reason := fmt.Sprintf("Relevant to %s scope", scopeType)
		if l.UseCount > 5 {
			reason += ", frequently used"
		}

		all = append(all, PrioritizedLearning{
			Learning: l,
			Priority: finalPriority,
			Reason:   reason,
		})
	}

	// File-level learnings (highest priority)
	fileLearnings, _ := b.memory.GetLearnings("file", filePath, 10)
	for i := range fileLearnings {
		addLearning(fileLearnings[i], "file", 1.0)
	}
	// Room-level learnings
	if cfg.ScopeInheritance && room != "" {
		roomLearnings, _ := b.memory.GetLearnings("room", room, 10)
		for i := range roomLearnings {
			addLearning(roomLearnings[i], "room", 0.7)
		}
	}
	// Palace-level learnings
	if cfg.ScopeInheritance {
		palaceLearnings, _ := b.memory.GetLearnings("palace", "", 10)
		for i := range palaceLearnings {
			addLearning(palaceLearnings[i], "palace", 0.5)
		}
	}

	// Sort by priority descending
	sort.Slice(all, func(i, j int) bool {
		return all[i].Priority > all[j].Priority
	})

	// Limit to fit within token budget (rough estimate: 50 tokens per learning)
	maxLearnings := cfg.MaxTokens / 100
	if maxLearnings < 5 {
		maxLearnings = 5
	}
	if len(all) > maxLearnings {
		all = all[:maxLearnings]
	}

	return all
}

// gatherRelevantDecisions collects decisions relevant to the file/room.
func (b *Butler) gatherRelevantDecisions(filePath, room string, cfg *config.AutoInjectionConfig) []memory.Decision {
	var all []memory.Decision
	seen := make(map[string]bool)

	addDecision := func(d memory.Decision) {
		if seen[d.ID] {
			return
		}
		seen[d.ID] = true
		all = append(all, d)
	}

	// File-scoped decisions
	fileDecisions, _ := b.memory.GetDecisions("active", "", "file", filePath, 5)
	for i := range fileDecisions {
		addDecision(fileDecisions[i])
	}
	// Room-scoped decisions
	if cfg.ScopeInheritance && room != "" {
		roomDecisions, _ := b.memory.GetDecisions("active", "", "room", room, 5)
		for i := range roomDecisions {
			addDecision(roomDecisions[i])
		}
	}
	// Palace-scoped decisions
	if cfg.ScopeInheritance {
		palaceDecisions, _ := b.memory.GetDecisions("active", "", "palace", "", 5)
		for i := range palaceDecisions {
			addDecision(palaceDecisions[i])
		}
	}

	return all
}

// gatherFailures collects failure information for the file and room.
func (b *Butler) gatherFailures(filePath, room string) []FileFailure {
	var failures []FileFailure

	// Get file intel for the specific file
	intel, err := b.memory.GetFileIntel(filePath)
	if err == nil && intel != nil && intel.FailureCount > 0 {
		severity := "low"
		if intel.FailureCount >= 5 {
			severity = "high"
		} else if intel.FailureCount >= 2 {
			severity = "medium"
		}

		failures = append(failures, FileFailure{
			Path:         filePath,
			FailureCount: intel.FailureCount,
			Severity:     severity,
		})
	}

	// Get other fragile files in the same room
	fragileFiles, _ := b.memory.GetFragileFiles(5)
	for i := range fragileFiles {
		f := &fragileFiles[i]
		if f.Path == filePath {
			continue // Already added
		}
		// Check if in same room
		fileRoom := b.resolveRoom(f.Path)
		if fileRoom == room && room != "" {
			severity := "low"
			if f.FailureCount >= 5 {
				severity = "high"
			} else if f.FailureCount >= 2 {
				severity = "medium"
			}

			failures = append(failures, FileFailure{
				Path:         f.Path,
				FailureCount: f.FailureCount,
				Severity:     severity,
			})
		}
	}

	return failures
}

// estimateTokens provides a rough token count for the context.
func (b *Butler) estimateTokens(ctx *AutoInjectedContext) int {
	tokens := 0

	// Estimate ~50 tokens per learning
	tokens += len(ctx.Learnings) * 50

	// Estimate ~80 tokens per decision (include rationale)
	tokens += len(ctx.Decisions) * 80

	// Estimate ~20 tokens per failure
	tokens += len(ctx.Failures) * 20

	// Estimate ~30 tokens per warning
	tokens += len(ctx.Warnings) * 30

	return tokens
}

// GetScopeExplanation returns an explanation of scope inheritance for a file.
func (b *Butler) GetScopeExplanation(filePath string, scopeCfg *config.ScopeConfig) (*ScopeExplanation, error) {
	if scopeCfg == nil {
		scopeCfg = config.DefaultScopeConfig()
	}

	room := b.resolveRoom(filePath)

	explanation := &ScopeExplanation{
		FilePath:     filePath,
		ResolvedRoom: room,
		TotalRecords: make(map[string]int),
	}

	// Build inheritance chain
	chain := []ScopeLevel{
		{Scope: "file", Path: filePath, Active: true},
	}

	if b.memory != nil {
		// Count file-level records
		fileLearnings, _ := b.memory.GetLearnings("file", filePath, 100)
		explanation.TotalRecords["file"] = len(fileLearnings)
		chain[0].RecordCount = len(fileLearnings)

		// Room level
		if room != "" {
			roomActive := scopeCfg.InheritFromRoom
			roomLearnings, _ := b.memory.GetLearnings("room", room, 100)
			explanation.TotalRecords["room"] = len(roomLearnings)
			chain = append(chain, ScopeLevel{
				Scope:       "room",
				Path:        room,
				RecordCount: len(roomLearnings),
				Active:      roomActive,
			})
		}

		// Palace level
		palaceActive := scopeCfg.InheritFromPalace
		palaceLearnings, _ := b.memory.GetLearnings("palace", "", 100)
		explanation.TotalRecords["palace"] = len(palaceLearnings)
		chain = append(chain, ScopeLevel{
			Scope:       "palace",
			Path:        "",
			RecordCount: len(palaceLearnings),
			Active:      palaceActive,
		})

		// Corridor level
		corridorActive := scopeCfg.InheritFromCorridor
		chain = append(chain, ScopeLevel{
			Scope:       "corridor",
			Path:        "~/.palace/corridors/personal.db",
			RecordCount: 0, // Would need to query corridor separately
			Active:      corridorActive,
		})
	}

	explanation.InheritanceChain = chain

	return explanation, nil
}

// GetImpact returns impact analysis for a file or symbol.
func (b *Butler) GetImpact(target string) (*index.ImpactResult, error) {
	return index.GetImpact(b.db, target)
}

// ListSymbols lists all symbols of a given kind.
func (b *Butler) ListSymbols(kind string, limit int) ([]index.SymbolInfo, error) {
	return index.SearchSymbolsByKind(b.db, kind, limit)
}

// GetSymbol returns a specific symbol by name.
func (b *Butler) GetSymbol(name, filePath string) (*index.SymbolInfo, error) {
	return index.GetSymbol(b.db, name, filePath)
}

// GetFileSymbols returns all symbols in a file.
func (b *Butler) GetFileSymbols(filePath string) ([]index.SymbolInfo, error) {
	return index.ListExportedSymbols(b.db, filePath)
}

// GetDependencyGraph returns the import graph for a set of files.
func (b *Butler) GetDependencyGraph(rootFiles []string) ([]index.DependencyNode, error) {
	return index.GetDependencyGraph(b.db, rootFiles)
}

// GetIncomingCalls returns all locations that call the given symbol.
func (b *Butler) GetIncomingCalls(symbolName string) ([]index.CallSite, error) {
	return index.GetIncomingCalls(b.db, symbolName)
}

// GetOutgoingCalls returns all functions called by the given symbol.
func (b *Butler) GetOutgoingCalls(symbolName, filePath string) ([]index.CallSite, error) {
	return index.GetOutgoingCalls(b.db, symbolName, filePath)
}

// GetCallGraph returns the complete call graph for a file.
func (b *Butler) GetCallGraph(filePath string) (*index.CallGraph, error) {
	return index.GetCallGraph(b.db, filePath)
}

// GetCallChain returns the recursive call chain for a symbol.
func (b *Butler) GetCallChain(symbolName, filePath, direction string, maxDepth int) (*index.CallChainResult, error) {
	return index.GetCallChain(b.db, symbolName, filePath, direction, maxDepth)
}

// BoundedContextConfig configures the bounded authoritative context query.
// This uses explicit item counts and character limits, not token heuristics.
type BoundedContextConfig struct {
	// MaxDecisions is the maximum number of decisions to include.
	// Default: 10
	MaxDecisions int

	// MaxLearnings is the maximum number of learnings to include.
	// Default: 10
	MaxLearnings int

	// MaxContentLen is the maximum characters per content item.
	// Content exceeding this limit is truncated with "...".
	// Default: 500
	MaxContentLen int
}

// DefaultBoundedContextConfig returns sensible defaults.
func DefaultBoundedContextConfig() *BoundedContextConfig {
	return &BoundedContextConfig{
		MaxDecisions:  10,
		MaxLearnings:  10,
		MaxContentLen: 500,
	}
}

// BoundedContextResult contains the bounded authoritative context.
type BoundedContextResult struct {
	// FilePath is the input file path.
	FilePath string `json:"filePath"`

	// Room is the resolved room for the file.
	Room string `json:"room,omitempty"`

	// ScopeChain shows the scope inheritance chain.
	ScopeChain []BoundedScopeLevel `json:"scopeChain"`

	// Decisions are authoritative decisions across the scope chain.
	Decisions []BoundedDecision `json:"decisions,omitempty"`

	// Learnings are authoritative learnings across the scope chain.
	Learnings []BoundedLearning `json:"learnings,omitempty"`

	// TotalDecisions is the count before limiting.
	TotalDecisions int `json:"totalDecisions"`

	// TotalLearnings is the count before limiting.
	TotalLearnings int `json:"totalLearnings"`

	// Truncated indicates whether any content was truncated.
	Truncated bool `json:"truncated"`
}

// BoundedScopeLevel represents a scope level in the bounded result.
type BoundedScopeLevel struct {
	Scope    string `json:"scope"`
	Path     string `json:"path"`
	Priority int    `json:"priority"`
}

// BoundedDecision is a decision with source scope and truncated content.
type BoundedDecision struct {
	ID          string `json:"id"`
	Content     string `json:"content"`
	Rationale   string `json:"rationale,omitempty"`
	Scope       string `json:"scope"`
	ScopePath   string `json:"scopePath"`
	SourceScope string `json:"sourceScope"` // Which scope level it came from
}

// BoundedLearning is a learning with source scope and truncated content.
type BoundedLearning struct {
	ID          string  `json:"id"`
	Content     string  `json:"content"`
	Confidence  float64 `json:"confidence"`
	Scope       string  `json:"scope"`
	ScopePath   string  `json:"scopePath"`
	SourceScope string  `json:"sourceScope"` // Which scope level it came from
}

// GetBoundedAuthoritativeContext returns authoritative decisions and learnings
// for a file path with deterministic, bounded results.
//
// This method:
// - Uses centralized scope expansion (file -> room -> palace)
// - Applies explicit item counts and character limits (no token heuristics)
// - Uses deterministic truncation (first N chars + "...")
// - Only returns authoritative records (approved/legacy_approved)
func (b *Butler) GetBoundedAuthoritativeContext(filePath string, cfg *BoundedContextConfig) (*BoundedContextResult, error) {
	if cfg == nil {
		cfg = DefaultBoundedContextConfig()
	}

	result := &BoundedContextResult{
		FilePath: filePath,
	}

	// Resolve room from file path
	room := b.resolveRoom(filePath)
	result.Room = room

	if b.memory == nil {
		return result, nil
	}

	// Create memory query config from bounded config
	memoryCfg := &memory.AuthoritativeQueryConfig{
		MaxDecisions:      cfg.MaxDecisions,
		MaxLearnings:      cfg.MaxLearnings,
		MaxContentLen:     cfg.MaxContentLen,
		AuthoritativeOnly: true,
	}

	// Use centralized scope expansion and query
	scopedResult, err := b.memory.GetAuthoritativeState(
		memory.ScopeFile,
		filePath,
		b.resolveRoom, // Pass our room resolver
		memoryCfg,
	)
	if err != nil {
		return nil, fmt.Errorf("get authoritative state: %w", err)
	}

	// Convert scope chain
	for _, level := range scopedResult.ScopeChain {
		result.ScopeChain = append(result.ScopeChain, BoundedScopeLevel{
			Scope:    string(level.Scope),
			Path:     level.Path,
			Priority: level.Priority,
		})
	}

	// Convert decisions
	for _, sd := range scopedResult.Decisions {
		result.Decisions = append(result.Decisions, BoundedDecision{
			ID:          sd.Decision.ID,
			Content:     sd.Decision.Content,
			Rationale:   memoryCfg.TruncateContent(sd.Decision.Rationale),
			Scope:       sd.Decision.Scope,
			ScopePath:   sd.Decision.ScopePath,
			SourceScope: string(sd.SourceScope.Scope),
		})
	}

	// Convert learnings
	for _, sl := range scopedResult.Learnings {
		result.Learnings = append(result.Learnings, BoundedLearning{
			ID:          sl.Learning.ID,
			Content:     sl.Learning.Content,
			Confidence:  sl.Learning.Confidence,
			Scope:       sl.Learning.Scope,
			ScopePath:   sl.Learning.ScopePath,
			SourceScope: string(sl.SourceScope.Scope),
		})
	}

	result.TotalDecisions = scopedResult.TotalDecisions
	result.TotalLearnings = scopedResult.TotalLearnings
	result.Truncated = scopedResult.Truncated

	return result, nil
}
