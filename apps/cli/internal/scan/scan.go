package scan

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/fsutil"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/gitutil"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/validate"
)

// resolveAndValidateRoot converts a path to absolute and verifies it exists.
func resolveAndValidateRoot(root string) (string, error) {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return "", fmt.Errorf("directory does not exist: %s", rootPath)
	}
	return rootPath, nil
}

// RunIncremental performs an incremental scan, only processing changed files.
// Returns the number of changes applied and an error if any.
// If there are no changes, returns (0, nil).
// This function uses hash-based change detection by default.
// For git-based detection, use RunIncrementalGit.
func RunIncremental(root string) (index.IncrementalScanSummary, error) {
	rootPath, err := resolveAndValidateRoot(root)
	if err != nil {
		return index.IncrementalScanSummary{}, err
	}

	// Check if index exists - if not, need full scan
	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return index.IncrementalScanSummary{}, fmt.Errorf("no index found")
	}

	guardrails := config.LoadGuardrails(rootPath)

	db, err := index.Open(dbPath)
	if err != nil {
		return index.IncrementalScanSummary{}, err
	}
	defer db.Close()

	// Detect changes
	changes, err := index.DetectChanges(db, rootPath, guardrails)
	if err != nil {
		return index.IncrementalScanSummary{}, fmt.Errorf("detect changes: %w", err)
	}

	if len(changes) == 0 {
		// Count unchanged files
		var count int
		if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM files").Scan(&count); err != nil {
			return index.IncrementalScanSummary{}, fmt.Errorf("count files: %w", err)
		}
		return index.IncrementalScanSummary{
			FilesUnchanged: count,
		}, nil
	}

	// Count files before changes to calculate unchanged correctly
	var initialCount int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM files").Scan(&initialCount); err != nil {
		return index.IncrementalScanSummary{}, fmt.Errorf("count initial files: %w", err)
	}

	// Apply incremental changes
	summary, err := index.IncrementalScan(db, rootPath, changes)
	if err != nil {
		return summary, fmt.Errorf("incremental scan: %w", err)
	}

	// Calculate unchanged files:
	// Unchanged = (files that existed before) - (files that were modified) - (files that were deleted)
	// Note: Added files don't affect unchanged count since they weren't there before
	summary.FilesUnchanged = initialCount - summary.FilesModified - summary.FilesDeleted

	// Update the commit hash in the latest scan if in a git repo
	if gitutil.IsGitRepo(rootPath) {
		commitHash, _ := gitutil.GetHeadCommit(rootPath)
		if commitHash != "" {
			_, _ = db.ExecContext(context.Background(),
				"UPDATE scans SET commit_hash = ? WHERE id = (SELECT MAX(id) FROM scans)",
				commitHash)
		}
	}

	return summary, nil
}

// RunIncrementalGit performs an incremental scan using git diff to detect changes.
// This is faster than hash-based detection for large repositories.
// Falls back to RunIncremental if not in a git repo or if git diff fails.
func RunIncrementalGit(root string) (index.IncrementalScanSummary, error) {
	rootPath, err := resolveAndValidateRoot(root)
	if err != nil {
		return index.IncrementalScanSummary{}, err
	}

	// Check if this is a git repo
	if !gitutil.IsGitRepo(rootPath) {
		return index.IncrementalScanSummary{}, fmt.Errorf("not a git repository")
	}

	// Check if index exists - if not, need full scan
	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return index.IncrementalScanSummary{}, fmt.Errorf("no index found")
	}

	db, err := index.Open(dbPath)
	if err != nil {
		return index.IncrementalScanSummary{}, err
	}
	defer db.Close()

	// Get the last scan's commit hash
	lastScan, err := index.LatestScan(db)
	if err != nil {
		return index.IncrementalScanSummary{}, fmt.Errorf("get last scan: %w", err)
	}

	// If no previous scan or no commit hash, fall back to hash-based detection
	if lastScan.ID == 0 || lastScan.CommitHash == "" {
		return index.IncrementalScanSummary{}, fmt.Errorf("no previous git-based scan found")
	}

	// Get changed files from git
	guardrails := config.LoadGuardrails(rootPath)
	added, modified, deleted, err := gitutil.GetChangedFilesSinceCommit(rootPath, lastScan.CommitHash)
	if err != nil {
		return index.IncrementalScanSummary{}, fmt.Errorf("git diff: %w", err)
	}

	// Filter files based on guardrails
	added = filterFiles(added, rootPath, guardrails)
	modified = filterFiles(modified, rootPath, guardrails)
	deleted = filterFiles(deleted, rootPath, guardrails)

	// Convert to FileChange format
	var changes []index.FileChange
	for _, path := range added {
		changes = append(changes, index.FileChange{Path: path, Action: "added"})
	}
	for _, path := range modified {
		changes = append(changes, index.FileChange{Path: path, Action: "modified"})
	}
	for _, path := range deleted {
		changes = append(changes, index.FileChange{Path: path, Action: "deleted"})
	}

	if len(changes) == 0 {
		// Count unchanged files
		var count int
		if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM files").Scan(&count); err != nil {
			return index.IncrementalScanSummary{}, fmt.Errorf("count files: %w", err)
		}
		return index.IncrementalScanSummary{
			FilesUnchanged: count,
		}, nil
	}

	// Count files before changes
	var initialCount int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM files").Scan(&initialCount); err != nil {
		return index.IncrementalScanSummary{}, fmt.Errorf("count initial files: %w", err)
	}

	// Apply incremental changes
	summary, err := index.IncrementalScan(db, rootPath, changes)
	if err != nil {
		return summary, fmt.Errorf("incremental scan: %w", err)
	}

	summary.FilesUnchanged = initialCount - summary.FilesModified - summary.FilesDeleted

	// Update commit hash
	commitHash, _ := gitutil.GetHeadCommit(rootPath)
	if commitHash != "" {
		_, _ = db.ExecContext(context.Background(),
			"UPDATE scans SET commit_hash = ? WHERE id = (SELECT MAX(id) FROM scans)",
			commitHash)
	}

	return summary, nil
}

// filterFiles filters a list of file paths based on guardrails.
func filterFiles(files []string, rootPath string, guardrails config.Guardrails) []string {
	var result []string
	for _, file := range files {
		// Check if file matches guardrails (should be excluded)
		if !fsutil.MatchesGuardrail(file, guardrails) {
			result = append(result, file)
		}
	}
	return result
}

// Run performs a full scan of the workspace
func Run(root string) (index.ScanSummary, int, error) {
	rootPath, err := resolveAndValidateRoot(root)
	if err != nil {
		return index.ScanSummary{}, 0, err
	}

	if _, err := config.EnsureLayout(rootPath); err != nil {
		return index.ScanSummary{}, 0, err
	}

	guardrails := config.LoadGuardrails(rootPath)
	startedAt := time.Now().UTC()

	records, err := index.BuildFileRecords(rootPath, guardrails)
	if err != nil {
		return index.ScanSummary{}, 0, err
	}

	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	db, err := index.Open(dbPath)
	if err != nil {
		return index.ScanSummary{}, 0, err
	}
	defer db.Close()

	// Get current git commit hash if in a git repo
	var commitHash string
	if gitutil.IsGitRepo(rootPath) {
		commitHash, _ = gitutil.GetHeadCommit(rootPath)
	}

	summary, err := index.WriteScanWithOptions(db, rootPath, records, startedAt, index.WriteScanOptions{
		CommitHash: commitHash,
	})
	if err != nil {
		return index.ScanSummary{}, 0, err
	}

	scanArtifactPath := filepath.Join(rootPath, ".palace", "index", "scan.json")
	now := time.Now().UTC().Format(time.RFC3339)
	artifact := model.ScanSummary{
		SchemaVersion:     "1.0.0",
		Kind:              "palace/scan",
		ScanID:            uuid.NewString(),
		DBScanID:          summary.ID,
		StartedAt:         summary.StartedAt.UTC().Format(time.RFC3339),
		CompletedAt:       summary.CompletedAt.UTC().Format(time.RFC3339),
		FileCount:         summary.FileCount,
		ChunkCount:        summary.ChunkCount,
		SymbolCount:       summary.SymbolCount,
		RelationshipCount: summary.RelationshipCount,
		ScanHash:          summary.ScanHash,
		Provenance: model.Provenance{
			CreatedBy: "palace scan",
			CreatedAt: now,
		},
	}

	if err := model.WriteScanSummary(scanArtifactPath, artifact); err != nil {
		return index.ScanSummary{}, 0, err
	}
	if err := validate.JSON(scanArtifactPath, "scan"); err != nil {
		return index.ScanSummary{}, 0, err
	}

	return summary, len(records), nil
}
