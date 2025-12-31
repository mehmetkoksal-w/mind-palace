package index

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestShouldExcludeFile(t *testing.T) {
	tests := []struct {
		path     string
		patterns []string
		expected bool
	}{
		{"main.go", []string{"*.go"}, true},
		{"main.go", []string{"*.ts"}, false},
		{"vendor/pkg/file.go", []string{"vendor/*"}, true},
		{"internal/pkg/file_test.go", DefaultExcludePatterns, true},
		{"src/app.ts", DefaultExcludePatterns, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%v", tt.path, tt.patterns), func(t *testing.T) {
			if got := shouldExcludeFile(tt.path, tt.patterns); got != tt.expected {
				t.Errorf("shouldExcludeFile(%q, %v) = %v, want %v", tt.path, tt.patterns, got, tt.expected)
			}
		})
	}
}

func TestOracleIntegrated(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open memory DB: %v", err)
	}
	defer db.Close()

	// Initialize schema
	stmts := []string{
		`CREATE TABLE files (path TEXT PRIMARY KEY, hash TEXT, size INTEGER, mod_time TEXT, indexed_at TEXT, language TEXT);`,
		`CREATE TABLE chunks (id INTEGER PRIMARY KEY, path TEXT, chunk_index INTEGER, start_line INTEGER, end_line INTEGER, content TEXT);`,
		`CREATE VIRTUAL TABLE chunks_fts USING fts5(path, content, chunk_index);`,
		`CREATE TABLE symbols (id INTEGER PRIMARY KEY, file_path TEXT, name TEXT, kind TEXT, line_start INTEGER, line_end INTEGER, signature TEXT, doc_comment TEXT, parent_id INTEGER, exported INTEGER);`,
		`CREATE VIRTUAL TABLE symbols_fts USING fts5(name, file_path, kind, doc_comment);`,
		`CREATE TABLE relationships (id INTEGER PRIMARY KEY, source_file TEXT, source_symbol_id INTEGER, target_file TEXT, target_symbol TEXT, kind TEXT, line INTEGER, column INTEGER);`,
		`CREATE TABLE decisions (id TEXT PRIMARY KEY, room TEXT, title TEXT, summary TEXT, rationale TEXT, affected_files TEXT, created_at TEXT, created_by TEXT);`,
	}
	for _, s := range stmts {
		db.Exec(s)
	}

	// Insert test data
	db.Exec(`INSERT INTO files VALUES (?, ?, ?, ?, ?, ?);`, "auth.go", "h1", 100, "now", "now", "go")
	db.Exec(`INSERT INTO symbols VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`, 1, "auth.go", "Login", "function", 1, 10, "()", "Login function", nil, 1)
	db.Exec(`INSERT INTO symbols_fts VALUES (?, ?, ?, ?);`, "Login", "auth.go", "function", "Login function")
	db.Exec(`INSERT INTO relationships VALUES (?, ?, ?, ?, ?, ?, ?, ?);`, 1, "main.go", nil, "auth.go", nil, "import", 5, 1)

	t.Run("GetSymbol", func(t *testing.T) {
		sym, err := GetSymbol(db, "Login", "auth.go")
		if err != nil {
			t.Fatalf("GetSymbol failed: %v", err)
		}
		if sym.Name != "Login" {
			t.Errorf("Expected Login, got %s", sym.Name)
		}
	})

	t.Run("SearchSymbolsByKind", func(t *testing.T) {
		syms, err := SearchSymbolsByKind(db, "function", 10)
		if err != nil {
			t.Fatalf("SearchSymbolsByKind failed: %v", err)
		}
		if len(syms) == 0 {
			t.Error("Expected 1 symbol, got 0")
		}
	})

	t.Run("ListExportedSymbols", func(t *testing.T) {
		syms, err := ListExportedSymbols(db, "auth.go")
		if err != nil {
			t.Fatalf("ListExportedSymbols failed: %v", err)
		}
		if len(syms) == 0 {
			t.Error("Expected 1 exported symbol, got 0")
		}
	})

	t.Run("GetImpact", func(t *testing.T) {
		impact, err := GetImpact(db, "auth.go")
		if err != nil {
			t.Fatalf("GetImpact failed: %v", err)
		}
		if len(impact.Dependents) == 0 {
			t.Error("Expected 1 dependent (main.go), got 0")
		}
	})

	t.Run("RecordDecision", func(t *testing.T) {
		d := Decision{
			ID:        "d1",
			Title:     "Use OAuth",
			CreatedAt: time.Now().Format(time.RFC3339),
		}
		err := RecordDecision(db, d)
		if err != nil {
			t.Errorf("RecordDecision failed: %v", err)
		}
	})

	t.Run("GetContextForTaskWithOptions", func(t *testing.T) {
		// Insert chunk data
		db.Exec(`INSERT INTO chunks VALUES (?, ?, ?, ?, ?, ?);`, 2, "main.go", 0, 1, 5, "package main\nimport \"auth\"")
		db.Exec(`INSERT INTO chunks_fts VALUES (?, ?, ?);`, "main.go", "package main import auth", 0)

		ctx, err := GetContextForTaskWithOptions(db, "Login", 10, &ContextOptions{
			IncludeTests: true,
		})
		if err != nil {
			t.Fatalf("GetContextForTaskWithOptions failed: %v", err)
		}
		if len(ctx.Symbols) == 0 {
			t.Error("Expected matched symbol 'Login'")
		}
		if len(ctx.Files) == 0 {
			t.Error("Expected matched file 'auth.go'")
		}
	})

	t.Run("GetContextForTask", func(t *testing.T) {
		ctx, err := GetContextForTask(db, "Login", 10)
		if err != nil {
			t.Fatalf("GetContextForTask failed: %v", err)
		}
		if len(ctx.Symbols) == 0 {
			t.Error("Expected matched symbol 'Login'")
		}
	})
}

func TestApplyTokenBudgetTruncation(t *testing.T) {
	result := &ContextResult{
		Symbols: []SymbolInfo{
			{Name: "A", Kind: "function", FilePath: "a.go"},
			{Name: "B", Kind: "function", FilePath: "b.go"},
		},
		Files: []FileContext{
			{Path: "a.go", Language: "go", Snippet: "package main\nfunc A() {}"},
			{Path: "b.go", Language: "go", Snippet: "package main\nfunc B() {}"},
		},
		Imports: []ImportInfo{
			{SourceFile: "a.go", TargetFile: "fmt", Kind: "import"},
			{SourceFile: "b.go", TargetFile: "os", Kind: "import"},
		},
	}

	truncated := applyTokenBudget(result, 10)
	if truncated.TokenStats == nil || !truncated.TokenStats.Truncated {
		t.Fatalf("expected token budget truncation")
	}
	if len(truncated.Imports) > 1 {
		t.Fatalf("expected imports to be truncated")
	}
}

func TestOracleHelpers(t *testing.T) {
	sym := SymbolInfo{Name: "DoWork", Kind: "function", FilePath: "main.go", Signature: "func DoWork()"}
	if estimateSymbolTokens(sym) == 0 {
		t.Fatalf("expected non-zero token estimate")
	}

	files := []FileContext{
		{Path: "a.go", Language: "go", Snippet: "short", Relevance: 1.0},
		{Path: "b.go", Language: "go", Snippet: "long long long long", Relevance: 0.1},
	}
	truncated := truncateFileContexts(files, 50)
	if len(truncated) == 0 {
		t.Fatalf("expected some file contexts")
	}

	imports := []ImportInfo{
		{SourceFile: "a.go", TargetFile: "fmt", Kind: "import"},
	}
	if got := truncateImports(imports, 0); len(got) != len(imports) {
		t.Fatalf("expected imports unchanged when budget <= 0")
	}

	if out := truncateSnippet("abcdef", 3); out != "abc..." {
		t.Fatalf("truncateSnippet() = %q, want abc...", out)
	}
}
