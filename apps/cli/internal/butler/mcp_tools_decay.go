package butler

import (
	"fmt"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// getDecayConfig returns the effective decay configuration.
func (s *MCPServer) getDecayConfig() memory.DecayConfig {
	// Start with defaults
	cfg := memory.DefaultDecayConfig()

	// Override from palace config if available
	palaceCfg := s.butler.Config()
	if palaceCfg != nil && palaceCfg.ConfidenceDecay != nil {
		dc := palaceCfg.ConfidenceDecay
		cfg.Enabled = dc.Enabled
		if dc.DecayDays > 0 {
			cfg.DecayDays = dc.DecayDays
		}
		if dc.DecayRate > 0 {
			cfg.DecayRate = dc.DecayRate
		}
		if dc.DecayInterval > 0 {
			cfg.DecayInterval = dc.DecayInterval
		}
		if dc.MinConfidence > 0 {
			cfg.MinConfidence = dc.MinConfidence
		}
	}

	return cfg
}

// toolDecayStats returns statistics about confidence decay.
func (s *MCPServer) toolDecayStats(id any, _ map[string]interface{}) jsonRPCResponse {
	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not available")
	}

	cfg := s.getDecayConfig()
	stats, err := mem.GetDecayStats(cfg)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get decay stats: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Confidence Decay Statistics\n\n")

	if !cfg.Enabled {
		output.WriteString("**Note:** Confidence decay is currently disabled.\n\n")
	}

	output.WriteString("## Configuration\n\n")
	output.WriteString(fmt.Sprintf("- **Enabled:** %v\n", cfg.Enabled))
	output.WriteString(fmt.Sprintf("- **Decay starts after:** %d days of inactivity\n", cfg.DecayDays))
	output.WriteString(fmt.Sprintf("- **Decay rate:** %.0f%% per %d days\n", cfg.DecayRate*100, cfg.DecayInterval))
	output.WriteString(fmt.Sprintf("- **Minimum confidence floor:** %.0f%%\n", cfg.MinConfidence*100))

	output.WriteString("\n## Current State\n\n")
	output.WriteString(fmt.Sprintf("- **Total active learnings:** %d\n", stats.TotalLearnings))
	output.WriteString(fmt.Sprintf("- **At-risk (past threshold):** %d\n", stats.AtRiskCount))
	output.WriteString(fmt.Sprintf("- **Already decayed:** %d\n", stats.DecayedCount))
	output.WriteString(fmt.Sprintf("- **Average confidence:** %.0f%%\n", stats.AverageConfidence*100))

	if stats.OldestInactivedays > 0 {
		output.WriteString(fmt.Sprintf("- **Oldest inactive:** %d days\n", stats.OldestInactivedays))
	}

	if stats.NextDecayEligible > 0 {
		output.WriteString(fmt.Sprintf("\n**%d learnings** would be affected by running `decay_apply`.\n", stats.NextDecayEligible))
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolDecayPreview shows what would happen if decay was applied.
func (s *MCPServer) toolDecayPreview(id any, args map[string]interface{}) jsonRPCResponse {
	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not available")
	}

	limit := 20
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	cfg := s.getDecayConfig()
	result, err := mem.PreviewDecay(cfg, limit)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("preview decay: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Decay Preview\n\n")
	output.WriteString("*This is a preview - no changes have been made.*\n\n")

	if result.TotalAffected == 0 {
		output.WriteString("No learnings would be affected by decay.\n")
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: mcpToolResult{
				Content: []mcpContent{{Type: "text", Text: output.String()}},
			},
		}
	}

	output.WriteString(fmt.Sprintf("## Would Decay (%d learnings)\n\n", result.TotalAffected))
	output.WriteString(fmt.Sprintf("Average confidence drop: **%.1f%%**\n\n", result.AverageDecay*100))

	output.WriteString("| ID | Content | Days Inactive | Confidence |\n")
	output.WriteString("|-----|---------|---------------|------------|\n")
	for _, r := range result.DecayedRecords {
		output.WriteString(fmt.Sprintf("| `%s` | %s | %d | %.0f%% → %.0f%% |\n",
			r.ID, r.Content, r.DaysSinceAccess, r.OldConfidence*100, r.NewConfidence*100))
	}

	if len(result.AtRiskRecords) > 0 {
		output.WriteString(fmt.Sprintf("\n## At Risk (%d learnings approaching threshold)\n\n", len(result.AtRiskRecords)))
		output.WriteString("| ID | Content | Days Inactive | Days Until Decay |\n")
		output.WriteString("|-----|---------|---------------|------------------|\n")
		for _, r := range result.AtRiskRecords {
			output.WriteString(fmt.Sprintf("| `%s` | %s | %d | %d |\n",
				r.ID, r.Content, r.DaysSinceAccess, r.DaysUntilDecay))
		}
	}

	output.WriteString("\n---\n")
	output.WriteString("Use `decay_apply` to apply these changes, or `decay_reinforce` to prevent decay on specific learnings.\n")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolDecayApply applies confidence decay to inactive learnings.
func (s *MCPServer) toolDecayApply(id any, _ map[string]interface{}) jsonRPCResponse {
	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not available")
	}

	cfg := s.getDecayConfig()

	if !cfg.Enabled {
		return s.toolError(id, "confidence decay is disabled in configuration. Enable it in palace.jsonc with confidenceDecay.enabled: true")
	}

	result, err := mem.ApplyDecay(cfg)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("apply decay: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Decay Applied\n\n")

	if result.TotalDecayed == 0 {
		output.WriteString("No learnings were affected by decay.\n")
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: mcpToolResult{
				Content: []mcpContent{{Type: "text", Text: output.String()}},
			},
		}
	}

	output.WriteString(fmt.Sprintf("**%d learnings** updated.\n\n", result.TotalDecayed))
	output.WriteString(fmt.Sprintf("Average confidence drop: **%.1f%%**\n\n", result.AverageDecay*100))

	if len(result.DecayedRecords) > 0 {
		output.WriteString("## Affected Learnings\n\n")
		output.WriteString("| ID | Content | Old → New Confidence |\n")
		output.WriteString("|-----|---------|----------------------|\n")
		for _, r := range result.DecayedRecords {
			output.WriteString(fmt.Sprintf("| `%s` | %s | %.0f%% → %.0f%% |\n",
				r.ID, r.Content, r.OldConfidence*100, r.NewConfidence*100))
		}
	}

	output.WriteString("\n---\n")
	output.WriteString("To prevent future decay, use these learnings in queries or call `decay_reinforce`.\n")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolDecayReinforce reinforces a learning to prevent decay.
func (s *MCPServer) toolDecayReinforce(id any, args map[string]interface{}) jsonRPCResponse {
	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not available")
	}

	learningID, _ := args["learningId"].(string)
	if learningID == "" {
		return s.toolError(id, "learningId is required")
	}

	if err := mem.ReinforceLearning(learningID); err != nil {
		return s.toolError(id, fmt.Sprintf("reinforce learning: %v", err))
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: fmt.Sprintf("Learning `%s` reinforced. Decay timer reset.", learningID)}},
		},
	}
}

// toolDecayBoost boosts a learning's confidence.
func (s *MCPServer) toolDecayBoost(id any, args map[string]interface{}) jsonRPCResponse {
	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not available")
	}

	learningID, _ := args["learningId"].(string)
	if learningID == "" {
		return s.toolError(id, "learningId is required")
	}

	amount := 0.1
	if a, ok := args["amount"].(float64); ok && a > 0 {
		amount = a
	}

	if err := mem.BoostConfidence(learningID, amount, 1.0); err != nil {
		return s.toolError(id, fmt.Sprintf("boost confidence: %v", err))
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: fmt.Sprintf("Learning `%s` confidence boosted by %.0f%%.", learningID, amount*100)}},
		},
	}
}
