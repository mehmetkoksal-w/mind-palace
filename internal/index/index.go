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

	"github.com/koksalmehmet/mind-palace/internal/config"
	"github.com/koksalmehmet/mind-palace/internal/fsutil"
)

type FileRecord struct {
	Path    string
	Hash    string
	Size    int64
	ModTime time.Time
	Chunks  []fsutil.Chunk
}

type ScanSummary struct {
	ID          int64
	Root        string
	ScanHash    string
	FileCount   int
	ChunkCount  int
	StartedAt   time.Time
	CompletedAt time.Time
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

func ensureSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS files (
            path TEXT PRIMARY KEY,
            hash TEXT NOT NULL,
            size INTEGER NOT NULL,
            mod_time TEXT NOT NULL,
            indexed_at TEXT NOT NULL
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
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("ensure schema: %w", err)
		}
	}
	return nil
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
		records = append(records, FileRecord{
			Path:    rel,
			Hash:    fmt.Sprintf("%x", h[:]),
			Size:    info.Size(),
			ModTime: fsutil.NormalizeModTime(info.ModTime()),
			Chunks:  chunks,
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
	fileStmt, err := tx.Prepare(`INSERT INTO files(path, hash, size, mod_time, indexed_at) VALUES(?, ?, ?, ?, ?);`)
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

	for _, r := range records {
		if _, err := fileStmt.Exec(r.Path, r.Hash, r.Size, r.ModTime.Format(time.RFC3339), now.Format(time.RFC3339)); err != nil {
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
		ID:          scanID,
		Root:        root,
		ScanHash:    scanHash,
		FileCount:   len(records),
		ChunkCount:  chunkCount,
		StartedAt:   startedAt.UTC(),
		CompletedAt: now,
	}, nil
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
