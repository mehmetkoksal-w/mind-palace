package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/cli/flags"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/cli/util"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/index"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/memory"
)

func init() {
	Register(&Command{
		Name:        "status",
		Aliases:     []string{"st"},
		Description: "Show workspace status, index stats, and active agents",
		Run:         RunStatus,
	})
}

// StatusOptions contains options for the status command.
type StatusOptions struct {
	Root     string
	FilePath string
	Full     bool // Show full detailed statistics (like old stats command)
	Sessions bool // Show detailed session information
}

// StatusScanInfo represents the scan.json structure for reading index stats.
type StatusScanInfo struct {
	FileCount         int       `json:"fileCount"`
	SymbolCount       int       `json:"symbolCount"`
	RelationshipCount int       `json:"relationshipCount"`
	CompletedAt       time.Time `json:"completedAt"`
}

// StatusIndexStats holds statistics about the indexed codebase.
type StatusIndexStats struct {
	FileCount           int
	SymbolCount         int
	SymbolsByKind       map[string]int
	RelationshipCount   int
	RelationshipsByKind map[string]int
	ChunkCount          int
	LastScan            time.Time
	ScanHash            string
}

// StatusKnowledgeStats holds statistics about stored knowledge.
type StatusKnowledgeStats struct {
	Ideas     int
	Decisions int
	Learnings int
	Sessions  int
	Active    int
}

// RunStatus executes the status command.
func RunStatus(args []string) error {
	// Check for subcommands first
	if len(args) > 0 {
		switch args[0] {
		case "full":
			return runStatusFull(args[1:])
		}
	}

	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	full := fs.Bool("full", false, "show detailed statistics (symbols by kind, relationships, etc.)")
	sessions := fs.Bool("sessions", false, "show detailed session information")
	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	filePath := ""
	if len(remaining) > 0 {
		filePath = remaining[0]
	}

	opts := StatusOptions{
		Root:     *root,
		FilePath: filePath,
		Full:     *full,
		Sessions: *sessions,
	}

	if opts.Full {
		return executeStatusFull(opts)
	}
	return executeStatusBrief(opts)
}

// runStatusFull runs detailed statistics mode.
func runStatusFull(args []string) error {
	fs := flag.NewFlagSet("status full", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	return executeStatusFull(StatusOptions{
		Root: *root,
		Full: true,
	})
}

// executeStatusBrief shows a quick operational briefing (like old brief command).
func executeStatusBrief(opts StatusOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	fmt.Printf("\n📋 Status")
	if opts.FilePath != "" {
		fmt.Printf(" for: %s", opts.FilePath)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("═", 60))

	// Show index statistics from scan.json
	scanPath := filepath.Join(rootPath, ".palace", "index", "scan.json")
	if scanData, err := os.ReadFile(scanPath); err == nil {
		var scanInfo StatusScanInfo
		if json.Unmarshal(scanData, &scanInfo) == nil && scanInfo.FileCount > 0 {
			fmt.Printf("\n📊 Index Statistics:\n")
			fmt.Printf("  Files:         %d\n", scanInfo.FileCount)
			fmt.Printf("  Symbols:       %d\n", scanInfo.SymbolCount)
			fmt.Printf("  Relationships: %d\n", scanInfo.RelationshipCount)
			if !scanInfo.CompletedAt.IsZero() {
				fmt.Printf("  Last Scan:     %s\n", scanInfo.CompletedAt.Format("2006-01-02 15:04:05"))
			}
		}
	} else {
		fmt.Printf("\n⚠️  No index found. Run 'palace scan' to index the codebase.\n")
	}

	// Show memory statistics summary
	totalSessions, _ := mem.CountSessions(false)
	activeSessions, _ := mem.CountSessions(true)
	learningCount, _ := mem.CountLearnings()
	filesTracked, _ := mem.CountFilesTracked()

	if totalSessions > 0 || learningCount > 0 || filesTracked > 0 {
		fmt.Printf("\n🧠 Memory Statistics:\n")
		fmt.Printf("  Sessions:      %d total, %d active\n", totalSessions, activeSessions)
		fmt.Printf("  Learnings:     %d\n", learningCount)
		fmt.Printf("  Files Tracked: %d\n", filesTracked)
	}

	// Show active agents
	agents, err := mem.GetActiveAgents(5 * time.Minute)
	if err == nil && len(agents) > 0 {
		fmt.Printf("\n🤖 Active Agents:\n")
		for i := range agents {
			a := &agents[i]
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
			for i := range sessions {
				s := &sessions[i]
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
			for i := range fileLearnings {
				l := &fileLearnings[i]
				fmt.Printf("  • [%.0f%%] %s\n", l.Confidence*100, util.TruncateLine(l.Content, 50))
			}
		}
	}

	// Show relevant learnings
	learnings, err := mem.GetRelevantLearnings(opts.FilePath, "", 5)
	if err == nil && len(learnings) > 0 {
		fmt.Printf("\n💡 Relevant Learnings:\n")
		for i := range learnings {
			l := &learnings[i]
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
		for i := range ideas {
			idea := &ideas[i]
			fmt.Printf("  • [%s] %s\n", idea.ID, util.TruncateLine(idea.Content, 45))
		}
	}

	// Show brain decisions
	decisions, err := mem.GetDecisions("active", "", "", "", 5)
	if err == nil && len(decisions) > 0 {
		fmt.Printf("\n📋 Active Decisions:\n")
		for i := range decisions {
			d := &decisions[i]
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
		for i := range hotspots {
			h := &hotspots[i]
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

// executeStatusFull shows detailed statistics (like old stats command).
func executeStatusFull(opts StatusOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Palace Statistics")
	fmt.Println(strings.Repeat("=", 50))

	// Index statistics
	indexStats, err := getStatusIndexStats(rootPath)
	if err != nil {
		fmt.Printf("\nIndex: not available (%v)\n", err)
	} else {
		fmt.Println()
		fmt.Println("Index Statistics")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("  Files indexed:      %d\n", indexStats.FileCount)
		fmt.Printf("  Chunks:             %d\n", indexStats.ChunkCount)
		fmt.Printf("  Symbols:            %d\n", indexStats.SymbolCount)

		// Print symbols by kind
		if len(indexStats.SymbolsByKind) > 0 {
			for kind, count := range indexStats.SymbolsByKind {
				fmt.Printf("    - %-14s %d\n", kind+":", count)
			}
		}

		fmt.Printf("  Relationships:      %d\n", indexStats.RelationshipCount)

		// Print relationships by kind
		if len(indexStats.RelationshipsByKind) > 0 {
			for kind, count := range indexStats.RelationshipsByKind {
				fmt.Printf("    - %-14s %d\n", kind+":", count)
			}
		}

		if !indexStats.LastScan.IsZero() {
			fmt.Printf("  Last scan:          %s\n", indexStats.LastScan.Format("2006-01-02 15:04:05"))
		}
		if indexStats.ScanHash != "" {
			// Show first 12 characters of hash
			hash := indexStats.ScanHash
			if len(hash) > 12 {
				hash = hash[:12] + "..."
			}
			fmt.Printf("  Scan hash:          %s\n", hash)
		}
	}

	// Knowledge statistics
	knowledgeStats, err := getStatusKnowledgeStats(rootPath)
	if err != nil {
		fmt.Printf("\nKnowledge: not available (%v)\n", err)
	} else {
		fmt.Println()
		fmt.Println("Knowledge Records")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("  Ideas:              %d\n", knowledgeStats.Ideas)
		fmt.Printf("  Decisions:          %d\n", knowledgeStats.Decisions)
		fmt.Printf("  Learnings:          %d\n", knowledgeStats.Learnings)

		fmt.Println()
		fmt.Println("Sessions")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("  Total:              %d\n", knowledgeStats.Sessions)
		fmt.Printf("  Active:             %d\n", knowledgeStats.Active)
	}

	fmt.Println()
	return nil
}

// getStatusIndexStats retrieves statistics from the index database.
func getStatusIndexStats(rootPath string) (*StatusIndexStats, error) {
	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no index found")
	}

	db, err := index.Open(dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	ctx := context.Background()
	stats := &StatusIndexStats{
		SymbolsByKind:       make(map[string]int),
		RelationshipsByKind: make(map[string]int),
	}

	// Count files
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM files").Scan(&stats.FileCount); err != nil {
		return nil, fmt.Errorf("count files: %w", err)
	}

	// Count chunks
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM chunks").Scan(&stats.ChunkCount); err != nil {
		return nil, fmt.Errorf("count chunks: %w", err)
	}

	// Count symbols
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM symbols").Scan(&stats.SymbolCount); err != nil {
		return nil, fmt.Errorf("count symbols: %w", err)
	}

	// Count symbols by kind
	rows, err := db.QueryContext(ctx, "SELECT kind, COUNT(*) FROM symbols GROUP BY kind ORDER BY COUNT(*) DESC")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var kind string
			var count int
			if err := rows.Scan(&kind, &count); err == nil {
				stats.SymbolsByKind[kind] = count
			}
		}
		_ = rows.Err() // Check for iteration errors
	}

	// Count relationships
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM relationships").Scan(&stats.RelationshipCount); err != nil {
		// Relationships table might not exist in older schemas
		stats.RelationshipCount = 0
	}

	// Count relationships by kind
	rows, err = db.QueryContext(ctx, "SELECT kind, COUNT(*) FROM relationships GROUP BY kind ORDER BY COUNT(*) DESC")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var kind string
			var count int
			if err := rows.Scan(&kind, &count); err == nil {
				stats.RelationshipsByKind[kind] = count
			}
		}
		_ = rows.Err() // Check for iteration errors
	}

	// Get last scan info
	var completedAt sql.NullString
	var scanHash sql.NullString
	err = db.QueryRowContext(ctx,
		"SELECT completed_at, scan_hash FROM scans ORDER BY id DESC LIMIT 1").Scan(&completedAt, &scanHash)
	if err == nil {
		if completedAt.Valid {
			stats.LastScan, _ = time.Parse(time.RFC3339, completedAt.String)
		}
		if scanHash.Valid {
			stats.ScanHash = scanHash.String
		}
	}

	return stats, nil
}

// getStatusKnowledgeStats retrieves statistics from the memory database.
func getStatusKnowledgeStats(rootPath string) (*StatusKnowledgeStats, error) {
	mem, err := memory.Open(rootPath)
	if err != nil {
		return nil, err
	}
	defer mem.Close()

	stats := &StatusKnowledgeStats{}

	// Count ideas
	ideas, err := mem.GetIdeas("", "", "", 0)
	if err == nil {
		stats.Ideas = len(ideas)
	}

	// Count decisions
	decisions, err := mem.GetDecisions("", "", "", "", 0)
	if err == nil {
		stats.Decisions = len(decisions)
	}

	// Count learnings
	learnings, err := mem.GetRelevantLearnings("", "", 0)
	if err == nil {
		stats.Learnings = len(learnings)
	}

	// Count sessions
	sessions, err := mem.ListSessions(false, 0)
	if err == nil {
		stats.Sessions = len(sessions)
	}

	// Count active sessions
	activeSessions, err := mem.ListSessions(true, 0)
	if err == nil {
		stats.Active = len(activeSessions)
	}

	return stats, nil
}
