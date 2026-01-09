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

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/analysis"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/scan"
)

func init() {
	Register(&Command{
		Name:        "scan",
		Description: "Build/refresh the code index",
		Run:         RunScan,
	})
}

// ScanOptions contains the configuration for the scan command.
type ScanOptions struct {
	Root string
	Full bool
	Deep bool // Enable deep analysis (LSP-based call tracking for Dart)
}

// RunScan executes the scan command with parsed arguments.
func RunScan(args []string) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	full := fs.Bool("full", false, "force full rescan (default: incremental)")
	deep := fs.Bool("deep", false, "enable deep analysis (LSP-based call tracking for Dart/Flutter)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	return ExecuteScan(ScanOptions{
		Root: *root,
		Full: *full,
		Deep: *deep,
	})
}

// ExecuteScan performs the scan with the given options.
// This is separated for easier testing.
func ExecuteScan(opts ScanOptions) error {
	var err error
	if opts.Full {
		err = executeFullScan(opts.Root)
	} else {
		err = executeIncrementalScan(opts.Root)
	}

	if err != nil {
		return err
	}

	// Auto-detect Dart/Flutter projects and run deep analysis
	// unless explicitly disabled with --deep=false
	rootPath, _ := filepath.Abs(opts.Root)
	if opts.Deep || isDartFlutterProject(rootPath) {
		return executeDeepAnalysis(opts.Root)
	}

	return nil
}

// isDartFlutterProject checks if the workspace is a Dart/Flutter project
func isDartFlutterProject(rootPath string) bool {
	// Check for pubspec.yaml at root
	if _, err := os.Stat(filepath.Join(rootPath, "pubspec.yaml")); err == nil {
		return true
	}

	// Check common monorepo directories for pubspec.yaml
	monorepoPatterns := []string{"apps", "packages", "modules", "lib"}
	for _, dir := range monorepoPatterns {
		dirPath := filepath.Join(rootPath, dir)
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				if _, err := os.Stat(filepath.Join(dirPath, entry.Name(), "pubspec.yaml")); err == nil {
					return true
				}
			}
		}
	}

	return false
}

func executeFullScan(root string) error {
	summary, fileCount, err := scan.Run(root)
	if err != nil {
		return err
	}
	fmt.Printf("full scan: indexed %d files, %d symbols, %d relationships\n", fileCount, summary.SymbolCount, summary.RelationshipCount)
	fmt.Printf("scan hash: %s\n", summary.ScanHash)
	fmt.Printf("scan artifact written to %s\n", filepath.Join(summary.Root, ".palace", "index", "scan.json"))
	return nil
}

func executeIncrementalScan(root string) error {
	summary, err := scan.RunIncremental(root)
	if err != nil {
		// If no index exists, fall back to full scan silently
		// This is normal for fresh projects - no need for a warning
		if strings.Contains(err.Error(), "no index found") {
			return executeFullScan(root)
		}
		// For other errors, show warning and try full scan
		fmt.Fprintf(os.Stderr, "incremental scan failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "falling back to full scan...\n")
		return executeFullScan(root)
	}

	totalChanges := summary.FilesAdded + summary.FilesModified + summary.FilesDeleted
	if totalChanges == 0 {
		fmt.Printf("no changes detected (%d files unchanged)\n", summary.FilesUnchanged)
	} else {
		fmt.Printf("incremental scan: +%d added, ~%d modified, -%d deleted (took %v)\n",
			summary.FilesAdded, summary.FilesModified, summary.FilesDeleted, summary.Duration.Round(time.Millisecond))
		fmt.Printf("%d files unchanged\n", summary.FilesUnchanged)
	}
	return nil
}

// executeDeepAnalysis runs LSP-based deep analysis for Dart/Flutter projects
func executeDeepAnalysis(root string) error {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	// Check if this is a Dart/Flutter project
	hasDart := false
	for _, marker := range []string{"pubspec.yaml", "melos.yaml"} {
		if _, err := os.Stat(filepath.Join(rootPath, marker)); err == nil {
			hasDart = true
			break
		}
	}
	if !hasDart {
		// Check for monorepo pattern
		for _, dir := range []string{"apps", "packages"} {
			dirPath := filepath.Join(rootPath, dir)
			entries, err := os.ReadDir(dirPath)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if entry.IsDir() {
					if _, err := os.Stat(filepath.Join(dirPath, entry.Name(), "pubspec.yaml")); err == nil {
						hasDart = true
						break
					}
				}
			}
			if hasDart {
				break
			}
		}
	}

	if !hasDart {
		fmt.Println("deep analysis: skipped (not a Dart/Flutter project)")
		return nil
	}

	fmt.Println("\nStarting deep analysis (Dart LSP)...")
	startTime := time.Now()

	// Initialize Dart analyzer
	analyzer, err := analysis.NewDartAnalyzer(rootPath)
	if err != nil {
		return fmt.Errorf("initialize Dart analyzer: %w", err)
	}
	defer analyzer.Close()

	// Get list of Dart files from the index
	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	db, err := index.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open index: %w", err)
	}
	defer db.Close()

	dartFiles, err := getDartFilesFromIndex(db, rootPath)
	if err != nil {
		return fmt.Errorf("get dart files: %w", err)
	}

	if len(dartFiles) == 0 {
		fmt.Println("deep analysis: no Dart files found in index")
		return nil
	}

	fmt.Printf("analyzing %d Dart files for call relationships...\n", len(dartFiles))

	// Extract calls using quick scan (public symbols only for speed)
	calls, err := analyzer.QuickCallScan(dartFiles, func(current, total int, file string) {
		if current%10 == 0 || current == total {
			relFile := file
			if strings.HasPrefix(file, rootPath) {
				relFile, _ = filepath.Rel(rootPath, file)
			}
			fmt.Printf("  [%d/%d] %s\n", current, total, relFile)
		}
	})
	if err != nil {
		return fmt.Errorf("extract calls: %w", err)
	}

	// Store calls in the database
	callCount, err := storeCallRelationships(db, rootPath, calls)
	if err != nil {
		return fmt.Errorf("store calls: %w", err)
	}

	duration := time.Since(startTime).Round(time.Millisecond)
	fmt.Printf("deep analysis complete: %d call relationships extracted (took %v)\n", callCount, duration)

	return nil
}

// getDartFilesFromIndex retrieves all Dart file paths from the index
func getDartFilesFromIndex(db *sql.DB, rootPath string) ([]string, error) {
	rows, err := db.QueryContext(context.Background(), "SELECT path FROM files WHERE language = 'dart'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			continue
		}
		files = append(files, filepath.Join(rootPath, path))
	}

	return files, rows.Err()
}

// storeCallRelationships stores extracted call relationships in the database
func storeCallRelationships(db *sql.DB, rootPath string, calls []analysis.CallInfo) (int, error) {
	if len(calls) == 0 {
		return 0, nil
	}

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// First, delete existing LSP-extracted calls to avoid duplicates
	_, err = tx.ExecContext(context.Background(), "DELETE FROM relationships WHERE kind = 'call' AND source_file LIKE '%.dart'")
	if err != nil {
		return 0, fmt.Errorf("clear old calls: %w", err)
	}

	stmt, err := tx.PrepareContext(context.Background(), `
		INSERT INTO relationships(source_file, source_symbol_id, target_file, target_symbol, kind, line, column)
		VALUES(?, NULL, ?, ?, 'call', ?, 0)
	`)
	if err != nil {
		return 0, fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	// Deduplicate calls
	seen := make(map[string]bool)
	count := 0

	for _, call := range calls {
		// Make paths relative
		callerPath := call.CallerFile
		calleePath := call.CalleeFile
		if strings.HasPrefix(callerPath, rootPath) {
			callerPath, _ = filepath.Rel(rootPath, callerPath)
		}
		if strings.HasPrefix(calleePath, rootPath) {
			calleePath, _ = filepath.Rel(rootPath, calleePath)
		}

		// Skip if caller or callee is outside the project
		if strings.HasPrefix(callerPath, "/") || strings.HasPrefix(calleePath, "/") {
			continue
		}

		// Create unique key for deduplication
		key := fmt.Sprintf("%s:%s:%s:%s:%d", callerPath, call.CallerSymbol, calleePath, call.CalleeSymbol, call.CallerLine)
		if seen[key] {
			continue
		}
		seen[key] = true

		_, err := stmt.ExecContext(context.Background(), callerPath, calleePath, call.CalleeSymbol, call.CallerLine)
		if err != nil {
			continue // Skip individual errors
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}

	return count, nil
}
