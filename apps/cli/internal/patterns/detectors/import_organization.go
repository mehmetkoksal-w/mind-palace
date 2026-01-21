// Package detectors contains built-in pattern detectors.
package detectors

import (
	"context"
	"regexp"
	"strings"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/patterns"
)

// ImportOrganizationDetector detects import organization patterns.
// It looks for consistent grouping and ordering of imports.
type ImportOrganizationDetector struct {
	patterns.BaseDetector
}

// NewImportOrganizationDetector creates a new import organization detector.
func NewImportOrganizationDetector() *ImportOrganizationDetector {
	return &ImportOrganizationDetector{
		BaseDetector: patterns.NewBaseDetector(
			"imports/organization",
			patterns.CategoryStructural,
			"import-organization",
			"Import Organization",
			"Detects consistent import grouping patterns (stdlib, external, internal)",
			[]string{"go", "typescript", "javascript", "python"},
		),
	}
}

type importGroup int

const (
	groupStdlib importGroup = iota
	groupExternal
	groupInternal
	groupRelative
	groupUnknown
)

var (
	// TypeScript/JavaScript patterns
	tsNodeModulesPattern = regexp.MustCompile(`^["'][@a-zA-Z]`)
	tsRelativePattern    = regexp.MustCompile(`^["']\.\.?/`)

	// Python patterns
	pyStdlibPattern  = regexp.MustCompile(`^(import|from)\s+(os|sys|re|json|time|datetime|collections|itertools|functools|typing|pathlib|logging|unittest|math|random|copy|io|csv|http|urllib|socket|threading|multiprocessing|subprocess|shutil|glob|tempfile|pickle|hashlib|base64|struct|enum|dataclasses|abc|contextlib|operator|string|textwrap|difflib|heapq|bisect|array|weakref|types|gc|inspect|dis|traceback|warnings|argparse|getopt|configparser|secrets|uuid|platform|locale|gettext|calendar|pprint|decimal|fractions|statistics|cmath|numbers)`)
	pyRelativePattern = regexp.MustCompile(`^from\s+\.`)
)

// Detect implements the Detector interface.
//
func (d *ImportOrganizationDetector) Detect(_ context.Context, dctx *patterns.DetectionContext) (*patterns.DetectionResult, error) {
	content := string(dctx.FileContent)
	lines := strings.Split(content, "\n")
	lang := dctx.File.Language

	var locations []patterns.Location
	var outliers []patterns.Location

	// Track import groups found in order
	var groups []importGroup
	groupLocations := make(map[importGroup][]patterns.Location)

	// Track if imports are grouped (separated by blank lines)
	inImportBlock := false
	lastGroup := groupUnknown
	hasBlankLineBetweenGroups := false
	importGroupsFound := 0

	for lineNum, line := range lines {
		lineNo := lineNum + 1
		trimmed := strings.TrimSpace(line)

		switch lang {
		case "go":
			if strings.HasPrefix(trimmed, "import (") {
				inImportBlock = true
				continue
			}
			if inImportBlock && trimmed == ")" {
				inImportBlock = false
				continue
			}
			if inImportBlock || strings.HasPrefix(trimmed, "import ") {
				if trimmed == "" {
					hasBlankLineBetweenGroups = true
					importGroupsFound++
					continue
				}
				group := classifyGoImport(trimmed, dctx.File.Path)
				if group != groupUnknown {
					loc := patterns.Location{
						FilePath:  dctx.File.Path,
						LineStart: lineNo,
						LineEnd:   lineNo,
						Snippet:   trimmed,
					}
					groups = append(groups, group)
					groupLocations[group] = append(groupLocations[group], loc)
					lastGroup = group
				}
			}

		case "typescript", "javascript":
			if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "import{") {
				group := classifyTSImport(trimmed)
				loc := patterns.Location{
					FilePath:  dctx.File.Path,
					LineStart: lineNo,
					LineEnd:   lineNo,
					Snippet:   trimmed,
				}
				groups = append(groups, group)
				groupLocations[group] = append(groupLocations[group], loc)

				if lastGroup != groupUnknown && lastGroup != group {
					// Check if there was a blank line before this import
					if lineNum > 0 && strings.TrimSpace(lines[lineNum-1]) == "" {
						hasBlankLineBetweenGroups = true
					}
				}
				lastGroup = group
			}

		case "python":
			if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "from ") {
				group := classifyPythonImport(trimmed)
				loc := patterns.Location{
					FilePath:  dctx.File.Path,
					LineStart: lineNo,
					LineEnd:   lineNo,
					Snippet:   trimmed,
				}
				groups = append(groups, group)
				groupLocations[group] = append(groupLocations[group], loc)

				if lastGroup != groupUnknown && lastGroup != group {
					if lineNum > 0 && strings.TrimSpace(lines[lineNum-1]) == "" {
						hasBlankLineBetweenGroups = true
					}
				}
				lastGroup = group
			}
		}
	}

	if len(groups) == 0 {
		return nil, nil
	}

	// Check if imports follow expected order: stdlib -> external -> internal -> relative
	isOrdered := checkImportOrder(groups)

	// All imports that follow the pattern go to locations
	for _, locs := range groupLocations {
		locations = append(locations, locs...)
	}

	// If not ordered, mark out-of-order imports as outliers
	if !isOrdered {
		outliers = findOutOfOrderImports(groups, groupLocations)
	}

	// Calculate confidence
	consistency := 1.0
	if !isOrdered {
		consistency = 0.5
	}
	if hasBlankLineBetweenGroups {
		consistency = min(consistency+0.3, 1.0)
	}

	return &patterns.DetectionResult{
		Locations: locations,
		Outliers:  outliers,
		Confidence: patterns.ConfidenceFactors{
			Frequency:   patterns.CalculateFrequencyScore(len(locations), 5),
			Consistency: consistency,
			Spread:      0.5,
			Age:         0.3,
		},
		Metadata: map[string]any{
			"is_ordered":                 isOrdered,
			"has_group_separation":       hasBlankLineBetweenGroups,
			"stdlib_count":               len(groupLocations[groupStdlib]),
			"external_count":             len(groupLocations[groupExternal]),
			"internal_count":             len(groupLocations[groupInternal]),
			"relative_count":             len(groupLocations[groupRelative]),
			"distinct_groups":            importGroupsFound,
		},
	}, nil
}

func classifyGoImport(line, _ string) importGroup {
	// Extract the import path
	start := strings.Index(line, `"`)
	if start == -1 {
		return groupUnknown
	}
	end := strings.LastIndex(line, `"`)
	if end <= start {
		return groupUnknown
	}
	importPath := line[start+1 : end]

	// Check if it's stdlib (no dots in path before first slash)
	if !strings.Contains(strings.Split(importPath, "/")[0], ".") {
		return groupStdlib
	}

	// Check if it's internal (same module)
	// This is a heuristic - internal imports often contain the project name
	if strings.Contains(importPath, "internal/") {
		return groupInternal
	}

	return groupExternal
}

func classifyTSImport(line string) importGroup {
	if tsRelativePattern.MatchString(line) {
		return groupRelative
	}
	if tsNodeModulesPattern.MatchString(strings.TrimPrefix(strings.TrimPrefix(line, "import "), "import{")) {
		return groupExternal
	}
	return groupInternal
}

func classifyPythonImport(line string) importGroup {
	if pyRelativePattern.MatchString(line) {
		return groupRelative
	}
	if pyStdlibPattern.MatchString(line) {
		return groupStdlib
	}
	return groupExternal
}

func checkImportOrder(groups []importGroup) bool {
	if len(groups) <= 1 {
		return true
	}

	// Expected order: stdlib < external < internal < relative
	groupOrder := map[importGroup]int{
		groupStdlib:   0,
		groupExternal: 1,
		groupInternal: 2,
		groupRelative: 3,
		groupUnknown:  4,
	}

	maxSeen := -1
	for _, g := range groups {
		order := groupOrder[g]
		if order < maxSeen {
			return false
		}
		maxSeen = order
	}
	return true
}

func findOutOfOrderImports(groups []importGroup, groupLocations map[importGroup][]patterns.Location) []patterns.Location {
	// Simple implementation: mark imports that appear after a "later" group
	var outliers []patterns.Location

	groupOrder := map[importGroup]int{
		groupStdlib:   0,
		groupExternal: 1,
		groupInternal: 2,
		groupRelative: 3,
	}

	maxSeen := -1
	for i, g := range groups {
		order := groupOrder[g]
		if order < maxSeen {
			// This import is out of order
			if locs, ok := groupLocations[g]; ok && i < len(locs) {
				loc := locs[0]
				loc.IsOutlier = true
				loc.OutlierReason = "Import appears after imports that should come later"
				outliers = append(outliers, loc)
			}
		}
		if order > maxSeen {
			maxSeen = order
		}
	}

	return outliers
}

func init() {
	patterns.MustRegister(NewImportOrganizationDetector())
}
