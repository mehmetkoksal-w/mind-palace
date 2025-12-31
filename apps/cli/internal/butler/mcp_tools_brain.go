package butler

import (
	"fmt"
	"strings"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// toolStore stores a thought with auto-classification.
func (s *MCPServer) toolStore(id any, args map[string]interface{}) jsonRPCResponse {
	content, _ := args["content"].(string)
	if content == "" {
		return s.toolError(id, "content is required")
	}

	// Support both "kind" and "as" for backwards compatibility
	kindStr, _ := args["as"].(string)
	if kindStr == "" {
		kindStr, _ = args["kind"].(string)
	}
	scope, _ := args["scope"].(string)
	if scope == "" {
		scope = "palace"
	}
	scopePath, _ := args["scopePath"].(string)

	// Parse tags from array
	var tags []string
	if tagsRaw, ok := args["tags"].([]interface{}); ok {
		for _, t := range tagsRaw {
			if tag, ok := t.(string); ok && tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	// Determine kind
	var kind memory.RecordKind
	var classification memory.Classification

	if kindStr != "" {
		kind = memory.RecordKind(kindStr)
		classification = memory.Classification{Kind: kind, Confidence: 1.0, Signals: []string{"explicit"}}
	} else {
		// Auto-classify
		classification = memory.Classify(content)
		kind = classification.Kind
	}

	// Extract additional tags from content
	extractedTags := memory.ExtractTags(content)
	tags = append(tags, extractedTags...)

	// Store based on kind
	var recordID string
	var err error

	switch kind {
	case memory.RecordKindIdea:
		idea := memory.Idea{
			Content:   content,
			Scope:     scope,
			ScopePath: scopePath,
			Source:    "agent",
		}
		recordID, err = s.butler.AddIdea(idea)
	case memory.RecordKindDecision:
		dec := memory.Decision{
			Content:   content,
			Scope:     scope,
			ScopePath: scopePath,
			Source:    "agent",
		}
		recordID, err = s.butler.AddDecision(dec)
	case memory.RecordKindLearning:
		learning := memory.Learning{
			Content:    content,
			Scope:      scope,
			ScopePath:  scopePath,
			Source:     "agent",
			Confidence: 0.5,
		}
		recordID, err = s.butler.AddLearning(learning)
	}

	if err != nil {
		return s.toolError(id, fmt.Sprintf("store %s failed: %v", kind, err))
	}

	// Set tags if any
	if len(tags) > 0 {
		s.butler.SetTags(recordID, string(kind), tags)
	}

	var output strings.Builder
	output.WriteString("# Thought Remembered\n\n")
	fmt.Fprintf(&output, "**ID:** `%s`\n", recordID)
	fmt.Fprintf(&output, "**Type:** %s\n", kind)
	fmt.Fprintf(&output, "**Confidence:** %.0f%%\n", classification.Confidence*100)
	if len(classification.Signals) > 0 {
		fmt.Fprintf(&output, "**Signals:** %v\n", classification.Signals)
	}
	fmt.Fprintf(&output, "**Scope:** %s", scope)
	if scopePath != "" {
		fmt.Fprintf(&output, " (%s)", scopePath)
	}
	output.WriteString("\n")
	if len(tags) > 0 {
		fmt.Fprintf(&output, "**Tags:** %s\n", strings.Join(tags, ", "))
	}
	fmt.Fprintf(&output, "\n**Content:** %s\n", content)

	// Auto-check for contradictions if enabled
	var contradictions []memory.ContradictionResult
	cfg := s.butler.Config()
	if cfg != nil && cfg.ContradictionAutoCheck {
		if llmClient, err := s.butler.GetLLMClient(); err == nil && llmClient != nil {
			analyzer := memory.NewLLMContradictionAnalyzer(llmClient)
			embedder := s.butler.GetEmbedder()

			minConfidence := cfg.ContradictionMinConfidence
			if minConfidence <= 0 {
				minConfidence = 0.8
			}
			autoLink := cfg.ContradictionAutoLink

			mem := s.butler.Memory()
			if mem != nil {
				contradictions, _ = mem.AutoCheckContradictions(
					recordID, string(kind), content,
					analyzer, embedder, autoLink, minConfidence,
				)
			}
		}
	}

	// Add contradiction warnings to output
	if len(contradictions) > 0 {
		output.WriteString("\n---\n\n")
		output.WriteString("## Contradictions Detected\n\n")
		for i, c := range contradictions {
			fmt.Fprintf(&output, "### %d. `%s` (%.0f%% confidence)\n\n", i+1, c.Record2ID, c.Confidence*100)
			fmt.Fprintf(&output, "**Type:** %s\n", c.ContradictType)
			fmt.Fprintf(&output, "**Explanation:** %s\n\n", c.Explanation)
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

// toolRecallDecisions retrieves decisions from the brain.
func (s *MCPServer) toolRecallDecisions(id any, args map[string]interface{}) jsonRPCResponse {
	query, _ := args["query"].(string)
	status, _ := args["status"].(string)
	scope, _ := args["scope"].(string)
	scopePath, _ := args["scopePath"].(string)

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	var decisions []memory.Decision
	var err error

	if query != "" {
		decisions, err = s.butler.SearchDecisions(query, limit)
	} else {
		decisions, err = s.butler.GetDecisions(status, scope, scopePath, limit)
	}

	if err != nil {
		return s.toolError(id, fmt.Sprintf("get decisions failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Decisions\n\n")

	if len(decisions) == 0 {
		output.WriteString("No decisions found.\n")
	} else {
		for _, d := range decisions {
			statusIcon := "🔵"
			switch d.Status {
			case memory.DecisionStatusSuperseded:
				statusIcon = "🔄"
			case memory.DecisionStatusReversed:
				statusIcon = "↩️"
			}

			outcomeIcon := "❓"
			switch d.Outcome {
			case memory.DecisionOutcomeSuccessful:
				outcomeIcon = "✅"
			case memory.DecisionOutcomeFailed:
				outcomeIcon = "❌"
			case memory.DecisionOutcomeMixed:
				outcomeIcon = "⚖️"
			}

			fmt.Fprintf(&output, "## %s `%s` %s\n\n", statusIcon, d.ID, outcomeIcon)
			fmt.Fprintf(&output, "**Status:** %s | **Outcome:** %s\n", d.Status, d.Outcome)
			scopeInfo := d.Scope
			if d.ScopePath != "" {
				scopeInfo = fmt.Sprintf("%s:%s", d.Scope, d.ScopePath)
			}
			fmt.Fprintf(&output, "**Scope:** %s\n", scopeInfo)
			fmt.Fprintf(&output, "**Content:** %s\n", d.Content)
			if d.Rationale != "" {
				fmt.Fprintf(&output, "**Rationale:** %s\n", d.Rationale)
			}
			fmt.Fprintf(&output, "**Created:** %s\n\n", d.CreatedAt.Format(time.RFC3339))
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

// toolRecallIdeas retrieves ideas from the brain.
func (s *MCPServer) toolRecallIdeas(id any, args map[string]interface{}) jsonRPCResponse {
	query, _ := args["query"].(string)
	status, _ := args["status"].(string)
	scope, _ := args["scope"].(string)
	scopePath, _ := args["scopePath"].(string)

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	var ideas []memory.Idea
	var err error

	if query != "" {
		ideas, err = s.butler.SearchIdeas(query, limit)
	} else {
		ideas, err = s.butler.GetIdeas(status, scope, scopePath, limit)
	}

	if err != nil {
		return s.toolError(id, fmt.Sprintf("get ideas failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Ideas\n\n")

	if len(ideas) == 0 {
		output.WriteString("No ideas found.\n")
	} else {
		for _, i := range ideas {
			statusIcon := "💡"
			switch i.Status {
			case memory.IdeaStatusExploring:
				statusIcon = "🔍"
			case memory.IdeaStatusImplemented:
				statusIcon = "✅"
			case memory.IdeaStatusDropped:
				statusIcon = "❌"
			}

			fmt.Fprintf(&output, "## %s `%s` (%s)\n\n", statusIcon, i.ID, i.Status)
			scopeInfo := i.Scope
			if i.ScopePath != "" {
				scopeInfo = fmt.Sprintf("%s:%s", i.Scope, i.ScopePath)
			}
			fmt.Fprintf(&output, "**Scope:** %s\n", scopeInfo)
			fmt.Fprintf(&output, "**Content:** %s\n", i.Content)
			if i.Context != "" {
				fmt.Fprintf(&output, "**Context:** %s\n", i.Context)
			}
			fmt.Fprintf(&output, "**Created:** %s\n\n", i.CreatedAt.Format(time.RFC3339))
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

// toolRecallOutcome records the outcome of a decision.
func (s *MCPServer) toolRecallOutcome(id any, args map[string]interface{}) jsonRPCResponse {
	decisionID, _ := args["decisionId"].(string)
	if decisionID == "" {
		return s.toolError(id, "decisionId is required")
	}

	outcome, _ := args["outcome"].(string)
	if outcome == "" {
		return s.toolError(id, "outcome is required")
	}

	// Validate outcome
	validOutcomes := map[string]bool{"successful": true, "failed": true, "mixed": true}
	if !validOutcomes[outcome] {
		return s.toolError(id, "outcome must be 'successful', 'failed', or 'mixed'")
	}

	note, _ := args["note"].(string)

	if err := s.butler.RecordDecisionOutcome(decisionID, outcome, note); err != nil {
		return s.toolError(id, fmt.Sprintf("record outcome failed: %v", err))
	}

	outcomeIcon := "✅"
	switch outcome {
	case "failed":
		outcomeIcon = "❌"
	case "mixed":
		outcomeIcon = "⚖️"
	}

	var output strings.Builder
	output.WriteString("# Outcome Recorded\n\n")
	fmt.Fprintf(&output, "%s **Decision:** `%s`\n", outcomeIcon, decisionID)
	fmt.Fprintf(&output, "**Outcome:** %s\n", outcome)
	if note != "" {
		fmt.Fprintf(&output, "**Note:** %s\n", note)
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolRecallLink creates a relationship between records.
func (s *MCPServer) toolRecallLink(id any, args map[string]interface{}) jsonRPCResponse {
	sourceID, _ := args["sourceId"].(string)
	if sourceID == "" {
		return s.toolError(id, "sourceId is required")
	}

	targetID, _ := args["targetId"].(string)
	if targetID == "" {
		return s.toolError(id, "targetId is required")
	}

	relation, _ := args["relation"].(string)
	if relation == "" {
		return s.toolError(id, "relation is required")
	}

	// Validate relation
	validRelations := map[string]bool{
		memory.RelationSupports:    true,
		memory.RelationContradicts: true,
		memory.RelationImplements:  true,
		memory.RelationSupersedes:  true,
		memory.RelationInspiredBy:  true,
		memory.RelationRelated:     true,
	}
	if !validRelations[relation] {
		return s.toolError(id, fmt.Sprintf("invalid relation %q; valid: supports, contradicts, implements, supersedes, inspired_by, related", relation))
	}

	// Infer kinds from IDs
	sourceKind := inferKindFromID(sourceID)
	targetKind := inferKindFromID(targetID)

	link := memory.Link{
		SourceID:   sourceID,
		SourceKind: sourceKind,
		TargetID:   targetID,
		TargetKind: targetKind,
		Relation:   relation,
	}

	// For code links, validate and get mtime
	if targetKind == memory.TargetKindCode && s.butler.root != "" {
		_, mtime, err := memory.ValidateCodeTarget(s.butler.root, targetID)
		if err != nil {
			return s.toolError(id, fmt.Sprintf("invalid code target: %v", err))
		}
		link.TargetMtime = mtime
	}

	linkID, err := s.butler.AddLink(link)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("add link failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Link Created\n\n")
	fmt.Fprintf(&output, "**ID:** `%s`\n", linkID)
	fmt.Fprintf(&output, "**Source:** `%s` (%s)\n", sourceID, sourceKind)
	fmt.Fprintf(&output, "**Relation:** %s\n", relation)
	fmt.Fprintf(&output, "**Target:** `%s` (%s)\n", targetID, targetKind)

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolRecallLinks gets all links for a record.
func (s *MCPServer) toolRecallLinks(id any, args map[string]interface{}) jsonRPCResponse {
	recordID, _ := args["recordId"].(string)
	if recordID == "" {
		return s.toolError(id, "recordId is required")
	}

	direction, _ := args["direction"].(string)
	if direction == "" {
		direction = "all"
	}

	var links []memory.Link
	var err error

	switch direction {
	case "from":
		links, err = s.butler.memory.GetLinksForSource(recordID)
	case "to":
		links, err = s.butler.memory.GetLinksForTarget(recordID)
	default:
		links, err = s.butler.GetLinksForRecord(recordID)
	}

	if err != nil {
		return s.toolError(id, fmt.Sprintf("get links failed: %v", err))
	}

	var output strings.Builder
	fmt.Fprintf(&output, "# Links for `%s` (%d found)\n\n", recordID, len(links))

	if len(links) == 0 {
		output.WriteString("No links found.\n")
	} else {
		for _, l := range links {
			direction := "→"
			other := l.TargetID
			if l.TargetID == recordID {
				direction = "←"
				other = l.SourceID
			}
			staleIndicator := ""
			if l.IsStale {
				staleIndicator = " ⚠️ (stale)"
			}
			fmt.Fprintf(&output, "- `%s` %s `%s` (%s)%s\n", l.ID, direction, other, l.Relation, staleIndicator)
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

// toolRecallUnlink deletes a link by its ID.
func (s *MCPServer) toolRecallUnlink(id any, args map[string]interface{}) jsonRPCResponse {
	linkID, _ := args["linkId"].(string)
	if linkID == "" {
		return s.toolError(id, "linkId is required")
	}

	if err := s.butler.DeleteLink(linkID); err != nil {
		return s.toolError(id, fmt.Sprintf("delete link failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Link Deleted\n\n")
	fmt.Fprintf(&output, "Successfully deleted link `%s`\n", linkID)

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// ============================================================================
// Learning Lifecycle Tools
// ============================================================================

// toolRecallLearningLink links a learning to a decision for outcome feedback.
func (s *MCPServer) toolRecallLearningLink(id any, args map[string]interface{}) jsonRPCResponse {
	decisionID, _ := args["decisionId"].(string)
	if decisionID == "" {
		return s.toolError(id, "decisionId is required")
	}

	learningID, _ := args["learningId"].(string)
	if learningID == "" {
		return s.toolError(id, "learningId is required")
	}

	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	if err := mem.LinkLearningToDecision(decisionID, learningID); err != nil {
		return s.toolError(id, fmt.Sprintf("link failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Learning Linked to Decision\n\n")
	fmt.Fprintf(&output, "**Learning:** `%s`\n", learningID)
	fmt.Fprintf(&output, "**Decision:** `%s`\n\n", decisionID)
	output.WriteString("When the decision's outcome is recorded, the learning's confidence will be updated:\n")
	output.WriteString("- Successful outcome → +0.1 confidence\n")
	output.WriteString("- Failed outcome → -0.1 confidence\n")
	output.WriteString("- Mixed outcome → no change\n")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolRecallObsolete marks a learning as obsolete.
func (s *MCPServer) toolRecallObsolete(id any, args map[string]interface{}) jsonRPCResponse {
	learningID, _ := args["learningId"].(string)
	if learningID == "" {
		return s.toolError(id, "learningId is required")
	}

	reason, _ := args["reason"].(string)
	if reason == "" {
		return s.toolError(id, "reason is required")
	}

	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	if err := mem.MarkLearningObsolete(learningID, reason); err != nil {
		return s.toolError(id, fmt.Sprintf("mark obsolete failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Learning Marked Obsolete\n\n")
	fmt.Fprintf(&output, "**Learning:** `%s`\n", learningID)
	fmt.Fprintf(&output, "**Reason:** %s\n\n", reason)
	output.WriteString("The learning is now marked as obsolete and will not appear in active queries.\n")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolRecallArchive archives old, low-confidence learnings.
func (s *MCPServer) toolRecallArchive(id any, args map[string]interface{}) jsonRPCResponse {
	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	unusedDays := 90
	if days, ok := args["unusedDays"].(float64); ok && days > 0 {
		unusedDays = int(days)
	}

	maxConfidence := 0.3
	if conf, ok := args["maxConfidence"].(float64); ok && conf > 0 {
		maxConfidence = conf
	}

	archived, err := mem.ArchiveOldLearnings(unusedDays, maxConfidence)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("archive failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Archive Complete\n\n")
	fmt.Fprintf(&output, "**Archived:** %d learnings\n", archived)
	fmt.Fprintf(&output, "**Criteria:**\n")
	fmt.Fprintf(&output, "- Unused for: %d+ days\n", unusedDays)
	fmt.Fprintf(&output, "- Confidence: ≤%.1f%%\n\n", maxConfidence*100)

	if archived == 0 {
		output.WriteString("No learnings matched the archival criteria.\n")
	} else {
		output.WriteString("Archived learnings are preserved but hidden from active queries.\n")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolRecallLearningsByStatus retrieves learnings by lifecycle status.
func (s *MCPServer) toolRecallLearningsByStatus(id any, args map[string]interface{}) jsonRPCResponse {
	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	status, _ := args["status"].(string)
	if status == "" {
		status = "active" // Default to active
	}

	limit := 20
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	learnings, err := mem.GetLearningsByStatus(status, limit)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("query failed: %v", err))
	}

	var output strings.Builder
	statusTitle := strings.Title(status)
	output.WriteString(fmt.Sprintf("# %s Learnings\n\n", statusTitle))
	fmt.Fprintf(&output, "Found %d learnings with status '%s'\n\n", len(learnings), status)

	for i, l := range learnings {
		confidenceIcon := "🟢"
		if l.Confidence < 0.3 {
			confidenceIcon = "🔴"
		} else if l.Confidence < 0.6 {
			confidenceIcon = "🟡"
		}

		fmt.Fprintf(&output, "## %d. `%s` %s %.0f%%\n\n", i+1, l.ID, confidenceIcon, l.Confidence*100)
		fmt.Fprintf(&output, "**Content:** %s\n", l.Content)
		fmt.Fprintf(&output, "**Scope:** %s", l.Scope)
		if l.ScopePath != "" {
			fmt.Fprintf(&output, ":%s", l.ScopePath)
		}
		output.WriteString("\n")
		fmt.Fprintf(&output, "**Last Used:** %s\n", l.LastUsed.Format(time.RFC3339))
		fmt.Fprintf(&output, "**Use Count:** %d\n\n", l.UseCount)
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// ============================================================================
// Contradiction Detection Tools
// ============================================================================

// toolRecallContradictions finds records that contradict a given record using semantic analysis.
func (s *MCPServer) toolRecallContradictions(id any, args map[string]interface{}) jsonRPCResponse {
	recordID, _ := args["recordId"].(string)
	if recordID == "" {
		return s.toolError(id, "recordId is required")
	}

	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	// Get LLM client for analysis
	llmClient, err := s.butler.GetLLMClient()
	if err != nil || llmClient == nil {
		return s.toolError(id, "LLM not configured - set llmBackend in palace.jsonc")
	}

	minConfidence := 0.7
	if mc, ok := args["minConfidence"].(float64); ok && mc > 0 {
		minConfidence = mc
	}

	autoLink := true
	if al, ok := args["autoLink"].(bool); ok {
		autoLink = al
	}

	// Get the source record
	kind := inferKindFromID(recordID)
	record, err := mem.GetRecordForAnalysis(recordID, kind)
	if err != nil || record == nil {
		return s.toolError(id, fmt.Sprintf("record not found: %s", recordID))
	}

	// Create analyzer
	analyzer := memory.NewLLMContradictionAnalyzer(llmClient)
	embedder := s.butler.GetEmbedder()

	// Find contradictions
	contradictions, err := mem.AutoCheckContradictions(
		recordID, kind, record.Content,
		analyzer, embedder, autoLink, minConfidence,
	)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("analysis failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Contradiction Analysis\n\n")
	fmt.Fprintf(&output, "**Record:** `%s` (%s)\n", recordID, kind)
	fmt.Fprintf(&output, "**Content:** %s\n\n", record.Content)

	if len(contradictions) == 0 {
		output.WriteString("No contradictions found. ✅\n")
	} else {
		fmt.Fprintf(&output, "## Found %d Contradiction(s)\n\n", len(contradictions))
		for i, c := range contradictions {
			confidenceIcon := "🔴"
			if c.Confidence >= 0.8 {
				confidenceIcon = "⚠️"
			} else if c.Confidence >= 0.6 {
				confidenceIcon = "🟡"
			}

			fmt.Fprintf(&output, "### %d. `%s` %s %.0f%%\n\n", i+1, c.Record2ID, confidenceIcon, c.Confidence*100)
			fmt.Fprintf(&output, "**Type:** %s\n", c.ContradictType)
			fmt.Fprintf(&output, "**Explanation:** %s\n", c.Explanation)
			if autoLink && c.Confidence >= minConfidence {
				output.WriteString("**Status:** Auto-linked as contradiction\n")
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

// toolRecallContradictionCheck checks if two specific records contradict each other.
func (s *MCPServer) toolRecallContradictionCheck(id any, args map[string]interface{}) jsonRPCResponse {
	record1ID, _ := args["record1Id"].(string)
	if record1ID == "" {
		return s.toolError(id, "record1Id is required")
	}

	record2ID, _ := args["record2Id"].(string)
	if record2ID == "" {
		return s.toolError(id, "record2Id is required")
	}

	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	// Get LLM client for analysis
	llmClient, err := s.butler.GetLLMClient()
	if err != nil || llmClient == nil {
		return s.toolError(id, "LLM not configured - set llmBackend in palace.jsonc")
	}

	// Get both records
	kind1 := inferKindFromID(record1ID)
	kind2 := inferKindFromID(record2ID)

	record1, err := mem.GetRecordForAnalysis(record1ID, kind1)
	if err != nil || record1 == nil {
		return s.toolError(id, fmt.Sprintf("record not found: %s", record1ID))
	}

	record2, err := mem.GetRecordForAnalysis(record2ID, kind2)
	if err != nil || record2 == nil {
		return s.toolError(id, fmt.Sprintf("record not found: %s", record2ID))
	}

	// Analyze
	analyzer := memory.NewLLMContradictionAnalyzer(llmClient)
	result, err := analyzer.AnalyzeContradiction(*record1, *record2)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("analysis failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Contradiction Check\n\n")

	fmt.Fprintf(&output, "## Record 1: `%s`\n", record1ID)
	fmt.Fprintf(&output, "**Kind:** %s\n", kind1)
	fmt.Fprintf(&output, "**Content:** %s\n\n", record1.Content)

	fmt.Fprintf(&output, "## Record 2: `%s`\n", record2ID)
	fmt.Fprintf(&output, "**Kind:** %s\n", kind2)
	fmt.Fprintf(&output, "**Content:** %s\n\n", record2.Content)

	output.WriteString("## Analysis Result\n\n")
	if result.IsContradiction {
		fmt.Fprintf(&output, "⚠️ **CONTRADICTION DETECTED**\n\n")
		fmt.Fprintf(&output, "**Type:** %s\n", result.ContradictType)
		fmt.Fprintf(&output, "**Confidence:** %.0f%%\n", result.Confidence*100)
		fmt.Fprintf(&output, "**Explanation:** %s\n", result.Explanation)
	} else {
		output.WriteString("✅ **No Contradiction**\n\n")
		fmt.Fprintf(&output, "**Confidence:** %.0f%%\n", result.Confidence*100)
		fmt.Fprintf(&output, "**Explanation:** %s\n", result.Explanation)
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolRecallContradictionSummary returns a summary of all contradictions in the system.
func (s *MCPServer) toolRecallContradictionSummary(id any, args map[string]interface{}) jsonRPCResponse {
	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	summary, err := mem.GetContradictionSummary(limit)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get summary failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Contradiction Summary\n\n")
	fmt.Fprintf(&output, "**Total Contradiction Links:** %d\n", summary.TotalContradictionLinks)
	fmt.Fprintf(&output, "**Active Conflicts:** %d\n", summary.ActiveConflicts)
	fmt.Fprintf(&output, "**Resolved Conflicts:** %d\n\n", summary.ResolvedConflicts)

	if len(summary.TopContradictions) > 0 {
		output.WriteString("## Top Contradictions\n\n")
		for i, pair := range summary.TopContradictions {
			fmt.Fprintf(&output, "### %d. `%s` ⚔️ `%s`\n\n", i+1, pair.Record1.ID, pair.Record2.ID)
			fmt.Fprintf(&output, "**Record 1 (%s):** %s\n", pair.Record1.Kind, truncate(pair.Record1.Content, 100))
			fmt.Fprintf(&output, "**Record 2 (%s):** %s\n", pair.Record2.Kind, truncate(pair.Record2.Content, 100))
			fmt.Fprintf(&output, "**Link ID:** `%s`\n\n", pair.LinkID)
		}
	} else {
		output.WriteString("No recorded contradictions. ✅\n")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// truncate shortens a string to maxLen and adds "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
