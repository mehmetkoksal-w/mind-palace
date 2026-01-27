package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// MonorepoManager represents a detected monorepo tool
type MonorepoManager string

const (
	MonorepoNone       MonorepoManager = ""
	MonorepoPnpm       MonorepoManager = "pnpm"
	MonorepoYarn       MonorepoManager = "yarn"
	MonorepoNpm        MonorepoManager = "npm"
	MonorepoLerna      MonorepoManager = "lerna"
	MonorepoTurborepo  MonorepoManager = "turborepo"
	MonorepoNx         MonorepoManager = "nx"
	MonorepoRush       MonorepoManager = "rush"
	MonorepoMelos      MonorepoManager = "melos"
	MonorepoBazel      MonorepoManager = "bazel"
	MonorepoPants      MonorepoManager = "pants"
	MonorepoCargoWs    MonorepoManager = "cargo-workspace"
	MonorepoGradleComp MonorepoManager = "gradle-composite"
	MonorepoMavenMulti MonorepoManager = "maven-multi"
	MonorepoDotnetSln  MonorepoManager = "dotnet-sln"
	MonorepoGoWork     MonorepoManager = "go-workspace"
)

// MonorepoInfo holds comprehensive monorepo detection results
type MonorepoInfo struct {
	IsMonorepo   bool                `json:"isMonorepo"`
	Manager      MonorepoManager     `json:"manager,omitempty"`
	Confidence   float64             `json:"confidence"` // 0.0 to 1.0
	Projects     []ProjectInfo       `json:"projects"`
	RootPath     string              `json:"rootPath"`
	ConfigFile   string              `json:"configFile,omitempty"`
	CustomPaths  []string            `json:"customPaths,omitempty"`  // User-defined patterns
	Dependencies map[string][]string `json:"dependencies,omitempty"` // Inter-project deps
}

// ProjectInfo represents a detected subproject
type ProjectInfo struct {
	Name        string   `json:"name"`
	Path        string   `json:"path"` // Relative path from root
	Category    string   `json:"category"`
	Language    string   `json:"language,omitempty"`
	EntryPoints []string `json:"entryPoints,omitempty"`
	DependsOn   []string `json:"dependsOn,omitempty"`  // Names of projects this depends on
	DependedBy  []string `json:"dependedBy,omitempty"` // Names of projects that depend on this
}

// MonorepoConfig allows user customization of monorepo detection
type MonorepoConfig struct {
	// Patterns are additional glob patterns to scan for projects
	Patterns []string `json:"patterns,omitempty"`
	// ExcludePatterns are patterns to exclude from detection
	ExcludePatterns []string `json:"excludePatterns,omitempty"`
	// PreferredManager forces use of a specific manager detection
	PreferredManager string `json:"preferredManager,omitempty"`
	// DisableAutoDetect skips automatic manager detection
	DisableAutoDetect bool `json:"disableAutoDetect,omitempty"`
}

// DefaultDirectoryPatterns are the default directories to scan
var DefaultDirectoryPatterns = []string{
	"apps", "packages", "libs", "modules", "services",
	"projects", "plugins", "extensions", "tools",
	"examples", "samples", "demo", "demos",
	"core", "shared", "common", "internal",
	"cmd", "crates", "workspaces", "components",
	"clients", "servers", "workers", "lambdas",
	"functions", "microservices", "backends", "frontends",
}

// DetectMonorepo performs comprehensive monorepo detection
func DetectMonorepo(root string, config *MonorepoConfig) *MonorepoInfo {
	info := &MonorepoInfo{
		RootPath:     root,
		Projects:     []ProjectInfo{},
		Dependencies: make(map[string][]string),
	}

	// Phase 1: Detect monorepo manager
	manager, configFile, confidence := detectMonorepoManager(root)
	info.Manager = manager
	info.ConfigFile = configFile
	info.Confidence = confidence

	// Phase 2 & 3: Get project list from manager config or directory patterns
	var projects []ProjectInfo
	if manager != MonorepoNone && confidence >= 0.7 {
		// Parse workspace config for explicit project list
		projects = parseMonorepoProjects(root, manager, configFile)
		info.IsMonorepo = len(projects) > 0
	}

	// Fall back to or augment with directory scanning
	if len(projects) == 0 {
		projects = scanDirectoryPatterns(root, config)
		if len(projects) > 0 && info.Manager == MonorepoNone {
			info.IsMonorepo = true
			info.Confidence = 0.5 // Heuristic detection
		}
	}

	info.Projects = projects

	// Phase 4: Detect cross-project dependencies
	if len(projects) > 1 {
		info.Dependencies = detectCrossProjectDependencies(root, projects)
		// Update project dependency info
		for i := range info.Projects {
			proj := &info.Projects[i]
			if deps, ok := info.Dependencies[proj.Name]; ok {
				proj.DependsOn = deps
			}
			// Find reverse dependencies
			for name, deps := range info.Dependencies {
				for _, dep := range deps {
					if dep == proj.Name {
						proj.DependedBy = append(proj.DependedBy, name)
					}
				}
			}
		}
	}

	return info
}

// detectMonorepoManager detects which monorepo tool is being used
func detectMonorepoManager(root string) (MonorepoManager, string, float64) {
	// Check in order of specificity (most explicit first)

	// pnpm workspaces
	if fileExists(filepath.Join(root, "pnpm-workspace.yaml")) {
		return MonorepoPnpm, "pnpm-workspace.yaml", 1.0
	}

	// Lerna
	if fileExists(filepath.Join(root, "lerna.json")) {
		return MonorepoLerna, "lerna.json", 1.0
	}

	// Nx
	if fileExists(filepath.Join(root, "nx.json")) {
		return MonorepoNx, "nx.json", 1.0
	}

	// Rush
	if fileExists(filepath.Join(root, "rush.json")) {
		return MonorepoRush, "rush.json", 1.0
	}

	// Turborepo
	if fileExists(filepath.Join(root, "turbo.json")) {
		return MonorepoTurborepo, "turbo.json", 0.9 // Slightly lower since Turbo uses npm/yarn/pnpm workspaces
	}

	// Melos (Dart/Flutter)
	if fileExists(filepath.Join(root, "melos.yaml")) {
		return MonorepoMelos, "melos.yaml", 1.0
	}

	// Bazel
	if fileExists(filepath.Join(root, "WORKSPACE")) || fileExists(filepath.Join(root, "WORKSPACE.bazel")) ||
		fileExists(filepath.Join(root, "MODULE.bazel")) {
		configFile := "WORKSPACE"
		if fileExists(filepath.Join(root, "MODULE.bazel")) {
			configFile = "MODULE.bazel"
		}
		return MonorepoBazel, configFile, 1.0
	}

	// Pants
	if fileExists(filepath.Join(root, "pants.toml")) || fileExists(filepath.Join(root, "pants.ini")) {
		configFile := "pants.toml"
		if !fileExists(filepath.Join(root, "pants.toml")) {
			configFile = "pants.ini"
		}
		return MonorepoPants, configFile, 1.0
	}

	// Go workspaces
	if fileExists(filepath.Join(root, "go.work")) {
		return MonorepoGoWork, "go.work", 1.0
	}

	// Cargo workspaces (Rust)
	if isCargoWorkspace(root) {
		return MonorepoCargoWs, "Cargo.toml", 1.0
	}

	// Gradle composite builds / multi-project
	if isGradleMultiProject(root) {
		return MonorepoGradleComp, "settings.gradle", 0.9
	}

	// Maven multi-module
	if isMavenMultiModule(root) {
		return MonorepoMavenMulti, "pom.xml", 0.9
	}

	// .NET solution
	if slnFile := findDotnetSolution(root); slnFile != "" {
		return MonorepoDotnetSln, slnFile, 0.8
	}

	// npm/yarn workspaces (check package.json)
	if pkgWorkspaces := getPackageJsonWorkspaces(root); len(pkgWorkspaces) > 0 {
		// Check if yarn.lock exists to distinguish
		if fileExists(filepath.Join(root, "yarn.lock")) {
			return MonorepoYarn, "package.json", 0.9
		}
		if fileExists(filepath.Join(root, "package-lock.json")) {
			return MonorepoNpm, "package.json", 0.9
		}
		// Default to npm if no lock file
		return MonorepoNpm, "package.json", 0.8
	}

	return MonorepoNone, "", 0.0
}

// isCargoWorkspace checks if Cargo.toml defines a workspace
func isCargoWorkspace(root string) bool {
	cargoPath := filepath.Join(root, "Cargo.toml")
	data, err := os.ReadFile(cargoPath)
	if err != nil {
		return false
	}
	// Simple check for [workspace] section
	return strings.Contains(string(data), "[workspace]")
}

// isGradleMultiProject checks for Gradle multi-project builds
func isGradleMultiProject(root string) bool {
	for _, name := range []string{"settings.gradle", "settings.gradle.kts"} {
		path := filepath.Join(root, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(data)
		// Look for include statements
		if strings.Contains(content, "include(") || strings.Contains(content, "include ") ||
			strings.Contains(content, "includeBuild(") {
			return true
		}
	}
	return false
}

// isMavenMultiModule checks for Maven multi-module projects
func isMavenMultiModule(root string) bool {
	pomPath := filepath.Join(root, "pom.xml")
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return false
	}
	// Simple check for <modules> section
	return strings.Contains(string(data), "<modules>")
}

// findDotnetSolution finds a .sln file in root
func findDotnetSolution(root string) string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sln") {
			return entry.Name()
		}
	}
	return ""
}

// getPackageJsonWorkspaces reads workspaces from package.json
func getPackageJsonWorkspaces(root string) []string {
	pkgPath := filepath.Join(root, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil
	}

	var pkg struct {
		Workspaces interface{} `json:"workspaces"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	if pkg.Workspaces == nil {
		return nil
	}

	// Workspaces can be array or object with "packages" field
	switch ws := pkg.Workspaces.(type) {
	case []interface{}:
		var result []string
		for _, w := range ws {
			if s, ok := w.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case map[string]interface{}:
		if packages, ok := ws["packages"].([]interface{}); ok {
			var result []string
			for _, p := range packages {
				if s, ok := p.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}

// parseMonorepoProjects extracts project list from monorepo config files
func parseMonorepoProjects(root string, manager MonorepoManager, configFile string) []ProjectInfo {
	switch manager {
	case MonorepoPnpm:
		return parsePnpmWorkspace(root)
	case MonorepoYarn, MonorepoNpm:
		return parseNpmWorkspace(root)
	case MonorepoLerna:
		return parseLernaConfig(root)
	case MonorepoNx:
		return parseNxConfig(root)
	case MonorepoRush:
		return parseRushConfig(root)
	case MonorepoMelos:
		return parseMelosConfig(root)
	case MonorepoCargoWs:
		return parseCargoWorkspace(root)
	case MonorepoGoWork:
		return parseGoWorkspace(root)
	case MonorepoGradleComp:
		return parseGradleSettings(root)
	case MonorepoMavenMulti:
		return parseMavenModules(root)
	case MonorepoDotnetSln:
		return parseDotnetSolution(root, configFile)
	default:
		return nil
	}
}

// parsePnpmWorkspace parses pnpm-workspace.yaml
func parsePnpmWorkspace(root string) []ProjectInfo {
	path := filepath.Join(root, "pnpm-workspace.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var config struct {
		Packages []string `yaml:"packages"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil
	}

	return expandWorkspaceGlobs(root, config.Packages)
}

// parseNpmWorkspace parses workspaces from package.json
func parseNpmWorkspace(root string) []ProjectInfo {
	workspaces := getPackageJsonWorkspaces(root)
	return expandWorkspaceGlobs(root, workspaces)
}

// parseLernaConfig parses lerna.json
func parseLernaConfig(root string) []ProjectInfo {
	path := filepath.Join(root, "lerna.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var config struct {
		Packages []string `json:"packages"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil
	}

	if len(config.Packages) == 0 {
		// Default Lerna pattern
		config.Packages = []string{"packages/*"}
	}

	return expandWorkspaceGlobs(root, config.Packages)
}

// parseNxConfig parses nx.json and finds project.json files
func parseNxConfig(root string) []ProjectInfo {
	var projects []ProjectInfo

	// Nx can have projects defined in project.json files or in nx.json
	// First, look for workspace.json or project.json files
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			// Skip common non-project directories
			if name == "node_modules" || name == ".git" || name == "dist" || name == ".nx" {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == "project.json" {
			relDir := filepath.Dir(path)
			relPath, _ := filepath.Rel(root, relDir)
			if relPath == "." {
				return nil // Skip root project.json
			}

			// Read project.json for name
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			var proj struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(data, &proj); err != nil || proj.Name == "" {
				proj.Name = filepath.Base(relDir)
			}

			projects = append(projects, ProjectInfo{
				Name:     proj.Name,
				Path:     filepath.ToSlash(relPath),
				Category: categorizeFromPath(relPath),
				Language: detectProjectLanguage(relDir),
			})
		}
		return nil
	})

	if err != nil || len(projects) == 0 {
		// Fall back to scanning default directories
		return expandWorkspaceGlobs(root, []string{"apps/*", "libs/*", "packages/*"})
	}

	return projects
}

// parseRushConfig parses rush.json
func parseRushConfig(root string) []ProjectInfo {
	path := filepath.Join(root, "rush.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var config struct {
		Projects []struct {
			PackageName    string `json:"packageName"`
			ProjectFolder  string `json:"projectFolder"`
			ReviewCategory string `json:"reviewCategory"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil
	}

	var projects []ProjectInfo
	for _, p := range config.Projects {
		category := p.ReviewCategory
		if category == "" {
			category = categorizeFromPath(p.ProjectFolder)
		}
		projects = append(projects, ProjectInfo{
			Name:     p.PackageName,
			Path:     filepath.ToSlash(p.ProjectFolder),
			Category: category,
			Language: detectProjectLanguage(filepath.Join(root, p.ProjectFolder)),
		})
	}
	return projects
}

// parseMelosConfig parses melos.yaml
func parseMelosConfig(root string) []ProjectInfo {
	path := filepath.Join(root, "melos.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var config struct {
		Packages []string `yaml:"packages"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil
	}

	if len(config.Packages) == 0 {
		// Default Melos pattern
		config.Packages = []string{"packages/**"}
	}

	projects := expandWorkspaceGlobs(root, config.Packages)
	// Set language to Dart for all Melos projects
	for i := range projects {
		projects[i].Language = "dart"
	}
	return projects
}

// parseCargoWorkspace parses Cargo.toml workspace members
func parseCargoWorkspace(root string) []ProjectInfo {
	path := filepath.Join(root, "Cargo.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	content := string(data)

	// Parse workspace.members using regex (TOML parsing would be more robust)
	membersRe := regexp.MustCompile(`members\s*=\s*\[([^\]]+)\]`)
	matches := membersRe.FindStringSubmatch(content)
	if len(matches) < 2 {
		return nil
	}

	// Extract quoted strings
	stringRe := regexp.MustCompile(`"([^"]+)"`)
	members := stringRe.FindAllStringSubmatch(matches[1], -1)

	var patterns []string
	for _, m := range members {
		if len(m) > 1 {
			patterns = append(patterns, m[1])
		}
	}

	projects := expandWorkspaceGlobs(root, patterns)
	for i := range projects {
		projects[i].Language = "rust"
	}
	return projects
}

// parseGoWorkspace parses go.work file
func parseGoWorkspace(root string) []ProjectInfo {
	path := filepath.Join(root, "go.work")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var projects []ProjectInfo
	lines := strings.Split(string(data), "\n")
	inUse := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "use (") || line == "use (" {
			inUse = true
			continue
		}
		if inUse {
			if line == ")" {
				inUse = false
				continue
			}
			// Extract module path
			modPath := strings.Trim(line, " \t\"")
			if modPath != "" && !strings.HasPrefix(modPath, "//") {
				projects = append(projects, ProjectInfo{
					Name:     filepath.Base(modPath),
					Path:     filepath.ToSlash(modPath),
					Category: categorizeFromPath(modPath),
					Language: "go",
				})
			}
		} else if strings.HasPrefix(line, "use ") {
			// Single-line use directive
			modPath := strings.TrimPrefix(line, "use ")
			modPath = strings.Trim(modPath, " \t\"")
			if modPath != "" {
				projects = append(projects, ProjectInfo{
					Name:     filepath.Base(modPath),
					Path:     filepath.ToSlash(modPath),
					Category: categorizeFromPath(modPath),
					Language: "go",
				})
			}
		}
	}
	return projects
}

// parseGradleSettings parses settings.gradle(.kts)
func parseGradleSettings(root string) []ProjectInfo {
	var data []byte
	var err error

	for _, name := range []string{"settings.gradle.kts", "settings.gradle"} {
		data, err = os.ReadFile(filepath.Join(root, name))
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil
	}

	content := string(data)
	var projects []ProjectInfo

	// Match include("project") or include("project1", "project2") or include 'project'
	includeRe := regexp.MustCompile(`include\s*\(?["']([^"']+)["']`)
	matches := includeRe.FindAllStringSubmatch(content, -1)

	for _, m := range matches {
		if len(m) > 1 {
			// Gradle uses : as path separator
			projPath := strings.ReplaceAll(m[1], ":", "/")
			projPath = strings.TrimPrefix(projPath, "/")
			projects = append(projects, ProjectInfo{
				Name:     filepath.Base(projPath),
				Path:     projPath,
				Category: categorizeFromPath(projPath),
				Language: "java", // Could also be kotlin
			})
		}
	}
	return projects
}

// parseMavenModules parses Maven multi-module pom.xml
func parseMavenModules(root string) []ProjectInfo {
	path := filepath.Join(root, "pom.xml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	content := string(data)
	var projects []ProjectInfo

	// Simple regex to find <module>name</module>
	moduleRe := regexp.MustCompile(`<module>([^<]+)</module>`)
	matches := moduleRe.FindAllStringSubmatch(content, -1)

	for _, m := range matches {
		if len(m) > 1 {
			projects = append(projects, ProjectInfo{
				Name:     filepath.Base(m[1]),
				Path:     filepath.ToSlash(m[1]),
				Category: categorizeFromPath(m[1]),
				Language: "java",
			})
		}
	}
	return projects
}

// parseDotnetSolution parses a .sln file for project references
func parseDotnetSolution(root, slnFile string) []ProjectInfo {
	path := filepath.Join(root, slnFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	content := string(data)
	var projects []ProjectInfo

	// Match Project lines: Project("{GUID}") = "Name", "Path\To\Project.csproj", "{GUID}"
	projectRe := regexp.MustCompile(`Project\("[^"]+"\)\s*=\s*"([^"]+)",\s*"([^"]+)"`)
	matches := projectRe.FindAllStringSubmatch(content, -1)

	for _, m := range matches {
		if len(m) > 2 {
			name := m[1]
			projPath := m[2]
			// Convert backslashes and get directory
			projPath = strings.ReplaceAll(projPath, "\\", "/")
			projDir := filepath.Dir(projPath)
			if projDir == "." {
				projDir = name
			}
			projects = append(projects, ProjectInfo{
				Name:     name,
				Path:     projDir,
				Category: categorizeFromPath(projDir),
				Language: "csharp",
			})
		}
	}
	return projects
}

// expandWorkspaceGlobs expands glob patterns to actual project directories
func expandWorkspaceGlobs(root string, patterns []string) []ProjectInfo {
	var projects []ProjectInfo
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		// Normalize pattern - remove trailing slashes but keep glob patterns
		pattern = strings.TrimSuffix(pattern, "/")

		// Convert directory-level patterns to glob patterns
		// e.g., "packages/**" -> "packages/*" for immediate children
		if strings.HasSuffix(pattern, "/**") {
			pattern = strings.TrimSuffix(pattern, "/**") + "/*"
		}

		// Handle glob patterns
		if strings.Contains(pattern, "*") {
			matches, err := filepath.Glob(filepath.Join(root, pattern))
			if err != nil {
				continue
			}
			for _, match := range matches {
				info, err := os.Stat(match)
				if err != nil || !info.IsDir() {
					continue
				}
				relPath, _ := filepath.Rel(root, match)
				relPath = filepath.ToSlash(relPath)
				if seen[relPath] {
					continue
				}
				if !isValidProject(match) {
					continue
				}
				seen[relPath] = true
				projects = append(projects, ProjectInfo{
					Name:     filepath.Base(match),
					Path:     relPath,
					Category: categorizeFromPath(relPath),
					Language: detectProjectLanguage(match),
				})
			}
		} else {
			// Direct path - check if it's a directory containing projects or a project itself
			fullPath := filepath.Join(root, pattern)
			info, err := os.Stat(fullPath)
			if err != nil || !info.IsDir() {
				continue
			}

			// Check if it's a valid project itself
			if isValidProject(fullPath) {
				relPath := filepath.ToSlash(pattern)
				if !seen[relPath] {
					seen[relPath] = true
					projects = append(projects, ProjectInfo{
						Name:     filepath.Base(pattern),
						Path:     relPath,
						Category: categorizeFromPath(relPath),
						Language: detectProjectLanguage(fullPath),
					})
				}
			} else {
				// Not a project itself, scan for immediate subdirectories that are projects
				entries, err := os.ReadDir(fullPath)
				if err != nil {
					continue
				}
				for _, entry := range entries {
					if !entry.IsDir() {
						continue
					}
					subPath := filepath.Join(fullPath, entry.Name())
					relPath := filepath.ToSlash(filepath.Join(pattern, entry.Name()))
					if seen[relPath] {
						continue
					}
					if !isValidProject(subPath) {
						continue
					}
					seen[relPath] = true
					projects = append(projects, ProjectInfo{
						Name:     entry.Name(),
						Path:     relPath,
						Category: categorizeFromPath(relPath),
						Language: detectProjectLanguage(subPath),
					})
				}
			}
		}
	}
	return projects
}

// scanDirectoryPatterns scans default and custom directory patterns for projects
func scanDirectoryPatterns(root string, config *MonorepoConfig) []ProjectInfo {
	var projects []ProjectInfo
	seen := make(map[string]bool)

	patterns := DefaultDirectoryPatterns
	if config != nil && len(config.Patterns) > 0 {
		patterns = append(patterns, config.Patterns...)
	}

	excludeSet := make(map[string]bool)
	if config != nil {
		for _, p := range config.ExcludePatterns {
			excludeSet[p] = true
		}
	}

	for _, pattern := range patterns {
		patternDir := filepath.Join(root, pattern)
		entries, err := os.ReadDir(patternDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			subDir := filepath.Join(patternDir, entry.Name())
			relPath := filepath.ToSlash(filepath.Join(pattern, entry.Name()))

			if seen[relPath] || excludeSet[relPath] || excludeSet[entry.Name()] {
				continue
			}

			if !isValidProject(subDir) {
				continue
			}

			seen[relPath] = true
			projects = append(projects, ProjectInfo{
				Name:     normalizeProjectName(entry.Name()),
				Path:     relPath,
				Category: categorizeFromPath(pattern),
				Language: detectProjectLanguage(subDir),
			})
		}
	}
	return projects
}

// isValidProject checks if a directory is a valid project (has manifest)
func isValidProject(dir string) bool {
	manifests := []string{
		"package.json", "pubspec.yaml", "Cargo.toml", "go.mod",
		"pom.xml", "build.gradle", "build.gradle.kts",
		"pyproject.toml", "setup.py", "Gemfile", "composer.json",
		"Package.swift", "CMakeLists.txt", "project.json",
	}

	for _, manifest := range manifests {
		if fileExists(filepath.Join(dir, manifest)) {
			return true
		}
	}

	// Check for .csproj files
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".csproj") {
				return true
			}
		}
	}

	return false
}

// categorizeFromPath determines category from path components
func categorizeFromPath(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	if len(parts) == 0 {
		return "unknown"
	}

	firstDir := strings.ToLower(parts[0])
	switch firstDir {
	case "apps", "applications":
		return "application"
	case "packages", "libs", "libraries":
		return "library"
	case "services", "microservices", "backends":
		return "service"
	case "modules":
		return "module"
	case "plugins", "extensions":
		return "plugin"
	case "tools", "scripts":
		return "tool"
	case "examples", "samples", "demo", "demos":
		return "example"
	case "core", "shared", "common":
		return "shared"
	case "clients", "frontends":
		return "client"
	case "servers", "workers":
		return "server"
	case "crates":
		return "crate"
	case "cmd":
		return "command"
	default:
		return "project"
	}
}

// detectProjectLanguage detects the primary language of a project
func detectProjectLanguage(dir string) string {
	// Check in order of precedence
	checks := []struct {
		file string
		lang string
	}{
		{"go.mod", "go"},
		{"Cargo.toml", "rust"},
		{"pubspec.yaml", "dart"},
		{"package.json", "javascript"},
		{"pyproject.toml", "python"},
		{"setup.py", "python"},
		{"Gemfile", "ruby"},
		{"pom.xml", "java"},
		{"build.gradle", "java"},
		{"build.gradle.kts", "kotlin"},
		{"composer.json", "php"},
		{"Package.swift", "swift"},
		{"CMakeLists.txt", "cpp"},
	}

	for _, check := range checks {
		if fileExists(filepath.Join(dir, check.file)) {
			return check.lang
		}
	}

	// Check for .csproj
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".csproj") {
				return "csharp"
			}
		}
	}

	return ""
}

// normalizeProjectName converts directory name to a normalized project name
func normalizeProjectName(name string) string {
	// Replace underscores with hyphens
	name = strings.ReplaceAll(name, "_", "-")
	// Convert camelCase to kebab-case
	var result strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('-')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// detectCrossProjectDependencies analyzes inter-project dependencies
func detectCrossProjectDependencies(root string, projects []ProjectInfo) map[string][]string {
	deps := make(map[string][]string)

	// Build a map of project paths to names for quick lookup
	pathToName := make(map[string]string)
	for _, p := range projects {
		pathToName[p.Path] = p.Name
		// Also map by package name patterns
		pathToName[p.Name] = p.Name
	}

	for _, proj := range projects {
		projDeps := []string{}
		projPath := filepath.Join(root, proj.Path)

		switch proj.Language {
		case "javascript", "":
			// Check package.json for workspace dependencies
			pkgDeps := parsePackageJsonDeps(projPath, pathToName)
			projDeps = append(projDeps, pkgDeps...)
		case "dart":
			// Check pubspec.yaml for path dependencies
			dartDeps := parsePubspecDeps(projPath, root, pathToName)
			projDeps = append(projDeps, dartDeps...)
		case "go":
			// Check go.mod for replace directives
			goDeps := parseGoModDeps(projPath, pathToName)
			projDeps = append(projDeps, goDeps...)
		case "rust":
			// Check Cargo.toml for path dependencies
			rustDeps := parseCargoDeps(projPath, pathToName)
			projDeps = append(projDeps, rustDeps...)
		case "python":
			// Check pyproject.toml for local dependencies
			pyDeps := parsePyProjectDeps(projPath, root, pathToName)
			projDeps = append(projDeps, pyDeps...)
		}

		if len(projDeps) > 0 {
			deps[proj.Name] = uniqueStrings(projDeps)
		}
	}

	return deps
}

// parsePackageJsonDeps finds local/workspace dependencies
func parsePackageJsonDeps(projPath string, pathToName map[string]string) []string {
	pkgPath := filepath.Join(projPath, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	var deps []string
	checkDep := func(name, version string) {
		// Check for workspace: protocol or local paths
		if strings.HasPrefix(version, "workspace:") ||
			strings.HasPrefix(version, "file:") ||
			strings.HasPrefix(version, "link:") ||
			strings.HasPrefix(version, "../") ||
			strings.HasPrefix(version, "./") {
			// Try to match to a known project
			if projName, ok := pathToName[name]; ok {
				deps = append(deps, projName)
			} else {
				// Use package name directly if it might be a workspace package
				deps = append(deps, name)
			}
		}
	}

	for name, version := range pkg.Dependencies {
		checkDep(name, version)
	}
	for name, version := range pkg.DevDependencies {
		checkDep(name, version)
	}

	return deps
}

// parsePubspecDeps finds path dependencies in pubspec.yaml
func parsePubspecDeps(projPath, root string, pathToName map[string]string) []string {
	pubspecPath := filepath.Join(projPath, "pubspec.yaml")
	data, err := os.ReadFile(pubspecPath)
	if err != nil {
		return nil
	}

	var pubspec struct {
		Dependencies    map[string]interface{} `yaml:"dependencies"`
		DevDependencies map[string]interface{} `yaml:"dev_dependencies"`
	}
	if err := yaml.Unmarshal(data, &pubspec); err != nil {
		return nil
	}

	var deps []string
	checkDep := func(name string, config interface{}) {
		if configMap, ok := config.(map[string]interface{}); ok {
			if pathVal, hasPath := configMap["path"]; hasPath {
				if pathStr, ok := pathVal.(string); ok {
					// Resolve path to find project
					absPath := filepath.Join(projPath, pathStr)
					relPath, _ := filepath.Rel(root, absPath)
					relPath = filepath.ToSlash(relPath)
					if projName, ok := pathToName[relPath]; ok {
						deps = append(deps, projName)
					} else if projName, ok := pathToName[name]; ok {
						deps = append(deps, projName)
					}
				}
			}
		}
	}

	for name, config := range pubspec.Dependencies {
		checkDep(name, config)
	}
	for name, config := range pubspec.DevDependencies {
		checkDep(name, config)
	}

	return deps
}

// parseGoModDeps finds replace directives pointing to local paths
func parseGoModDeps(projPath string, pathToName map[string]string) []string {
	goModPath := filepath.Join(projPath, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return nil
	}

	var deps []string
	lines := strings.Split(string(data), "\n")
	inReplace := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "replace (") || line == "replace (" {
			inReplace = true
			continue
		}
		if inReplace && line == ")" {
			inReplace = false
			continue
		}

		if inReplace || strings.HasPrefix(line, "replace ") {
			// Parse: module => ./local/path or module => ../path
			if strings.Contains(line, "=>") {
				parts := strings.Split(line, "=>")
				if len(parts) == 2 {
					replacement := strings.TrimSpace(parts[1])
					if strings.HasPrefix(replacement, "./") || strings.HasPrefix(replacement, "../") {
						// Local replacement
						baseName := filepath.Base(replacement)
						if projName, ok := pathToName[baseName]; ok {
							deps = append(deps, projName)
						}
					}
				}
			}
		}
	}

	return deps
}

// parseCargoDeps finds path dependencies in Cargo.toml
func parseCargoDeps(projPath string, pathToName map[string]string) []string {
	cargoPath := filepath.Join(projPath, "Cargo.toml")
	data, err := os.ReadFile(cargoPath)
	if err != nil {
		return nil
	}

	content := string(data)
	var deps []string

	// Match: name = { path = "..." } or name.path = "..."
	pathRe := regexp.MustCompile(`(\w+)\s*=\s*\{[^}]*path\s*=\s*"([^"]+)"`)
	matches := pathRe.FindAllStringSubmatch(content, -1)

	for _, m := range matches {
		if len(m) > 2 {
			name := m[1]
			if projName, ok := pathToName[name]; ok {
				deps = append(deps, projName)
			}
		}
	}

	return deps
}

// parsePyProjectDeps finds local dependencies in pyproject.toml
func parsePyProjectDeps(projPath, _ string, pathToName map[string]string) []string {
	pyprojectPath := filepath.Join(projPath, "pyproject.toml")
	data, err := os.ReadFile(pyprojectPath)
	if err != nil {
		return nil
	}

	content := string(data)
	var deps []string

	// Match: package = { path = "..." } or package @ file:///path
	pathRe := regexp.MustCompile(`(\w[\w-]*)\s*=\s*\{[^}]*path\s*=\s*"([^"]+)"`)
	matches := pathRe.FindAllStringSubmatch(content, -1)

	for _, m := range matches {
		if len(m) > 2 {
			name := m[1]
			if projName, ok := pathToName[name]; ok {
				deps = append(deps, projName)
			}
		}
	}

	return deps
}

// uniqueStrings returns unique strings from a slice
func uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
