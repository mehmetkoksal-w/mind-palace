package butler

import (
	"encoding/json"
	"testing"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
)

// TestGetRouteToRecallE2E validates end-to-end flow:
// 1. get_route returns nodes with fetch_ref
// 2. fetch_ref format matches tool schema
// 3. Calling recall tools with --id returns the record
func TestGetRouteToRecallE2E(t *testing.T) {
	// Setup: Create butler with memory and rooms
	b, cleanup := setupButlerWithMemory(t)
	defer cleanup()

	// Add some approved records
	decisionID, _ := b.memory.AddDecision(memory.Decision{
		Content:   "Use JWT for authentication",
		Authority: string(memory.AuthorityApproved),
		Scope:     "palace",
	})

	learningID, _ := b.memory.AddLearning(memory.Learning{
		Content:    "Always validate JWTs on the server side",
		Authority:  string(memory.AuthorityApproved),
		Scope:      "palace",
		Confidence: 0.9,
	})

	// 1. Get route for "auth" intent
	route, err := b.GetRoute("understand auth", memory.ScopePalace, "", nil)
	if err != nil {
		t.Fatalf("GetRoute failed: %v", err)
	}

	if len(route.Nodes) == 0 {
		t.Fatal("Expected route to contain nodes")
	}

	// 2. Verify fetch_ref format for each node type
	var decisionNode, learningNode *RouteNode
	for i := range route.Nodes {
		node := &route.Nodes[i]
		switch node.Kind {
		case RouteNodeKindDecision:
			if node.ID == decisionID {
				decisionNode = node
			}
		case RouteNodeKindLearning:
			if node.ID == learningID {
				learningNode = node
			}
		}
	}

	// 3. Test decision fetch_ref
	if decisionNode != nil {
		expectedRef := "recall_decisions --id " + decisionID
		if decisionNode.FetchRef != expectedRef {
			t.Errorf("Decision fetch_ref = %q, expected %q", decisionNode.FetchRef, expectedRef)
		}

		// 4. Simulate calling recall_decisions with --id
		decision, err := b.memory.GetDecision(decisionID)
		if err != nil {
			t.Fatalf("GetDecision(%s) failed: %v", decisionID, err)
		}
		if decision.Content != "Use JWT for authentication" {
			t.Errorf("Decision content mismatch: got %q", decision.Content)
		}
	} else {
		t.Error("Route should include decision node")
	}

	// 5. Test learning fetch_ref
	if learningNode != nil {
		expectedRef := "recall --id " + learningID
		if learningNode.FetchRef != expectedRef {
			t.Errorf("Learning fetch_ref = %q, expected %q", learningNode.FetchRef, expectedRef)
		}

		// 6. Simulate calling recall with --id
		learning, err := b.memory.GetLearning(learningID)
		if err != nil {
			t.Fatalf("GetLearning(%s) failed: %v", learningID, err)
		}
		if learning.Content != "Always validate JWTs on the server side" {
			t.Errorf("Learning content mismatch: got %q", learning.Content)
		}
	} else {
		t.Error("Route should include learning node")
	}
}

// TestMCPToolRecallByID validates MCP tool handlers accept id parameter
func TestMCPToolRecallByID(t *testing.T) {
	// Setup MCP server
	b, cleanup := setupButlerWithMemory(t)
	defer cleanup()

	server := NewMCPServerWithMode(b, MCPModeAgent)

	// Add test records
	decisionID, _ := b.memory.AddDecision(memory.Decision{
		Content:   "Test decision",
		Authority: string(memory.AuthorityApproved),
		Scope:     "palace",
	})

	learningID, _ := b.memory.AddLearning(memory.Learning{
		Content:    "Test learning",
		Authority:  string(memory.AuthorityApproved),
		Scope:      "palace",
		Confidence: 0.8,
	})

	// Test recall_decisions --id
	t.Run("recall_decisions_by_id", func(t *testing.T) {
		resp := server.toolRecallDecisions(1, map[string]interface{}{
			"id": decisionID,
		})

		if resp.Error != nil {
			t.Fatalf("toolRecallDecisions failed: %v", resp.Error.Message)
		}

		result := resp.Result.(mcpToolResult)
		if len(result.Content) == 0 {
			t.Fatal("Expected content in response")
		}

		// Verify response contains decision ID
		text := result.Content[0].Text
		if !contains(text, decisionID) {
			t.Errorf("Response should contain decision ID %s: %s", decisionID, text)
		}
		if !contains(text, "Test decision") {
			t.Errorf("Response should contain decision content: %s", text)
		}
	})

	// Test recall --id
	t.Run("recall_by_id", func(t *testing.T) {
		resp := server.toolRecall(2, map[string]interface{}{
			"id": learningID,
		})

		if resp.Error != nil {
			t.Fatalf("toolRecall failed: %v", resp.Error.Message)
		}

		result := resp.Result.(mcpToolResult)
		if len(result.Content) == 0 {
			t.Fatal("Expected content in response")
		}

		// Verify response contains learning ID and content
		text := result.Content[0].Text
		if !contains(text, learningID) {
			t.Errorf("Response should contain learning ID %s: %s", learningID, text)
		}
		if !contains(text, "Test learning") {
			t.Errorf("Response should contain learning content: %s", text)
		}
	})
}

// TestGetRouteWithFetchRefE2E validates complete MCP workflow
func TestGetRouteWithFetchRefE2E(t *testing.T) {
	// Setup MCP server
	b, cleanup := setupButlerWithMemory(t)
	defer cleanup()

	server := NewMCPServerWithMode(b, MCPModeAgent)

	// Add approved decision
	decisionID, _ := b.memory.AddDecision(memory.Decision{
		Content:   "Use PostgreSQL for the database",
		Authority: string(memory.AuthorityApproved),
		Scope:     "palace",
	})

	// Step 1: Call get_route
	routeResp := server.toolGetRoute(1, map[string]interface{}{
		"intent": "understand database",
		"scope":  "palace",
	})

	if routeResp.Error != nil {
		t.Fatalf("get_route failed: %v", routeResp.Error.Message)
	}

	// Parse route result
	routeResult := routeResp.Result.(mcpToolResult)
	var route RouteResult
	if err := json.Unmarshal([]byte(routeResult.Content[0].Text), &route); err != nil {
		t.Fatalf("Failed to parse route JSON: %v", err)
	}

	// Find decision node
	var decisionNode *RouteNode
	for i := range route.Nodes {
		if route.Nodes[i].Kind == RouteNodeKindDecision && route.Nodes[i].ID == decisionID {
			decisionNode = &route.Nodes[i]
			break
		}
	}

	if decisionNode == nil {
		t.Fatal("Expected route to include decision node for 'database' intent")
	}

	// Verify fetch_ref format
	expectedFetchRef := "recall_decisions --id " + decisionID
	if decisionNode.FetchRef != expectedFetchRef {
		t.Errorf("fetch_ref = %q, expected %q", decisionNode.FetchRef, expectedFetchRef)
	}

	// Step 2: Follow fetch_ref (simulate agent calling recall_decisions --id)
	recallResp := server.toolRecallDecisions(2, map[string]interface{}{
		"id": decisionID,
	})

	if recallResp.Error != nil {
		t.Fatalf("recall_decisions --id %s failed: %v", decisionID, recallResp.Error.Message)
	}

	recallResult := recallResp.Result.(mcpToolResult)
	recallText := recallResult.Content[0].Text

	// Verify response contains full decision
	if !contains(recallText, decisionID) {
		t.Errorf("recall response should include decision ID")
	}
	if !contains(recallText, "PostgreSQL") {
		t.Errorf("recall response should include decision content")
	}
}

// Helper function for test setup
func setupButlerWithMemory(t *testing.T) (*Butler, func()) {
	t.Helper()

	// Create temporary directory for test memory
	tmpDir := t.TempDir()

	// Open memory with temporary directory
	mem, err := memory.Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}

	// Create butler with minimal configuration
	b := &Butler{
		memory:      mem,
		rooms:       make(map[string]model.Room),
		entryPoints: make(map[string]string),
		root:        tmpDir,
	}

	return b, func() {
		if mem != nil {
			mem.Close()
		}
	}
}

// contains helper is already defined in butler_test.go
