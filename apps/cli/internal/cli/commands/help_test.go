package commands_test

import (
	"strings"
	"testing"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/commands"
)

func TestRunHelpNoArgs(t *testing.T) {
	err := commands.RunHelp([]string{})
	if err != nil {
		t.Errorf("RunHelp with no args should show usage: %v", err)
	}
}

func TestShowUsage(t *testing.T) {
	err := commands.ShowUsage()
	if err != nil {
		t.Errorf("ShowUsage failed: %v", err)
	}
}

func TestShowHelpTopicBuild(t *testing.T) {
	err := commands.ShowHelpTopic("build")
	if err != nil {
		t.Errorf("ShowHelpTopic(build) failed: %v", err)
	}
}

func TestShowHelpTopicEnter(t *testing.T) {
	// "enter" is an alias for "build"
	err := commands.ShowHelpTopic("enter")
	if err != nil {
		t.Errorf("ShowHelpTopic(enter) failed: %v", err)
	}
}

func TestShowHelpTopicInit(t *testing.T) {
	// "init" is an alias for "build"
	err := commands.ShowHelpTopic("init")
	if err != nil {
		t.Errorf("ShowHelpTopic(init) failed: %v", err)
	}
}

func TestShowHelpTopicScan(t *testing.T) {
	err := commands.ShowHelpTopic("scan")
	if err != nil {
		t.Errorf("ShowHelpTopic(scan) failed: %v", err)
	}
}

func TestShowHelpTopicCheck(t *testing.T) {
	err := commands.ShowHelpTopic("check")
	if err != nil {
		t.Errorf("ShowHelpTopic(check) failed: %v", err)
	}
}

func TestShowHelpTopicVerify(t *testing.T) {
	// "verify" is an alias for "check"
	err := commands.ShowHelpTopic("verify")
	if err != nil {
		t.Errorf("ShowHelpTopic(verify) failed: %v", err)
	}
}

func TestShowHelpTopicExplore(t *testing.T) {
	err := commands.ShowHelpTopic("explore")
	if err != nil {
		t.Errorf("ShowHelpTopic(explore) failed: %v", err)
	}
}

func TestShowHelpTopicQuery(t *testing.T) {
	// "query" maps to explore
	err := commands.ShowHelpTopic("query")
	if err != nil {
		t.Errorf("ShowHelpTopic(query) failed: %v", err)
	}
}

func TestShowHelpTopicContext(t *testing.T) {
	// "context" maps to explore
	err := commands.ShowHelpTopic("context")
	if err != nil {
		t.Errorf("ShowHelpTopic(context) failed: %v", err)
	}
}

func TestShowHelpTopicGraph(t *testing.T) {
	// "graph" maps to explore
	err := commands.ShowHelpTopic("graph")
	if err != nil {
		t.Errorf("ShowHelpTopic(graph) failed: %v", err)
	}
}

func TestShowHelpTopicStore(t *testing.T) {
	err := commands.ShowHelpTopic("store")
	if err != nil {
		t.Errorf("ShowHelpTopic(store) failed: %v", err)
	}
}

func TestShowHelpTopicRemember(t *testing.T) {
	// "remember" maps to store
	err := commands.ShowHelpTopic("remember")
	if err != nil {
		t.Errorf("ShowHelpTopic(remember) failed: %v", err)
	}
}

func TestShowHelpTopicLearn(t *testing.T) {
	// "learn" maps to store
	err := commands.ShowHelpTopic("learn")
	if err != nil {
		t.Errorf("ShowHelpTopic(learn) failed: %v", err)
	}
}

func TestShowHelpTopicRecall(t *testing.T) {
	err := commands.ShowHelpTopic("recall")
	if err != nil {
		t.Errorf("ShowHelpTopic(recall) failed: %v", err)
	}
}

func TestShowHelpTopicReview(t *testing.T) {
	// "review" maps to recall
	err := commands.ShowHelpTopic("review")
	if err != nil {
		t.Errorf("ShowHelpTopic(review) failed: %v", err)
	}
}

func TestShowHelpTopicOutcome(t *testing.T) {
	// "outcome" maps to recall
	err := commands.ShowHelpTopic("outcome")
	if err != nil {
		t.Errorf("ShowHelpTopic(outcome) failed: %v", err)
	}
}

func TestShowHelpTopicLink(t *testing.T) {
	// "link" maps to recall
	err := commands.ShowHelpTopic("link")
	if err != nil {
		t.Errorf("ShowHelpTopic(link) failed: %v", err)
	}
}

func TestShowHelpTopicBrief(t *testing.T) {
	err := commands.ShowHelpTopic("brief")
	if err != nil {
		t.Errorf("ShowHelpTopic(brief) failed: %v", err)
	}
}

func TestShowHelpTopicIntel(t *testing.T) {
	// "intel" maps to brief
	err := commands.ShowHelpTopic("intel")
	if err != nil {
		t.Errorf("ShowHelpTopic(intel) failed: %v", err)
	}
}

func TestShowHelpTopicServe(t *testing.T) {
	err := commands.ShowHelpTopic("serve")
	if err != nil {
		t.Errorf("ShowHelpTopic(serve) failed: %v", err)
	}
}

func TestShowHelpTopicCI(t *testing.T) {
	err := commands.ShowHelpTopic("ci")
	if err != nil {
		t.Errorf("ShowHelpTopic(ci) failed: %v", err)
	}
}

func TestShowHelpTopicSession(t *testing.T) {
	err := commands.ShowHelpTopic("session")
	if err != nil {
		t.Errorf("ShowHelpTopic(session) failed: %v", err)
	}
}

func TestShowHelpTopicCorridor(t *testing.T) {
	err := commands.ShowHelpTopic("corridor")
	if err != nil {
		t.Errorf("ShowHelpTopic(corridor) failed: %v", err)
	}
}

func TestShowHelpTopicDashboard(t *testing.T) {
	err := commands.ShowHelpTopic("dashboard")
	if err != nil {
		t.Errorf("ShowHelpTopic(dashboard) failed: %v", err)
	}
}

func TestShowHelpTopicClean(t *testing.T) {
	err := commands.ShowHelpTopic("clean")
	if err != nil {
		t.Errorf("ShowHelpTopic(clean) failed: %v", err)
	}
}

func TestShowHelpTopicMaintenance(t *testing.T) {
	// "maintenance" maps to clean
	err := commands.ShowHelpTopic("maintenance")
	if err != nil {
		t.Errorf("ShowHelpTopic(maintenance) failed: %v", err)
	}
}

func TestShowHelpTopicArtifacts(t *testing.T) {
	err := commands.ShowHelpTopic("artifacts")
	if err != nil {
		t.Errorf("ShowHelpTopic(artifacts) failed: %v", err)
	}
}

func TestShowHelpTopicAll(t *testing.T) {
	err := commands.ShowHelpTopic("all")
	if err != nil {
		t.Errorf("ShowHelpTopic(all) failed: %v", err)
	}
}

func TestShowHelpTopicUnknown(t *testing.T) {
	err := commands.ShowHelpTopic("unknown")
	if err == nil {
		t.Error("ShowHelpTopic should return error for unknown topic")
	}
}

func TestExplainAll(t *testing.T) {
	result := commands.ExplainAll()
	if result == "" {
		t.Error("ExplainAll should return non-empty string")
	}

	// Check for expected sections
	expectedSections := []string{"SCAN", "CHECK", "QUERY", "CONTEXT", "GRAPH", "CI COMMANDS", "ARTIFACTS"}
	for _, section := range expectedSections {
		if !strings.Contains(result, section) {
			t.Errorf("ExplainAll should contain section %q", section)
		}
	}
}
