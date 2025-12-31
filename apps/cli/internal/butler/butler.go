package butler

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/llm"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
)

// Butler provides high-level access to the Mind Palace index and memory.
type Butler struct {
	db          *sql.DB
	root        string
	rooms       map[string]model.Room // cached room manifests
	entryPoints map[string]string     // path -> room name for entry points
	config      *config.PalaceConfig
	memory      *memory.Memory // session memory (optional, may be nil)
}

// New creates a new Butler instance.
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

	// Initialize session memory (non-fatal if fails)
	mem, err := memory.Open(root)
	if err == nil {
		b.memory = mem

		// Initialize embedding pipeline if configured
		if embedder := b.GetEmbedder(); embedder != nil {
			pipeline := memory.NewEmbeddingPipeline(mem, embedder, 2) // 2 workers
			mem.SetEmbeddingPipeline(pipeline)
			pipeline.Start()
		}
	}

	return b, nil
}

// Close closes the Butler's resources.
func (b *Butler) Close() error {
	if b.memory != nil {
		return b.memory.Close()
	}
	return nil
}

// GetWorkspaceName returns the name of the workspace (base directory name).
func (b *Butler) GetWorkspaceName() string {
	if b.root == "" {
		return "Unknown"
	}
	return filepath.Base(b.root)
}

// Memory returns the session memory (may be nil).
func (b *Butler) Memory() *memory.Memory {
	return b.memory
}

// GetEmbedder returns an embedder based on the palace configuration.
// Returns nil if embeddings are not configured or disabled.
func (b *Butler) GetEmbedder() memory.Embedder {
	if b.config == nil {
		return nil
	}

	embCfg := memory.EmbeddingConfig{
		Backend: b.config.EmbeddingBackend,
		Model:   b.config.EmbeddingModel,
		URL:     b.config.EmbeddingURL,
		APIKey:  b.config.EmbeddingAPIKey,
	}

	embedder, err := memory.NewEmbedder(embCfg)
	if err != nil {
		return nil
	}

	return embedder
}

// GetLLMClient returns an LLM client based on the palace configuration.
// Returns nil and an error if LLM is not configured or disabled.
func (b *Butler) GetLLMClient() (llm.Client, error) {
	if b.config == nil {
		return nil, llm.ErrNotConfigured
	}

	cfg := llm.Config{
		Backend: b.config.LLMBackend,
		Model:   b.config.LLMModel,
		URL:     b.config.LLMURL,
		APIKey:  b.config.LLMAPIKey,
	}

	return llm.NewClient(cfg)
}

// Config returns the palace configuration.
func (b *Butler) Config() *config.PalaceConfig {
	return b.config
}

// loadRooms loads room manifests from the .palace/rooms directory.
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

// ListRooms returns all configured rooms sorted by name.
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

// ReadFile reads indexed content for a file path.
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

// ReadRoom retrieves a room manifest by name.
func (b *Butler) ReadRoom(name string) (*model.Room, error) {
	room, ok := b.rooms[name]
	if !ok {
		return nil, fmt.Errorf("room not found: %s", name)
	}
	return &room, nil
}
