// Package detectors contains built-in pattern detectors.
package detectors

import (
	"context"
	"regexp"
	"strings"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/patterns"
)

// APIPatternDetector detects API design patterns.
// It identifies REST/HTTP patterns like response envelopes, error handling, etc.
type APIPatternDetector struct {
	patterns.BaseDetector
}

// NewAPIPatternDetector creates a new API pattern detector.
func NewAPIPatternDetector() *APIPatternDetector {
	return &APIPatternDetector{
		BaseDetector: patterns.NewBaseDetector(
			"api/api-patterns",
			patterns.CategoryAPI,
			"api-patterns",
			"API Patterns",
			"Detects API design patterns (REST, response envelopes, versioning)",
			[]string{"go", "typescript", "javascript", "python"},
		),
	}
}

var (
	// Go HTTP patterns
	goHTTPHandlerPattern   = regexp.MustCompile(`func\s+\w+\s*\(\s*w\s+http\.ResponseWriter`)
	goGinHandlerPattern    = regexp.MustCompile(`func\s+\w+\s*\(\s*c\s+\*gin\.Context`)
	goEchoHandlerPattern   = regexp.MustCompile(`func\s+\w+\s*\(\s*c\s+echo\.Context`)
	goChiRoutePattern      = regexp.MustCompile(`r\.(Get|Post|Put|Delete|Patch)\s*\(`)
	goGinRoutePattern      = regexp.MustCompile(`(router|r|g)\.(GET|POST|PUT|DELETE|PATCH)\s*\(`)
	goJSONEncodePattern    = regexp.MustCompile(`json\.NewEncoder\s*\([^)]*\)\s*\.Encode`)
	goJSONMarshalPattern   = regexp.MustCompile(`json\.Marshal`)
	goResponseEnvelope     = regexp.MustCompile(`(data|result|response|payload)\s*:`)

	// JavaScript/TypeScript patterns
	jsExpressPattern       = regexp.MustCompile(`(app|router)\.(get|post|put|delete|patch)\s*\(`)
	jsFastifyPattern       = regexp.MustCompile(`(fastify|server)\.(get|post|put|delete|patch)\s*\(`)
	jsResJSONPattern       = regexp.MustCompile(`res\.(json|send)\s*\(`)
	jsResponseEnvelope     = regexp.MustCompile(`(data|result|response|payload|success|error)\s*:`)
	jsStatusCodePattern    = regexp.MustCompile(`res\.status\s*\(\s*\d+\s*\)`)
	jsFetchPattern         = regexp.MustCompile(`fetch\s*\(`)
	jsAxiosPattern         = regexp.MustCompile(`axios\.(get|post|put|delete|patch)`)

	// Python patterns
	pyFastAPIPattern       = regexp.MustCompile(`@(app|router)\.(get|post|put|delete|patch)`)
	pyFlaskPattern         = regexp.MustCompile(`@(app|blueprint)\.(route|get|post|put|delete)`)
	pyDjangoViewPattern    = regexp.MustCompile(`(APIView|ViewSet|GenericView)`)
	pyJSONResponsePattern  = regexp.MustCompile(`(JsonResponse|jsonify|JSONResponse)`)
	pyResponseEnvelope     = regexp.MustCompile(`["'](data|result|response|payload|success|error)["']\s*:`)
)

type apiStyle string

const (
	styleRESTful         apiStyle = "restful"
	styleEnvelope        apiStyle = "envelope"
	styleDirectResponse  apiStyle = "direct"
)

type apiFramework string

const (
	fwNetHTTP    apiFramework = "net/http"
	fwGin        apiFramework = "gin"
	fwEcho       apiFramework = "echo"
	fwChi        apiFramework = "chi"
	fwExpress    apiFramework = "express"
	fwFastify    apiFramework = "fastify"
	fwFastAPI    apiFramework = "fastapi"
	fwFlask      apiFramework = "flask"
	fwDjango     apiFramework = "django"
)

// Detect implements the Detector interface.
func (d *APIPatternDetector) Detect(ctx context.Context, dctx *patterns.DetectionContext) (*patterns.DetectionResult, error) {
	content := string(dctx.FileContent)
	lines := strings.Split(content, "\n")
	lang := dctx.File.Language

	var locations []patterns.Location
	var outliers []patterns.Location

	// Track API patterns
	frameworkCounts := make(map[apiFramework]int)
	frameworkLocations := make(map[apiFramework][]patterns.Location)

	handlerCount := 0
	routeCount := 0
	envelopeCount := 0
	directResponseCount := 0
	totalAPIUsages := 0

	for lineNum, line := range lines {
		lineNo := lineNum + 1

		switch lang {
		case "go":
			// Check for HTTP handlers
			if goHTTPHandlerPattern.MatchString(line) {
				handlerCount++
				totalAPIUsages++
				frameworkCounts[fwNetHTTP]++
				addAPILocation(frameworkLocations, fwNetHTTP, dctx.File.Path, lineNo, line)
			}
			if goGinHandlerPattern.MatchString(line) {
				handlerCount++
				totalAPIUsages++
				frameworkCounts[fwGin]++
				addAPILocation(frameworkLocations, fwGin, dctx.File.Path, lineNo, line)
			}
			if goEchoHandlerPattern.MatchString(line) {
				handlerCount++
				totalAPIUsages++
				frameworkCounts[fwEcho]++
				addAPILocation(frameworkLocations, fwEcho, dctx.File.Path, lineNo, line)
			}

			// Check for routes
			if goChiRoutePattern.MatchString(line) {
				routeCount++
				frameworkCounts[fwChi]++
			}
			if goGinRoutePattern.MatchString(line) {
				routeCount++
			}

			// Check for response patterns
			if goResponseEnvelope.MatchString(line) && (goJSONEncodePattern.MatchString(line) || goJSONMarshalPattern.MatchString(line)) {
				envelopeCount++
			} else if goJSONEncodePattern.MatchString(line) || goJSONMarshalPattern.MatchString(line) {
				directResponseCount++
			}

		case "typescript", "javascript":
			// Check for Express routes
			if jsExpressPattern.MatchString(line) {
				routeCount++
				totalAPIUsages++
				frameworkCounts[fwExpress]++
				addAPILocation(frameworkLocations, fwExpress, dctx.File.Path, lineNo, line)
			}

			// Check for Fastify routes
			if jsFastifyPattern.MatchString(line) {
				routeCount++
				totalAPIUsages++
				frameworkCounts[fwFastify]++
				addAPILocation(frameworkLocations, fwFastify, dctx.File.Path, lineNo, line)
			}

			// Check for response patterns
			if jsResJSONPattern.MatchString(line) {
				if jsResponseEnvelope.MatchString(line) {
					envelopeCount++
				} else {
					directResponseCount++
				}
			}

		case "python":
			// Check for FastAPI
			if pyFastAPIPattern.MatchString(line) {
				routeCount++
				totalAPIUsages++
				frameworkCounts[fwFastAPI]++
				addAPILocation(frameworkLocations, fwFastAPI, dctx.File.Path, lineNo, line)
			}

			// Check for Flask
			if pyFlaskPattern.MatchString(line) {
				routeCount++
				totalAPIUsages++
				frameworkCounts[fwFlask]++
				addAPILocation(frameworkLocations, fwFlask, dctx.File.Path, lineNo, line)
			}

			// Check for Django
			if pyDjangoViewPattern.MatchString(line) {
				handlerCount++
				totalAPIUsages++
				frameworkCounts[fwDjango]++
				addAPILocation(frameworkLocations, fwDjango, dctx.File.Path, lineNo, line)
			}

			// Check for response patterns
			if pyJSONResponsePattern.MatchString(line) {
				if pyResponseEnvelope.MatchString(line) {
					envelopeCount++
				} else {
					directResponseCount++
				}
			}
		}
	}

	if totalAPIUsages == 0 {
		return nil, nil
	}

	// Determine dominant framework
	var dominantFW apiFramework
	maxCount := 0
	for fw, count := range frameworkCounts {
		if count > maxCount {
			maxCount = count
			dominantFW = fw
		}
	}

	// Determine response style
	responseStyle := styleDirectResponse
	if envelopeCount > directResponseCount {
		responseStyle = styleEnvelope
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
			Frequency:   patterns.CalculateFrequencyScore(totalAPIUsages, 5),
			Consistency: consistency,
			Spread:      0.5,
			Age:         0.3,
		},
		Metadata: map[string]any{
			"dominant_framework":    string(dominantFW),
			"response_style":        string(responseStyle),
			"handler_count":         handlerCount,
			"route_count":           routeCount,
			"envelope_responses":    envelopeCount,
			"direct_responses":      directResponseCount,
			"uses_response_envelope": envelopeCount > 0,
		},
	}, nil
}

func addAPILocation(m map[apiFramework][]patterns.Location, fw apiFramework, path string, line int, content string) {
	loc := patterns.Location{
		FilePath:  path,
		LineStart: line,
		LineEnd:   line,
		Snippet:   strings.TrimSpace(content),
	}
	m[fw] = append(m[fw], loc)
}

func init() {
	patterns.MustRegister(NewAPIPatternDetector())
}
