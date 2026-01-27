// Package index provides the core database functionality for indexing project files and symbols.
package index

import (
	"context"
	"database/sql"
	"math"
)

// FileUsageScore represents usage metrics for a file
type FileUsageScore struct {
	Path          string  `json:"path"`
	IncomingCalls int     `json:"incomingCalls"` // How many times symbols in this file are called
	OutgoingCalls int     `json:"outgoingCalls"` // How many calls this file makes
	ImportedBy    int     `json:"importedBy"`    // How many files import this one
	Imports       int     `json:"imports"`       // How many files this imports
	SymbolCount   int     `json:"symbolCount"`   // Number of symbols defined
	UsageScore    float64 `json:"usageScore"`    // Computed normalized score (0-1)
}

// UsageWeights configures how different metrics contribute to usage score
type UsageWeights struct {
	IncomingCallWeight float64 // Weight for incoming calls (default: 2.0)
	ImportedByWeight   float64 // Weight for being imported (default: 1.5)
	OutgoingCallWeight float64 // Weight for outgoing calls (default: 0.5)
	ImportWeight       float64 // Weight for imports (default: 0.3)
	SymbolWeight       float64 // Weight for symbol count (default: 0.2)
}

// DefaultUsageWeights returns sensible defaults for usage scoring
func DefaultUsageWeights() *UsageWeights {
	return &UsageWeights{
		IncomingCallWeight: 2.0,
		ImportedByWeight:   1.5,
		OutgoingCallWeight: 0.5,
		ImportWeight:       0.3,
		SymbolWeight:       0.2,
	}
}

// GetFileUsageScores computes usage scores for a list of files.
// Higher scores indicate more "central" or heavily-used files.
func GetFileUsageScores(db *sql.DB, files []string) (map[string]*FileUsageScore, error) {
	return GetFileUsageScoresWithWeights(db, files, nil)
}

// GetFileUsageScoresWithWeights computes usage scores with custom weights.
func GetFileUsageScoresWithWeights(db *sql.DB, files []string, weights *UsageWeights) (map[string]*FileUsageScore, error) {
	if weights == nil {
		weights = DefaultUsageWeights()
	}

	scores := make(map[string]*FileUsageScore)
	maxRawScore := 0.0

	for _, path := range files {
		score := &FileUsageScore{Path: path}

		// Count incoming calls (symbols in this file being called from elsewhere)
		err := db.QueryRowContext(context.Background(), `
			SELECT COUNT(*) FROM relationships r
			JOIN symbols s ON r.target_symbol = s.name
			WHERE s.file_path = ? AND r.kind = 'call' AND r.source_file != ?
		`, path, path).Scan(&score.IncomingCalls)
		if err != nil && err != sql.ErrNoRows {
			score.IncomingCalls = 0
		}

		// Count outgoing calls (calls made from this file)
		err = db.QueryRowContext(context.Background(), `
			SELECT COUNT(*) FROM relationships
			WHERE source_file = ? AND kind = 'call'
		`, path).Scan(&score.OutgoingCalls)
		if err != nil && err != sql.ErrNoRows {
			score.OutgoingCalls = 0
		}

		// Count files that import this file
		err = db.QueryRowContext(context.Background(), `
			SELECT COUNT(DISTINCT source_file) FROM relationships
			WHERE target_file = ? AND kind = 'import'
		`, path).Scan(&score.ImportedBy)
		if err != nil && err != sql.ErrNoRows {
			score.ImportedBy = 0
		}

		// Count files this imports
		err = db.QueryRowContext(context.Background(), `
			SELECT COUNT(DISTINCT target_file) FROM relationships
			WHERE source_file = ? AND kind = 'import'
		`, path).Scan(&score.Imports)
		if err != nil && err != sql.ErrNoRows {
			score.Imports = 0
		}

		// Count symbols defined in this file
		err = db.QueryRowContext(context.Background(), `
			SELECT COUNT(*) FROM symbols WHERE file_path = ?
		`, path).Scan(&score.SymbolCount)
		if err != nil && err != sql.ErrNoRows {
			score.SymbolCount = 0
		}

		// Compute raw weighted score
		rawScore := float64(score.IncomingCalls)*weights.IncomingCallWeight +
			float64(score.ImportedBy)*weights.ImportedByWeight +
			float64(score.OutgoingCalls)*weights.OutgoingCallWeight +
			float64(score.Imports)*weights.ImportWeight +
			float64(score.SymbolCount)*weights.SymbolWeight

		score.UsageScore = rawScore
		if rawScore > maxRawScore {
			maxRawScore = rawScore
		}

		scores[path] = score
	}

	// Normalize scores to 0-1 range
	if maxRawScore > 0 {
		for _, s := range scores {
			s.UsageScore = s.UsageScore / maxRawScore
		}
	}

	return scores, nil
}

// GetMostImportedFiles returns files that are imported by the most other files.
// These are typically core utility or foundation files.
func GetMostImportedFiles(db *sql.DB, limit int) ([]FileUsageScore, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := db.QueryContext(context.Background(), `
		SELECT target_file, COUNT(DISTINCT source_file) as import_count
		FROM relationships
		WHERE kind = 'import' AND target_file IS NOT NULL AND target_file != ''
		GROUP BY target_file
		ORDER BY import_count DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []FileUsageScore
	for rows.Next() {
		var score FileUsageScore
		if err := rows.Scan(&score.Path, &score.ImportedBy); err != nil {
			continue
		}
		// Normalize to 0-1 (first file has score 1.0)
		if len(results) == 0 {
			score.UsageScore = 1.0
		} else if results[0].ImportedBy > 0 {
			score.UsageScore = float64(score.ImportedBy) / float64(results[0].ImportedBy)
		}
		results = append(results, score)
	}

	return results, rows.Err()
}

// GetMostConnectedFiles returns files with the highest combined incoming and outgoing relationships.
// These are "hub" files that connect different parts of the codebase.
func GetMostConnectedFiles(db *sql.DB, limit int) ([]FileUsageScore, error) {
	if limit <= 0 {
		limit = 20
	}

	// This query counts both incoming and outgoing relationships per file
	rows, err := db.QueryContext(context.Background(), `
		WITH file_connections AS (
			SELECT source_file as file, COUNT(*) as outgoing, 0 as incoming
			FROM relationships
			WHERE kind IN ('call', 'import', 'reference')
			GROUP BY source_file
			UNION ALL
			SELECT target_file as file, 0 as outgoing, COUNT(*) as incoming
			FROM relationships
			WHERE kind IN ('call', 'import', 'reference') AND target_file IS NOT NULL
			GROUP BY target_file
		)
		SELECT file, SUM(outgoing) as total_out, SUM(incoming) as total_in
		FROM file_connections
		WHERE file IS NOT NULL AND file != ''
		GROUP BY file
		ORDER BY (SUM(outgoing) + SUM(incoming)) DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []FileUsageScore
	maxScore := 0.0
	for rows.Next() {
		var score FileUsageScore
		var outgoing, incoming int
		if err := rows.Scan(&score.Path, &outgoing, &incoming); err != nil {
			continue
		}
		score.OutgoingCalls = outgoing
		score.IncomingCalls = incoming
		combined := float64(outgoing + incoming)
		if combined > maxScore {
			maxScore = combined
		}
		score.UsageScore = combined
		results = append(results, score)
	}

	// Normalize
	if maxScore > 0 {
		for i := range results {
			results[i].UsageScore = results[i].UsageScore / maxScore
		}
	}

	return results, rows.Err()
}

// GetSymbolCentrality computes how "central" a symbol is based on call relationships.
// Symbols with high centrality are called from many places and/or call many other symbols.
func GetSymbolCentrality(db *sql.DB, symbolName, filePath string) (float64, error) {
	var inCount, outCount int

	// Count incoming calls to this symbol
	err := db.QueryRowContext(context.Background(), `
		SELECT COUNT(*) FROM relationships
		WHERE kind = 'call'
		AND (target_symbol = ? OR target_symbol LIKE ? OR target_symbol LIKE ?)
	`, symbolName, "%."+symbolName, "%::"+symbolName).Scan(&inCount)
	if err != nil && err != sql.ErrNoRows {
		inCount = 0
	}

	// Count outgoing calls from this symbol (if we have file path)
	if filePath != "" {
		// Get symbol's line range
		var startLine, endLine int
		err := db.QueryRowContext(context.Background(), `
			SELECT line_start, line_end FROM symbols
			WHERE name = ? AND file_path = ?
			LIMIT 1
		`, symbolName, filePath).Scan(&startLine, &endLine)
		if err == nil {
			// Count calls within that range
			err = db.QueryRowContext(context.Background(), `
				SELECT COUNT(*) FROM relationships
				WHERE kind = 'call' AND source_file = ?
				AND line >= ? AND line <= ?
			`, filePath, startLine, endLine).Scan(&outCount)
			if err != nil && err != sql.ErrNoRows {
				outCount = 0
			}
		}
	}

	// Compute centrality score (weighted combination)
	// Incoming calls weighted more heavily as they indicate usage
	centrality := float64(inCount)*2.0 + float64(outCount)*1.0

	// Apply log scaling to prevent extreme values from dominating
	if centrality > 0 {
		centrality = math.Log2(centrality + 1)
	}

	// Normalize to rough 0-1 range (assuming max ~100 calls)
	return math.Min(centrality/7.0, 1.0), nil
}

// BatchGetSymbolCentrality gets centrality scores for multiple symbols efficiently.
func BatchGetSymbolCentrality(db *sql.DB, symbols []SymbolInfo) (map[string]float64, error) {
	result := make(map[string]float64)

	for _, sym := range symbols {
		key := sym.FilePath + ":" + sym.Name
		centrality, err := GetSymbolCentrality(db, sym.Name, sym.FilePath)
		if err != nil {
			result[key] = 0
			continue
		}
		result[key] = centrality
	}

	return result, nil
}
