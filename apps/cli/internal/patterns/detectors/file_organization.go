// Package detectors contains built-in pattern detectors.
package detectors

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/patterns"
)

// FileOrganizationDetector detects file organization patterns.
// It identifies consistent file naming and directory structure patterns.
type FileOrganizationDetector struct {
	patterns.BaseDetector
}

// NewFileOrganizationDetector creates a new file organization detector.
func NewFileOrganizationDetector() *FileOrganizationDetector {
	return &FileOrganizationDetector{
		BaseDetector: patterns.NewBaseDetector(
			"structural/file-organization",
			patterns.CategoryStructural,
			"file-organization",
			"File Organization Patterns",
			"Detects file naming conventions and directory structure patterns",
			[]string{}, // All languages
		),
	}
}

// Detect implements the Detector interface.
func (d *FileOrganizationDetector) Detect(_ context.Context, dctx *patterns.DetectionContext) (*patterns.DetectionResult, error) {
	// This detector works across files, so we analyze based on the file path
	filePath := dctx.File.Path
	fileName := filepath.Base(filePath)
	dir := filepath.Dir(filePath)

	var locations []patterns.Location
	var outliers []patterns.Location

	// Analyze file naming patterns
	filePatterns := analyzeFileName(fileName)

	if len(filePatterns) == 0 {
		return nil, nil
	}

	loc := patterns.Location{
		FilePath:  filePath,
		LineStart: 1,
		LineEnd:   1,
		Snippet:   fileName,
	}
	locations = append(locations, loc)

	// Calculate scores based on conventions detected
	consistency := 1.0
	if filePatterns["has_prefix"] && filePatterns["has_suffix"] {
		consistency = 0.9 // Good naming
	}

	return &patterns.DetectionResult{
		Locations: locations,
		Outliers:  outliers,
		Confidence: patterns.ConfidenceFactors{
			Frequency:   0.5,
			Consistency: consistency,
			Spread:      0.5,
			Age:         0.3,
		},
		Metadata: map[string]any{
			"file_name":     fileName,
			"directory":     dir,
			"is_test_file":  filePatterns["is_test"],
			"is_index_file": filePatterns["is_index"],
			"has_prefix":    filePatterns["has_prefix"],
			"has_suffix":    filePatterns["has_suffix"],
		},
	}, nil
}

func analyzeFileName(name string) map[string]bool {
	result := make(map[string]bool)
	nameLower := strings.ToLower(name)

	// Test file patterns
	result["is_test"] = strings.Contains(nameLower, "_test.") ||
		strings.Contains(nameLower, ".test.") ||
		strings.Contains(nameLower, ".spec.")

	// Index file patterns
	result["is_index"] = strings.HasPrefix(nameLower, "index.") ||
		strings.HasPrefix(nameLower, "main.") ||
		strings.HasPrefix(nameLower, "__init__.")

	// Common prefixes
	result["has_prefix"] = strings.HasPrefix(nameLower, "use") || // hooks
		strings.HasPrefix(nameLower, "get") ||
		strings.HasPrefix(nameLower, "create") ||
		strings.HasPrefix(nameLower, "make") ||
		strings.HasPrefix(nameLower, "with") ||
		strings.HasPrefix(nameLower, "is")

	// Common suffixes
	result["has_suffix"] = strings.Contains(nameLower, "_controller.") ||
		strings.Contains(nameLower, "_service.") ||
		strings.Contains(nameLower, "_repository.") ||
		strings.Contains(nameLower, "_handler.") ||
		strings.Contains(nameLower, "_model.") ||
		strings.Contains(nameLower, "_utils.") ||
		strings.Contains(nameLower, "_helper.") ||
		strings.Contains(nameLower, "_types.")

	return result
}

func init() {
	patterns.MustRegister(NewFileOrganizationDetector())
}
