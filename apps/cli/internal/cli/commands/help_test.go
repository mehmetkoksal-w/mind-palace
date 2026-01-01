package commands_test

import (
	"strings"
	"testing"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/cli/commands"
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

func TestShowHelpTopicInit(t *testing.T) {
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

func TestShowHelpTopicExplore(t *testing.T) {
	err := commands.ShowHelpTopic("explore")
	if err != nil {
		t.Errorf("ShowHelpTopic(explore) failed: %v", err)
	}
}

func TestShowHelpTopicStore(t *testing.T) {
	err := commands.ShowHelpTopic("store")
	if err != nil {
		t.Errorf("ShowHelpTopic(store) failed: %v", err)
	}
}

func TestShowHelpTopicRecall(t *testing.T) {
	err := commands.ShowHelpTopic("recall")
	if err != nil {
		t.Errorf("ShowHelpTopic(recall) failed: %v", err)
	}
}

func TestShowHelpTopicBrief(t *testing.T) {
	err := commands.ShowHelpTopic("brief")
	if err != nil {
		t.Errorf("ShowHelpTopic(brief) failed: %v", err)
	}
}

func TestShowHelpTopicServe(t *testing.T) {
	err := commands.ShowHelpTopic("serve")
	if err != nil {
		t.Errorf("ShowHelpTopic(serve) failed: %v", err)
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

	// Check for expected sections (Canonical commands)
	expectedSections := []string{"SCAN", "CHECK", "EXPLORE", "BRIEF", "CLEAN"}
	for _, section := range expectedSections {
		if !strings.Contains(result, section) {
			t.Errorf("ExplainAll should contain section %q", section)
		}
	}
}
