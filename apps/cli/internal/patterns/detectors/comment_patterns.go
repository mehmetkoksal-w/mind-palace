// Package detectors contains built-in pattern detectors.
package detectors

import (
	"context"
	"regexp"
	"strings"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/analysis"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/patterns"
)

// CommentPatternDetector detects comment and documentation patterns.
// It identifies consistent documentation styles like JSDoc, godoc, docstrings, etc.
type CommentPatternDetector struct {
	patterns.BaseDetector
}

// NewCommentPatternDetector creates a new comment pattern detector.
func NewCommentPatternDetector() *CommentPatternDetector {
	return &CommentPatternDetector{
		BaseDetector: patterns.NewBaseDetector(
			"documentation/comment-patterns",
			patterns.CategoryDocumentation,
			"comment-patterns",
			"Comment Patterns",
			"Detects consistent documentation and comment styles",
			[]string{}, // All languages
		),
	}
}

var (
	// Go doc pattern: // FunctionName ...
	godocPattern = regexp.MustCompile(`^//\s+[A-Z][a-zA-Z]+\s+`)

	// TODO/FIXME/NOTE comments
	todoPattern = regexp.MustCompile(`(?i)(TODO|FIXME|NOTE|HACK|XXX|BUG)\s*[:\-]?\s*`)

	// Single line comment patterns
	singleLineCommentPattern = regexp.MustCompile(`^\s*(//|#)\s*`)
)

type commentStyle string

const (
	styleJSDoc         commentStyle = "jsdoc"
	styleGoDoc         commentStyle = "godoc"
	stylePythonDoc     commentStyle = "python-docstring"
	styleInlineComment commentStyle = "inline"
	styleBlockComment  commentStyle = "block"
	styleTodoComment   commentStyle = "todo"
)

// Detect implements the Detector interface.
func (d *CommentPatternDetector) Detect(ctx context.Context, dctx *patterns.DetectionContext) (*patterns.DetectionResult, error) {
	content := string(dctx.FileContent)
	lines := strings.Split(content, "\n")
	lang := dctx.File.Language

	var locations []patterns.Location
	var outliers []patterns.Location

	// Count different comment styles
	commentCounts := make(map[commentStyle]int)
	styleLocations := make(map[commentStyle][]patterns.Location)

	// Track function documentation
	documentedFunctions := 0
	undocumentedFunctions := 0

	// Analyze line by line
	inBlockComment := false
	inPythonDocstring := false
	blockStartLine := 0

	for lineNum, line := range lines {
		lineNo := lineNum + 1
		trimmed := strings.TrimSpace(line)

		// Check for JSDoc
		if strings.HasPrefix(trimmed, "/**") && !strings.HasSuffix(trimmed, "*/") {
			inBlockComment = true
			blockStartLine = lineNo
			commentCounts[styleJSDoc]++
			continue
		}
		if inBlockComment && strings.HasSuffix(trimmed, "*/") {
			loc := patterns.Location{
				FilePath:  dctx.File.Path,
				LineStart: blockStartLine,
				LineEnd:   lineNo,
				Snippet:   "/** ... */",
			}
			styleLocations[styleJSDoc] = append(styleLocations[styleJSDoc], loc)
			inBlockComment = false
			continue
		}

		// Check for Python docstrings
		if (lang == "python") && (strings.HasPrefix(trimmed, `"""`) || strings.HasPrefix(trimmed, `'''`)) {
			if inPythonDocstring {
				loc := patterns.Location{
					FilePath:  dctx.File.Path,
					LineStart: blockStartLine,
					LineEnd:   lineNo,
					Snippet:   "docstring",
				}
				styleLocations[stylePythonDoc] = append(styleLocations[stylePythonDoc], loc)
				commentCounts[stylePythonDoc]++
				inPythonDocstring = false
			} else {
				inPythonDocstring = true
				blockStartLine = lineNo
			}
			continue
		}

		// Check for Go doc comments
		if lang == "go" && godocPattern.MatchString(line) {
			commentCounts[styleGoDoc]++
			loc := patterns.Location{
				FilePath:  dctx.File.Path,
				LineStart: lineNo,
				LineEnd:   lineNo,
				Snippet:   trimmed,
			}
			styleLocations[styleGoDoc] = append(styleLocations[styleGoDoc], loc)
			continue
		}

		// Check for TODO/FIXME comments
		if todoPattern.MatchString(line) {
			commentCounts[styleTodoComment]++
			loc := patterns.Location{
				FilePath:  dctx.File.Path,
				LineStart: lineNo,
				LineEnd:   lineNo,
				Snippet:   trimmed,
			}
			styleLocations[styleTodoComment] = append(styleLocations[styleTodoComment], loc)
			continue
		}

		// Check for inline comments
		if singleLineCommentPattern.MatchString(line) {
			commentCounts[styleInlineComment]++
		}
	}

	// Check function documentation using symbols
	functions := extractFunctions(dctx.File.Symbols)
	for _, fn := range functions {
		// Check if there's a comment immediately before the function
		if fn.LineStart > 1 {
			prevLine := lines[fn.LineStart-2] // -2 because lines are 0-indexed and we want line before
			if singleLineCommentPattern.MatchString(prevLine) || strings.HasSuffix(strings.TrimSpace(prevLine), "*/") {
				documentedFunctions++
			} else {
				undocumentedFunctions++
			}
		} else {
			undocumentedFunctions++
		}
	}

	// Build locations from all styles
	for _, locs := range styleLocations {
		locations = append(locations, locs...)
	}

	// Mark undocumented exported functions as outliers (for Go)
	if lang == "go" && undocumentedFunctions > 0 {
		for _, fn := range functions {
			if len(fn.Name) > 0 && fn.Name[0] >= 'A' && fn.Name[0] <= 'Z' {
				// Exported function
				if fn.LineStart > 1 {
					prevLine := lines[fn.LineStart-2]
					if !singleLineCommentPattern.MatchString(prevLine) {
						outliers = append(outliers, patterns.Location{
							FilePath:      dctx.File.Path,
							LineStart:     fn.LineStart,
							LineEnd:       fn.LineStart,
							Snippet:       fn.Name,
							IsOutlier:     true,
							OutlierReason: "Exported function lacks documentation comment",
						})
					}
				}
			}
		}
	}

	totalComments := 0
	for _, count := range commentCounts {
		totalComments += count
	}

	if totalComments == 0 && len(functions) == 0 {
		return nil, nil
	}

	// Calculate documentation ratio
	docRatio := 0.0
	if documentedFunctions+undocumentedFunctions > 0 {
		docRatio = float64(documentedFunctions) / float64(documentedFunctions+undocumentedFunctions)
	}

	// Determine dominant style
	dominantStyle := ""
	maxCount := 0
	for style, count := range commentCounts {
		if count > maxCount {
			maxCount = count
			dominantStyle = string(style)
		}
	}

	return &patterns.DetectionResult{
		Locations: locations,
		Outliers:  outliers,
		Confidence: patterns.ConfidenceFactors{
			Frequency:   patterns.CalculateFrequencyScore(totalComments, 10),
			Consistency: docRatio,
			Spread:      0.5,
			Age:         0.3,
		},
		Metadata: map[string]any{
			"jsdoc_count":            commentCounts[styleJSDoc],
			"godoc_count":            commentCounts[styleGoDoc],
			"python_docstring_count": commentCounts[stylePythonDoc],
			"todo_count":             commentCounts[styleTodoComment],
			"inline_comment_count":   commentCounts[styleInlineComment],
			"documented_functions":   documentedFunctions,
			"undocumented_functions": undocumentedFunctions,
			"documentation_ratio":    docRatio,
			"dominant_style":         dominantStyle,
		},
	}, nil
}

// extractFunctions extracts function symbols from a list of symbols.
func extractFunctions(symbols []analysis.Symbol) []analysis.Symbol {
	var functions []analysis.Symbol
	for _, sym := range symbols {
		if sym.Kind == analysis.KindFunction || sym.Kind == analysis.KindMethod {
			functions = append(functions, sym)
		}
	}
	return functions
}

func init() {
	patterns.MustRegister(NewCommentPatternDetector())
}
