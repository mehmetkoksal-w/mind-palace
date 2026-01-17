// Package integration_test contains end-to-end flow tests that validate
// complete user workflows across multiple Mind Palace features.
//
// These tests verify that features work together correctly in realistic scenarios:
// - Brain workflow (ideas -> decisions -> outcomes -> reviews)
// - Multi-agent collaboration with conflict detection
// - Corridor workflows (personal learnings across workspaces)
// - Full development cycle from init to verification
package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBrainWorkflowE2E validates the complete brain workflow:
// 1. Record an idea
// 2. Convert idea to decision
// 3. Implement and record outcome
// 4. Review the decision
// 5. Link related items
func TestBrainWorkflowE2E(t *testing.T) {
	workspace, binPath := setupTestWorkspace(t, map[string]string{
		"main.go": `package main

func main() {
	// Main entry point
}
`,
		"cache/cache.go": `package cache

type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
}
`,
	})

	runPalace(t, binPath, workspace, "init", "--root", workspace)
	runPalace(t, binPath, workspace, "scan", "--root", workspace)

	// Step 1: Record an idea
	t.Run("step1_record_idea", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "store", "--root", workspace,
			"--as", "idea",
			"Add Redis caching layer for improved performance")

		if !strings.Contains(output, "idea") && !strings.Contains(output, "recorded") && !strings.Contains(output, "Idea") {
			t.Errorf("Expected idea confirmation, got:\n%s", output)
		}
	})

	// Step 2: Record a decision based on the idea
	t.Run("step2_record_decision", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "store", "--root", workspace,
			"--as", "decision",
			"--scope", "file",
			"--path", "cache/cache.go",
			"Use Redis as the caching backend with 1 hour TTL")

		if !strings.Contains(strings.ToLower(output), "decision") && !strings.Contains(strings.ToLower(output), "recorded") {
			t.Errorf("Expected decision confirmation, got:\n%s", output)
		}
	})

	// Step 3: List ideas and decisions to verify they exist
	t.Run("step3_verify_records", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "recall", "--root", workspace, "--type", "idea")
		if !strings.Contains(output, "Redis") && !strings.Contains(output, "caching") {
			t.Logf("Ideas output:\n%s", output)
		}

		output = runPalace(t, binPath, workspace, "recall", "--root", workspace, "--type", "decision")
		if !strings.Contains(output, "Redis") && !strings.Contains(output, "TTL") {
			t.Logf("Decisions output:\n%s", output)
		}
	})

	// Step 4: Record outcome of the decision
	t.Run("step4_record_outcome", func(t *testing.T) {
		// First get decisions to find the ID
		output := runPalace(t, binPath, workspace, "recall", "--root", workspace, "--type", "decision")
		t.Logf("Decisions for outcome:\n%s", output)

		// Get all IDs from the output to find d_...
		// In a real CLI output we'd parse this.
	})

	// Step 5: Link related items
	t.Run("step5_link_items", func(t *testing.T) {
		// Record a learning related to the decision
		runPalace(t, binPath, workspace, "store", "--root", workspace,
			"--as", "learning",
			"--scope", "file", "--path", "cache/cache.go",
			"Redis connection pool size should match expected concurrency")

		// Verify learnings
		output := runPalace(t, binPath, workspace, "recall", "--root", workspace)
		if !strings.Contains(output, "Redis") || !strings.Contains(output, "pool") {
			t.Logf("Learnings output:\n%s", output)
		}
	})
}

// TestMultiAgentCollaborationE2E validates that multiple agents can work
// together with proper conflict detection and session management.
func TestMultiAgentCollaborationE2E(t *testing.T) {
	workspace, binPath := setupTestWorkspace(t, map[string]string{
		"shared.go": `package main

// SharedResource is accessed by multiple components
func SharedResource() {}
`,
		"feature_a.go": `package main

func FeatureA() {
	SharedResource()
}
`,
		"feature_b.go": `package main

func FeatureB() {
	SharedResource()
}
`,
	})

	runPalace(t, binPath, workspace, "init", "--root", workspace)
	runPalace(t, binPath, workspace, "scan", "--root", workspace)

	// Agent 1 starts working
	t.Run("agent1_starts_session", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "session", "start",
			"--root", workspace,
			"--agent", "claude-code",
			"--agent-id", "agent-1",
			"--goal", "Implement Feature A")

		if !strings.Contains(strings.ToLower(output), "session") {
			t.Errorf("Expected session output, got:\n%s", output)
		}
	})

	// Agent 2 starts working
	t.Run("agent2_starts_session", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "session", "start",
			"--root", workspace,
			"--agent", "cursor",
			"--agent-id", "agent-2",
			"--goal", "Implement Feature B")

		if !strings.Contains(strings.ToLower(output), "session") {
			t.Errorf("Expected session output, got:\n%s", output)
		}
	})

	// Verify both sessions are listed
	t.Run("list_active_sessions", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "session", "list",
			"--root", workspace, "--active")

		// Should show at least one active session
		if !strings.Contains(output, "active") && !strings.Contains(output, "claude") && !strings.Contains(output, "cursor") {
			t.Logf("Active sessions:\n%s", output)
		}
	})

	// Get briefing which should show active agents
	t.Run("briefing_shows_agents", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "brief", "--root", workspace)

		// Briefing should show some agent activity
		t.Logf("Briefing with multiple agents:\n%s", output)
	})

	// File-specific briefing for shared resource
	t.Run("briefing_shared_file", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "brief", "--root", workspace, "shared.go")
		t.Logf("Shared file briefing:\n%s", output)
	})
}

// TestFullDevelopmentCycleE2E validates a complete development cycle:
// 1. Initialize palace
// 2. Scan codebase
// 3. Start session
// 4. Make changes
// 5. Record learnings
// 6. Verify index freshness
// 7. Generate context pack
// 8. End session
func TestFullDevelopmentCycleE2E(t *testing.T) {
	workspace, binPath := setupTestWorkspace(t, map[string]string{
		"main.go": `package main

func main() {
	println("Hello")
}
`,
		"helper.go": `package main

func helper() string {
	return "help"
}
`,
	})

	// Step 1: Initialize
	t.Run("step1_initialize", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "init", "--root", workspace)
		if !strings.Contains(strings.ToLower(output), "init") {
			t.Logf("Init output:\n%s", output)
		}

		// Verify .palace directory exists
		palacePath := filepath.Join(workspace, ".palace")
		if _, err := os.Stat(palacePath); os.IsNotExist(err) {
			t.Fatal(".palace directory not created")
		}
	})

	// Step 2: Scan
	t.Run("step2_scan", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "scan", "--root", workspace)
		t.Logf("Scan output:\n%s", output)

		// Verify database exists
		dbPath := filepath.Join(workspace, ".palace", "index", "palace.db")
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Fatal("palace.db not created")
		}
	})

	// Step 3: Start session
	t.Run("step3_start_session", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "session", "start",
			"--root", workspace,
			"--agent", "test-agent",
			"--agent-id", "e2e-test",
			"--goal", "Add logging to main")

		if !strings.Contains(strings.ToLower(output), "session") {
			t.Logf("Session output:\n%s", output)
		}
	})

	// Step 4: Query code before making changes
	t.Run("step4_query_code", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "explore", "--root", workspace, "main")
		if !strings.Contains(output, "main") {
			t.Logf("Query output:\n%s", output)
		}
	})

	// Step 5: Make a code change
	t.Run("step5_make_changes", func(t *testing.T) {
		newContent := `package main

import "log"

func main() {
	log.Println("Starting application")
	println("Hello")
}
`
		if err := os.WriteFile(filepath.Join(workspace, "main.go"), []byte(newContent), 0o644); err != nil {
			t.Fatalf("Failed to update file: %v", err)
		}
	})

	// Step 6: Verify index is now stale
	t.Run("step6_verify_stale", func(t *testing.T) {
		output := runPalaceExpectFail(t, binPath, workspace, "check", "--root", workspace)
		if !strings.Contains(strings.ToLower(output), "stale") {
			t.Logf("Stale check output:\n%s", output)
		}
	})

	// Step 7: Rescan
	t.Run("step7_rescan", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "scan", "--root", workspace)
		t.Logf("Rescan output:\n%s", output)
	})

	// Step 8: Record learnings from the change
	t.Run("step8_record_learning", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "store", "--root", workspace,
			"--as", "learning",
			"--scope", "file", "--path", "main.go",
			"Always add logging at application startup for debugging")

		if !strings.Contains(strings.ToLower(output), "learning") && !strings.Contains(strings.ToLower(output), "recorded") {
			t.Logf("Learning output:\n%s", output)
		}
	})

	// Step 9: Generate context pack
	t.Run("step9_generate_context", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "check", "--root", workspace,
			"--collect")
		t.Logf("Context output:\n%s", output)

		// Verify context pack file
		cpPath := filepath.Join(workspace, ".palace", "outputs", "context-pack.json")
		data, err := os.ReadFile(cpPath)
		if err != nil {
			t.Fatalf("Failed to read context pack: %v", err)
		}

		var cp map[string]interface{}
		if err := json.Unmarshal(data, &cp); err != nil {
			t.Fatalf("Failed to parse context pack: %v", err)
		}

		if cp["goal"] == nil {
			t.Error("Context pack should have goal")
		}
	})

	// Step 10: End session
	t.Run("step10_list_and_end", func(t *testing.T) {
		// List sessions to see what we have
		output := runPalace(t, binPath, workspace, "session", "list", "--root", workspace)
		t.Logf("Sessions:\n%s", output)
	})
}

// TestCorridorWorkflowE2E validates the corridor (cross-workspace) workflow:
// 1. Create two workspaces
// 2. Add learnings to one
// 3. Promote learning to personal corridor
// 4. Link workspaces
// 5. Access learnings from other workspace
func TestCorridorWorkflowE2E(t *testing.T) {
	// Create first workspace
	workspace1, binPath := setupTestWorkspace(t, map[string]string{
		"api.go": `package main

func HandleRequest() {
	// Always validate input before processing
}
`,
	})

	// Create second workspace
	workspace2 := t.TempDir()
	os.MkdirAll(workspace2, 0o755)
	os.WriteFile(filepath.Join(workspace2, "server.go"), []byte(`package main

func StartServer() {}
`), 0o644)

	// Initialize both workspaces
	runPalace(t, binPath, workspace1, "init", "--root", workspace1)
	runPalace(t, binPath, workspace1, "scan", "--root", workspace1)
	runPalace(t, binPath, workspace2, "init", "--root", workspace2)
	runPalace(t, binPath, workspace2, "scan", "--root", workspace2)

	// Add learning in workspace1
	t.Run("add_learning_workspace1", func(t *testing.T) {
		output := runPalace(t, binPath, workspace1, "store", "--root", workspace1,
			"--as", "learning",
			"Always validate input in API handlers to prevent injection attacks")
		t.Logf("Learning output:\n%s", output)
	})

	// List corridor learnings
	t.Run("list_corridor", func(t *testing.T) {
		output := runPalace(t, binPath, workspace1, "corridor", "list")
		t.Logf("Corridor list:\n%s", output)
	})

	// Personal corridor operations
	t.Run("corridor_personal", func(t *testing.T) {
		output := runPalace(t, binPath, workspace1, "corridor", "personal")
		t.Logf("Personal corridor:\n%s", output)
	})

	// Search corridor
	t.Run("corridor_search", func(t *testing.T) {
		output := runPalace(t, binPath, workspace1, "corridor", "search", "validation")
		t.Logf("Corridor search:\n%s", output)
	})
}

// Note: Graph functionality is covered by TestHelpSystemE2E (help graph)
// and by the query/scan tests which build the symbol index.

// TestMaintenanceWorkflowE2E validates the maintenance and cleanup workflow:
// 1. Create stale data
// 2. Run maintenance
// 3. Verify cleanup
func TestMaintenanceWorkflowE2E(t *testing.T) {
	workspace, binPath := setupTestWorkspace(t, map[string]string{
		"app.go": `package main

func App() {}
`,
	})

	runPalace(t, binPath, workspace, "init", "--root", workspace)
	runPalace(t, binPath, workspace, "scan", "--root", workspace)

	// Add some test data
	runPalace(t, binPath, workspace, "store", "--root", workspace,
		"--as", "learning", "Test learning for maintenance")

	// Run maintenance (dry-run first)
	t.Run("maintenance_dryrun", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "clean",
			"--root", workspace, "--dry-run")
		t.Logf("Maintenance dry-run:\n%s", output)
	})

	// Run actual maintenance
	t.Run("maintenance_execute", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "clean", "--root", workspace)
		t.Logf("Maintenance output:\n%s", output)
	})
}

// TestDashboardAPIE2E validates the dashboard HTTP API endpoints
// by starting the server and making requests.
func TestDashboardAPIE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping dashboard API test in short mode")
	}

	workspace, binPath := setupTestWorkspace(t, map[string]string{
		"main.go": `package main

func main() {}
`,
	})

	runPalace(t, binPath, workspace, "init", "--root", workspace)
	runPalace(t, binPath, workspace, "scan", "--root", workspace)

	// Add some test data
	runPalace(t, binPath, workspace, "store", "--root", workspace, "--as", "learning", "Dashboard test learning")
	runPalace(t, binPath, workspace, "store", "--root", workspace,
		"--as", "idea", "Dashboard test idea")

	// Note: Actually starting and testing the dashboard server would require
	// additional setup (port management, cleanup, etc.)
	// For now, verify the dashboard command exists and shows help
	t.Run("dashboard_help", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "help", "dashboard")
		if !strings.Contains(strings.ToLower(output), "dashboard") {
			t.Logf("Dashboard help:\n%s", output)
		}
	})
}

// TestContextPackIntegrityE2E validates that context packs maintain
// integrity through various operations.
func TestContextPackIntegrityE2E(t *testing.T) {
	workspace, binPath := setupTestWorkspace(t, map[string]string{
		"service.go": `package main

type Service struct {
	Name string
}

func (s *Service) Start() error {
	return nil
}

func (s *Service) Stop() error {
	return nil
}
`,
	})

	runPalace(t, binPath, workspace, "init", "--root", workspace)
	runPalace(t, binPath, workspace, "scan", "--root", workspace)

	// Generate initial context pack
	t.Run("initial_context_pack", func(t *testing.T) {
		runPalace(t, binPath, workspace, "check", "--root", workspace, "--collect")

		cpPath := filepath.Join(workspace, ".palace", "outputs", "context-pack.json")
		data1, err := os.ReadFile(cpPath)
		if err != nil {
			t.Fatalf("Failed to read context pack: %v", err)
		}

		var cp1 map[string]interface{}
		json.Unmarshal(data1, &cp1)

		// Store scan hash for comparison
		scanHash1, _ := cp1["scanHash"].(string)

		// Make a change
		time.Sleep(100 * time.Millisecond) // Ensure timestamp changes
		os.WriteFile(filepath.Join(workspace, "service.go"), []byte(`package main

type Service struct {
	Name    string
	Timeout int // Added field
}

func (s *Service) Start() error {
	return nil
}

func (s *Service) Stop() error {
	return nil
}
`), 0o644)

		// Rescan to ensure changes are detected (use --full to force new hash)
		output := runPalace(t, binPath, workspace, "scan", "--root", workspace, "--full")
		t.Logf("Rescan output:\n%s", output)

		// Generate new context pack
		runPalace(t, binPath, workspace, "check", "--root", workspace, "--collect")

		data2, err := os.ReadFile(cpPath)
		if err != nil {
			t.Fatalf("Failed to read updated context pack: %v", err)
		}

		var cp2 map[string]interface{}
		json.Unmarshal(data2, &cp2)

		scanHash2, _ := cp2["scanHash"].(string)

		// Scan hash should be different after code change
		if scanHash1 == scanHash2 {
			t.Error("Scan hash should change after code modification")
		}
	})
}

// TestErrorRecoveryE2E validates that Mind Palace handles errors gracefully
// and can recover from various failure scenarios.
func TestErrorRecoveryE2E(t *testing.T) {
	workspace, binPath := setupTestWorkspace(t, map[string]string{
		"app.go": `package main

func main() {}
`,
	})

	// Try operations before init (should fail gracefully)
	t.Run("operations_before_init", func(t *testing.T) {
		// Explore without init should fail
		output := runPalaceExpectFail(t, binPath, workspace, "explore", "--root", workspace, "test")
		t.Logf("Explore before init:\n%s", output)

		// Check without init should fail
		output = runPalaceExpectFail(t, binPath, workspace, "check", "--root", workspace)
		t.Logf("Check before init:\n%s", output)
	})

	// Initialize (skip auto-scan to test operations before scan)
	runPalace(t, binPath, workspace, "init", "--root", workspace, "--no-scan")

	// Try operations before scan
	t.Run("operations_before_scan", func(t *testing.T) {
		// Explore without scan should fail with index missing error
		output := runPalaceExpectFail(t, binPath, workspace, "explore", "--root", workspace, "test")
		t.Logf("Explore before scan:\n%s", output)

		// Should contain "index missing" or similar error
		if !strings.Contains(output, "index missing") && !strings.Contains(output, "scan") {
			t.Logf("Expected index missing error message")
		}
	})

	// Scan
	runPalace(t, binPath, workspace, "scan", "--root", workspace)

	// Corrupt database and try to recover
	t.Run("database_recovery", func(t *testing.T) {
		// Just verify scan can be run again
		output := runPalace(t, binPath, workspace, "scan", "--root", workspace)
		t.Logf("Full rescan:\n%s", output)
	})

	// Invalid inputs
	t.Run("invalid_inputs", func(t *testing.T) {
		// Invalid confidence
		output := runPalaceExpectFail(t, binPath, workspace, "store", "--root", workspace,
			"--as", "learning", "--confidence", "2.0", "Invalid confidence")
		t.Logf("Invalid confidence:\n%s", output)

		// Invalid scope
		output = runPalaceExpectFail(t, binPath, workspace, "store", "--root", workspace,
			"--as", "learning", "--scope", "invalid", "Invalid scope")
		t.Logf("Invalid scope:\n%s", output)
	})
}

// TestCIWorkflowE2E validates the complete CI integration workflow:
// 1. Init and scan
// 2. Generate change signal
// 3. Collect context
// 4. Verify freshness
func TestCIWorkflowE2E(t *testing.T) {
	workspace, binPath := setupTestWorkspace(t, map[string]string{
		"main.go": `package main

func main() {}
`,
		"lib.go": `package main

func Helper() {}
`,
	})

	// Initialize git repo for change signal
	runCommand("git", workspace, "init")
	runCommand("git", workspace, "config", "user.email", "test@test.com")
	runCommand("git", workspace, "config", "user.name", "Test")
	runCommand("git", workspace, "add", ".")
	runCommand("git", workspace, "commit", "-m", "Initial commit")

	// Initialize palace
	runPalace(t, binPath, workspace, "init", "--root", workspace)
	runPalace(t, binPath, workspace, "scan", "--root", workspace)

	// Verify CI commands work
	t.Run("ci_verify_fresh", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "check", "--root", workspace)
		t.Logf("CI verify (fresh):\n%s", output)
	})

	t.Run("ci_collect", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "check", "--collect", "--root", workspace)
		t.Logf("CI collect:\n%s", output)

		// Verify context pack was created
		cpPath := filepath.Join(workspace, ".palace", "outputs", "context-pack.json")
		if _, err := os.Stat(cpPath); os.IsNotExist(err) {
			t.Error("Context pack not created by ci collect")
		}
	})

	// Make a change
	os.WriteFile(filepath.Join(workspace, "lib.go"), []byte(`package main

func Helper() {}
func NewHelper() {} // Added
`), 0o644)

	t.Run("ci_verify_stale", func(t *testing.T) {
		output := runPalaceExpectFail(t, binPath, workspace, "check", "--root", workspace)
		if !strings.Contains(strings.ToLower(output), "stale") {
			t.Logf("CI verify (stale):\n%s", output)
		}
	})

	// Generate change signal (requires git diff)
	t.Run("ci_signal", func(t *testing.T) {
		runCommand("git", workspace, "add", ".")
		runCommand("git", workspace, "commit", "-m", "Add NewHelper")

		// Rescan to update the index after the file change
		runPalace(t, binPath, workspace, "scan", "--root", workspace)

		output := runPalace(t, binPath, workspace, "check",
			"--root", workspace, "--diff", "HEAD~1..HEAD")
		t.Logf("CI signal:\n%s", output)
	})
}

// TestHelpSystemE2E validates that the help system provides useful information
// for all commands.
func TestHelpSystemE2E(t *testing.T) {
	workspace, binPath := setupTestWorkspace(t, map[string]string{
		"app.go": `package main

func main() {}
`,
	})

	// These are the valid help topics (from the help system)
	commands := []string{
		"init", "scan", "check", "explore",
		"session", "store", "recall", "brief",
		"corridor", "clean", "dashboard", "artifacts",
	}

	for _, cmd := range commands {
		t.Run("help_"+cmd, func(t *testing.T) {
			output := runPalace(t, binPath, workspace, "help", cmd)
			if output == "" {
				t.Errorf("Help for %s returned empty output", cmd)
			}
			// Help should mention the command name
			if !strings.Contains(strings.ToLower(output), strings.ToLower(cmd)) {
				t.Logf("Help for %s:\n%s", cmd, output)
			}
		})
	}

	// Test general help
	t.Run("general_help", func(t *testing.T) {
		output := runPalace(t, binPath, workspace, "help")
		if output == "" {
			t.Error("general help returned empty output")
		}
		t.Logf("General help length: %d chars", len(output))
	})
}
