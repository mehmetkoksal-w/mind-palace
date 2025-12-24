package collect

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
)

func TestMergeOrderedUnique(t *testing.T) {
	tests := []struct {
		name      string
		primary   []string
		secondary []string
		expected  []string
	}{
		{
			name:      "both empty",
			primary:   []string{},
			secondary: []string{},
			expected:  []string{},
		},
		{
			name:      "primary only",
			primary:   []string{"a", "b", "c"},
			secondary: []string{},
			expected:  []string{"a", "b", "c"},
		},
		{
			name:      "secondary only",
			primary:   []string{},
			secondary: []string{"x", "y", "z"},
			expected:  []string{"x", "y", "z"},
		},
		{
			name:      "no duplicates",
			primary:   []string{"a", "b"},
			secondary: []string{"c", "d"},
			expected:  []string{"a", "b", "c", "d"},
		},
		{
			name:      "with duplicates",
			primary:   []string{"a", "b", "c"},
			secondary: []string{"b", "c", "d"},
			expected:  []string{"a", "b", "c", "d"},
		},
		{
			name:      "primary has duplicates internally",
			primary:   []string{"a", "a", "b"},
			secondary: []string{"c"},
			expected:  []string{"a", "b", "c"},
		},
		{
			name:      "preserves primary order",
			primary:   []string{"z", "a", "m"},
			secondary: []string{"b", "c"},
			expected:  []string{"z", "a", "m", "b", "c"},
		},
		{
			name:      "nil primary",
			primary:   nil,
			secondary: []string{"a", "b"},
			expected:  []string{"a", "b"},
		},
		{
			name:      "nil secondary",
			primary:   []string{"a", "b"},
			secondary: nil,
			expected:  []string{"a", "b"},
		},
		{
			name:      "both nil",
			primary:   nil,
			secondary: nil,
			expected:  []string{},
		},
		{
			name:      "filters empty strings",
			primary:   []string{"a", "", "b"},
			secondary: []string{"", "c", ""},
			expected:  []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeOrderedUnique(tt.primary, tt.secondary)

			// Handle nil vs empty slice comparison
			if len(result) == 0 && len(tt.expected) == 0 {
				return // Both empty, pass
			}

			if len(result) != len(tt.expected) {
				t.Errorf("length mismatch: got %d, want %d\ngot: %v\nwant: %v",
					len(result), len(tt.expected), result, tt.expected)
				return
			}

			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("index %d: got %q, want %q", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestFilterExisting(t *testing.T) {
	stored := map[string]index.FileMetadata{
		"src/main.go":    {Hash: "abc123", Size: 100, ModTime: time.Now()},
		"src/utils.go":   {Hash: "def456", Size: 200, ModTime: time.Now()},
		"pkg/handler.go": {Hash: "ghi789", Size: 300, ModTime: time.Now()},
	}

	tests := []struct {
		name     string
		paths    []string
		expected []string
	}{
		{
			name:     "all exist",
			paths:    []string{"src/main.go", "src/utils.go"},
			expected: []string{"src/main.go", "src/utils.go"},
		},
		{
			name:     "none exist",
			paths:    []string{"nonexistent.go", "missing.go"},
			expected: []string{},
		},
		{
			name:     "some exist",
			paths:    []string{"src/main.go", "nonexistent.go", "pkg/handler.go"},
			expected: []string{"src/main.go", "pkg/handler.go"},
		},
		{
			name:     "empty paths",
			paths:    []string{},
			expected: []string{},
		},
		{
			name:     "nil paths",
			paths:    nil,
			expected: []string{},
		},
		{
			name:     "preserves order",
			paths:    []string{"pkg/handler.go", "src/utils.go", "src/main.go"},
			expected: []string{"pkg/handler.go", "src/utils.go", "src/main.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterExisting(tt.paths, stored)

			if len(result) == 0 && len(tt.expected) == 0 {
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("length mismatch: got %d, want %d\ngot: %v\nwant: %v",
					len(result), len(tt.expected), result, tt.expected)
				return
			}

			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("index %d: got %q, want %q", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestFilterExistingEmptyStore(t *testing.T) {
	stored := map[string]index.FileMetadata{}
	paths := []string{"a.go", "b.go", "c.go"}

	result := filterExisting(paths, stored)

	if len(result) != 0 {
		t.Errorf("expected empty result for empty store, got %v", result)
	}
}

func TestPrioritizeHits(t *testing.T) {
	hits := []index.ChunkHit{
		{Path: "src/main.go", ChunkIndex: 0, StartLine: 1, EndLine: 10},
		{Path: "src/utils.go", ChunkIndex: 0, StartLine: 1, EndLine: 10},
		{Path: "pkg/handler.go", ChunkIndex: 0, StartLine: 1, EndLine: 10},
		{Path: "pkg/router.go", ChunkIndex: 0, StartLine: 1, EndLine: 10},
	}

	tests := []struct {
		name         string
		changedPaths []string
		expectedFirst []string // Expected paths at the beginning
	}{
		{
			name:          "no changed paths",
			changedPaths:  []string{},
			expectedFirst: []string{"src/main.go", "src/utils.go"}, // Original order
		},
		{
			name:          "nil changed paths",
			changedPaths:  nil,
			expectedFirst: []string{"src/main.go", "src/utils.go"},
		},
		{
			name:          "prioritize single",
			changedPaths:  []string{"pkg/handler.go"},
			expectedFirst: []string{"pkg/handler.go"},
		},
		{
			name:          "prioritize multiple",
			changedPaths:  []string{"pkg/handler.go", "pkg/router.go"},
			expectedFirst: []string{"pkg/handler.go", "pkg/router.go"},
		},
		{
			name:          "changed paths not in hits",
			changedPaths:  []string{"nonexistent.go"},
			expectedFirst: []string{"src/main.go"}, // Original order preserved
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := prioritizeHits(hits, tt.changedPaths)

			if len(result) != len(hits) {
				t.Errorf("result length changed: got %d, want %d", len(result), len(hits))
				return
			}

			// Check first elements match expected
			for i, expected := range tt.expectedFirst {
				if i >= len(result) {
					break
				}
				if result[i].Path != expected {
					t.Errorf("position %d: got path %q, want %q", i, result[i].Path, expected)
				}
			}
		})
	}
}

func TestPrioritizeHitsEmptyInput(t *testing.T) {
	result := prioritizeHits([]index.ChunkHit{}, []string{"a.go"})
	if len(result) != 0 {
		t.Errorf("expected empty result for empty hits, got %v", result)
	}
}

func TestPrioritizeHitsPreservesAllHits(t *testing.T) {
	hits := []index.ChunkHit{
		{Path: "a.go", ChunkIndex: 0},
		{Path: "b.go", ChunkIndex: 1},
		{Path: "c.go", ChunkIndex: 2},
	}
	changedPaths := []string{"c.go"}

	result := prioritizeHits(hits, changedPaths)

	// All hits should still be present
	pathSet := make(map[string]bool)
	for _, h := range result {
		pathSet[h.Path] = true
	}

	for _, h := range hits {
		if !pathSet[h.Path] {
			t.Errorf("missing hit for path %q", h.Path)
		}
	}
}

func TestCollectEntryPoints(t *testing.T) {
	t.Run("empty room name returns nil", func(t *testing.T) {
		result := collectEntryPoints("/some/path", "")
		if result != nil {
			t.Errorf("expected nil for empty room name, got %v", result)
		}
	})

	t.Run("nonexistent room file returns nil", func(t *testing.T) {
		tmpDir := t.TempDir()
		result := collectEntryPoints(tmpDir, "nonexistent")
		if result != nil {
			t.Errorf("expected nil for nonexistent room, got %v", result)
		}
	})

	t.Run("invalid room file returns nil", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .palace/rooms directory
		roomsDir := filepath.Join(tmpDir, ".palace", "rooms")
		if err := os.MkdirAll(roomsDir, 0755); err != nil {
			t.Fatalf("failed to create rooms dir: %v", err)
		}

		// Create an invalid room file (missing schema)
		roomContent := `{ invalid json }`
		roomPath := filepath.Join(roomsDir, "badroom.jsonc")
		if err := os.WriteFile(roomPath, []byte(roomContent), 0644); err != nil {
			t.Fatalf("failed to write room file: %v", err)
		}

		// Should return nil for invalid room file (validation fails)
		result := collectEntryPoints(tmpDir, "badroom")
		if result != nil {
			t.Errorf("expected nil for invalid room file, got %v", result)
		}
	})
}

func TestRunRequiresIndex(t *testing.T) {
	tmpDir := t.TempDir()

	// Create minimal palace layout without index
	palaceDir := filepath.Join(tmpDir, ".palace")
	if err := os.MkdirAll(filepath.Join(palaceDir, "outputs"), 0755); err != nil {
		t.Fatalf("failed to create palace dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(palaceDir, "rooms"), 0755); err != nil {
		t.Fatalf("failed to create rooms dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(palaceDir, "playbooks"), 0755); err != nil {
		t.Fatalf("failed to create playbooks dir: %v", err)
	}

	// Create palace.jsonc
	palaceConfig := `{
  "$schema": "./schemas/palace.schema.json",
  "name": "test-palace"
}`
	if err := os.WriteFile(filepath.Join(palaceDir, "palace.jsonc"), []byte(palaceConfig), 0644); err != nil {
		t.Fatalf("failed to write palace config: %v", err)
	}

	// Create schemas
	schemasDir := filepath.Join(palaceDir, "schemas")
	if err := os.MkdirAll(schemasDir, 0755); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	palaceSchema := `{"$schema": "http://json-schema.org/draft-07/schema#", "type": "object"}`
	if err := os.WriteFile(filepath.Join(schemasDir, "palace.schema.json"), []byte(palaceSchema), 0644); err != nil {
		t.Fatalf("failed to write palace schema: %v", err)
	}

	_, err := Run(tmpDir, "", Options{})
	if err == nil {
		t.Error("expected error when no index exists")
	}
}

func TestOptions(t *testing.T) {
	// Test that Options struct works correctly
	opts := Options{
		AllowStale: true,
	}

	if !opts.AllowStale {
		t.Error("AllowStale should be true")
	}

	opts2 := Options{}
	if opts2.AllowStale {
		t.Error("Default AllowStale should be false")
	}
}

func TestResult(t *testing.T) {
	// Test Result struct
	result := Result{
		CorridorWarnings: []string{"warning1", "warning2"},
	}

	if len(result.CorridorWarnings) != 2 {
		t.Errorf("expected 2 warnings, got %d", len(result.CorridorWarnings))
	}
}

// Benchmark tests
func BenchmarkMergeOrderedUnique(b *testing.B) {
	primary := make([]string, 100)
	secondary := make([]string, 100)
	for i := 0; i < 100; i++ {
		primary[i] = filepath.Join("src", "file"+string(rune('a'+i%26))+".go")
		secondary[i] = filepath.Join("pkg", "file"+string(rune('a'+i%26))+".go")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mergeOrderedUnique(primary, secondary)
	}
}

func BenchmarkFilterExisting(b *testing.B) {
	stored := make(map[string]index.FileMetadata)
	paths := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		path := filepath.Join("src", "file"+string(rune('a'+i%26))+".go")
		paths[i] = path
		if i%2 == 0 {
			stored[path] = index.FileMetadata{Hash: "hash", Size: 100}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filterExisting(paths, stored)
	}
}

func BenchmarkPrioritizeHits(b *testing.B) {
	hits := make([]index.ChunkHit, 100)
	changedPaths := make([]string, 20)
	for i := 0; i < 100; i++ {
		hits[i] = index.ChunkHit{
			Path:      filepath.Join("src", "file"+string(rune('a'+i%26))+".go"),
			StartLine: i * 10,
			EndLine:   i*10 + 10,
		}
	}
	for i := 0; i < 20; i++ {
		changedPaths[i] = hits[i*5].Path
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		prioritizeHits(hits, changedPaths)
	}
}
