package corridor

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
)

func TestNamespacePath(t *testing.T) {
	tests := []struct {
		neighbor string
		path     string
		expected string
	}{
		{"backend", "src/api.ts", "corridor://backend/src/api.ts"},
		{"backend", "/src/api.ts", "corridor://backend/src/api.ts"},
		{"frontend", "components/Button.tsx", "corridor://frontend/components/Button.tsx"},
		{"core", "", "corridor://core/"},
	}

	for _, tt := range tests {
		result := NamespacePath(tt.neighbor, tt.path)
		if result != tt.expected {
			t.Errorf("NamespacePath(%q, %q) = %q, want %q", tt.neighbor, tt.path, result, tt.expected)
		}
	}
}

func TestParseNamespacedPath(t *testing.T) {
	tests := []struct {
		path           string
		wantNeighbor   string
		wantRelative   string
		wantIsCorridor bool
	}{
		{"corridor://backend/src/api.ts", "backend", "src/api.ts", true},
		{"corridor://frontend/components/Button.tsx", "frontend", "components/Button.tsx", true},
		{"corridor://core/", "core", "", true},
		{"corridor://core", "core", "", true},
		{"src/local.go", "", "src/local.go", false},
		{"/absolute/path.go", "", "/absolute/path.go", false},
	}

	for _, tt := range tests {
		neighbor, relative, isCorridor := ParseNamespacedPath(tt.path)
		if neighbor != tt.wantNeighbor || relative != tt.wantRelative || isCorridor != tt.wantIsCorridor {
			t.Errorf("ParseNamespacedPath(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.path, neighbor, relative, isCorridor,
				tt.wantNeighbor, tt.wantRelative, tt.wantIsCorridor)
		}
	}
}

func TestParseTTL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "24h0m0s"},
		{"24h", "24h0m0s"},
		{"1h", "1h0m0s"},
		{"30m", "30m0s"},
		{"invalid", "24h0m0s"},
	}

	for _, tt := range tests {
		result := parseTTL(tt.input)
		if result.String() != tt.expected {
			t.Errorf("parseTTL(%q) = %s, want %s", tt.input, result.String(), tt.expected)
		}
	}
}

func TestExpandEnv(t *testing.T) {
	t.Setenv("TEST_TOKEN", "secret123")

	tests := []struct {
		input    string
		expected string
	}{
		{"$TEST_TOKEN", "secret123"},
		{"$NONEXISTENT", ""},
		{"literal", "literal"},
		{"", ""},
	}

	for _, tt := range tests {
		result := expandEnv(tt.input)
		if result != tt.expected {
			t.Errorf("expandEnv(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestStripJSONComments(t *testing.T) {
	input := `{
  // This is a comment
  "name": "test",
  "value": 123 // inline comment
}`

	result := stripJSONComments([]byte(input))

	// Parse result to verify it's valid JSON with correct values
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("stripJSONComments produced invalid JSON: %v\nGot: %s", err, result)
	}

	if parsed["name"] != "test" {
		t.Errorf("expected name='test', got %v", parsed["name"])
	}
	if parsed["value"] != float64(123) {
		t.Errorf("expected value=123, got %v", parsed["value"])
	}
}

func TestStripJSONCommentsPreservesURLs(t *testing.T) {
	// This tests that URLs with // in strings are NOT treated as comments
	input := `{
  "url": "https://example.com/path",
  "another": "http://test.com" // this is a comment
}`

	result := stripJSONComments([]byte(input))

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("stripJSONComments produced invalid JSON: %v\nGot: %s", err, result)
	}

	// Verify URLs are preserved
	if parsed["url"] != "https://example.com/path" {
		t.Errorf("URL was incorrectly modified: got %v", parsed["url"])
	}
	if parsed["another"] != "http://test.com" {
		t.Errorf("URL was incorrectly modified: got %v", parsed["another"])
	}
}

func TestApplyAuth(t *testing.T) {
	t.Setenv("AUTH_TOKEN", "test-bearer-token")
	t.Setenv("AUTH_USER", "testuser")
	t.Setenv("AUTH_PASS", "testpass")
	t.Setenv("AUTH_VALUE", "custom-value")

	tests := []struct {
		name       string
		auth       *config.AuthConfig
		wantHeader string
		wantValue  string
	}{
		{
			name:       "nil auth does nothing",
			auth:       nil,
			wantHeader: "",
			wantValue:  "",
		},
		{
			name: "bearer auth",
			auth: &config.AuthConfig{
				Type:  "bearer",
				Token: "$AUTH_TOKEN",
			},
			wantHeader: "Authorization",
			wantValue:  "Bearer test-bearer-token",
		},
		{
			name: "bearer auth with literal token",
			auth: &config.AuthConfig{
				Type:  "bearer",
				Token: "literal-token",
			},
			wantHeader: "Authorization",
			wantValue:  "Bearer literal-token",
		},
		{
			name: "header auth",
			auth: &config.AuthConfig{
				Type:   "header",
				Header: "X-Custom-Auth",
				Value:  "$AUTH_VALUE",
			},
			wantHeader: "X-Custom-Auth",
			wantValue:  "custom-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com", nil)
			applyAuth(req, tt.auth)

			if tt.wantHeader == "" {
				// No headers should be set for nil auth
				return
			}

			got := req.Header.Get(tt.wantHeader)
			if got != tt.wantValue {
				t.Errorf("Header %q = %q, want %q", tt.wantHeader, got, tt.wantValue)
			}
		})
	}
}

func TestApplyAuthBasic(t *testing.T) {
	t.Setenv("AUTH_USER", "testuser")
	t.Setenv("AUTH_PASS", "testpass")

	req := httptest.NewRequest("GET", "http://example.com", nil)
	auth := &config.AuthConfig{
		Type: "basic",
		User: "$AUTH_USER",
		Pass: "$AUTH_PASS",
	}
	applyAuth(req, auth)

	user, pass, ok := req.BasicAuth()
	if !ok {
		t.Fatal("expected basic auth to be set")
	}
	if user != "testuser" {
		t.Errorf("user = %q, want %q", user, "testuser")
	}
	if pass != "testpass" {
		t.Errorf("pass = %q, want %q", pass, "testpass")
	}
}

func TestCacheOperations(t *testing.T) {
	tmpDir := t.TempDir()

	// Test writeCacheMeta and loadCacheMeta
	meta := CacheMeta{
		FetchedAt: time.Now().UTC().Truncate(time.Second),
		ETag:      "test-etag",
		URL:       "https://example.com/context-pack.json",
		TTL:       "1h",
	}

	if err := writeCacheMeta(tmpDir, meta); err != nil {
		t.Fatalf("writeCacheMeta error: %v", err)
	}

	loadedMeta, err := loadCacheMeta(tmpDir)
	if err != nil {
		t.Fatalf("loadCacheMeta error: %v", err)
	}

	if loadedMeta.ETag != meta.ETag {
		t.Errorf("ETag = %q, want %q", loadedMeta.ETag, meta.ETag)
	}
	if loadedMeta.URL != meta.URL {
		t.Errorf("URL = %q, want %q", loadedMeta.URL, meta.URL)
	}
	if loadedMeta.TTL != meta.TTL {
		t.Errorf("TTL = %q, want %q", loadedMeta.TTL, meta.TTL)
	}
}

func TestLoadCacheMetaMissing(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := loadCacheMeta(tmpDir)
	if err == nil {
		t.Error("expected error for missing cache meta")
	}
}

func TestCacheContextPack(t *testing.T) {
	tmpDir := t.TempDir()

	data := []byte(`{"goal": "test", "schemaVersion": "1.0.0"}`)
	if err := cacheContextPack(tmpDir, data); err != nil {
		t.Fatalf("cacheContextPack error: %v", err)
	}

	// Verify the file was created
	cpPath := filepath.Join(tmpDir, "context-pack.json")
	if _, err := os.Stat(cpPath); err != nil {
		t.Errorf("context-pack.json not created: %v", err)
	}
}

func TestCacheRooms(t *testing.T) {
	tmpDir := t.TempDir()

	rooms := []model.Room{
		{Name: "room1", EntryPoints: []string{"main.go"}},
		{Name: "room2", EntryPoints: []string{"index.ts"}},
	}

	if err := cacheRooms(tmpDir, rooms); err != nil {
		t.Fatalf("cacheRooms error: %v", err)
	}

	// Verify rooms were created
	roomsDir := filepath.Join(tmpDir, "rooms")
	for _, room := range rooms {
		roomPath := filepath.Join(roomsDir, room.Name+".json")
		if _, err := os.Stat(roomPath); err != nil {
			t.Errorf("room file %s not created: %v", room.Name, err)
		}
	}
}

func TestCacheRoomsEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	// Empty rooms should not error
	if err := cacheRooms(tmpDir, nil); err != nil {
		t.Errorf("cacheRooms with empty rooms should not error: %v", err)
	}

	if err := cacheRooms(tmpDir, []model.Room{}); err != nil {
		t.Errorf("cacheRooms with empty slice should not error: %v", err)
	}
}

func TestLoadFromCache(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a context pack
	cp := model.ContextPack{
		SchemaVersion: "1.0.0",
		Goal:          "test goal",
	}
	cpData, _ := json.Marshal(cp)
	if err := os.WriteFile(filepath.Join(tmpDir, "context-pack.json"), cpData, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create rooms
	roomsDir := filepath.Join(tmpDir, "rooms")
	if err := os.MkdirAll(roomsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	room := model.Room{Name: "test-room"}
	roomData, _ := json.Marshal(room)
	if err := os.WriteFile(filepath.Join(roomsDir, "test-room.json"), roomData, 0o644); err != nil {
		t.Fatal(err)
	}

	// Load from cache
	loadedCP, loadedRooms, err := loadFromCache(tmpDir)
	if err != nil {
		t.Fatalf("loadFromCache error: %v", err)
	}

	if loadedCP.Goal != "test goal" {
		t.Errorf("Goal = %q, want %q", loadedCP.Goal, "test goal")
	}

	if len(loadedRooms) != 1 {
		t.Errorf("len(rooms) = %d, want 1", len(loadedRooms))
	}
}

func TestLoadFromCacheMissing(t *testing.T) {
	tmpDir := t.TempDir()

	_, _, err := loadFromCache(tmpDir)
	if err == nil {
		t.Error("expected error for missing cache")
	}
}

func TestLoadRoomsFromCacheDir(t *testing.T) {
	tmpDir := t.TempDir()
	roomsDir := filepath.Join(tmpDir, "rooms")
	if err := os.MkdirAll(roomsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create valid room
	room := model.Room{Name: "valid-room"}
	roomData, _ := json.Marshal(room)
	if err := os.WriteFile(filepath.Join(roomsDir, "valid.json"), roomData, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create invalid JSON file (should be skipped)
	if err := os.WriteFile(filepath.Join(roomsDir, "invalid.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	rooms := loadRoomsFromCacheDir(tmpDir)
	if len(rooms) != 1 {
		t.Errorf("len(rooms) = %d, want 1", len(rooms))
	}
	if rooms[0].Name != "valid-room" {
		t.Errorf("room name = %q, want %q", rooms[0].Name, "valid-room")
	}
}

func TestFetchNeighborsWithDisabled(t *testing.T) {
	tmpDir := t.TempDir()

	disabled := false
	neighbors := map[string]config.NeighborConfig{
		"disabled": {
			Enabled: &disabled, // explicitly disabled
			URL:     "http://example.com",
		},
	}

	result, err := FetchNeighbors(tmpDir, neighbors)
	if err != nil {
		t.Fatalf("FetchNeighbors error: %v", err)
	}

	// Disabled neighbor should not be fetched
	if len(result.Corridors) != 0 {
		t.Errorf("len(Corridors) = %d, want 0 (disabled neighbor)", len(result.Corridors))
	}
}

func TestFetchNeighborsNoURLOrPath(t *testing.T) {
	tmpDir := t.TempDir()

	neighbors := map[string]config.NeighborConfig{
		"empty": {}, // no URL or localPath
	}

	result, err := FetchNeighbors(tmpDir, neighbors)
	if err != nil {
		t.Fatalf("FetchNeighbors error: %v", err)
	}

	if len(result.Corridors) != 1 {
		t.Fatalf("len(Corridors) = %d, want 1", len(result.Corridors))
	}

	if result.Corridors[0].Error == "" {
		t.Error("expected error for neighbor with no URL or path")
	}
}

func TestFetchFromURLWithServer(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cp := model.ContextPack{
		SchemaVersion: "1.0.0",
		Goal:          "server test",
	}
	payload, _ := json.Marshal(cp)
	withTestTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return newResponse(http.StatusOK, string(payload)), nil
	}))

	neighbor := config.NeighborConfig{URL: "https://example.com/test"}

	ctx := fetchFromURL("test", neighbor, cacheDir)
	if ctx.Error != "" {
		t.Errorf("unexpected error: %s", ctx.Error)
	}
	if ctx.ContextPack == nil {
		t.Error("expected context pack to be fetched")
	}
	if ctx.ContextPack != nil && ctx.ContextPack.Goal != "server test" {
		t.Errorf("Goal = %q, want %q", ctx.ContextPack.Goal, "server test")
	}
}

func TestFetchFromURLServerError(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}

	withTestTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return newResponse(http.StatusInternalServerError, "fail"), nil
	}))

	neighbor := config.NeighborConfig{URL: "https://example.com/error"}

	ctx := fetchFromURL("test", neighbor, cacheDir)
	if ctx.Error == "" {
		t.Error("expected error for server error")
	}
}

func TestFetchFromLocal(t *testing.T) {
	// Create a local palace structure
	localDir := t.TempDir()
	palaceDir := filepath.Join(localDir, ".palace")
	outputsDir := filepath.Join(palaceDir, "outputs")
	roomsDir := filepath.Join(palaceDir, "rooms")

	for _, dir := range []string{outputsDir, roomsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Create context pack
	cp := model.ContextPack{
		SchemaVersion: "1.0.0",
		Goal:          "local test",
	}
	cpData, _ := json.Marshal(cp)
	if err := os.WriteFile(filepath.Join(outputsDir, "context-pack.json"), cpData, 0o644); err != nil {
		t.Fatal(err)
	}

	cacheDir := t.TempDir()
	ctx := fetchFromLocal(localDir, "local-test", localDir, cacheDir)

	if ctx.Error != "" {
		t.Errorf("unexpected error: %s", ctx.Error)
	}
	if ctx.ContextPack == nil {
		t.Error("expected context pack")
	}
	if ctx.ContextPack != nil && ctx.ContextPack.Goal != "local test" {
		t.Errorf("Goal = %q, want %q", ctx.ContextPack.Goal, "local test")
	}
}

func TestFallbackToCache(t *testing.T) {
	// Setup cache with data
	cacheDir := t.TempDir()
	cp := model.ContextPack{SchemaVersion: "1.0.0", Goal: "cached"}
	cpData, _ := json.Marshal(cp)
	if err := os.WriteFile(filepath.Join(cacheDir, "context-pack.json"), cpData, 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := CorridorContext{Name: "test"}
	result := fallbackToCache(ctx, cacheDir, "test error")

	if result.ContextPack == nil {
		t.Error("expected context pack from cache")
	}
	if !result.FromCache {
		t.Error("expected FromCache to be true")
	}
	if result.Error == "" {
		t.Error("expected error message to be preserved")
	}
}

func TestFallbackToCacheNoCache(t *testing.T) {
	cacheDir := t.TempDir()
	ctx := CorridorContext{Name: "test"}
	result := fallbackToCache(ctx, cacheDir, "test error")

	if result.ContextPack != nil {
		t.Error("expected nil context pack when no cache")
	}
	if result.Error == "" {
		t.Error("expected error message")
	}
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
		Status:     http.StatusText(status),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
