package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildProfile(t *testing.T) {
	t.Run("builds profile for Go project", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644); err != nil {
			t.Fatal(err)
		}

		profile := BuildProfile(dir)

		if profile.SchemaVersion != "1.0.0" {
			t.Errorf("SchemaVersion = %q, want %q", profile.SchemaVersion, "1.0.0")
		}
		if profile.Kind != "palace/project-profile" {
			t.Errorf("Kind = %q, want %q", profile.Kind, "palace/project-profile")
		}
		if len(profile.Languages) == 0 || profile.Languages[0] != "go" {
			t.Errorf("Languages = %v, want [go]", profile.Languages)
		}
		if profile.Capabilities["tests.run"].Command != "go test ./..." {
			t.Errorf("tests.run command = %q, want %q", profile.Capabilities["tests.run"].Command, "go test ./...")
		}
	})

	t.Run("builds profile for JavaScript project", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		profile := BuildProfile(dir)

		if len(profile.Languages) == 0 || profile.Languages[0] != "javascript" {
			t.Errorf("Languages = %v, want [javascript]", profile.Languages)
		}
		if profile.Capabilities["tests.run"].Command != "npm test" {
			t.Errorf("tests.run command = %q, want %q", profile.Capabilities["tests.run"].Command, "npm test")
		}
	})

	t.Run("builds profile for Python project", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		profile := BuildProfile(dir)

		if len(profile.Languages) == 0 || profile.Languages[0] != "python" {
			t.Errorf("Languages = %v, want [python]", profile.Languages)
		}
		if profile.Capabilities["tests.run"].Command != "pytest" {
			t.Errorf("tests.run command = %q, want %q", profile.Capabilities["tests.run"].Command, "pytest")
		}
	})

	t.Run("detects unknown for empty project", func(t *testing.T) {
		dir := t.TempDir()

		profile := BuildProfile(dir)

		if len(profile.Languages) == 0 || profile.Languages[0] != "unknown" {
			t.Errorf("Languages = %v, want [unknown]", profile.Languages)
		}
	})

	t.Run("detects multiple languages", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		profile := BuildProfile(dir)

		if len(profile.Languages) < 2 {
			t.Errorf("Languages = %v, want at least 2 languages", profile.Languages)
		}
	})

	t.Run("includes all capabilities", func(t *testing.T) {
		dir := t.TempDir()
		profile := BuildProfile(dir)

		expectedCaps := []string{"search.text", "read.file", "graph.deps", "tests.run", "lint.run", "symbols.lookup"}
		for _, cap := range expectedCaps {
			if _, ok := profile.Capabilities[cap]; !ok {
				t.Errorf("missing capability: %s", cap)
			}
		}
	})

	t.Run("includes provenance", func(t *testing.T) {
		dir := t.TempDir()
		profile := BuildProfile(dir)

		if profile.Provenance["createdBy"] != "palace detect" {
			t.Errorf("createdBy = %q, want %q", profile.Provenance["createdBy"], "palace detect")
		}
		if profile.Provenance["createdAt"] == "" {
			t.Error("createdAt should not be empty")
		}
	})
}

func TestDetectLanguages(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		expected []string
	}{
		{
			name:     "Rust project",
			files:    map[string]string{"Cargo.toml": ""},
			expected: []string{"rust"},
		},
		{
			name:     "Dart project",
			files:    map[string]string{"pubspec.yaml": ""},
			expected: []string{"dart"},
		},
		{
			name:     "Ruby project",
			files:    map[string]string{"Gemfile": ""},
			expected: []string{"ruby"},
		},
		{
			name:     "Java project with Maven",
			files:    map[string]string{"pom.xml": ""},
			expected: []string{"java"},
		},
		{
			name:     "Java project with Gradle",
			files:    map[string]string{"build.gradle": ""},
			expected: []string{"java"},
		},
		{
			name:     "Java project with Gradle Kotlin",
			files:    map[string]string{"build.gradle.kts": ""},
			expected: []string{"java"},
		},
		{
			name:     "Swift project",
			files:    map[string]string{"Package.swift": ""},
			expected: []string{"swift"},
		},
		{
			name:     "PHP project",
			files:    map[string]string{"composer.json": ""},
			expected: []string{"php"},
		},
		{
			name:     "Python with pyproject.toml",
			files:    map[string]string{"pyproject.toml": ""},
			expected: []string{"python"},
		},
		{
			name:     "Python with setup.py",
			files:    map[string]string{"setup.py": ""},
			expected: []string{"python"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for filename, content := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			result := detectLanguages(dir)

			if len(result) != len(tt.expected) || result[0] != tt.expected[0] {
				t.Errorf("detectLanguages() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDetectLanguagesCSharp(t *testing.T) {
	t.Run("detects C# with .csproj", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "project.csproj"), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		result := detectLanguages(dir)
		found := false
		for _, lang := range result {
			if lang == "csharp" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("detectLanguages() = %v, expected to contain 'csharp'", result)
		}
	})

	t.Run("detects C# with .sln", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "solution.sln"), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		result := detectLanguages(dir)
		found := false
		for _, lang := range result {
			if lang == "csharp" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("detectLanguages() = %v, expected to contain 'csharp'", result)
		}
	})
}

func TestHasFileWithExtension(t *testing.T) {
	t.Run("returns true when file with extension exists", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "test.csproj"), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		if !hasFileWithExtension(dir, ".csproj") {
			t.Error("hasFileWithExtension() = false, want true")
		}
	})

	t.Run("returns false when no file with extension exists", func(t *testing.T) {
		dir := t.TempDir()

		if hasFileWithExtension(dir, ".csproj") {
			t.Error("hasFileWithExtension() = true, want false")
		}
	})

	t.Run("returns false for invalid directory", func(t *testing.T) {
		if hasFileWithExtension("/nonexistent/path", ".txt") {
			t.Error("hasFileWithExtension() = true, want false for invalid dir")
		}
	})

	t.Run("ignores directories", func(t *testing.T) {
		dir := t.TempDir()
		// Create a directory with .csproj extension (unusual but possible)
		if err := os.Mkdir(filepath.Join(dir, "folder.csproj"), 0755); err != nil {
			t.Fatal(err)
		}

		if hasFileWithExtension(dir, ".csproj") {
			t.Error("hasFileWithExtension() should ignore directories")
		}
	})
}

func TestDefaultGraphCommand(t *testing.T) {
	tests := []struct {
		langs    []string
		expected string
	}{
		{[]string{"go"}, "go list -deps ./..."},
		{[]string{"javascript"}, "npm ls"},
		{[]string{"dart"}, "dart pub deps"},
		{[]string{"rust"}, "cargo tree"},
		{[]string{"python"}, "pip list"},
		{[]string{"ruby"}, "bundle list"},
		{[]string{"java"}, "gradle dependencies || mvn dependency:tree"},
		{[]string{"csharp"}, "dotnet list package"},
		{[]string{"swift"}, "swift package show-dependencies"},
		{[]string{"php"}, "composer show"},
		{[]string{"unknown"}, "echo graph deps not configured"},
		{[]string{}, "echo graph deps not configured"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := defaultGraphCommand(tt.langs)
			if result != tt.expected {
				t.Errorf("defaultGraphCommand(%v) = %q, want %q", tt.langs, result, tt.expected)
			}
		})
	}
}

func TestDefaultTestCommand(t *testing.T) {
	tests := []struct {
		langs    []string
		expected string
	}{
		{[]string{"go"}, "go test ./..."},
		{[]string{"javascript"}, "npm test"},
		{[]string{"dart"}, "flutter test || dart test"},
		{[]string{"rust"}, "cargo test"},
		{[]string{"python"}, "pytest"},
		{[]string{"ruby"}, "bundle exec rspec"},
		{[]string{"java"}, "gradle test || mvn test"},
		{[]string{"csharp"}, "dotnet test"},
		{[]string{"swift"}, "swift test"},
		{[]string{"php"}, "composer test || ./vendor/bin/phpunit"},
		{[]string{"unknown"}, "echo tests not configured"},
		{[]string{}, "echo tests not configured"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := defaultTestCommand(tt.langs)
			if result != tt.expected {
				t.Errorf("defaultTestCommand(%v) = %q, want %q", tt.langs, result, tt.expected)
			}
		})
	}
}

func TestDefaultLintCommand(t *testing.T) {
	tests := []struct {
		langs    []string
		expected string
	}{
		{[]string{"go"}, "go vet ./..."},
		{[]string{"javascript"}, "npm run lint"},
		{[]string{"dart"}, "flutter analyze || dart analyze"},
		{[]string{"rust"}, "cargo clippy"},
		{[]string{"python"}, "ruff check . || flake8"},
		{[]string{"ruby"}, "bundle exec rubocop"},
		{[]string{"java"}, "gradle check || mvn verify"},
		{[]string{"csharp"}, "dotnet format --verify-no-changes"},
		{[]string{"swift"}, "swiftlint"},
		{[]string{"php"}, "composer lint || ./vendor/bin/phpstan analyse"},
		{[]string{"unknown"}, "echo lint not configured"},
		{[]string{}, "echo lint not configured"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := defaultLintCommand(tt.langs)
			if result != tt.expected {
				t.Errorf("defaultLintCommand(%v) = %q, want %q", tt.langs, result, tt.expected)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	t.Run("returns true for existing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.txt")
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		if !fileExists(path) {
			t.Error("fileExists() = false, want true")
		}
	})

	t.Run("returns false for non-existing file", func(t *testing.T) {
		if fileExists("/nonexistent/path/file.txt") {
			t.Error("fileExists() = true, want false")
		}
	})

	t.Run("returns true for directory", func(t *testing.T) {
		dir := t.TempDir()
		// fileExists should return true for directories too
		if !fileExists(dir) {
			t.Error("fileExists(dir) = false, want true")
		}
	})
}

func TestPriorityLanguageSelection(t *testing.T) {
	// When multiple languages are detected, first detected gets priority for commands
	dir := t.TempDir()
	// Add Go first (alphabetically earlier in detection order)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644); err != nil {
		t.Fatal(err)
	}
	// Add JavaScript
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	profile := BuildProfile(dir)

	// First language detected should dictate commands (Go comes before JavaScript in detection)
	if profile.Capabilities["tests.run"].Command != "go test ./..." {
		t.Errorf("Expected Go test command, got %q", profile.Capabilities["tests.run"].Command)
	}
}

func TestHasMonorepoSubproject(t *testing.T) {
	t.Run("detects Flutter monorepo with apps/*/pubspec.yaml", func(t *testing.T) {
		dir := t.TempDir()
		appsDir := filepath.Join(dir, "apps", "my_app")
		if err := os.MkdirAll(appsDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(appsDir, "pubspec.yaml"), []byte("name: my_app"), 0644); err != nil {
			t.Fatal(err)
		}

		if !hasMonorepoSubproject(dir, "pubspec.yaml") {
			t.Error("hasMonorepoSubproject() = false, want true")
		}
	})

	t.Run("detects Flutter monorepo with packages/*/pubspec.yaml", func(t *testing.T) {
		dir := t.TempDir()
		pkgDir := filepath.Join(dir, "packages", "shared_utils")
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pkgDir, "pubspec.yaml"), []byte("name: shared_utils"), 0644); err != nil {
			t.Fatal(err)
		}

		if !hasMonorepoSubproject(dir, "pubspec.yaml") {
			t.Error("hasMonorepoSubproject() = false, want true")
		}
	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		dir := t.TempDir()
		if hasMonorepoSubproject(dir, "pubspec.yaml") {
			t.Error("hasMonorepoSubproject() = true, want false")
		}
	})

	t.Run("returns false when apps dir exists but no subprojects", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "apps", "empty_dir"), 0755); err != nil {
			t.Fatal(err)
		}

		if hasMonorepoSubproject(dir, "pubspec.yaml") {
			t.Error("hasMonorepoSubproject() = true, want false")
		}
	})

	t.Run("detects Java monorepo with modules/*/build.gradle", func(t *testing.T) {
		dir := t.TempDir()
		moduleDir := filepath.Join(dir, "modules", "api")
		if err := os.MkdirAll(moduleDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(moduleDir, "build.gradle"), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		if !hasMonorepoSubproject(dir, "build.gradle") {
			t.Error("hasMonorepoSubproject() = false, want true")
		}
	})
}

func TestDetectLanguagesWithMelos(t *testing.T) {
	t.Run("detects Dart with melos.yaml at root", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "melos.yaml"), []byte("name: my_project"), 0644); err != nil {
			t.Fatal(err)
		}

		result := detectLanguages(dir)

		found := false
		for _, lang := range result {
			if lang == "dart" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("detectLanguages() = %v, expected to contain 'dart'", result)
		}
	})
}

func TestDetectLanguagesWithMonorepoPattern(t *testing.T) {
	t.Run("detects Dart in monorepo without root pubspec.yaml", func(t *testing.T) {
		dir := t.TempDir()
		// Create a Flutter monorepo structure without root pubspec.yaml
		appsDir := filepath.Join(dir, "apps", "driver_app")
		if err := os.MkdirAll(appsDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(appsDir, "pubspec.yaml"), []byte("name: driver_app"), 0644); err != nil {
			t.Fatal(err)
		}

		result := detectLanguages(dir)

		found := false
		for _, lang := range result {
			if lang == "dart" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("detectLanguages() = %v, expected to contain 'dart' for monorepo pattern", result)
		}
	})

	t.Run("detects Java in monorepo with modules pattern", func(t *testing.T) {
		dir := t.TempDir()
		moduleDir := filepath.Join(dir, "modules", "core")
		if err := os.MkdirAll(moduleDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(moduleDir, "build.gradle"), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		result := detectLanguages(dir)

		found := false
		for _, lang := range result {
			if lang == "java" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("detectLanguages() = %v, expected to contain 'java' for monorepo pattern", result)
		}
	})
}
