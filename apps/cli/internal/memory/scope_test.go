package memory

import (
	"os"
	"testing"
)

// setupTestMemory creates a temporary memory database for testing.
func setupTestMemory(t *testing.T) *Memory {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "scope-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	mem, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	t.Cleanup(func() { mem.Close() })

	return mem
}

func TestExpandScope_FileScope(t *testing.T) {
	// Test file scope expansion with room resolver
	roomResolver := func(path string) string {
		if path == "src/auth/jwt.go" {
			return "authentication"
		}
		return ""
	}

	chain := ExpandScope(ScopeFile, "src/auth/jwt.go", roomResolver)

	if len(chain) != 3 {
		t.Fatalf("expected 3 levels, got %d", len(chain))
	}

	// Verify file level
	if chain[0].Scope != ScopeFile {
		t.Errorf("expected file scope, got %s", chain[0].Scope)
	}
	if chain[0].Path != "src/auth/jwt.go" {
		t.Errorf("expected path 'src/auth/jwt.go', got %s", chain[0].Path)
	}
	if chain[0].Priority != 0 {
		t.Errorf("expected priority 0, got %d", chain[0].Priority)
	}

	// Verify room level
	if chain[1].Scope != ScopeRoom {
		t.Errorf("expected room scope, got %s", chain[1].Scope)
	}
	if chain[1].Path != "authentication" {
		t.Errorf("expected path 'authentication', got %s", chain[1].Path)
	}
	if chain[1].Priority != 1 {
		t.Errorf("expected priority 1, got %d", chain[1].Priority)
	}

	// Verify palace level
	if chain[2].Scope != ScopePalace {
		t.Errorf("expected palace scope, got %s", chain[2].Scope)
	}
	if chain[2].Path != "" {
		t.Errorf("expected empty path, got %s", chain[2].Path)
	}
	if chain[2].Priority != 2 {
		t.Errorf("expected priority 2, got %d", chain[2].Priority)
	}
}

func TestExpandScope_FileScopeNoRoom(t *testing.T) {
	// Test file scope expansion without room resolver
	chain := ExpandScope(ScopeFile, "src/unknown/file.go", nil)

	if len(chain) != 2 {
		t.Fatalf("expected 2 levels (file + palace), got %d", len(chain))
	}

	// Verify file level
	if chain[0].Scope != ScopeFile {
		t.Errorf("expected file scope, got %s", chain[0].Scope)
	}

	// Verify palace level
	if chain[1].Scope != ScopePalace {
		t.Errorf("expected palace scope, got %s", chain[1].Scope)
	}
}

func TestExpandScope_RoomScope(t *testing.T) {
	chain := ExpandScope(ScopeRoom, "api", nil)

	if len(chain) != 2 {
		t.Fatalf("expected 2 levels, got %d", len(chain))
	}

	// Verify room level
	if chain[0].Scope != ScopeRoom {
		t.Errorf("expected room scope, got %s", chain[0].Scope)
	}
	if chain[0].Path != "api" {
		t.Errorf("expected path 'api', got %s", chain[0].Path)
	}
	if chain[0].Priority != 1 {
		t.Errorf("expected priority 1, got %d", chain[0].Priority)
	}

	// Verify palace level
	if chain[1].Scope != ScopePalace {
		t.Errorf("expected palace scope, got %s", chain[1].Scope)
	}
	if chain[1].Priority != 2 {
		t.Errorf("expected priority 2, got %d", chain[1].Priority)
	}
}

func TestExpandScope_PalaceScope(t *testing.T) {
	chain := ExpandScope(ScopePalace, "", nil)

	if len(chain) != 1 {
		t.Fatalf("expected 1 level, got %d", len(chain))
	}

	// Verify palace level only
	if chain[0].Scope != ScopePalace {
		t.Errorf("expected palace scope, got %s", chain[0].Scope)
	}
	if chain[0].Path != "" {
		t.Errorf("expected empty path, got %s", chain[0].Path)
	}
	if chain[0].Priority != 2 {
		t.Errorf("expected priority 2, got %d", chain[0].Priority)
	}
}

func TestExpandScope_UnknownScope(t *testing.T) {
	// Unknown scope should default to palace
	chain := ExpandScope(Scope("unknown"), "", nil)

	if len(chain) != 1 {
		t.Fatalf("expected 1 level, got %d", len(chain))
	}

	if chain[0].Scope != ScopePalace {
		t.Errorf("expected palace scope, got %s", chain[0].Scope)
	}
}

func TestExpandScope_Deterministic(t *testing.T) {
	// Same inputs should always produce same outputs
	roomResolver := func(path string) string {
		if path == "test.go" {
			return "testing"
		}
		return ""
	}

	chain1 := ExpandScope(ScopeFile, "test.go", roomResolver)
	chain2 := ExpandScope(ScopeFile, "test.go", roomResolver)

	if len(chain1) != len(chain2) {
		t.Fatalf("chains have different lengths: %d vs %d", len(chain1), len(chain2))
	}

	for i := range chain1 {
		if chain1[i].Scope != chain2[i].Scope {
			t.Errorf("scope mismatch at %d: %s vs %s", i, chain1[i].Scope, chain2[i].Scope)
		}
		if chain1[i].Path != chain2[i].Path {
			t.Errorf("path mismatch at %d: %s vs %s", i, chain1[i].Path, chain2[i].Path)
		}
		if chain1[i].Priority != chain2[i].Priority {
			t.Errorf("priority mismatch at %d: %d vs %d", i, chain1[i].Priority, chain2[i].Priority)
		}
	}
}

func TestTruncateContent_NoTruncation(t *testing.T) {
	cfg := &AuthoritativeQueryConfig{MaxContentLen: 100}

	content := "Short content"
	result := cfg.TruncateContent(content)

	if result != content {
		t.Errorf("expected no truncation, got %s", result)
	}
}

func TestTruncateContent_WithTruncation(t *testing.T) {
	cfg := &AuthoritativeQueryConfig{MaxContentLen: 20}

	content := "This is a very long content that should be truncated"
	result := cfg.TruncateContent(content)

	if len(result) != 20 {
		t.Errorf("expected length 20, got %d", len(result))
	}
	if result[len(result)-3:] != "..." {
		t.Errorf("expected ellipsis at end, got %s", result)
	}
	// Content should be first 17 chars + "..."
	expected := content[:17] + "..."
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestTruncateContent_ExactLength(t *testing.T) {
	cfg := &AuthoritativeQueryConfig{MaxContentLen: 10}

	content := "Exactly 10"
	result := cfg.TruncateContent(content)

	if result != content {
		t.Errorf("expected no truncation for exact length, got %s", result)
	}
}

func TestTruncateContent_VeryShortLimit(t *testing.T) {
	cfg := &AuthoritativeQueryConfig{MaxContentLen: 3}

	content := "Long content"
	result := cfg.TruncateContent(content)

	if result != "..." {
		t.Errorf("expected '...' for very short limit, got %s", result)
	}
}

func TestTruncateContent_ZeroLimit(t *testing.T) {
	cfg := &AuthoritativeQueryConfig{MaxContentLen: 0}

	content := "Some content"
	result := cfg.TruncateContent(content)

	// Zero limit should return original content (no truncation)
	if result != content {
		t.Errorf("expected original content for zero limit, got %s", result)
	}
}

func TestTruncateContent_Deterministic(t *testing.T) {
	cfg := &AuthoritativeQueryConfig{MaxContentLen: 15}

	content := "This is deterministic content"

	result1 := cfg.TruncateContent(content)
	result2 := cfg.TruncateContent(content)

	if result1 != result2 {
		t.Errorf("truncation is not deterministic: %s vs %s", result1, result2)
	}
}

func TestDefaultAuthoritativeQueryConfig(t *testing.T) {
	cfg := DefaultAuthoritativeQueryConfig()

	if cfg.MaxDecisions != 10 {
		t.Errorf("expected MaxDecisions 10, got %d", cfg.MaxDecisions)
	}
	if cfg.MaxLearnings != 10 {
		t.Errorf("expected MaxLearnings 10, got %d", cfg.MaxLearnings)
	}
	if cfg.MaxContentLen != 500 {
		t.Errorf("expected MaxContentLen 500, got %d", cfg.MaxContentLen)
	}
	if !cfg.AuthoritativeOnly {
		t.Error("expected AuthoritativeOnly true by default")
	}
}

func TestGetAuthoritativeState(t *testing.T) {
	mem := setupTestMemory(t)

	// Add some test decisions with different scopes and authorities
	_, err := mem.AddDecision(Decision{
		Content:   "Palace decision",
		Scope:     "palace",
		ScopePath: "",
		Status:    "active",
		Source:    "test",
		Authority: string(AuthorityApproved),
	})
	if err != nil {
		t.Fatalf("failed to add palace decision: %v", err)
	}

	_, err = mem.AddDecision(Decision{
		Content:   "Room decision",
		Scope:     "room",
		ScopePath: "test-room",
		Status:    "active",
		Source:    "test",
		Authority: string(AuthorityApproved),
	})
	if err != nil {
		t.Fatalf("failed to add room decision: %v", err)
	}

	_, err = mem.AddDecision(Decision{
		Content:   "Proposed decision should not appear",
		Scope:     "palace",
		ScopePath: "",
		Status:    "active",
		Source:    "agent",
		Authority: string(AuthorityProposed),
	})
	if err != nil {
		t.Fatalf("failed to add proposed decision: %v", err)
	}

	// Add some test learnings
	_, err = mem.AddLearning(Learning{
		Content:   "Palace learning",
		Scope:     "palace",
		ScopePath: "",
		Source:    "test",
		Authority: string(AuthorityLegacyApproved),
	})
	if err != nil {
		t.Fatalf("failed to add palace learning: %v", err)
	}

	// Room resolver
	roomResolver := func(path string) string {
		if path == "test-file.go" {
			return "test-room"
		}
		return ""
	}

	cfg := &AuthoritativeQueryConfig{
		MaxDecisions:      10,
		MaxLearnings:      10,
		MaxContentLen:     500,
		AuthoritativeOnly: true,
	}

	result, err := mem.GetAuthoritativeState(ScopeFile, "test-file.go", roomResolver, cfg)
	if err != nil {
		t.Fatalf("failed to get authoritative state: %v", err)
	}

	// Verify scope chain
	if len(result.ScopeChain) != 3 {
		t.Errorf("expected 3 scope levels, got %d", len(result.ScopeChain))
	}

	// Verify decisions - should not include proposed one
	foundPalace := false
	foundRoom := false
	for _, sd := range result.Decisions {
		if sd.Decision.Content == "Palace decision" {
			foundPalace = true
		}
		if sd.Decision.Content == "Room decision" {
			foundRoom = true
		}
		if sd.Decision.Content == "Proposed decision should not appear" {
			t.Error("proposed decision should not be included")
		}
	}
	if !foundPalace {
		t.Error("palace decision should be included")
	}
	if !foundRoom {
		t.Error("room decision should be included")
	}

	// Verify learnings
	if len(result.Learnings) != 1 {
		t.Errorf("expected 1 learning, got %d", len(result.Learnings))
	}
}

func TestGetAuthoritativeState_BoundsEnforced(t *testing.T) {
	mem := setupTestMemory(t)
	
	// Add more decisions than the limit
	for i := 0; i < 15; i++ {
		_, err := mem.AddDecision(Decision{
			Content:   "Decision " + string(rune('A'+i)),
			Scope:     "palace",
			ScopePath: "",
			Status:    "active",
			Source:    "test",
			Authority: string(AuthorityApproved),
		})
		if err != nil {
			t.Fatalf("failed to add decision: %v", err)
		}
	}

	cfg := &AuthoritativeQueryConfig{
		MaxDecisions:      5,
		MaxLearnings:      5,
		MaxContentLen:     500,
		AuthoritativeOnly: true,
	}

	result, err := mem.GetAuthoritativeState(ScopePalace, "", nil, cfg)
	if err != nil {
		t.Fatalf("failed to get authoritative state: %v", err)
	}

	// Should be bounded by MaxDecisions
	if len(result.Decisions) > 5 {
		t.Errorf("expected at most 5 decisions, got %d", len(result.Decisions))
	}
}

func TestGetAuthoritativeState_ContentTruncation(t *testing.T) {
	mem := setupTestMemory(t)
	
	longContent := "This is a very long decision content that should be truncated to fit within the maximum content length limit set in the configuration"

	_, err := mem.AddDecision(Decision{
		Content:   longContent,
		Scope:     "palace",
		ScopePath: "",
		Status:    "active",
		Source:    "test",
		Authority: string(AuthorityApproved),
	})
	if err != nil {
		t.Fatalf("failed to add decision: %v", err)
	}

	cfg := &AuthoritativeQueryConfig{
		MaxDecisions:      10,
		MaxLearnings:      10,
		MaxContentLen:     50,
		AuthoritativeOnly: true,
	}

	result, err := mem.GetAuthoritativeState(ScopePalace, "", nil, cfg)
	if err != nil {
		t.Fatalf("failed to get authoritative state: %v", err)
	}

	if len(result.Decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(result.Decisions))
	}

	if len(result.Decisions[0].Decision.Content) > 50 {
		t.Errorf("content should be truncated to 50 chars, got %d", len(result.Decisions[0].Decision.Content))
	}

	if !result.Truncated {
		t.Error("expected Truncated flag to be true")
	}
}

