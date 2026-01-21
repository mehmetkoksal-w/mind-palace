// Package detectors contains built-in pattern detectors.
package detectors

import (
	"context"
	"regexp"
	"strings"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/patterns"
)

// ConfigPatternDetector detects configuration patterns.
// It identifies how configuration is loaded and managed (env vars, config files, etc.).
type ConfigPatternDetector struct {
	patterns.BaseDetector
}

// NewConfigPatternDetector creates a new config pattern detector.
func NewConfigPatternDetector() *ConfigPatternDetector {
	return &ConfigPatternDetector{
		BaseDetector: patterns.NewBaseDetector(
			"config/config-patterns",
			patterns.CategoryConfig,
			"config-patterns",
			"Configuration Patterns",
			"Detects configuration management patterns (env vars, config files, validation)",
			[]string{"go", "typescript", "javascript", "python"},
		),
	}
}

var (
	// Go config patterns
	goEnvPattern      = regexp.MustCompile(`os\.(Getenv|LookupEnv)\s*\(`)
	goViperPattern    = regexp.MustCompile(`viper\.(Get|Set|Bind|Read)`)
	goEnvConfigPattern = regexp.MustCompile(`envconfig\.(Process|Usage)`)
	goFlagPattern     = regexp.MustCompile(`flag\.(String|Int|Bool|Parse)`)

	// JavaScript/TypeScript config patterns
	jsProcessEnvPattern = regexp.MustCompile(`process\.env\.`)
	jsDotenvPattern     = regexp.MustCompile(`(dotenv|require\s*\(\s*['"]dotenv['"])`)
	jsConfigFilePattern = regexp.MustCompile(`(config\.(json|js|ts)|\.env)`)
	jsZodPattern        = regexp.MustCompile(`z\.(object|string|number|boolean)`)

	// Python config patterns
	pyOsEnvPattern      = regexp.MustCompile(`os\.(environ|getenv)`)
	pyConfigParserPattern = regexp.MustCompile(`configparser\.|ConfigParser\(`)
	pyPydanticPattern   = regexp.MustCompile(`(BaseSettings|pydantic_settings)`)
	pyDotenvPattern     = regexp.MustCompile(`(dotenv|load_dotenv)`)
)

type configStyle string

const (
	styleEnvVar      configStyle = "env-var"
	styleConfigFile  configStyle = "config-file"
	styleFlag        configStyle = "flag"
	styleValidated   configStyle = "validated"
)

type configLibrary string

const (
	libOsGetenv    configLibrary = "os-getenv"
	libViper       configLibrary = "viper"
	libEnvconfig   configLibrary = "envconfig"
	libGoFlag      configLibrary = "go-flag"
	libProcessEnv  configLibrary = "process-env"
	libDotenv      configLibrary = "dotenv"
	libZod         configLibrary = "zod"
	libPyEnv       configLibrary = "py-os-environ"
	libConfigParser configLibrary = "configparser"
	libPydantic    configLibrary = "pydantic"
)

// Detect implements the Detector interface.
func (d *ConfigPatternDetector) Detect(ctx context.Context, dctx *patterns.DetectionContext) (*patterns.DetectionResult, error) {
	content := string(dctx.FileContent)
	lines := strings.Split(content, "\n")
	lang := dctx.File.Language

	var locations []patterns.Location
	var outliers []patterns.Location

	// Track configuration usage
	libraryCounts := make(map[configLibrary]int)
	libraryLocations := make(map[configLibrary][]patterns.Location)

	styleCounts := make(map[configStyle]int)
	totalConfigUsages := 0

	for lineNum, line := range lines {
		lineNo := lineNum + 1

		switch lang {
		case "go":
			// Check for os.Getenv
			if goEnvPattern.MatchString(line) {
				totalConfigUsages++
				libraryCounts[libOsGetenv]++
				styleCounts[styleEnvVar]++
				addConfigLocation(libraryLocations, libOsGetenv, dctx.File.Path, lineNo, line)
			}

			// Check for viper
			if goViperPattern.MatchString(line) {
				totalConfigUsages++
				libraryCounts[libViper]++
				styleCounts[styleConfigFile]++
				addConfigLocation(libraryLocations, libViper, dctx.File.Path, lineNo, line)
			}

			// Check for envconfig
			if goEnvConfigPattern.MatchString(line) {
				totalConfigUsages++
				libraryCounts[libEnvconfig]++
				styleCounts[styleValidated]++
				addConfigLocation(libraryLocations, libEnvconfig, dctx.File.Path, lineNo, line)
			}

			// Check for flag
			if goFlagPattern.MatchString(line) {
				totalConfigUsages++
				libraryCounts[libGoFlag]++
				styleCounts[styleFlag]++
				addConfigLocation(libraryLocations, libGoFlag, dctx.File.Path, lineNo, line)
			}

		case "typescript", "javascript":
			// Check for process.env
			if jsProcessEnvPattern.MatchString(line) {
				totalConfigUsages++
				libraryCounts[libProcessEnv]++
				styleCounts[styleEnvVar]++
				addConfigLocation(libraryLocations, libProcessEnv, dctx.File.Path, lineNo, line)
			}

			// Check for dotenv
			if jsDotenvPattern.MatchString(line) {
				totalConfigUsages++
				libraryCounts[libDotenv]++
				styleCounts[styleConfigFile]++
				addConfigLocation(libraryLocations, libDotenv, dctx.File.Path, lineNo, line)
			}

			// Check for zod (validation)
			if jsZodPattern.MatchString(line) {
				styleCounts[styleValidated]++
				libraryCounts[libZod]++
				addConfigLocation(libraryLocations, libZod, dctx.File.Path, lineNo, line)
			}

		case "python":
			// Check for os.environ/getenv
			if pyOsEnvPattern.MatchString(line) {
				totalConfigUsages++
				libraryCounts[libPyEnv]++
				styleCounts[styleEnvVar]++
				addConfigLocation(libraryLocations, libPyEnv, dctx.File.Path, lineNo, line)
			}

			// Check for configparser
			if pyConfigParserPattern.MatchString(line) {
				totalConfigUsages++
				libraryCounts[libConfigParser]++
				styleCounts[styleConfigFile]++
				addConfigLocation(libraryLocations, libConfigParser, dctx.File.Path, lineNo, line)
			}

			// Check for pydantic settings
			if pyPydanticPattern.MatchString(line) {
				totalConfigUsages++
				libraryCounts[libPydantic]++
				styleCounts[styleValidated]++
				addConfigLocation(libraryLocations, libPydantic, dctx.File.Path, lineNo, line)
			}

			// Check for python-dotenv
			if pyDotenvPattern.MatchString(line) {
				totalConfigUsages++
				libraryCounts[libDotenv]++
				styleCounts[styleConfigFile]++
				addConfigLocation(libraryLocations, libDotenv, dctx.File.Path, lineNo, line)
			}
		}
	}

	if totalConfigUsages == 0 {
		return nil, nil
	}

	// Determine dominant library
	var dominantLib configLibrary
	maxCount := 0
	for lib, count := range libraryCounts {
		if count > maxCount {
			maxCount = count
			dominantLib = lib
		}
	}

	// Determine dominant style
	var dominantStyle configStyle
	maxStyleCount := 0
	for style, count := range styleCounts {
		if count > maxStyleCount {
			maxStyleCount = count
			dominantStyle = style
		}
	}

	// Build locations and identify outliers (mixed patterns)
	for lib, locs := range libraryLocations {
		if lib == dominantLib {
			locations = append(locations, locs...)
		} else {
			// Non-dominant libraries could be outliers in a consistent codebase
			for _, loc := range locs {
				loc.IsOutlier = true
				loc.OutlierReason = "Uses " + string(lib) + " instead of " + string(dominantLib)
				outliers = append(outliers, loc)
			}
		}
	}

	// Calculate consistency
	consistency := float64(len(locations)) / float64(totalConfigUsages)

	// Check if validation is used
	hasValidation := styleCounts[styleValidated] > 0

	return &patterns.DetectionResult{
		Locations: locations,
		Outliers:  outliers,
		Confidence: patterns.ConfidenceFactors{
			Frequency:   patterns.CalculateFrequencyScore(totalConfigUsages, 5),
			Consistency: consistency,
			Spread:      0.5,
			Age:         0.3,
		},
		Metadata: map[string]any{
			"dominant_library":   string(dominantLib),
			"dominant_style":     string(dominantStyle),
			"total_config_uses":  totalConfigUsages,
			"has_validation":     hasValidation,
			"env_var_count":      styleCounts[styleEnvVar],
			"config_file_count":  styleCounts[styleConfigFile],
			"flag_count":         styleCounts[styleFlag],
			"validated_count":    styleCounts[styleValidated],
		},
	}, nil
}

func addConfigLocation(m map[configLibrary][]patterns.Location, lib configLibrary, path string, line int, content string) {
	loc := patterns.Location{
		FilePath:  path,
		LineStart: line,
		LineEnd:   line,
		Snippet:   strings.TrimSpace(content),
	}
	m[lib] = append(m[lib], loc)
}

func init() {
	patterns.MustRegister(NewConfigPatternDetector())
}
