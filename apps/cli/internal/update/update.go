package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	GitHubOwner     = "koksalmehmet"
	GitHubRepo      = "mind-palace"
	releasesAPIURL  = "https://api.github.com/repos/%s/%s/releases/latest"
	cacheFileName   = "update-check.json"
	cacheTTL        = 24 * time.Hour
	downloadTimeout = 60 * time.Second
	checksumTimeout = 15 * time.Second
)

type Release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Body    string  `json:"body"`
	HTMLURL string  `json:"html_url"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type CheckResult struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	ReleaseURL      string
	DownloadURL     string
	ChecksumURL     string
}

type cacheEntry struct {
	LatestVersion string    `json:"latest_version"`
	ReleaseURL    string    `json:"release_url"`
	CheckedAt     time.Time `json:"checked_at"`
}

func Check(currentVersion string) (*CheckResult, error) {
	release, err := fetchLatestRelease()
	if err != nil {
		return nil, fmt.Errorf("fetch release: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentClean := strings.TrimPrefix(currentVersion, "v")

	result := &CheckResult{
		CurrentVersion:  currentClean,
		LatestVersion:   latestVersion,
		UpdateAvailable: compareVersions(currentClean, latestVersion) < 0,
		ReleaseURL:      release.HTMLURL,
	}

	if result.UpdateAvailable {
		assetName := buildAssetName()
		for _, asset := range release.Assets {
			if asset.Name == assetName {
				result.DownloadURL = asset.BrowserDownloadURL
			}
			if asset.Name == assetName+".sha256" {
				result.ChecksumURL = asset.BrowserDownloadURL
			}
		}
	}

	return result, nil
}

func CheckCached(currentVersion string, cacheDir string) (*CheckResult, error) {
	cachePath := filepath.Join(cacheDir, cacheFileName)
	if cached, ok := loadCache(cachePath); ok {
		currentClean := strings.TrimPrefix(currentVersion, "v")
		return &CheckResult{
			CurrentVersion:  currentClean,
			LatestVersion:   cached.LatestVersion,
			UpdateAvailable: compareVersions(currentClean, cached.LatestVersion) < 0,
			ReleaseURL:      cached.ReleaseURL,
		}, nil
	}

	result, err := Check(currentVersion)
	if err != nil {
		return nil, err
	}

	saveCache(cachePath, cacheEntry{
		LatestVersion: result.LatestVersion,
		ReleaseURL:    result.ReleaseURL,
		CheckedAt:     time.Now(),
	})

	return result, nil
}

func Update(currentVersion string, progressFn func(string)) error {
	if progressFn == nil {
		progressFn = func(string) {}
	}

	progressFn("Checking for updates...")
	result, err := Check(currentVersion)
	if err != nil {
		return err
	}

	if !result.UpdateAvailable {
		return errors.New("already at latest version")
	}

	if result.DownloadURL == "" {
		return fmt.Errorf("no binary available for %s/%s; download manually from %s", runtime.GOOS, runtime.GOARCH, result.ReleaseURL)
	}

	progressFn(fmt.Sprintf("Downloading v%s...", result.LatestVersion))
	tmpFile, err := downloadToTemp(result.DownloadURL)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer os.Remove(tmpFile)

	if result.ChecksumURL != "" {
		progressFn("Verifying checksum...")
		if err := verifyChecksum(tmpFile, result.ChecksumURL); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolve symlinks: %w", err)
	}

	progressFn("Installing update...")
	if err := replaceExecutable(execPath, tmpFile); err != nil {
		return fmt.Errorf("replace executable: %w", err)
	}

	progressFn(fmt.Sprintf("Updated to v%s", result.LatestVersion))
	return nil
}

func fetchLatestRelease() (*Release, error) {
	url := fmt.Sprintf(releasesAPIURL, GitHubOwner, GitHubRepo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "palace-cli")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("no releases found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: %s", resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func buildAssetName() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	ext := ""
	if goos == "windows" {
		ext = ".exe"
	}

	return fmt.Sprintf("palace-%s-%s%s", goos, goarch, ext)
}

func downloadToTemp(url string) (string, error) {
	client := &http.Client{Timeout: downloadTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}

	tmpFile, err := os.CreateTemp("", "palace-update-*")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

func verifyChecksum(filePath, checksumURL string) error {
	client := &http.Client{Timeout: checksumTimeout}
	resp, err := client.Get(checksumURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch checksum: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	parts := strings.Fields(string(body))
	if len(parts) == 0 {
		return errors.New("invalid checksum format")
	}
	expectedHash := strings.ToLower(parts[0])

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actualHash := hex.EncodeToString(h.Sum(nil))

	if actualHash != expectedHash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}

func replaceExecutable(currentPath, newPath string) error {
	backupPath := currentPath + ".backup"

	if err := os.Rename(currentPath, backupPath); err != nil {
		return fmt.Errorf("backup current: %w", err)
	}

	newFile, err := os.Open(newPath)
	if err != nil {
		os.Rename(backupPath, currentPath)
		return err
	}
	defer newFile.Close()

	destFile, err := os.OpenFile(currentPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		os.Rename(backupPath, currentPath)
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, newFile); err != nil {
		destFile.Close()
		os.Remove(currentPath)
		os.Rename(backupPath, currentPath)
		return err
	}

	os.Remove(backupPath)
	return nil
}

func loadCache(path string) (cacheEntry, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return cacheEntry{}, false
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return cacheEntry{}, false
	}

	if time.Since(entry.CheckedAt) > cacheTTL {
		return cacheEntry{}, false
	}

	return entry, true
}

func saveCache(path string, entry cacheEntry) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	os.WriteFile(path, data, 0644)
}

func compareVersions(a, b string) int {
	aParts := parseVersion(a)
	bParts := parseVersion(b)

	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		if aParts[i] < bParts[i] {
			return -1
		}
		if aParts[i] > bParts[i] {
			return 1
		}
	}

	if len(aParts) < len(bParts) {
		return -1
	}
	if len(aParts) > len(bParts) {
		return 1
	}

	return 0
}

func parseVersion(v string) []int {
	v = strings.TrimPrefix(v, "v")

	preRelease := ""
	if idx := strings.IndexAny(v, "-+"); idx != -1 {
		preRelease = v[idx:]
		v = v[:idx]
	}

	parts := strings.Split(v, ".")
	result := make([]int, 0, len(parts)+1)

	for _, p := range parts {
		var n int
		fmt.Sscanf(p, "%d", &n)
		result = append(result, n)
	}

	if preRelease != "" {
		result = append(result, -1)
	}

	return result
}

func GetCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".palace", "cache"), nil
}
