package butler

import (
	"fmt"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
)

// toolExploreContext gets complete context for a task (the ORACLE query).
func (s *MCPServer) toolExploreContext(id any, args map[string]interface{}) jsonRPCResponse {
	task, _ := args["task"].(string)
	if task == "" {
		return s.toolError(id, "task is required")
	}

	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	maxTokens := 0
	if mt, ok := args["maxTokens"].(float64); ok {
		maxTokens = int(mt)
	}

	includeTests := false
	if it, ok := args["includeTests"].(bool); ok {
		includeTests = it
	}

	// Memory-enhanced options (default to true)
	includeLearnings := true
	if il, ok := args["includeLearnings"].(bool); ok {
		includeLearnings = il
	}

	includeFileIntel := true
	if ifi, ok := args["includeFileIntel"].(bool); ok {
		includeFileIntel = ifi
	}

	includeIdeas := true
	if ii, ok := args["includeIdeas"].(bool); ok {
		includeIdeas = ii
	}

	includeDecisions := true
	if id, ok := args["includeDecisions"].(bool); ok {
		includeDecisions = id
	}

	// Use enhanced context with memory data
	opts := EnhancedContextOptions{
		Query:            task,
		Limit:            limit,
		MaxTokens:        maxTokens,
		IncludeTests:     includeTests,
		IncludeLearnings: includeLearnings,
		IncludeFileIntel: includeFileIntel,
		IncludeIdeas:     includeIdeas,
		IncludeDecisions: includeDecisions,
	}

	result, err := s.butler.GetEnhancedContext(opts)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get context failed: %v", err))
	}

	// Format as readable markdown
	var output strings.Builder
	output.WriteString("# Context for Task\n\n")
	fmt.Fprintf(&output, "**Task:** %s\n\n", task)
	fmt.Fprintf(&output, "**Total Files:** %d\n\n", result.TotalFiles)

	// Show learnings first (important context from session memory)
	if len(result.Learnings) > 0 {
		output.WriteString("## 💡 Relevant Learnings\n\n")
		output.WriteString("_Knowledge accumulated from previous sessions:_\n\n")
		for _, l := range result.Learnings {
			scopeInfo := ""
			if l.Scope != "palace" {
				scopeInfo = fmt.Sprintf(" [%s]", l.Scope)
			}
			fmt.Fprintf(&output, "- **[%.0f%%]**%s %s\n", l.Confidence*100, scopeInfo, l.Content)
		}
		output.WriteString("\n")
	}

	// Show brain ideas
	if len(result.BrainIdeas) > 0 {
		output.WriteString("## 💭 Relevant Ideas\n\n")
		output.WriteString("_Exploratory thoughts that may be useful:_\n\n")
		for _, idea := range result.BrainIdeas {
			statusIcon := "🔵"
			switch idea.Status {
			case "exploring":
				statusIcon = "🔍"
			case "implemented":
				statusIcon = "✅"
			case "dropped":
				statusIcon = "❌"
			}
			fmt.Fprintf(&output, "- %s `%s` %s\n", statusIcon, idea.ID, idea.Content)
		}
		output.WriteString("\n")
	}

	// Show brain decisions
	if len(result.BrainDecisions) > 0 {
		output.WriteString("## 📋 Relevant Brain Decisions\n\n")
		output.WriteString("_Past architectural choices from the brain:_\n\n")
		for _, d := range result.BrainDecisions {
			statusIcon := "🟢"
			switch d.Status {
			case "superseded":
				statusIcon = "🔄"
			case "reversed":
				statusIcon = "↩️"
			}
			outcomeIcon := ""
			switch d.Outcome {
			case "successful":
				outcomeIcon = " ✅"
			case "failed":
				outcomeIcon = " ❌"
			case "mixed":
				outcomeIcon = " ⚖️"
			}
			fmt.Fprintf(&output, "- %s `%s`%s %s\n", statusIcon, d.ID, outcomeIcon, d.Content)
			if d.Rationale != "" {
				fmt.Fprintf(&output, "  _Rationale: %s_\n", d.Rationale)
			}
		}
		output.WriteString("\n")
	}

	// Show related links between brain records
	if len(result.RelatedLinks) > 0 {
		output.WriteString("## 🔗 Related Links\n\n")
		output.WriteString("_Relationships between ideas, decisions, and code:_\n\n")
		for _, link := range result.RelatedLinks {
			staleWarning := ""
			if link.IsStale {
				staleWarning = " ⚠️ stale"
			}
			fmt.Fprintf(&output, "- `%s` **%s** `%s` (%s)%s\n",
				link.SourceID, link.Relation, link.TargetID, link.TargetKind, staleWarning)
		}
		output.WriteString("\n")
	}

	// Show decision conflicts
	if len(result.DecisionConflicts) > 0 {
		output.WriteString("## ⚠️ Decision Conflicts\n\n")
		output.WriteString("_Potential conflicts with existing decisions:_\n\n")
		for _, conflict := range result.DecisionConflicts {
			conflictIcon := "⚠️"
			if conflict.ConflictType == "contradicts" {
				conflictIcon = "❌"
			}
			fmt.Fprintf(&output, "- %s **%s** `%s`\n", conflictIcon, conflict.ConflictType, conflict.ConflictingID)
			fmt.Fprintf(&output, "  %s\n", conflict.Reason)
			fmt.Fprintf(&output, "  _Decision: %s_\n", conflict.Decision.Content)
			if conflict.Decision.Outcome != "unknown" {
				fmt.Fprintf(&output, "  _Outcome: %s_\n", conflict.Decision.Outcome)
			}
		}
		output.WriteString("\n")
	}

	if len(result.Symbols) > 0 {
		output.WriteString("## Relevant Symbols\n\n")
		for _, sym := range result.Symbols {
			exportMark := ""
			if sym.Exported {
				exportMark = " (exported)"
			}
			fmt.Fprintf(&output, "- **%s** `%s`%s\n", sym.Kind, sym.Name, exportMark)
			fmt.Fprintf(&output, "  - File: `%s` (lines %d-%d)\n", sym.FilePath, sym.LineStart, sym.LineEnd)
			if sym.Signature != "" {
				fmt.Fprintf(&output, "  - Signature: `%s`\n", sym.Signature)
			}
			if sym.DocComment != "" {
				fmt.Fprintf(&output, "  - Doc: %s\n", sym.DocComment)
			}
		}
		output.WriteString("\n")
	}

	if len(result.Files) > 0 {
		output.WriteString("## Relevant Files\n\n")
		for _, f := range result.Files {
			// Check for file intel warnings
			fileWarning := ""
			if result.FileIntel != nil {
				if intel, ok := result.FileIntel[f.Path]; ok {
					if intel.FailureCount > 0 && intel.EditCount > 0 {
						failureRate := float64(intel.FailureCount) / float64(intel.EditCount) * 100
						if failureRate > 20 {
							fileWarning = fmt.Sprintf(" ⚠️ (%.0f%% failure rate)", failureRate)
						}
					}
				}
			}
			fmt.Fprintf(&output, "### `%s` (%s)%s\n", f.Path, f.Language, fileWarning)
			if f.Snippet != "" {
				fmt.Fprintf(&output, "Lines %d-%d:\n```\n%s\n```\n\n", f.ChunkStart, f.ChunkEnd, f.Snippet)
			}
			if len(f.Symbols) > 0 {
				output.WriteString("**Symbols in file:**\n")
				for _, sym := range f.Symbols {
					fmt.Fprintf(&output, "- `%s` (%s)\n", sym.Name, sym.Kind)
				}
			}
			output.WriteString("\n")
		}
	}

	// Show file intel summary if any files have significant history
	if len(result.FileIntel) > 0 {
		hasSignificantIntel := false
		for _, intel := range result.FileIntel {
			if intel.EditCount >= 3 || intel.FailureCount > 0 {
				hasSignificantIntel = true
				break
			}
		}
		if hasSignificantIntel {
			output.WriteString("## 📊 File Intelligence\n\n")
			for path, intel := range result.FileIntel {
				if intel.EditCount >= 3 || intel.FailureCount > 0 {
					warning := ""
					if intel.FailureCount > 0 {
						warning = fmt.Sprintf(" ⚠️ %d failures", intel.FailureCount)
					}
					fmt.Fprintf(&output, "- `%s`: %d edits%s\n", path, intel.EditCount, warning)
				}
			}
			output.WriteString("\n")
		}
	}

	if len(result.Imports) > 0 {
		output.WriteString("## Import Relationships\n\n")
		for _, imp := range result.Imports {
			fmt.Fprintf(&output, "- `%s` imports `%s`\n", imp.SourceFile, imp.TargetFile)
		}
		output.WriteString("\n")
	}

	if len(result.Decisions) > 0 {
		output.WriteString("## Related Decisions\n\n")
		for _, d := range result.Decisions {
			fmt.Fprintf(&output, "### %s\n", d.Title)
			fmt.Fprintf(&output, "%s\n\n", d.Summary)
		}
	}

	if len(result.Warnings) > 0 {
		output.WriteString("## Warnings\n\n")
		for _, w := range result.Warnings {
			fmt.Fprintf(&output, "- ⚠️ %s\n", w)
		}
	}

	if result.TokenStats != nil {
		output.WriteString("\n## Token Usage\n\n")
		fmt.Fprintf(&output, "- **Total tokens:** %d\n", result.TokenStats.TotalTokens)
		fmt.Fprintf(&output, "- **Symbol tokens:** %d\n", result.TokenStats.SymbolTokens)
		fmt.Fprintf(&output, "- **File tokens:** %d\n", result.TokenStats.FileTokens)
		fmt.Fprintf(&output, "- **Import tokens:** %d\n", result.TokenStats.ImportTokens)
		if result.TokenStats.Budget > 0 {
			fmt.Fprintf(&output, "- **Budget:** %d\n", result.TokenStats.Budget)
			if result.TokenStats.Truncated {
				output.WriteString("- ⚠️ Results truncated to fit budget\n")
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

// toolExploreImpact analyzes the impact of changing a file or symbol.
func (s *MCPServer) toolExploreImpact(id any, args map[string]interface{}) jsonRPCResponse {
	target, _ := args["target"].(string)
	if target == "" {
		return s.toolError(id, "target is required")
	}

	result, err := s.butler.GetImpact(target)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get impact failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Impact Analysis\n\n")
	fmt.Fprintf(&output, "**Target:** `%s`\n\n", target)

	if len(result.Dependents) > 0 {
		output.WriteString("## Files that depend on this (will be affected by changes)\n\n")
		for _, dep := range result.Dependents {
			fmt.Fprintf(&output, "- `%s`\n", dep)
		}
		output.WriteString("\n")
	} else {
		output.WriteString("## Dependents\nNo files depend on this target.\n\n")
	}

	if len(result.Dependencies) > 0 {
		output.WriteString("## Files this depends on\n\n")
		for _, dep := range result.Dependencies {
			fmt.Fprintf(&output, "- `%s`\n", dep)
		}
		output.WriteString("\n")
	} else {
		output.WriteString("## Dependencies\nThis target has no dependencies.\n\n")
	}

	if len(result.Symbols) > 0 {
		output.WriteString("## Symbols in this file\n\n")
		for _, sym := range result.Symbols {
			fmt.Fprintf(&output, "- **%s** `%s`", sym.Kind, sym.Name)
			if sym.Signature != "" {
				fmt.Fprintf(&output, " - `%s`", sym.Signature)
			}
			output.WriteString("\n")
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

// toolExploreSymbols lists symbols of a specific kind in the codebase.
func (s *MCPServer) toolExploreSymbols(id any, args map[string]interface{}) jsonRPCResponse {
	kind, _ := args["kind"].(string)
	if kind == "" {
		return s.toolError(id, "kind is required")
	}

	limit := 50
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	symbols, err := s.butler.ListSymbols(kind, limit)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("list symbols failed: %v", err))
	}

	var output strings.Builder
	fmt.Fprintf(&output, "# Symbols of kind: %s\n\n", kind)
	fmt.Fprintf(&output, "Found %d symbols:\n\n", len(symbols))

	for _, sym := range symbols {
		exportMark := ""
		if sym.Exported {
			exportMark = " ✓"
		}
		fmt.Fprintf(&output, "- `%s`%s\n", sym.Name, exportMark)
		fmt.Fprintf(&output, "  - File: `%s` (lines %d-%d)\n", sym.FilePath, sym.LineStart, sym.LineEnd)
		if sym.Signature != "" {
			fmt.Fprintf(&output, "  - Signature: `%s`\n", sym.Signature)
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

// toolExploreSymbol gets detailed information about a specific symbol.
func (s *MCPServer) toolExploreSymbol(id any, args map[string]interface{}) jsonRPCResponse {
	name, _ := args["name"].(string)
	if name == "" {
		return s.toolError(id, "name is required")
	}

	filePath, _ := args["file"].(string)

	sym, err := s.butler.GetSymbol(name, filePath)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("symbol not found: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Symbol Details\n\n")
	fmt.Fprintf(&output, "**Name:** `%s`\n", sym.Name)
	fmt.Fprintf(&output, "**Kind:** %s\n", sym.Kind)
	fmt.Fprintf(&output, "**File:** `%s`\n", sym.FilePath)
	fmt.Fprintf(&output, "**Lines:** %d-%d\n", sym.LineStart, sym.LineEnd)
	fmt.Fprintf(&output, "**Exported:** %v\n", sym.Exported)
	if sym.Signature != "" {
		fmt.Fprintf(&output, "\n**Signature:**\n```\n%s\n```\n", sym.Signature)
	}
	if sym.DocComment != "" {
		fmt.Fprintf(&output, "\n**Documentation:**\n%s\n", sym.DocComment)
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolExploreFile gets all exported symbols in a specific file.
func (s *MCPServer) toolExploreFile(id any, args map[string]interface{}) jsonRPCResponse {
	file, _ := args["file"].(string)
	if file == "" {
		return s.toolError(id, "file is required")
	}

	symbols, err := s.butler.GetFileSymbols(file)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get file symbols failed: %v", err))
	}

	var output strings.Builder
	fmt.Fprintf(&output, "# Exported Symbols in `%s`\n\n", file)

	if len(symbols) == 0 {
		output.WriteString("No exported symbols found in this file.\n")
	} else {
		for _, sym := range symbols {
			fmt.Fprintf(&output, "## %s `%s`\n", sym.Kind, sym.Name)
			fmt.Fprintf(&output, "- Lines: %d-%d\n", sym.LineStart, sym.LineEnd)
			if sym.Signature != "" {
				fmt.Fprintf(&output, "- Signature: `%s`\n", sym.Signature)
			}
			if sym.DocComment != "" {
				fmt.Fprintf(&output, "- Doc: %s\n", sym.DocComment)
			}
			output.WriteString("\n")
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

// toolExploreDeps gets the dependency graph for one or more files.
func (s *MCPServer) toolExploreDeps(id any, args map[string]interface{}) jsonRPCResponse {
	filesRaw, ok := args["files"].([]interface{})
	if !ok || len(filesRaw) == 0 {
		return s.toolError(id, "files array is required")
	}

	files := make([]string, 0, len(filesRaw))
	for _, f := range filesRaw {
		if s, ok := f.(string); ok {
			files = append(files, s)
		}
	}

	graph, err := s.butler.GetDependencyGraph(files)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get dependencies failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Dependency Graph\n\n")

	for _, node := range graph {
		fmt.Fprintf(&output, "## `%s`", node.File)
		if node.Language != "" {
			fmt.Fprintf(&output, " (%s)", node.Language)
		}
		output.WriteString("\n")

		if len(node.Imports) > 0 {
			output.WriteString("**Imports:**\n")
			for _, imp := range node.Imports {
				fmt.Fprintf(&output, "- `%s`\n", imp)
			}
		} else {
			output.WriteString("No imports.\n")
		}
		output.WriteString("\n")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolContextAutoInject returns auto-assembled context for a file.
// This is designed to be called when an AI agent focuses on a file.
func (s *MCPServer) toolContextAutoInject(id any, args map[string]interface{}) jsonRPCResponse {
	filePath, _ := args["file_path"].(string)
	if filePath == "" {
		return s.toolError(id, "file_path is required")
	}

	// Build config from args or use defaults
	cfg := s.butler.Config().AutoInjection
	if cfg == nil {
		cfg = config.DefaultAutoInjectionConfig()
	}

	// Override config from args if provided
	if mt, ok := args["maxTokens"].(float64); ok {
		cfg.MaxTokens = int(mt)
	}
	if il, ok := args["includeLearnings"].(bool); ok {
		cfg.IncludeLearnings = il
	}
	if id, ok := args["includeDecisions"].(bool); ok {
		cfg.IncludeDecisions = id
	}
	if if_, ok := args["includeFailures"].(bool); ok {
		cfg.IncludeFailures = if_
	}

	ctx, err := s.butler.GetAutoInjectionContext(filePath, cfg)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get auto context failed: %v", err))
	}

	// Format as readable markdown for agent consumption
	var output strings.Builder
	output.WriteString("# Context for File\n\n")
	fmt.Fprintf(&output, "**File:** `%s`\n", filePath)
	if ctx.Room != "" {
		fmt.Fprintf(&output, "**Room:** `%s`\n", ctx.Room)
	}
	fmt.Fprintf(&output, "**Estimated Tokens:** %d\n\n", ctx.TotalTokens)

	// Show warnings first (most important)
	if len(ctx.Warnings) > 0 {
		output.WriteString("## ⚠️ Warnings\n\n")
		for _, w := range ctx.Warnings {
			icon := "⚠️"
			switch w.Type {
			case "contradiction":
				icon = "❌"
			case "fragile_file":
				icon = "🔥"
			case "unreviewed_decision":
				icon = "❓"
			}
			fmt.Fprintf(&output, "- %s **%s**: %s\n", icon, w.Type, w.Message)
			if w.Details != "" {
				fmt.Fprintf(&output, "  > %s\n", truncate(w.Details, 100))
			}
		}
		output.WriteString("\n")
	}

	// Show learnings
	if len(ctx.Learnings) > 0 {
		output.WriteString("## 💡 Relevant Learnings\n\n")
		for _, pl := range ctx.Learnings {
			l := pl.Learning
			priorityIcon := "📌"
			if pl.Priority >= 1.0 {
				priorityIcon = "⭐"
			} else if pl.Priority < 0.6 {
				priorityIcon = "💤"
			}
			fmt.Fprintf(&output, "- %s **[%.0f%%]** %s\n", priorityIcon, l.Confidence*100, l.Content)
			fmt.Fprintf(&output, "  _Why: %s_\n", pl.Reason)
		}
		output.WriteString("\n")
	}

	// Show decisions
	if len(ctx.Decisions) > 0 {
		output.WriteString("## 📋 Active Decisions\n\n")
		for _, d := range ctx.Decisions {
			statusIcon := "🟢"
			if d.Outcome == "failed" {
				statusIcon = "❌"
			} else if d.Outcome == "mixed" {
				statusIcon = "⚖️"
			}
			fmt.Fprintf(&output, "- %s **%s**\n", statusIcon, d.Content)
			if d.Rationale != "" {
				fmt.Fprintf(&output, "  _Rationale: %s_\n", d.Rationale)
			}
			fmt.Fprintf(&output, "  _Scope: %s | Status: %s_\n", d.Scope, d.Outcome)
		}
		output.WriteString("\n")
	}

	// Show failures
	if len(ctx.Failures) > 0 {
		output.WriteString("## 🔥 File Failures\n\n")
		for _, f := range ctx.Failures {
			severityIcon := "🟡"
			if f.Severity == "high" {
				severityIcon = "🔴"
			} else if f.Severity == "low" {
				severityIcon = "⚪"
			}
			fmt.Fprintf(&output, "- %s `%s`: %d failures\n", severityIcon, f.Path, f.FailureCount)
		}
		output.WriteString("\n")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolScopeExplain explains the scope inheritance for a file.
func (s *MCPServer) toolScopeExplain(id any, args map[string]interface{}) jsonRPCResponse {
	filePath, _ := args["file_path"].(string)
	if filePath == "" {
		return s.toolError(id, "file_path is required")
	}

	scopeCfg := s.butler.Config().Scope

	explanation, err := s.butler.GetScopeExplanation(filePath, scopeCfg)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get scope explanation failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Scope Inheritance Explanation\n\n")
	fmt.Fprintf(&output, "**File:** `%s`\n", filePath)
	if explanation.ResolvedRoom != "" {
		fmt.Fprintf(&output, "**Resolved Room:** `%s`\n", explanation.ResolvedRoom)
	}
	output.WriteString("\n")

	output.WriteString("## Inheritance Chain\n\n")
	output.WriteString("Knowledge flows from broader to narrower scope:\n\n")

	for i, level := range explanation.InheritanceChain {
		indent := strings.Repeat("  ", i)
		activeIcon := "✅"
		if !level.Active {
			activeIcon = "❌"
		}

		path := level.Path
		if path == "" {
			path = "(workspace)"
		}

		fmt.Fprintf(&output, "%s%s **%s** → `%s` (%d records)\n",
			indent, activeIcon, level.Scope, path, level.RecordCount)
	}
	output.WriteString("\n")

	output.WriteString("## Records by Scope\n\n")
	for scope, count := range explanation.TotalRecords {
		fmt.Fprintf(&output, "- **%s**: %d records\n", scope, count)
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}
