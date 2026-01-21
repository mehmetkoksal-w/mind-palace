package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Pattern represents a detected code pattern.
type Pattern struct {
	ID          string    `json:"id"`
	Category    string    `json:"category"`
	Subcategory string    `json:"subcategory"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	DetectorID  string    `json:"detector_id"`

	// Confidence scoring
	Confidence       float64 `json:"confidence"`
	FrequencyScore   float64 `json:"frequency_score"`
	ConsistencyScore float64 `json:"consistency_score"`
	SpreadScore      float64 `json:"spread_score"`
	AgeScore         float64 `json:"age_score"`

	// Status and governance
	Status     string `json:"status"`     // discovered, approved, ignored
	Authority  string `json:"authority"`  // proposed, approved, legacy_approved
	LearningID string `json:"learning_id"`

	// Metadata
	Metadata  map[string]any `json:"metadata"`
	FirstSeen time.Time      `json:"first_seen"`
	LastSeen  time.Time      `json:"last_seen"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// PatternLocation represents where a pattern is found in code.
type PatternLocation struct {
	ID            string    `json:"id"`
	PatternID     string    `json:"pattern_id"`
	FilePath      string    `json:"file_path"`
	LineStart     int       `json:"line_start"`
	LineEnd       int       `json:"line_end"`
	Snippet       string    `json:"snippet"`
	IsOutlier     bool      `json:"is_outlier"`
	OutlierReason string    `json:"outlier_reason"`
	CreatedAt     time.Time `json:"created_at"`
}

// PatternFilters contains filters for listing patterns.
type PatternFilters struct {
	Category      string
	Subcategory   string
	Status        string
	DetectorID    string
	MinConfidence float64
	FilePath      string
	Limit         int
	Offset        int
}

// AddPattern creates a new pattern in the database.
func (m *Memory) AddPattern(p Pattern) (string, error) {
	if p.ID == "" {
		p.ID = "pat_" + uuid.New().String()[:8]
	}

	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = now
	}
	if p.FirstSeen.IsZero() {
		p.FirstSeen = now
	}
	if p.LastSeen.IsZero() {
		p.LastSeen = now
	}
	if p.Status == "" {
		p.Status = "discovered"
	}
	if p.Authority == "" {
		p.Authority = "proposed"
	}

	metadataJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return "", fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		INSERT INTO patterns (
			id, category, subcategory, name, description, detector_id,
			confidence, frequency_score, consistency_score, spread_score, age_score,
			status, authority, learning_id, metadata,
			first_seen, last_seen, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = m.db.ExecContext(context.Background(), query,
		p.ID, p.Category, p.Subcategory, p.Name, p.Description, p.DetectorID,
		p.Confidence, p.FrequencyScore, p.ConsistencyScore, p.SpreadScore, p.AgeScore,
		p.Status, p.Authority, p.LearningID, string(metadataJSON),
		p.FirstSeen.Format(time.RFC3339), p.LastSeen.Format(time.RFC3339),
		p.CreatedAt.Format(time.RFC3339), p.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return "", fmt.Errorf("insert pattern: %w", err)
	}

	return p.ID, nil
}

// GetPattern retrieves a pattern by ID.
func (m *Memory) GetPattern(id string) (*Pattern, error) {
	query := `
		SELECT id, category, subcategory, name, description, detector_id,
			confidence, frequency_score, consistency_score, spread_score, age_score,
			status, authority, learning_id, metadata,
			first_seen, last_seen, created_at, updated_at
		FROM patterns
		WHERE id = ?
	`

	row := m.db.QueryRowContext(context.Background(), query, id)
	return scanPattern(row)
}

// GetPatterns retrieves patterns with optional filters.
func (m *Memory) GetPatterns(filters PatternFilters) ([]Pattern, error) {
	var conditions []string
	var args []any

	if filters.Category != "" {
		conditions = append(conditions, "category = ?")
		args = append(args, filters.Category)
	}
	if filters.Subcategory != "" {
		conditions = append(conditions, "subcategory = ?")
		args = append(args, filters.Subcategory)
	}
	if filters.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, filters.Status)
	}
	if filters.DetectorID != "" {
		conditions = append(conditions, "detector_id = ?")
		args = append(args, filters.DetectorID)
	}
	if filters.MinConfidence > 0 {
		conditions = append(conditions, "confidence >= ?")
		args = append(args, filters.MinConfidence)
	}

	query := `
		SELECT id, category, subcategory, name, description, detector_id,
			confidence, frequency_score, consistency_score, spread_score, age_score,
			status, authority, learning_id, metadata,
			first_seen, last_seen, created_at, updated_at
		FROM patterns
	`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY confidence DESC, created_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filters.Limit)
		if filters.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", filters.Offset)
		}
	}

	rows, err := m.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("query patterns: %w", err)
	}
	defer rows.Close()

	var patterns []Pattern
	for rows.Next() {
		p, err := scanPatternRows(rows)
		if err != nil {
			return nil, fmt.Errorf("scan pattern: %w", err)
		}
		patterns = append(patterns, *p)
	}

	return patterns, rows.Err()
}

// UpdatePattern updates an existing pattern.
func (m *Memory) UpdatePattern(p Pattern) error {
	p.UpdatedAt = time.Now().UTC()

	metadataJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		UPDATE patterns SET
			category = ?, subcategory = ?, name = ?, description = ?, detector_id = ?,
			confidence = ?, frequency_score = ?, consistency_score = ?, spread_score = ?, age_score = ?,
			status = ?, authority = ?, learning_id = ?, metadata = ?,
			first_seen = ?, last_seen = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = m.db.ExecContext(context.Background(), query,
		p.Category, p.Subcategory, p.Name, p.Description, p.DetectorID,
		p.Confidence, p.FrequencyScore, p.ConsistencyScore, p.SpreadScore, p.AgeScore,
		p.Status, p.Authority, p.LearningID, string(metadataJSON),
		p.FirstSeen.Format(time.RFC3339), p.LastSeen.Format(time.RFC3339),
		p.UpdatedAt.Format(time.RFC3339),
		p.ID,
	)
	return err
}

// UpdatePatternStatus updates the status of a pattern.
func (m *Memory) UpdatePatternStatus(id, status string) error {
	query := `UPDATE patterns SET status = ?, updated_at = ? WHERE id = ?`
	_, err := m.db.ExecContext(context.Background(), query,
		status, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

// ApprovePattern approves a pattern and optionally links it to a learning.
func (m *Memory) ApprovePattern(id, learningID string) error {
	query := `UPDATE patterns SET status = 'approved', authority = 'approved', learning_id = ?, updated_at = ? WHERE id = ?`
	_, err := m.db.ExecContext(context.Background(), query,
		learningID, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

// ApprovePatternWithLearning approves a pattern and creates a linked learning.
// This is the main governance integration point - patterns become enforceable learnings.
func (m *Memory) ApprovePatternWithLearning(patternID string) (string, error) {
	// Get the pattern
	pattern, err := m.GetPattern(patternID)
	if err != nil {
		return "", fmt.Errorf("get pattern: %w", err)
	}

	// Create a learning from the pattern
	learning := Learning{
		Scope:      "palace", // Patterns are palace-wide by default
		ScopePath:  "",
		Content:    formatPatternAsLearning(pattern),
		Confidence: pattern.Confidence,
		Source:     "pattern",
		Authority:  string(AuthorityApproved),
	}

	learningID, err := m.AddLearning(learning)
	if err != nil {
		return "", fmt.Errorf("create learning: %w", err)
	}

	// Link the pattern to the learning
	if err := m.ApprovePattern(patternID, learningID); err != nil {
		return "", fmt.Errorf("approve pattern: %w", err)
	}

	return learningID, nil
}

// formatPatternAsLearning converts pattern details into learning content.
func formatPatternAsLearning(p *Pattern) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("[%s] %s: %s", p.Category, p.Name, p.Description))

	// Add metadata hints if available
	if len(p.Metadata) > 0 {
		sb.WriteString(" (")
		first := true
		for k, v := range p.Metadata {
			if !first {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%s: %v", k, v))
			first = false
			// Limit metadata in learning content
			if sb.Len() > 500 {
				sb.WriteString("...")
				break
			}
		}
		sb.WriteString(")")
	}

	return sb.String()
}

// IgnorePattern marks a pattern as ignored.
func (m *Memory) IgnorePattern(id string) error {
	return m.UpdatePatternStatus(id, "ignored")
}

// DeletePattern removes a pattern and its locations.
func (m *Memory) DeletePattern(id string) error {
	// Locations are deleted via CASCADE
	_, err := m.db.ExecContext(context.Background(), "DELETE FROM patterns WHERE id = ?", id)
	return err
}

// SearchPatterns performs full-text search on patterns.
func (m *Memory) SearchPatterns(query string, limit int) ([]Pattern, error) {
	if limit <= 0 {
		limit = 50
	}

	sqlQuery := `
		SELECT p.id, p.category, p.subcategory, p.name, p.description, p.detector_id,
			p.confidence, p.frequency_score, p.consistency_score, p.spread_score, p.age_score,
			p.status, p.authority, p.learning_id, p.metadata,
			p.first_seen, p.last_seen, p.created_at, p.updated_at
		FROM patterns p
		JOIN patterns_fts fts ON p.rowid = fts.rowid
		WHERE patterns_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`

	rows, err := m.db.QueryContext(context.Background(), sqlQuery, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search patterns: %w", err)
	}
	defer rows.Close()

	var patterns []Pattern
	for rows.Next() {
		p, err := scanPatternRows(rows)
		if err != nil {
			return nil, fmt.Errorf("scan pattern: %w", err)
		}
		patterns = append(patterns, *p)
	}

	return patterns, rows.Err()
}

// AddPatternLocation adds a location where a pattern is found.
func (m *Memory) AddPatternLocation(loc PatternLocation) (string, error) {
	if loc.ID == "" {
		loc.ID = "ploc_" + uuid.New().String()[:8]
	}
	if loc.CreatedAt.IsZero() {
		loc.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO pattern_locations (
			id, pattern_id, file_path, line_start, line_end,
			snippet, is_outlier, outlier_reason, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	isOutlier := 0
	if loc.IsOutlier {
		isOutlier = 1
	}

	_, err := m.db.ExecContext(context.Background(), query,
		loc.ID, loc.PatternID, loc.FilePath, loc.LineStart, loc.LineEnd,
		loc.Snippet, isOutlier, loc.OutlierReason, loc.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return "", fmt.Errorf("insert pattern location: %w", err)
	}

	return loc.ID, nil
}

// GetPatternLocations retrieves all locations for a pattern.
func (m *Memory) GetPatternLocations(patternID string) ([]PatternLocation, error) {
	query := `
		SELECT id, pattern_id, file_path, line_start, line_end,
			snippet, is_outlier, outlier_reason, created_at
		FROM pattern_locations
		WHERE pattern_id = ?
		ORDER BY file_path, line_start
	`

	rows, err := m.db.QueryContext(context.Background(), query, patternID)
	if err != nil {
		return nil, fmt.Errorf("query pattern locations: %w", err)
	}
	defer rows.Close()

	var locations []PatternLocation
	for rows.Next() {
		var loc PatternLocation
		var isOutlier int
		var createdAt string

		err := rows.Scan(
			&loc.ID, &loc.PatternID, &loc.FilePath, &loc.LineStart, &loc.LineEnd,
			&loc.Snippet, &isOutlier, &loc.OutlierReason, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan pattern location: %w", err)
		}

		loc.IsOutlier = isOutlier == 1
		loc.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		locations = append(locations, loc)
	}

	return locations, rows.Err()
}

// GetPatternOutliers retrieves outlier locations for a pattern.
func (m *Memory) GetPatternOutliers(patternID string) ([]PatternLocation, error) {
	query := `
		SELECT id, pattern_id, file_path, line_start, line_end,
			snippet, is_outlier, outlier_reason, created_at
		FROM pattern_locations
		WHERE pattern_id = ? AND is_outlier = 1
		ORDER BY file_path, line_start
	`

	rows, err := m.db.QueryContext(context.Background(), query, patternID)
	if err != nil {
		return nil, fmt.Errorf("query pattern outliers: %w", err)
	}
	defer rows.Close()

	var locations []PatternLocation
	for rows.Next() {
		var loc PatternLocation
		var isOutlier int
		var createdAt string

		err := rows.Scan(
			&loc.ID, &loc.PatternID, &loc.FilePath, &loc.LineStart, &loc.LineEnd,
			&loc.Snippet, &isOutlier, &loc.OutlierReason, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan pattern location: %w", err)
		}

		loc.IsOutlier = isOutlier == 1
		loc.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		locations = append(locations, loc)
	}

	return locations, rows.Err()
}

// GetOutliersForFile retrieves all pattern outliers in a specific file.
func (m *Memory) GetOutliersForFile(filePath string) ([]PatternLocation, error) {
	query := `
		SELECT id, pattern_id, file_path, line_start, line_end,
			snippet, is_outlier, outlier_reason, created_at
		FROM pattern_locations
		WHERE file_path = ? AND is_outlier = 1
		ORDER BY line_start
	`

	rows, err := m.db.QueryContext(context.Background(), query, filePath)
	if err != nil {
		return nil, fmt.Errorf("query file outliers: %w", err)
	}
	defer rows.Close()

	var locations []PatternLocation
	for rows.Next() {
		var loc PatternLocation
		var isOutlier int
		var createdAt string

		err := rows.Scan(
			&loc.ID, &loc.PatternID, &loc.FilePath, &loc.LineStart, &loc.LineEnd,
			&loc.Snippet, &isOutlier, &loc.OutlierReason, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan pattern location: %w", err)
		}

		loc.IsOutlier = isOutlier == 1
		loc.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		locations = append(locations, loc)
	}

	return locations, rows.Err()
}

// DeletePatternLocations removes all locations for a pattern.
func (m *Memory) DeletePatternLocations(patternID string) error {
	_, err := m.db.ExecContext(context.Background(),
		"DELETE FROM pattern_locations WHERE pattern_id = ?", patternID)
	return err
}

// CountPatterns returns the number of patterns, optionally filtered by status.
func (m *Memory) CountPatterns(status string) (int, error) {
	query := "SELECT COUNT(*) FROM patterns"
	var args []any
	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}

	var count int
	err := m.db.QueryRowContext(context.Background(), query, args...).Scan(&count)
	return count, err
}

// GetPatternsByFile retrieves all patterns that have locations in a specific file.
func (m *Memory) GetPatternsByFile(filePath string) ([]Pattern, error) {
	query := `
		SELECT DISTINCT p.id, p.category, p.subcategory, p.name, p.description, p.detector_id,
			p.confidence, p.frequency_score, p.consistency_score, p.spread_score, p.age_score,
			p.status, p.authority, p.learning_id, p.metadata,
			p.first_seen, p.last_seen, p.created_at, p.updated_at
		FROM patterns p
		JOIN pattern_locations pl ON p.id = pl.pattern_id
		WHERE pl.file_path = ?
		ORDER BY p.confidence DESC
	`

	rows, err := m.db.QueryContext(context.Background(), query, filePath)
	if err != nil {
		return nil, fmt.Errorf("query patterns by file: %w", err)
	}
	defer rows.Close()

	var patterns []Pattern
	for rows.Next() {
		p, err := scanPatternRows(rows)
		if err != nil {
			return nil, fmt.Errorf("scan pattern: %w", err)
		}
		patterns = append(patterns, *p)
	}

	return patterns, rows.Err()
}

// BulkApprovePatterns approves all patterns meeting the confidence threshold.
func (m *Memory) BulkApprovePatterns(minConfidence float64) (int, error) {
	query := `
		UPDATE patterns
		SET status = 'approved', authority = 'approved', updated_at = ?
		WHERE status = 'discovered' AND confidence >= ?
	`

	result, err := m.db.ExecContext(context.Background(), query,
		time.Now().UTC().Format(time.RFC3339), minConfidence)
	if err != nil {
		return 0, err
	}

	affected, _ := result.RowsAffected()
	return int(affected), nil
}

// BulkApprovePatternsWithLearnings approves patterns and creates linked learnings.
// Returns the number of patterns approved and a map of pattern IDs to learning IDs.
func (m *Memory) BulkApprovePatternsWithLearnings(minConfidence float64) (int, map[string]string, error) {
	// Get discovered patterns meeting threshold
	patterns, err := m.GetPatterns(PatternFilters{
		Status:        "discovered",
		MinConfidence: minConfidence,
		Limit:         1000,
	})
	if err != nil {
		return 0, nil, fmt.Errorf("get patterns: %w", err)
	}

	learningMap := make(map[string]string)
	approved := 0

	for _, p := range patterns {
		learningID, err := m.ApprovePatternWithLearning(p.ID)
		if err != nil {
			// Log error but continue with other patterns
			continue
		}
		learningMap[p.ID] = learningID
		approved++
	}

	return approved, learningMap, nil
}

// Helper functions for scanning

func scanPattern(row *sql.Row) (*Pattern, error) {
	var p Pattern
	var metadataJSON string
	var firstSeen, lastSeen, createdAt, updatedAt string

	err := row.Scan(
		&p.ID, &p.Category, &p.Subcategory, &p.Name, &p.Description, &p.DetectorID,
		&p.Confidence, &p.FrequencyScore, &p.ConsistencyScore, &p.SpreadScore, &p.AgeScore,
		&p.Status, &p.Authority, &p.LearningID, &metadataJSON,
		&firstSeen, &lastSeen, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan pattern: %w", err)
	}

	p.FirstSeen, _ = time.Parse(time.RFC3339, firstSeen)
	p.LastSeen, _ = time.Parse(time.RFC3339, lastSeen)
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	if metadataJSON != "" {
		_ = json.Unmarshal([]byte(metadataJSON), &p.Metadata)
	}
	if p.Metadata == nil {
		p.Metadata = make(map[string]any)
	}

	return &p, nil
}

func scanPatternRows(rows *sql.Rows) (*Pattern, error) {
	var p Pattern
	var metadataJSON string
	var firstSeen, lastSeen, createdAt, updatedAt string

	err := rows.Scan(
		&p.ID, &p.Category, &p.Subcategory, &p.Name, &p.Description, &p.DetectorID,
		&p.Confidence, &p.FrequencyScore, &p.ConsistencyScore, &p.SpreadScore, &p.AgeScore,
		&p.Status, &p.Authority, &p.LearningID, &metadataJSON,
		&firstSeen, &lastSeen, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	p.FirstSeen, _ = time.Parse(time.RFC3339, firstSeen)
	p.LastSeen, _ = time.Parse(time.RFC3339, lastSeen)
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	if metadataJSON != "" {
		_ = json.Unmarshal([]byte(metadataJSON), &p.Metadata)
	}
	if p.Metadata == nil {
		p.Metadata = make(map[string]any)
	}

	return &p, nil
}
