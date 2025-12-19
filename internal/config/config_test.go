package config

import (
	"os"
	"path/filepath"
	"testing"
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
	expectedDoNot := append(defaultGuardrails().DoNotTouchGlobs, "custom/**", "zzz/**")
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
