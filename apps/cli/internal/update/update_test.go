package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
		{"major double digit", "10.0.0", "9.0.0", 1},
		{"minor double digit", "1.10.0", "1.9.0", 1},
		{"patch double digit", "1.0.10", "1.0.9", 1},
		{"empty vs version", "", "1.0.0", -1},
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
		{"1.0.0+build123", []int{1, 0, 0, -1}},
		{"10.20.30", []int{10, 20, 30}},
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
	if !contains(name, "palace-") {
		t.Errorf("buildAssetName() = %q, should start with palace-", name)
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

func TestCheckResult(t *testing.T) {
	result := CheckResult{
		CurrentVersion:  "1.0.0",
		LatestVersion:   "2.0.0",
		UpdateAvailable: true,
		ReleaseURL:      "https://github.com/owner/repo/releases/v2.0.0",
		DownloadURL:     "https://github.com/owner/repo/releases/download/v2.0.0/binary",
		ChecksumURL:     "https://github.com/owner/repo/releases/download/v2.0.0/binary.sha256",
	}

	if result.CurrentVersion != "1.0.0" {
		t.Errorf("CurrentVersion = %q, want %q", result.CurrentVersion, "1.0.0")
	}
	if result.LatestVersion != "2.0.0" {
		t.Errorf("LatestVersion = %q, want %q", result.LatestVersion, "2.0.0")
	}
	if !result.UpdateAvailable {
		t.Error("UpdateAvailable should be true")
	}
}

func TestRelease(t *testing.T) {
	release := Release{
		TagName: "v1.0.0",
		Name:    "Release 1.0.0",
		Body:    "Release notes here",
		HTMLURL: "https://github.com/owner/repo/releases/v1.0.0",
		Assets: []Asset{
			{
				Name:               "palace-darwin-amd64",
				Size:               1024000,
				BrowserDownloadURL: "https://github.com/owner/repo/releases/download/v1.0.0/palace-darwin-amd64",
			},
		},
	}

	if release.TagName != "v1.0.0" {
		t.Errorf("TagName = %q, want %q", release.TagName, "v1.0.0")
	}
	if len(release.Assets) != 1 {
		t.Errorf("Assets length = %d, want %d", len(release.Assets), 1)
	}
	if release.Assets[0].Size != 1024000 {
		t.Errorf("Asset Size = %d, want %d", release.Assets[0].Size, 1024000)
	}
}

func TestCacheEntry(t *testing.T) {
	entry := cacheEntry{
		LatestVersion: "1.2.3",
		ReleaseURL:    "https://example.com/release",
		CheckedAt:     time.Now(),
	}

	// Serialize and deserialize
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var decoded cacheEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if decoded.LatestVersion != entry.LatestVersion {
		t.Errorf("LatestVersion = %q, want %q", decoded.LatestVersion, entry.LatestVersion)
	}
	if decoded.ReleaseURL != entry.ReleaseURL {
		t.Errorf("ReleaseURL = %q, want %q", decoded.ReleaseURL, entry.ReleaseURL)
	}
}

func TestConstants(t *testing.T) {
	if GitHubOwner == "" {
		t.Error("GitHubOwner should not be empty")
	}
	if GitHubRepo == "" {
		t.Error("GitHubRepo should not be empty")
	}
	if cacheTTL <= 0 {
		t.Error("cacheTTL should be positive")
	}
	if downloadTimeout <= 0 {
		t.Error("downloadTimeout should be positive")
	}
	if checksumTimeout <= 0 {
		t.Error("checksumTimeout should be positive")
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func withTestTransport(t *testing.T, rt http.RoundTripper) {
	t.Helper()
	original := http.DefaultTransport
	http.DefaultTransport = rt
	t.Cleanup(func() {
		http.DefaultTransport = original
	})
}

func newResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestFetchLatestReleaseSuccess(t *testing.T) {
	release := Release{
		TagName: "v2.0.0",
		HTMLURL: "https://example.com/release/v2.0.0",
		Assets: []Asset{
			{Name: buildAssetName(), BrowserDownloadURL: "https://example.com/bin"},
			{Name: buildAssetName() + ".sha256", BrowserDownloadURL: "https://example.com/bin.sha256"},
		},
	}
	payload, _ := json.Marshal(release)

	withTestTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "api.github.com" {
			return nil, fmt.Errorf("unexpected host: %s", req.URL.Host)
		}
		return newResponse(http.StatusOK, string(payload)), nil
	}))

	got, err := fetchLatestRelease()
	if err != nil {
		t.Fatalf("fetchLatestRelease() error = %v", err)
	}
	if got.TagName != "v2.0.0" {
		t.Errorf("TagName = %q, want %q", got.TagName, "v2.0.0")
	}
	if len(got.Assets) != 2 {
		t.Errorf("Assets length = %d, want %d", len(got.Assets), 2)
	}
}

func TestFetchLatestReleaseNotFound(t *testing.T) {
	withTestTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return newResponse(http.StatusNotFound, "not found"), nil
	}))

	_, err := fetchLatestRelease()
	if err == nil || !strings.Contains(err.Error(), "no releases found") {
		t.Fatalf("fetchLatestRelease() error = %v, want no releases found", err)
	}
}

func TestDownloadToTemp(t *testing.T) {
	url := "https://example.com/download"
	withTestTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != url {
			return nil, fmt.Errorf("unexpected url: %s", req.URL.String())
		}
		return newResponse(http.StatusOK, "binary-data"), nil
	}))

	path, err := downloadToTemp(url)
	if err != nil {
		t.Fatalf("downloadToTemp() error = %v", err)
	}
	defer os.Remove(path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "binary-data" {
		t.Errorf("downloaded data = %q, want %q", string(data), "binary-data")
	}
}

func TestDownloadToTempStatusError(t *testing.T) {
	url := "https://example.com/download"
	withTestTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return newResponse(http.StatusInternalServerError, "fail"), nil
	}))

	_, err := downloadToTemp(url)
	if err == nil || !strings.Contains(err.Error(), "download failed") {
		t.Fatalf("downloadToTemp() error = %v, want status error", err)
	}
}

func TestVerifyChecksum(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.bin")
	content := []byte("hello-world")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	hash := sha256.Sum256(content)
	sum := hex.EncodeToString(hash[:])
	checksumURL := "https://example.com/checksum"

	withTestTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != checksumURL {
			return nil, fmt.Errorf("unexpected url: %s", req.URL.String())
		}
		return newResponse(http.StatusOK, sum+" file.bin"), nil
	}))

	if err := verifyChecksum(path, checksumURL); err != nil {
		t.Fatalf("verifyChecksum() error = %v", err)
	}
}

func TestVerifyChecksumInvalidFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.bin")
	if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	checksumURL := "https://example.com/checksum"
	withTestTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return newResponse(http.StatusOK, "   "), nil
	}))

	err := verifyChecksum(path, checksumURL)
	if err == nil || !strings.Contains(err.Error(), "invalid checksum format") {
		t.Fatalf("verifyChecksum() error = %v, want invalid checksum format", err)
	}
}

func TestCheckUpdateAvailable(t *testing.T) {
	release := Release{
		TagName: "v2.0.0",
		HTMLURL: "https://example.com/release/v2.0.0",
		Assets: []Asset{
			{Name: buildAssetName(), BrowserDownloadURL: "https://example.com/bin"},
			{Name: buildAssetName() + ".sha256", BrowserDownloadURL: "https://example.com/bin.sha256"},
		},
	}
	payload, _ := json.Marshal(release)

	withTestTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return newResponse(http.StatusOK, string(payload)), nil
	}))

	result, err := Check("1.0.0")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !result.UpdateAvailable {
		t.Error("UpdateAvailable should be true")
	}
	if result.DownloadURL == "" || result.ChecksumURL == "" {
		t.Error("DownloadURL and ChecksumURL should be set")
	}
}

func TestUpdateAlreadyLatest(t *testing.T) {
	release := Release{
		TagName: "v1.0.0",
		HTMLURL: "https://example.com/release/v1.0.0",
	}
	payload, _ := json.Marshal(release)

	withTestTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return newResponse(http.StatusOK, string(payload)), nil
	}))

	err := Update("1.0.0", nil)
	if err == nil || !strings.Contains(err.Error(), "already at latest version") {
		t.Fatalf("Update() error = %v, want already at latest version", err)
	}
}

func TestUpdateNoBinaryAvailable(t *testing.T) {
	release := Release{
		TagName: "v2.0.0",
		HTMLURL: "https://example.com/release/v2.0.0",
		Assets: []Asset{
			{Name: "unrelated-asset", BrowserDownloadURL: "https://example.com/other"},
		},
	}
	payload, _ := json.Marshal(release)

	withTestTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return newResponse(http.StatusOK, string(payload)), nil
	}))

	err := Update("1.0.0", nil)
	if err == nil || !strings.Contains(err.Error(), "no binary available") {
		t.Fatalf("Update() error = %v, want no binary available", err)
	}
}

func TestCheckCachedWithValidCache(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "update-check.json")
	entry := cacheEntry{
		LatestVersion: "2.0.0",
		ReleaseURL:    "https://example.com/release",
		CheckedAt:     time.Now(),
	}
	data, _ := json.Marshal(entry)
	os.WriteFile(cachePath, data, 0644)

	result, err := CheckCached("1.0.0", dir)
	if err != nil {
		t.Fatalf("CheckCached() error = %v", err)
	}
	if !result.UpdateAvailable {
		t.Error("UpdateAvailable should be true")
	}
	if result.LatestVersion != "2.0.0" {
		t.Errorf("LatestVersion = %q, want %q", result.LatestVersion, "2.0.0")
	}
}

func TestCheckCachedWithExpiredCache(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "update-check.json")
	entry := cacheEntry{
		LatestVersion: "2.0.0",
		ReleaseURL:    "https://example.com/release",
		CheckedAt:     time.Now().Add(-48 * time.Hour), // Expired
	}
	data, _ := json.Marshal(entry)
	os.WriteFile(cachePath, data, 0644)

	release := Release{
		TagName: "v3.0.0",
		HTMLURL: "https://example.com/release/v3.0.0",
	}
	payload, _ := json.Marshal(release)

	withTestTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return newResponse(http.StatusOK, string(payload)), nil
	}))

	result, err := CheckCached("1.0.0", dir)
	if err != nil {
		t.Fatalf("CheckCached() error = %v", err)
	}
	if result.LatestVersion != "3.0.0" {
		t.Errorf("LatestVersion = %q, want %q", result.LatestVersion, "3.0.0")
	}
}

func TestReplaceExecutable(t *testing.T) {
	dir := t.TempDir()

	// Create "current" executable
	currentPath := filepath.Join(dir, "current")
	if err := os.WriteFile(currentPath, []byte("old-content"), 0755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Create "new" executable
	newPath := filepath.Join(dir, "new")
	if err := os.WriteFile(newPath, []byte("new-content"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := replaceExecutable(currentPath, newPath); err != nil {
		t.Fatalf("replaceExecutable() error = %v", err)
	}

	// Verify content was replaced
	content, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "new-content" {
		t.Errorf("content = %q, want %q", string(content), "new-content")
	}

	// Verify backup was cleaned up
	backupPath := currentPath + ".backup"
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Error("backup file should have been removed")
	}
}

func TestReplaceExecutableFailsIfCurrentMissing(t *testing.T) {
	dir := t.TempDir()
	currentPath := filepath.Join(dir, "missing")
	newPath := filepath.Join(dir, "new")

	if err := os.WriteFile(newPath, []byte("new-content"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := replaceExecutable(currentPath, newPath)
	if err == nil {
		t.Error("replaceExecutable() should fail if current is missing")
	}
}

func TestReplaceExecutableFailsIfNewMissing(t *testing.T) {
	dir := t.TempDir()
	currentPath := filepath.Join(dir, "current")
	newPath := filepath.Join(dir, "missing")

	if err := os.WriteFile(currentPath, []byte("old-content"), 0755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := replaceExecutable(currentPath, newPath)
	if err == nil {
		t.Error("replaceExecutable() should fail if new is missing")
	}

	// Verify original was restored from backup
	content, _ := os.ReadFile(currentPath)
	if string(content) != "old-content" {
		t.Errorf("content = %q, want %q (original should be restored)", string(content), "old-content")
	}
}
