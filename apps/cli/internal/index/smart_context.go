// Package index provides the core database functionality for indexing project files and symbols.
package index

import (
	"database/sql"
	"math"
	"sort"
	"time"
)

// ContextScore combines all scoring factors for a file in context
type ContextScore struct {
	Path            string  `json:"path"`
	RelevanceScore  float64 `json:"relevanceScore"`  // From FTS/semantic match (0-1)
	UsageScore      float64 `json:"usageScore"`      // From call graph centrality (0-1)
	RecencyScore    float64 `json:"recencyScore"`    // From edit history (0-1)
	DependencyScore float64 `json:"dependencyScore"` // From import depth (0-1, closer = higher)
	FinalScore      float64 `json:"finalScore"`      // Weighted combination
}

// SmartContextOptions configures smart context expansion behavior
type SmartContextOptions struct {
	// Dependency expansion
	ExpandDependencies bool // Include imported files in context (default: true)
	DependencyDepth    int  // How many levels of imports to follow (default: 1)
	BothDirections     bool // Include files that import the matched files (default: false)

	// Usage-based prioritization
	PrioritizeByUsage bool // Rank heavily-used symbols higher (default: true)

	// Edit history
	BoostRecentEdits bool          // Boost recently edited files (default: true)
	RecentEditWindow time.Duration // Time window for recency boost (default: 7 days)

	// Scoring weights (should sum to 1.0)
	RelevanceWeight  float64 // Weight for FTS/semantic relevance (default: 0.4)
	UsageWeight      float64 // Weight for usage-based scoring (default: 0.3)
	RecencyWeight    float64 // Weight for recency-based scoring (default: 0.2)
	DependencyWeight float64 // Weight for dependency-based scoring (default: 0.1)

	// Limits
	MaxFiles int // Maximum files to return (default: 50)
}

// DefaultSmartContextOptions returns sensible defaults for smart context
func DefaultSmartContextOptions() *SmartContextOptions {
	return &SmartContextOptions{
		ExpandDependencies: true,
		DependencyDepth:    1,
		BothDirections:     false,
		PrioritizeByUsage:  true,
		BoostRecentEdits:   true,
		RecentEditWindow:   7 * 24 * time.Hour, // 1 week
		RelevanceWeight:    0.4,
		UsageWeight:        0.3,
		RecencyWeight:      0.2,
		DependencyWeight:   0.1,
		MaxFiles:           50,
	}
}

// FileEditInfo represents edit history information for a file
type FileEditInfo struct {
	Path       string
	EditCount  int
	LastEdited time.Time
}

// SmartContextResult contains the scored and expanded context
type SmartContextResult struct {
	Files        []ContextScore    `json:"files"`
	ExpandedFrom []string          `json:"expandedFrom"` // Original seed files
	Options      string            `json:"options"`      // Description of applied options
	Stats        SmartContextStats `json:"stats"`
}

// SmartContextStats provides statistics about the smart context operation
type SmartContextStats struct {
	SeedFiles     int     `json:"seedFiles"`
	ExpandedFiles int     `json:"expandedFiles"`
	TotalFiles    int     `json:"totalFiles"`
	AvgScore      float64 `json:"avgScore"`
	MaxScore      float64 `json:"maxScore"`
}

// ComputeSmartContext applies intelligent scoring and expansion to context files.
// It takes initial context (from query) and enhances it with dependency expansion,
// usage scoring, and recency boosting.
func ComputeSmartContext(
	db *sql.DB,
	initialFiles []FileContext,
	editHistory map[string]*FileEditInfo,
	opts *SmartContextOptions,
) (*SmartContextResult, error) {
	if opts == nil {
		opts = DefaultSmartContextOptions()
	}
	if opts.MaxFiles <= 0 {
		opts.MaxFiles = 50
	}

	result := &SmartContextResult{
		Files:        make([]ContextScore, 0),
		ExpandedFrom: make([]string, 0, len(initialFiles)),
	}

	// Track all files we're scoring
	fileScores := make(map[string]*ContextScore)

	// 1. Start with initial files (seed files)
	seedPaths := make([]string, 0, len(initialFiles))
	for _, f := range initialFiles {
		seedPaths = append(seedPaths, f.Path)
		result.ExpandedFrom = append(result.ExpandedFrom, f.Path)
		fileScores[f.Path] = &ContextScore{
			Path:            f.Path,
			RelevanceScore:  f.Relevance,
			DependencyScore: 1.0, // Seed files have maximum dependency score
		}
	}

	// 2. Expand with dependencies if enabled
	if opts.ExpandDependencies && opts.DependencyDepth > 0 {
		expandOpts := &ExpandOptions{
			MaxDepth:        opts.DependencyDepth,
			IncludeKinds:    []string{"import"},
			ExcludePatterns: DefaultExcludePatterns,
			MaxFiles:        opts.MaxFiles,
			BothDirections:  opts.BothDirections,
		}

		expanded, err := ExpandWithDependencies(db, seedPaths, expandOpts)
		if err == nil {
			for _, ef := range expanded {
				if _, exists := fileScores[ef.Path]; !exists {
					// New file from expansion
					// Dependency score decreases with depth
					depScore := 1.0 - (float64(ef.Depth) * 0.3)
					if depScore < 0.1 {
						depScore = 0.1
					}
					fileScores[ef.Path] = &ContextScore{
						Path:            ef.Path,
						RelevanceScore:  0.5, // Default relevance for expanded files
						DependencyScore: depScore,
					}
				}
			}
		}
	}

	// 3. Compute usage scores if enabled
	if opts.PrioritizeByUsage {
		allPaths := make([]string, 0, len(fileScores))
		for path := range fileScores {
			allPaths = append(allPaths, path)
		}

		usageScores, err := GetFileUsageScores(db, allPaths)
		if err == nil {
			for path, score := range fileScores {
				if usage, ok := usageScores[path]; ok {
					score.UsageScore = usage.UsageScore
				}
			}
		}
	}

	// 4. Apply recency boost if enabled
	if opts.BoostRecentEdits && editHistory != nil {
		now := time.Now()
		for path, score := range fileScores {
			if editInfo, ok := editHistory[path]; ok {
				if !editInfo.LastEdited.IsZero() {
					age := now.Sub(editInfo.LastEdited)
					if age <= opts.RecentEditWindow {
						// Recency score: 1.0 for just edited, 0.0 at window boundary
						recencyRatio := 1.0 - (float64(age) / float64(opts.RecentEditWindow))
						score.RecencyScore = math.Max(0, recencyRatio)
					}
				}
			}
		}
	}

	// 5. Compute final weighted scores
	for _, score := range fileScores {
		score.FinalScore =
			score.RelevanceScore*opts.RelevanceWeight +
				score.UsageScore*opts.UsageWeight +
				score.RecencyScore*opts.RecencyWeight +
				score.DependencyScore*opts.DependencyWeight
	}

	// 6. Sort by final score (descending)
	sortedScores := make([]*ContextScore, 0, len(fileScores))
	for _, s := range fileScores {
		sortedScores = append(sortedScores, s)
	}
	sort.Slice(sortedScores, func(i, j int) bool {
		return sortedScores[i].FinalScore > sortedScores[j].FinalScore
	})

	// 7. Apply limit and build result
	limit := opts.MaxFiles
	if limit > len(sortedScores) {
		limit = len(sortedScores)
	}

	var totalScore, maxScore float64
	for i := 0; i < limit; i++ {
		score := *sortedScores[i]
		result.Files = append(result.Files, score)
		totalScore += score.FinalScore
		if score.FinalScore > maxScore {
			maxScore = score.FinalScore
		}
	}

	// 8. Compute stats
	seedCount := len(seedPaths)
	expandedCount := len(fileScores) - seedCount
	if expandedCount < 0 {
		expandedCount = 0
	}

	result.Stats = SmartContextStats{
		SeedFiles:     seedCount,
		ExpandedFiles: expandedCount,
		TotalFiles:    len(result.Files),
		MaxScore:      maxScore,
	}
	if len(result.Files) > 0 {
		result.Stats.AvgScore = totalScore / float64(len(result.Files))
	}

	// Build options description
	optParts := make([]string, 0)
	if opts.ExpandDependencies {
		optParts = append(optParts, "dependency-expansion")
	}
	if opts.PrioritizeByUsage {
		optParts = append(optParts, "usage-prioritization")
	}
	if opts.BoostRecentEdits {
		optParts = append(optParts, "recency-boost")
	}
	if len(optParts) > 0 {
		result.Options = "enabled: " + joinWithComma(optParts)
	} else {
		result.Options = "basic (no smart features)"
	}

	return result, nil
}

// joinWithComma joins strings with commas
func joinWithComma(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += ", " + parts[i]
	}
	return result
}

// EnhanceContextResult applies smart scoring to an existing ContextResult.
// This is a convenience function for enhancing Oracle query results.
func EnhanceContextResult(
	db *sql.DB,
	original *ContextResult,
	editHistory map[string]*FileEditInfo,
	opts *SmartContextOptions,
) (*ContextResult, error) {
	if opts == nil {
		opts = DefaultSmartContextOptions()
	}

	// Apply smart context computation
	smartResult, err := ComputeSmartContext(db, original.Files, editHistory, opts)
	if err != nil {
		return original, err // Return original on error
	}

	// Build new file list from smart scores
	fileMap := make(map[string]*FileContext)
	for i := range original.Files {
		fileMap[original.Files[i].Path] = &original.Files[i]
	}

	// Create enhanced files list
	enhanced := make([]FileContext, 0, len(smartResult.Files))
	for _, score := range smartResult.Files {
		if fc, ok := fileMap[score.Path]; ok {
			// Update relevance with final score
			newFC := *fc
			newFC.Relevance = score.FinalScore
			enhanced = append(enhanced, newFC)
		} else {
			// New file from expansion - need to fetch basic info
			lang, _ := getFileLanguage(db, score.Path)
			enhanced = append(enhanced, FileContext{
				Path:      score.Path,
				Language:  lang,
				Relevance: score.FinalScore,
			})
		}
	}

	// Create enhanced result
	result := &ContextResult{
		Query:      original.Query,
		Files:      enhanced,
		Symbols:    original.Symbols,
		Imports:    original.Imports,
		Decisions:  original.Decisions,
		Warnings:   original.Warnings,
		TotalFiles: len(enhanced),
		TokenStats: original.TokenStats,
	}

	return result, nil
}

// QuickScoreFiles scores files without full smart context computation.
// Useful for quick prioritization in listing scenarios.
func QuickScoreFiles(
	db *sql.DB,
	files []string,
	editHistory map[string]*FileEditInfo,
) ([]ContextScore, error) {
	scores := make([]ContextScore, 0, len(files))

	// Get usage scores in batch
	usageScores, _ := GetFileUsageScores(db, files)

	now := time.Now()
	weekAgo := now.Add(-7 * 24 * time.Hour)

	for _, path := range files {
		score := ContextScore{
			Path:           path,
			RelevanceScore: 1.0, // All files equal relevance in quick mode
		}

		// Add usage score
		if usage, ok := usageScores[path]; ok {
			score.UsageScore = usage.UsageScore
		}

		// Add recency score
		if editHistory != nil {
			if edit, ok := editHistory[path]; ok && !edit.LastEdited.IsZero() {
				if edit.LastEdited.After(weekAgo) {
					age := now.Sub(edit.LastEdited)
					score.RecencyScore = 1.0 - (float64(age) / float64(7*24*time.Hour))
					if score.RecencyScore < 0 {
						score.RecencyScore = 0
					}
				}
			}
		}

		// Simple final score
		score.FinalScore = score.RelevanceScore*0.3 + score.UsageScore*0.4 + score.RecencyScore*0.3

		scores = append(scores, score)
	}

	// Sort by final score
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].FinalScore > scores[j].FinalScore
	})

	return scores, nil
}
