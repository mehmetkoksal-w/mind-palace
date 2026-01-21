package butler

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/config"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/memory"
)

// toolContextFocus sets the current task focus for smart context prioritization.
func (s *MCPServer) toolContextFocus(id any, args map[string]interface{}) jsonRPCResponse {
	task, _ := args["task"].(string)
	if task == "" {
		// Return current focus if no task provided
		if s.currentTaskFocus == "" {
			return jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Result: mcpToolResult{
					Content: []mcpContent{{Type: "text", Text: "No task focus currently set. Use `context_focus({task: '...'})` to set one."}},
				},
			}
		}
		var output strings.Builder
		output.WriteString("# Current Task Focus\n\n")
		fmt.Fprintf(&output, "**Task:** %s\n", s.currentTaskFocus)
		if len(s.focusKeywords) > 0 {
			fmt.Fprintf(&output, "**Keywords:** %s\n", strings.Join(s.focusKeywords, ", "))
		}
		if len(s.contextPriorityUp) > 0 {
			fmt.Fprintf(&output, "**Pinned Records:** %d\n", len(s.contextPriorityUp))
		}
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: mcpToolResult{
				Content: []mcpContent{{Type: "text", Text: output.String()}},
			},
		}
	}

	// Set new focus
	s.currentTaskFocus = task
	s.focusKeywords = extractKeywords(task)

	// Clear pinned records when focus changes
	s.contextPriorityUp = nil

	// Handle pin requests
	if pinRaw, ok := args["pin"].([]interface{}); ok {
		for _, p := range pinRaw {
			if pid, ok := p.(string); ok && pid != "" {
				s.contextPriorityUp = append(s.contextPriorityUp, pid)
			}
		}
	}

	var output strings.Builder
	output.WriteString("# Task Focus Set\n\n")
	fmt.Fprintf(&output, "**Task:** %s\n", task)
	fmt.Fprintf(&output, "**Keywords:** %s\n", strings.Join(s.focusKeywords, ", "))

	if len(s.contextPriorityUp) > 0 {
		fmt.Fprintf(&output, "**Pinned Records:** %d\n", len(s.contextPriorityUp))
	}

	output.WriteString("\n---\n")
	output.WriteString("Context will now be prioritized based on relevance to this task.\n")
	output.WriteString("Use `context_get` to retrieve focused context.\n")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolContextGet retrieves context prioritized by current focus.
func (s *MCPServer) toolContextGet(id any, args map[string]interface{}) jsonRPCResponse {
	maxTokens := 2000
	if mt, ok := args["maxTokens"].(float64); ok && mt > 0 {
		maxTokens = int(mt)
	}

	filePath, _ := args["file"].(string)

	var output strings.Builder
	output.WriteString("# Focused Context\n\n")

	if s.currentTaskFocus != "" {
		fmt.Fprintf(&output, "**Current Focus:** %s\n\n", s.currentTaskFocus)
	}

	mem := s.butler.Memory()
	if mem == nil {
		output.WriteString("Memory not initialized.\n")
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: mcpToolResult{
				Content: []mcpContent{{Type: "text", Text: output.String()}},
			},
		}
	}

	// Get prioritized learnings
	cfg := s.butler.Config()
	var autoInjectCfg *config.AutoInjectionConfig
	if cfg != nil && cfg.AutoInjection != nil {
		autoInjectCfg = cfg.AutoInjection
	} else {
		autoInjectCfg = config.DefaultAutoInjectionConfig()
	}
	autoInjectCfg.MaxTokens = maxTokens

	// Get base context
	ctx, err := s.butler.GetAutoInjectionContext(filePath, autoInjectCfg)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get context failed: %v", err))
	}

	// Apply focus-based prioritization
	prioritizedLearnings := s.prioritizeByFocus(ctx.Learnings)

	// Add pinned records first
	pinnedLearnings := s.getPinnedLearnings(mem)

	// Build output
	if len(pinnedLearnings) > 0 {
		output.WriteString("## 📌 Pinned Learnings\n\n")
		for i := range pinnedLearnings {
			l := &pinnedLearnings[i]
			fmt.Fprintf(&output, "- `%s` (%.0f%%): %s\n", l.ID, l.Confidence*100, truncateString(l.Content, 80))
		}
		output.WriteString("\n")
	}

	if len(prioritizedLearnings) > 0 {
		output.WriteString("## 📚 Relevant Learnings\n\n")
		// Show top prioritized learnings
		shown := 0
		for i := range prioritizedLearnings {
			if shown >= 10 {
				remaining := len(prioritizedLearnings) - shown
				if remaining > 0 {
					fmt.Fprintf(&output, "- ... and %d more\n", remaining)
				}
				break
			}
			pl := &prioritizedLearnings[i]
			l := &pl.Learning
			relevance := ""
			if pl.Priority > 0.8 {
				relevance = " ⭐"
			}
			fmt.Fprintf(&output, "- `%s` (%.0f%%)%s: %s\n", l.ID, l.Confidence*100, relevance, truncateString(l.Content, 80))
			shown++
		}
		output.WriteString("\n")
	}

	if len(ctx.Decisions) > 0 {
		output.WriteString("## 📋 Active Decisions\n\n")
		for i := range ctx.Decisions {
			d := &ctx.Decisions[i]
			fmt.Fprintf(&output, "- `%s`: %s\n", d.ID, truncateString(d.Content, 80))
		}
		output.WriteString("\n")
	}

	if len(ctx.Warnings) > 0 {
		output.WriteString("## ⚠️ Warnings\n\n")
		for i := range ctx.Warnings {
			w := &ctx.Warnings[i]
			fmt.Fprintf(&output, "- %s\n", w.Message)
		}
		output.WriteString("\n")
	}

	// Add token estimate
	fmt.Fprintf(&output, "---\n**Estimated tokens:** %d / %d\n", ctx.TotalTokens, maxTokens)

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolContextPin pins specific records to always include in context.
func (s *MCPServer) toolContextPin(id any, args map[string]interface{}) jsonRPCResponse {
	recordID, _ := args["id"].(string)
	unpin, _ := args["unpin"].(bool)

	if recordID == "" {
		// List pinned records
		if len(s.contextPriorityUp) == 0 {
			return jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Result: mcpToolResult{
					Content: []mcpContent{{Type: "text", Text: "No records currently pinned."}},
				},
			}
		}
		var output strings.Builder
		output.WriteString("# Pinned Records\n\n")
		for _, pid := range s.contextPriorityUp {
			fmt.Fprintf(&output, "- `%s`\n", pid)
		}
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: mcpToolResult{
				Content: []mcpContent{{Type: "text", Text: output.String()}},
			},
		}
	}

	if unpin {
		// Remove from pinned
		newPinned := make([]string, 0, len(s.contextPriorityUp))
		for _, pid := range s.contextPriorityUp {
			if pid != recordID {
				newPinned = append(newPinned, pid)
			}
		}
		s.contextPriorityUp = newPinned
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: mcpToolResult{
				Content: []mcpContent{{Type: "text", Text: fmt.Sprintf("Record `%s` unpinned from context.", recordID)}},
			},
		}
	}

	// Add to pinned
	for _, pid := range s.contextPriorityUp {
		if pid == recordID {
			return jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Result: mcpToolResult{
					Content: []mcpContent{{Type: "text", Text: fmt.Sprintf("Record `%s` is already pinned.", recordID)}},
				},
			}
		}
	}
	s.contextPriorityUp = append(s.contextPriorityUp, recordID)

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: fmt.Sprintf("Record `%s` pinned to context. It will be included in all context retrievals.", recordID)}},
		},
	}
}

// prioritizeByFocus reorders learnings based on current task focus.
func (s *MCPServer) prioritizeByFocus(learnings []PrioritizedLearning) []PrioritizedLearning {
	if s.currentTaskFocus == "" || len(s.focusKeywords) == 0 {
		return learnings
	}

	// Score each learning by keyword matches
	for i := range learnings {
		pl := &learnings[i]
		matchScore := 0.0
		contentLower := strings.ToLower(pl.Learning.Content)

		for _, kw := range s.focusKeywords {
			if strings.Contains(contentLower, kw) {
				matchScore += 0.2
			}
		}

		// Boost recent learnings
		daysSince := time.Since(pl.Learning.LastUsed).Hours() / 24
		if daysSince < 7 {
			matchScore += 0.1
		}

		pl.Priority += matchScore
	}

	// Re-sort by priority
	sort.Slice(learnings, func(i, j int) bool {
		return learnings[i].Priority > learnings[j].Priority
	})

	return learnings
}

// getPinnedLearnings retrieves the pinned learning records.
func (s *MCPServer) getPinnedLearnings(mem *memory.Memory) []memory.Learning {
	var pinned []memory.Learning
	for _, pid := range s.contextPriorityUp {
		if strings.HasPrefix(pid, "l_") {
			if l, err := mem.GetLearning(pid); err == nil && l != nil {
				pinned = append(pinned, *l)
			}
		}
	}
	return pinned
}

// extractKeywords extracts meaningful keywords from a task description.
func extractKeywords(text string) []string {
	// Common stop words to filter out
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "as": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true, "did": true,
		"will": true, "would": true, "could": true, "should": true, "may": true,
		"might": true, "must": true, "shall": true, "can": true, "need": true,
		"i": true, "you": true, "he": true, "she": true, "it": true, "we": true,
		"they": true, "what": true, "which": true, "who": true, "when": true,
		"where": true, "why": true, "how": true, "this": true, "that": true,
		"these": true, "those": true, "am": true, "not": true, "no": true,
	}

	words := strings.Fields(strings.ToLower(text))
	var keywords []string
	seen := make(map[string]bool)

	for _, word := range words {
		// Clean word
		word = strings.Trim(word, ".,;:!?()[]{}\"'")
		if len(word) < 3 {
			continue
		}
		if stopWords[word] {
			continue
		}
		if seen[word] {
			continue
		}
		seen[word] = true
		keywords = append(keywords, word)
	}

	return keywords
}
