// Package detectors contains built-in pattern detectors.
package detectors

import (
	"context"
	"regexp"
	"strings"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/analysis"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/patterns"
)

// NamingConventionDetector detects naming convention patterns in code.
// It analyzes function, variable, and type names to identify consistent naming styles.
type NamingConventionDetector struct {
	patterns.BaseDetector
}

// NewNamingConventionDetector creates a new naming convention detector.
func NewNamingConventionDetector() *NamingConventionDetector {
	return &NamingConventionDetector{
		BaseDetector: patterns.NewBaseDetector(
			"naming/conventions",
			patterns.CategoryNaming,
			"naming-conventions",
			"Naming Conventions",
			"Detects consistent naming patterns (camelCase, PascalCase, snake_case) across the codebase",
			[]string{}, // All languages
		),
	}
}

var (
	// camelCase: starts lowercase, has uppercase letters
	camelCasePattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)
	// PascalCase: starts uppercase, has lowercase letters
	pascalCasePattern = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)
	// snake_case: lowercase with underscores
	snakeCasePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	// SCREAMING_SNAKE_CASE: uppercase with underscores
	screamingSnakePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	// kebab-case: lowercase with hyphens
	kebabCasePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
)

type namingStyle string

const (
	styleCamelCase      namingStyle = "camelCase"
	stylePascalCase     namingStyle = "PascalCase"
	styleSnakeCase      namingStyle = "snake_case"
	styleScreamingSnake namingStyle = "SCREAMING_SNAKE_CASE"
	styleKebabCase      namingStyle = "kebab-case"
	styleUnknown        namingStyle = "unknown"
)

func detectNamingStyle(name string) namingStyle {
	// Skip very short names or names with special characters
	if len(name) < 2 {
		return styleUnknown
	}

	switch {
	case screamingSnakePattern.MatchString(name) && strings.Contains(name, "_"):
		return styleScreamingSnake
	case pascalCasePattern.MatchString(name) && hasLowerCase(name):
		return stylePascalCase
	case camelCasePattern.MatchString(name) && hasUpperCase(name):
		return styleCamelCase
	case snakeCasePattern.MatchString(name) && strings.Contains(name, "_"):
		return styleSnakeCase
	case kebabCasePattern.MatchString(name) && strings.Contains(name, "-"):
		return styleKebabCase
	default:
		return styleUnknown
	}
}

func hasUpperCase(s string) bool {
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

func hasLowerCase(s string) bool {
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			return true
		}
	}
	return false
}

// Detect implements the Detector interface.
func (d *NamingConventionDetector) Detect(_ context.Context, dctx *patterns.DetectionContext) (*patterns.DetectionResult, error) {
	fa := dctx.File

	// Count naming styles for different symbol types
	functionStyles := make(map[namingStyle][]patterns.Location)
	typeStyles := make(map[namingStyle][]patterns.Location)
	varStyles := make(map[namingStyle][]patterns.Location)
	constStyles := make(map[namingStyle][]patterns.Location)

	// Analyze symbols
	for _, sym := range fa.Symbols {
		style := detectNamingStyle(sym.Name)
		if style == styleUnknown {
			continue
		}

		loc := patterns.Location{
			FilePath:  fa.Path,
			LineStart: sym.LineStart,
			LineEnd:   sym.LineEnd,
			Snippet:   sym.Name,
		}

		switch sym.Kind {
		case analysis.KindFunction, analysis.KindMethod:
			functionStyles[style] = append(functionStyles[style], loc)
		case analysis.KindClass, analysis.KindInterface, analysis.KindType, analysis.KindEnum:
			typeStyles[style] = append(typeStyles[style], loc)
		case analysis.KindConstant:
			constStyles[style] = append(constStyles[style], loc)
		case analysis.KindVariable, analysis.KindProperty:
			varStyles[style] = append(varStyles[style], loc)
		}
	}

	// Determine dominant styles and outliers
	var locations []patterns.Location
	var outliers []patterns.Location

	// Process function names
	dominantFnStyle, fnLocs, fnOutliers := findDominantStyle(functionStyles)
	locations = append(locations, fnLocs...)
	outliers = append(outliers, fnOutliers...)

	// Process type names
	dominantTypeStyle, typeLocs, typeOutliers := findDominantStyle(typeStyles)
	locations = append(locations, typeLocs...)
	outliers = append(outliers, typeOutliers...)

	// Process variable names
	dominantVarStyle, varLocs, varOutliers := findDominantStyle(varStyles)
	locations = append(locations, varLocs...)
	outliers = append(outliers, varOutliers...)

	// Process constant names
	dominantConstStyle, constLocs, constOutliers := findDominantStyle(constStyles)
	locations = append(locations, constLocs...)
	outliers = append(outliers, constOutliers...)

	if len(locations) == 0 {
		return nil, nil
	}

	// Calculate confidence
	totalSymbols := len(locations) + len(outliers)
	consistency := float64(len(locations)) / float64(totalSymbols)

	return &patterns.DetectionResult{
		Locations: locations,
		Outliers:  outliers,
		Confidence: patterns.ConfidenceFactors{
			Frequency:   patterns.CalculateFrequencyScore(len(locations), 10),
			Consistency: consistency,
			Spread:      0.5, // Single file
			Age:         0.3,
		},
		Metadata: map[string]any{
			"dominant_function_style": string(dominantFnStyle),
			"dominant_type_style":     string(dominantTypeStyle),
			"dominant_variable_style": string(dominantVarStyle),
			"dominant_constant_style": string(dominantConstStyle),
			"total_symbols":           totalSymbols,
			"outlier_count":           len(outliers),
		},
	}, nil
}

// findDominantStyle finds the most common style and marks others as outliers.
func findDominantStyle(styles map[namingStyle][]patterns.Location) (namingStyle, []patterns.Location, []patterns.Location) {
	if len(styles) == 0 {
		return styleUnknown, nil, nil
	}

	// Find dominant style
	var dominant namingStyle
	maxCount := 0
	for style, locs := range styles {
		if len(locs) > maxCount {
			maxCount = len(locs)
			dominant = style
		}
	}

	var locations []patterns.Location
	var outliers []patterns.Location

	for style, locs := range styles {
		if style == dominant {
			locations = append(locations, locs...)
		} else {
			// Mark non-dominant as outliers
			for _, loc := range locs {
				loc.IsOutlier = true
				loc.OutlierReason = "Uses " + string(style) + " instead of " + string(dominant)
				outliers = append(outliers, loc)
			}
		}
	}

	return dominant, locations, outliers
}

func init() {
	patterns.MustRegister(NewNamingConventionDetector())
}
