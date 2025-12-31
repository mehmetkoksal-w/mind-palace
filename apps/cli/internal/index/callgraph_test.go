package index

import (
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestCallGraph(t *testing.T) {
	// Use index.Open to get a real DB with real schema
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Insert test data using real table names
	now := time.Now().UTC().Format(time.RFC3339)
	db.Exec(`INSERT INTO files(path, hash, size, mod_time, indexed_at, language) VALUES (?, ?, ?, ?, ?, ?);`, "main.go", "h1", 100, now, now, "go")
	db.Exec(`INSERT INTO symbols(id, file_path, name, kind, line_start, line_end, signature, doc_comment, parent_id, exported) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`, 1, "main.go", "main", "function", 1, 10, "()", "", nil, 1)
	db.Exec(`INSERT INTO symbols(id, file_path, name, kind, line_start, line_end, signature, doc_comment, parent_id, exported) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`, 2, "main.go", "helper", "function", 11, 20, "()", "", nil, 1)

	// 'call' from helper (line 15) to lib.Func
	db.Exec(`INSERT INTO relationships(source_file, source_symbol_id, target_symbol, kind, line, column) VALUES (?, ?, ?, ?, ?, ?);`, "main.go", 2, "lib.Func", "call", 15, 1)
	// 'call' from main (line 5) to helper
	db.Exec(`INSERT INTO relationships(source_file, source_symbol_id, target_symbol, kind, line, column) VALUES (?, ?, ?, ?, ?, ?);`, "main.go", 1, "helper", "call", 5, 1)

	t.Run("GetIncomingCalls", func(t *testing.T) {
		calls, err := GetIncomingCalls(db, "helper")
		if err != nil {
			t.Fatalf("GetIncomingCalls failed: %v", err)
		}
		if len(calls) == 0 {
			t.Error("Expected 1 incoming call to 'helper'")
		}
		if calls[0].CallerSymbol != "main" {
			t.Errorf("Expected caller 'main', got %q", calls[0].CallerSymbol)
		}
	})

	t.Run("GetOutgoingCalls", func(t *testing.T) {
		calls, err := GetOutgoingCalls(db, "helper", "main.go")
		if err != nil {
			t.Fatalf("GetOutgoingCalls failed: %v", err)
		}
		if len(calls) == 0 {
			t.Error("Expected 1 outgoing call from 'helper'")
		}
		if calls[0].CalleeSymbol != "lib.Func" {
			t.Errorf("Expected callee 'lib.Func', got %q", calls[0].CalleeSymbol)
		}
	})

	t.Run("GetCallGraph", func(t *testing.T) {
		graph, err := GetCallGraph(db, "main.go")
		if err != nil {
			t.Fatalf("GetCallGraph failed: %v", err)
		}
		if len(graph.OutgoingCalls) == 0 {
			t.Error("Expected outgoing calls in graph")
		}
	})

	t.Run("GetCallersCount", func(t *testing.T) {
		count, err := GetCallersCount(db, "helper")
		if err != nil {
			t.Fatalf("GetCallersCount failed: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected count 1, got %d", count)
		}
	})

	t.Run("GetMostCalledSymbols", func(t *testing.T) {
		res, err := GetMostCalledSymbols(db, 10)
		if err != nil {
			t.Fatalf("GetMostCalledSymbols failed: %v", err)
		}
		if len(res) == 0 {
			t.Error("Expected results")
		}
	})
}
