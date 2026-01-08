package index

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/analysis"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/fsutil"
)

// FileChange represents a detected change to a file
type FileChange struct {
	Path    string // Relative path from root
	Action  string // "added", "modified", "deleted"
	OldHash string // Hash before change (empty for added)
	NewHash string // Hash after change (empty for deleted)
}

// IncrementalScanSummary contains results of an incremental scan
type IncrementalScanSummary struct {
	FilesAdded     int
	FilesModified  int
	FilesDeleted   int
	FilesUnchanged int
	Duration       time.Duration
}

// DetectChanges compares the filesystem against the database index
// and returns a list of files that have changed.
func DetectChanges(db *sql.DB, root string, guardrails config.Guardrails) ([]FileChange, error) {
	// Get all indexed files with their hashes
	indexed := make(map[string]string) // path -> hash
	rows, err := db.QueryContext(context.Background(), "SELECT path, hash FROM files")
	if err != nil {
		return nil, fmt.Errorf("query indexed files: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var path, hash string
		if err := rows.Scan(&path, &hash); err != nil {
			return nil, fmt.Errorf("scan indexed file: %w", err)
		}
		indexed[path] = hash
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate indexed files: %w", err)
	}

	// List files on disk
	diskFiles, err := fsutil.ListFiles(root, guardrails)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	sort.Strings(diskFiles)

	var changes []FileChange

	// Check each file on disk
	for _, relPath := range diskFiles {
		absPath := filepath.Join(root, relPath)
		data, err := os.ReadFile(absPath)
		if err != nil {
			// Skip files we can't read
			continue
		}

		h := sha256.Sum256(data)
		newHash := fmt.Sprintf("%x", h[:])

		if oldHash, exists := indexed[relPath]; exists {
			// File exists in index
			if oldHash != newHash {
				changes = append(changes, FileChange{
					Path:    relPath,
					Action:  "modified",
					OldHash: oldHash,
					NewHash: newHash,
				})
			}
			// Mark as seen by removing from indexed map
			delete(indexed, relPath)
		} else {
			// New file
			changes = append(changes, FileChange{
				Path:    relPath,
				Action:  "added",
				OldHash: "",
				NewHash: newHash,
			})
		}
	}

	// Remaining entries in indexed map are deleted files
	for path, hash := range indexed {
		changes = append(changes, FileChange{
			Path:    path,
			Action:  "deleted",
			OldHash: hash,
			NewHash: "",
		})
	}

	return changes, nil
}

// IncrementalScan only processes files that have changed since the last scan.
// It's much faster than a full scan for large codebases with few changes.
func IncrementalScan(db *sql.DB, root string, changes []FileChange) (IncrementalScanSummary, error) {
	startTime := time.Now()
	summary := IncrementalScanSummary{}

	if len(changes) == 0 {
		summary.Duration = time.Since(startTime)
		return summary, nil
	}

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return summary, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, change := range changes {
		switch change.Action {
		case "deleted":
			if err := deleteFileFromIndex(tx, change.Path); err != nil {
				return summary, fmt.Errorf("delete %s: %w", change.Path, err)
			}
			summary.FilesDeleted++

		case "added", "modified":
			// Remove old data for this file (safe for added files too)
			if err := deleteFileFromIndex(tx, change.Path); err != nil {
				return summary, fmt.Errorf("delete old %s: %w", change.Path, err)
			}

			// Read and index the file
			absPath := filepath.Join(root, change.Path)
			if err := indexSingleFile(tx, change.Path, absPath); err != nil {
				return summary, fmt.Errorf("index %s: %w", change.Path, err)
			}

			if change.Action == "added" {
				summary.FilesAdded++
			} else {
				summary.FilesModified++
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return summary, fmt.Errorf("commit: %w", err)
	}

	summary.Duration = time.Since(startTime)
	return summary, nil
}

// deleteFileFromIndex removes a file and all associated data from the index
func deleteFileFromIndex(tx *sql.Tx, relPath string) error {
	// FTS tables need to be cleaned first
	if _, err := tx.ExecContext(context.Background(), "DELETE FROM chunks_fts WHERE path = ?", relPath); err != nil {
		return fmt.Errorf("delete from chunks_fts: %w", err)
	}
	if _, err := tx.ExecContext(context.Background(), "DELETE FROM symbols_fts WHERE file_path = ?", relPath); err != nil {
		return fmt.Errorf("delete from symbols_fts: %w", err)
	}

	// Regular tables (CASCADE handles relationships)
	if _, err := tx.ExecContext(context.Background(), "DELETE FROM files WHERE path = ?", relPath); err != nil {
		return fmt.Errorf("delete from files: %w", err)
	}

	return nil
}

// indexSingleFile indexes a single file into the database
func indexSingleFile(tx *sql.Tx, relPath, absPath string) error {
	// Read file info and content
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	h := sha256.Sum256(data)
	hash := fmt.Sprintf("%x", h[:])
	now := time.Now().UTC().Format(time.RFC3339)

	// Detect language and analyze
	lang := analysis.DetectLanguage(relPath)
	var fileAnalysis *analysis.FileAnalysis
	if lang != analysis.LangUnknown {
		fa, err := analysis.Analyze(data, relPath)
		if err == nil {
			fileAnalysis = fa
		}
	}

	// Insert file record
	_, err = tx.ExecContext(context.Background(), `INSERT INTO files(path, hash, size, mod_time, indexed_at, language) VALUES(?, ?, ?, ?, ?, ?);`,
		relPath, hash, info.Size(), fsutil.NormalizeModTime(info.ModTime()).Format(time.RFC3339), now, string(lang))
	if err != nil {
		return fmt.Errorf("insert file: %w", err)
	}

	// Insert chunks
	chunks := fsutil.ChunkContent(string(data), 120, 8*1024)
	for i, chunk := range chunks {
		_, err = tx.ExecContext(context.Background(), `INSERT INTO chunks(path, chunk_index, start_line, end_line, content) VALUES(?, ?, ?, ?, ?);`,
			relPath, i, chunk.StartLine, chunk.EndLine, chunk.Content)
		if err != nil {
			return fmt.Errorf("insert chunk: %w", err)
		}
		_, err = tx.ExecContext(context.Background(), `INSERT INTO chunks_fts(path, content, chunk_index) VALUES(?, ?, ?);`,
			relPath, chunk.Content, i)
		if err != nil {
			return fmt.Errorf("insert chunk_fts: %w", err)
		}
	}

	// Insert symbols if analysis succeeded
	if fileAnalysis != nil {
		if err := insertSymbolsRecursive(tx, relPath, fileAnalysis.Symbols, nil); err != nil {
			return fmt.Errorf("insert symbols: %w", err)
		}

		// Insert relationships
		for _, rel := range fileAnalysis.Relationships {
			_, err = tx.ExecContext(context.Background(), `INSERT INTO relationships(source_file, source_symbol_id, target_file, target_symbol, kind, line, column) VALUES(?, ?, ?, ?, ?, ?, ?);`,
				relPath, nil, rel.TargetFile, rel.TargetSymbol, string(rel.Kind), rel.Line, rel.Column)
			if err != nil {
				return fmt.Errorf("insert relationship: %w", err)
			}
		}
	}

	return nil
}

// insertSymbolsRecursive inserts symbols and their children recursively
func insertSymbolsRecursive(tx *sql.Tx, filePath string, symbols []analysis.Symbol, parentID *int64) error {
	for _, sym := range symbols {
		exported := 0
		if sym.Exported {
			exported = 1
		}

		result, err := tx.ExecContext(context.Background(), `INSERT INTO symbols(file_path, name, kind, line_start, line_end, signature, doc_comment, parent_id, exported) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?);`,
			filePath, sym.Name, string(sym.Kind), sym.LineStart, sym.LineEnd, sym.Signature, sym.DocComment, parentID, exported)
		if err != nil {
			return fmt.Errorf("insert symbol %s: %w", sym.Name, err)
		}

		// Insert into FTS
		_, err = tx.ExecContext(context.Background(), `INSERT INTO symbols_fts(name, file_path, kind, doc_comment) VALUES(?, ?, ?, ?);`,
			sym.Name, filePath, string(sym.Kind), sym.DocComment)
		if err != nil {
			return fmt.Errorf("insert symbol_fts %s: %w", sym.Name, err)
		}

		// Insert children recursively
		if len(sym.Children) > 0 {
			symID, err := result.LastInsertId()
			if err != nil {
				return fmt.Errorf("get symbol id for %s: %w", sym.Name, err)
			}
			if err := insertSymbolsRecursive(tx, filePath, sym.Children, &symID); err != nil {
				return err
			}
		}
	}
	return nil
}
