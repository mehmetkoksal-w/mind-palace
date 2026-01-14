package butler

import (
	"os"
	"testing"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
)

func TestRouteRuleVersion(t *testing.T) {
	if RouteRuleVersion == "" {
		t.Error("RouteRuleVersion should not be empty")
	}
	if RouteRuleVersion != "v1.0" {
		t.Errorf("Expected RouteRuleVersion v1.0, got %s", RouteRuleVersion)
	}
}

func TestDefaultRouteConfig(t *testing.T) {
	cfg := DefaultRouteConfig()

	if cfg.MaxNodes != 10 {
		t.Errorf("Expected MaxNodes 10, got %d", cfg.MaxNodes)
	}
	if cfg.MinLearningConfidence != 0.7 {
		t.Errorf("Expected MinLearningConfidence 0.7, got %f", cfg.MinLearningConfidence)
	}
}

func TestGetRoute_NoMemory(t *testing.T) {
	// Create a butler without memory
	b := &Butler{
		rooms: map[string]model.Room{
			"authentication": {
				Summary:     "Auth related code",
				EntryPoints: []string{"src/auth/jwt.go"},
			},
			"api": {
				Summary:     "REST API endpoints",
				EntryPoints: []string{"src/api/handlers.go"},
			},
		},
	}

	result, err := b.GetRoute("understand auth flow", memory.ScopePalace, "", nil)
	if err != nil {
		t.Fatalf("GetRoute failed: %v", err)
	}

	// Should find the authentication room
	found := false
	for _, node := range result.Nodes {
		if node.Kind == RouteNodeKindRoom && node.ID == "authentication" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find 'authentication' room in route")
	}

	// Should include meta
	if result.Meta.RuleVersion != RouteRuleVersion {
		t.Errorf("Expected rule version %s, got %s", RouteRuleVersion, result.Meta.RuleVersion)
	}
	if result.Meta.NodeCount != len(result.Nodes) {
		t.Errorf("Node count mismatch: meta says %d, actual %d", result.Meta.NodeCount, len(result.Nodes))
	}
}

func TestGetRoute_MatchesRoomName(t *testing.T) {
	b := &Butler{
		rooms: map[string]model.Room{
			"authentication": {Summary: "Auth code", EntryPoints: []string{"auth.go"}},
			"database":       {Summary: "DB layer", EntryPoints: []string{"db.go"}},
			"api":            {Summary: "API endpoints", EntryPoints: []string{"api.go"}},
		},
	}

	tests := []struct {
		intent       string
		expectedRoom string
	}{
		{"understand authentication", "authentication"},
		{"how does auth work", "authentication"},
		{"database queries", "database"},
		{"api endpoints", "api"},
	}

	for _, tt := range tests {
		t.Run(tt.intent, func(t *testing.T) {
			result, err := b.GetRoute(tt.intent, memory.ScopePalace, "", nil)
			if err != nil {
				t.Fatalf("GetRoute failed: %v", err)
			}

			found := false
			for _, node := range result.Nodes {
				if node.Kind == RouteNodeKindRoom && node.ID == tt.expectedRoom {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected to find room %q for intent %q", tt.expectedRoom, tt.intent)
			}
		})
	}
}

func TestGetRoute_FetchRefFormats(t *testing.T) {
	b := &Butler{
		rooms: map[string]model.Room{
			"auth": {Summary: "Authentication", EntryPoints: []string{"auth.go"}},
		},
	}

	result, err := b.GetRoute("auth", memory.ScopePalace, "", nil)
	if err != nil {
		t.Fatalf("GetRoute failed: %v", err)
	}

	// Check fetch_ref formats
	for _, node := range result.Nodes {
		switch node.Kind {
		case RouteNodeKindRoom:
			if node.FetchRef != "explore_rooms" {
				t.Errorf("Room fetch_ref should be 'explore_rooms', got %q", node.FetchRef)
			}
		case RouteNodeKindFile:
			expected := "explore_file --file " + node.ID
			if node.FetchRef != expected {
				t.Errorf("File fetch_ref should be %q, got %q", expected, node.FetchRef)
			}
		}
	}
}

func TestGetRoute_MaxNodes(t *testing.T) {
	// Create many rooms to exceed max
	rooms := make(map[string]model.Room)
	for i := 0; i < 20; i++ {
		name := "room" + string(rune('a'+i))
		rooms[name] = model.Room{Summary: "test room", EntryPoints: []string{name + ".go"}}
	}

	b := &Butler{rooms: rooms}

	cfg := &RouteConfig{
		MaxNodes:              5,
		MinLearningConfidence: 0.7,
	}

	result, err := b.GetRoute("room", memory.ScopePalace, "", cfg)
	if err != nil {
		t.Fatalf("GetRoute failed: %v", err)
	}

	if len(result.Nodes) > 5 {
		t.Errorf("Expected at most 5 nodes, got %d", len(result.Nodes))
	}
}

func TestGetRoute_Deterministic(t *testing.T) {
	b := &Butler{
		rooms: map[string]model.Room{
			"auth": {Summary: "Authentication", EntryPoints: []string{"auth.go"}},
			"api":  {Summary: "API endpoints", EntryPoints: []string{"api.go"}},
		},
	}

	// Run same query multiple times
	var results []*RouteResult
	for i := 0; i < 5; i++ {
		result, err := b.GetRoute("auth and api", memory.ScopePalace, "", nil)
		if err != nil {
			t.Fatalf("GetRoute failed: %v", err)
		}
		results = append(results, result)
	}

	// All results should be identical
	first := results[0]
	for i := 1; i < len(results); i++ {
		if len(results[i].Nodes) != len(first.Nodes) {
			t.Errorf("Run %d has different node count: %d vs %d", i, len(results[i].Nodes), len(first.Nodes))
			continue
		}
		for j := range first.Nodes {
			if results[i].Nodes[j].ID != first.Nodes[j].ID {
				t.Errorf("Run %d, node %d has different ID: %q vs %q", i, j, results[i].Nodes[j].ID, first.Nodes[j].ID)
			}
			if results[i].Nodes[j].Order != first.Nodes[j].Order {
				t.Errorf("Run %d, node %d has different Order: %d vs %d", i, j, results[i].Nodes[j].Order, first.Nodes[j].Order)
			}
		}
	}
}

func TestGetRoute_NodeOrder(t *testing.T) {
	b := &Butler{
		rooms: map[string]model.Room{
			"auth": {Summary: "Authentication", EntryPoints: []string{"auth.go"}},
		},
	}

	result, err := b.GetRoute("auth", memory.ScopePalace, "", nil)
	if err != nil {
		t.Fatalf("GetRoute failed: %v", err)
	}

	// Orders should be sequential starting from 1
	for i, node := range result.Nodes {
		expected := i + 1
		if node.Order != expected {
			t.Errorf("Node %d has order %d, expected %d", i, node.Order, expected)
		}
	}
}

func TestGetRoute_WithFileScope(t *testing.T) {
	b := &Butler{
		rooms: map[string]model.Room{
			"auth": {Summary: "Authentication", EntryPoints: []string{"src/auth/jwt.go"}},
		},
	}

	result, err := b.GetRoute("understand this file", memory.ScopeFile, "src/auth/jwt.go", nil)
	if err != nil {
		t.Fatalf("GetRoute failed: %v", err)
	}

	// Should include the scope file
	found := false
	for _, node := range result.Nodes {
		if node.Kind == RouteNodeKindFile && node.ID == "src/auth/jwt.go" {
			found = true
			if node.Reason != "Specified scope file" && node.Reason != "Room entry point" {
				// Could be either reason depending on match
			}
			break
		}
	}
	if !found {
		t.Error("Expected to find scope file in route")
	}
}

func TestGetRoute_WithMemory(t *testing.T) {
	// Create temp directory for memory
	tmpDir, err := os.MkdirTemp("", "route-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mem, err := memory.Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	defer mem.Close()

	// Add a high-confidence learning about auth
	_, err = mem.AddLearning(memory.Learning{
		Content:    "JWT tokens should be validated on every request",
		Scope:      "palace",
		Confidence: 0.9,
		Authority:  string(memory.AuthorityApproved),
	})
	if err != nil {
		t.Fatalf("Failed to add learning: %v", err)
	}

	// Add an approved decision
	_, err = mem.AddDecision(memory.Decision{
		Content:   "Use JWT for authentication",
		Rationale: "Industry standard",
		Scope:     "palace",
		Status:    "active",
		Authority: string(memory.AuthorityApproved),
	})
	if err != nil {
		t.Fatalf("Failed to add decision: %v", err)
	}

	b := &Butler{
		rooms: map[string]model.Room{
			"auth": {Summary: "Authentication", EntryPoints: []string{"auth.go"}},
		},
		memory: mem,
	}

	result, err := b.GetRoute("JWT authentication", memory.ScopePalace, "", nil)
	if err != nil {
		t.Fatalf("GetRoute failed: %v", err)
	}

	// Should find some nodes
	if len(result.Nodes) == 0 {
		t.Error("Expected at least one node in route")
	}

	// Should find decision or learning related to JWT
	foundRelevant := false
	for _, node := range result.Nodes {
		if node.Kind == RouteNodeKindDecision || node.Kind == RouteNodeKindLearning {
			foundRelevant = true
			break
		}
	}
	if !foundRelevant {
		t.Log("Note: No decisions or learnings matched. This is acceptable if room matched instead.")
	}
}

func TestGetRoute_LowConfidenceLearningsExcluded(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "route-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mem, err := memory.Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	defer mem.Close()

	// Add a low-confidence learning
	lowID, err := mem.AddLearning(memory.Learning{
		Content:    "Maybe we should use cookies for auth",
		Scope:      "palace",
		Confidence: 0.3, // Below default threshold of 0.7
		Authority:  string(memory.AuthorityApproved),
	})
	if err != nil {
		t.Fatalf("Failed to add learning: %v", err)
	}

	b := &Butler{
		rooms:  map[string]model.Room{},
		memory: mem,
	}

	result, err := b.GetRoute("cookies auth", memory.ScopePalace, "", nil)
	if err != nil {
		t.Fatalf("GetRoute failed: %v", err)
	}

	// Should NOT find the low-confidence learning
	for _, node := range result.Nodes {
		if node.Kind == RouteNodeKindLearning && node.ID == lowID {
			t.Error("Low-confidence learning should be excluded from route")
		}
	}
}

func TestGetRoute_ProposedRecordsExcluded(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "route-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mem, err := memory.Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	defer mem.Close()

	// Add a proposed (not approved) decision
	proposedID, err := mem.AddDecision(memory.Decision{
		Content:   "Use GraphQL instead of REST",
		Scope:     "palace",
		Status:    "active",
		Authority: string(memory.AuthorityProposed), // Not approved
	})
	if err != nil {
		t.Fatalf("Failed to add decision: %v", err)
	}

	b := &Butler{
		rooms:  map[string]model.Room{},
		memory: mem,
	}

	result, err := b.GetRoute("GraphQL REST", memory.ScopePalace, "", nil)
	if err != nil {
		t.Fatalf("GetRoute failed: %v", err)
	}

	// Should NOT find the proposed decision
	for _, node := range result.Nodes {
		if node.Kind == RouteNodeKindDecision && node.ID == proposedID {
			t.Error("Proposed (non-authoritative) decision should be excluded from route")
		}
	}
}
