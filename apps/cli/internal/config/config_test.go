package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/koksalmehmet/mind-palace/apps/cli/schemas"
)

func TestLoadGuardrailsMergeExtendsDefaults(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".palace"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	palacePath := filepath.Join(dir, ".palace", "palace.jsonc")
	content := `{
        "guardrails": {
            "doNotTouchGlobs": ["custom/**", "zzz/**", ".git/**"],
            "readOnlyGlobs": ["readonly/**"]
        }
    }`
	if err := os.WriteFile(palacePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write palace: %v", err)
	}

	g := LoadGuardrails(dir)
	expectedDoNot := append([]string{}, defaultGuardrails().DoNotTouchGlobs...)
	expectedDoNot = append(expectedDoNot, "custom/**", "zzz/**")
	if !equalSlices(g.DoNotTouchGlobs, expectedDoNot) {
		t.Fatalf("doNotTouchGlobs mismatch: got %v, want %v", g.DoNotTouchGlobs, expectedDoNot)
	}
	expectedRO := []string{"readonly/**"}
	if !equalSlices(g.ReadOnlyGlobs, expectedRO) {
		t.Fatalf("readOnlyGlobs mismatch: got %v, want %v", g.ReadOnlyGlobs, expectedRO)
	}
}

func TestGuardrailNormalizationOrder(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".palace"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	palacePath := filepath.Join(dir, ".palace", "palace.jsonc")
	content := `{
        "guardrails": {
            "doNotTouchGlobs": ["  custom\\\\**  ", "zzz/**", ".git/**"]
        }
    }`
	if err := os.WriteFile(palacePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write palace: %v", err)
	}

	g := LoadGuardrails(dir)
	defaults := defaultGuardrails().DoNotTouchGlobs
	if len(g.DoNotTouchGlobs) != len(defaults)+2 {
		t.Fatalf("unexpected merged length: %v", g.DoNotTouchGlobs)
	}
	if g.DoNotTouchGlobs[len(defaults)] != "custom/**" || g.DoNotTouchGlobs[len(defaults)+1] != "zzz/**" {
		t.Fatalf("user globs ordering incorrect: %v", g.DoNotTouchGlobs)
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestCopySchemasRefreshesDrift(t *testing.T) {
	dir := t.TempDir()
	if _, err := EnsureLayout(dir); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	schemaDir := filepath.Join(dir, ".palace", "schemas")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatalf("mkdir schemas: %v", err)
	}

	// Write a drifted schema copy.
	dest := filepath.Join(schemaDir, "context-pack.schema.json")
	if err := os.WriteFile(dest, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write drifted: %v", err)
	}

	if err := CopySchemas(dir, false); err != nil {
		t.Fatalf("copy schemas: %v", err)
	}

	embedded, err := schemas.List()
	if err != nil {
		t.Fatalf("list schemas: %v", err)
	}
	want := embedded["context-pack"]
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("schema not refreshed to embedded copy")
	}
}
func TestWriteTemplate(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "room.jsonc")

	// Should fail for unknown template
	err := WriteTemplate(dest, "nonexistent", nil, false)
	if err == nil {
		t.Error("expected error for nonexistent template")
	}

	// Should succeed for valid template
	err = WriteTemplate(dest, "rooms/project-overview.jsonc", map[string]string{"name": "test"}, false)
	if err != nil {
		t.Fatalf("WriteTemplate failed: %v", err)
	}

	if _, err := os.Stat(dest); err != nil {
		t.Error("Expected file to be created")
	}
}

func TestLoadPalaceConfigCorrupted(t *testing.T) {
	dir := t.TempDir()
	palaceDir := filepath.Join(dir, ".palace")
	os.MkdirAll(palaceDir, 0o755)

	os.WriteFile(filepath.Join(palaceDir, "palace.jsonc"), []byte("{ broken json"), 0o644)

	_, err := LoadPalaceConfig(dir)
	if err == nil {
		t.Error("expected error for corrupted config")
	}

	// LoadGuardrails should fallback to default on error
	g := LoadGuardrails(dir)
	if len(g.DoNotTouchGlobs) == 0 {
		t.Error("expected default guardrails when config is corrupted")
	}
}

func TestMergeGlobs(t *testing.T) {
	defaults := []string{"a", "b"}
	user := []string{"b", "c", "  ", ""}
	merged := mergeGlobs(defaults, user)

	expected := []string{"a", "b", "c"}
	if !equalSlices(merged, expected) {
		t.Errorf("got %v, want %v", merged, expected)
	}
}

func TestWriteJSONError(t *testing.T) {
	// Root IS a file, so WriteJSON should fail to create directory/file if path is invalid
	dir := t.TempDir()
	path := filepath.Join(dir, "file")
	os.WriteFile(path, []byte("test"), 0o644)

	err := WriteJSON(filepath.Join(path, "impossible"), map[string]string{})
	if err == nil {
		t.Error("expected error for impossible path")
	}
}

func TestWriteJSON(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "test.json")
	data := map[string]string{"foo": "bar"}

	if err := WriteJSON(dest, data); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	content, _ := os.ReadFile(dest)
	if !strings.Contains(string(content), `"foo": "bar"`) {
		t.Errorf("Unexpected content: %s", string(content))
	}
}

func TestNormalizeGlob(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"  foo/bar  ", "foo/bar"},
		{"foo\\\\bar", "foo/bar"},
		{"foo//bar", "foo/bar"},
		{"", ""},
		{"  ", ""},
	}
	for _, c := range cases {
		got := normalizeGlob(c.input)
		if got != c.expected {
			t.Errorf("normalizeGlob(%q) = %q, want %q", c.input, got, c.expected)
		}
	}
}

func TestEnsureLayoutErrors(t *testing.T) {
	// Root is a file, MkdirAll should fail
	dir := t.TempDir()
	path := filepath.Join(dir, "file")
	os.WriteFile(path, []byte("test"), 0o644)

	_, err := EnsureLayout(filepath.Join(path, "subdir"))
	if err == nil {
		t.Error("expected error when root path prefix is a file")
	}
}
