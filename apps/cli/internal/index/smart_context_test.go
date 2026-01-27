package index

import (
	"database/sql"
	"testing"
	"time"
)

// setupSmartContextTestDB creates a test database with sample data for smart context tests
func setupSmartContextTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	// Create schema
	stmts := []string{
		`CREATE TABLE files (
			path TEXT PRIMARY KEY,
			hash TEXT NOT NULL,
			size INTEGER NOT NULL,
			mod_time TEXT NOT NULL,
			indexed_at TEXT NOT NULL,
			language TEXT DEFAULT ''
		)`,
		`CREATE TABLE chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL,
			chunk_index INTEGER NOT NULL,
			start_line INTEGER NOT NULL,
			end_line INTEGER NOT NULL,
			content TEXT NOT NULL
		)`,
		`CREATE VIRTUAL TABLE chunks_fts USING fts5(path, content, chunk_index)`,
		`CREATE TABLE symbols (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			file_path TEXT NOT NULL,
			name TEXT NOT NULL,
			kind TEXT NOT NULL,
			line_start INTEGER NOT NULL,
			line_end INTEGER NOT NULL,
			signature TEXT DEFAULT '',
			doc_comment TEXT DEFAULT '',
			parent_id INTEGER DEFAULT NULL,
			exported INTEGER DEFAULT 0
		)`,
		`CREATE VIRTUAL TABLE symbols_fts USING fts5(name, file_path, kind, doc_comment)`,
		`CREATE TABLE relationships (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_file TEXT NOT NULL,
			source_symbol_id INTEGER DEFAULT NULL,
			target_file TEXT DEFAULT NULL,
			target_symbol TEXT DEFAULT NULL,
			kind TEXT NOT NULL,
			line INTEGER DEFAULT 0,
			column INTEGER DEFAULT 0
		)`,
		`CREATE INDEX idx_rel_source ON relationships(source_file)`,
		`CREATE INDEX idx_rel_target ON relationships(target_file)`,
		`CREATE INDEX idx_rel_kind ON relationships(kind)`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("exec %s: %v", stmt[:50], err)
		}
	}

	// Insert sample files
	files := []struct {
		path, lang string
	}{
		{"src/main.go", "go"},
		{"src/config/config.go", "go"},
		{"src/config/loader.go", "go"},
		{"src/handlers/auth.go", "go"},
		{"src/handlers/user.go", "go"},
		{"src/utils/helpers.go", "go"},
		{"src/models/user.go", "go"},
	}

	for _, f := range files {
		_, err := db.Exec(`INSERT INTO files (path, hash, size, mod_time, indexed_at, language)
			VALUES (?, 'hash123', 1000, datetime('now'), datetime('now'), ?)`, f.path, f.lang)
		if err != nil {
			t.Fatalf("insert file %s: %v", f.path, err)
		}
	}

	// Insert sample symbols
	symbols := []struct {
		filePath, name, kind string
		lineStart, lineEnd   int
		exported             int
	}{
		{"src/main.go", "main", "function", 1, 20, 0},
		{"src/config/config.go", "Config", "struct", 5, 15, 1},
		{"src/config/config.go", "LoadConfig", "function", 20, 40, 1},
		{"src/config/loader.go", "loadFromFile", "function", 10, 30, 0},
		{"src/handlers/auth.go", "AuthHandler", "struct", 5, 10, 1},
		{"src/handlers/auth.go", "HandleLogin", "method", 15, 50, 1},
		{"src/handlers/user.go", "UserHandler", "struct", 5, 10, 1},
		{"src/handlers/user.go", "GetUser", "method", 15, 40, 1},
		{"src/utils/helpers.go", "FormatDate", "function", 5, 15, 1},
		{"src/utils/helpers.go", "ParseID", "function", 20, 30, 1},
		{"src/models/user.go", "User", "struct", 5, 20, 1},
	}

	for _, s := range symbols {
		_, err := db.Exec(`INSERT INTO symbols (file_path, name, kind, line_start, line_end, exported)
			VALUES (?, ?, ?, ?, ?, ?)`, s.filePath, s.name, s.kind, s.lineStart, s.lineEnd, s.exported)
		if err != nil {
			t.Fatalf("insert symbol %s: %v", s.name, err)
		}

		// Also insert into FTS
		_, err = db.Exec(`INSERT INTO symbols_fts (name, file_path, kind, doc_comment)
			VALUES (?, ?, ?, '')`, s.name, s.filePath, s.kind)
		if err != nil {
			t.Fatalf("insert symbol fts %s: %v", s.name, err)
		}
	}

	// Insert import relationships
	imports := []struct {
		source, target string
	}{
		{"src/main.go", "src/config/config.go"},
		{"src/main.go", "src/handlers/auth.go"},
		{"src/main.go", "src/handlers/user.go"},
		{"src/config/config.go", "src/config/loader.go"},
		{"src/handlers/auth.go", "src/utils/helpers.go"},
		{"src/handlers/auth.go", "src/models/user.go"},
		{"src/handlers/user.go", "src/utils/helpers.go"},
		{"src/handlers/user.go", "src/models/user.go"},
	}

	for _, imp := range imports {
		_, err := db.Exec(`INSERT INTO relationships (source_file, target_file, kind, line)
			VALUES (?, ?, 'import', 1)`, imp.source, imp.target)
		if err != nil {
			t.Fatalf("insert import %s -> %s: %v", imp.source, imp.target, err)
		}
	}

	// Insert call relationships
	calls := []struct {
		sourceFile, targetSymbol string
		line                     int
	}{
		{"src/main.go", "LoadConfig", 10},
		{"src/main.go", "AuthHandler.HandleLogin", 15},
		{"src/handlers/auth.go", "FormatDate", 25},
		{"src/handlers/auth.go", "ParseID", 30},
		{"src/handlers/auth.go", "User", 35},
		{"src/handlers/user.go", "FormatDate", 25},
		{"src/handlers/user.go", "User", 30},
		{"src/config/config.go", "loadFromFile", 25},
	}

	for _, c := range calls {
		_, err := db.Exec(`INSERT INTO relationships (source_file, target_symbol, kind, line)
			VALUES (?, ?, 'call', ?)`, c.sourceFile, c.targetSymbol, c.line)
		if err != nil {
			t.Fatalf("insert call %s -> %s: %v", c.sourceFile, c.targetSymbol, err)
		}
	}

	return db
}

func TestExpandWithDependencies(t *testing.T) {
	db := setupSmartContextTestDB(t)
	defer db.Close()

	t.Run("expands imports from seed file", func(t *testing.T) {
		expanded, err := ExpandWithDependencies(db, []string{"src/main.go"}, nil)
		if err != nil {
			t.Fatalf("ExpandWithDependencies() error = %v", err)
		}

		// main.go imports config.go, auth.go, user.go
		if len(expanded) < 4 {
			t.Errorf("expected at least 4 files, got %d", len(expanded))
		}

		// Verify seed file has depth 0
		for _, ef := range expanded {
			if ef.Path == "src/main.go" {
				if ef.Depth != 0 {
					t.Errorf("seed file depth = %d, want 0", ef.Depth)
				}
				if len(ef.Imports) != 3 {
					t.Errorf("seed file imports = %d, want 3", len(ef.Imports))
				}
			}
		}
	})

	t.Run("respects max depth", func(t *testing.T) {
		opts := &ExpandOptions{
			MaxDepth: 1,
			MaxFiles: 50,
		}
		expanded, err := ExpandWithDependencies(db, []string{"src/main.go"}, opts)
		if err != nil {
			t.Fatalf("ExpandWithDependencies() error = %v", err)
		}

		// Should not include depth-2 files like loader.go, helpers.go
		for _, ef := range expanded {
			if ef.Depth > 1 {
				t.Errorf("found file at depth %d, max was 1: %s", ef.Depth, ef.Path)
			}
		}
	})

	t.Run("respects max files", func(t *testing.T) {
		opts := &ExpandOptions{
			MaxDepth: 10,
			MaxFiles: 3,
		}
		expanded, err := ExpandWithDependencies(db, []string{"src/main.go"}, opts)
		if err != nil {
			t.Fatalf("ExpandWithDependencies() error = %v", err)
		}

		if len(expanded) > 3 {
			t.Errorf("got %d files, max was 3", len(expanded))
		}
	})

	t.Run("sorted by depth", func(t *testing.T) {
		expanded, err := ExpandWithDependencies(db, []string{"src/main.go"}, nil)
		if err != nil {
			t.Fatalf("ExpandWithDependencies() error = %v", err)
		}

		prevDepth := 0
		for _, ef := range expanded {
			if ef.Depth < prevDepth {
				t.Errorf("files not sorted by depth: %s at depth %d came after depth %d", ef.Path, ef.Depth, prevDepth)
			}
			prevDepth = ef.Depth
		}
	})
}

func TestGetFileUsageScores(t *testing.T) {
	db := setupSmartContextTestDB(t)
	defer db.Close()

	t.Run("computes usage scores", func(t *testing.T) {
		files := []string{
			"src/utils/helpers.go",
			"src/models/user.go",
			"src/main.go",
		}

		scores, err := GetFileUsageScores(db, files)
		if err != nil {
			t.Fatalf("GetFileUsageScores() error = %v", err)
		}

		// helpers.go is imported by auth.go and user.go, and its functions are called
		helpersScore := scores["src/utils/helpers.go"]
		if helpersScore == nil {
			t.Fatal("missing score for helpers.go")
		}
		if helpersScore.ImportedBy < 2 {
			t.Errorf("helpers.go ImportedBy = %d, want >= 2", helpersScore.ImportedBy)
		}

		// user.go model is used by auth.go and user.go
		userScore := scores["src/models/user.go"]
		if userScore == nil {
			t.Fatal("missing score for models/user.go")
		}
		if userScore.ImportedBy < 2 {
			t.Errorf("models/user.go ImportedBy = %d, want >= 2", userScore.ImportedBy)
		}
	})

	t.Run("scores are normalized", func(t *testing.T) {
		files := []string{
			"src/utils/helpers.go",
			"src/models/user.go",
			"src/main.go",
		}

		scores, err := GetFileUsageScores(db, files)
		if err != nil {
			t.Fatalf("GetFileUsageScores() error = %v", err)
		}

		for path, score := range scores {
			if score.UsageScore < 0 || score.UsageScore > 1 {
				t.Errorf("%s UsageScore = %f, want 0-1", path, score.UsageScore)
			}
		}
	})
}

func TestGetMostImportedFiles(t *testing.T) {
	db := setupSmartContextTestDB(t)
	defer db.Close()

	t.Run("returns most imported files", func(t *testing.T) {
		results, err := GetMostImportedFiles(db, 10)
		if err != nil {
			t.Fatalf("GetMostImportedFiles() error = %v", err)
		}

		if len(results) == 0 {
			t.Fatal("expected some results")
		}

		// helpers.go and models/user.go should be top imported
		topPaths := make(map[string]bool)
		for i := 0; i < 3 && i < len(results); i++ {
			topPaths[results[i].Path] = true
		}

		// At least one of the commonly imported files should be in top 3
		hasExpected := topPaths["src/utils/helpers.go"] || topPaths["src/models/user.go"]
		if !hasExpected {
			t.Errorf("expected helpers.go or models/user.go in top results, got: %v", topPaths)
		}
	})
}

func TestComputeSmartContext(t *testing.T) {
	db := setupSmartContextTestDB(t)
	defer db.Close()

	t.Run("combines all scoring factors", func(t *testing.T) {
		// Create sample initial files
		initialFiles := []FileContext{
			{Path: "src/handlers/auth.go", Relevance: 0.9},
			{Path: "src/handlers/user.go", Relevance: 0.8},
		}

		// Create edit history
		editHistory := map[string]*FileEditInfo{
			"src/handlers/auth.go": {
				Path:       "src/handlers/auth.go",
				EditCount:  10,
				LastEdited: time.Now().Add(-2 * time.Hour), // Recent
			},
			"src/utils/helpers.go": {
				Path:       "src/utils/helpers.go",
				EditCount:  5,
				LastEdited: time.Now().Add(-1 * time.Hour), // Very recent
			},
		}

		opts := DefaultSmartContextOptions()
		result, err := ComputeSmartContext(db, initialFiles, editHistory, opts)
		if err != nil {
			t.Fatalf("ComputeSmartContext() error = %v", err)
		}

		// Should have expanded to include dependencies
		if result.Stats.ExpandedFiles == 0 {
			t.Error("expected some expanded files")
		}

		// Total should be seed + expanded
		if result.Stats.TotalFiles != result.Stats.SeedFiles+result.Stats.ExpandedFiles {
			t.Errorf("TotalFiles = %d, SeedFiles = %d, ExpandedFiles = %d",
				result.Stats.TotalFiles, result.Stats.SeedFiles, result.Stats.ExpandedFiles)
		}

		// Files should be sorted by final score
		for i := 1; i < len(result.Files); i++ {
			if result.Files[i].FinalScore > result.Files[i-1].FinalScore {
				t.Error("files not sorted by final score")
				break
			}
		}
	})

	t.Run("respects options", func(t *testing.T) {
		initialFiles := []FileContext{
			{Path: "src/main.go", Relevance: 1.0},
		}

		// Disable all smart features
		opts := &SmartContextOptions{
			ExpandDependencies: false,
			PrioritizeByUsage:  false,
			BoostRecentEdits:   false,
			MaxFiles:           50,
			RelevanceWeight:    1.0,
		}

		result, err := ComputeSmartContext(db, initialFiles, nil, opts)
		if err != nil {
			t.Fatalf("ComputeSmartContext() error = %v", err)
		}

		// Should only have seed file, no expansion
		if len(result.Files) != 1 {
			t.Errorf("expected 1 file without expansion, got %d", len(result.Files))
		}

		// All scores except relevance should be zero
		for _, f := range result.Files {
			if f.UsageScore != 0 || f.RecencyScore != 0 {
				t.Errorf("expected zero usage/recency scores when disabled, got usage=%f recency=%f",
					f.UsageScore, f.RecencyScore)
			}
		}
	})

	t.Run("recency scoring works", func(t *testing.T) {
		initialFiles := []FileContext{
			{Path: "src/handlers/auth.go", Relevance: 1.0},
		}

		now := time.Now()
		editHistory := map[string]*FileEditInfo{
			"src/handlers/auth.go": {
				Path:       "src/handlers/auth.go",
				EditCount:  5,
				LastEdited: now.Add(-1 * time.Hour), // 1 hour ago
			},
		}

		opts := &SmartContextOptions{
			ExpandDependencies: false,
			PrioritizeByUsage:  false,
			BoostRecentEdits:   true,
			RecentEditWindow:   7 * 24 * time.Hour,
			RecencyWeight:      1.0,
			MaxFiles:           50,
		}

		result, err := ComputeSmartContext(db, initialFiles, editHistory, opts)
		if err != nil {
			t.Fatalf("ComputeSmartContext() error = %v", err)
		}

		if len(result.Files) == 0 {
			t.Fatal("expected at least one file")
		}

		// File edited 1 hour ago should have high recency score
		if result.Files[0].RecencyScore < 0.9 {
			t.Errorf("expected high recency score for recently edited file, got %f", result.Files[0].RecencyScore)
		}
	})
}

func TestQuickScoreFiles(t *testing.T) {
	db := setupSmartContextTestDB(t)
	defer db.Close()

	t.Run("quick scores files", func(t *testing.T) {
		files := []string{
			"src/handlers/auth.go",
			"src/utils/helpers.go",
			"src/main.go",
		}

		editHistory := map[string]*FileEditInfo{
			"src/handlers/auth.go": {
				Path:       "src/handlers/auth.go",
				LastEdited: time.Now().Add(-1 * time.Hour),
			},
		}

		scores, err := QuickScoreFiles(db, files, editHistory)
		if err != nil {
			t.Fatalf("QuickScoreFiles() error = %v", err)
		}

		if len(scores) != 3 {
			t.Errorf("expected 3 scores, got %d", len(scores))
		}

		// Should be sorted by final score
		for i := 1; i < len(scores); i++ {
			if scores[i].FinalScore > scores[i-1].FinalScore {
				t.Error("scores not sorted by final score")
				break
			}
		}
	})
}

func TestGetImportGraph(t *testing.T) {
	db := setupSmartContextTestDB(t)
	defer db.Close()

	t.Run("returns import graph", func(t *testing.T) {
		files := []string{"src/main.go", "src/handlers/auth.go"}

		graph, err := GetImportGraph(db, files)
		if err != nil {
			t.Fatalf("GetImportGraph() error = %v", err)
		}

		// main.go should have imports
		mainEntry := graph["src/main.go"]
		if mainEntry == nil {
			t.Fatal("missing main.go in graph")
		}
		if len(mainEntry.Imports) != 3 {
			t.Errorf("main.go imports = %d, want 3", len(mainEntry.Imports))
		}

		// auth.go should be imported by main.go
		authEntry := graph["src/handlers/auth.go"]
		if authEntry == nil {
			t.Fatal("missing auth.go in graph")
		}
		// Note: ImportedBy is from perspective of this query's files only
	})
}

func TestGetRelatedFilesBySymbol(t *testing.T) {
	db := setupSmartContextTestDB(t)
	defer db.Close()

	t.Run("finds files that call symbol", func(t *testing.T) {
		// FormatDate is called from auth.go and user.go
		results, err := GetRelatedFilesBySymbol(db, "FormatDate", 10)
		if err != nil {
			t.Fatalf("GetRelatedFilesBySymbol() error = %v", err)
		}

		if len(results) < 2 {
			t.Errorf("expected at least 2 files calling FormatDate, got %d", len(results))
		}

		// Check that files calling FormatDate are included
		found := make(map[string]bool)
		for _, r := range results {
			found[r.Path] = true
		}

		if !found["src/handlers/auth.go"] || !found["src/handlers/user.go"] {
			t.Errorf("expected auth.go and user.go in results, got: %v", found)
		}
	})
}
