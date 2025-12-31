package index

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/analysis"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/fsutil"
)

type FileRecord struct {
	Path     string
	Hash     string
	Size     int64
	ModTime  time.Time
	Chunks   []fsutil.Chunk
	Language string
	Analysis *analysis.FileAnalysis
}

type ScanSummary struct {
	ID                int64
	Root              string
	ScanHash          string
	FileCount         int
	ChunkCount        int
	SymbolCount       int
	RelationshipCount int
	StartedAt         time.Time
	CompletedAt       time.Time
}

func Open(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA foreign_keys=ON;",
		"PRAGMA synchronous=NORMAL;",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("apply pragma %s: %w", p, err)
		}
	}
	if err := ensureSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// indexSchemaVersionTable creates the schema version tracking table
const indexSchemaVersionTable = `
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL
);
`

// indexMigrations is an ordered list of database migrations for the index DB.
// Each migration is a function that takes a transaction and applies schema changes.
// Migrations are applied in order, starting from version 0.
// IMPORTANT: Never modify existing migrations, only add new ones.
var indexMigrations = []func(*sql.Tx) error{
	// Migration 0: Initial schema
	indexMigrateV0,
}

// indexMigrateV0 creates the initial index schema (version 0)
func indexMigrateV0(tx *sql.Tx) error {
	stmts := []string{
		// Core file storage
		`CREATE TABLE IF NOT EXISTS files (
            path TEXT PRIMARY KEY,
            hash TEXT NOT NULL,
            size INTEGER NOT NULL,
            mod_time TEXT NOT NULL,
            indexed_at TEXT NOT NULL,
            language TEXT DEFAULT ''
        );`,
		`CREATE TABLE IF NOT EXISTS chunks (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            path TEXT NOT NULL,
            chunk_index INTEGER NOT NULL,
            start_line INTEGER NOT NULL,
            end_line INTEGER NOT NULL,
            content TEXT NOT NULL,
            FOREIGN KEY(path) REFERENCES files(path) ON DELETE CASCADE
        );`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(
			path,
			content,
			chunk_index,
			tokenize="unicode61 tokenchars '_.:@#$-'"
		);`,
		`CREATE TABLE IF NOT EXISTS scans (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            root TEXT NOT NULL,
            scan_hash TEXT NOT NULL,
            started_at TEXT NOT NULL,
            completed_at TEXT NOT NULL
        );`,
		`CREATE INDEX IF NOT EXISTS idx_chunks_path ON chunks(path);`,

		// Symbols: functions, classes, methods, variables
		`CREATE TABLE IF NOT EXISTS symbols (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            file_path TEXT NOT NULL,
            name TEXT NOT NULL,
            kind TEXT NOT NULL,
            line_start INTEGER NOT NULL,
            line_end INTEGER NOT NULL,
            signature TEXT DEFAULT '',
            doc_comment TEXT DEFAULT '',
            parent_id INTEGER DEFAULT NULL,
            exported INTEGER DEFAULT 0,
            FOREIGN KEY(file_path) REFERENCES files(path) ON DELETE CASCADE,
            FOREIGN KEY(parent_id) REFERENCES symbols(id) ON DELETE CASCADE
        );`,
		`CREATE INDEX IF NOT EXISTS idx_symbols_file ON symbols(file_path);`,
		`CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name);`,
		`CREATE INDEX IF NOT EXISTS idx_symbols_kind ON symbols(kind);`,

		// FTS for symbol names
		`CREATE VIRTUAL TABLE IF NOT EXISTS symbols_fts USING fts5(
			name,
			file_path,
			kind,
			doc_comment,
			tokenize="unicode61 tokenchars '_'"
		);`,

		// Relationships: imports, calls, references, extends
		`CREATE TABLE IF NOT EXISTS relationships (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            source_file TEXT NOT NULL,
            source_symbol_id INTEGER DEFAULT NULL,
            target_file TEXT DEFAULT NULL,
            target_symbol TEXT DEFAULT NULL,
            kind TEXT NOT NULL,
            line INTEGER DEFAULT 0,
            column INTEGER DEFAULT 0,
            FOREIGN KEY(source_file) REFERENCES files(path) ON DELETE CASCADE,
            FOREIGN KEY(source_symbol_id) REFERENCES symbols(id) ON DELETE CASCADE
        );`,
		`CREATE INDEX IF NOT EXISTS idx_rel_source ON relationships(source_file);`,
		`CREATE INDEX IF NOT EXISTS idx_rel_target ON relationships(target_file);`,
		`CREATE INDEX IF NOT EXISTS idx_rel_kind ON relationships(kind);`,

		// Decisions: architectural memory
		`CREATE TABLE IF NOT EXISTS decisions (
            id TEXT PRIMARY KEY,
            room TEXT DEFAULT '',
            title TEXT NOT NULL,
            summary TEXT DEFAULT '',
            rationale TEXT DEFAULT '',
            affected_files TEXT DEFAULT '[]',
            created_at TEXT NOT NULL,
            created_by TEXT DEFAULT ''
        );`,

		// Sessions: task continuity
		`CREATE TABLE IF NOT EXISTS sessions (
            id TEXT PRIMARY KEY,
            goal TEXT DEFAULT '',
            room TEXT DEFAULT '',
            started_at TEXT NOT NULL,
            last_activity TEXT NOT NULL,
            files_touched TEXT DEFAULT '[]',
            learnings TEXT DEFAULT '[]',
            warnings TEXT DEFAULT '[]',
            state TEXT DEFAULT '{}'
        );`,

		// Rooms: stored room metadata for quick access
		`CREATE TABLE IF NOT EXISTS rooms (
            name TEXT PRIMARY KEY,
            summary TEXT DEFAULT '',
            entry_points TEXT DEFAULT '[]',
            file_patterns TEXT DEFAULT '[]',
            updated_at TEXT NOT NULL
        );`,
	}

	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	}
	return nil
}

func ensureSchema(db *sql.DB) error {
	// Create schema version table first
	if _, err := db.Exec(indexSchemaVersionTable); err != nil {
		return fmt.Errorf("create schema_version table: %w", err)
	}

	// Get current schema version
	var currentVersion int
	row := db.QueryRow("SELECT COALESCE(MAX(version), -1) FROM schema_version")
	if err := row.Scan(&currentVersion); err != nil {
		return fmt.Errorf("get schema version: %w", err)
	}

	// Run pending migrations
	for i := currentVersion + 1; i < len(indexMigrations); i++ {
		if err := runIndexMigration(db, i); err != nil {
			return fmt.Errorf("run migration %d: %w", i, err)
		}
	}

	return nil
}

// runIndexMigration executes a single migration in a transaction
func runIndexMigration(db *sql.DB, version int) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Run the migration
	if err := indexMigrations[version](tx); err != nil {
		return fmt.Errorf("execute migration: %w", err)
	}

	// Record the migration
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.Exec("INSERT INTO schema_version (version, applied_at) VALUES (?, ?)", version, now); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	return tx.Commit()
}

// GetIndexSchemaVersion returns the current index schema version
func GetIndexSchemaVersion(db *sql.DB) (int, error) {
	var version int
	row := db.QueryRow("SELECT COALESCE(MAX(version), -1) FROM schema_version")
	err := row.Scan(&version)
	return version, err
}

func BuildFileRecords(root string, guardrails config.Guardrails) ([]FileRecord, error) {
	files, err := fsutil.ListFiles(root, guardrails)
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	records := make([]FileRecord, 0, len(files))
	for _, rel := range files {
		abs := filepath.Join(root, rel)
		info, err := os.Stat(abs)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", rel, err)
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", rel, err)
		}
		h := sha256.Sum256(data)
		chunks := fsutil.ChunkContent(string(data), 120, 8*1024)

		// Perform language analysis
		lang := analysis.DetectLanguage(rel)
		var fileAnalysis *analysis.FileAnalysis
		if lang != analysis.LangUnknown {
			fa, err := analysis.Analyze(data, rel)
			if err == nil {
				fileAnalysis = fa
			}
		}

		records = append(records, FileRecord{
			Path:     rel,
			Hash:     fmt.Sprintf("%x", h[:]),
			Size:     info.Size(),
			ModTime:  fsutil.NormalizeModTime(info.ModTime()),
			Chunks:   chunks,
			Language: string(lang),
			Analysis: fileAnalysis,
		})
	}
	return records, nil
}

func WriteScan(db *sql.DB, root string, records []FileRecord, startedAt time.Time) (ScanSummary, error) {
	tx, err := db.Begin()
	if err != nil {
		return ScanSummary{}, err
	}
	defer tx.Rollback()

	clearStmts := []string{
		"DELETE FROM relationships;",
		"DELETE FROM symbols_fts;",
		"DELETE FROM symbols;",
		"DELETE FROM chunks;",
		"DELETE FROM chunks_fts;",
		"DELETE FROM files;",
	}
	for _, stmt := range clearStmts {
		if _, err := tx.Exec(stmt); err != nil {
			return ScanSummary{}, fmt.Errorf("reset index: %w", err)
		}
	}

	now := time.Now().UTC()
	chunkCount := 0
	symbolCount := 0
	relationshipCount := 0

	fileStmt, err := tx.Prepare(`INSERT INTO files(path, hash, size, mod_time, indexed_at, language) VALUES(?, ?, ?, ?, ?, ?);`)
	if err != nil {
		return ScanSummary{}, err
	}
	defer fileStmt.Close()

	chunkStmt, err := tx.Prepare(`INSERT INTO chunks(path, chunk_index, start_line, end_line, content) VALUES(?, ?, ?, ?, ?);`)
	if err != nil {
		return ScanSummary{}, err
	}
	defer chunkStmt.Close()

	ftsStmt, err := tx.Prepare(`INSERT INTO chunks_fts(path, content, chunk_index) VALUES(?, ?, ?);`)
	if err != nil {
		return ScanSummary{}, err
	}
	defer ftsStmt.Close()

	symbolStmt, err := tx.Prepare(`INSERT INTO symbols(file_path, name, kind, line_start, line_end, signature, doc_comment, parent_id, exported) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?);`)
	if err != nil {
		return ScanSummary{}, err
	}
	defer symbolStmt.Close()

	symbolFtsStmt, err := tx.Prepare(`INSERT INTO symbols_fts(name, file_path, kind, doc_comment) VALUES(?, ?, ?, ?);`)
	if err != nil {
		return ScanSummary{}, err
	}
	defer symbolFtsStmt.Close()

	relStmt, err := tx.Prepare(`INSERT INTO relationships(source_file, source_symbol_id, target_file, target_symbol, kind, line, column) VALUES(?, ?, ?, ?, ?, ?, ?);`)
	if err != nil {
		return ScanSummary{}, err
	}
	defer relStmt.Close()

	for _, r := range records {
		if _, err := fileStmt.Exec(r.Path, r.Hash, r.Size, r.ModTime.Format(time.RFC3339), now.Format(time.RFC3339), r.Language); err != nil {
			return ScanSummary{}, fmt.Errorf("insert file %s: %w", r.Path, err)
		}

		for _, c := range r.Chunks {
			chunkCount++
			if _, err := chunkStmt.Exec(r.Path, c.Index, c.StartLine, c.EndLine, c.Content); err != nil {
				return ScanSummary{}, fmt.Errorf("insert chunk %s:%d: %w", r.Path, c.Index, err)
			}
			if _, err := ftsStmt.Exec(r.Path, c.Content, c.Index); err != nil {
				return ScanSummary{}, fmt.Errorf("insert fts %s:%d: %w", r.Path, c.Index, err)
			}
		}

		// Insert symbols and relationships from analysis
		if r.Analysis != nil {
			symCount, err := insertSymbols(tx, symbolStmt, symbolFtsStmt, r.Path, r.Analysis.Symbols, nil)
			if err != nil {
				return ScanSummary{}, fmt.Errorf("insert symbols %s: %w", r.Path, err)
			}
			symbolCount += symCount

			for _, rel := range r.Analysis.Relationships {
				relationshipCount++
				if _, err := relStmt.Exec(r.Path, nil, rel.TargetFile, rel.TargetSymbol, string(rel.Kind), rel.Line, rel.Column); err != nil {
					return ScanSummary{}, fmt.Errorf("insert relationship %s: %w", r.Path, err)
				}
			}
		}
	}

	scanHash := computeScanHash(records)
	res, err := tx.Exec(`INSERT INTO scans(root, scan_hash, started_at, completed_at) VALUES(?, ?, ?, ?);`, root, scanHash, startedAt.UTC().Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return ScanSummary{}, fmt.Errorf("insert scan: %w", err)
	}
	scanID, _ := res.LastInsertId()

	if err := tx.Commit(); err != nil {
		return ScanSummary{}, err
	}

	return ScanSummary{
		ID:                scanID,
		Root:              root,
		ScanHash:          scanHash,
		FileCount:         len(records),
		ChunkCount:        chunkCount,
		SymbolCount:       symbolCount,
		RelationshipCount: relationshipCount,
		StartedAt:         startedAt.UTC(),
		CompletedAt:       now,
	}, nil
}

func insertSymbols(tx *sql.Tx, symbolStmt, symbolFtsStmt *sql.Stmt, filePath string, symbols []analysis.Symbol, parentID *int64) (int, error) {
	count := 0
	for _, sym := range symbols {
		exported := 0
		if sym.Exported {
			exported = 1
		}

		res, err := symbolStmt.Exec(filePath, sym.Name, string(sym.Kind), sym.LineStart, sym.LineEnd, sym.Signature, sym.DocComment, parentID, exported)
		if err != nil {
			return count, err
		}
		count++

		if _, err := symbolFtsStmt.Exec(sym.Name, filePath, string(sym.Kind), sym.DocComment); err != nil {
			return count, err
		}

		// Insert children recursively
		if len(sym.Children) > 0 {
			symID, _ := res.LastInsertId()
			childCount, err := insertSymbols(tx, symbolStmt, symbolFtsStmt, filePath, sym.Children, &symID)
			if err != nil {
				return count, err
			}
			count += childCount
		}
	}
	return count, nil
}

func computeScanHash(records []FileRecord) string {
	h := sha256.New()
	for _, r := range records {
		h.Write([]byte(r.Path))
		h.Write([]byte(r.Hash))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func LatestScan(db *sql.DB) (ScanSummary, error) {
	row := db.QueryRow(`SELECT id, root, scan_hash, completed_at FROM scans ORDER BY id DESC LIMIT 1;`)
	var id int64
	var root, hash, completed string
	if err := row.Scan(&id, &root, &hash, &completed); err != nil {
		if err == sql.ErrNoRows {
			return ScanSummary{}, nil
		}
		return ScanSummary{}, err
	}
	t, err := time.Parse(time.RFC3339, completed)
	if err != nil {
		return ScanSummary{}, fmt.Errorf("parse completed_at: %w", err)
	}
	return ScanSummary{ID: id, Root: root, ScanHash: hash, CompletedAt: t}, nil
}

type FileMetadata struct {
	Hash    string
	Size    int64
	ModTime time.Time
}

func LoadFileMetadata(db *sql.DB) (map[string]FileMetadata, error) {
	rows, err := db.Query(`SELECT path, hash, size, mod_time FROM files;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]FileMetadata)
	for rows.Next() {
		var path, hash, modTimeStr string
		var size int64
		if err := rows.Scan(&path, &hash, &size, &modTimeStr); err != nil {
			return nil, err
		}
		mt, err := time.Parse(time.RFC3339, modTimeStr)
		if err != nil {
			return nil, fmt.Errorf("parse mod_time for %s: %w", path, err)
		}
		out[path] = FileMetadata{Hash: hash, Size: size, ModTime: mt}
	}
	return out, rows.Err()
}

// DBHandle aliases sql.DB for external packages.
type DBHandle = sql.DB

type ChunkHit struct {
	Path       string
	ChunkIndex int
	StartLine  int
	EndLine    int
	Content    string
}

type ChunkRow struct {
	ChunkIndex int
	StartLine  int
	EndLine    int
	Content    string
}

func SearchChunks(db *sql.DB, query string, limit int) ([]ChunkHit, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	escaped := sanitizeFTSQuery(query)
	rows, err := db.Query(`
        SELECT c.path, c.chunk_index, c.start_line, c.end_line, c.content
        FROM chunks_fts
        JOIN chunks c ON c.path = chunks_fts.path AND c.chunk_index = chunks_fts.chunk_index
        WHERE chunks_fts MATCH ?
        ORDER BY c.path, c.chunk_index
        LIMIT ?;
    `, escaped, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hits []ChunkHit
	for rows.Next() {
		var h ChunkHit
		if err := rows.Scan(&h.Path, &h.ChunkIndex, &h.StartLine, &h.EndLine, &h.Content); err != nil {
			return nil, err
		}
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

func sanitizeFTSQuery(q string) string {
	trimmed := strings.TrimSpace(q)
	trimmed = strings.ReplaceAll(trimmed, "\"", "\"\"")
	return fmt.Sprintf("\"%s\"", trimmed)
}

func GetChunksForFile(db *sql.DB, path string) ([]ChunkRow, error) {
	rows, err := db.Query(`SELECT chunk_index, start_line, end_line, content FROM chunks WHERE path = ? ORDER BY chunk_index ASC;`, path)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChunkRow
	for rows.Next() {
		var row ChunkRow
		if err := rows.Scan(&row.ChunkIndex, &row.StartLine, &row.EndLine, &row.Content); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
