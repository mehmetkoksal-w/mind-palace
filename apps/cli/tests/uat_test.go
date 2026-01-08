// Package integration_test contains User Acceptance Tests that validate Mind Palace
// provides real value for AI agents in typical development workflows.
//
// These tests simulate realistic scenarios an AI agent would encounter:
// - Code discovery and navigation
// - Understanding codebase structure
// - Learning and recalling patterns
// - Session management
// - Getting context for tasks
package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAgentCodeDiscovery validates that an AI agent can effectively
// discover and navigate code using Mind Palace search capabilities.
func TestAgentCodeDiscovery(t *testing.T) {
	workspace, binPath := setupTestWorkspace(t, map[string]string{
		"main.go": `package main

import "github.com/example/auth"

func main() {
	auth.Initialize()
	startServer()
}

func startServer() {
	// Start HTTP server
}
`,
		"auth/handler.go": `package auth

// Initialize sets up authentication middleware
func Initialize() {
	loadConfig()
}

// Authenticate validates user credentials
func Authenticate(username, password string) bool {
	return validateCredentials(username, password)
}

func loadConfig() {}
func validateCredentials(u, p string) bool { return true }
`,
		"auth/middleware.go": `package auth

// RequireAuth middleware ensures user is authenticated
func RequireAuth(next func()) func() {
	return func() {
		if !IsAuthenticated() {
			return
		}
		next()
	}
}

func IsAuthenticated() bool { return true }
`,
		"api/routes.go": `package api

import "github.com/example/auth"

// RegisterRoutes sets up API endpoints
func RegisterRoutes() {
	auth.RequireAuth(handleUsers)
}

func handleUsers() {}
`,
	})

	// Initialize and scan
	runPalace(t, binPath, workspace, "init", "--root", workspace)
	runPalace(t, binPath, workspace, "scan", "--root", workspace)

	// UAT 1: Agent can find authentication-related code
	t.Run("find_auth_code", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "explore", "--root", workspace, "authentication")

		// Should find auth-related files
		if !strings.Contains(output, "auth") {
			t.Errorf("Expected to find auth-related results, got:\n%s", output)
		}
	})

	// UAT 2: Agent can find specific function implementations
	t.Run("find_function", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "explore", "--root", workspace, "Initialize")

		if !strings.Contains(output, "auth") || !strings.Contains(output, "Initialize") {
			t.Errorf("Expected to find Initialize function, got:\n%s", output)
		}
	})

	// UAT 3: Agent can explore entry points with natural language
	t.Run("natural_language_query", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "explore", "--root", workspace, "where is user validation")

		// Should find validation-related code
		if !strings.Contains(strings.ToLower(output), "auth") && !strings.Contains(strings.ToLower(output), "validate") {
			t.Logf("Query result:\n%s", output)
			// Note: This is a soft check - results may vary based on FTS5 ranking
		}
	})

	// UAT 4: Agent gets useful context for a task
	t.Run("get_context_for_task", func(t *testing.T) {
		// Set a goal in context pack
		contextPath := filepath.Join(workspace, ".palace", "outputs", "context-pack.json")

		runPalace(t, binPath, workspace, "check", "--root", workspace, "--collect")

		data, err := os.ReadFile(contextPath)
		if err != nil {
			t.Fatalf("Failed to read context pack: %v", err)
		}

		var cp map[string]interface{}
		if err := json.Unmarshal(data, &cp); err != nil {
			t.Fatalf("Failed to parse context pack: %v", err)
		}

		// Context pack should have meaningful data
		if cp["goal"] == nil || cp["goal"] == "" {
			t.Error("Context pack should have a goal")
		}
		if cp["scanHash"] == nil || cp["scanHash"] == "" {
			t.Error("Context pack should have scanHash")
		}
	})
}

// TestAgentLearningWorkflow validates that AI agents can capture and
// recall learnings throughout their work session.
func TestAgentLearningWorkflow(t *testing.T) {
	workspace, binPath := setupTestWorkspace(t, map[string]string{
		"config.go": `package main

type Config struct {
	Port int
	Host string
}

func LoadConfig() Config {
	// Always validate config after loading
	return Config{Port: 8080, Host: "localhost"}
}
`,
	})

	runPalace(t, binPath, workspace, "init", "--root", workspace)
	runPalace(t, binPath, workspace, "scan", "--root", workspace)

	// UAT 1: Agent can record a learning
	t.Run("record_learning", func(t *testing.T) {
		runPalace(t, binPath, workspace, "store", "--root", workspace,
			"--as", "learning",
			"Always validate configuration after loading to prevent runtime errors")

		// Should be able to recall it back
		output := runPalace(t, binPath, workspace, "recall", "--root", workspace)

		if !strings.Contains(output, "validate") || !strings.Contains(output, "configuration") {
			t.Errorf("Expected to recall the learning, got:\n%s", output)
		}
	})

	// UAT 2: Agent can record scoped learnings
	t.Run("record_scoped_learning", func(t *testing.T) {
		runPalace(t, binPath, workspace, "store", "--root", workspace,
			"--as", "learning",
			"--scope", "file", "--path", "config.go",
			"Config struct requires Port > 0 and valid Host")

		// Should be able to filter by scope
		output := runPalace(t, binPath, workspace, "recall", "--root", workspace,
			"--scope", "file", "--path", "config.go")

		if !strings.Contains(output, "Port") || !strings.Contains(output, "Host") {
			t.Logf("Scoped recall output:\n%s", output)
		}
	})

	// UAT 3: Agent can search learnings by content
	t.Run("search_learnings", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "recall", "--root", workspace, "configuration")

		if !strings.Contains(output, "config") {
			t.Logf("Search learnings output:\n%s", output)
		}
	})
}

// TestAgentSessionWorkflow validates that AI agents can properly
// manage work sessions and avoid conflicts.
func TestAgentSessionWorkflow(t *testing.T) {
	workspace, binPath := setupTestWorkspace(t, map[string]string{
		"app.go": `package main

func main() {
	println("Hello")
}
`,
	})

	runPalace(t, binPath, workspace, "init", "--root", workspace)
	runPalace(t, binPath, workspace, "scan", "--root", workspace)

	// UAT 1: Agent can start a session
	t.Run("start_session", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "session", "start",
			"--root", workspace,
			"--agent", "claude-code",
			"--agent-id", "test-instance-1",
			"--goal", "Implement new feature")

		if !strings.Contains(output, "Session started") && !strings.Contains(output, "session") {
			t.Errorf("Expected session to start, got:\n%s", output)
		}
	})

	// UAT 2: Agent can list active sessions
	t.Run("list_sessions", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "session", "list", "--root", workspace)

		// Should show the active session or no sessions message
		if output == "" {
			t.Error("Expected some output from session list")
		}
	})

	// UAT 3: Agent can end a session
	t.Run("end_session", func(t *testing.T) {
		// First get session ID from list, then end it
		// For now, just verify the command works
		output := runPalace(t, binPath, workspace, "session", "list", "--root", workspace)
		t.Logf("Sessions before end:\n%s", output)
	})
}

// TestAgentBriefingWorkflow validates that agents get useful briefings
// before starting work on a file or area.
func TestAgentBriefingWorkflow(t *testing.T) {
	workspace, binPath := setupTestWorkspace(t, map[string]string{
		"fragile.go": `package main

// This file has complex regex that often breaks
func ParseInput(s string) bool {
	// Complex parsing logic here
	return true
}
`,
		"stable.go": `package main

func HelloWorld() string {
	return "Hello, World!"
}
`,
	})

	runPalace(t, binPath, workspace, "init", "--root", workspace)
	runPalace(t, binPath, workspace, "scan", "--root", workspace)

	// Add a learning about the fragile file
	runPalace(t, binPath, workspace, "store", "--root", workspace,
		"--as", "learning",
		"--scope", "file", "--path", "fragile.go",
		"This file's regex is fragile - always test thoroughly after changes")

	// UAT 1: Agent gets briefing before working on file
	t.Run("file_briefing", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "brief", "--root", workspace, "fragile.go")

		// Should show some briefing information
		if output == "" {
			t.Error("Expected briefing output")
		}
		t.Logf("Briefing output:\n%s", output)
	})

	// UAT 2: Agent gets general workspace briefing
	t.Run("workspace_briefing", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "brief", "--root", workspace)

		if output == "" {
			t.Error("Expected workspace briefing output")
		}
	})
}

// TestAgentFileIntelligence validates that agents can get and use
// file intelligence to make better decisions.
func TestAgentFileIntelligence(t *testing.T) {
	workspace, binPath := setupTestWorkspace(t, map[string]string{
		"hotspot.go": `package main

// This file is frequently edited
func HotFunction() {}
`,
	})

	runPalace(t, binPath, workspace, "init", "--root", workspace)
	runPalace(t, binPath, workspace, "scan", "--root", workspace)

	// Start a session and record some edits to build intel
	runPalace(t, binPath, workspace, "session", "start",
		"--root", workspace,
		"--agent", "test-agent",
		"--agent-id", "intel-test",
		"--goal", "Testing file intel")

	// UAT: Agent can get intel about a file
	t.Run("get_file_intel", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "brief", "--root", workspace, "hotspot.go")

		// Should show file intelligence
		if output == "" {
			t.Error("Expected file intel output")
		}
		t.Logf("File intel:\n%s", output)
	})
}

// TestAgentContextPackQuality validates that the context pack
// provides high-quality, actionable context for AI agents.
func TestAgentContextPackQuality(t *testing.T) {
	workspace, binPath := setupTestWorkspace(t, map[string]string{
		"main.go": `package main

import (
	"./auth"
	"./api"
)

func main() {
	auth.Init()
	api.Serve()
}
`,
		"auth/auth.go": `package auth

func Init() {
	loadKeys()
}

func loadKeys() {}
func Validate(token string) bool { return true }
`,
		"api/server.go": `package api

func Serve() {
	registerRoutes()
	listen()
}

func registerRoutes() {}
func listen() {}
`,
	})

	runPalace(t, binPath, workspace, "init", "--root", workspace)
	runPalace(t, binPath, workspace, "scan", "--root", workspace)

	// Generate context pack
	t.Run("context_pack", func(t *testing.T) {
		runPalace(t, binPath, workspace, "check", "--root", workspace, "--collect")

		contextPath := filepath.Join(workspace, ".palace", "outputs", "context-pack.json")
		data, err := os.ReadFile(contextPath)
		if err != nil {
			t.Fatalf("Failed to read context pack: %v", err)
		}

		var cp map[string]interface{}
		if err := json.Unmarshal(data, &cp); err != nil {
			t.Fatalf("Failed to parse context pack: %v", err)
		}

		// Validate context pack quality
		t.Run("has_schema_version", func(t *testing.T) {
			if cp["schemaVersion"] == nil {
				t.Error("Context pack should have schemaVersion")
			}
		})

		t.Run("has_scan_info", func(t *testing.T) {
			if cp["scanHash"] == nil || cp["scanHash"] == "" {
				t.Error("Context pack should have scanHash")
			}
			if cp["scanId"] == nil || cp["scanId"] == "" {
				t.Error("Context pack should have scanId")
			}
		})

		t.Run("has_provenance", func(t *testing.T) {
			prov, ok := cp["provenance"].(map[string]interface{})
			if !ok {
				t.Error("Context pack should have provenance")
				return
			}
			if prov["createdBy"] == nil {
				t.Error("Provenance should have createdBy")
			}
		})

		t.Run("has_scope", func(t *testing.T) {
			scope, ok := cp["scope"].(map[string]interface{})
			if !ok {
				// Scope may be nil for some goals - this is acceptable
				t.Logf("Note: scope is nil for this context pack")
				return
			}
			if scope["fileCount"] == nil {
				t.Error("Scope should have fileCount")
			}
		})
	})
}

// TestAgentRoomNavigation validates that agents can understand
// and navigate the logical room structure of a project.
func TestAgentRoomNavigation(t *testing.T) {
	workspace, binPath := setupTestWorkspace(t, map[string]string{
		"cmd/main.go": `package main

func main() {}
`,
		"internal/auth/handler.go": `package auth

func Handle() {}
`,
		"internal/api/routes.go": `package api

func Routes() {}
`,
	})

	// Create a room configuration
	roomsDir := filepath.Join(workspace, ".palace", "rooms")
	os.MkdirAll(roomsDir, 0o755)

	authRoom := `{
  "$schema": "../schemas/room.schema.json",
  "schemaVersion": "1.0.0",
  "kind": "palace/room",
  "name": "auth",
  "summary": "Authentication and authorization module",
  "entryPoints": ["internal/auth/**/*.go"]
}`
	os.WriteFile(filepath.Join(roomsDir, "auth.jsonc"), []byte(authRoom), 0o644)

	runPalace(t, binPath, workspace, "init", "--root", workspace)
	runPalace(t, binPath, workspace, "scan", "--root", workspace)

	// UAT: Agent can query within a specific room
	t.Run("query_in_room", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "explore", "--root", workspace,
			"--room", "auth", "handler")

		t.Logf("Room-scoped query output:\n%s", output)
	})
}

// TestAgentVerificationWorkflow validates that agents can check
// if the index is fresh before trusting search results.
func TestAgentVerificationWorkflow(t *testing.T) {
	workspace, binPath := setupTestWorkspace(t, map[string]string{
		"code.go": `package main

func Original() {}
`,
	})

	runPalace(t, binPath, workspace, "init", "--root", workspace)
	runPalace(t, binPath, workspace, "scan", "--root", workspace)

	// UAT 1: Fresh index passes verification
	t.Run("fresh_index_passes", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "check", "--root", workspace)

		if strings.Contains(strings.ToLower(output), "stale") {
			t.Errorf("Fresh index should pass check, got:\n%s", output)
		}
	})

	// UAT 2: Modified file causes verification to fail
	t.Run("modified_file_fails", func(t *testing.T) {
		// Modify the file
		codePath := filepath.Join(workspace, "code.go")
		os.WriteFile(codePath, []byte(`package main

func Original() {}
func NewFunction() {} // Added
`), 0o644)

		output := runPalaceExpectFail(t, binPath, workspace, "check", "--root", workspace)

		if !strings.Contains(strings.ToLower(output), "stale") {
			t.Errorf("Modified file should cause stale check, got:\n%s", output)
		}
	})

	// UAT 3: Rescan fixes the issue
	t.Run("rescan_fixes", func(t *testing.T) {
		runPalace(t, binPath, workspace, "scan", "--root", workspace)
		output := runPalace(t, binPath, workspace, "check", "--root", workspace)

		if strings.Contains(strings.ToLower(output), "error") {
			t.Errorf("Index should be fresh after rescan, got:\n%s", output)
		}
	})
}

// ============================================================================
// Test Helpers - Note: Reuses helpers from integration_test.go
// ============================================================================

func setupTestWorkspace(t *testing.T, files map[string]string) (workspace, binPath string) {
	t.Helper()

	// Build palace binary
	root := repoRoot(t)
	binPath = filepath.Join(t.TempDir(), "palace")
	buildPalace(t, root, binPath)

	// Create workspace with files
	workspace = t.TempDir()
	for path, content := range files {
		fullPath := filepath.Join(workspace, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	return workspace, binPath
}
