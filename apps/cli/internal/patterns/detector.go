package patterns

import (
	"context"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/analysis"
)

// Detector is the interface that all pattern detectors must implement.
type Detector interface {
	// ID returns the unique identifier for this detector (e.g., "api/response-envelope").
	ID() string

	// Category returns the pattern category (e.g., "api", "structural").
	Category() PatternCategory

	// Subcategory returns the specific subcategory (e.g., "response-envelope").
	Subcategory() string

	// Name returns a human-readable name for the pattern.
	Name() string

	// Description returns a description of what this pattern represents.
	Description() string

	// Languages returns the programming languages this detector supports.
	// An empty slice means all languages.
	Languages() []string

	// Detect runs the detection logic and returns results.
	Detect(ctx context.Context, dctx *DetectionContext) (*DetectionResult, error)
}

// DetectionContext provides context for pattern detection.
type DetectionContext struct {
	// File is the analyzed file being processed.
	File *analysis.FileAnalysis

	// FileContent is the raw content of the file.
	FileContent []byte

	// AllFiles provides access to all analyzed files for cross-file patterns.
	// This is lazily loaded and may be nil for single-file detectors.
	AllFiles []*analysis.FileAnalysis

	// FilesByPath provides quick lookup of files by path.
	FilesByPath map[string]*analysis.FileAnalysis

	// WorkspaceRoot is the root directory of the workspace.
	WorkspaceRoot string

	// TotalFileCount is the total number of files in the workspace.
	TotalFileCount int

	// Config contains detector-specific configuration.
	Config map[string]any
}

// DetectionResult contains the results of running a detector.
type DetectionResult struct {
	// Locations are places where the pattern was found.
	Locations []Location

	// Outliers are places that almost match but deviate from the pattern.
	Outliers []Location

	// Confidence contains the factors used to calculate confidence.
	Confidence ConfidenceFactors

	// Metadata contains detector-specific data about the pattern.
	Metadata map[string]any
}

// ConfidenceFactors contains the individual factors that contribute to confidence.
type ConfidenceFactors struct {
	// Frequency measures how often the pattern appears (0.0-1.0).
	// Higher values mean the pattern is more common.
	Frequency float64 `json:"frequency"`

	// Consistency measures uniformity of implementations (0.0-1.0).
	// Higher values mean implementations are more similar.
	Consistency float64 `json:"consistency"`

	// Spread measures how many files use the pattern (0.0-1.0).
	// Higher values mean the pattern is used in more files.
	Spread float64 `json:"spread"`

	// Age measures how long the pattern has existed (0.0-1.0).
	// Higher values mean the pattern is older/more established.
	Age float64 `json:"age"`
}

// Confidence weights for calculating overall score.
const (
	WeightFrequency   = 0.30
	WeightConsistency = 0.30
	WeightSpread      = 0.25
	WeightAge         = 0.15
)

// Score calculates the overall confidence score from the factors.
func (cf ConfidenceFactors) Score() float64 {
	return cf.Frequency*WeightFrequency +
		cf.Consistency*WeightConsistency +
		cf.Spread*WeightSpread +
		cf.Age*WeightAge
}

// Level returns the confidence level for the calculated score.
func (cf ConfidenceFactors) Level() ConfidenceLevel {
	return GetConfidenceLevel(cf.Score())
}

// BaseDetector provides common functionality for detectors.
// Embed this in detector implementations for convenience.
type BaseDetector struct {
	id          string
	category    PatternCategory
	subcategory string
	name        string
	description string
	languages   []string
}

// NewBaseDetector creates a new BaseDetector with the given parameters.
func NewBaseDetector(id string, category PatternCategory, subcategory, name, description string, languages []string) BaseDetector {
	return BaseDetector{
		id:          id,
		category:    category,
		subcategory: subcategory,
		name:        name,
		description: description,
		languages:   languages,
	}
}

func (b BaseDetector) ID() string              { return b.id }
func (b BaseDetector) Category() PatternCategory { return b.category }
func (b BaseDetector) Subcategory() string     { return b.subcategory }
func (b BaseDetector) Name() string            { return b.name }
func (b BaseDetector) Description() string     { return b.description }
func (b BaseDetector) Languages() []string     { return b.languages }

// SupportsLanguage checks if the detector supports the given language.
func (b BaseDetector) SupportsLanguage(lang string) bool {
	// Empty languages list means all languages supported
	if len(b.languages) == 0 {
		return true
	}
	for _, l := range b.languages {
		if l == lang {
			return true
		}
	}
	return false
}
