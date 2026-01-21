// Package detectors contains built-in pattern detectors.
package detectors

import (
	"context"
	"regexp"
	"strings"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/patterns"
)

// LoggingPatternDetector detects logging patterns and conventions.
// It identifies consistent logging libraries, levels, and formats.
type LoggingPatternDetector struct {
	patterns.BaseDetector
}

// NewLoggingPatternDetector creates a new logging pattern detector.
func NewLoggingPatternDetector() *LoggingPatternDetector {
	return &LoggingPatternDetector{
		BaseDetector: patterns.NewBaseDetector(
			"logging/logging-patterns",
			patterns.CategoryLogging,
			"logging-patterns",
			"Logging Patterns",
			"Detects logging conventions (libraries, levels, structured logging)",
			[]string{"go", "typescript", "javascript", "python"},
		),
	}
}

var (
	// Go logging patterns
	goStdLogPattern     = regexp.MustCompile(`log\.(Print|Printf|Println|Fatal|Fatalf|Fatalln|Panic|Panicf|Panicln)\s*\(`)
	goZapPattern        = regexp.MustCompile(`(zap\.(L|S|New)|logger\.(Info|Debug|Warn|Error|Fatal|With))\s*\(`)
	goLogrusPattern     = regexp.MustCompile(`(logrus\.|log\.(WithField|WithFields|WithError))\s*\(`)
	goSlogPattern       = regexp.MustCompile(`slog\.(Info|Debug|Warn|Error|With|Log)\s*\(`)
	goZerologPattern    = regexp.MustCompile(`(zerolog\.|log\.(Info|Debug|Warn|Error|Fatal)\(\)\.Msg)`)

	// JavaScript/TypeScript logging patterns
	jsConsolePattern = regexp.MustCompile(`console\.(log|info|warn|error|debug|trace)\s*\(`)
	jsWinstonPattern = regexp.MustCompile(`(winston|logger)\.(info|debug|warn|error|log)\s*\(`)
	jsPinoPattern    = regexp.MustCompile(`(pino|logger)\.(info|debug|warn|error|fatal|trace)\s*\(`)

	// Python logging patterns
	pyLoggingPattern    = regexp.MustCompile(`logging\.(info|debug|warning|error|critical|exception)\s*\(`)
	pyLoggerPattern     = regexp.MustCompile(`(logger|log)\.(info|debug|warning|error|critical|exception)\s*\(`)
	pyPrintPattern      = regexp.MustCompile(`print\s*\(`)
	pyLogStructPattern  = regexp.MustCompile(`structlog\.`)

	// Common structured logging patterns
	structuredLogPattern = regexp.MustCompile(`\.(With|WithField|WithFields|WithError|WithContext)\s*\(`)
)

type loggingStyle string

const (
	styleStdLib      loggingStyle = "stdlib"
	styleStructured  loggingStyle = "structured"
	styleConsole     loggingStyle = "console"
	styleCustom      loggingStyle = "custom"
)

type loggingLibrary string

const (
	libGoStd     loggingLibrary = "go-log"
	libZap       loggingLibrary = "zap"
	libLogrus    loggingLibrary = "logrus"
	libSlog      loggingLibrary = "slog"
	libZerolog   loggingLibrary = "zerolog"
	libConsole   loggingLibrary = "console"
	libWinston   loggingLibrary = "winston"
	libPino      loggingLibrary = "pino"
	libPyLogging loggingLibrary = "python-logging"
	libStructlog loggingLibrary = "structlog"
	libPrint     loggingLibrary = "print"
)

// Detect implements the Detector interface.
//
//nolint:gocognit,gocyclo // pattern detection is complex by design
func (d *LoggingPatternDetector) Detect(_ context.Context, dctx *patterns.DetectionContext) (*patterns.DetectionResult, error) {
	content := string(dctx.FileContent)
	lines := strings.Split(content, "\n")
	lang := dctx.File.Language

	var locations []patterns.Location
	var outliers []patterns.Location

	// Track logging usage
	libraryCounts := make(map[loggingLibrary]int)
	libraryLocations := make(map[loggingLibrary][]patterns.Location)

	levelCounts := make(map[string]int) // info, debug, warn, error, etc.
	structuredCount := 0
	totalLogCalls := 0

	for lineNum, line := range lines {
		lineNo := lineNum + 1

		switch lang {
		case "go":
			// Check for standard library log
			if goStdLogPattern.MatchString(line) {
				totalLogCalls++
				libraryCounts[libGoStd]++
				addLogLocation(libraryLocations, libGoStd, dctx.File.Path, lineNo, line)
				extractLogLevel(line, levelCounts)
			}

			// Check for zap
			if goZapPattern.MatchString(line) {
				totalLogCalls++
				libraryCounts[libZap]++
				addLogLocation(libraryLocations, libZap, dctx.File.Path, lineNo, line)
				extractLogLevel(line, levelCounts)
			}

			// Check for logrus
			if goLogrusPattern.MatchString(line) {
				totalLogCalls++
				libraryCounts[libLogrus]++
				addLogLocation(libraryLocations, libLogrus, dctx.File.Path, lineNo, line)
				extractLogLevel(line, levelCounts)
			}

			// Check for slog
			if goSlogPattern.MatchString(line) {
				totalLogCalls++
				libraryCounts[libSlog]++
				addLogLocation(libraryLocations, libSlog, dctx.File.Path, lineNo, line)
				extractLogLevel(line, levelCounts)
			}

			// Check for zerolog
			if goZerologPattern.MatchString(line) {
				totalLogCalls++
				libraryCounts[libZerolog]++
				addLogLocation(libraryLocations, libZerolog, dctx.File.Path, lineNo, line)
				extractLogLevel(line, levelCounts)
			}

		case "typescript", "javascript":
			// Check for console
			if jsConsolePattern.MatchString(line) {
				totalLogCalls++
				libraryCounts[libConsole]++
				addLogLocation(libraryLocations, libConsole, dctx.File.Path, lineNo, line)
				extractLogLevel(line, levelCounts)
			}

			// Check for winston
			if jsWinstonPattern.MatchString(line) {
				totalLogCalls++
				libraryCounts[libWinston]++
				addLogLocation(libraryLocations, libWinston, dctx.File.Path, lineNo, line)
				extractLogLevel(line, levelCounts)
			}

			// Check for pino
			if jsPinoPattern.MatchString(line) {
				totalLogCalls++
				libraryCounts[libPino]++
				addLogLocation(libraryLocations, libPino, dctx.File.Path, lineNo, line)
				extractLogLevel(line, levelCounts)
			}

		case "python":
			// Check for print (potential outlier)
			if pyPrintPattern.MatchString(line) {
				totalLogCalls++
				libraryCounts[libPrint]++
				addLogLocation(libraryLocations, libPrint, dctx.File.Path, lineNo, line)
			}

			// Check for logging module
			if pyLoggingPattern.MatchString(line) || pyLoggerPattern.MatchString(line) {
				totalLogCalls++
				libraryCounts[libPyLogging]++
				addLogLocation(libraryLocations, libPyLogging, dctx.File.Path, lineNo, line)
				extractLogLevel(line, levelCounts)
			}

			// Check for structlog
			if pyLogStructPattern.MatchString(line) {
				totalLogCalls++
				libraryCounts[libStructlog]++
				addLogLocation(libraryLocations, libStructlog, dctx.File.Path, lineNo, line)
			}
		}

		// Check for structured logging patterns
		if structuredLogPattern.MatchString(line) {
			structuredCount++
		}
	}

	if totalLogCalls == 0 {
		return nil, nil
	}

	// Determine dominant library
	var dominantLib loggingLibrary
	maxCount := 0
	for lib, count := range libraryCounts {
		if count > maxCount {
			maxCount = count
			dominantLib = lib
		}
	}

	// Build locations and outliers
	for lib, locs := range libraryLocations {
		if lib == dominantLib {
			locations = append(locations, locs...)
		} else {
			// Non-dominant libraries are potential outliers
			for _, loc := range locs {
				loc.IsOutlier = true
				loc.OutlierReason = "Uses " + string(lib) + " instead of " + string(dominantLib)
				outliers = append(outliers, loc)
			}
		}
	}

	// Mark print statements as outliers in Python (if there are proper logging calls)
	if lang == "python" && libraryCounts[libPyLogging] > 0 {
		if printLocs, ok := libraryLocations[libPrint]; ok {
			for _, loc := range printLocs {
				loc.IsOutlier = true
				loc.OutlierReason = "Uses print() instead of logging module"
				outliers = append(outliers, loc)
			}
		}
	}

	// Mark console.log as outliers in JS/TS if proper logger exists
	if (lang == "typescript" || lang == "javascript") && (libraryCounts[libWinston] > 0 || libraryCounts[libPino] > 0) {
		if consoleLocs, ok := libraryLocations[libConsole]; ok {
			for _, loc := range consoleLocs {
				loc.IsOutlier = true
				loc.OutlierReason = "Uses console.log instead of structured logger"
				outliers = append(outliers, loc)
			}
		}
	}

	// Determine logging style
	style := styleStdLib
	if structuredCount > totalLogCalls/2 {
		style = styleStructured
	}
	if dominantLib == libConsole || dominantLib == libPrint {
		style = styleConsole
	}

	// Calculate consistency
	consistency := float64(len(locations)) / float64(totalLogCalls)

	return &patterns.DetectionResult{
		Locations: locations,
		Outliers:  outliers,
		Confidence: patterns.ConfidenceFactors{
			Frequency:   patterns.CalculateFrequencyScore(totalLogCalls, 10),
			Consistency: consistency,
			Spread:      0.5,
			Age:         0.3,
		},
		Metadata: map[string]any{
			"dominant_library":    string(dominantLib),
			"logging_style":       string(style),
			"total_log_calls":     totalLogCalls,
			"structured_count":    structuredCount,
			"level_distribution":  levelCounts,
			"libraries_used":      getLibraryNames(libraryCounts),
		},
	}, nil
}

func addLogLocation(m map[loggingLibrary][]patterns.Location, lib loggingLibrary, path string, line int, content string) {
	loc := patterns.Location{
		FilePath:  path,
		LineStart: line,
		LineEnd:   line,
		Snippet:   strings.TrimSpace(content),
	}
	m[lib] = append(m[lib], loc)
}

func extractLogLevel(line string, levelCounts map[string]int) {
	lineLower := strings.ToLower(line)
	levels := []string{"debug", "info", "warn", "warning", "error", "fatal", "critical", "trace"}
	for _, level := range levels {
		if strings.Contains(lineLower, level) {
			levelCounts[level]++
			return
		}
	}
}

func getLibraryNames(counts map[loggingLibrary]int) []string {
	names := make([]string, 0, len(counts))
	for lib := range counts {
		names = append(names, string(lib))
	}
	return names
}

func init() {
	patterns.MustRegister(NewLoggingPatternDetector())
}
