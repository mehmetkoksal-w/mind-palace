package corridor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/jsonc"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
)

type CacheMeta struct {
	FetchedAt time.Time `json:"fetchedAt"`
	ETag      string    `json:"etag,omitempty"`
	URL       string    `json:"url"`
	TTL       string    `json:"ttl"`
}

type CorridorContext struct {
	Name        string             `json:"name"`
	ContextPack *model.ContextPack `json:"contextPack,omitempty"`
	Rooms       []model.Room       `json:"rooms,omitempty"`
	FromCache   bool               `json:"fromCache"`
	FetchedAt   time.Time          `json:"fetchedAt"`
	Error       string             `json:"error,omitempty"`
}

type FetchResult struct {
	Corridors []CorridorContext `json:"corridors"`
	Errors    []string          `json:"errors,omitempty"`
}

func FetchNeighbors(root string, neighbors map[string]config.NeighborConfig) (*FetchResult, error) {
	cacheDir := filepath.Join(root, ".palace", "cache", "neighbors")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	result := &FetchResult{
		Corridors: make([]CorridorContext, 0, len(neighbors)),
	}

	for name, neighbor := range neighbors {
		if neighbor.Enabled != nil && !*neighbor.Enabled {
			continue
		}

		ctx := fetchNeighbor(root, name, neighbor, cacheDir)
		result.Corridors = append(result.Corridors, ctx)
		if ctx.Error != "" {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", name, ctx.Error))
		}
	}

	return result, nil
}

func fetchNeighbor(root, name string, neighbor config.NeighborConfig, cacheDir string) CorridorContext {
	ctx := CorridorContext{
		Name:      name,
		FetchedAt: time.Now().UTC(),
	}

	neighborCacheDir := filepath.Join(cacheDir, name)
	if err := os.MkdirAll(neighborCacheDir, 0o755); err != nil {
		ctx.Error = fmt.Sprintf("create neighbor cache: %v", err)
		return ctx
	}

	if neighbor.LocalPath != "" {
		return fetchFromLocal(root, name, neighbor.LocalPath, neighborCacheDir)
	}

	if neighbor.URL != "" {
		return fetchFromURL(name, neighbor, neighborCacheDir)
	}

	ctx.Error = "no url or localPath specified"
	return ctx
}

func fetchFromLocal(root, name, localPath string, cacheDir string) CorridorContext {
	ctx := CorridorContext{
		Name:      name,
		FetchedAt: time.Now().UTC(),
	}

	var absPath string
	if filepath.IsAbs(localPath) {
		absPath = localPath
	} else {
		absPath = filepath.Join(root, localPath)
	}

	cpPath := filepath.Join(absPath, ".palace", "outputs", "context-pack.json")
	cpData, err := os.ReadFile(cpPath)
	if err != nil {
		ctx.Error = fmt.Sprintf("read local context-pack: %v", err)
		return ctx
	}

	var cp model.ContextPack
	if err := json.Unmarshal(cpData, &cp); err != nil {
		ctx.Error = fmt.Sprintf("parse local context-pack: %v", err)
		return ctx
	}

	ctx.ContextPack = &cp

	roomsDir := filepath.Join(absPath, ".palace", "rooms")
	ctx.Rooms = loadRoomsFromDir(roomsDir)

	// Cache errors are non-critical - main operation already succeeded
	_ = cacheContextPack(cacheDir, cpData)
	_ = cacheRooms(cacheDir, ctx.Rooms)
	_ = writeCacheMeta(cacheDir, CacheMeta{
		FetchedAt: ctx.FetchedAt,
		URL:       "local://" + absPath,
		TTL:       "0",
	})

	return ctx
}

func fetchFromURL(name string, neighbor config.NeighborConfig, cacheDir string) CorridorContext {
	ctx := CorridorContext{
		Name:      name,
		FetchedAt: time.Now().UTC(),
	}

	meta, err := loadCacheMeta(cacheDir)
	ttl := parseTTL(neighbor.TTL)

	if err == nil && time.Since(meta.FetchedAt) < ttl {
		cp, rooms, err := loadFromCache(cacheDir)
		if err == nil {
			ctx.ContextPack = cp
			ctx.Rooms = rooms
			ctx.FromCache = true
			return ctx
		}
	}

	req, err := http.NewRequest("GET", neighbor.URL, nil)
	if err != nil {
		return fallbackToCache(ctx, cacheDir, fmt.Sprintf("create request: %v", err))
	}

	applyAuth(req, neighbor.Auth)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fallbackToCache(ctx, cacheDir, fmt.Sprintf("fetch: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fallbackToCache(ctx, cacheDir, fmt.Sprintf("HTTP %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fallbackToCache(ctx, cacheDir, fmt.Sprintf("read body: %v", err))
	}

	var cp model.ContextPack
	if err := json.Unmarshal(body, &cp); err != nil {
		return fallbackToCache(ctx, cacheDir, fmt.Sprintf("parse json: %v", err))
	}

	ctx.ContextPack = &cp

	// Cache errors are non-critical - main operation already succeeded
	_ = cacheContextPack(cacheDir, body)
	_ = writeCacheMeta(cacheDir, CacheMeta{
		FetchedAt: ctx.FetchedAt,
		ETag:      resp.Header.Get("ETag"),
		URL:       neighbor.URL,
		TTL:       neighbor.TTL,
	})

	return ctx
}

func applyAuth(req *http.Request, auth *config.AuthConfig) {
	if auth == nil {
		return
	}

	switch auth.Type {
	case "bearer":
		token := expandEnv(auth.Token)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	case "basic":
		user := expandEnv(auth.User)
		pass := expandEnv(auth.Pass)
		req.SetBasicAuth(user, pass)
	case "header":
		header := auth.Header
		value := expandEnv(auth.Value)
		if header != "" && value != "" {
			req.Header.Set(header, value)
		}
	}
}

func expandEnv(s string) string {
	if strings.HasPrefix(s, "$") {
		return os.Getenv(strings.TrimPrefix(s, "$"))
	}
	return s
}

func fallbackToCache(ctx CorridorContext, cacheDir, errMsg string) CorridorContext {
	cp, rooms, err := loadFromCache(cacheDir)
	if err != nil {
		ctx.Error = fmt.Sprintf("%s (no cache available)", errMsg)
		return ctx
	}

	ctx.ContextPack = cp
	ctx.Rooms = rooms
	ctx.FromCache = true
	ctx.Error = fmt.Sprintf("%s (using cache)", errMsg)
	return ctx
}

func loadFromCache(cacheDir string) (*model.ContextPack, []model.Room, error) {
	cpPath := filepath.Join(cacheDir, "context-pack.json")
	cpData, err := os.ReadFile(cpPath)
	if err != nil {
		return nil, nil, err
	}

	var cp model.ContextPack
	if err := json.Unmarshal(cpData, &cp); err != nil {
		return nil, nil, err
	}

	rooms := loadRoomsFromCacheDir(cacheDir)

	return &cp, rooms, nil
}

func cacheContextPack(cacheDir string, data []byte) error {
	path := filepath.Join(cacheDir, "context-pack.json")
	return os.WriteFile(path, data, 0o644)
}

func cacheRooms(cacheDir string, rooms []model.Room) error {
	if len(rooms) == 0 {
		return nil
	}

	roomsDir := filepath.Join(cacheDir, "rooms")
	if err := os.MkdirAll(roomsDir, 0o755); err != nil {
		return err
	}

	var successCount int
	var lastErr error
	for _, room := range rooms {
		data, err := json.MarshalIndent(room, "", "  ")
		if err != nil {
			lastErr = err
			continue
		}
		path := filepath.Join(roomsDir, room.Name+".json")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			lastErr = err
			continue
		}
		successCount++
	}

	// Return error only if ALL rooms failed to cache
	if successCount == 0 && lastErr != nil {
		return fmt.Errorf("failed to cache any rooms: %w", lastErr)
	}
	return nil
}

func loadRoomsFromDir(dir string) []model.Room {
	entries, err := filepath.Glob(filepath.Join(dir, "*.jsonc"))
	if err != nil {
		return nil
	}

	var rooms []model.Room
	for _, path := range entries {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		// Strip JSONC comments (simple approach: just try parsing)
		var room model.Room
		if err := json.Unmarshal(data, &room); err != nil {
			// Try with comment stripping
			clean := stripJSONComments(data)
			if err := json.Unmarshal(clean, &room); err != nil {
				continue
			}
		}
		rooms = append(rooms, room)
	}
	return rooms
}

func loadRoomsFromCacheDir(cacheDir string) []model.Room {
	roomsDir := filepath.Join(cacheDir, "rooms")
	entries, err := filepath.Glob(filepath.Join(roomsDir, "*.json"))
	if err != nil {
		return nil
	}

	var rooms []model.Room
	for _, path := range entries {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var room model.Room
		if err := json.Unmarshal(data, &room); err != nil {
			continue
		}
		rooms = append(rooms, room)
	}
	return rooms
}

func loadCacheMeta(cacheDir string) (*CacheMeta, error) {
	path := filepath.Join(cacheDir, ".meta.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta CacheMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func writeCacheMeta(cacheDir string, meta CacheMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(cacheDir, ".meta.json")
	return os.WriteFile(path, data, 0o644)
}

func parseTTL(ttlStr string) time.Duration {
	if ttlStr == "" {
		return 24 * time.Hour // Default: 24 hours
	}
	d, err := time.ParseDuration(ttlStr)
	if err != nil {
		return 24 * time.Hour
	}
	return d
}

func stripJSONComments(data []byte) []byte {
	// Use the proper JSONC parser that handles strings correctly
	return jsonc.Clean(data)
}

func NamespacePath(neighborName, path string) string {
	return fmt.Sprintf("corridor://%s/%s", neighborName, strings.TrimPrefix(path, "/"))
}

func ParseNamespacedPath(path string) (neighborName, relativePath string, isCorridor bool) {
	if !strings.HasPrefix(path, "corridor://") {
		return "", path, false
	}
	trimmed := strings.TrimPrefix(path, "corridor://")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], true
	}
	return parts[0], "", true
}
