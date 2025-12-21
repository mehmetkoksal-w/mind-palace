package butler

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/koksalmehmet/mind-palace/internal/config"
	"github.com/koksalmehmet/mind-palace/internal/model"
)

type SearchResult struct {
	Path       string  `json:"path"`
	Room       string  `json:"room,omitempty"`
	ChunkIndex int     `json:"chunkIndex"`
	StartLine  int     `json:"startLine"`
	EndLine    int     `json:"endLine"`
	Snippet    string  `json:"snippet"`
	Score      float64 `json:"score"`
	IsEntry    bool    `json:"isEntry,omitempty"`
}

type GroupedResults struct {
	Room    string         `json:"room"`
	Summary string         `json:"summary,omitempty"`
	Results []SearchResult `json:"results"`
}

type SearchOptions struct {
	Limit      int    // Maximum results (default 20)
	RoomFilter string // Optional: filter to specific room
}

type Butler struct {
	db          *sql.DB
	root        string
	rooms       map[string]model.Room // cached room manifests
	entryPoints map[string]string     // path -> room name for entry points
	config      *config.PalaceConfig
}

func New(db *sql.DB, root string) (*Butler, error) {
	b := &Butler{
		db:          db,
		root:        root,
		rooms:       make(map[string]model.Room),
		entryPoints: make(map[string]string),
	}

	cfg, err := config.LoadPalaceConfig(root)
	if err != nil {
		// Not fatal - use defaults
		b.config = nil
	} else {
		b.config = cfg
	}

	if err := b.loadRooms(); err != nil {
		return nil, fmt.Errorf("load rooms: %w", err)
	}

	return b, nil
}

func (b *Butler) loadRooms() error {
	roomsDir := filepath.Join(b.root, ".palace", "rooms")
	entries, err := filepath.Glob(filepath.Join(roomsDir, "*.jsonc"))
	if err != nil {
		return err
	}

	for _, path := range entries {
		var room model.Room
		if err := decodeJSONCFile(path, &room); err != nil {
			continue // Skip invalid room files
		}
		b.rooms[room.Name] = room
		for _, ep := range room.EntryPoints {
			b.entryPoints[ep] = room.Name
		}
	}
	return nil
}

func (b *Butler) Search(query string, opts SearchOptions) ([]GroupedResults, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}

	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	ftsQuery := preprocessQuery(query)

	rows, err := b.db.Query(`
		SELECT 
			c.path,
			c.chunk_index,
			c.start_line,
			c.end_line,
			c.content,
			bm25(chunks_fts) as base_score
		FROM chunks_fts
		JOIN chunks c ON c.path = chunks_fts.path AND c.chunk_index = chunks_fts.chunk_index
		WHERE chunks_fts MATCH ?
		ORDER BY base_score
		LIMIT ?;
	`, ftsQuery, opts.Limit*3) // Over-fetch for re-ranking

	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var baseScore float64
		if err := rows.Scan(&r.Path, &r.ChunkIndex, &r.StartLine, &r.EndLine, &r.Snippet, &baseScore); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}

		r.Score = b.calculateScore(baseScore, r.Path, query)

		if roomName, ok := b.entryPoints[r.Path]; ok {
			r.Room = roomName
			r.IsEntry = true
		} else {
			r.Room = b.inferRoom(r.Path)
		}

		if opts.RoomFilter != "" && r.Room != opts.RoomFilter {
			continue
		}

		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return b.groupByRoom(results), nil
}

func (b *Butler) calculateScore(baseScore float64, path, query string) float64 {
	score := -baseScore

	if _, isEntry := b.entryPoints[path]; isEntry {
		score *= 3.0
	}

	queryLower := strings.ToLower(query)
	pathLower := strings.ToLower(path)
	if strings.Contains(pathLower, queryLower) {
		score *= 2.5
	} else {
		for _, word := range strings.Fields(query) {
			if len(word) > 2 && strings.Contains(pathLower, strings.ToLower(word)) {
				score *= 1.5
				break
			}
		}
	}

	ext := strings.ToLower(filepath.Ext(path))
	codeExts := map[string]bool{
		".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true,
		".py": true, ".rs": true, ".java": true, ".c": true, ".cpp": true,
		".rb": true, ".swift": true, ".kt": true,
	}
	if codeExts[ext] {
		score *= 1.2
	}

	return score
}

func (b *Butler) inferRoom(path string) string {
	// Check if path prefix matches any room's entry points' directories
	for name, room := range b.rooms {
		for _, ep := range room.EntryPoints {
			epDir := filepath.Dir(ep)
			if epDir != "." && strings.HasPrefix(path, epDir) {
				return name
			}
		}
	}

	// Default room from config
	if b.config != nil && b.config.DefaultRoom != "" {
		return b.config.DefaultRoom
	}

	return ""
}

func (b *Butler) groupByRoom(results []SearchResult) []GroupedResults {
	groups := make(map[string]*GroupedResults)
	order := []string{}

	for _, r := range results {
		roomName := r.Room
		if roomName == "" {
			roomName = "_ungrouped"
		}

		if _, exists := groups[roomName]; !exists {
			summary := ""
			if room, ok := b.rooms[roomName]; ok {
				summary = room.Summary
			}
			groups[roomName] = &GroupedResults{
				Room:    roomName,
				Summary: summary,
				Results: []SearchResult{},
			}
			order = append(order, roomName)
		}
		groups[roomName].Results = append(groups[roomName].Results, r)
	}

	// Convert to slice maintaining discovery order
	var grouped []GroupedResults
	for _, roomName := range order {
		grouped = append(grouped, *groups[roomName])
	}

	return grouped
}

func (b *Butler) ListRooms() []model.Room {
	rooms := make([]model.Room, 0, len(b.rooms))
	for _, room := range b.rooms {
		rooms = append(rooms, room)
	}
	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].Name < rooms[j].Name
	})
	return rooms
}

func (b *Butler) ReadFile(path string) (string, error) {
	// First, try to read from the chunks table for indexed content
	rows, err := b.db.Query(
		`SELECT content FROM chunks WHERE path = ? ORDER BY chunk_index ASC;`,
		path,
	)
	if err != nil {
		return "", fmt.Errorf("query chunks for %s: %w", path, err)
	}
	defer rows.Close()

	var parts []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return "", err
		}
		parts = append(parts, content)
	}

	if len(parts) == 0 {
		return "", fmt.Errorf("file not found in index: %s", path)
	}

	return strings.Join(parts, "\n"), nil
}

func (b *Butler) ReadRoom(name string) (*model.Room, error) {
	room, ok := b.rooms[name]
	if !ok {
		return nil, fmt.Errorf("room not found: %s", name)
	}
	return &room, nil
}

func preprocessQuery(query string) string {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return ""
	}

	isCodeLike := strings.ContainsAny(trimmed, ".:_-@#$&()[]{}") ||
		strings.Contains(trimmed, "::") ||
		strings.Contains(trimmed, "->")

	if isCodeLike {
		// Exact phrase search with quote escaping
		escaped := strings.ReplaceAll(trimmed, "\"", "\"\"")
		return fmt.Sprintf("\"%s\"", escaped)
	}

	words := strings.Fields(trimmed)
	if len(words) == 0 {
		return ""
	}

	var terms []string
	for _, word := range words {
		if len(word) < 2 {
			continue
		}
		// Escape and add prefix operator
		escaped := strings.ReplaceAll(word, "\"", "\"\"")
		terms = append(terms, fmt.Sprintf("\"%s\"*", escaped))
	}

	if len(terms) == 0 {
		escaped := strings.ReplaceAll(trimmed, "\"", "\"\"")
		return fmt.Sprintf("\"%s\"", escaped)
	}

	return strings.Join(terms, " OR ")
}

func decodeJSONCFile(path string, v interface{}) error {
	// Use the jsonc package from the project
	return jsonCDecode(path, v)
}

var jsonCDecode func(path string, v interface{}) error

func SetJSONCDecoder(fn func(path string, v interface{}) error) {
	jsonCDecode = fn
}
