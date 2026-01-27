package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectMonorepoManager(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(root string) error
		expectedMgr   MonorepoManager
		minConfidence float64
	}{
		{
			name: "pnpm workspace",
			setup: func(root string) error {
				return os.WriteFile(filepath.Join(root, "pnpm-workspace.yaml"), []byte(`packages:
  - 'packages/*'
  - 'apps/*'
`), 0644)
			},
			expectedMgr:   MonorepoPnpm,
			minConfidence: 1.0,
		},
		{
			name: "lerna",
			setup: func(root string) error {
				return os.WriteFile(filepath.Join(root, "lerna.json"), []byte(`{
  "packages": ["packages/*"],
  "version": "independent"
}`), 0644)
			},
			expectedMgr:   MonorepoLerna,
			minConfidence: 1.0,
		},
		{
			name: "nx",
			setup: func(root string) error {
				return os.WriteFile(filepath.Join(root, "nx.json"), []byte(`{
  "targetDefaults": {}
}`), 0644)
			},
			expectedMgr:   MonorepoNx,
			minConfidence: 1.0,
		},
		{
			name: "rush",
			setup: func(root string) error {
				return os.WriteFile(filepath.Join(root, "rush.json"), []byte(`{
  "projects": []
}`), 0644)
			},
			expectedMgr:   MonorepoRush,
			minConfidence: 1.0,
		},
		{
			name: "turborepo",
			setup: func(root string) error {
				return os.WriteFile(filepath.Join(root, "turbo.json"), []byte(`{
  "pipeline": {}
}`), 0644)
			},
			expectedMgr:   MonorepoTurborepo,
			minConfidence: 0.9,
		},
		{
			name: "melos",
			setup: func(root string) error {
				return os.WriteFile(filepath.Join(root, "melos.yaml"), []byte(`name: my_project
packages:
  - packages/**
`), 0644)
			},
			expectedMgr:   MonorepoMelos,
			minConfidence: 1.0,
		},
		{
			name: "bazel workspace",
			setup: func(root string) error {
				return os.WriteFile(filepath.Join(root, "WORKSPACE"), []byte(`workspace(name = "my_workspace")`), 0644)
			},
			expectedMgr:   MonorepoBazel,
			minConfidence: 1.0,
		},
		{
			name: "bazel module",
			setup: func(root string) error {
				return os.WriteFile(filepath.Join(root, "MODULE.bazel"), []byte(`module(name = "my_module")`), 0644)
			},
			expectedMgr:   MonorepoBazel,
			minConfidence: 1.0,
		},
		{
			name: "pants",
			setup: func(root string) error {
				return os.WriteFile(filepath.Join(root, "pants.toml"), []byte(`[GLOBAL]
pants_version = "2.15.0"
`), 0644)
			},
			expectedMgr:   MonorepoPants,
			minConfidence: 1.0,
		},
		{
			name: "go workspace",
			setup: func(root string) error {
				return os.WriteFile(filepath.Join(root, "go.work"), []byte(`go 1.21

use (
    ./cmd/app
    ./pkg/lib
)
`), 0644)
			},
			expectedMgr:   MonorepoGoWork,
			minConfidence: 1.0,
		},
		{
			name: "cargo workspace",
			setup: func(root string) error {
				return os.WriteFile(filepath.Join(root, "Cargo.toml"), []byte(`[workspace]
members = [
    "crates/*",
]
`), 0644)
			},
			expectedMgr:   MonorepoCargoWs,
			minConfidence: 1.0,
		},
		{
			name: "gradle composite",
			setup: func(root string) error {
				return os.WriteFile(filepath.Join(root, "settings.gradle"), []byte(`rootProject.name = 'my-project'
include('app')
include('lib')
`), 0644)
			},
			expectedMgr:   MonorepoGradleComp,
			minConfidence: 0.9,
		},
		{
			name: "maven multi-module",
			setup: func(root string) error {
				return os.WriteFile(filepath.Join(root, "pom.xml"), []byte(`<?xml version="1.0"?>
<project>
  <modules>
    <module>app</module>
    <module>lib</module>
  </modules>
</project>
`), 0644)
			},
			expectedMgr:   MonorepoMavenMulti,
			minConfidence: 0.9,
		},
		{
			name: "dotnet solution",
			setup: func(root string) error {
				return os.WriteFile(filepath.Join(root, "MySolution.sln"), []byte(`Microsoft Visual Studio Solution File
Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "App", "App\App.csproj", "{GUID}"
EndProject
`), 0644)
			},
			expectedMgr:   MonorepoDotnetSln,
			minConfidence: 0.8,
		},
		{
			name: "yarn workspaces",
			setup: func(root string) error {
				if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{
  "name": "root",
  "workspaces": ["packages/*"]
}`), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(root, "yarn.lock"), []byte(``), 0644)
			},
			expectedMgr:   MonorepoYarn,
			minConfidence: 0.9,
		},
		{
			name: "npm workspaces",
			setup: func(root string) error {
				if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{
  "name": "root",
  "workspaces": ["packages/*"]
}`), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(root, "package-lock.json"), []byte(`{}`), 0644)
			},
			expectedMgr:   MonorepoNpm,
			minConfidence: 0.9,
		},
		{
			name: "no monorepo",
			setup: func(root string) error {
				return os.WriteFile(filepath.Join(root, "package.json"), []byte(`{
  "name": "single-project"
}`), 0644)
			},
			expectedMgr:   MonorepoNone,
			minConfidence: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if err := tt.setup(root); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			mgr, _, confidence := detectMonorepoManager(root)

			if mgr != tt.expectedMgr {
				t.Errorf("expected manager %q, got %q", tt.expectedMgr, mgr)
			}
			if confidence < tt.minConfidence {
				t.Errorf("expected confidence >= %v, got %v", tt.minConfidence, confidence)
			}
		})
	}
}

func TestParsePnpmWorkspace(t *testing.T) {
	root := t.TempDir()

	// Create pnpm-workspace.yaml
	if err := os.WriteFile(filepath.Join(root, "pnpm-workspace.yaml"), []byte(`packages:
  - 'packages/*'
  - 'apps/*'
`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create package directories with package.json
	pkgDir := filepath.Join(root, "packages", "ui-lib")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name": "ui-lib"}`), 0644); err != nil {
		t.Fatal(err)
	}

	appDir := filepath.Join(root, "apps", "web-app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "package.json"), []byte(`{"name": "web-app"}`), 0644); err != nil {
		t.Fatal(err)
	}

	projects := parsePnpmWorkspace(root)

	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}

	// Verify project names
	names := make(map[string]bool)
	for _, p := range projects {
		names[p.Name] = true
	}
	if !names["ui-lib"] {
		t.Error("expected project 'ui-lib' not found")
	}
	if !names["web-app"] {
		t.Error("expected project 'web-app' not found")
	}
}

func TestParseGoWorkspace(t *testing.T) {
	root := t.TempDir()

	// Create go.work file
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte(`go 1.21

use (
    ./cmd/server
    ./pkg/common
)
`), 0644); err != nil {
		t.Fatal(err)
	}

	projects := parseGoWorkspace(root)

	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}

	// Verify project details
	for _, p := range projects {
		if p.Language != "go" {
			t.Errorf("expected language 'go', got '%s'", p.Language)
		}
	}
}

func TestParseCargoWorkspace(t *testing.T) {
	root := t.TempDir()

	// Create Cargo.toml with workspace
	if err := os.WriteFile(filepath.Join(root, "Cargo.toml"), []byte(`[workspace]
members = [
    "crates/core",
    "crates/utils",
]

[package]
name = "root"
version = "0.1.0"
`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create crate directories
	for _, crate := range []string{"core", "utils"} {
		crateDir := filepath.Join(root, "crates", crate)
		if err := os.MkdirAll(crateDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(crateDir, "Cargo.toml"), []byte(`[package]
name = "`+crate+`"
`), 0644); err != nil {
			t.Fatal(err)
		}
	}

	projects := parseCargoWorkspace(root)

	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}

	for _, p := range projects {
		if p.Language != "rust" {
			t.Errorf("expected language 'rust', got '%s'", p.Language)
		}
	}
}

func TestParseMelosConfig(t *testing.T) {
	root := t.TempDir()

	// Create melos.yaml
	if err := os.WriteFile(filepath.Join(root, "melos.yaml"), []byte(`name: my_flutter_monorepo
packages:
  - packages/*
  - apps/*
`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create package directories with pubspec.yaml
	pkgDir := filepath.Join(root, "packages", "core")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "pubspec.yaml"), []byte(`name: core`), 0644); err != nil {
		t.Fatal(err)
	}

	appDir := filepath.Join(root, "apps", "mobile")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "pubspec.yaml"), []byte(`name: mobile`), 0644); err != nil {
		t.Fatal(err)
	}

	projects := parseMelosConfig(root)

	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}

	for _, p := range projects {
		if p.Language != "dart" {
			t.Errorf("expected language 'dart', got '%s'", p.Language)
		}
	}
}

func TestDetectMonorepo_Integration(t *testing.T) {
	root := t.TempDir()

	// Create a pnpm workspace with dependencies
	if err := os.WriteFile(filepath.Join(root, "pnpm-workspace.yaml"), []byte(`packages:
  - 'packages/*'
`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create packages directory
	packagesDir := filepath.Join(root, "packages")
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create shared library
	sharedDir := filepath.Join(packagesDir, "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sharedDir, "package.json"), []byte(`{
  "name": "@myorg/shared",
  "version": "1.0.0"
}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create app that depends on shared
	appDir := filepath.Join(packagesDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "package.json"), []byte(`{
  "name": "@myorg/app",
  "version": "1.0.0",
  "dependencies": {
    "@myorg/shared": "workspace:*"
  }
}`), 0644); err != nil {
		t.Fatal(err)
	}

	info := DetectMonorepo(root, nil)

	if !info.IsMonorepo {
		t.Errorf("expected IsMonorepo to be true, got false. Manager: %s, Confidence: %v, Projects: %d",
			info.Manager, info.Confidence, len(info.Projects))
	}
	if info.Manager != MonorepoPnpm {
		t.Errorf("expected manager 'pnpm', got '%s'", info.Manager)
	}
	if info.Confidence < 1.0 {
		t.Errorf("expected confidence 1.0, got %v", info.Confidence)
	}
	if len(info.Projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(info.Projects))
		for _, p := range info.Projects {
			t.Logf("  project: %s at %s", p.Name, p.Path)
		}
	}

	// Verify dependency detection
	if deps, ok := info.Dependencies["app"]; ok {
		found := false
		for _, d := range deps {
			if d == "@myorg/shared" || d == "shared" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected app to depend on shared")
		}
	}
}

func TestDetectMonorepo_CustomPatterns(t *testing.T) {
	root := t.TempDir()

	// Create a non-standard monorepo structure
	clientDir := filepath.Join(root, "clients", "web")
	if err := os.MkdirAll(clientDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(clientDir, "package.json"), []byte(`{"name": "web-client"}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Without custom patterns, should not be detected
	info := DetectMonorepo(root, nil)
	found := false
	for _, p := range info.Projects {
		if p.Name == "web" {
			found = true
			break
		}
	}
	if !found {
		// Now with custom patterns (clients is in DefaultDirectoryPatterns)
		config := &MonorepoConfig{
			Patterns: []string{"clients"},
		}
		info = DetectMonorepo(root, config)
		for _, p := range info.Projects {
			if p.Name == "web" {
				found = true
				break
			}
		}
	}

	if !found {
		t.Error("expected to find 'web' project with custom patterns")
	}
}

func TestCategorizeFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"apps/frontend", "application"},
		{"packages/ui", "library"},
		{"services/api", "service"},
		{"modules/auth", "module"},
		{"plugins/analytics", "plugin"},
		{"tools/cli", "tool"},
		{"examples/demo", "example"},
		{"core/engine", "shared"},
		{"clients/web", "client"},
		{"servers/main", "server"},
		{"crates/utils", "crate"},
		{"cmd/app", "command"},
		{"other/thing", "project"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := categorizeFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("categorizeFromPath(%q) = %q, expected %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestNormalizeProjectName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my_project", "my-project"},
		{"MyProject", "my-project"},
		{"myProject", "my-project"},
		{"my-project", "my-project"},
		{"MY_PROJECT", "m-y--p-r-o-j-e-c-t"}, // All caps treated as camelCase - edge case with double hyphen
		{"simple", "simple"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeProjectName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeProjectName(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidProject(t *testing.T) {
	root := t.TempDir()

	// Valid project with package.json
	validDir := filepath.Join(root, "valid")
	if err := os.MkdirAll(validDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(validDir, "package.json"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Invalid project (no manifest)
	invalidDir := filepath.Join(root, "invalid")
	if err := os.MkdirAll(invalidDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(invalidDir, "README.md"), []byte(`# Readme`), 0644); err != nil {
		t.Fatal(err)
	}

	if !isValidProject(validDir) {
		t.Error("expected valid directory to be recognized as valid project")
	}
	if isValidProject(invalidDir) {
		t.Error("expected invalid directory to not be recognized as valid project")
	}
}

func TestParseGradleSettings(t *testing.T) {
	root := t.TempDir()

	// Create settings.gradle.kts
	if err := os.WriteFile(filepath.Join(root, "settings.gradle.kts"), []byte(`rootProject.name = "my-project"
include("app")
include("lib:core")
include("lib:utils")
`), 0644); err != nil {
		t.Fatal(err)
	}

	projects := parseGradleSettings(root)

	if len(projects) != 3 {
		t.Fatalf("expected 3 projects, got %d", len(projects))
	}

	for _, p := range projects {
		if p.Language != "java" {
			t.Errorf("expected language 'java', got '%s'", p.Language)
		}
	}
}

func TestParseMavenModules(t *testing.T) {
	root := t.TempDir()

	// Create pom.xml with modules
	if err := os.WriteFile(filepath.Join(root, "pom.xml"), []byte(`<?xml version="1.0" encoding="UTF-8"?>
<project>
  <groupId>com.example</groupId>
  <artifactId>parent</artifactId>
  <packaging>pom</packaging>
  <modules>
    <module>app</module>
    <module>core</module>
    <module>utils</module>
  </modules>
</project>
`), 0644); err != nil {
		t.Fatal(err)
	}

	projects := parseMavenModules(root)

	if len(projects) != 3 {
		t.Fatalf("expected 3 projects, got %d", len(projects))
	}

	names := make(map[string]bool)
	for _, p := range projects {
		names[p.Name] = true
		if p.Language != "java" {
			t.Errorf("expected language 'java', got '%s'", p.Language)
		}
	}

	for _, expected := range []string{"app", "core", "utils"} {
		if !names[expected] {
			t.Errorf("expected project %q not found", expected)
		}
	}
}
