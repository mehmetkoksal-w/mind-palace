package jsonc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestClean(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "removes single-line comments",
			input: `{
				// comment
				"key": "value"
			}`,
		},
		{
			name:  "removes multi-line comments",
			input: `{"key": /* comment */ "value"}`,
		},
		{
			name:  "plain JSON passes through",
			input: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Clean([]byte(tt.input))
			// Verify result is valid JSON
			var dest map[string]any
			if err := json.Unmarshal(result, &dest); err != nil {
				t.Errorf("Clean() produced invalid JSON: %v", err)
			}
			if dest["key"] != "value" {
				t.Errorf("Clean() key = %v, want %q", dest["key"], "value")
			}
		})
	}
}

func TestCleanEmptyInput(t *testing.T) {
	result := Clean([]byte{})
	if len(result) != 0 {
		t.Errorf("Clean([]) = %q, want empty", string(result))
	}
}

func TestDecodeFile(t *testing.T) {
	t.Run("decodes valid JSONC file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.jsonc")

		content := `{
			// This is a comment
			"name": "test",
			"count": 42
		}`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		var dest struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}

		if err := DecodeFile(path, &dest); err != nil {
			t.Fatalf("DecodeFile() error = %v", err)
		}

		if dest.Name != "test" {
			t.Errorf("Name = %q, want %q", dest.Name, "test")
		}
		if dest.Count != 42 {
			t.Errorf("Count = %d, want %d", dest.Count, 42)
		}
	})

	t.Run("decodes file with block comments", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "block.jsonc")

		content := `{
			/* block comment */
			"enabled": true
		}`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		var dest struct {
			Enabled bool `json:"enabled"`
		}

		if err := DecodeFile(path, &dest); err != nil {
			t.Fatalf("DecodeFile() error = %v", err)
		}

		if !dest.Enabled {
			t.Error("Enabled = false, want true")
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		var dest map[string]any
		err := DecodeFile("/nonexistent/path.jsonc", &dest)
		if err == nil {
			t.Error("DecodeFile() expected error for missing file")
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "invalid.jsonc")

		if err := os.WriteFile(path, []byte(`{invalid json`), 0o644); err != nil {
			t.Fatal(err)
		}

		var dest map[string]any
		err := DecodeFile(path, &dest)
		if err == nil {
			t.Error("DecodeFile() expected error for invalid JSON")
		}
	})
}
