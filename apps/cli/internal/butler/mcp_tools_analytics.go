package butler

import (
	"fmt"
	"strings"
	"time"
)

// toolSessionAnalytics provides aggregate statistics about sessions.
func (s *MCPServer) toolSessionAnalytics(id any, args map[string]interface{}) jsonRPCResponse {
	days := 30
	if d, ok := args["days"].(float64); ok && d > 0 {
		days = int(d)
	}

	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not available")
	}

	since := time.Now().Add(-time.Duration(days) * 24 * time.Hour)

	var output strings.Builder
	output.WriteString("# Session Analytics\n\n")
	fmt.Fprintf(&output, "**Period:** Last %d days\n\n", days)

	// Get all sessions in the period
	sessions, err := mem.ListSessions(false, 1000)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("list sessions failed: %v", err))
	}

	// Filter to period and calculate stats
	totalSessions := 0
	completedSessions := 0
	abandonedSessions := 0
	activeSessions := 0
	agentCounts := make(map[string]int)
	var totalDuration time.Duration
	successfulOutcomes := 0
	failedOutcomes := 0

	for i := range sessions {
		sess := &sessions[i]
		if sess.StartedAt.Before(since) {
			continue
		}
		totalSessions++
		agentCounts[sess.AgentType]++

		switch sess.State {
		case "completed":
			completedSessions++
			if !sess.LastActivity.IsZero() {
				totalDuration += sess.LastActivity.Sub(sess.StartedAt)
			}
		case "abandoned":
			abandonedSessions++
		case "active":
			activeSessions++
		}
	}

	// Count activity outcomes
	for i := range sessions {
		sess := &sessions[i]
		if sess.StartedAt.Before(since) {
			continue
		}
		activities, _ := mem.GetActivities(sess.ID, "", 100)
		for j := range activities {
			switch activities[j].Outcome {
			case "success":
				successfulOutcomes++
			case "failure":
				failedOutcomes++
			}
		}
	}

	output.WriteString("## Session Overview\n\n")
	fmt.Fprintf(&output, "- **Total Sessions:** %d\n", totalSessions)
	fmt.Fprintf(&output, "- **Completed:** %d (%.1f%%)\n", completedSessions, safePercent(completedSessions, totalSessions))
	fmt.Fprintf(&output, "- **Abandoned:** %d (%.1f%%)\n", abandonedSessions, safePercent(abandonedSessions, totalSessions))
	fmt.Fprintf(&output, "- **Active:** %d\n", activeSessions)
	output.WriteString("\n")

	if completedSessions > 0 {
		avgDuration := totalDuration / time.Duration(completedSessions)
		fmt.Fprintf(&output, "**Average Session Duration:** %s\n\n", formatDuration(avgDuration))
	}

	output.WriteString("## By Agent Type\n\n")
	for agent, count := range agentCounts {
		fmt.Fprintf(&output, "- **%s:** %d sessions (%.1f%%)\n", agent, count, safePercent(count, totalSessions))
	}
	output.WriteString("\n")

	totalOutcomes := successfulOutcomes + failedOutcomes
	if totalOutcomes > 0 {
		output.WriteString("## Activity Outcomes\n\n")
		fmt.Fprintf(&output, "- **Successful:** %d (%.1f%%)\n", successfulOutcomes, safePercent(successfulOutcomes, totalOutcomes))
		fmt.Fprintf(&output, "- **Failed:** %d (%.1f%%)\n", failedOutcomes, safePercent(failedOutcomes, totalOutcomes))
		output.WriteString("\n")
	}

	// Handoff stats
	handoffMu.RLock()
	pendingHandoffs := 0
	completedHandoffs := 0
	for _, h := range handoffStore {
		switch h.Status {
		case "pending":
			pendingHandoffs++
		case "completed":
			completedHandoffs++
		}
	}
	handoffMu.RUnlock()

	if pendingHandoffs > 0 || completedHandoffs > 0 {
		output.WriteString("## Handoffs\n\n")
		fmt.Fprintf(&output, "- **Pending:** %d\n", pendingHandoffs)
		fmt.Fprintf(&output, "- **Completed:** %d\n", completedHandoffs)
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

// toolLearningEffectiveness tracks which learnings are being used and their impact.
func (s *MCPServer) toolLearningEffectiveness(id any, args map[string]interface{}) jsonRPCResponse {
	limit := 20
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	sortBy, _ := args["sort"].(string)
	if sortBy == "" {
		sortBy = "use_count"
	}

	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not available")
	}

	var output strings.Builder
	output.WriteString("# Learning Effectiveness\n\n")

	// Get learnings sorted by use count
	learnings, err := mem.GetLearningsByEffectiveness(limit, sortBy)
	if err != nil {
		// Fall back to regular learnings if effectiveness query fails
		learnings, err = mem.GetRelevantLearnings("", "", limit)
		if err != nil {
			return s.toolError(id, fmt.Sprintf("get learnings failed: %v", err))
		}
	}

	if len(learnings) == 0 {
		output.WriteString("No learnings found.\n")
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: mcpToolResult{
				Content: []mcpContent{{Type: "text", Text: output.String()}},
			},
		}
	}

	// Calculate stats
	totalUseCount := 0
	highConfidenceCount := 0
	lowConfidenceCount := 0
	recentlyUsedCount := 0
	oneWeekAgo := time.Now().Add(-7 * 24 * time.Hour)

	for i := range learnings {
		l := &learnings[i]
		totalUseCount += l.UseCount
		if l.Confidence >= 0.8 {
			highConfidenceCount++
		} else if l.Confidence < 0.5 {
			lowConfidenceCount++
		}
		if l.LastUsed.After(oneWeekAgo) {
			recentlyUsedCount++
		}
	}

	output.WriteString("## Overview\n\n")
	fmt.Fprintf(&output, "- **Total Learnings:** %d\n", len(learnings))
	fmt.Fprintf(&output, "- **Total Uses:** %d\n", totalUseCount)
	fmt.Fprintf(&output, "- **Recently Used (7 days):** %d\n", recentlyUsedCount)
	fmt.Fprintf(&output, "- **High Confidence (>=80%%):** %d\n", highConfidenceCount)
	fmt.Fprintf(&output, "- **Low Confidence (<50%%):** %d\n", lowConfidenceCount)
	output.WriteString("\n")

	output.WriteString("## Most Effective Learnings\n\n")
	for i := range learnings {
		if i >= 10 {
			break
		}
		l := &learnings[i]
		effectivenessIcon := "📊"
		if l.UseCount >= 5 && l.Confidence >= 0.8 {
			effectivenessIcon = "⭐"
		} else if l.UseCount == 0 {
			effectivenessIcon = "💤"
		}
		fmt.Fprintf(&output, "- %s `%s` - Used: %d | Confidence: %.0f%%\n",
			effectivenessIcon, l.ID, l.UseCount, l.Confidence*100)
		fmt.Fprintf(&output, "  %s\n", truncateString(l.Content, 70))
	}

	// Unused learnings warning
	unusedCount := 0
	for i := range learnings {
		if learnings[i].UseCount == 0 {
			unusedCount++
		}
	}
	if unusedCount > 0 {
		output.WriteString("\n## Suggestions\n\n")
		fmt.Fprintf(&output, "- **%d learnings** have never been used - consider reviewing their relevance\n", unusedCount)
		if lowConfidenceCount > 0 {
			fmt.Fprintf(&output, "- **%d learnings** have low confidence - consider reinforcing or archiving\n", lowConfidenceCount)
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

// toolWorkspaceHealth provides overall health metrics for the workspace.
func (s *MCPServer) toolWorkspaceHealth(id any, _ map[string]interface{}) jsonRPCResponse {
	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not available")
	}

	var output strings.Builder
	output.WriteString("# Workspace Health Dashboard\n\n")

	// Knowledge health
	output.WriteString("## Knowledge Health\n\n")

	totalLearnings, _ := mem.CountLearnings()
	fmt.Fprintf(&output, "- **Total Learnings:** %d\n", totalLearnings)

	// Get decay stats
	decayCfg := s.getDecayConfig()
	decayStats, _ := mem.GetDecayStats(decayCfg)
	if decayStats != nil {
		healthyCount := totalLearnings - decayStats.AtRiskCount - decayStats.DecayedCount
		healthyPercent := safePercent(healthyCount, totalLearnings)

		healthIcon := "🟢"
		if healthyPercent < 50 {
			healthIcon = "🔴"
		} else if healthyPercent < 75 {
			healthIcon = "🟡"
		}

		fmt.Fprintf(&output, "- %s **Healthy:** %d (%.1f%%)\n", healthIcon, healthyCount, healthyPercent)
		if decayStats.AtRiskCount > 0 {
			fmt.Fprintf(&output, "- ⚠️ **At Risk:** %d\n", decayStats.AtRiskCount)
		}
		if decayStats.DecayedCount > 0 {
			fmt.Fprintf(&output, "- 📉 **Decayed:** %d\n", decayStats.DecayedCount)
		}
	}
	output.WriteString("\n")

	// Contradiction health
	contradictions, _ := mem.GetContradictionSummary(10)
	if contradictions != nil {
		output.WriteString("## Contradiction Health\n\n")
		if contradictions.TotalContradictionLinks == 0 {
			output.WriteString("- 🟢 **No contradictions detected**\n")
		} else {
			output.WriteString(fmt.Sprintf("- ⚠️ **%d contradictions** need attention\n", contradictions.TotalContradictionLinks))
		}
		output.WriteString("\n")
	}

	// Session health
	output.WriteString("## Session Health\n\n")
	totalSessions, _ := mem.CountSessions(false)
	activeSessions, _ := mem.CountSessions(true)
	fmt.Fprintf(&output, "- **Total Sessions:** %d\n", totalSessions)
	fmt.Fprintf(&output, "- **Active Sessions:** %d\n", activeSessions)

	// Recent session success rate
	recentSessions, _ := mem.ListSessions(false, 20)
	completedRecent := 0
	abandonedRecent := 0
	for i := range recentSessions {
		switch recentSessions[i].State {
		case "completed":
			completedRecent++
		case "abandoned":
			abandonedRecent++
		}
	}
	if completedRecent+abandonedRecent > 0 {
		successRate := safePercent(completedRecent, completedRecent+abandonedRecent)
		successIcon := "🟢"
		if successRate < 50 {
			successIcon = "🔴"
		} else if successRate < 75 {
			successIcon = "🟡"
		}
		fmt.Fprintf(&output, "- %s **Recent Success Rate:** %.1f%%\n", successIcon, successRate)
	}
	output.WriteString("\n")

	// Postmortem health
	recentPostmortems, _ := mem.GetPostmortemsSince(time.Now().Add(-30 * 24 * time.Hour))
	output.WriteString("## Failure Tracking\n\n")
	if len(recentPostmortems) == 0 {
		output.WriteString("- 🟢 **No postmortems in last 30 days**\n")
	} else {
		criticalCount := 0
		unresolvedCount := 0
		for i := range recentPostmortems {
			if recentPostmortems[i].Severity == "critical" {
				criticalCount++
			}
			if recentPostmortems[i].Status != "resolved" {
				unresolvedCount++
			}
		}
		fmt.Fprintf(&output, "- **Postmortems (30 days):** %d\n", len(recentPostmortems))
		if criticalCount > 0 {
			fmt.Fprintf(&output, "- 🔴 **Critical:** %d\n", criticalCount)
		}
		if unresolvedCount > 0 {
			fmt.Fprintf(&output, "- ⚠️ **Unresolved:** %d\n", unresolvedCount)
		}
	}
	output.WriteString("\n")

	// Handoff health
	handoffMu.RLock()
	pendingHandoffs := 0
	urgentHandoffs := 0
	expiredHandoffs := 0
	now := time.Now()
	for _, h := range handoffStore {
		if h.Status == "pending" {
			if h.ExpiresAt.Before(now) {
				expiredHandoffs++
			} else {
				pendingHandoffs++
				if h.Priority == "urgent" {
					urgentHandoffs++
				}
			}
		}
	}
	handoffMu.RUnlock()

	output.WriteString("## Handoff Health\n\n")
	if pendingHandoffs == 0 && expiredHandoffs == 0 {
		output.WriteString("- 🟢 **No pending handoffs**\n")
	} else {
		fmt.Fprintf(&output, "- **Pending:** %d\n", pendingHandoffs)
		if urgentHandoffs > 0 {
			fmt.Fprintf(&output, "- 🔴 **Urgent:** %d\n", urgentHandoffs)
		}
		if expiredHandoffs > 0 {
			fmt.Fprintf(&output, "- ⚠️ **Expired:** %d\n", expiredHandoffs)
		}
	}
	output.WriteString("\n")

	// Overall health score
	output.WriteString("---\n\n")
	output.WriteString("## Overall Health Score\n\n")

	healthScore := 100
	issues := []string{}

	if decayStats != nil && decayStats.AtRiskCount > totalLearnings/4 {
		healthScore -= 15
		issues = append(issues, "Many learnings at risk of decay")
	}
	if contradictions != nil && contradictions.TotalContradictionLinks > 5 {
		healthScore -= 10
		issues = append(issues, "Multiple contradictions detected")
	}
	if len(recentPostmortems) > 3 {
		healthScore -= 10
		issues = append(issues, "Multiple recent failures")
	}
	if urgentHandoffs > 0 {
		healthScore -= 10
		issues = append(issues, "Urgent handoffs waiting")
	}
	if expiredHandoffs > 0 {
		healthScore -= 5
		issues = append(issues, "Expired handoffs need cleanup")
	}

	scoreIcon := "🟢"
	if healthScore < 50 {
		scoreIcon = "🔴"
	} else if healthScore < 75 {
		scoreIcon = "🟡"
	}

	fmt.Fprintf(&output, "%s **Health Score: %d/100**\n\n", scoreIcon, healthScore)

	if len(issues) > 0 {
		output.WriteString("**Issues:**\n")
		for _, issue := range issues {
			fmt.Fprintf(&output, "- %s\n", issue)
		}
	} else {
		output.WriteString("**Status:** All systems healthy!\n")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// safePercent calculates percentage safely (handles division by zero).
func safePercent(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}
