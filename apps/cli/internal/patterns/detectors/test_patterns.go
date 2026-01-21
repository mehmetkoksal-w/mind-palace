// Package detectors contains built-in pattern detectors.
package detectors

import (
	"context"
	"regexp"
	"strings"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/patterns"
)

// TestPatternDetector detects testing patterns and conventions.
// It identifies test file organization, naming conventions, and test structure.
type TestPatternDetector struct {
	patterns.BaseDetector
}

// NewTestPatternDetector creates a new test pattern detector.
func NewTestPatternDetector() *TestPatternDetector {
	return &TestPatternDetector{
		BaseDetector: patterns.NewBaseDetector(
			"testing/test-patterns",
			patterns.CategoryTesting,
			"test-patterns",
			"Test Patterns",
			"Detects testing conventions and patterns (naming, structure, assertions)",
			[]string{"go", "typescript", "javascript", "python"},
		),
	}
}

var (
	// Go test patterns
	goTestFuncPattern  = regexp.MustCompile(`func\s+(Test[A-Z]\w*)\s*\(`)
	goTableTestPattern = regexp.MustCompile(`tests?\s*:?=\s*\[\]struct`)
	goSubtestPattern   = regexp.MustCompile(`t\.Run\s*\(`)
	goAssertPattern    = regexp.MustCompile(`(assert\.|require\.|t\.(Error|Fatal|Fail))`)
	goMockPattern      = regexp.MustCompile(`(mock\.|Mock[A-Z]|NewMock)`)

	// JavaScript/TypeScript test patterns
	jsDescribePattern = regexp.MustCompile(`describe\s*\(\s*["'\x60]`)
	jsItPattern       = regexp.MustCompile(`(it|test)\s*\(\s*["'\x60]`)
	jsExpectPattern   = regexp.MustCompile(`expect\s*\(`)
	jsMockPattern     = regexp.MustCompile(`(jest\.mock|vi\.mock|sinon\.|Mock[A-Z])`)

	// Python test patterns
	pyTestFuncPattern   = regexp.MustCompile(`def\s+(test_\w+)\s*\(`)
	pyTestClassPattern  = regexp.MustCompile(`class\s+(Test\w+)`)
	pyAssertPattern     = regexp.MustCompile(`(assert\s+|self\.assert|pytest\.raises)`)
	pyFixturePattern    = regexp.MustCompile(`@pytest\.fixture`)
	pyMockPattern       = regexp.MustCompile(`(mock\.|Mock\(|patch\()`)
)

type testStyle string

const (
	styleTableDriven   testStyle = "table-driven"
	styleSubtests      testStyle = "subtests"
	styleBDD           testStyle = "bdd"
	styleUnitTest      testStyle = "unit"
	styleMocked        testStyle = "mocked"
)

// Detect implements the Detector interface.
//
//nolint:gocognit // pattern detection is complex by design
func (d *TestPatternDetector) Detect(_ context.Context, dctx *patterns.DetectionContext) (*patterns.DetectionResult, error) {
	// Only analyze test files
	if !isTestFile(dctx.File.Path) {
		return nil, nil
	}

	content := string(dctx.FileContent)
	lines := strings.Split(content, "\n")
	lang := dctx.File.Language

	var locations []patterns.Location
	var outliers []patterns.Location

	// Track patterns found
	patternCounts := make(map[testStyle]int)
	styleLocations := make(map[testStyle][]patterns.Location)

	// Counters
	testFunctions := 0
	assertions := 0
	mocks := 0

	switch lang {
	case "go":
		for lineNum, line := range lines {
			lineNo := lineNum + 1

			// Test functions
			if matches := goTestFuncPattern.FindStringSubmatch(line); matches != nil {
				testFunctions++
				loc := patterns.Location{
					FilePath:  dctx.File.Path,
					LineStart: lineNo,
					LineEnd:   lineNo,
					Snippet:   matches[1],
				}
				locations = append(locations, loc)
			}

			// Table-driven tests
			if goTableTestPattern.MatchString(line) {
				patternCounts[styleTableDriven]++
				styleLocations[styleTableDriven] = append(styleLocations[styleTableDriven], patterns.Location{
					FilePath:  dctx.File.Path,
					LineStart: lineNo,
					LineEnd:   lineNo,
					Snippet:   strings.TrimSpace(line),
				})
			}

			// Subtests
			if goSubtestPattern.MatchString(line) {
				patternCounts[styleSubtests]++
			}

			// Assertions
			if goAssertPattern.MatchString(line) {
				assertions++
			}

			// Mocks
			if goMockPattern.MatchString(line) {
				mocks++
				patternCounts[styleMocked]++
			}
		}

	case "typescript", "javascript":
		for lineNum, line := range lines {
			lineNo := lineNum + 1

			// BDD style (describe/it)
			if jsDescribePattern.MatchString(line) {
				patternCounts[styleBDD]++
				styleLocations[styleBDD] = append(styleLocations[styleBDD], patterns.Location{
					FilePath:  dctx.File.Path,
					LineStart: lineNo,
					LineEnd:   lineNo,
					Snippet:   strings.TrimSpace(line),
				})
			}

			// Test cases
			if jsItPattern.MatchString(line) {
				testFunctions++
				locations = append(locations, patterns.Location{
					FilePath:  dctx.File.Path,
					LineStart: lineNo,
					LineEnd:   lineNo,
					Snippet:   strings.TrimSpace(line),
				})
			}

			// Assertions
			if jsExpectPattern.MatchString(line) {
				assertions++
			}

			// Mocks
			if jsMockPattern.MatchString(line) {
				mocks++
				patternCounts[styleMocked]++
			}
		}

	case "python":
		for lineNum, line := range lines {
			lineNo := lineNum + 1

			// Test functions
			if matches := pyTestFuncPattern.FindStringSubmatch(line); matches != nil {
				testFunctions++
				locations = append(locations, patterns.Location{
					FilePath:  dctx.File.Path,
					LineStart: lineNo,
					LineEnd:   lineNo,
					Snippet:   matches[1],
				})
			}

			// Test classes
			if pyTestClassPattern.MatchString(line) {
				patternCounts[styleUnitTest]++
			}

			// Assertions
			if pyAssertPattern.MatchString(line) {
				assertions++
			}

			// Fixtures
			if pyFixturePattern.MatchString(line) {
				patternCounts[styleTableDriven]++ // Fixtures are similar to table-driven in intent
			}

			// Mocks
			if pyMockPattern.MatchString(line) {
				mocks++
				patternCounts[styleMocked]++
			}
		}
	}

	if testFunctions == 0 {
		return nil, nil
	}

	// Add style locations to main locations
	for _, locs := range styleLocations {
		locations = append(locations, locs...)
	}

	// Determine dominant test style
	dominantStyle := styleUnitTest
	maxCount := 0
	for style, count := range patternCounts {
		if count > maxCount {
			maxCount = count
			dominantStyle = style
		}
	}

	// Calculate assertion density
	assertionDensity := float64(assertions) / float64(testFunctions)

	// Mark tests with no assertions as potential outliers
	// This is a heuristic - in reality we'd need more sophisticated analysis

	return &patterns.DetectionResult{
		Locations: locations,
		Outliers:  outliers,
		Confidence: patterns.ConfidenceFactors{
			Frequency:   patterns.CalculateFrequencyScore(testFunctions, 5),
			Consistency: min(assertionDensity/3.0, 1.0), // ~3 assertions per test is good
			Spread:      0.5,
			Age:         0.3,
		},
		Metadata: map[string]any{
			"test_functions":     testFunctions,
			"assertions":         assertions,
			"mocks":              mocks,
			"assertion_density":  assertionDensity,
			"dominant_style":     string(dominantStyle),
			"table_driven_count": patternCounts[styleTableDriven],
			"subtest_count":      patternCounts[styleSubtests],
			"bdd_style_count":    patternCounts[styleBDD],
			"mocked_count":       patternCounts[styleMocked],
		},
	}, nil
}

func isTestFile(path string) bool {
	lowerPath := strings.ToLower(path)

	// Go test files
	if strings.HasSuffix(lowerPath, "_test.go") {
		return true
	}

	// JavaScript/TypeScript test files
	if strings.Contains(lowerPath, ".test.") || strings.Contains(lowerPath, ".spec.") {
		return true
	}
	if strings.Contains(lowerPath, "__tests__") {
		return true
	}

	// Python test files
	if strings.HasPrefix(strings.ToLower(getFileName(path)), "test_") {
		return true
	}
	if strings.HasSuffix(lowerPath, "_test.py") {
		return true
	}

	return false
}

func getFileName(path string) string {
	// Get the last component of the path
	parts := strings.Split(strings.ReplaceAll(path, "\\", "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}

func init() {
	patterns.MustRegister(NewTestPatternDetector())
}
