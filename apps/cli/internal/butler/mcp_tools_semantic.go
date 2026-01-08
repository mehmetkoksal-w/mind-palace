package butler

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// toolSearchSemantic performs semantic search using embeddings.
func (s *MCPServer) toolSearchSemantic(id any, args map[string]interface{}) jsonRPCResponse {
	query, _ := args["query"].(string)
	if query == "" {
		return s.toolError(id, "query is required")
	}

	// Get embedder
	embedder := s.butler.GetEmbedder()
	if embedder == nil {
		return s.toolError(id, "semantic search requires an embedding backend to be configured. See 'palace config' to set up Ollama or OpenAI embeddings.")
	}

	// Parse options
	opts := memory.DefaultSemanticSearchOptions()

	if kinds, ok := args["kinds"].([]interface{}); ok {
		for _, k := range kinds {
			if kStr, ok := k.(string); ok {
				opts.Kinds = append(opts.Kinds, kStr)
			}
		}
	}

	if limit, ok := args["limit"].(float64); ok && limit > 0 {
		opts.Limit = int(limit)
	}

	if minSim, ok := args["minSimilarity"].(float64); ok && minSim > 0 {
		opts.MinSimilarity = float32(minSim)
	}

	if scope, ok := args["scope"].(string); ok {
		opts.Scope = scope
	}

	if scopePath, ok := args["scopePath"].(string); ok {
		opts.ScopePath = scopePath
	}

	// Perform search
	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	results, err := mem.SemanticSearch(embedder, query, opts)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("semantic search failed: %v", err))
	}

	// Format results
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Found %d semantically similar results:\n\n", len(results)))

	for i, r := range results {
		output.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, r.Kind, r.ID))
		output.WriteString(fmt.Sprintf("   Similarity: %.1f%%\n", r.Similarity*100))
		output.WriteString(fmt.Sprintf("   Content: %s\n", truncateString(r.Content, 100)))
		output.WriteString("\n")
	}

	// Also include JSON data for programmatic use
	formatted := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		formatted = append(formatted, map[string]interface{}{
			"id":         r.ID,
			"kind":       r.Kind,
			"content":    r.Content,
			"similarity": r.Similarity,
			"createdAt":  r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	jsonData, _ := json.MarshalIndent(formatted, "", "  ")
	output.WriteString("\n---\nJSON Data:\n")
	output.WriteString(string(jsonData))

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolSearchHybrid performs combined keyword + semantic search.
func (s *MCPServer) toolSearchHybrid(id any, args map[string]interface{}) jsonRPCResponse {
	query, _ := args["query"].(string)
	if query == "" {
		return s.toolError(id, "query is required")
	}

	// Parse options
	opts := memory.DefaultSemanticSearchOptions()

	if kinds, ok := args["kinds"].([]interface{}); ok {
		for _, k := range kinds {
			if kStr, ok := k.(string); ok {
				opts.Kinds = append(opts.Kinds, kStr)
			}
		}
	}

	if limit, ok := args["limit"].(float64); ok && limit > 0 {
		opts.Limit = int(limit)
	}

	// Get embedder (may be nil - hybrid search falls back to keyword only)
	embedder := s.butler.GetEmbedder()

	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	results, err := mem.HybridSearch(embedder, query, opts)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("hybrid search failed: %v", err))
	}

	// Format results
	searchMode := "hybrid (keyword + semantic)"
	if embedder == nil {
		searchMode = "keyword only (embeddings not configured)"
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Found %d results (%s):\n\n", len(results), searchMode))

	for i, r := range results {
		output.WriteString(fmt.Sprintf("%d. [%s] %s (%s match)\n", i+1, r.Kind, r.ID, r.MatchType))
		if r.Similarity > 0 {
			output.WriteString(fmt.Sprintf("   Similarity: %.1f%%\n", r.Similarity*100))
		}
		if r.FTSScore != 0 {
			output.WriteString(fmt.Sprintf("   FTS Score: %.2f\n", r.FTSScore))
		}
		output.WriteString(fmt.Sprintf("   Content: %s\n", truncateString(r.Content, 100)))
		output.WriteString("\n")
	}

	// Also include JSON data for programmatic use
	formatted := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		result := map[string]interface{}{
			"id":        r.ID,
			"kind":      r.Kind,
			"content":   r.Content,
			"matchType": r.MatchType,
			"createdAt": r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if r.Similarity > 0 {
			result["similarity"] = r.Similarity
		}
		if r.FTSScore != 0 {
			result["ftsScore"] = r.FTSScore
		}
		formatted = append(formatted, result)
	}

	jsonData, _ := json.MarshalIndent(formatted, "", "  ")
	output.WriteString("\n---\nJSON Data:\n")
	output.WriteString(string(jsonData))

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolSearchSimilar finds records similar to a given record.
func (s *MCPServer) toolSearchSimilar(id any, args map[string]interface{}) jsonRPCResponse {
	recordID, _ := args["recordId"].(string)
	if recordID == "" {
		return s.toolError(id, "recordId is required")
	}

	// Get embedder
	embedder := s.butler.GetEmbedder()
	if embedder == nil {
		return s.toolError(id, "finding similar records requires an embedding backend to be configured")
	}

	limit := 5
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	minSimilarity := float32(0.6)
	if minSim, ok := args["minSimilarity"].(float64); ok && minSim > 0 {
		minSimilarity = float32(minSim)
	}

	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	// Get the embedding for the source record
	sourceEmbedding, err := mem.GetEmbedding(recordID)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("record not found or has no embedding: %v", err))
	}

	// Find similar embeddings (search all kinds)
	results, err := mem.FindSimilarEmbeddings(sourceEmbedding, "", limit+1, minSimilarity)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("failed to find similar records: %v", err))
	}

	// Format results
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Records similar to %s:\n\n", recordID))

	formatted := make([]map[string]interface{}, 0, len(results))
	count := 0
	for _, r := range results {
		if r.RecordID == recordID {
			continue // Skip the source record
		}

		// Get record content
		content, createdAt, err := mem.GetRecordContent(r.RecordID, r.RecordKind)
		if err != nil {
			continue
		}

		count++
		output.WriteString(fmt.Sprintf("%d. [%s] %s\n", count, r.RecordKind, r.RecordID))
		output.WriteString(fmt.Sprintf("   Similarity: %.1f%%\n", r.Similarity*100))
		output.WriteString(fmt.Sprintf("   Content: %s\n", truncateString(content, 100)))
		output.WriteString("\n")

		formatted = append(formatted, map[string]interface{}{
			"id":         r.RecordID,
			"kind":       r.RecordKind,
			"content":    content,
			"similarity": r.Similarity,
			"createdAt":  createdAt.Format("2006-01-02T15:04:05Z"),
		})

		if count >= limit {
			break
		}
	}

	if count == 0 {
		output.WriteString("No similar records found.\n")
	}

	// Also include JSON data for programmatic use
	jsonData, _ := json.MarshalIndent(formatted, "", "  ")
	output.WriteString("\n---\nJSON Data:\n")
	output.WriteString(string(jsonData))

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// truncateString truncates a string to maxLen characters.
func truncateString(s string, maxLen int) string {
	// Remove newlines and extra whitespace
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// toolEmbeddingSync generates embeddings for records that don't have them.
func (s *MCPServer) toolEmbeddingSync(id any, args map[string]interface{}) jsonRPCResponse {
	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	// Get embedder
	embedder := s.butler.GetEmbedder()
	if embedder == nil {
		return s.toolError(id, "embedding sync requires an embedding backend to be configured. See 'palace config' to set up Ollama or OpenAI embeddings.")
	}

	// Parse options
	limit := 100
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	var kinds []string
	if kindsRaw, ok := args["kinds"].([]interface{}); ok {
		for _, k := range kindsRaw {
			if kStr, ok := k.(string); ok {
				kinds = append(kinds, kStr)
			}
		}
	}
	if len(kinds) == 0 {
		kinds = []string{"idea", "decision", "learning"}
	}

	// Get pipeline or create temporary one
	pipeline := mem.GetEmbeddingPipeline()
	if pipeline == nil {
		pipeline = memory.NewEmbeddingPipeline(mem, embedder, 2)
	}

	// Process pending embeddings
	processed, err := pipeline.ProcessPending(kinds, limit)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("embedding sync failed: %v", err))
	}

	// Get stats
	stats, err := mem.GetEmbeddingStats(mem.GetEmbeddingPipeline())
	if err != nil {
		stats = &memory.EmbeddingStats{}
	}

	// Format output
	var output strings.Builder
	output.WriteString("Embedding Sync Complete\n\n")
	output.WriteString(fmt.Sprintf("Processed: %d records\n", processed))
	output.WriteString(fmt.Sprintf("Total embeddings: %d\n", stats.TotalEmbeddings))
	fmt.Fprintf(&output, "\nBy kind:\n")
	for kind, count := range stats.ByKind {
		pending := stats.PendingByKind[kind]
		output.WriteString(fmt.Sprintf("  %s: %d embedded, %d pending\n", kind, count, pending))
	}

	if stats.PipelineRunning {
		output.WriteString(fmt.Sprintf("\nPipeline: running (queue size: %d)\n", stats.QueueSize))
	} else {
		output.WriteString("\nPipeline: not running\n")
	}

	// JSON data
	jsonData, _ := json.MarshalIndent(map[string]interface{}{
		"processed": processed,
		"stats":     stats,
	}, "", "  ")
	output.WriteString("\n---\nJSON Data:\n")
	output.WriteString(string(jsonData))

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolEmbeddingStats returns statistics about the embedding system.
func (s *MCPServer) toolEmbeddingStats(id any, _ map[string]interface{}) jsonRPCResponse {
	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	stats, err := mem.GetEmbeddingStats(mem.GetEmbeddingPipeline())
	if err != nil {
		return s.toolError(id, fmt.Sprintf("failed to get embedding stats: %v", err))
	}

	// Format output
	var output strings.Builder
	output.WriteString("Embedding Statistics\n\n")
	output.WriteString(fmt.Sprintf("Total embeddings: %d\n", stats.TotalEmbeddings))
	output.WriteString("\nBy kind:\n")
	for kind, count := range stats.ByKind {
		pending := stats.PendingByKind[kind]
		output.WriteString(fmt.Sprintf("  %s: %d embedded, %d pending\n", kind, count, pending))
	}

	if stats.PipelineRunning {
		output.WriteString(fmt.Sprintf("\nPipeline: running (queue size: %d)\n", stats.QueueSize))
	} else {
		output.WriteString("\nPipeline: not running\n")
	}

	embedder := s.butler.GetEmbedder()
	if embedder != nil {
		output.WriteString(fmt.Sprintf("\nEmbedder: %s\n", embedder.Model()))
	} else {
		output.WriteString("\nEmbedder: not configured\n")
	}

	// JSON data
	jsonData, _ := json.MarshalIndent(stats, "", "  ")
	output.WriteString("\n---\nJSON Data:\n")
	output.WriteString(string(jsonData))

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}
