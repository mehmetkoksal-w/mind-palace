package butler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/llm"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/memory"
)

// SmartBriefingRequest contains parameters for generating a smart briefing.
type SmartBriefingRequest struct {
	Context     string `json:"context"`     // "file", "room", "task", "workspace"
	ContextPath string `json:"contextPath"` // file path, room name, or task description
	Style       string `json:"style"`       // "summary", "detailed", "actionable"
}

// SmartBriefing contains an LLM-generated briefing.
type SmartBriefing struct {
	Summary        string            `json:"summary"`
	KeyPoints      []string          `json:"keyPoints"`
	Warnings       []BriefingWarning `json:"warnings"`
	Suggestions    []string          `json:"suggestions"`
	RelatedRecords []BriefingRecord  `json:"relatedRecords"`
	GeneratedAt    time.Time         `json:"generatedAt"`
}

// BriefingWarning represents a warning in the briefing.
type BriefingWarning struct {
	Type     string `json:"type"` // "contradiction", "stale", "at_risk"
	Message  string `json:"message"`
	RecordID string `json:"recordId,omitempty"`
}

// BriefingRecord represents a related record in the briefing.
type BriefingRecord struct {
	ID      string  `json:"id"`
	Kind    string  `json:"kind"`
	Content string  `json:"content"`
	Score   float64 `json:"score,omitempty"`
}

// toolBriefingSmart generates an LLM-powered smart briefing.
func (s *MCPServer) toolBriefingSmart(id any, args map[string]interface{}) jsonRPCResponse {
	contextType, _ := args["context"].(string)
	if contextType == "" {
		contextType = "workspace"
	}

	contextPath, _ := args["contextPath"].(string)
	style, _ := args["style"].(string)
	if style == "" {
		style = "summary"
	}

	// Check if LLM is configured
	llmClient, err := s.butler.GetLLMClient()
	if err != nil || llmClient == nil {
		return s.generateNonLLMBriefing(id, contextType, contextPath)
	}

	// Gather context data
	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not available")
	}

	var contextData strings.Builder
	contextData.WriteString(fmt.Sprintf("Generate a %s briefing for ", style))

	switch contextType {
	case "file":
		contextData.WriteString(fmt.Sprintf("the file '%s'.\n\n", contextPath))
		// Get file-specific learnings
		learnings, _ := mem.GetFileLearnings(contextPath)
		if len(learnings) > 0 {
			contextData.WriteString("## Learnings for this file:\n")
			for i := range learnings {
				l := &learnings[i]
				contextData.WriteString(fmt.Sprintf("- [%s] (%.0f%% confidence): %s\n", l.ID, l.Confidence*100, l.Content))
			}
		}
		// Get file intel
		intel, _ := mem.GetFileIntel(contextPath)
		if intel != nil {
			contextData.WriteString(fmt.Sprintf("\n## File History:\n- Edits: %d\n- Failures: %d\n", intel.EditCount, intel.FailureCount))
		}

	case "room":
		contextData.WriteString(fmt.Sprintf("the room '%s'.\n\n", contextPath))
		// Get room-specific knowledge
		learnings, _ := mem.GetRelevantLearnings("", contextPath, 10)
		if len(learnings) > 0 {
			contextData.WriteString("## Learnings in this room:\n")
			for i := range learnings {
				l := &learnings[i]
				contextData.WriteString(fmt.Sprintf("- [%s] (%.0f%% confidence): %s\n", l.ID, l.Confidence*100, l.Content))
			}
		}

	case "task":
		contextData.WriteString(fmt.Sprintf("the task: '%s'.\n\n", contextPath))
		// Search for relevant knowledge
		if embedder := s.butler.GetEmbedder(); embedder != nil {
			opts := memory.DefaultSemanticSearchOptions()
			opts.Limit = 10
			results, _ := mem.SemanticSearch(embedder, contextPath, opts)
			if len(results) > 0 {
				contextData.WriteString("## Relevant knowledge:\n")
				for i := range results {
					r := &results[i]
					contextData.WriteString(fmt.Sprintf("- [%s] %s: %s\n", r.ID, r.Kind, truncateForBriefing(r.Content, 100)))
				}
			}
		}

	default: // workspace
		contextData.WriteString("the entire workspace.\n\n")
		// Get workspace stats
		totalLearnings, _ := mem.CountLearnings()
		totalSessions, _ := mem.CountSessions(false)
		activeSessions, _ := mem.CountSessions(true)
		contextData.WriteString(fmt.Sprintf("## Workspace Overview:\n- Total learnings: %d\n- Total sessions: %d (active: %d)\n\n", totalLearnings, totalSessions, activeSessions))

		// Get recent high-confidence learnings
		learnings, _ := mem.GetRelevantLearnings("", "", 10)
		if len(learnings) > 0 {
			contextData.WriteString("## Key learnings:\n")
			for i := range learnings {
				l := &learnings[i]
				contextData.WriteString(fmt.Sprintf("- [%s] (%.0f%%): %s\n", l.ID, l.Confidence*100, truncateForBriefing(l.Content, 100)))
			}
		}

		// Get pending handoffs
		pendingHandoffs := getPendingHandoffsForAgent("any")
		if len(pendingHandoffs) > 0 {
			contextData.WriteString(fmt.Sprintf("\n## Pending Handoffs: %d\n", len(pendingHandoffs)))
			for _, h := range pendingHandoffs {
				contextData.WriteString(fmt.Sprintf("- [%s] Priority: %s, Task: %s\n", h.ID, h.Priority, truncateForBriefing(h.Task, 80)))
			}
		}

		// Get recent postmortems (last 7 days)
		recentPostmortems, _ := mem.GetPostmortemsSince(time.Now().Add(-7 * 24 * time.Hour))
		if len(recentPostmortems) > 0 {
			contextData.WriteString(fmt.Sprintf("\n## Recent Postmortems: %d in last 7 days\n", len(recentPostmortems)))
			for i := range recentPostmortems {
				if i >= 3 {
					break
				}
				p := &recentPostmortems[i]
				contextData.WriteString(fmt.Sprintf("- [%s] %s: %s\n", p.ID, p.Severity, truncateForBriefing(p.Title, 60)))
			}
		}

		// Get recent decisions
		recentDecisions, _ := mem.GetDecisionsSince(time.Now().Add(-7*24*time.Hour), 5)
		if len(recentDecisions) > 0 {
			contextData.WriteString(fmt.Sprintf("\n## Recent Decisions: %d in last 7 days\n", len(recentDecisions)))
			for i := range recentDecisions {
				d := &recentDecisions[i]
				outcomeIcon := ""
				if d.Outcome != "" {
					outcomeIcon = fmt.Sprintf(" [%s]", d.Outcome)
				}
				contextData.WriteString(fmt.Sprintf("- [%s]%s: %s\n", d.ID, outcomeIcon, truncateForBriefing(d.Content, 80)))
			}
		}
	}

	// Get contradictions
	contradictions, _ := mem.GetContradictionSummary(5)
	if contradictions != nil && contradictions.TotalContradictionLinks > 0 {
		contextData.WriteString(fmt.Sprintf("\n## Contradictions: %d active\n", contradictions.TotalContradictionLinks))
	}

	// Get decay stats
	decayCfg := s.getDecayConfig()
	decayStats, _ := mem.GetDecayStats(decayCfg)
	if decayStats != nil && decayStats.AtRiskCount > 0 {
		contextData.WriteString(fmt.Sprintf("\n## At-risk learnings: %d (approaching decay)\n", decayStats.AtRiskCount))
	}

	// Generate briefing with LLM
	prompt := buildBriefingPrompt(contextData.String(), style)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := llmClient.Complete(ctx, prompt, llm.CompletionOptions{})
	if err != nil {
		return s.generateNonLLMBriefing(id, contextType, contextPath)
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: response}},
		},
	}
}

// generateNonLLMBriefing creates a briefing without LLM (template-based).
func (s *MCPServer) generateNonLLMBriefing(id any, contextType, contextPath string) jsonRPCResponse {
	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not available")
	}

	var output strings.Builder
	output.WriteString("# Smart Briefing\n\n")
	output.WriteString("*Note: LLM not configured. Using template-based briefing.*\n\n")

	switch contextType {
	case "file":
		output.WriteString(fmt.Sprintf("## File: %s\n\n", contextPath))
		learnings, _ := mem.GetFileLearnings(contextPath)
		if len(learnings) > 0 {
			output.WriteString("### Learnings\n\n")
			for i := range learnings {
				l := &learnings[i]
				output.WriteString(fmt.Sprintf("- **%.0f%%** - %s\n", l.Confidence*100, l.Content))
			}
		} else {
			output.WriteString("No learnings recorded for this file yet.\n")
		}

	case "room":
		output.WriteString(fmt.Sprintf("## Room: %s\n\n", contextPath))
		learnings, _ := mem.GetRelevantLearnings("", contextPath, 10)
		if len(learnings) > 0 {
			output.WriteString("### Key Knowledge\n\n")
			for i := range learnings {
				l := &learnings[i]
				output.WriteString(fmt.Sprintf("- **%.0f%%** - %s\n", l.Confidence*100, l.Content))
			}
		}

	default:
		output.WriteString("## Workspace Overview\n\n")
		totalLearnings, _ := mem.CountLearnings()
		totalSessions, _ := mem.CountSessions(false)
		activeSessions, _ := mem.CountSessions(true)

		output.WriteString("### Statistics\n\n")
		output.WriteString(fmt.Sprintf("- **Learnings:** %d\n", totalLearnings))
		output.WriteString(fmt.Sprintf("- **Total sessions:** %d\n", totalSessions))
		output.WriteString(fmt.Sprintf("- **Active sessions:** %d\n", activeSessions))

		learnings, _ := mem.GetRelevantLearnings("", "", 5)
		if len(learnings) > 0 {
			output.WriteString("\n### Top Learnings\n\n")
			for i := range learnings {
				l := &learnings[i]
				output.WriteString(fmt.Sprintf("- **%.0f%%** - %s\n", l.Confidence*100, truncateForBriefing(l.Content, 80)))
			}
		}

		// Pending handoffs
		pendingHandoffs := getPendingHandoffsForAgent("any")
		if len(pendingHandoffs) > 0 {
			output.WriteString("\n### Pending Handoffs\n\n")
			for _, h := range pendingHandoffs {
				priorityIcon := "🔵"
				switch h.Priority {
				case "high":
					priorityIcon = "🟠"
				case "urgent":
					priorityIcon = "🔴"
				}
				output.WriteString(fmt.Sprintf("- %s `%s`: %s\n", priorityIcon, h.ID, truncateForBriefing(h.Task, 60)))
			}
		}

		// Recent postmortems
		recentPostmortems, _ := mem.GetPostmortemsSince(time.Now().Add(-7 * 24 * time.Hour))
		if len(recentPostmortems) > 0 {
			output.WriteString("\n### Recent Postmortems\n\n")
			for i := range recentPostmortems {
				if i >= 3 {
					output.WriteString(fmt.Sprintf("- ... and %d more\n", len(recentPostmortems)-3))
					break
				}
				p := &recentPostmortems[i]
				severityIcon := "⚠️"
				if p.Severity == "critical" {
					severityIcon = "🔴"
				}
				output.WriteString(fmt.Sprintf("- %s `%s`: %s\n", severityIcon, p.ID, truncateForBriefing(p.Title, 60)))
			}
		}

		// Recent decisions
		recentDecisions, _ := mem.GetDecisionsSince(time.Now().Add(-7*24*time.Hour), 5)
		if len(recentDecisions) > 0 {
			output.WriteString("\n### Recent Decisions\n\n")
			for i := range recentDecisions {
				d := &recentDecisions[i]
				outcomeIcon := "📋"
				switch d.Outcome {
				case "successful":
					outcomeIcon = "✅"
				case "failed":
					outcomeIcon = "❌"
				}
				output.WriteString(fmt.Sprintf("- %s `%s`: %s\n", outcomeIcon, d.ID, truncateForBriefing(d.Content, 60)))
			}
		}
	}

	// Add warnings
	decayCfg := s.getDecayConfig()
	decayStats, _ := mem.GetDecayStats(decayCfg)
	contradictions, _ := mem.GetContradictionSummary(5)

	if (decayStats != nil && decayStats.AtRiskCount > 0) || (contradictions != nil && contradictions.TotalContradictionLinks > 0) {
		output.WriteString("\n### Warnings\n\n")
		if decayStats != nil && decayStats.AtRiskCount > 0 {
			output.WriteString(fmt.Sprintf("- **%d learnings** at risk of decay\n", decayStats.AtRiskCount))
		}
		if contradictions != nil && contradictions.TotalContradictionLinks > 0 {
			output.WriteString(fmt.Sprintf("- **%d contradictions** detected\n", contradictions.TotalContradictionLinks))
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

// buildBriefingPrompt constructs the LLM prompt for briefing generation.
func buildBriefingPrompt(contextData, style string) string {
	var prompt strings.Builder
	prompt.WriteString("You are a knowledge assistant for a software project. ")
	prompt.WriteString("Based on the following context, generate a helpful briefing.\n\n")

	switch style {
	case "detailed":
		prompt.WriteString("Style: Detailed - Include comprehensive information, explanations, and context.\n\n")
	case "actionable":
		prompt.WriteString("Style: Actionable - Focus on concrete next steps and recommendations.\n\n")
	default:
		prompt.WriteString("Style: Summary - Be concise and highlight the most important points.\n\n")
	}

	prompt.WriteString("Context:\n")
	prompt.WriteString(contextData)
	prompt.WriteString("\n\nGenerate a markdown-formatted briefing with the following sections:\n")
	prompt.WriteString("1. **Summary** - A brief overview\n")
	prompt.WriteString("2. **Key Points** - The most important things to know\n")
	prompt.WriteString("3. **Warnings** - Any contradictions, stale knowledge, or concerns\n")
	prompt.WriteString("4. **Suggestions** - Recommended actions or next steps\n")

	return prompt.String()
}

// truncateForBriefing truncates text for briefing display.
func truncateForBriefing(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
