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
			name:     "Swift project",
			files:    map[string]string{"Package.swift": ""},
			expected: []string{"swift"},
		},
		{
			name:     "PHP project",
			files:    map[string]string{"composer.json": ""},
			expected: []string{"php"},
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
}
