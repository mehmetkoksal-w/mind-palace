package commands

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/util"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

func init() {
	Register(&Command{
		Name:        "brief",
		Description: "Get briefing on workspace or file (status, agents, intel, learnings)",
		Run:         RunBrief,
	})
}

// BriefOptions contains options for the brief command.
type BriefOptions struct {
	Root     string
	FilePath string
	Sessions bool
}

// RunBrief executes the brief command.
func RunBrief(args []string) error {
	fs := flag.NewFlagSet("brief", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	sessions := fs.Bool("sessions", false, "show detailed session information")
	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	filePath := ""
	if len(remaining) > 0 {
		filePath = remaining[0]
	}

	return ExecuteBrief(BriefOptions{
		Root:     *root,
		FilePath: filePath,
		Sessions: *sessions,
	})
}

// ExecuteBrief shows a briefing for the project or a specific file.
func ExecuteBrief(opts BriefOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	fmt.Printf("\n📋 Briefing")
	if opts.FilePath != "" {
		fmt.Printf(" for: %s", opts.FilePath)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("═", 60))

	// Show active agents
	agents, err := mem.GetActiveAgents(5 * time.Minute)
	if err == nil && len(agents) > 0 {
		fmt.Printf("\n🤖 Active Agents:\n")
		for _, a := range agents {
			currentFile := ""
			if a.CurrentFile != "" {
				currentFile = fmt.Sprintf(" working on %s", a.CurrentFile)
			}
			fmt.Printf("  • %s (%s)%s\n", a.AgentType, a.SessionID[:12], currentFile)
		}
	}

	// Show detailed sessions if requested
	if opts.Sessions {
		sessions, err := mem.ListSessions(false, 10)
		if err == nil && len(sessions) > 0 {
			fmt.Printf("\n📋 Recent Sessions:\n")
			for _, s := range sessions {
				stateIcon := "✅"
				switch s.State {
				case "active":
					stateIcon = "🔄"
				case "abandoned":
					stateIcon = "❌"
				}
				goal := ""
				if s.Goal != "" {
					goal = fmt.Sprintf(" - %s", util.TruncateLine(s.Goal, 30))
				}
				fmt.Printf("  %s %s [%s]%s\n", stateIcon, s.ID[:12], s.AgentType, goal)
			}
		}
	}

	// Show conflict warning if file specified
	if opts.FilePath != "" {
		conflict, err := mem.CheckConflict("", opts.FilePath)
		if err == nil && conflict != nil {
			fmt.Printf("\n⚠️  Conflict Warning:\n")
			fmt.Printf("  Another agent (%s) touched this file recently\n", conflict.OtherAgent)
			fmt.Printf("  Session: %s | Last touched: %s\n", conflict.OtherSession[:12], conflict.LastTouched.Format("15:04:05"))
		}

		// Show file intel
		intel, err := mem.GetFileIntel(opts.FilePath)
		if err == nil {
			fmt.Printf("\n📊 File Intelligence: %s\n", opts.FilePath)
			fmt.Printf("  Edit Count:    %d\n", intel.EditCount)
			fmt.Printf("  Failure Count: %d\n", intel.FailureCount)
			if intel.EditCount > 0 {
				failureRate := float64(intel.FailureCount) / float64(intel.EditCount) * 100
				fmt.Printf("  Failure Rate:  %.1f%%", failureRate)
				if failureRate > 20 {
					fmt.Printf(" ⚠️")
				}
				fmt.Println()
			}
			if !intel.LastEdited.IsZero() {
				fmt.Printf("  Last Edited:   %s\n", intel.LastEdited.Format(time.RFC3339))
			}
			if intel.LastEditor != "" {
				fmt.Printf("  Last Editor:   %s\n", intel.LastEditor)
			}
		}

		// Show file-specific learnings
		fileLearnings, err := mem.GetFileLearnings(opts.FilePath)
		if err == nil && len(fileLearnings) > 0 {
			fmt.Printf("\n📝 File-Specific Learnings:\n")
			for _, l := range fileLearnings {
				fmt.Printf("  • [%.0f%%] %s\n", l.Confidence*100, util.TruncateLine(l.Content, 50))
			}
		}
	}

	// Show relevant learnings
	learnings, err := mem.GetRelevantLearnings(opts.FilePath, "", 5)
	if err == nil && len(learnings) > 0 {
		fmt.Printf("\n💡 Relevant Learnings:\n")
		for _, l := range learnings {
			scopeInfo := ""
			if l.Scope != "palace" {
				scopeInfo = fmt.Sprintf(" [%s]", l.Scope)
			}
			fmt.Printf("  • [%.0f%%]%s %s\n", l.Confidence*100, scopeInfo, util.TruncateLine(l.Content, 45))
		}
	}

	// Show brain ideas
	ideas, err := mem.GetIdeas("active", "", "", 5)
	if err == nil && len(ideas) > 0 {
		fmt.Printf("\n💭 Active Ideas:\n")
		for _, idea := range ideas {
			fmt.Printf("  • [%s] %s\n", idea.ID, util.TruncateLine(idea.Content, 45))
		}
	}

	// Show brain decisions
	decisions, err := mem.GetDecisions("active", "", "", "", 5)
	if err == nil && len(decisions) > 0 {
		fmt.Printf("\n📋 Active Decisions:\n")
		for _, d := range decisions {
			outcomeIcon := ""
			switch d.Outcome {
			case "successful":
				outcomeIcon = " ✅"
			case "failed":
				outcomeIcon = " ❌"
			case "mixed":
				outcomeIcon = " ⚖️"
			}
			fmt.Printf("  • [%s]%s %s\n", d.ID, outcomeIcon, util.TruncateLine(d.Content, 40))
		}
	}

	// Show hotspots
	hotspots, err := mem.GetFileHotspots(5)
	if err == nil && len(hotspots) > 0 {
		fmt.Printf("\n🔥 Hotspots (most edited files):\n")
		for _, h := range hotspots {
			warning := ""
			if h.FailureCount > 0 {
				warning = fmt.Sprintf(" (⚠️ %d failures)", h.FailureCount)
			}
			fmt.Printf("  • %s (%d edits)%s\n", h.Path, h.EditCount, warning)
		}
	}

	fmt.Println()
	return nil
}
