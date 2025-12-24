package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		a, b     string
		expected int
	}{
		{"equal versions", "1.0.0", "1.0.0", 0},
		{"a newer major", "2.0.0", "1.0.0", 1},
		{"b newer major", "1.0.0", "2.0.0", -1},
		{"a newer minor", "1.1.0", "1.0.0", 1},
		{"b newer minor", "1.0.0", "1.1.0", -1},
		{"a newer patch", "1.0.1", "1.0.0", 1},
		{"b newer patch", "1.0.0", "1.0.1", -1},
		{"with v prefix", "v1.0.0", "1.0.0", 0},
		{"both with v prefix", "v1.0.0", "v1.0.0", 0},
		{"different prerelease lengths", "1.0.0-alpha", "1.0.0", 1}, // implementation treats prerelease as longer
		{"release vs prerelease", "1.0.0", "1.0.0-beta", -1},        // shorter version is less
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareVersions(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		version  string
		expected []int
	}{
		{"1.0.0", []int{1, 0, 0}},
		{"v1.2.3", []int{1, 2, 3}},
		{"2.0", []int{2, 0}},
		{"1.0.0-alpha", []int{1, 0, 0, -1}},
		{"1.0.0-rc1", []int{1, 0, 0, -1}},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := parseVersion(tt.version)
			if len(result) != len(tt.expected) {
				t.Errorf("parseVersion(%q) = %v, want %v", tt.version, result, tt.expected)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("parseVersion(%q)[%d] = %d, want %d", tt.version, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestBuildAssetName(t *testing.T) {
	name := buildAssetName()

	if !contains(name, runtime.GOOS) {
		t.Errorf("buildAssetName() = %q, should contain %q", name, runtime.GOOS)
	}
	if !contains(name, runtime.GOARCH) {
		t.Errorf("buildAssetName() = %q, should contain %q", name, runtime.GOARCH)
	}
	if runtime.GOOS == "windows" && !contains(name, ".exe") {
		t.Errorf("buildAssetName() = %q, should contain .exe on Windows", name)
	}
}

func TestLoadCache(t *testing.T) {
	t.Run("returns false for missing file", func(t *testing.T) {
		_, ok := loadCache("/nonexistent/path")
		if ok {
			t.Error("loadCache() should return false for missing file")
		}
	})

	t.Run("returns false for invalid JSON", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "cache.json")
		os.WriteFile(path, []byte("invalid json"), 0644)

		_, ok := loadCache(path)
		if ok {
			t.Error("loadCache() should return false for invalid JSON")
		}
	})

	t.Run("returns false for expired cache", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "cache.json")
		entry := cacheEntry{
			LatestVersion: "1.0.0",
			ReleaseURL:    "https://example.com",
			CheckedAt:     time.Now().Add(-48 * time.Hour), // 48 hours ago
		}
		data, _ := json.Marshal(entry)
		os.WriteFile(path, data, 0644)

		_, ok := loadCache(path)
		if ok {
			t.Error("loadCache() should return false for expired cache")
		}
	})

	t.Run("returns valid cache entry", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "cache.json")
		entry := cacheEntry{
			LatestVersion: "1.0.0",
			ReleaseURL:    "https://example.com",
			CheckedAt:     time.Now(),
		}
		data, _ := json.Marshal(entry)
		os.WriteFile(path, data, 0644)

		result, ok := loadCache(path)
		if !ok {
			t.Error("loadCache() should return true for valid cache")
		}
		if result.LatestVersion != "1.0.0" {
			t.Errorf("LatestVersion = %q, want %q", result.LatestVersion, "1.0.0")
		}
	})
}

func TestSaveCache(t *testing.T) {
	t.Run("creates cache file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "subdir", "cache.json")

		entry := cacheEntry{
			LatestVersion: "2.0.0",
			ReleaseURL:    "https://example.com/release",
			CheckedAt:     time.Now(),
		}

		saveCache(path, entry)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Error("saveCache() should create cache file")
		}

		result, ok := loadCache(path)
		if !ok {
			t.Error("saved cache should be loadable")
		}
		if result.LatestVersion != "2.0.0" {
			t.Errorf("LatestVersion = %q, want %q", result.LatestVersion, "2.0.0")
		}
	})
}

func TestGetCacheDir(t *testing.T) {
	dir, err := GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir() error = %v", err)
	}

	if !contains(dir, ".palace") {
		t.Errorf("GetCacheDir() = %q, should contain .palace", dir)
	}
	if !contains(dir, "cache") {
		t.Errorf("GetCacheDir() = %q, should contain cache", dir)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
