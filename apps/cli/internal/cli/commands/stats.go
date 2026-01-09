package commands

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

func init() {
	Register(&Command{
		Name:        "stats",
		Description: "Show index and knowledge statistics",
		Run:         RunStats,
	})
}

// StatsOptions contains the configuration for the stats command.
type StatsOptions struct {
	Root string
}

// IndexStats holds statistics about the indexed codebase.
type IndexStats struct {
	FileCount         int
	SymbolCount       int
	SymbolsByKind     map[string]int
	RelationshipCount int
	RelationshipsByKind map[string]int
	ChunkCount        int
	LastScan          time.Time
	ScanHash          string
}

// KnowledgeStats holds statistics about stored knowledge.
type KnowledgeStats struct {
	Ideas     int
	Decisions int
	Learnings int
	Sessions  int
	Active    int
}

// RunStats executes the stats command with parsed arguments.
func RunStats(args []string) error {
	fs := flag.NewFlagSet("stats", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	if err := fs.Parse(args); err != nil {
		return err
	}

	return ExecuteStats(StatsOptions{
		Root: *root,
	})
}

// ExecuteStats shows statistics for the palace index and knowledge.
func ExecuteStats(opts StatsOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Palace Statistics")
	fmt.Println(strings.Repeat("=", 50))

	// Index statistics
	indexStats, err := getIndexStats(rootPath)
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
	knowledgeStats, err := getKnowledgeStats(rootPath)
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

// getIndexStats retrieves statistics from the index database.
func getIndexStats(rootPath string) (*IndexStats, error) {
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
	stats := &IndexStats{
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

// getKnowledgeStats retrieves statistics from the memory database.
func getKnowledgeStats(rootPath string) (*KnowledgeStats, error) {
	mem, err := memory.Open(rootPath)
	if err != nil {
		return nil, err
	}
	defer mem.Close()

	stats := &KnowledgeStats{}

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
