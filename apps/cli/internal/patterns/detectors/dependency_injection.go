// Package detectors contains built-in pattern detectors.
package detectors

import (
	"context"
	"regexp"
	"strings"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/analysis"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/patterns"
)

// DependencyInjectionDetector detects dependency injection patterns.
// It identifies how dependencies are passed to components (constructor, field, parameter).
type DependencyInjectionDetector struct {
	patterns.BaseDetector
}

// NewDependencyInjectionDetector creates a new DI pattern detector.
func NewDependencyInjectionDetector() *DependencyInjectionDetector {
	return &DependencyInjectionDetector{
		BaseDetector: patterns.NewBaseDetector(
			"structural/dependency-injection",
			patterns.CategoryStructural,
			"dependency-injection",
			"Dependency Injection Patterns",
			"Detects dependency injection patterns (constructor, field, provider)",
			[]string{"go", "typescript", "javascript", "python"},
		),
	}
}

var (
	// Go DI patterns
	goConstructorPattern     = regexp.MustCompile(`func\s+New\w+\s*\([^)]*\)\s*\*?\w+`)
	goInterfaceParamPattern  = regexp.MustCompile(`\w+\s+\w+er\b`)
	goWirePattern            = regexp.MustCompile(`wire\.(Build|Bind|Value|Struct)`)
	goFxPattern              = regexp.MustCompile(`fx\.(Provide|Invoke|Module|Option)`)
	goDigPattern             = regexp.MustCompile(`dig\.(Container|Provide|Invoke)`)

	// TypeScript/JavaScript DI patterns
	tsConstructorInjectPattern = regexp.MustCompile(`constructor\s*\([^)]*private|public`)
	tsInjectablePattern       = regexp.MustCompile(`@Injectable\s*\(`)
	tsInjectPattern           = regexp.MustCompile(`@Inject\s*\(`)
	tsNestModulePattern       = regexp.MustCompile(`@Module\s*\(`)
	tsProviderPattern         = regexp.MustCompile(`providers\s*:\s*\[`)
	tsUseClassPattern         = regexp.MustCompile(`useClass\s*:`)

	// Python DI patterns
	pyInjectPattern         = regexp.MustCompile(`@inject`)
	pyDependsPattern        = regexp.MustCompile(`Depends\s*\(`)
	pyContainerPattern      = regexp.MustCompile(`(Container|container)\s*\(`)
	pyPuncqPattern          = regexp.MustCompile(`dependency_injector\.|punq\.`)
)

type diStyle string

const (
	styleConstructorDI  diStyle = "constructor"
	styleFieldDI        diStyle = "field"
	styleContainerDI    diStyle = "container"
	styleProviderDI     diStyle = "provider"
)

type diFramework string

const (
	fwManual     diFramework = "manual"
	fwWire       diFramework = "wire"
	fwFx         diFramework = "fx"
	fwDig        diFramework = "dig"
	fwNestJS     diFramework = "nestjs"
	fwAngular    diFramework = "angular"
	fwFastAPIDI  diFramework = "fastapi-depends"
	fwPunq       diFramework = "punq"
)

// Detect implements the Detector interface.
func (d *DependencyInjectionDetector) Detect(ctx context.Context, dctx *patterns.DetectionContext) (*patterns.DetectionResult, error) {
	content := string(dctx.FileContent)
	lines := strings.Split(content, "\n")
	lang := dctx.File.Language

	var locations []patterns.Location
	var outliers []patterns.Location

	// Track DI patterns
	frameworkCounts := make(map[diFramework]int)
	frameworkLocations := make(map[diFramework][]patterns.Location)

	styleCounts := make(map[diStyle]int)
	totalDIUsages := 0

	// Also check symbols for constructor patterns
	constructorInjections := 0
	for _, sym := range dctx.File.Symbols {
		if sym.Kind == analysis.KindFunction || sym.Kind == analysis.KindMethod {
			// Check for New* constructor pattern in Go
			if lang == "go" && strings.HasPrefix(sym.Name, "New") {
				constructorInjections++
			}
			// Check for constructor in TypeScript/JavaScript
			if (lang == "typescript" || lang == "javascript") && sym.Kind == analysis.KindConstructor {
				constructorInjections++
			}
		}
	}

	for lineNum, line := range lines {
		lineNo := lineNum + 1

		switch lang {
		case "go":
			// Check for constructor pattern
			if goConstructorPattern.MatchString(line) {
				totalDIUsages++
				styleCounts[styleConstructorDI]++
				frameworkCounts[fwManual]++
				addDILocation(frameworkLocations, fwManual, dctx.File.Path, lineNo, line)
			}

			// Check for wire
			if goWirePattern.MatchString(line) {
				totalDIUsages++
				styleCounts[styleContainerDI]++
				frameworkCounts[fwWire]++
				addDILocation(frameworkLocations, fwWire, dctx.File.Path, lineNo, line)
			}

			// Check for fx
			if goFxPattern.MatchString(line) {
				totalDIUsages++
				styleCounts[styleContainerDI]++
				frameworkCounts[fwFx]++
				addDILocation(frameworkLocations, fwFx, dctx.File.Path, lineNo, line)
			}

			// Check for dig
			if goDigPattern.MatchString(line) {
				totalDIUsages++
				styleCounts[styleContainerDI]++
				frameworkCounts[fwDig]++
				addDILocation(frameworkLocations, fwDig, dctx.File.Path, lineNo, line)
			}

		case "typescript", "javascript":
			// Check for NestJS patterns
			if tsInjectablePattern.MatchString(line) || tsNestModulePattern.MatchString(line) {
				totalDIUsages++
				styleCounts[styleProviderDI]++
				frameworkCounts[fwNestJS]++
				addDILocation(frameworkLocations, fwNestJS, dctx.File.Path, lineNo, line)
			}

			// Check for Angular patterns
			if tsInjectPattern.MatchString(line) {
				totalDIUsages++
				styleCounts[styleFieldDI]++
				frameworkCounts[fwAngular]++
				addDILocation(frameworkLocations, fwAngular, dctx.File.Path, lineNo, line)
			}

			// Check for constructor injection
			if tsConstructorInjectPattern.MatchString(line) {
				totalDIUsages++
				styleCounts[styleConstructorDI]++
				// Don't add to framework counts since this is a pattern, not a framework
			}

		case "python":
			// Check for FastAPI Depends
			if pyDependsPattern.MatchString(line) {
				totalDIUsages++
				styleCounts[styleProviderDI]++
				frameworkCounts[fwFastAPIDI]++
				addDILocation(frameworkLocations, fwFastAPIDI, dctx.File.Path, lineNo, line)
			}

			// Check for dependency_injector or punq
			if pyPuncqPattern.MatchString(line) {
				totalDIUsages++
				styleCounts[styleContainerDI]++
				frameworkCounts[fwPunq]++
				addDILocation(frameworkLocations, fwPunq, dctx.File.Path, lineNo, line)
			}

			// Check for inject decorator
			if pyInjectPattern.MatchString(line) {
				totalDIUsages++
				styleCounts[styleFieldDI]++
			}
		}
	}

	if totalDIUsages == 0 {
		return nil, nil
	}

	// Determine dominant framework
	var dominantFW diFramework
	maxCount := 0
	for fw, count := range frameworkCounts {
		if count > maxCount {
			maxCount = count
			dominantFW = fw
		}
	}

	// Determine dominant style
	var dominantStyle diStyle
	maxStyleCount := 0
	for style, count := range styleCounts {
		if count > maxStyleCount {
			maxStyleCount = count
			dominantStyle = style
		}
	}

	// Build locations
	for fw, locs := range frameworkLocations {
		if fw == dominantFW {
			locations = append(locations, locs...)
		} else {
			for _, loc := range locs {
				loc.IsOutlier = true
				loc.OutlierReason = "Uses " + string(fw) + " instead of " + string(dominantFW)
				outliers = append(outliers, loc)
			}
		}
	}

	// Calculate consistency
	consistency := 1.0
	if len(outliers) > 0 {
		consistency = float64(len(locations)) / float64(len(locations)+len(outliers))
	}

	return &patterns.DetectionResult{
		Locations: locations,
		Outliers:  outliers,
		Confidence: patterns.ConfidenceFactors{
			Frequency:   patterns.CalculateFrequencyScore(totalDIUsages, 5),
			Consistency: consistency,
			Spread:      0.5,
			Age:         0.3,
		},
		Metadata: map[string]any{
			"dominant_framework":      string(dominantFW),
			"dominant_style":          string(dominantStyle),
			"constructor_injections":  constructorInjections,
			"container_di_count":      styleCounts[styleContainerDI],
			"provider_di_count":       styleCounts[styleProviderDI],
			"field_di_count":          styleCounts[styleFieldDI],
			"total_di_usages":         totalDIUsages,
		},
	}, nil
}

func addDILocation(m map[diFramework][]patterns.Location, fw diFramework, path string, line int, content string) {
	loc := patterns.Location{
		FilePath:  path,
		LineStart: line,
		LineEnd:   line,
		Snippet:   strings.TrimSpace(content),
	}
	m[fw] = append(m[fw], loc)
}

func init() {
	patterns.MustRegister(NewDependencyInjectionDetector())
}
