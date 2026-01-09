package butler

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
)

func TestPreprocessQuery(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		mustContain  []string // Substrings the result must contain
		mustNotEqual string   // The result must not equal this (for non-empty)
	}{
		{
			name:         "empty query",
			query:        "",
			mustContain:  nil,
			mustNotEqual: "something",
		},
		{
			name:         "whitespace only",
			query:        "   ",
			mustContain:  nil,
			mustNotEqual: "something",
		},
		{
			name:         "code-like with underscore - tokenized and expanded",
			query:        "func_name",
			mustContain:  []string{`"func"`, `"name"`, `"func_name"`}, // Now tokenized
			mustNotEqual: "",
		},
		{
			name:         "code-like with dot - exact match",
			query:        "Class.method",
			mustContain:  []string{`"Class.method"`}, // Exact phrase for code symbols
			mustNotEqual: "",
		},
		{
			name:         "code-like with double colon",
			query:        "pkg::path",
			mustContain:  []string{`"pkg::path"`},
			mustNotEqual: "",
		},
		{
			name:         "natural language single word with synonyms",
			query:        "authentication",
			mustContain:  []string{`"authentication"`}, // Original term included
			mustNotEqual: "",
		},
		{
			name:         "natural language multiple words with synonyms",
			query:        "where is auth",
			mustContain:  []string{`"where"`, `"auth"`, `"authentication"`}, // Synonym expansion
			mustNotEqual: "",
		},
		{
			name:         "short words filtered",
			query:        "a b search",
			mustContain:  []string{`"search"`},
			mustNotEqual: "",
		},
		{
			name:         "CamelCase tokenization",
			query:        "handleAuth function",
			mustContain:  []string{`"handle"`, `"auth"`, `"handleAuth"`, `"function"`}, // CamelCase split
			mustNotEqual: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := preprocessQuery(tt.query)

			// Check empty case
			if len(tt.mustContain) == 0 {
				if result != "" {
					t.Errorf("preprocessQuery(%q) = %q, expected empty", tt.query, result)
				}
				return
			}

			// Check that result contains expected substrings
			for _, substr := range tt.mustContain {
				if !contains(result, substr) {
					t.Errorf("preprocessQuery(%q) = %q, expected to contain %q", tt.query, result, substr)
				}
			}
		})
	}
}

func TestPreprocessQueryWithFuzzy(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		mustContain []string
	}{
		{
			name:        "word with fuzzy variants",
			query:       "search",
			mustContain: []string{`"search"`},
		},
		{
			name:        "empty query",
			query:       "",
			mustContain: nil,
		},
		{
			name:        "CamelCase with fuzzy",
			query:       "ProcessData",
			mustContain: []string{`"ProcessData"`, `"process"`, `"data"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := preprocessQueryWithFuzzy(tt.query)

			if len(tt.mustContain) == 0 {
				if result != "" {
					t.Errorf("preprocessQueryWithFuzzy(%q) = %q, expected empty", tt.query, result)
				}
				return
			}

			for _, substr := range tt.mustContain {
				if !contains(result, substr) {
					t.Errorf("preprocessQueryWithFuzzy(%q) = %q, expected to contain %q", tt.query, result, substr)
				}
			}
		})
	}
}

func TestPreprocessQueryWithOptions(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		useSynonyms bool
		expectEmpty bool
	}{
		{
			name:        "with synonyms",
			query:       "auth",
			useSynonyms: true,
			expectEmpty: false,
		},
		{
			name:        "without synonyms",
			query:       "auth",
			useSynonyms: false,
			expectEmpty: false,
		},
		{
			name:        "empty query",
			query:       "",
			useSynonyms: true,
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := preprocessQueryWithOptions(tt.query, tt.useSynonyms)

			if tt.expectEmpty && result != "" {
				t.Errorf("preprocessQueryWithOptions(%q, %v) = %q, expected empty", tt.query, tt.useSynonyms, result)
			}
			if !tt.expectEmpty && result == "" {
				t.Errorf("preprocessQueryWithOptions(%q, %v) = empty, expected non-empty", tt.query, tt.useSynonyms)
			}
		})
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || substr == "" ||
		(s != "" && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCalculateScore(t *testing.T) {
	b := &Butler{
		entryPoints: map[string]string{
			"README.md": "project-overview",
		},
	}

	tests := []struct {
		name        string
		baseScore   float64
		path        string
		query       string
		expectBoost bool
	}{
		{
			name:        "entry point boost",
			baseScore:   -1.0,
			path:        "README.md",
			query:       "test",
			expectBoost: true,
		},
		{
			name:        "path match boost",
			baseScore:   -1.0,
			path:        "internal/auth/handler.go",
			query:       "auth",
			expectBoost: true,
		},
		{
			name:        "no boost",
			baseScore:   -1.0,
			path:        "config.json",
			query:       "test",
			expectBoost: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := b.calculateScore(tt.baseScore, tt.path, tt.query)

			basePositive := 1.0

			if tt.expectBoost {
				if score <= basePositive*1.2 { // Minimum boost is 1.2 for code files
					t.Errorf("Expected boost for %s with query %q, got score %.2f", tt.path, tt.query, score)
				}
			}
		})
	}
}

func TestGroupByRoom(t *testing.T) {
	b := &Butler{
		rooms: map[string]model.Room{
			"auth": {Name: "auth", Summary: "Authentication module"},
		},
		entryPoints: map[string]string{},
	}

	results := []SearchResult{
		{Path: "auth/handler.go", Room: "auth", Score: 10.0},
		{Path: "auth/service.go", Room: "auth", Score: 8.0},
		{Path: "main.go", Room: "", Score: 5.0},
	}

	grouped := b.groupByRoom(results)

	if len(grouped) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(grouped))
	}

	if grouped[0].Room != "auth" {
		t.Errorf("Expected first group to be 'auth', got %q", grouped[0].Room)
	}

	if len(grouped[0].Results) != 2 {
		t.Errorf("Expected 2 results in 'auth' group, got %d", len(grouped[0].Results))
	}
}

func TestInferRoom(t *testing.T) {
	b := &Butler{
		rooms: map[string]model.Room{
			"auth": {
				Name:        "auth",
				EntryPoints: []string{"internal/auth/**"},
			},
			"api": {
				Name:        "api",
				EntryPoints: []string{"internal/api/**"},
			},
		},
		entryPoints: map[string]string{
			"internal/auth/handler.go": "auth",
			"internal/api/routes.go":   "api",
		},
	}

	tests := []struct {
		path     string
		expected string
	}{
		{"internal/auth/handler.go", "auth"},
		{"internal/api/routes.go", "api"},
		{"unknown/path.go", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := b.inferRoom(tt.path)
			if result != tt.expected {
				t.Errorf("inferRoom(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestListRooms(t *testing.T) {
	b := &Butler{
		rooms: map[string]model.Room{
			"auth": {Name: "auth", Summary: "Auth module"},
			"api":  {Name: "api", Summary: "API module"},
		},
	}

	rooms := b.ListRooms()

	if len(rooms) != 2 {
		t.Errorf("Expected 2 rooms, got %d", len(rooms))
	}

	// Check that rooms are sorted by name
	names := make([]string, len(rooms))
	for i, r := range rooms {
		names[i] = r.Name
	}

	for i := 0; i < len(names)-1; i++ {
		if names[i] > names[i+1] {
			t.Errorf("Rooms not sorted: %v", names)
			break
		}
	}
}

func TestReadRoom(t *testing.T) {
	b := &Butler{
		rooms: map[string]model.Room{
			"auth": {Name: "auth", Summary: "Auth module"},
		},
	}

	t.Run("existing room", func(t *testing.T) {
		room, err := b.ReadRoom("auth")
		if err != nil {
			t.Fatalf("ReadRoom failed: %v", err)
		}
		if room.Name != "auth" {
			t.Errorf("Expected room name 'auth', got %q", room.Name)
		}
	})

	t.Run("non-existing room", func(t *testing.T) {
		_, err := b.ReadRoom("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existing room")
		}
	})
}

func TestSearchResult(t *testing.T) {
	result := SearchResult{
		Path:       "src/auth/handler.go",
		Room:       "auth",
		ChunkIndex: 0,
		StartLine:  10,
		EndLine:    20,
		Snippet:    "func HandleAuth() { ... }",
		Score:      10.5,
		IsEntry:    true,
	}

	if result.Path != "src/auth/handler.go" {
		t.Error("Path not set correctly")
	}
	if result.Room != "auth" {
		t.Error("Room not set correctly")
	}
	if result.ChunkIndex != 0 {
		t.Error("ChunkIndex not set correctly")
	}
	if result.StartLine != 10 {
		t.Error("StartLine not set correctly")
	}
	if result.EndLine != 20 {
		t.Error("EndLine not set correctly")
	}
	if result.Snippet != "func HandleAuth() { ... }" {
		t.Error("Snippet not set correctly")
	}
	if result.Score != 10.5 {
		t.Error("Score not set correctly")
	}
	if !result.IsEntry {
		t.Error("IsEntry not set correctly")
	}
}

func TestGroupedResults(t *testing.T) {
	grouped := GroupedResults{
		Room:    "auth",
		Summary: "Authentication module",
		Results: []SearchResult{
			{Path: "auth/handler.go", Score: 10.0},
			{Path: "auth/service.go", Score: 8.0},
		},
	}

	if grouped.Room != "auth" {
		t.Error("Room not set correctly")
	}
	if grouped.Summary != "Authentication module" {
		t.Error("Summary not set correctly")
	}
	if len(grouped.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(grouped.Results))
	}
}

func TestSearchOptions(t *testing.T) {
	opts := SearchOptions{
		Limit:      20,
		RoomFilter: "auth",
		FuzzyMatch: true,
	}

	if opts.Limit != 20 {
		t.Error("Limit not set correctly")
	}
	if opts.RoomFilter != "auth" {
		t.Error("RoomFilter not set correctly")
	}
	if !opts.FuzzyMatch {
		t.Error("FuzzyMatch not set correctly")
	}
}

func TestEnhancedContextOptions(t *testing.T) {
	opts := EnhancedContextOptions{
		Query:            "auth handler",
		Limit:            10,
		MaxTokens:        4096,
		IncludeTests:     true,
		IncludeLearnings: true,
		IncludeFileIntel: true,
		SessionID:        "session-123",
	}

	if opts.Query != "auth handler" {
		t.Error("Query not set correctly")
	}
	if opts.Limit != 10 {
		t.Error("Limit not set correctly")
	}
	if opts.MaxTokens != 4096 {
		t.Error("MaxTokens not set correctly")
	}
	if !opts.IncludeTests {
		t.Error("IncludeTests not set correctly")
	}
	if !opts.IncludeLearnings {
		t.Error("IncludeLearnings not set correctly")
	}
	if !opts.IncludeFileIntel {
		t.Error("IncludeFileIntel not set correctly")
	}
	if opts.SessionID != "session-123" {
		t.Error("SessionID not set correctly")
	}
}

func TestSetJSONCDecoder(t *testing.T) {
	// Save original
	original := jsonCDecode

	// Set custom decoder
	customDecoder := func(path string, v interface{}) error {
		return nil
	}

	SetJSONCDecoder(customDecoder)

	// Verify it was set
	if jsonCDecode == nil {
		t.Error("jsonCDecode should not be nil after SetJSONCDecoder")
	}

	// Restore original
	jsonCDecode = original
}

func TestButlerIntegrated(t *testing.T) {
	// Initialize in-memory DB
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open memory DB: %v", err)
	}
	defer db.Close()

	// Initialize schema manually for testing (mimicking index.indexMigrateV0)
	stmts := []string{
		`CREATE TABLE files (path TEXT PRIMARY KEY, hash TEXT, size INTEGER, mod_time TEXT, indexed_at TEXT, language TEXT);`,
		`CREATE TABLE chunks (id INTEGER PRIMARY KEY, path TEXT, chunk_index INTEGER, start_line INTEGER, end_line INTEGER, content TEXT);`,
		`CREATE VIRTUAL TABLE chunks_fts USING fts5(path, content, chunk_index, tokenize="unicode61 tokenchars '_.:@#$-'");`,
		`CREATE TABLE symbols (id INTEGER PRIMARY KEY, file_path TEXT, name TEXT, kind TEXT, line_start INTEGER, line_end INTEGER, signature TEXT, doc_comment TEXT, parent_id INTEGER, exported INTEGER);`,
		`CREATE VIRTUAL TABLE symbols_fts USING fts5(name, file_path, kind, doc_comment, tokenize="unicode61 tokenchars '_'");`,
		`CREATE TABLE scans (id INTEGER PRIMARY KEY, root TEXT, scan_hash TEXT, started_at TEXT, completed_at TEXT);`,
		`CREATE TABLE rooms (name TEXT PRIMARY KEY, summary TEXT, entry_points TEXT, file_patterns TEXT, updated_at TEXT);`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(context.Background(), s); err != nil {
			t.Fatalf("Failed to init schema: %v", err)
		}
	}

	// Insert test data
	now := time.Now().UTC().Format(time.RFC3339)
	db.ExecContext(context.Background(), `INSERT INTO files VALUES (?, ?, ?, ?, ?, ?);`, "auth.go", "h1", 100, now, now, "go")
	db.ExecContext(context.Background(), `INSERT INTO chunks VALUES (?, ?, ?, ?, ?, ?);`, 1, "auth.go", 0, 1, 10, "func HandleAuth() {}")
	db.ExecContext(context.Background(), `INSERT INTO chunks_fts VALUES (?, ?, ?);`, "auth.go", "func HandleAuth() {}", 0)
	db.ExecContext(context.Background(), `INSERT INTO symbols VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`, 1, "auth.go", "HandleAuth", "function", 1, 10, "()", "Auth handler", nil, 1)
	db.ExecContext(context.Background(), `INSERT INTO symbols_fts VALUES (?, ?, ?, ?);`, "HandleAuth", "auth.go", "function", "Auth handler")
	db.ExecContext(context.Background(), `INSERT INTO rooms VALUES (?, ?, ?, ?, ?);`, "auth", "Auth module", `["auth.go"]`, `[]`, now)

	b := &Butler{
		db:          db,
		root:        "/tmp",
		rooms:       make(map[string]model.Room),
		entryPoints: make(map[string]string),
	}
	b.rooms["auth"] = model.Room{Name: "auth", Summary: "Auth module", EntryPoints: []string{"auth.go"}}
	b.entryPoints["auth.go"] = "auth"

	t.Run("Search", func(t *testing.T) {
		results, err := b.Search("HandleAuth", SearchOptions{Limit: 10})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) == 0 {
			t.Error("Expected search results, got 0")
		}
	})

	t.Run("ListRooms", func(t *testing.T) {
		rooms := b.ListRooms()
		if len(rooms) != 1 {
			t.Errorf("Expected 1 room, got %d", len(rooms))
		}
	})

	t.Run("ReadFile", func(t *testing.T) {
		content, err := b.ReadFile("auth.go")
		if err != nil {
			t.Fatalf("ReadFile failed: %v", err)
		}
		if content != "func HandleAuth() {}" {
			t.Errorf("Unexpected content: %q", content)
		}
	})

	t.Run("GetIndexInfo", func(t *testing.T) {
		info := b.GetIndexInfo()
		if info == nil {
			t.Fatal("Expected IndexInfo, got nil")
		}
		if info.FileCount != 1 {
			t.Errorf("Expected 1 file, got %d", info.FileCount)
		}
	})

	// Test Memory Wrappers
	tmpRoot := t.TempDir()
	mem, err := memory.Open(tmpRoot)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	defer mem.Close()
	b.memory = mem

	t.Run("Session", func(t *testing.T) {
		sess, err := b.StartSession("coder", "id1", "fix bug")
		if err != nil {
			t.Fatalf("StartSession failed: %v", err)
		}
		if sess.Goal != "fix bug" {
			t.Errorf("Expected goal 'fix bug', got %q", sess.Goal)
		}

		err = b.LogActivity(sess.ID, memory.Activity{Kind: "edit", Target: "auth.go"})
		if err != nil {
			t.Errorf("LogActivity failed: %v", err)
		}

		retrieved, err := b.GetSession(sess.ID)
		if err != nil {
			t.Fatalf("GetSession failed: %v", err)
		}
		if retrieved.ID != sess.ID {
			t.Error("Session ID mismatch")
		}

		sessions, _ := b.ListSessions(false, 10)
		if len(sessions) == 0 {
			t.Error("Expected sessions in list")
		}
	})

	t.Run("Learnings", func(t *testing.T) {
		id, err := b.AddLearning(memory.Learning{Content: "Go is fast", Scope: "lang"})
		if err != nil {
			t.Fatalf("AddLearning failed: %v", err)
		}
		if id == "" {
			t.Error("Expected learning ID")
		}

		learnings, err := b.GetLearnings("lang", "", 10)
		if err != nil || len(learnings) == 0 {
			t.Error("Expected learning in result")
		}

		found, err := b.SearchLearnings("Go", 10)
		if err != nil || len(found) == 0 {
			t.Error("Expected learning in search")
		}

		err = b.ReinforceLearning(id)
		if err != nil {
			t.Errorf("ReinforceLearning failed: %v", err)
		}
	})

	t.Run("Intelligence", func(t *testing.T) {
		b.RecordFileEdit("auth.go", "coder")
		intel, err := b.GetFileIntel("auth.go")
		if err != nil || intel == nil {
			t.Error("Expected file intel")
		}
		if intel.EditCount != 1 {
			t.Errorf("Expected 1 edit, got %d", intel.EditCount)
		}
	})

	t.Run("Agents", func(t *testing.T) {
		// Advance heartbeat for active agent test
		db.ExecContext(context.Background(), "UPDATE agents SET last_heartbeat = ?", time.Now().UTC().Format(time.RFC3339))
		agents, err := b.GetActiveAgents()
		if err != nil {
			t.Errorf("GetActiveAgents failed: %v", err)
		}
		// might be empty if we didn't start one correctly in this context
		t.Logf("Active agents: %v", agents)
	})

	t.Run("Briefing", func(t *testing.T) {
		brief, err := b.GetBrief("auth.go")
		if err != nil {
			t.Fatalf("GetBrief failed: %v", err)
		}
		if brief.FilePath != "auth.go" {
			t.Error("FilePath mismatch in briefing")
		}
	})
}

func TestGetIndexInfoStale(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open memory DB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	stmts := []string{
		`CREATE TABLE files (path TEXT PRIMARY KEY, hash TEXT, size INTEGER, mod_time TEXT, indexed_at TEXT, language TEXT);`,
		`CREATE TABLE scans (id INTEGER PRIMARY KEY, root TEXT, scan_hash TEXT, started_at TEXT, completed_at TEXT, commit_hash TEXT);`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(context.Background(), s); err != nil {
			t.Fatalf("Failed to init schema: %v", err)
		}
	}

	oldTime := time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	db.ExecContext(context.Background(), `INSERT INTO files VALUES (?, ?, ?, ?, ?, ?);`, "stale.go", "h1", 100, oldTime, oldTime, "go")
	db.ExecContext(context.Background(), `INSERT INTO scans (root, scan_hash, started_at, completed_at) VALUES (?, ?, ?, ?);`, "/tmp", "hash", oldTime, oldTime)

	b := &Butler{db: db}
	info := b.GetIndexInfo()
	if info == nil {
		t.Fatal("Expected IndexInfo, got nil")
	}
	if info.Status != "stale" {
		t.Fatalf("Expected stale status, got %q", info.Status)
	}
}

func TestReadFileNotFound(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open memory DB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.ExecContext(context.Background(), `CREATE TABLE chunks (path TEXT, chunk_index INTEGER, content TEXT);`); err != nil {
		t.Fatalf("Failed to create chunks table: %v", err)
	}

	b := &Butler{db: db}
	if _, err := b.ReadFile("missing.go"); err == nil {
		t.Fatal("Expected error for missing file")
	}
}

func TestInferRoomDefault(t *testing.T) {
	b := &Butler{
		rooms:       map[string]model.Room{},
		entryPoints: map[string]string{},
		config:      &config.PalaceConfig{DefaultRoom: "core"},
	}

	if room := b.inferRoom("unknown/path.go"); room != "core" {
		t.Fatalf("inferRoom() = %q, want %q", room, "core")
	}
}

func TestHasMemory(t *testing.T) {
	t.Run("with memory", func(t *testing.T) {
		tmpRoot := t.TempDir()
		mem, err := memory.Open(tmpRoot)
		if err != nil {
			t.Fatalf("memory.Open() error = %v", err)
		}
		t.Cleanup(func() { _ = mem.Close() })

		b := &Butler{memory: mem}
		if !b.HasMemory() {
			t.Error("Expected HasMemory() to return true")
		}
	})

	t.Run("without memory", func(t *testing.T) {
		b := &Butler{}
		if b.HasMemory() {
			t.Error("Expected HasMemory() to return false")
		}
	})
}

func TestGetActivities(t *testing.T) {
	tmpRoot := t.TempDir()
	mem, err := memory.Open(tmpRoot)
	if err != nil {
		t.Fatalf("memory.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	b := &Butler{memory: mem}

	// Create a session and log activity
	sess, _ := mem.StartSession("agent", "id-1", "goal")
	mem.LogActivity(sess.ID, memory.Activity{Kind: "file_edit", Target: "test.go"})

	activities, err := b.GetActivities(sess.ID, "", 10)
	if err != nil {
		t.Fatalf("GetActivities() error = %v", err)
	}
	if len(activities) != 1 {
		t.Errorf("Expected 1 activity, got %d", len(activities))
	}
}

func TestGetActivitiesNoMemory(t *testing.T) {
	b := &Butler{}
	_, err := b.GetActivities("sess-1", "", 10)
	if err == nil {
		t.Error("Expected error when memory is nil")
	}
}

func TestGetRelevantLearnings(t *testing.T) {
	tmpRoot := t.TempDir()
	mem, err := memory.Open(tmpRoot)
	if err != nil {
		t.Fatalf("memory.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	b := &Butler{memory: mem}

	// Add a learning
	mem.AddLearning(memory.Learning{Content: "testing is important", Scope: "file", ScopePath: "test.go"})

	learnings, err := b.GetRelevantLearnings("test.go", "", 10)
	if err != nil {
		t.Fatalf("GetRelevantLearnings() error = %v", err)
	}
	if len(learnings) != 1 {
		t.Errorf("Expected 1 learning, got %d", len(learnings))
	}
}

func TestGetRelevantLearningsNoMemory(t *testing.T) {
	b := &Butler{}
	_, err := b.GetRelevantLearnings("test.go", "", 10)
	if err == nil {
		t.Error("Expected error when memory is nil")
	}
}

func TestBrainMethods(t *testing.T) {
	tmpRoot := t.TempDir()
	mem, err := memory.Open(tmpRoot)
	if err != nil {
		t.Fatalf("memory.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	b := &Butler{memory: mem}

	t.Run("AddIdea", func(t *testing.T) {
		id, err := b.AddIdea(memory.Idea{Content: "test idea", Status: "active"})
		if err != nil {
			t.Fatalf("AddIdea() error = %v", err)
		}
		if id == "" {
			t.Error("Expected non-empty ID")
		}
	})

	t.Run("GetIdeas", func(t *testing.T) {
		ideas, err := b.GetIdeas("active", "", "", 10)
		if err != nil {
			t.Fatalf("GetIdeas() error = %v", err)
		}
		if len(ideas) == 0 {
			t.Error("Expected at least 1 idea")
		}
	})

	t.Run("SearchIdeas", func(t *testing.T) {
		ideas, err := b.SearchIdeas("test", 10)
		if err != nil {
			t.Fatalf("SearchIdeas() error = %v", err)
		}
		if len(ideas) == 0 {
			t.Error("Expected at least 1 idea in search")
		}
	})

	t.Run("AddDecision", func(t *testing.T) {
		id, err := b.AddDecision(memory.Decision{Content: "test decision", Status: "active"})
		if err != nil {
			t.Fatalf("AddDecision() error = %v", err)
		}
		if id == "" {
			t.Error("Expected non-empty ID")
		}
	})

	t.Run("GetDecisions", func(t *testing.T) {
		decisions, err := b.GetDecisions("active", "", "", 10)
		if err != nil {
			t.Fatalf("GetDecisions() error = %v", err)
		}
		if len(decisions) == 0 {
			t.Error("Expected at least 1 decision")
		}
	})

	t.Run("SearchDecisions", func(t *testing.T) {
		decisions, err := b.SearchDecisions("test", 10)
		if err != nil {
			t.Fatalf("SearchDecisions() error = %v", err)
		}
		if len(decisions) == 0 {
			t.Error("Expected at least 1 decision in search")
		}
	})

	t.Run("RecordDecisionOutcome", func(t *testing.T) {
		id, _ := b.AddDecision(memory.Decision{Content: "outcome decision", Status: "active"})
		err := b.RecordDecisionOutcome(id, "successful", "it worked")
		if err != nil {
			t.Fatalf("RecordDecisionOutcome() error = %v", err)
		}
	})

	t.Run("SetTags", func(t *testing.T) {
		id, _ := b.AddIdea(memory.Idea{Content: "tagged idea"})
		err := b.SetTags(id, "idea", []string{"tag1", "tag2"})
		if err != nil {
			t.Fatalf("SetTags() error = %v", err)
		}
	})

	t.Run("AddLink", func(t *testing.T) {
		id1, _ := b.AddIdea(memory.Idea{Content: "idea 1"})
		id2, _ := b.AddIdea(memory.Idea{Content: "idea 2"})
		linkID, err := b.AddLink(memory.Link{
			SourceID:   id1,
			SourceKind: "idea",
			TargetID:   id2,
			TargetKind: "idea",
			Relation:   "related",
		})
		if err != nil {
			t.Fatalf("AddLink() error = %v", err)
		}
		if linkID == "" {
			t.Error("Expected non-empty link ID")
		}
	})
}

func TestBrainMethodsNoMemory(t *testing.T) {
	b := &Butler{}

	t.Run("AddIdea", func(t *testing.T) {
		_, err := b.AddIdea(memory.Idea{Content: "test"})
		if err == nil {
			t.Error("Expected error when memory is nil")
		}
	})

	t.Run("GetIdeas", func(t *testing.T) {
		_, err := b.GetIdeas("", "", "", 10)
		if err == nil {
			t.Error("Expected error when memory is nil")
		}
	})

	t.Run("SearchIdeas", func(t *testing.T) {
		_, err := b.SearchIdeas("test", 10)
		if err == nil {
			t.Error("Expected error when memory is nil")
		}
	})

	t.Run("AddDecision", func(t *testing.T) {
		_, err := b.AddDecision(memory.Decision{Content: "test"})
		if err == nil {
			t.Error("Expected error when memory is nil")
		}
	})

	t.Run("GetDecisions", func(t *testing.T) {
		_, err := b.GetDecisions("", "", "", 10)
		if err == nil {
			t.Error("Expected error when memory is nil")
		}
	})

	t.Run("SearchDecisions", func(t *testing.T) {
		_, err := b.SearchDecisions("test", 10)
		if err == nil {
			t.Error("Expected error when memory is nil")
		}
	})

	t.Run("RecordDecisionOutcome", func(t *testing.T) {
		err := b.RecordDecisionOutcome("id", "success", "note")
		if err == nil {
			t.Error("Expected error when memory is nil")
		}
	})

	t.Run("SetTags", func(t *testing.T) {
		err := b.SetTags("id", "idea", []string{"tag"})
		if err == nil {
			t.Error("Expected error when memory is nil")
		}
	})

	t.Run("AddLink", func(t *testing.T) {
		_, err := b.AddLink(memory.Link{})
		if err == nil {
			t.Error("Expected error when memory is nil")
		}
	})
}

func TestGetContextForTask(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open memory DB: %v", err)
	}
	defer db.Close()

	// Initialize schema
	stmts := []string{
		`CREATE TABLE files (path TEXT PRIMARY KEY, hash TEXT, size INTEGER, mod_time TEXT, indexed_at TEXT, language TEXT);`,
		`CREATE TABLE chunks (id INTEGER PRIMARY KEY, path TEXT, chunk_index INTEGER, start_line INTEGER, end_line INTEGER, content TEXT);`,
		`CREATE VIRTUAL TABLE chunks_fts USING fts5(path, content, chunk_index, tokenize="unicode61 tokenchars '_.:@#$-'");`,
		`CREATE TABLE symbols (id INTEGER PRIMARY KEY, file_path TEXT, name TEXT, kind TEXT, line_start INTEGER, line_end INTEGER, signature TEXT, doc_comment TEXT, parent_id INTEGER, exported INTEGER);`,
		`CREATE VIRTUAL TABLE symbols_fts USING fts5(name, file_path, kind, doc_comment, tokenize="unicode61 tokenchars '_'");`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(context.Background(), s); err != nil {
			t.Fatalf("Failed to init schema: %v", err)
		}
	}

	// Insert test data
	now := time.Now().UTC().Format(time.RFC3339)
	db.ExecContext(context.Background(), `INSERT INTO files VALUES (?, ?, ?, ?, ?, ?);`, "auth.go", "h1", 100, now, now, "go")
	db.ExecContext(context.Background(), `INSERT INTO chunks VALUES (?, ?, ?, ?, ?, ?);`, 1, "auth.go", 0, 1, 10, "func HandleAuth() {}")
	db.ExecContext(context.Background(), `INSERT INTO chunks_fts VALUES (?, ?, ?);`, "auth.go", "func HandleAuth() {}", 0)

	b := &Butler{
		db:          db,
		root:        "/tmp",
		rooms:       make(map[string]model.Room),
		entryPoints: make(map[string]string),
	}

	ctx, err := b.GetContextForTask("auth", 10)
	if err != nil {
		t.Fatalf("GetContextForTask() error = %v", err)
	}
	if ctx == nil {
		t.Error("Expected non-nil context")
	}
}

func TestGetContextForTaskWithOptions(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open memory DB: %v", err)
	}
	defer db.Close()

	// Initialize schema
	stmts := []string{
		`CREATE TABLE files (path TEXT PRIMARY KEY, hash TEXT, size INTEGER, mod_time TEXT, indexed_at TEXT, language TEXT);`,
		`CREATE TABLE chunks (id INTEGER PRIMARY KEY, path TEXT, chunk_index INTEGER, start_line INTEGER, end_line INTEGER, content TEXT);`,
		`CREATE VIRTUAL TABLE chunks_fts USING fts5(path, content, chunk_index, tokenize="unicode61 tokenchars '_.:@#$-'");`,
		`CREATE TABLE symbols (id INTEGER PRIMARY KEY, file_path TEXT, name TEXT, kind TEXT, line_start INTEGER, line_end INTEGER, signature TEXT, doc_comment TEXT, parent_id INTEGER, exported INTEGER);`,
		`CREATE VIRTUAL TABLE symbols_fts USING fts5(name, file_path, kind, doc_comment, tokenize="unicode61 tokenchars '_'");`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(context.Background(), s); err != nil {
			t.Fatalf("Failed to init schema: %v", err)
		}
	}

	b := &Butler{
		db:          db,
		root:        "/tmp",
		rooms:       make(map[string]model.Room),
		entryPoints: make(map[string]string),
	}

	ctx, err := b.GetContextForTaskWithOptions("auth", 5, 1000, false)
	if err != nil {
		t.Fatalf("GetContextForTaskWithOptions() error = %v", err)
	}
	// Context may be empty if no results, but should not error
	_ = ctx
}

func TestGetEnhancedContext(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open memory DB: %v", err)
	}
	defer db.Close()

	// Initialize schema
	stmts := []string{
		`CREATE TABLE files (path TEXT PRIMARY KEY, hash TEXT, size INTEGER, mod_time TEXT, indexed_at TEXT, language TEXT);`,
		`CREATE TABLE chunks (id INTEGER PRIMARY KEY, path TEXT, chunk_index INTEGER, start_line INTEGER, end_line INTEGER, content TEXT);`,
		`CREATE VIRTUAL TABLE chunks_fts USING fts5(path, content, chunk_index, tokenize="unicode61 tokenchars '_.:@#$-'");`,
		`CREATE TABLE symbols (id INTEGER PRIMARY KEY, file_path TEXT, name TEXT, kind TEXT, line_start INTEGER, line_end INTEGER, signature TEXT, doc_comment TEXT, parent_id INTEGER, exported INTEGER);`,
		`CREATE VIRTUAL TABLE symbols_fts USING fts5(name, file_path, kind, doc_comment, tokenize="unicode61 tokenchars '_'");`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(context.Background(), s); err != nil {
			t.Fatalf("Failed to init schema: %v", err)
		}
	}

	tmpRoot := t.TempDir()
	mem, err := memory.Open(tmpRoot)
	if err != nil {
		t.Fatalf("memory.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	b := &Butler{
		db:          db,
		root:        "/tmp",
		rooms:       make(map[string]model.Room),
		entryPoints: make(map[string]string),
		memory:      mem,
	}

	ctx, err := b.GetEnhancedContext(EnhancedContextOptions{Query: "auth", Limit: 5, IncludeLearnings: true})
	if err != nil {
		t.Fatalf("GetEnhancedContext() error = %v", err)
	}
	if ctx == nil {
		t.Error("Expected non-nil context")
	}
}

func TestLinkMethods(t *testing.T) {
	tmpRoot := t.TempDir()
	mem, err := memory.Open(tmpRoot)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	db := mem.DB()
	b := &Butler{
		db:          db,
		root:        tmpRoot,
		rooms:       make(map[string]model.Room),
		entryPoints: make(map[string]string),
		memory:      mem,
	}

	// Create test ideas to link
	id1, err := b.AddIdea(memory.Idea{Content: "link source idea"})
	if err != nil {
		t.Fatalf("AddIdea() error = %v", err)
	}
	id2, err := b.AddIdea(memory.Idea{Content: "link target idea"})
	if err != nil {
		t.Fatalf("AddIdea() error = %v", err)
	}

	// Add link
	linkID, err := b.AddLink(memory.Link{
		SourceID:   id1,
		SourceKind: "idea",
		TargetID:   id2,
		TargetKind: "idea",
		Relation:   "related",
	})
	if err != nil {
		t.Fatalf("AddLink() error = %v", err)
	}

	t.Run("GetLink", func(t *testing.T) {
		link, err := b.GetLink(linkID)
		if err != nil {
			t.Fatalf("GetLink() error = %v", err)
		}
		if link == nil {
			t.Fatal("Expected link to be non-nil")
		}
		if link.SourceID != id1 {
			t.Errorf("Expected SourceID %s, got %s", id1, link.SourceID)
		}
		if link.TargetID != id2 {
			t.Errorf("Expected TargetID %s, got %s", id2, link.TargetID)
		}
	})

	t.Run("GetLinksForRecord", func(t *testing.T) {
		links, err := b.GetLinksForRecord(id1)
		if err != nil {
			t.Fatalf("GetLinksForRecord() error = %v", err)
		}
		if len(links) == 0 {
			t.Error("Expected at least 1 link")
		}
	})

	t.Run("DeleteLink", func(t *testing.T) {
		err := b.DeleteLink(linkID)
		if err != nil {
			t.Fatalf("DeleteLink() error = %v", err)
		}
		// Verify deleted
		_, err = b.GetLink(linkID)
		if err == nil {
			t.Error("Expected error getting deleted link")
		}
	})
}

func TestGetEnhancedContextComprehensive(t *testing.T) {
	tmpRoot := t.TempDir()
	mem, err := memory.Open(tmpRoot)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	// Use index.Open to get a properly initialized index DB with schema
	dbPath := filepath.Join(tmpRoot, "test-index.db")
	indexDB, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open index db: %v", err)
	}
	t.Cleanup(func() { _ = indexDB.Close() })

	b := &Butler{
		db:          indexDB,
		root:        tmpRoot,
		rooms:       make(map[string]model.Room),
		entryPoints: make(map[string]string),
		memory:      mem,
	}

	// Create test data for enhanced context
	// Add an idea
	ideaID, _ := b.AddIdea(memory.Idea{Content: "auth implementation idea"})
	// Add a decision
	decisionID, _ := b.AddDecision(memory.Decision{Content: "use JWT for auth", Status: "active"})
	// Add a learning
	_, _ = b.AddLearning(memory.Learning{Content: "JWT requires secret rotation"})
	// Add a link between idea and decision
	_, _ = b.AddLink(memory.Link{
		SourceID:   ideaID,
		SourceKind: "idea",
		TargetID:   decisionID,
		TargetKind: "decision",
		Relation:   "related",
	})

	t.Run("with all options enabled", func(t *testing.T) {
		ctx, err := b.GetEnhancedContext(EnhancedContextOptions{
			Query:            "auth",
			Limit:            10,
			IncludeLearnings: true,
			IncludeFileIntel: true,
			IncludeIdeas:     true,
			IncludeDecisions: true,
		})
		if err != nil {
			t.Fatalf("GetEnhancedContext() error = %v", err)
		}
		if ctx == nil {
			t.Fatal("Expected non-nil context")
		}
	})

	t.Run("with default limit", func(t *testing.T) {
		ctx, err := b.GetEnhancedContext(EnhancedContextOptions{
			Query: "auth",
			Limit: 0, // Should default to 20
		})
		if err != nil {
			t.Fatalf("GetEnhancedContext() error = %v", err)
		}
		if ctx == nil {
			t.Fatal("Expected non-nil context")
		}
	})

	t.Run("without memory", func(t *testing.T) {
		noMemButler := &Butler{
			db:          indexDB,
			root:        tmpRoot,
			rooms:       make(map[string]model.Room),
			entryPoints: make(map[string]string),
			memory:      nil,
		}
		ctx, err := noMemButler.GetEnhancedContext(EnhancedContextOptions{
			Query:            "auth",
			Limit:            10,
			IncludeLearnings: true,
		})
		if err != nil {
			t.Fatalf("GetEnhancedContext() error = %v", err)
		}
		if ctx == nil {
			t.Fatal("Expected non-nil context")
		}
	})
}

func TestMemoryNilPaths(t *testing.T) {
	tmpRoot := t.TempDir()
	// Use memory.Open just to get an initialized DB, then set memory to nil
	mem, err := memory.Open(tmpRoot)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	db := mem.DB()
	t.Cleanup(func() { _ = mem.Close() })

	b := &Butler{
		db:          db,
		root:        tmpRoot,
		rooms:       make(map[string]model.Room),
		entryPoints: make(map[string]string),
		memory:      nil, // Explicitly set to nil to test error paths
	}

	t.Run("StartSession without memory", func(t *testing.T) {
		_, err := b.StartSession("claude-code", "test-agent", "test desc")
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("EndSession without memory", func(t *testing.T) {
		err := b.EndSession("some-id", "resolved", "summary")
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("GetSession without memory", func(t *testing.T) {
		_, err := b.GetSession("some-id")
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("ListSessions without memory", func(t *testing.T) {
		_, err := b.ListSessions(false, 10)
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("LogActivity without memory", func(t *testing.T) {
		err := b.LogActivity("sess-id", memory.Activity{})
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("RecordOutcome without memory", func(t *testing.T) {
		err := b.RecordOutcome("some-id", "success", "summary")
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("AddLearning without memory", func(t *testing.T) {
		_, err := b.AddLearning(memory.Learning{})
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("GetLearnings without memory", func(t *testing.T) {
		_, err := b.GetLearnings("", "", 10)
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("SearchLearnings without memory", func(t *testing.T) {
		_, err := b.SearchLearnings("query", 10)
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("ReinforceLearning without memory", func(t *testing.T) {
		err := b.ReinforceLearning("some-id")
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("GetFileIntel without memory", func(t *testing.T) {
		_, err := b.GetFileIntel("some/path")
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("RecordFileEdit without memory", func(t *testing.T) {
		err := b.RecordFileEdit("some/path.go", "claude-code")
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("GetActivities without memory", func(t *testing.T) {
		_, err := b.GetActivities("sess", "", 10)
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("GetRelevantLearnings without memory", func(t *testing.T) {
		_, err := b.GetRelevantLearnings("path", "", 10)
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("GetActiveAgents without memory", func(t *testing.T) {
		_, err := b.GetActiveAgents()
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("CheckConflict without memory", func(t *testing.T) {
		// CheckConflict returns nil, nil when no memory (no conflict if memory not available)
		conflict, err := b.CheckConflict("sess-id", "some/file")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if conflict != nil {
			t.Error("Expected nil conflict when memory is nil")
		}
	})

	t.Run("HasMemory without memory", func(t *testing.T) {
		if b.HasMemory() {
			t.Error("Expected HasMemory() to return false")
		}
	})

	t.Run("AddIdea without memory", func(t *testing.T) {
		_, err := b.AddIdea(memory.Idea{})
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("AddDecision without memory", func(t *testing.T) {
		_, err := b.AddDecision(memory.Decision{})
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("GetDecisions without memory", func(t *testing.T) {
		_, err := b.GetDecisions("", "", "", 10)
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("SearchDecisions without memory", func(t *testing.T) {
		_, err := b.SearchDecisions("query", 10)
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("GetIdeas without memory", func(t *testing.T) {
		_, err := b.GetIdeas("", "", "", 10)
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("SearchIdeas without memory", func(t *testing.T) {
		_, err := b.SearchIdeas("query", 10)
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("RecordDecisionOutcome without memory", func(t *testing.T) {
		err := b.RecordDecisionOutcome("id", "outcome", "notes")
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("SetTags without memory", func(t *testing.T) {
		err := b.SetTags("id", "kind", []string{"tag"})
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("AddLink without memory", func(t *testing.T) {
		_, err := b.AddLink(memory.Link{})
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("GetLink without memory", func(t *testing.T) {
		_, err := b.GetLink("id")
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("GetLinksForRecord without memory", func(t *testing.T) {
		_, err := b.GetLinksForRecord("id")
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("DeleteLink without memory", func(t *testing.T) {
		err := b.DeleteLink("id")
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("AddConversation without memory", func(t *testing.T) {
		_, err := b.AddConversation(memory.Conversation{})
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("GetConversation without memory", func(t *testing.T) {
		_, err := b.GetConversation("id")
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("GetConversations without memory", func(t *testing.T) {
		_, err := b.GetConversations("", "", 10)
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("SearchConversations without memory", func(t *testing.T) {
		_, err := b.SearchConversations("query", 10)
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})

	t.Run("GetConversationForSession without memory", func(t *testing.T) {
		_, err := b.GetConversationForSession("sess")
		if err == nil {
			t.Error("Expected error for nil memory")
		}
	})
}

func TestConversationMethods(t *testing.T) {
	tmpRoot := t.TempDir()
	mem, err := memory.Open(tmpRoot)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	db := mem.DB()
	b := &Butler{
		db:          db,
		root:        tmpRoot,
		rooms:       make(map[string]model.Room),
		entryPoints: make(map[string]string),
		memory:      mem,
	}

	t.Run("AddConversation", func(t *testing.T) {
		id, err := b.AddConversation(memory.Conversation{
			SessionID: "test-session-123",
			Summary:   "Test conversation summary",
			AgentType: "test-agent",
		})
		if err != nil {
			t.Fatalf("AddConversation() error = %v", err)
		}
		if id == "" {
			t.Error("Expected non-empty conversation ID")
		}

		t.Run("GetConversation", func(t *testing.T) {
			conv, err := b.GetConversation(id)
			if err != nil {
				t.Fatalf("GetConversation() error = %v", err)
			}
			if conv == nil {
				t.Fatal("Expected conversation to be non-nil")
			}
			if conv.Summary != "Test conversation summary" {
				t.Errorf("Expected summary 'Test conversation summary', got %q", conv.Summary)
			}
		})

		t.Run("GetConversationForSession", func(t *testing.T) {
			conv, err := b.GetConversationForSession("test-session-123")
			if err != nil {
				t.Fatalf("GetConversationForSession() error = %v", err)
			}
			if conv == nil {
				t.Fatal("Expected conversation to be non-nil")
			}
			if conv.SessionID != "test-session-123" {
				t.Errorf("Expected session ID 'test-session-123', got %q", conv.SessionID)
			}
		})
	})

	t.Run("GetConversations", func(t *testing.T) {
		convs, err := b.GetConversations("", "", 10)
		if err != nil {
			t.Fatalf("GetConversations() error = %v", err)
		}
		if len(convs) == 0 {
			t.Error("Expected at least 1 conversation")
		}
	})

	t.Run("SearchConversations", func(t *testing.T) {
		convs, err := b.SearchConversations("test", 10)
		if err != nil {
			t.Fatalf("SearchConversations() error = %v", err)
		}
		// May or may not find results depending on FTS5 setup
		_ = convs
	})
}
