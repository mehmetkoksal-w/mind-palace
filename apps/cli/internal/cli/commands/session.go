package commands

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/flags"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/util"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

func init() {
	Register(&Command{
		Name:        "session",
		Description: "Manage AI agent sessions",
		Run:         RunSession,
	})
}

// RunSession dispatches to the appropriate session subcommand.
func RunSession(args []string) error {
	if len(args) == 0 {
		return errors.New(`usage: palace session <command>

Commands:
  start    Start a new session
  end      End a session
  list     List sessions
  show     Show session details

Examples:
  palace session start --agent claude --goal "implement auth"
  palace session end SESSION_ID
  palace session list --active
  palace session show SESSION_ID`)
	}

	switch args[0] {
	case "start":
		return RunSessionStart(args[1:])
	case "end":
		return RunSessionEnd(args[1:])
	case "list":
		return RunSessionList(args[1:])
	case "show":
		return RunSessionShow(args[1:])
	default:
		return fmt.Errorf("unknown session command: %s\nRun 'palace help session' for usage", args[0])
	}
}

// SessionStartOptions contains the configuration for session start.
type SessionStartOptions struct {
	Root    string
	Agent   string
	AgentID string
	Goal    string
}

// RunSessionStart executes the session start subcommand.
func RunSessionStart(args []string) error {
	fs := flag.NewFlagSet("session start", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	agent := fs.String("agent", "cli", "agent type (claude, cursor, aider, etc.)")
	agentID := fs.String("agent-id", "", "unique agent instance ID")
	goal := fs.String("goal", "", "session goal")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Use remaining args as goal if not set via flag
	goalVal := *goal
	if goalVal == "" && len(fs.Args()) > 0 {
		goalVal = strings.Join(fs.Args(), " ")
	}

	return ExecuteSessionStart(SessionStartOptions{
		Root:    *root,
		Agent:   *agent,
		AgentID: *agentID,
		Goal:    goalVal,
	})
}

// ExecuteSessionStart starts a new session.
func ExecuteSessionStart(opts SessionStartOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	session, err := mem.StartSession(opts.Agent, opts.AgentID, opts.Goal)
	if err != nil {
		return fmt.Errorf("start session: %w", err)
	}

	fmt.Printf("Session started: %s\n", session.ID)
	fmt.Printf("  Agent: %s\n", session.AgentType)
	if session.Goal != "" {
		fmt.Printf("  Goal: %s\n", session.Goal)
	}
	return nil
}

// SessionEndOptions contains the configuration for session end.
type SessionEndOptions struct {
	Root      string
	SessionID string
	State     string
	Summary   string
}

// RunSessionEnd executes the session end subcommand.
func RunSessionEnd(args []string) error {
	fs := flag.NewFlagSet("session end", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	state := fs.String("state", "completed", "final state (completed, abandoned)")
	summary := fs.String("summary", "", "session summary")
	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("usage: palace session end SESSION_ID [--state completed|abandoned] [--summary \"...\"]")
	}

	return ExecuteSessionEnd(SessionEndOptions{
		Root:      *root,
		SessionID: remaining[0],
		State:     *state,
		Summary:   *summary,
	})
}

// ExecuteSessionEnd ends a session.
func ExecuteSessionEnd(opts SessionEndOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	if err := mem.EndSession(opts.SessionID, opts.State, opts.Summary); err != nil {
		return fmt.Errorf("end session: %w", err)
	}

	fmt.Printf("Session %s ended (state: %s)\n", opts.SessionID, opts.State)
	return nil
}

// SessionListOptions contains the configuration for session list.
type SessionListOptions struct {
	Root   string
	Active bool
	Limit  int
}

// RunSessionList executes the session list subcommand.
func RunSessionList(args []string) error {
	fs := flag.NewFlagSet("session list", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	active := fs.Bool("active", false, "show only active sessions")
	limit := fs.Int("limit", 10, "maximum sessions to show")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate inputs
	if err := flags.ValidateLimit(*limit); err != nil {
		return err
	}

	return ExecuteSessionList(SessionListOptions{
		Root:   *root,
		Active: *active,
		Limit:  *limit,
	})
}

// ExecuteSessionList lists sessions.
func ExecuteSessionList(opts SessionListOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	sessions, err := mem.ListSessions(opts.Active, opts.Limit)
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	fmt.Printf("\n📋 Sessions\n")
	fmt.Println(strings.Repeat("─", 60))

	for i := range sessions {
		s := &sessions[i]
		stateIcon := "✅"
		switch s.State {
		case "active":
			stateIcon = "🔄"
		case "abandoned":
			stateIcon = "❌"
		}
		fmt.Printf("%s %s [%s] %s\n", stateIcon, s.ID, s.AgentType, s.State)
		if s.Goal != "" {
			fmt.Printf("   Goal: %s\n", util.TruncateLine(s.Goal, 50))
		}
		fmt.Printf("   Started: %s\n", s.StartedAt.Format(time.RFC3339))
		fmt.Println()
	}

	return nil
}

// SessionShowOptions contains the configuration for session show.
type SessionShowOptions struct {
	Root      string
	SessionID string
}

// RunSessionShow executes the session show subcommand.
func RunSessionShow(args []string) error {
	fs := flag.NewFlagSet("session show", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("usage: palace session show SESSION_ID")
	}

	return ExecuteSessionShow(SessionShowOptions{
		Root:      *root,
		SessionID: remaining[0],
	})
}

// ExecuteSessionShow shows session details.
func ExecuteSessionShow(opts SessionShowOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	session, err := mem.GetSession(opts.SessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	fmt.Printf("\n📋 Session: %s\n", session.ID)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Agent:      %s\n", session.AgentType)
	if session.AgentID != "" {
		fmt.Printf("Agent ID:   %s\n", session.AgentID)
	}
	fmt.Printf("State:      %s\n", session.State)
	fmt.Printf("Started:    %s\n", session.StartedAt.Format(time.RFC3339))
	fmt.Printf("Last Active: %s\n", session.LastActivity.Format(time.RFC3339))
	if session.Goal != "" {
		fmt.Printf("Goal:       %s\n", session.Goal)
	}
	if session.Summary != "" {
		fmt.Printf("Summary:    %s\n", session.Summary)
	}

	// Show recent activities
	activities, err := mem.GetActivities(opts.SessionID, "", 10)
	if err == nil && len(activities) > 0 {
		fmt.Printf("\n📝 Recent Activities:\n")
		for _, a := range activities {
			outcomeIcon := "•"
			switch a.Outcome {
			case "success":
				outcomeIcon = "✓"
			case "failure":
				outcomeIcon = "✗"
			}
			target := ""
			if a.Target != "" {
				target = fmt.Sprintf(" → %s", a.Target)
			}
			fmt.Printf("  %s [%s] %s%s\n", outcomeIcon, a.Kind, a.Timestamp.Format("15:04:05"), target)
		}
	}

	return nil
}
