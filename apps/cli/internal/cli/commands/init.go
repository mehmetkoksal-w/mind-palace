package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/project"
)

func init() {
	Register(&Command{
		Name:        "build",
		Aliases:     []string{"init", "enter"}, // Keep init and enter as aliases for backwards compatibility
		Description: "Build the palace (initialize in current directory)",
		Run:         RunInit,
	})
}

// InitOptions contains the configuration for the init command.
type InitOptions struct {
	Root        string
	Force       bool
	WithOutputs bool
	Detect      bool
}

// RunInit initializes a new Mind Palace in the specified directory.
func RunInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	force := fs.Bool("force", false, "overwrite existing curated files")
	withOutputs := fs.Bool("with-outputs", false, "also create generated outputs (context-pack)")
	detect := fs.Bool("detect", false, "auto-detect project type and generate profile")
	if err := fs.Parse(args); err != nil {
		return err
	}

	opts := InitOptions{
		Root:        *root,
		Force:       *force,
		WithOutputs: *withOutputs,
		Detect:      *detect,
	}

	return ExecuteInit(opts)
}

// ExecuteInit performs the initialization with the given options.
// This is separated for easier testing.
func ExecuteInit(opts InitOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}
	if _, err := config.EnsureLayout(rootPath); err != nil {
		return err
	}
	if err := config.CopySchemas(rootPath, opts.Force); err != nil {
		return err
	}

	// Auto-detect project type if requested
	language := "unknown"
	var monorepoRooms []MonorepoRoom
	if opts.Detect {
		profile := project.BuildProfile(rootPath)
		profilePath := filepath.Join(rootPath, ".palace", "project-profile.json")
		if err := config.WriteJSON(profilePath, profile); err != nil {
			return err
		}
		if len(profile.Languages) > 0 {
			language = profile.Languages[0]
		}
		fmt.Printf("detected project type: %s\n", language)

		// Detect monorepo structure and auto-generate rooms
		monorepoRooms = detectMonorepoRooms(rootPath)
		if len(monorepoRooms) > 0 {
			fmt.Printf("detected monorepo structure with %d subprojects\n", len(monorepoRooms))
		}
	}

	replacements := map[string]string{
		"projectName": filepath.Base(rootPath),
		"language":    language,
	}
	if err := config.WriteTemplate(filepath.Join(rootPath, ".palace", "palace.jsonc"), "palace.jsonc", replacements, opts.Force); err != nil {
		return err
	}
	if err := config.WriteTemplate(filepath.Join(rootPath, ".palace", "rooms", "project-overview.jsonc"), "rooms/project-overview.jsonc", map[string]string{}, opts.Force); err != nil {
		return err
	}

	// Write auto-detected monorepo rooms
	for _, room := range monorepoRooms {
		roomPath := filepath.Join(rootPath, ".palace", "rooms", room.Name+".jsonc")
		if _, err := os.Stat(roomPath); err == nil && !opts.Force {
			continue // Don't overwrite existing rooms
		}
		if err := writeMonorepoRoom(roomPath, room); err != nil {
			fmt.Printf("warning: could not create room %s: %v\n", room.Name, err)
		} else {
			fmt.Printf("  created room: %s (%s)\n", room.Name, room.RelPath)
		}
	}

	if err := config.WriteTemplate(filepath.Join(rootPath, ".palace", "playbooks", "default.jsonc"), "playbooks/default.jsonc", map[string]string{}, opts.Force); err != nil {
		return err
	}
	// Only write default project-profile template if NOT detecting (detect writes its own profile)
	if !opts.Detect {
		if err := config.WriteTemplate(filepath.Join(rootPath, ".palace", "project-profile.json"), "project-profile.json", map[string]string{}, opts.Force); err != nil {
			return err
		}
	}

	if opts.WithOutputs {
		cpPath := filepath.Join(rootPath, ".palace", "outputs", "context-pack.json")
		if _, err := os.Stat(cpPath); os.IsNotExist(err) || opts.Force {
			cpReplacements := map[string]string{
				"goal":      "unspecified",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			if err := config.WriteTemplate(cpPath, "outputs/context-pack.json", cpReplacements, opts.Force); err != nil {
				return err
			}
		}
	}

	fmt.Printf("initialized palace in %s\n", filepath.Join(rootPath, ".palace"))
	return nil
}

// MonorepoRoom represents a detected subproject in a monorepo
type MonorepoRoom struct {
	Name        string   // Room name (e.g., "driver-app")
	RelPath     string   // Relative path (e.g., "apps/driver_app")
	Category    string   // Category (e.g., "apps", "packages")
	EntryPoints []string // Detected entry points
}

// detectMonorepoRooms looks for common monorepo patterns and returns rooms for each subproject
func detectMonorepoRooms(root string) []MonorepoRoom {
	var rooms []MonorepoRoom

	// Common monorepo directory patterns
	patterns := []struct {
		dir      string
		category string
	}{
		{"apps", "application"},
		{"packages", "package"},
		{"libs", "library"},
		{"modules", "module"},
		{"services", "service"},
	}

	for _, pattern := range patterns {
		patternDir := filepath.Join(root, pattern.dir)
		entries, err := os.ReadDir(patternDir)
		if err != nil {
			continue // Directory doesn't exist
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			subDir := filepath.Join(patternDir, entry.Name())
			entryPoints := detectEntryPoints(subDir, entry.Name())
			if len(entryPoints) == 0 {
				continue // Not a valid subproject
			}

			// Normalize room name (snake_case to kebab-case, etc.)
			roomName := normalizeRoomName(entry.Name())

			rooms = append(rooms, MonorepoRoom{
				Name:        roomName,
				RelPath:     filepath.Join(pattern.dir, entry.Name()),
				Category:    pattern.category,
				EntryPoints: entryPoints,
			})
		}
	}

	return rooms
}

// detectEntryPoints finds common entry point files in a subproject directory
func detectEntryPoints(dir, name string) []string {
	var entryPoints []string

	// Check for common project manifest files (indicates valid subproject)
	manifests := []string{
		"pubspec.yaml",       // Dart/Flutter
		"package.json",       // Node.js
		"Cargo.toml",         // Rust
		"go.mod",             // Go
		"pom.xml",            // Maven (Java)
		"build.gradle",       // Gradle (Java/Kotlin)
		"build.gradle.kts",   // Gradle Kotlin DSL
		"pyproject.toml",     // Python
		"setup.py",           // Python legacy
		"Gemfile",            // Ruby
		"composer.json",      // PHP
		"Package.swift",      // Swift
		"CMakeLists.txt",     // C/C++
		"*.csproj",           // .NET
	}

	hasManifest := false
	for _, manifest := range manifests {
		if strings.Contains(manifest, "*") {
			// Handle glob patterns like *.csproj
			matches, _ := filepath.Glob(filepath.Join(dir, manifest))
			if len(matches) > 0 {
				hasManifest = true
				break
			}
		} else if _, err := os.Stat(filepath.Join(dir, manifest)); err == nil {
			hasManifest = true
			break
		}
	}

	if !hasManifest {
		return nil // Not a valid subproject
	}

	// Add relative paths for common entry points
	commonEntries := []string{
		"lib/main.dart",             // Flutter
		"lib/" + name + ".dart",     // Dart package
		"src/index.ts",              // TypeScript
		"src/index.tsx",             // React TypeScript
		"src/index.js",              // JavaScript
		"src/main.ts",               // Angular/other
		"src/App.tsx",               // React
		"src/App.vue",               // Vue
		"index.ts",                  // Root index
		"index.js",                  // Root index JS
		"main.go",                   // Go
		"cmd/main.go",               // Go cmd pattern
		"src/main.rs",               // Rust
		"src/lib.rs",                // Rust library
		"__init__.py",               // Python
		"app.py",                    // Python Flask
		"main.py",                   // Python
	}

	for _, entry := range commonEntries {
		if _, err := os.Stat(filepath.Join(dir, entry)); err == nil {
			entryPoints = append(entryPoints, entry)
		}
	}

	// If no specific entry points found, add README if exists
	if len(entryPoints) == 0 {
		if _, err := os.Stat(filepath.Join(dir, "README.md")); err == nil {
			entryPoints = append(entryPoints, "README.md")
		}
	}

	// If still no entry points but has manifest, add the manifest
	if len(entryPoints) == 0 {
		for _, manifest := range []string{"pubspec.yaml", "package.json", "Cargo.toml", "go.mod"} {
			if _, err := os.Stat(filepath.Join(dir, manifest)); err == nil {
				entryPoints = append(entryPoints, manifest)
				break
			}
		}
	}

	return entryPoints
}

// normalizeRoomName converts directory names to kebab-case room names
func normalizeRoomName(name string) string {
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

// RoomConfig represents the JSON structure for a room file
type RoomConfig struct {
	SchemaVersion string     `json:"schemaVersion"`
	Kind          string     `json:"kind"`
	Name          string     `json:"name"`
	Summary       string     `json:"summary"`
	EntryPoints   []string   `json:"entryPoints"`
	Capabilities  []string   `json:"capabilities"`
	Provenance    Provenance `json:"provenance"`
}

// Provenance tracks who/when created a config
type Provenance struct {
	CreatedBy string `json:"createdBy"`
	CreatedAt string `json:"createdAt"`
}

// writeMonorepoRoom creates a room configuration file for a detected monorepo subproject
func writeMonorepoRoom(path string, room MonorepoRoom) error {
	// Prefix entry points with the relative path
	var entryPoints []string
	for _, ep := range room.EntryPoints {
		entryPoints = append(entryPoints, filepath.Join(room.RelPath, ep))
	}

	roomConfig := RoomConfig{
		SchemaVersion: "1.0.0",
		Kind:          "palace/room",
		Name:          room.Name,
		Summary:       fmt.Sprintf("Auto-detected %s at %s", room.Category, room.RelPath),
		EntryPoints:   entryPoints,
		Capabilities:  []string{"read.file", "search.text"},
		Provenance: Provenance{
			CreatedBy: "palace init --detect",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		},
	}

	data, err := json.MarshalIndent(roomConfig, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}
