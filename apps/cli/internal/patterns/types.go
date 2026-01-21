// Package patterns provides automated pattern detection for codebases.
// It detects code conventions and architectural patterns, scores them by confidence,
// and integrates with Mind Palace's governance workflow.
package patterns

import (
	"time"
)

// Pattern represents a detected code pattern.
type Pattern struct {
	ID          string    `json:"id"`
	Category    string    `json:"category"`     // api, auth, errors, structural, types, etc.
	Subcategory string    `json:"subcategory"`  // More specific classification
	Name        string    `json:"name"`         // Human-readable pattern name
	Description string    `json:"description"`  // What this pattern represents
	DetectorID  string    `json:"detector_id"`  // ID of detector that found this

	// Confidence scoring
	Confidence       float64 `json:"confidence"`        // Overall score 0.0-1.0
	FrequencyScore   float64 `json:"frequency_score"`   // How often pattern appears
	ConsistencyScore float64 `json:"consistency_score"` // Uniformity of implementations
	SpreadScore      float64 `json:"spread_score"`      // Number of files using it
	AgeScore         float64 `json:"age_score"`         // How long pattern has existed

	// Status and governance
	Status     PatternStatus `json:"status"`      // discovered, approved, ignored
	Authority  string        `json:"authority"`   // proposed, approved, legacy_approved
	LearningID string        `json:"learning_id"` // Link to learning when approved

	// Locations
	Locations []Location `json:"locations"` // Where pattern is found
	Outliers  []Location `json:"outliers"`  // Deviations from pattern

	// Metadata
	Metadata  map[string]any `json:"metadata"`   // Detector-specific data
	FirstSeen time.Time      `json:"first_seen"`
	LastSeen  time.Time      `json:"last_seen"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// PatternStatus represents the lifecycle status of a pattern.
type PatternStatus string

const (
	// StatusDiscovered means the pattern was found but not yet reviewed.
	StatusDiscovered PatternStatus = "discovered"
	// StatusApproved means the pattern was approved for enforcement.
	StatusApproved PatternStatus = "approved"
	// StatusIgnored means the pattern was explicitly ignored.
	StatusIgnored PatternStatus = "ignored"
)

// Location represents where a pattern or outlier is found in code.
type Location struct {
	ID            string    `json:"id"`
	PatternID     string    `json:"pattern_id"`
	FilePath      string    `json:"file_path"`
	LineStart     int       `json:"line_start"`
	LineEnd       int       `json:"line_end"`
	Snippet       string    `json:"snippet"`        // Code snippet for context
	IsOutlier     bool      `json:"is_outlier"`     // True if this deviates from pattern
	OutlierReason string    `json:"outlier_reason"` // Why it's an outlier
	CreatedAt     time.Time `json:"created_at"`
}

// ConfidenceLevel represents a confidence threshold category.
type ConfidenceLevel string

const (
	// ConfidenceHigh indicates high confidence (>= 0.85).
	ConfidenceHigh ConfidenceLevel = "high"
	// ConfidenceMedium indicates medium confidence (0.70 - 0.84).
	ConfidenceMedium ConfidenceLevel = "medium"
	// ConfidenceLow indicates low confidence (0.50 - 0.69).
	ConfidenceLow ConfidenceLevel = "low"
	// ConfidenceUncertain indicates uncertain confidence (< 0.50).
	ConfidenceUncertain ConfidenceLevel = "uncertain"
)

// GetConfidenceLevel returns the confidence level for a given score.
func GetConfidenceLevel(score float64) ConfidenceLevel {
	switch {
	case score >= 0.85:
		return ConfidenceHigh
	case score >= 0.70:
		return ConfidenceMedium
	case score >= 0.50:
		return ConfidenceLow
	default:
		return ConfidenceUncertain
	}
}

// PatternCategory represents a category of patterns.
type PatternCategory string

const (
	CategoryAPI           PatternCategory = "api"
	CategoryAuth          PatternCategory = "auth"
	CategorySecurity      PatternCategory = "security"
	CategoryErrors        PatternCategory = "errors"
	CategoryLogging       PatternCategory = "logging"
	CategoryDataAccess    PatternCategory = "data-access"
	CategoryConfig        PatternCategory = "config"
	CategoryTesting       PatternCategory = "testing"
	CategoryPerformance   PatternCategory = "performance"
	CategoryComponents    PatternCategory = "components"
	CategoryStyling       PatternCategory = "styling"
	CategoryStructural    PatternCategory = "structural"
	CategoryTypes         PatternCategory = "types"
	CategoryAccessibility PatternCategory = "accessibility"
	CategoryDocumentation PatternCategory = "documentation"
	CategoryNaming        PatternCategory = "naming"
	CategoryComplexity    PatternCategory = "complexity"
)

// AllCategories returns all valid pattern categories.
func AllCategories() []PatternCategory {
	return []PatternCategory{
		CategoryAPI,
		CategoryAuth,
		CategorySecurity,
		CategoryErrors,
		CategoryLogging,
		CategoryDataAccess,
		CategoryConfig,
		CategoryTesting,
		CategoryPerformance,
		CategoryComponents,
		CategoryStyling,
		CategoryStructural,
		CategoryTypes,
		CategoryAccessibility,
		CategoryDocumentation,
		CategoryNaming,
		CategoryComplexity,
	}
}

// ListFilters contains filters for listing patterns.
type ListFilters struct {
	Category      string        // Filter by category
	Subcategory   string        // Filter by subcategory
	Status        PatternStatus // Filter by status
	DetectorID    string        // Filter by detector
	MinConfidence float64       // Minimum confidence threshold
	FilePath      string        // Filter by file path
	Limit         int           // Maximum results
	Offset        int           // Pagination offset
}
