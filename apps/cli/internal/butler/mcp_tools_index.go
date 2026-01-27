package butler

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/scan"
)

// ============================================================
// INDEX TOOL - Manage the code index
// ============================================================

// dispatchIndex handles the index tool with action parameter.
func (s *MCPServer) dispatchIndex(id any, args map[string]interface{}, action string) jsonRPCResponse {
	if action == "" {
		action = "status" // default action
	}

	switch action {
	case "status":
		return s.toolIndexStatus(id)
	case "scan":
		return s.toolIndexScan(id, args)
	case "rescan":
		return s.toolIndexRescan(id, args)
	case "stats":
		return s.toolIndexStats(id)
	default:
		return consolidatedToolError(id, "index", "action", action)
	}
}

// toolIndexStatus checks if the index is fresh.
func (s *MCPServer) toolIndexStatus(id any) jsonRPCResponse {
	ctx := context.Background()

	// Get scan metadata
	var lastScanStr sql.NullString
	var scanHash sql.NullString
	var fileCount int
	var symbolCount int

	// Query metadata
	err := s.butler.db.QueryRowContext(ctx,
		`SELECT value FROM scan_metadata WHERE key = 'completed_at'`).Scan(&lastScanStr)
	if err != nil {
		lastScanStr.Valid = false
	}

	err = s.butler.db.QueryRowContext(ctx,
		`SELECT value FROM scan_metadata WHERE key = 'scan_hash'`).Scan(&scanHash)
	if err != nil {
		scanHash.Valid = false
	}

	// Count files and symbols
	s.butler.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT path) FROM chunks`).Scan(&fileCount)
	s.butler.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM symbols`).Scan(&symbolCount)

	var output strings.Builder
	output.WriteString("# Index Status\n\n")

	// Determine freshness
	fresh := false
	var lastScan time.Time
	if lastScanStr.Valid {
		lastScan, _ = time.Parse(time.RFC3339, lastScanStr.String)
		age := time.Since(lastScan)
		fresh = age < 24*time.Hour

		if fresh {
			output.WriteString("✅ **Index is fresh**\n\n")
		} else {
			output.WriteString("⚠️ **Index may be stale**\n\n")
			fmt.Fprintf(&output, "Last scan was %s ago. Consider running `action=rescan`.\n\n", formatIndexDuration(age))
		}

		fmt.Fprintf(&output, "| Metric | Value |\n")
		fmt.Fprintf(&output, "|--------|-------|\n")
		fmt.Fprintf(&output, "| Last Scan | %s |\n", lastScan.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(&output, "| Files Indexed | %d |\n", fileCount)
		fmt.Fprintf(&output, "| Symbols | %d |\n", symbolCount)
		if scanHash.Valid && len(scanHash.String) > 12 {
			fmt.Fprintf(&output, "| Scan Hash | %s... |\n", scanHash.String[:12])
		}
	} else {
		output.WriteString("❌ **No index found**\n\n")
		output.WriteString("Run `action=scan` or `action=rescan` to create the index.\n")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolIndexScan triggers an incremental scan.
func (s *MCPServer) toolIndexScan(id any, args map[string]interface{}) jsonRPCResponse {
	workers := 0 // auto-detect
	if w, ok := args["workers"].(float64); ok && w > 0 {
		workers = int(w)
	}

	start := time.Now()

	// Run incremental scan using scan package
	opts := scan.RunOptions{
		Workers: workers,
	}

	result, _, err := scan.RunWithOptions(s.butler.root, opts)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("scan failed: %v", err))
	}

	elapsed := time.Since(start)

	var output strings.Builder
	output.WriteString("# ✅ Incremental Scan Complete\n\n")
	fmt.Fprintf(&output, "| Metric | Value |\n")
	fmt.Fprintf(&output, "|--------|-------|\n")
	fmt.Fprintf(&output, "| Duration | %s |\n", formatDuration(elapsed))
	fmt.Fprintf(&output, "| Files Scanned | %d |\n", result.FileCount)
	fmt.Fprintf(&output, "| Symbols Found | %d |\n", result.SymbolCount)
	fmt.Fprintf(&output, "| Relationships | %d |\n", result.RelationshipCount)

	if workers > 0 {
		fmt.Fprintf(&output, "| Workers | %d |\n", workers)
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolIndexRescan forces a full rescan.
func (s *MCPServer) toolIndexRescan(id any, args map[string]interface{}) jsonRPCResponse {
	workers := 0 // auto-detect
	if w, ok := args["workers"].(float64); ok && w > 0 {
		workers = int(w)
	}

	start := time.Now()

	// Clear existing index first
	ctx := context.Background()
	_, _ = s.butler.db.ExecContext(ctx, `DELETE FROM chunks`)
	_, _ = s.butler.db.ExecContext(ctx, `DELETE FROM symbols`)
	_, _ = s.butler.db.ExecContext(ctx, `DELETE FROM edges`)
	_, _ = s.butler.db.ExecContext(ctx, `DELETE FROM scan_metadata`)

	// Run full scan
	opts := scan.RunOptions{
		Workers: workers,
	}

	result, _, err := scan.RunWithOptions(s.butler.root, opts)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("rescan failed: %v", err))
	}

	elapsed := time.Since(start)

	var output strings.Builder
	output.WriteString("# ✅ Full Rescan Complete\n\n")
	fmt.Fprintf(&output, "| Metric | Value |\n")
	fmt.Fprintf(&output, "|--------|-------|\n")
	fmt.Fprintf(&output, "| Duration | %s |\n", formatIndexDuration(elapsed))
	fmt.Fprintf(&output, "| Files Scanned | %d |\n", result.FileCount)
	fmt.Fprintf(&output, "| Symbols Found | %d |\n", result.SymbolCount)
	fmt.Fprintf(&output, "| Relationships | %d |\n", result.RelationshipCount)

	if workers > 0 {
		fmt.Fprintf(&output, "| Workers | %d |\n", workers)
	}

	output.WriteString("\nThe index has been completely rebuilt.\n")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolIndexStats returns detailed index statistics.
func (s *MCPServer) toolIndexStats(id any) jsonRPCResponse {
	ctx := context.Background()

	stats := getIndexStatsFromDB(s.butler.db, ctx)

	var output strings.Builder
	output.WriteString("# Index Statistics\n\n")

	output.WriteString("## Overview\n")
	fmt.Fprintf(&output, "| Metric | Count |\n")
	fmt.Fprintf(&output, "|--------|-------|\n")
	fmt.Fprintf(&output, "| Files | %d |\n", stats.FileCount)
	fmt.Fprintf(&output, "| Chunks | %d |\n", stats.ChunkCount)
	fmt.Fprintf(&output, "| Symbols | %d |\n", stats.SymbolCount)
	fmt.Fprintf(&output, "| Relationships | %d |\n", stats.RelationshipCount)

	if len(stats.SymbolsByKind) > 0 {
		output.WriteString("\n## Symbols by Kind\n")
		fmt.Fprintf(&output, "| Kind | Count |\n")
		fmt.Fprintf(&output, "|------|-------|\n")
		for kind, count := range stats.SymbolsByKind {
			fmt.Fprintf(&output, "| %s | %d |\n", kind, count)
		}
	}

	if len(stats.RelationshipsByKind) > 0 {
		output.WriteString("\n## Relationships by Kind\n")
		fmt.Fprintf(&output, "| Kind | Count |\n")
		fmt.Fprintf(&output, "|------|-------|\n")
		for kind, count := range stats.RelationshipsByKind {
			fmt.Fprintf(&output, "| %s | %d |\n", kind, count)
		}
	}

	if !stats.LastScan.IsZero() {
		output.WriteString("\n## Scan Info\n")
		fmt.Fprintf(&output, "| Metric | Value |\n")
		fmt.Fprintf(&output, "|--------|-------|\n")
		fmt.Fprintf(&output, "| Last Scan | %s |\n", stats.LastScan.Format("2006-01-02 15:04:05"))
		if stats.ScanHash != "" {
			hash := stats.ScanHash
			if len(hash) > 20 {
				hash = hash[:20] + "..."
			}
			fmt.Fprintf(&output, "| Scan Hash | %s |\n", hash)
		}
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// indexStats holds statistics about the indexed codebase.
type indexStats struct {
	FileCount           int
	SymbolCount         int
	SymbolsByKind       map[string]int
	RelationshipCount   int
	RelationshipsByKind map[string]int
	ChunkCount          int
	LastScan            time.Time
	ScanHash            string
}

// getIndexStatsFromDB retrieves statistics from the index database.
func getIndexStatsFromDB(db *sql.DB, ctx context.Context) *indexStats {
	stats := &indexStats{
		SymbolsByKind:       make(map[string]int),
		RelationshipsByKind: make(map[string]int),
	}

	// Count files
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM files").Scan(&stats.FileCount)

	// Count chunks
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM chunks").Scan(&stats.ChunkCount)

	// Count symbols
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM symbols").Scan(&stats.SymbolCount)

	// Count symbols by kind
	rows, err := db.QueryContext(ctx, "SELECT kind, COUNT(*) FROM symbols GROUP BY kind ORDER BY COUNT(*) DESC")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var kind string
			var count int
			if rows.Scan(&kind, &count) == nil {
				stats.SymbolsByKind[kind] = count
			}
		}
	}

	// Count relationships
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM relationships").Scan(&stats.RelationshipCount)

	// Count relationships by kind
	rows, err = db.QueryContext(ctx, "SELECT kind, COUNT(*) FROM relationships GROUP BY kind ORDER BY COUNT(*) DESC")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var kind string
			var count int
			if rows.Scan(&kind, &count) == nil {
				stats.RelationshipsByKind[kind] = count
			}
		}
	}

	// Get scan metadata
	var lastScanStr string
	if db.QueryRowContext(ctx, `SELECT value FROM scan_metadata WHERE key = 'completed_at'`).Scan(&lastScanStr) == nil {
		stats.LastScan, _ = time.Parse(time.RFC3339, lastScanStr)
	}

	_ = db.QueryRowContext(ctx, `SELECT value FROM scan_metadata WHERE key = 'scan_hash'`).Scan(&stats.ScanHash)

	return stats
}

// formatIndexDuration formats a duration in a human-readable way.
func formatIndexDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}
