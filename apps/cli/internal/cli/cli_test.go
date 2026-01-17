package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/commands"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
)

func TestBoolFlag(t *testing.T) {
	var f boolFlag

	// Test default state
	if f.Value != false || f.WasSet != false {
		t.Fatalf("expected default false/unset, got value=%v set=%v", f.Value, f.WasSet)
	}

	// Test setting to true
	if err := f.Set("true"); err != nil {
		t.Fatalf("unexpected error setting true: %v", err)
	}
	if !f.Value || !f.WasSet {
		t.Fatalf("expected true/set after Set(true), got value=%v set=%v", f.Value, f.WasSet)
	}

	// Test setting to false
	f = boolFlag{}
	if err := f.Set("false"); err != nil {
		t.Fatalf("unexpected error setting false: %v", err)
	}
	if f.Value || !f.WasSet {
		t.Fatalf("expected false/set after Set(false), got value=%v set=%v", f.Value, f.WasSet)
	}

	// Test empty string (boolean flag without value)
	f = boolFlag{}
	if err := f.Set(""); err != nil {
		t.Fatalf("unexpected error setting empty: %v", err)
	}
	if !f.Value || !f.WasSet {
		t.Fatalf("expected true/set after Set(\"\"), got value=%v set=%v", f.Value, f.WasSet)
	}

	// Test invalid value
	f = boolFlag{}
	if err := f.Set("invalid"); err == nil {
		t.Fatal("expected error for invalid value")
	}

	// Test String()
	f = boolFlag{Value: true}
	if f.String() != "true" {
		t.Fatalf("expected String()=\"true\", got %q", f.String())
	}
	f = boolFlag{Value: false}
	if f.String() != "false" {
		t.Fatalf("expected String()=\"false\", got %q", f.String())
	}

	// Test IsBoolFlag
	if !f.IsBoolFlag() {
		t.Fatal("expected IsBoolFlag() to return true")
	}
}

func TestTruncateLine(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a longer line", 10, "this is a ..."},
		{"", 5, ""},
	}

	for _, tt := range tests {
		got := truncateLine(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateLine(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestValidateConfidence(t *testing.T) {
	tests := []struct {
		name    string
		value   float64
		wantErr bool
	}{
		{"zero is valid", 0.0, false},
		{"one is valid", 1.0, false},
		{"mid-range is valid", 0.5, false},
		{"negative is invalid", -0.1, true},
		{"above one is invalid", 1.1, true},
		{"large negative is invalid", -10.0, true},
		{"large positive is invalid", 10.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfidence(tt.value)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateLimit(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{"zero is valid", 0, false},
		{"positive is valid", 10, false},
		{"large positive is valid", 1000, false},
		{"negative is invalid", -1, true},
		{"large negative is invalid", -100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLimit(tt.value)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateScope(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"file is valid", "file", false},
		{"room is valid", "room", false},
		{"palace is valid", "palace", false},
		{"invalid scope", "invalid", true},
		{"empty is invalid", "", true},
		{"uppercase file is invalid", "FILE", true},
		{"folder is invalid", "folder", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateScope(tt.value)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{"port 1 is valid", 1, false},
		{"port 80 is valid", 80, false},
		{"port 443 is valid", 443, false},
		{"port 8080 is valid", 8080, false},
		{"port 65535 is valid", 65535, false},
		{"port 0 is invalid", 0, true},
		{"negative port is invalid", -1, true},
		{"port above 65535 is invalid", 65536, true},
		{"large port is invalid", 100000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePort(tt.value)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestSetBuildInfo(t *testing.T) {
	// Save original values
	origVersion := buildVersion
	origCommit := buildCommit
	origDate := buildDate

	// Restore after test
	defer func() {
		buildVersion = origVersion
		buildCommit = origCommit
		buildDate = origDate
	}()

	// Test setting all values
	SetBuildInfo("1.2.3", "abc123", "2024-01-01")
	if buildVersion != "1.2.3" {
		t.Errorf("buildVersion = %q, want %q", buildVersion, "1.2.3")
	}
	if buildCommit != "abc123" {
		t.Errorf("buildCommit = %q, want %q", buildCommit, "abc123")
	}
	if buildDate != "2024-01-01" {
		t.Errorf("buildDate = %q, want %q", buildDate, "2024-01-01")
	}

	// Test empty values don't override
	SetBuildInfo("", "", "")
	if buildVersion != "1.2.3" {
		t.Errorf("empty string should not override buildVersion, got %q", buildVersion)
	}
	if buildCommit != "abc123" {
		t.Errorf("empty string should not override buildCommit, got %q", buildCommit)
	}
	if buildDate != "2024-01-01" {
		t.Errorf("empty string should not override buildDate, got %q", buildDate)
	}
}

func TestGetVersion(t *testing.T) {
	origVersion := buildVersion
	defer func() { buildVersion = origVersion }()

	buildVersion = "test-version"
	if got := GetVersion(); got != "test-version" {
		t.Errorf("GetVersion() = %q, want %q", got, "test-version")
	}
}

func TestRunUnknownCommand(t *testing.T) {
	err := Run([]string{"unknown-command"})
	if err == nil {
		t.Error("expected error for unknown command")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("error should mention 'unknown command', got: %v", err)
	}
}

func TestRunNoArgs(t *testing.T) {
	// No args should show usage (not error)
	err := Run([]string{})
	if err != nil {
		t.Errorf("Run with no args should not error, got: %v", err)
	}
}

func TestRunVersion(t *testing.T) {
	// Test version command variants
	for _, cmd := range []string{"version", "--version", "-v"} {
		t.Run(cmd, func(t *testing.T) {
			err := Run([]string{cmd})
			if err != nil {
				t.Errorf("Run(%q) error: %v", cmd, err)
			}
		})
	}
}

func TestRunHelp(t *testing.T) {
	// Test help command variants
	for _, cmd := range []string{"help", "-h", "--help"} {
		t.Run(cmd, func(t *testing.T) {
			err := Run([]string{cmd})
			if err != nil {
				t.Errorf("Run(%q) error: %v", cmd, err)
			}
		})
	}
}

func TestCmdVersionParse(t *testing.T) {
	// Test version command parses correctly
	err := cmdVersion([]string{})
	if err != nil {
		t.Errorf("cmdVersion() error: %v", err)
	}

	// Test with invalid flag
	err = cmdVersion([]string{"--invalid"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestUsage(t *testing.T) {
	// Calling usage should not error and should return nil
	err := usage()
	if err != nil {
		t.Errorf("usage() error: %v", err)
	}
}

func TestMustAbs(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"current dir", "."},
		{"relative path", "./foo/bar"},
		{"absolute path", "/tmp/test"},
		{"empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// mustAbs should not panic and should return a string
			result := mustAbs(tt.input)
			if result == "" && tt.input != "" {
				t.Errorf("mustAbs(%q) returned empty string", tt.input)
			}
		})
	}
}

func TestScopeFileCount(t *testing.T) {
	tests := []struct {
		name string
		cp   model.ContextPack
		want int
	}{
		{
			name: "nil scope returns 0",
			cp:   model.ContextPack{Scope: nil},
			want: 0,
		},
		{
			name: "scope with count",
			cp: model.ContextPack{
				Scope: &model.ScopeInfo{FileCount: 42},
			},
			want: 42,
		},
		{
			name: "scope with zero count",
			cp: model.ContextPack{
				Scope: &model.ScopeInfo{FileCount: 0},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scopeFileCount(tt.cp)
			if got != tt.want {
				t.Errorf("scopeFileCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestExplainAll(t *testing.T) {
	result := commands.ExplainAll()
	if result == "" {
		t.Error("ExplainAll() should return non-empty string")
	}

	// Check for expected sections (Canonical commands)
	expectedSections := []string{"SCAN", "CHECK", "EXPLORE", "BRIEF", "CLEAN"}
	for _, section := range expectedSections {
		if !strings.Contains(result, section) {
			t.Errorf("ExplainAll() should contain section %q", section)
		}
	}
}

func TestCmdInitInvalidFlag(t *testing.T) {
	err := cmdInit([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestCmdScanInvalidFlag(t *testing.T) {
	err := cmdScan([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestCmdCheckInvalidFlag(t *testing.T) {
	err := cmdCheck([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestCmdExploreInvalidFlag(t *testing.T) {
	err := cmdExplore([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestCmdExploreNoQuery(t *testing.T) {
	err := cmdExplore([]string{})
	if err == nil {
		t.Error("expected error for missing query")
	}
}

func TestCmdExploreMapNoArgs(t *testing.T) {
	err := cmdExplore([]string{"--map"})
	if err == nil {
		t.Error("expected error for missing map arguments")
	}
}

func TestCmdSessionNoArgs(t *testing.T) {
	err := cmdSession([]string{})
	if err == nil {
		t.Error("expected error for missing subcommand")
	}
}

func TestCmdSessionUnknownSubcommand(t *testing.T) {
	err := cmdSession([]string{"unknown"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}

func TestCmdCorridorNoArgs(t *testing.T) {
	err := cmdCorridor([]string{})
	if err == nil {
		t.Error("expected error for missing subcommand")
	}
}

func TestCmdCorridorUnknownSubcommand(t *testing.T) {
	err := cmdCorridor([]string{"unknown"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}

func TestCmdCleanInvalidFlag(t *testing.T) {
	err := cmdClean([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestCmdHelpKnownCommands(t *testing.T) {
	// Only test commands that have help topics defined in cmdHelp
	commandNames := []string{"init", "scan", "check", "explore", "store", "recall",
		"brief", "serve", "session", "corridor", "dashboard", "clean"}

	for _, cmd := range commandNames {
		t.Run(cmd, func(t *testing.T) {
			err := cmdHelp([]string{cmd})
			if err != nil {
				t.Errorf("cmdHelp(%q) error: %v", cmd, err)
			}
		})
	}
}

func TestCmdHelpUnknownCommand(t *testing.T) {
	err := cmdHelp([]string{"unknown-command"})
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestCmdHelpAll(t *testing.T) {
	err := cmdHelp([]string{"all"})
	if err != nil {
		t.Errorf("cmdHelp(all) error: %v", err)
	}
}

func TestBoolFlagAdditionalCases(t *testing.T) {
	// Test "1" value
	f := boolFlag{}
	if err := f.Set("1"); err != nil {
		t.Errorf("Set(\"1\") error: %v", err)
	}
	if !f.Value {
		t.Error("Set(\"1\") should set value to true")
	}

	// Test "0" value
	f = boolFlag{}
	if err := f.Set("0"); err != nil {
		t.Errorf("Set(\"0\") error: %v", err)
	}
	if f.Value {
		t.Error("Set(\"0\") should set value to false")
	}

	// Test "TRUE" (uppercase)
	f = boolFlag{}
	if err := f.Set("TRUE"); err != nil {
		t.Errorf("Set(\"TRUE\") error: %v", err)
	}
	if !f.Value {
		t.Error("Set(\"TRUE\") should set value to true")
	}

	// Test "FALSE" (uppercase)
	f = boolFlag{}
	if err := f.Set("FALSE"); err != nil {
		t.Errorf("Set(\"FALSE\") error: %v", err)
	}
	if f.Value {
		t.Error("Set(\"FALSE\") should set value to false")
	}
}

func TestRunWithHelpSubcommand(t *testing.T) {
	// Test help with specific command
	err := Run([]string{"help", "init"})
	if err != nil {
		t.Errorf("Run(help init) error: %v", err)
	}
}

func TestCmdServeInvalidFlag(t *testing.T) {
	err := cmdServe([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestCmdDashboardInvalidFlag(t *testing.T) {
	err := cmdDashboard([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestCmdUpdateInvalidFlag(t *testing.T) {
	err := cmdUpdate([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestCmdStoreNoContent(t *testing.T) {
	err := cmdStore([]string{})
	if err == nil {
		t.Error("expected error for missing content")
	}
}

func TestCmdRecallInvalidFlag(t *testing.T) {
	err := cmdRecall([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestCmdBriefInvalidFlag(t *testing.T) {
	err := cmdBrief([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestCmdCorridorLinkNoArgs(t *testing.T) {
	err := commands.ExecuteCorridorLink([]string{})
	if err == nil {
		t.Error("expected error for missing arguments")
	}
}

func TestCmdCorridorUnlinkNoArgs(t *testing.T) {
	err := commands.ExecuteCorridorUnlink([]string{})
	if err == nil {
		t.Error("expected error for missing arguments")
	}
}

func TestCmdCorridorPromoteNoArgs(t *testing.T) {
	err := commands.ExecuteCorridorPromote([]string{})
	if err == nil {
		t.Error("expected error for missing arguments")
	}
}

func TestCmdSessionEndNoID(t *testing.T) {
	err := commands.RunSessionEnd([]string{})
	if err == nil {
		t.Error("expected error for missing session ID")
	}
}

func TestCmdSessionShowNoID(t *testing.T) {
	err := commands.RunSessionShow([]string{})
	if err == nil {
		t.Error("expected error for missing session ID")
	}
}

func TestCmdCheckWithSignalNoDiff(t *testing.T) {
	// --signal requires --diff, so this should fail
	err := commands.ExecuteCheck(commands.CheckOptions{
		Root:   ".",
		Signal: true,
	})
	if err == nil {
		t.Error("expected error for --signal without --diff")
	}
}

func TestCmdInitCreatesLayout(t *testing.T) {
	root := t.TempDir()
	err := cmdInit([]string{"--root", root})
	if err != nil {
		t.Fatalf("cmdInit() error: %v", err)
	}

	expected := []string{
		filepath.Join(root, ".palace", "palace.jsonc"),
		filepath.Join(root, ".palace", "rooms", "project-overview.jsonc"),
		filepath.Join(root, ".palace", "playbooks", "default.jsonc"),
		filepath.Join(root, ".palace", "project-profile.json"),
	}
	for _, path := range expected {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file %s to exist: %v", path, err)
		}
	}
}

func TestCmdExploreWithFullFlag(t *testing.T) {
	root := t.TempDir()
	if err := cmdInit([]string{"--root", root}); err != nil {
		t.Fatalf("cmdInit() error: %v", err)
	}

	// Create index first
	db := seedIndexForCLI(t, root)
	db.Close()

	// Test explore with --full flag (shows extended context)
	err := cmdExplore([]string{"--root", root, "--full", "DoWork"})
	if err != nil {
		t.Fatalf("cmdExplore --full error: %v", err)
	}
}

func TestCmdStoreAndRecall(t *testing.T) {
	root := t.TempDir()
	if err := cmdInit([]string{"--root", root}); err != nil {
		t.Fatalf("cmdInit() error: %v", err)
	}

	if err := cmdStore([]string{"--root", root, "Always run tests"}); err != nil {
		t.Fatalf("cmdStore() error: %v", err)
	}

	if err := cmdRecall([]string{"--root", root, "tests"}); err != nil {
		t.Fatalf("cmdRecall() error: %v", err)
	}
}

func TestCmdSessionStartEnd(t *testing.T) {
	root := t.TempDir()
	if err := cmdInit([]string{"--root", root}); err != nil {
		t.Fatalf("cmdInit() error: %v", err)
	}

	if err := commands.RunSessionStart([]string{"--root", root, "--agent", "cli", "--goal", "test"}); err != nil {
		t.Fatalf("cmdSessionStart() error: %v", err)
	}
}

func TestCmdExploreAndMap(t *testing.T) {
	root := t.TempDir()
	if err := cmdInit([]string{"--root", root}); err != nil {
		t.Fatalf("cmdInit() error: %v", err)
	}

	db := seedIndexForCLI(t, root)
	db.Close()

	// Test search mode
	if err := cmdExplore([]string{"--root", root, "DoWork"}); err != nil {
		t.Fatalf("cmdExplore() error: %v", err)
	}

	// Test map mode: callers
	if err := cmdExplore([]string{"--root", root, "--map", "DoWork"}); err != nil {
		t.Fatalf("cmdExplore --map callers error: %v", err)
	}

	// Test map mode: callees
	if err := cmdExplore([]string{"--root", root, "--map", "DoWork", "--file", "main.go"}); err != nil {
		t.Fatalf("cmdExplore --map callees error: %v", err)
	}

	// Test map mode: file call graph
	if err := cmdExplore([]string{"--root", root, "--map", "--file", "main.go"}); err != nil {
		t.Fatalf("cmdExplore --map file error: %v", err)
	}
}

func seedIndexForCLI(t *testing.T, root string) *index.DBHandle {
	t.Helper()

	dbPath := filepath.Join(root, ".palace", "index", "palace.db")
	db, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("index.Open() error = %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	files := []string{"main.go", "caller.go"}
	for _, path := range files {
		if _, err := db.ExecContext(context.Background(), `INSERT INTO files (path, hash, size, mod_time, indexed_at, language) VALUES (?, ?, ?, ?, ?, ?)`,
			path, "hash", 1, now, now, "go"); err != nil {
			t.Fatalf("insert file error = %v", err)
		}
	}

	content := "package main\nfunc DoWork() {}\n"
	if _, err := db.ExecContext(context.Background(), `INSERT INTO chunks (path, chunk_index, start_line, end_line, content) VALUES (?, ?, ?, ?, ?)`,
		"main.go", 0, 1, 2, content); err != nil {
		t.Fatalf("insert chunk error = %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO chunks_fts (path, content, chunk_index) VALUES (?, ?, ?)`,
		"main.go", content, 0); err != nil {
		t.Fatalf("insert chunk fts error = %v", err)
	}

	if _, err := db.ExecContext(context.Background(), `INSERT INTO symbols (file_path, name, kind, line_start, line_end, signature, doc_comment, exported) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"main.go", "DoWork", "function", 1, 20, "func DoWork()", "", 1); err != nil {
		t.Fatalf("insert symbol error = %v", err)
	}

	if _, err := db.ExecContext(context.Background(), `INSERT INTO relationships (source_file, target_file, target_symbol, kind, line) VALUES (?, ?, ?, ?, ?)`,
		"caller.go", "", "DoWork", "call", 10); err != nil {
		t.Fatalf("insert relationship error = %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO relationships (source_file, target_file, target_symbol, kind, line) VALUES (?, ?, ?, ?, ?)`,
		"main.go", "", "Helper", "call", 2); err != nil {
		t.Fatalf("insert relationship error = %v", err)
	}

	return db
}

func TestCmdSessionLifecycle(t *testing.T) {
	root := t.TempDir()
	if err := commands.RunSessionStart([]string{"--root", root, "--agent", "cli", "--goal", "ship it"}); err != nil {
		t.Fatalf("cmdSessionStart() error: %v", err)
	}

	mem, err := memory.Open(root)
	if err != nil {
		t.Fatalf("memory.Open() error: %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	sessions, err := mem.ListSessions(false, 1)
	if err != nil {
		t.Fatalf("ListSessions() error: %v", err)
	}
	if len(sessions) == 0 {
		t.Fatalf("expected at least one session")
	}
	sessionID := sessions[0].ID

	if err := mem.LogActivity(sessionID, memory.Activity{Kind: "edit", Target: "main.go", Outcome: "success"}); err != nil {
		t.Fatalf("LogActivity() error: %v", err)
	}

	if err := commands.RunSessionShow([]string{"--root", root, sessionID}); err != nil {
		t.Fatalf("cmdSessionShow() error: %v", err)
	}
	if err := commands.RunSessionList([]string{"--root", root, "--active"}); err != nil {
		t.Fatalf("cmdSessionList() error: %v", err)
	}
	if err := commands.RunSessionEnd([]string{"--root", root, "--state", "completed", "--summary", "done", sessionID}); err != nil {
		t.Fatalf("cmdSessionEnd() error: %v", err)
	}
}

func TestCmdStoreRecallBrief(t *testing.T) {
	root := t.TempDir()
	// Force as learning with --direct since auto-classification might not detect this as a learning
	if err := cmdStore([]string{"--root", root, "--scope", "file", "--path", "src/main.go", "--confidence", "0.7", "--as", "learning", "--direct", "Remember to test"}); err != nil {
		t.Fatalf("cmdStore() error: %v", err)
	}
	if err := cmdRecall([]string{"--root", root, "--scope", "file", "--path", "src/main.go"}); err != nil {
		t.Fatalf("cmdRecall() error: %v", err)
	}

	mem, err := memory.Open(root)
	if err != nil {
		t.Fatalf("memory.Open() error: %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	learnings, err := mem.GetLearnings("file", "src/main.go", 1)
	if err != nil || len(learnings) == 0 {
		t.Fatalf("GetLearnings() error = %v", err)
	}
	if err := mem.AssociateLearningWithFile("src/main.go", learnings[0].ID); err != nil {
		t.Fatalf("AssociateLearningWithFile() error: %v", err)
	}
	if err := mem.RecordFileEdit("src/main.go", "cli"); err != nil {
		t.Fatalf("RecordFileEdit() error: %v", err)
	}
	if err := mem.RecordFileFailure("src/main.go"); err != nil {
		t.Fatalf("RecordFileFailure() error: %v", err)
	}

	// Test brief with file path (includes file intel)
	if err := cmdBrief([]string{"--root", root, "src/main.go"}); err != nil {
		t.Fatalf("cmdBrief() error: %v", err)
	}
}
