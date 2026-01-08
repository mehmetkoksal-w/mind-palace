package validate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJSONC(t *testing.T) {
	t.Run("validates valid palace config", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "palace.jsonc")

		content := `{
			// Palace configuration
			"schemaVersion": "1.0.0",
			"kind": "palace/config",
			"project": {
				"name": "test-palace"
			},
			"provenance": {
				"createdBy": "test",
				"createdAt": "2024-01-01T00:00:00Z"
			}
		}`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		err := JSONC(path, "palace")
		if err != nil {
			t.Errorf("JSONC() error = %v", err)
		}
	})

	t.Run("returns error for invalid data against schema", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "invalid.jsonc")

		// Missing required fields
		content := `{"invalid": true}`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		err := JSONC(path, "palace")
		if err == nil {
			t.Error("JSONC() expected validation error")
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		err := JSONC("/nonexistent/file.jsonc", "palace")
		if err == nil {
			t.Error("JSONC() expected error for missing file")
		}
	})

	t.Run("returns error for invalid schema name", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.jsonc")
		if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
			t.Fatal(err)
		}

		err := JSONC(path, "nonexistent-schema")
		if err == nil {
			t.Error("JSONC() expected error for invalid schema")
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "invalid.jsonc")
		if err := os.WriteFile(path, []byte(`{not valid`), 0o644); err != nil {
			t.Fatal(err)
		}

		err := JSONC(path, "palace")
		if err == nil {
			t.Error("JSONC() expected error for invalid JSON")
		}
	})
}

func TestJSON(t *testing.T) {
	t.Run("validates valid project profile", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "project-profile.json")

		content := `{
			"schemaVersion": "1.0.0",
			"kind": "palace/project-profile",
			"projectRoot": "/test/project",
			"capabilities": {},
			"provenance": {
				"createdBy": "test",
				"createdAt": "2024-01-01T00:00:00Z"
			}
		}`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		err := JSON(path, "project-profile")
		if err != nil {
			t.Errorf("JSON() error = %v", err)
		}
	})

	t.Run("returns error for invalid data against schema", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "invalid.json")

		// Missing required fields
		content := `{"invalid": true}`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		err := JSON(path, "project-profile")
		if err == nil {
			t.Error("JSON() expected validation error")
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		err := JSON("/nonexistent/file.json", "project-profile")
		if err == nil {
			t.Error("JSON() expected error for missing file")
		}
	})

	t.Run("returns error for invalid schema name", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.json")
		if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
			t.Fatal(err)
		}

		err := JSON(path, "nonexistent-schema")
		if err == nil {
			t.Error("JSON() expected error for invalid schema")
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "invalid.json")
		if err := os.WriteFile(path, []byte(`{not valid`), 0o644); err != nil {
			t.Fatal(err)
		}

		err := JSON(path, "project-profile")
		if err == nil {
			t.Error("JSON() expected error for invalid JSON")
		}
	})
}
