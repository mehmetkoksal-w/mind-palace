package corridor

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
	_ "modernc.org/sqlite"
)

// parseTimeOrZero parses RFC3339 time string, returning zero time on error.
// Used for database timestamps where zero time is a safe fallback.
func parseTimeOrZero(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// GlobalCorridor manages ~/.palace/corridors/ for cross-workspace learning.
type GlobalCorridor struct {
	db       *sql.DB
	basePath string
}

// PersonalLearning represents a learning in the personal corridor.
type PersonalLearning struct {
	ID              string    `json:"id"`
	OriginWorkspace string    `json:"originWorkspace"`
	Content         string    `json:"content"`
	Confidence      float64   `json:"confidence"`
	Source          string    `json:"source"`
	CreatedAt       time.Time `json:"createdAt"`
	LastUsed        time.Time `json:"lastUsed"`
	UseCount        int       `json:"useCount"`
	Tags            []string  `json:"tags"`
}

// LinkedWorkspace represents a linked workspace.
type LinkedWorkspace struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	AddedAt      time.Time `json:"addedAt"`
	LastAccessed time.Time `json:"lastAccessed,omitempty"`
}

// GlobalPath returns the path to the global palace directory (~/.palace).
func GlobalPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".palace"), nil
}

// EnsureGlobalLayout creates the global palace directory structure.
func EnsureGlobalLayout() (string, error) {
	basePath, err := GlobalPath()
	if err != nil {
		return "", err
	}

	dirs := []string{
		basePath,
		filepath.Join(basePath, "corridors"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return "", fmt.Errorf("create %s: %w", d, err)
		}
	}
	return basePath, nil
}

// OpenGlobal opens the global corridor database.
func OpenGlobal() (*GlobalCorridor, error) {
	basePath, err := EnsureGlobalLayout()
	if err != nil {
		return nil, err
	}

	dbPath := filepath.Join(basePath, "corridors", "personal.db")
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open corridor db: %w", err)
	}

	if err := initCorridorDB(db); err != nil {
		db.Close()
		return nil, err
	}

	return &GlobalCorridor{
		db:       db,
		basePath: basePath,
	}, nil
}

// Close closes the database connection.
func (g *GlobalCorridor) Close() error {
	if g.db != nil {
		return g.db.Close()
	}
	return nil
}

// AddPersonalLearning adds a learning to the personal corridor.
func (g *GlobalCorridor) AddPersonalLearning(l PersonalLearning) error {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if l.CreatedAt.IsZero() {
		l.CreatedAt = time.Now().UTC()
	}
	if l.LastUsed.IsZero() {
		l.LastUsed = time.Now().UTC()
	}

	tagsJSON, _ := json.Marshal(l.Tags)

	_, err := g.db.Exec(`
		INSERT INTO learnings (id, origin_workspace, content, confidence, source, created_at, last_used, use_count, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			content = excluded.content,
			confidence = excluded.confidence,
			last_used = ?,
			use_count = use_count + 1
	`, l.ID, l.OriginWorkspace, l.Content, l.Confidence, l.Source,
		l.CreatedAt.Format(time.RFC3339), l.LastUsed.Format(time.RFC3339), l.UseCount, string(tagsJSON),
		now)
	return err
}

// GetPersonalLearnings retrieves personal learnings, optionally filtered by query.
func (g *GlobalCorridor) GetPersonalLearnings(query string, limit int) ([]PersonalLearning, error) {
	if limit <= 0 {
		limit = 20
	}

	var rows *sql.Rows
	var err error

	if query == "" {
		rows, err = g.db.Query(`
			SELECT id, origin_workspace, content, confidence, source, created_at, last_used, use_count, tags
			FROM learnings
			ORDER BY confidence DESC, use_count DESC
			LIMIT ?
		`, limit)
	} else {
		rows, err = g.db.Query(`
			SELECT id, origin_workspace, content, confidence, source, created_at, last_used, use_count, tags
			FROM learnings
			WHERE content LIKE ?
			ORDER BY confidence DESC, use_count DESC
			LIMIT ?
		`, "%"+query+"%", limit)
	}
	if err != nil {
		return nil, fmt.Errorf("query personal learnings: %w", err)
	}
	defer rows.Close()

	return scanPersonalLearnings(rows)
}

// ReinforceLearning increases confidence for a learning.
func (g *GlobalCorridor) ReinforceLearning(id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := g.db.Exec(`
		UPDATE learnings
		SET confidence = MIN(1.0, confidence + 0.1),
		    use_count = use_count + 1,
		    last_used = ?
		WHERE id = ?
	`, now, id)
	return err
}

// DeleteLearning removes a learning from the personal corridor.
func (g *GlobalCorridor) DeleteLearning(id string) error {
	_, err := g.db.Exec(`DELETE FROM learnings WHERE id = ?`, id)
	return err
}

// Link connects a workspace to the global corridor.
func (g *GlobalCorridor) Link(name, localPath string) error {
	absPath, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// Verify the path exists and has a .palace directory
	palacePath := filepath.Join(absPath, ".palace")
	if _, err := os.Stat(palacePath); os.IsNotExist(err) {
		return fmt.Errorf("no .palace directory found at %s", absPath)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = g.db.Exec(`
		INSERT INTO links (name, path, added_at, last_accessed)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			path = excluded.path,
			last_accessed = excluded.last_accessed
	`, name, absPath, now, now)
	return err
}

// Unlink removes a workspace link.
func (g *GlobalCorridor) Unlink(name string) error {
	result, err := g.db.Exec(`DELETE FROM links WHERE name = ?`, name)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("no link named %q", name)
	}
	return nil
}

// GetLinks returns all linked workspaces.
func (g *GlobalCorridor) GetLinks() ([]LinkedWorkspace, error) {
	rows, err := g.db.Query(`
		SELECT name, path, added_at, last_accessed
		FROM links
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("query links: %w", err)
	}
	defer rows.Close()

	var links []LinkedWorkspace
	for rows.Next() {
		var l LinkedWorkspace
		var addedAt, lastAccessed sql.NullString
		if err := rows.Scan(&l.Name, &l.Path, &addedAt, &lastAccessed); err != nil {
			return nil, fmt.Errorf("scan link: %w", err)
		}
		if addedAt.Valid {
			l.AddedAt = parseTimeOrZero(addedAt.String)
		}
		if lastAccessed.Valid {
			l.LastAccessed = parseTimeOrZero(lastAccessed.String)
		}
		links = append(links, l)
	}
	return links, nil
}

// GetLinkedLearnings retrieves learnings from a specific linked workspace.
func (g *GlobalCorridor) GetLinkedLearnings(name string, limit int) ([]memory.Learning, error) {
	if limit <= 0 {
		limit = 20
	}

	// Get the link path
	var path string
	err := g.db.QueryRow(`SELECT path FROM links WHERE name = ?`, name).Scan(&path)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no link named %q", name)
	}
	if err != nil {
		return nil, fmt.Errorf("get link: %w", err)
	}

	// Update last accessed (non-critical, ok to fail)
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = g.db.Exec(`UPDATE links SET last_accessed = ? WHERE name = ?`, now, name)

	// Open the linked workspace memory
	memoryPath := filepath.Join(path, ".palace", "memory.db")
	if _, err := os.Stat(memoryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no memory.db at linked workspace %s", name)
	}

	mem, err := memory.Open(path) // Pass workspace root, not memory.db path
	if err != nil {
		return nil, fmt.Errorf("open linked memory: %w", err)
	}
	defer mem.Close()

	return mem.GetLearnings("palace", "", limit)
}

// GetAllLinkedLearnings retrieves learnings from all linked workspaces.
func (g *GlobalCorridor) GetAllLinkedLearnings(limit int) ([]memory.Learning, error) {
	links, err := g.GetLinks()
	if err != nil {
		return nil, err
	}

	var allLearnings []memory.Learning
	perLink := limit / max(len(links), 1)
	if perLink < 5 {
		perLink = 5
	}

	for _, link := range links {
		learnings, err := g.GetLinkedLearnings(link.Name, perLink)
		if err != nil {
			continue // Skip failed links
		}
		allLearnings = append(allLearnings, learnings...)
	}

	// Sort by confidence and limit
	if len(allLearnings) > limit {
		allLearnings = allLearnings[:limit]
	}
	return allLearnings, nil
}

// PromoteFromWorkspace promotes a learning from a workspace to the personal corridor.
func (g *GlobalCorridor) PromoteFromWorkspace(workspaceName string, l memory.Learning) error {
	pl := PersonalLearning{
		ID:              l.ID,
		OriginWorkspace: workspaceName,
		Content:         l.Content,
		Confidence:      l.Confidence,
		Source:          "promoted",
		CreatedAt:       l.CreatedAt,
		LastUsed:        time.Now().UTC(),
		UseCount:        l.UseCount,
		Tags:            []string{},
	}
	return g.AddPersonalLearning(pl)
}

// AutoPromote checks workspace learnings and promotes high-confidence ones.
func (g *GlobalCorridor) AutoPromote(workspaceName string, mem *memory.Memory) ([]PersonalLearning, error) {
	// Get high-confidence, frequently used learnings
	learnings, err := mem.GetLearnings("palace", "", 100)
	if err != nil {
		return nil, err
	}

	var promoted []PersonalLearning
	for _, l := range learnings {
		// Promote if confidence >= 0.8 and use_count >= 3
		if l.Confidence >= 0.8 && l.UseCount >= 3 {
			// Check if already promoted
			var exists int
			err := g.db.QueryRow(`SELECT 1 FROM learnings WHERE id = ?`, l.ID).Scan(&exists)
			if err != nil && err != sql.ErrNoRows {
				// DB error - skip this learning
				continue
			}
			if exists == 1 {
				continue
			}

			if err := g.PromoteFromWorkspace(workspaceName, l); err != nil {
				continue
			}
			promoted = append(promoted, PersonalLearning{
				ID:              l.ID,
				OriginWorkspace: workspaceName,
				Content:         l.Content,
				Confidence:      l.Confidence,
			})
		}
	}
	return promoted, nil
}

// Stats returns statistics about the personal corridor.
func (g *GlobalCorridor) Stats() (map[string]any, error) {
	var learningCount int
	g.db.QueryRow(`SELECT COUNT(*) FROM learnings`).Scan(&learningCount)

	var linkCount int
	g.db.QueryRow(`SELECT COUNT(*) FROM links`).Scan(&linkCount)

	var avgConfidence float64
	g.db.QueryRow(`SELECT AVG(confidence) FROM learnings`).Scan(&avgConfidence)

	return map[string]any{
		"learningCount":     learningCount,
		"linkedWorkspaces":  linkCount,
		"averageConfidence": avgConfidence,
	}, nil
}

func scanPersonalLearnings(rows *sql.Rows) ([]PersonalLearning, error) {
	var learnings []PersonalLearning
	for rows.Next() {
		var l PersonalLearning
		var createdAt, lastUsed, tagsJSON string
		if err := rows.Scan(&l.ID, &l.OriginWorkspace, &l.Content, &l.Confidence, &l.Source, &createdAt, &lastUsed, &l.UseCount, &tagsJSON); err != nil {
			return nil, fmt.Errorf("scan learning: %w", err)
		}
		l.CreatedAt = parseTimeOrZero(createdAt)
		l.LastUsed = parseTimeOrZero(lastUsed)
		json.Unmarshal([]byte(tagsJSON), &l.Tags)
		learnings = append(learnings, l)
	}
	return learnings, nil
}

// ValidateLinks checks all linked workspaces exist and returns stale ones.
func (g *GlobalCorridor) ValidateLinks() ([]string, error) {
	var stale []string
	links, err := g.GetLinks()
	if err != nil {
		return nil, err
	}

	for _, link := range links {
		memPath := filepath.Join(link.Path, ".palace", "memory.db")
		if _, err := os.Stat(memPath); os.IsNotExist(err) {
			stale = append(stale, link.Name)
		}
	}
	return stale, nil
}

// PruneStaleLinks removes links to non-existent workspaces.
func (g *GlobalCorridor) PruneStaleLinks() ([]string, error) {
	stale, err := g.ValidateLinks()
	if err != nil {
		return nil, err
	}
	for _, name := range stale {
		g.Unlink(name)
	}
	return stale, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
