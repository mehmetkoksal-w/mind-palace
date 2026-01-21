package butler

import (
	"fmt"
	"strings"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/memory"
)

// toolPatternsGet retrieves patterns from the memory store with filtering.
func (s *MCPServer) toolPatternsGet(id any, args map[string]interface{}) jsonRPCResponse {
	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	// Parse filters
	filters := memory.PatternFilters{
		Limit: 50,
	}

	if category, ok := args["category"].(string); ok && category != "" {
		filters.Category = category
	}
	if status, ok := args["status"].(string); ok && status != "" {
		filters.Status = status
	}
	if minConf, ok := args["min_confidence"].(float64); ok && minConf > 0 {
		filters.MinConfidence = minConf
	}
	if limit, ok := args["limit"].(float64); ok && limit > 0 {
		filters.Limit = int(limit)
	}

	patterns, err := mem.GetPatterns(filters)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get patterns failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Detected Patterns\n\n")

	if len(patterns) == 0 {
		output.WriteString("No patterns found matching the criteria.\n\n")
		output.WriteString("Run `palace patterns scan` to detect patterns in the codebase.\n")
	} else {
		// Group by status
		discovered := 0
		approved := 0
		ignored := 0
		for i := range patterns {
			switch patterns[i].Status {
			case "discovered":
				discovered++
			case "approved":
				approved++
			case "ignored":
				ignored++
			}
		}

		fmt.Fprintf(&output, "**Found:** %d patterns (%d discovered, %d approved, %d ignored)\n\n",
			len(patterns), discovered, approved, ignored)

		// Group by category
		byCategory := make(map[string][]*memory.Pattern)
		for i := range patterns {
			byCategory[patterns[i].Category] = append(byCategory[patterns[i].Category], &patterns[i])
		}

		for category, pats := range byCategory {
			fmt.Fprintf(&output, "## %s\n\n", strings.ToUpper(category))
			for _, p := range pats {
				statusIcon := "?"
				switch p.Status {
				case "discovered":
					statusIcon = "🔵"
				case "approved":
					statusIcon = "✅"
				case "ignored":
					statusIcon = "⬜"
				}

				confLevel := "LOW"
				if p.Confidence >= 0.85 {
					confLevel = "HIGH"
				} else if p.Confidence >= 0.70 {
					confLevel = "MED"
				}

				fmt.Fprintf(&output, "%s **%s** (`%s`)\n", statusIcon, p.Name, p.ID)
				fmt.Fprintf(&output, "   Confidence: %.0f%% (%s) | Detector: %s\n",
					p.Confidence*100, confLevel, p.DetectorID)
				if p.Description != "" {
					fmt.Fprintf(&output, "   %s\n", p.Description)
				}
				output.WriteString("\n")
			}
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

// toolPatternShow shows details of a specific pattern.
//
//nolint:gocognit // complex by design - formats detailed pattern info
func (s *MCPServer) toolPatternShow(id any, args map[string]interface{}) jsonRPCResponse {
	patternID, _ := args["pattern_id"].(string)
	if patternID == "" {
		return s.toolError(id, "pattern_id is required")
	}

	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	pattern, err := mem.GetPattern(patternID)
	if err != nil || pattern == nil {
		return s.toolError(id, fmt.Sprintf("pattern not found: %s", patternID))
	}

	locations, _ := mem.GetPatternLocations(patternID)

	var output strings.Builder
	fmt.Fprintf(&output, "# Pattern: %s\n\n", pattern.Name)

	statusIcon := "🔵"
	switch pattern.Status {
	case "approved":
		statusIcon = "✅"
	case "ignored":
		statusIcon = "⬜"
	}

	fmt.Fprintf(&output, "**ID:** `%s`\n", pattern.ID)
	fmt.Fprintf(&output, "**Status:** %s %s\n", statusIcon, pattern.Status)
	fmt.Fprintf(&output, "**Category:** %s/%s\n", pattern.Category, pattern.Subcategory)
	fmt.Fprintf(&output, "**Detector:** %s\n", pattern.DetectorID)
	output.WriteString("\n")

	fmt.Fprintf(&output, "## Confidence: %.1f%%\n\n", pattern.Confidence*100)
	fmt.Fprintf(&output, "| Factor | Score |\n")
	fmt.Fprintf(&output, "|--------|-------|\n")
	fmt.Fprintf(&output, "| Frequency (30%%) | %.1f%% |\n", pattern.FrequencyScore*100)
	fmt.Fprintf(&output, "| Consistency (30%%) | %.1f%% |\n", pattern.ConsistencyScore*100)
	fmt.Fprintf(&output, "| Spread (25%%) | %.1f%% |\n", pattern.SpreadScore*100)
	fmt.Fprintf(&output, "| Age (15%%) | %.1f%% |\n", pattern.AgeScore*100)
	output.WriteString("\n")

	if pattern.Description != "" {
		fmt.Fprintf(&output, "## Description\n\n%s\n\n", pattern.Description)
	}

	// Show locations
	matches := 0
	outliers := 0
	for i := range locations {
		if locations[i].IsOutlier {
			outliers++
		} else {
			matches++
		}
	}

	fmt.Fprintf(&output, "## Locations: %d matches, %d outliers\n\n", matches, outliers)

	// Show first few locations
	shown := 0
	for i := range locations {
		if locations[i].IsOutlier {
			continue
		}
		if shown >= 5 {
			fmt.Fprintf(&output, "... and %d more locations\n", matches-shown)
			break
		}
		fmt.Fprintf(&output, "- `%s:%d`", locations[i].FilePath, locations[i].LineStart)
		if locations[i].Snippet != "" {
			snippet := locations[i].Snippet
			if len(snippet) > 60 {
				snippet = snippet[:57] + "..."
			}
			fmt.Fprintf(&output, " - `%s`", snippet)
		}
		output.WriteString("\n")
		shown++
	}

	// Show outliers
	if outliers > 0 {
		output.WriteString("\n### Outliers (Deviations)\n\n")
		shown = 0
		for i := range locations {
			if !locations[i].IsOutlier {
				continue
			}
			if shown >= 3 {
				fmt.Fprintf(&output, "... and %d more outliers\n", outliers-shown)
				break
			}
			fmt.Fprintf(&output, "- `%s:%d`", locations[i].FilePath, locations[i].LineStart)
			if locations[i].OutlierReason != "" {
				fmt.Fprintf(&output, " - %s", locations[i].OutlierReason)
			}
			output.WriteString("\n")
			shown++
		}
	}

	if pattern.Status == "discovered" {
		output.WriteString("\n---\n")
		output.WriteString("Use `pattern_approve` to approve this pattern or `pattern_ignore` to ignore it.\n")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolPatternApprove approves a pattern and optionally creates a learning.
func (s *MCPServer) toolPatternApprove(id any, args map[string]interface{}) jsonRPCResponse {
	patternID, _ := args["pattern_id"].(string)
	if patternID == "" {
		return s.toolError(id, "pattern_id is required")
	}

	withLearning, _ := args["with_learning"].(bool)

	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	pattern, err := mem.GetPattern(patternID)
	if err != nil || pattern == nil {
		return s.toolError(id, fmt.Sprintf("pattern not found: %s", patternID))
	}

	if pattern.Status == "approved" {
		return s.toolError(id, fmt.Sprintf("pattern %s is already approved", patternID))
	}

	var output strings.Builder
	output.WriteString("# Pattern Approved\n\n")

	if withLearning {
		learningID, err := mem.ApprovePatternWithLearning(patternID)
		if err != nil {
			return s.toolError(id, fmt.Sprintf("approve with learning failed: %v", err))
		}
		fmt.Fprintf(&output, "**Pattern:** `%s`\n", patternID)
		fmt.Fprintf(&output, "**Name:** %s\n", pattern.Name)
		fmt.Fprintf(&output, "**Learning Created:** `%s`\n\n", learningID)
		output.WriteString("The pattern has been approved and a corresponding learning was created.\n")
		output.WriteString("The learning can now be used to enforce this pattern across the codebase.\n")
	} else {
		if err := mem.ApprovePattern(patternID, ""); err != nil {
			return s.toolError(id, fmt.Sprintf("approve failed: %v", err))
		}
		fmt.Fprintf(&output, "**Pattern:** `%s`\n", patternID)
		fmt.Fprintf(&output, "**Name:** %s\n\n", pattern.Name)
		output.WriteString("The pattern has been approved.\n")
		output.WriteString("Use `with_learning: true` to also create a learning for enforcement.\n")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolPatternIgnore ignores a pattern so it won't be shown in future results.
func (s *MCPServer) toolPatternIgnore(id any, args map[string]interface{}) jsonRPCResponse {
	patternID, _ := args["pattern_id"].(string)
	if patternID == "" {
		return s.toolError(id, "pattern_id is required")
	}

	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	pattern, err := mem.GetPattern(patternID)
	if err != nil || pattern == nil {
		return s.toolError(id, fmt.Sprintf("pattern not found: %s", patternID))
	}

	if pattern.Status == "ignored" {
		return s.toolError(id, fmt.Sprintf("pattern %s is already ignored", patternID))
	}

	if err := mem.IgnorePattern(patternID); err != nil {
		return s.toolError(id, fmt.Sprintf("ignore failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Pattern Ignored\n\n")
	fmt.Fprintf(&output, "**Pattern:** `%s`\n", patternID)
	fmt.Fprintf(&output, "**Name:** %s\n\n", pattern.Name)
	output.WriteString("The pattern has been marked as ignored and will not appear in future scans.\n")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolPatternStats returns statistics about detected patterns.
func (s *MCPServer) toolPatternStats(id any, _ map[string]interface{}) jsonRPCResponse {
	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	// Get all patterns
	patterns, err := mem.GetPatterns(memory.PatternFilters{Limit: 10000})
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get patterns failed: %v", err))
	}

	// Calculate stats
	total := len(patterns)
	discovered := 0
	approved := 0
	ignored := 0
	byCategory := make(map[string]int)
	totalConfidence := 0.0

	for i := range patterns {
		switch patterns[i].Status {
		case "discovered":
			discovered++
		case "approved":
			approved++
		case "ignored":
			ignored++
		}
		byCategory[patterns[i].Category]++
		totalConfidence += patterns[i].Confidence
	}

	avgConfidence := 0.0
	if total > 0 {
		avgConfidence = totalConfidence / float64(total)
	}

	var output strings.Builder
	output.WriteString("# Pattern Statistics\n\n")

	fmt.Fprintf(&output, "| Metric | Value |\n")
	fmt.Fprintf(&output, "|--------|-------|\n")
	fmt.Fprintf(&output, "| Total Patterns | %d |\n", total)
	fmt.Fprintf(&output, "| Discovered | %d |\n", discovered)
	fmt.Fprintf(&output, "| Approved | %d |\n", approved)
	fmt.Fprintf(&output, "| Ignored | %d |\n", ignored)
	fmt.Fprintf(&output, "| Avg Confidence | %.1f%% |\n", avgConfidence*100)
	output.WriteString("\n")

	if len(byCategory) > 0 {
		output.WriteString("## By Category\n\n")
		fmt.Fprintf(&output, "| Category | Count |\n")
		fmt.Fprintf(&output, "|----------|-------|\n")
		for cat, count := range byCategory {
			fmt.Fprintf(&output, "| %s | %d |\n", cat, count)
		}
		output.WriteString("\n")
	}

	if discovered > 0 {
		output.WriteString("---\n")
		fmt.Fprintf(&output, "**%d patterns** are waiting for review.\n", discovered)
		output.WriteString("Use `patterns_get` with `status: discovered` to see them.\n")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}
