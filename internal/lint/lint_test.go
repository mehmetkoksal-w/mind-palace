package lint

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRun(t *testing.T) {
	t.Run("returns nil for valid palace structure", func(t *testing.T) {
		dir := t.TempDir()
		palaceDir := filepath.Join(dir, ".palace")
		if err := os.MkdirAll(palaceDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Create valid palace.jsonc
		palaceConfig := `{
			"schemaVersion": "1.0.0",
			"kind": "palace/config",
			"project": {"name": "test"},
			"provenance": {"createdBy": "test", "createdAt": "2024-01-01T00:00:00Z"}
		}`
		if err := os.WriteFile(filepath.Join(palaceDir, "palace.jsonc"), []byte(palaceConfig), 0644); err != nil {
			t.Fatal(err)
		}

		// Create valid project-profile.json
		profile := `{
			"schemaVersion": "1.0.0",
			"kind": "palace/project-profile",
			"projectRoot": "/test",
			"capabilities": {},
			"provenance": {"createdBy": "test", "createdAt": "2024-01-01T00:00:00Z"}
		}`
		if err := os.WriteFile(filepath.Join(palaceDir, "project-profile.json"), []byte(profile), 0644); err != nil {
			t.Fatal(err)
		}

		err := Run(dir)
		if err != nil {
			t.Errorf("Run() error = %v", err)
		}
	})

	t.Run("returns error for invalid palace config", func(t *testing.T) {
		dir := t.TempDir()
		palaceDir := filepath.Join(dir, ".palace")
		if err := os.MkdirAll(palaceDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Create invalid palace.jsonc
		if err := os.WriteFile(filepath.Join(palaceDir, "palace.jsonc"), []byte(`{"invalid": true}`), 0644); err != nil {
			t.Fatal(err)
		}

		err := Run(dir)
		if err == nil {
			t.Error("Run() expected error for invalid palace config")
		}
	})

	t.Run("returns error for missing palace directory", func(t *testing.T) {
		dir := t.TempDir()

		err := Run(dir)
		if err == nil {
			t.Error("Run() expected error for missing palace directory")
		}
	})
}
