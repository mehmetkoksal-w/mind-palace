package memory

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Link represents a relationship between records (ideas, decisions, learnings, code).
type Link struct {
	ID          string    `json:"id"`          // Prefix: "l_"
	SourceID    string    `json:"sourceId"`    // ID of source record
	SourceKind  string    `json:"sourceKind"`  // "idea", "decision", "learning"
	TargetID    string    `json:"targetId"`    // ID or path (e.g., "auth/jwt.go:15-45")
	TargetKind  string    `json:"targetKind"`  // "idea", "decision", "learning", "code", "url"
	Relation    string    `json:"relation"`    // Relation type
	TargetMtime time.Time `json:"targetMtime"` // For code links: file mtime at link creation
	IsStale     bool      `json:"isStale"`     // True if file changed since link
	CreatedAt   time.Time `json:"createdAt"`
}

// Relation types
const (
	RelationSupports    = "supports"    // Evidence for
	RelationContradicts = "contradicts" // Conflicts with
	RelationImplements  = "implements"  // Code that implements
	RelationSupersedes  = "supersedes"  // Replaces old decision
	RelationInspiredBy  = "inspired_by" // Came from this
	RelationRelated     = "related"     // General relationship
)

// Target kinds
const (
	TargetKindIdea     = "idea"
	TargetKindDecision = "decision"
	TargetKindLearning = "learning"
	TargetKindCode     = "code"
	TargetKindURL      = "url"
)

// ValidRelations lists all valid relation types.
var ValidRelations = []string{
	RelationSupports,
	RelationContradicts,
	RelationImplements,
	RelationSupersedes,
	RelationInspiredBy,
	RelationRelated,
}

// CodeTarget represents a parsed code reference (e.g., "auth/jwt.go:15-45").
type CodeTarget struct {
	FilePath  string
	StartLine int
	EndLine   int
}

// AddLink creates a new link between records.
func (m *Memory) AddLink(link Link) (string, error) {
	if link.SourceID == "" {
		return "", fmt.Errorf("source_id is required")
	}
	if link.SourceKind == "" {
		return "", fmt.Errorf("source_kind is required")
	}
	if link.TargetID == "" {
		return "", fmt.Errorf("target_id is required")
	}
	if link.TargetKind == "" {
		return "", fmt.Errorf("target_kind is required")
	}
	if link.Relation == "" {
		return "", fmt.Errorf("relation is required")
	}

	// Validate relation type
	if !isValidRelation(link.Relation) {
		return "", fmt.Errorf("invalid relation %q; valid relations: %v", link.Relation, ValidRelations)
	}

	// Generate ID
	link.ID = "l_" + uuid.New().String()[:8]
	link.CreatedAt = time.Now().UTC()

	// Handle target mtime for code links
	targetMtime := ""
	if link.TargetMtime != (time.Time{}) {
		targetMtime = link.TargetMtime.Format(time.RFC3339)
	}

	_, err := m.db.Exec(`
		INSERT INTO links (id, source_id, source_kind, target_id, target_kind, relation, target_mtime, is_stale, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		link.ID, link.SourceID, link.SourceKind, link.TargetID, link.TargetKind,
		link.Relation, targetMtime, boolToInt(link.IsStale), link.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return "", fmt.Errorf("insert link: %w", err)
	}

	return link.ID, nil
}

// GetLink retrieves a link by ID.
func (m *Memory) GetLink(id string) (*Link, error) {
	row := m.db.QueryRow(`
		SELECT id, source_id, source_kind, target_id, target_kind, relation, target_mtime, is_stale, created_at
		FROM links WHERE id = ?`, id)

	link, err := scanLink(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("link not found: %s", id)
		}
		return nil, err
	}
	return link, nil
}

// GetLinksForSource retrieves all links where the given ID is the source.
func (m *Memory) GetLinksForSource(sourceID string) ([]Link, error) {
	rows, err := m.db.Query(`
		SELECT id, source_id, source_kind, target_id, target_kind, relation, target_mtime, is_stale, created_at
		FROM links WHERE source_id = ? ORDER BY created_at DESC`, sourceID)
	if err != nil {
		return nil, fmt.Errorf("query links: %w", err)
	}
	defer rows.Close()

	return scanLinks(rows)
}

// GetLinksForTarget retrieves all links where the given ID is the target.
func (m *Memory) GetLinksForTarget(targetID string) ([]Link, error) {
	rows, err := m.db.Query(`
		SELECT id, source_id, source_kind, target_id, target_kind, relation, target_mtime, is_stale, created_at
		FROM links WHERE target_id = ? ORDER BY created_at DESC`, targetID)
	if err != nil {
		return nil, fmt.Errorf("query links: %w", err)
	}
	defer rows.Close()

	return scanLinks(rows)
}

// GetAllLinksFor retrieves all links where the given ID is either source or target.
func (m *Memory) GetAllLinksFor(id string) ([]Link, error) {
	rows, err := m.db.Query(`
		SELECT id, source_id, source_kind, target_id, target_kind, relation, target_mtime, is_stale, created_at
		FROM links WHERE source_id = ? OR target_id = ? ORDER BY created_at DESC`, id, id)
	if err != nil {
		return nil, fmt.Errorf("query links: %w", err)
	}
	defer rows.Close()

	return scanLinks(rows)
}

// GetLinksByRelation retrieves all links with a specific relation type.
func (m *Memory) GetLinksByRelation(relation string, limit int) ([]Link, error) {
	rows, err := m.db.Query(`
		SELECT id, source_id, source_kind, target_id, target_kind, relation, target_mtime, is_stale, created_at
		FROM links WHERE relation = ? ORDER BY created_at DESC LIMIT ?`, relation, limit)
	if err != nil {
		return nil, fmt.Errorf("query links: %w", err)
	}
	defer rows.Close()

	return scanLinks(rows)
}

// DeleteLink deletes a link by ID.
func (m *Memory) DeleteLink(id string) error {
	result, err := m.db.Exec(`DELETE FROM links WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete link: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("link not found: %s", id)
	}
	return nil
}

// DeleteLinksForRecord deletes all links where the given ID is source or target.
// This is called when a record is deleted (ON DELETE CASCADE alternative for app-level).
func (m *Memory) DeleteLinksForRecord(recordID string) error {
	_, err := m.db.Exec(`DELETE FROM links WHERE source_id = ? OR target_id = ?`, recordID, recordID)
	return err
}

// MarkLinkStale marks a link as stale (code file has changed).
func (m *Memory) MarkLinkStale(id string, isStale bool) error {
	_, err := m.db.Exec(`UPDATE links SET is_stale = ? WHERE id = ?`, boolToInt(isStale), id)
	return err
}

// GetStaleLinks retrieves all links marked as stale.
func (m *Memory) GetStaleLinks() ([]Link, error) {
	rows, err := m.db.Query(`
		SELECT id, source_id, source_kind, target_id, target_kind, relation, target_mtime, is_stale, created_at
		FROM links WHERE is_stale = 1 ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query stale links: %w", err)
	}
	defer rows.Close()

	return scanLinks(rows)
}

// CheckAndUpdateStaleness checks all code links and marks them as stale if file changed.
func (m *Memory) CheckAndUpdateStaleness(rootPath string) (int, error) {
	links, err := m.GetLinksByRelation(RelationImplements, 1000)
	if err != nil {
		return 0, err
	}

	staleCount := 0
	for _, link := range links {
		if link.TargetKind != TargetKindCode {
			continue
		}

		// Parse the code target
		target, err := ParseCodeTarget(link.TargetID)
		if err != nil {
			continue
		}

		// Get current file mtime
		fullPath := filepath.Join(rootPath, target.FilePath)
		info, err := os.Stat(fullPath)
		if err != nil {
			// File doesn't exist anymore - mark as stale
			if !link.IsStale {
				m.MarkLinkStale(link.ID, true)
				staleCount++
			}
			continue
		}

		// Check if file was modified after link creation
		if info.ModTime().After(link.TargetMtime) && !link.IsStale {
			m.MarkLinkStale(link.ID, true)
			staleCount++
		}
	}

	return staleCount, nil
}

// ValidateCodeTarget validates that a code target exists and line range is valid.
// Returns the file's mtime if valid.
func ValidateCodeTarget(rootPath string, target string) (*CodeTarget, time.Time, error) {
	parsed, err := ParseCodeTarget(target)
	if err != nil {
		return nil, time.Time{}, err
	}

	fullPath := filepath.Join(rootPath, parsed.FilePath)

	// Check file exists
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, time.Time{}, fmt.Errorf("file does not exist: %s", parsed.FilePath)
		}
		return nil, time.Time{}, fmt.Errorf("stat file: %w", err)
	}

	if info.IsDir() {
		return nil, time.Time{}, fmt.Errorf("path is a directory: %s", parsed.FilePath)
	}

	// If line range specified, validate it
	if parsed.StartLine > 0 || parsed.EndLine > 0 {
		lineCount, err := countFileLines(fullPath)
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("count lines: %w", err)
		}

		if parsed.StartLine > lineCount {
			return nil, time.Time{}, fmt.Errorf("start line %d exceeds file length %d", parsed.StartLine, lineCount)
		}
		if parsed.EndLine > 0 && parsed.EndLine > lineCount {
			return nil, time.Time{}, fmt.Errorf("end line %d exceeds file length %d", parsed.EndLine, lineCount)
		}
		if parsed.EndLine > 0 && parsed.StartLine > parsed.EndLine {
			return nil, time.Time{}, fmt.Errorf("start line %d is after end line %d", parsed.StartLine, parsed.EndLine)
		}
	}

	return parsed, info.ModTime(), nil
}

// ParseCodeTarget parses a code reference like "auth/jwt.go:15-45" or "auth/jwt.go:15".
func ParseCodeTarget(target string) (*CodeTarget, error) {
	result := &CodeTarget{}

	// Check for line range pattern: file:start-end or file:line
	linePattern := regexp.MustCompile(`^(.+):(\d+)(?:-(\d+))?$`)
	matches := linePattern.FindStringSubmatch(target)

	if matches != nil {
		result.FilePath = matches[1]
		result.StartLine, _ = strconv.Atoi(matches[2])
		if matches[3] != "" {
			result.EndLine, _ = strconv.Atoi(matches[3])
		} else {
			result.EndLine = result.StartLine // Single line
		}
	} else {
		// Just a file path
		result.FilePath = target
	}

	// Normalize path
	result.FilePath = filepath.Clean(result.FilePath)

	return result, nil
}

// Helper functions

func isValidRelation(relation string) bool {
	for _, r := range ValidRelations {
		if r == relation {
			return true
		}
	}
	return false
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func scanLink(row *sql.Row) (*Link, error) {
	var link Link
	var targetMtime, createdAt string
	var isStale int

	err := row.Scan(&link.ID, &link.SourceID, &link.SourceKind, &link.TargetID, &link.TargetKind,
		&link.Relation, &targetMtime, &isStale, &createdAt)
	if err != nil {
		return nil, err
	}

	link.IsStale = isStale == 1
	if targetMtime != "" {
		link.TargetMtime = parseTimeOrZero(targetMtime)
	}
	link.CreatedAt = parseTimeOrZero(createdAt)

	return &link, nil
}

func scanLinks(rows *sql.Rows) ([]Link, error) {
	var links []Link
	for rows.Next() {
		var link Link
		var targetMtime, createdAt string
		var isStale int

		err := rows.Scan(&link.ID, &link.SourceID, &link.SourceKind, &link.TargetID, &link.TargetKind,
			&link.Relation, &targetMtime, &isStale, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("scan link: %w", err)
		}

		link.IsStale = isStale == 1
		if targetMtime != "" {
			link.TargetMtime = parseTimeOrZero(targetMtime)
		}
		link.CreatedAt = parseTimeOrZero(createdAt)

		links = append(links, link)
	}
	return links, nil
}

func countFileLines(path string) (int, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	// Count newlines + 1 for last line (if file is not empty)
	if len(content) == 0 {
		return 0, nil
	}

	lines := strings.Count(string(content), "\n")
	// If file doesn't end with newline, add 1 for the last line
	if content[len(content)-1] != '\n' {
		lines++
	}
	return lines, nil
}
