// Package detectors contains built-in pattern detectors.
package detectors

import (
	"context"
	"fmt"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/analysis"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/patterns"
)

// FunctionLengthDetector detects function length patterns.
// It identifies what typical function sizes are in the codebase
// and flags unusually long functions as outliers.
type FunctionLengthDetector struct {
	patterns.BaseDetector
}

// NewFunctionLengthDetector creates a new function length detector.
func NewFunctionLengthDetector() *FunctionLengthDetector {
	return &FunctionLengthDetector{
		BaseDetector: patterns.NewBaseDetector(
			"complexity/function-length",
			patterns.CategoryComplexity,
			"function-length",
			"Function Length Pattern",
			"Detects typical function sizes and identifies unusually long functions",
			[]string{}, // All languages
		),
	}
}

// Detect implements the Detector interface.
func (d *FunctionLengthDetector) Detect(_ context.Context, dctx *patterns.DetectionContext) (*patterns.DetectionResult, error) {
	fa := dctx.File

	// Extract functions from symbols
	var functions []analysis.Symbol
	for _, sym := range fa.Symbols {
		if sym.Kind == analysis.KindFunction || sym.Kind == analysis.KindMethod {
			functions = append(functions, sym)
		}
	}

	if len(functions) == 0 {
		return nil, nil
	}

	var locations []patterns.Location
	var outliers []patterns.Location

	// Calculate function lengths
	lengths := make([]int, 0, len(functions))
	totalLines := 0

	for _, fn := range functions {
		length := fn.LineEnd - fn.LineStart + 1
		lengths = append(lengths, length)
		totalLines += length
	}

	// Calculate average and identify threshold for outliers
	avgLength := float64(totalLines) / float64(len(functions))

	// Consider functions longer than 2x average or > 50 lines as outliers
	outlierThreshold := max(int(avgLength*2), 50)

	shortFunctions := 0  // <= 20 lines
	mediumFunctions := 0 // 21-50 lines
	longFunctions := 0   // > 50 lines

	for i, fn := range functions {
		length := lengths[i]
		loc := patterns.Location{
			FilePath:  fa.Path,
			LineStart: fn.LineStart,
			LineEnd:   fn.LineEnd,
			Snippet:   fn.Name,
		}

		switch {
		case length <= 20:
			shortFunctions++
			locations = append(locations, loc)
		case length <= 50:
			mediumFunctions++
			locations = append(locations, loc)
		default:
			longFunctions++
			if length > outlierThreshold {
				loc.IsOutlier = true
				loc.OutlierReason = fmt.Sprintf("Function is unusually long (%d lines, threshold: %d)", length, outlierThreshold)
				outliers = append(outliers, loc)
			} else {
				locations = append(locations, loc)
			}
		}
	}

	// Determine dominant pattern
	dominantPattern := "short"
	if mediumFunctions > shortFunctions && mediumFunctions > longFunctions {
		dominantPattern = "medium"
	} else if longFunctions > shortFunctions && longFunctions > mediumFunctions {
		dominantPattern = "long"
	}

	// Calculate consistency based on variance from average
	consistency := 1.0
	if len(outliers) > 0 {
		consistency = float64(len(locations)) / float64(len(locations)+len(outliers))
	}

	return &patterns.DetectionResult{
		Locations: locations,
		Outliers:  outliers,
		Confidence: patterns.ConfidenceFactors{
			Frequency:   patterns.CalculateFrequencyScore(len(functions), 5),
			Consistency: consistency,
			Spread:      0.5,
			Age:         0.3,
		},
		Metadata: map[string]any{
			"average_length":    avgLength,
			"outlier_threshold": outlierThreshold,
			"short_functions":   shortFunctions,
			"medium_functions":  mediumFunctions,
			"long_functions":    longFunctions,
			"dominant_pattern":  dominantPattern,
			"total_functions":   len(functions),
		},
	}, nil
}

func init() {
	patterns.MustRegister(NewFunctionLengthDetector())
}
