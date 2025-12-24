package lint

import (
	"os"
	"path/filepath"
	"strings"
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

	t.Run("validates rooms", func(t *testing.T) {
		dir := t.TempDir()
		palaceDir := filepath.Join(dir, ".palace")
		roomsDir := filepath.Join(palaceDir, "rooms")
		if err := os.MkdirAll(roomsDir, 0755); err != nil {
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

		// Create valid room
		room := `{
			"schemaVersion": "1.0.0",
			"kind": "palace/room",
			"name": "test-room",
			"summary": "Test room",
			"entryPoints": ["main.go"],
			"provenance": {"createdBy": "test", "createdAt": "2024-01-01T00:00:00Z"}
		}`
		if err := os.WriteFile(filepath.Join(roomsDir, "test.jsonc"), []byte(room), 0644); err != nil {
			t.Fatal(err)
		}

		err := Run(dir)
		if err != nil {
			t.Errorf("Run() error = %v", err)
		}
	})

	t.Run("returns error for invalid room", func(t *testing.T) {
		dir := t.TempDir()
		palaceDir := filepath.Join(dir, ".palace")
		roomsDir := filepath.Join(palaceDir, "rooms")
		if err := os.MkdirAll(roomsDir, 0755); err != nil {
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

		// Create invalid room (missing required fields)
		if err := os.WriteFile(filepath.Join(roomsDir, "bad.jsonc"), []byte(`{"name": "only-name"}`), 0644); err != nil {
			t.Fatal(err)
		}

		err := Run(dir)
		if err == nil {
			t.Error("Run() expected error for invalid room")
		}
	})

	t.Run("validates playbooks", func(t *testing.T) {
		dir := t.TempDir()
		palaceDir := filepath.Join(dir, ".palace")
		playbooksDir := filepath.Join(palaceDir, "playbooks")
		if err := os.MkdirAll(playbooksDir, 0755); err != nil {
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

		// Create valid playbook
		playbook := `{
			"schemaVersion": "1.0.0",
			"kind": "palace/playbook",
			"name": "test-playbook",
			"summary": "Test playbook",
			"rooms": ["room1"],
			"provenance": {"createdBy": "test", "createdAt": "2024-01-01T00:00:00Z"}
		}`
		if err := os.WriteFile(filepath.Join(playbooksDir, "test.jsonc"), []byte(playbook), 0644); err != nil {
			t.Fatal(err)
		}

		err := Run(dir)
		if err != nil {
			t.Errorf("Run() error = %v", err)
		}
	})

	t.Run("returns error for invalid playbook", func(t *testing.T) {
		dir := t.TempDir()
		palaceDir := filepath.Join(dir, ".palace")
		playbooksDir := filepath.Join(palaceDir, "playbooks")
		if err := os.MkdirAll(playbooksDir, 0755); err != nil {
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

		// Create invalid playbook (missing required fields)
		if err := os.WriteFile(filepath.Join(playbooksDir, "bad.jsonc"), []byte(`{"name": "only-name"}`), 0644); err != nil {
			t.Fatal(err)
		}

		err := Run(dir)
		if err == nil {
			t.Error("Run() expected error for invalid playbook")
		}
	})

	t.Run("collects multiple errors", func(t *testing.T) {
		dir := t.TempDir()
		palaceDir := filepath.Join(dir, ".palace")
		roomsDir := filepath.Join(palaceDir, "rooms")
		playbooksDir := filepath.Join(palaceDir, "playbooks")
		if err := os.MkdirAll(roomsDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(playbooksDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Create invalid palace.jsonc
		if err := os.WriteFile(filepath.Join(palaceDir, "palace.jsonc"), []byte(`{"bad": true}`), 0644); err != nil {
			t.Fatal(err)
		}

		// Create invalid room
		if err := os.WriteFile(filepath.Join(roomsDir, "bad.jsonc"), []byte(`{"bad": true}`), 0644); err != nil {
			t.Fatal(err)
		}

		// Create invalid playbook
		if err := os.WriteFile(filepath.Join(playbooksDir, "bad.jsonc"), []byte(`{"bad": true}`), 0644); err != nil {
			t.Fatal(err)
		}

		err := Run(dir)
		if err == nil {
			t.Error("Run() expected error")
		}

		// Should have multiple errors joined
		errStr := err.Error()
		if !strings.Contains(errStr, "palace.jsonc") {
			t.Error("error should mention palace.jsonc")
		}
	})
}
