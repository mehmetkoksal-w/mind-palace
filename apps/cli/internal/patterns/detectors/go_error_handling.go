// Package detectors contains built-in pattern detectors.
package detectors

import (
	"context"
	"regexp"
	"strings"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/patterns"
)

// GoErrorHandlingDetector detects Go error handling patterns.
// It looks for consistent error checking patterns like:
// - if err != nil { return ... }
// - if err != nil { return fmt.Errorf(...) }
type GoErrorHandlingDetector struct {
	patterns.BaseDetector
}

// NewGoErrorHandlingDetector creates a new Go error handling detector.
func NewGoErrorHandlingDetector() *GoErrorHandlingDetector {
	return &GoErrorHandlingDetector{
		BaseDetector: patterns.NewBaseDetector(
			"errors/go-error-handling",
			patterns.CategoryErrors,
			"go-error-handling",
			"Go Error Handling Pattern",
			"Detects consistent error handling patterns in Go code (if err != nil checks)",
			[]string{"go"},
		),
	}
}

var (
	// Pattern: if err != nil { return ... }
	goErrCheckPattern = regexp.MustCompile(`if\s+err\s*!=\s*nil\s*\{`)

	// Pattern: if err := ...; err != nil { ... }
	goErrInlinePattern = regexp.MustCompile(`if\s+\w+\s*:?=\s*[^;]+;\s*\w+\s*!=\s*nil\s*\{`)

	// Pattern: errors being ignored (assigned to _)
	goErrIgnoredPattern = regexp.MustCompile(`_\s*=\s*\w+\([^)]*\)`)

	// Pattern: error wrapping with fmt.Errorf
	goErrWrapPattern = regexp.MustCompile(`fmt\.Errorf\s*\([^)]*%w`)

	// Pattern: naked return of error
	goErrReturnPattern = regexp.MustCompile(`return\s+(\w+,\s*)?err\s*$`)
)

// Detect implements the Detector interface.
func (d *GoErrorHandlingDetector) Detect(ctx context.Context, dctx *patterns.DetectionContext) (*patterns.DetectionResult, error) {
	// Only process Go files
	if dctx.File.Language != "go" {
		return nil, nil
	}

	content := string(dctx.FileContent)
	lines := strings.Split(content, "\n")

	var locations []patterns.Location
	var outliers []patterns.Location

	// Count different error handling styles
	standardChecks := 0
	inlineChecks := 0
	wrappedErrors := 0
	ignoredErrors := 0
	nakedReturns := 0

	for lineNum, line := range lines {
		lineNo := lineNum + 1 // 1-based

		// Check for standard error check
		if goErrCheckPattern.MatchString(line) {
			standardChecks++
			locations = append(locations, patterns.Location{
				FilePath:  dctx.File.Path,
				LineStart: lineNo,
				LineEnd:   lineNo,
				Snippet:   strings.TrimSpace(line),
			})
		}

		// Check for inline error check
		if goErrInlinePattern.MatchString(line) {
			inlineChecks++
			locations = append(locations, patterns.Location{
				FilePath:  dctx.File.Path,
				LineStart: lineNo,
				LineEnd:   lineNo,
				Snippet:   strings.TrimSpace(line),
			})
		}

		// Check for error wrapping
		if goErrWrapPattern.MatchString(line) {
			wrappedErrors++
		}

		// Check for ignored errors (outliers)
		if goErrIgnoredPattern.MatchString(line) && strings.Contains(line, "err") {
			ignoredErrors++
			outliers = append(outliers, patterns.Location{
				FilePath:      dctx.File.Path,
				LineStart:     lineNo,
				LineEnd:       lineNo,
				Snippet:       strings.TrimSpace(line),
				IsOutlier:     true,
				OutlierReason: "Error is ignored (assigned to _)",
			})
		}

		// Check for naked error returns
		if goErrReturnPattern.MatchString(line) {
			nakedReturns++
		}
	}

	// Only report if we found error handling patterns
	totalChecks := standardChecks + inlineChecks
	if totalChecks == 0 {
		return nil, nil
	}

	// Calculate confidence factors
	// Frequency: based on number of error checks
	frequency := patterns.CalculateFrequencyScore(totalChecks, len(lines)/10) // Roughly 1 check per 10 lines is good

	// Consistency: high if most errors are wrapped, low if many are naked returns
	consistency := 1.0
	if totalChecks > 0 {
		wrapRatio := float64(wrappedErrors) / float64(totalChecks)
		nakedRatio := float64(nakedReturns) / float64(totalChecks)
		consistency = wrapRatio*0.6 + (1-nakedRatio)*0.4
	}

	// Spread: based on whether file has error handling
	spread := 0.5 // Single file

	// Age: new detection, so low
	age := 0.3

	return &patterns.DetectionResult{
		Locations: locations,
		Outliers:  outliers,
		Confidence: patterns.ConfidenceFactors{
			Frequency:   frequency,
			Consistency: consistency,
			Spread:      spread,
			Age:         age,
		},
		Metadata: map[string]any{
			"standard_checks": standardChecks,
			"inline_checks":   inlineChecks,
			"wrapped_errors":  wrappedErrors,
			"ignored_errors":  ignoredErrors,
			"naked_returns":   nakedReturns,
		},
	}, nil
}

func init() {
	patterns.MustRegister(NewGoErrorHandlingDetector())
}
