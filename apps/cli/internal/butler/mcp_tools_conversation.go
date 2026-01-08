package butler

import (
	"fmt"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// toolConversationStore stores the current conversation for future context.
func (s *MCPServer) toolConversationStore(id any, args map[string]interface{}) jsonRPCResponse {
	summary, _ := args["summary"].(string)
	if summary == "" {
		return s.toolError(id, "summary is required")
	}

	// Parse messages array
	var messages []memory.Message
	if msgsRaw, ok := args["messages"].([]interface{}); ok {
		for _, m := range msgsRaw {
			if msgMap, ok := m.(map[string]interface{}); ok {
				role, _ := msgMap["role"].(string)
				content, _ := msgMap["content"].(string)
				if role != "" && content != "" {
					messages = append(messages, memory.Message{
						Role:    role,
						Content: content,
					})
				}
			}
		}
	}

	if len(messages) == 0 {
		return s.toolError(id, "messages array is required and must not be empty")
	}

	agentType, _ := args["agentType"].(string)
	if agentType == "" {
		agentType = "claude-code"
	}
	sessionID, _ := args["sessionId"].(string)

	conv := memory.Conversation{
		AgentType: agentType,
		Summary:   summary,
		Messages:  messages,
		SessionID: sessionID,
	}

	convID, err := s.butler.AddConversation(conv)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("store conversation failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Conversation Stored\n\n")
	fmt.Fprintf(&output, "**ID:** `%s`\n", convID)
	fmt.Fprintf(&output, "**Summary:** %s\n", summary)
	fmt.Fprintf(&output, "**Messages:** %d\n", len(messages))
	if sessionID != "" {
		fmt.Fprintf(&output, "**Session:** %s\n", sessionID)
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolConversationExtract extracts ideas, decisions, and learnings from a conversation using AI.
func (s *MCPServer) toolConversationExtract(id any, args map[string]interface{}) jsonRPCResponse {
	conversationID, _ := args["conversationId"].(string)
	if conversationID == "" {
		return s.toolError(id, "conversationId is required")
	}

	// Get the conversation
	conv, err := s.butler.GetConversation(conversationID)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get conversation failed: %v", err))
	}

	// Get LLM client
	llmClient, err := s.butler.GetLLMClient()
	if err != nil || llmClient == nil {
		return s.toolError(id, "LLM not configured - set llmBackend in palace.jsonc")
	}

	// Create extractor and extract
	extractor := memory.NewLLMExtractor(llmClient, s.butler.memory)
	recordIDs, err := extractor.ExtractFromConversation(*conv)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("extraction failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Conversation Extraction Complete\n\n")
	fmt.Fprintf(&output, "**Conversation:** `%s`\n", conversationID)
	fmt.Fprintf(&output, "**Summary:** %s\n", conv.Summary)
	fmt.Fprintf(&output, "**Messages Analyzed:** %d\n\n", len(conv.Messages))

	if len(recordIDs) == 0 {
		output.WriteString("No notable ideas, decisions, or learnings were extracted.\n")
	} else {
		fmt.Fprintf(&output, "## Extracted %d Record(s)\n\n", len(recordIDs))
		for _, rid := range recordIDs {
			fmt.Fprintf(&output, "- `%s`\n", rid)
		}
		output.WriteString("\nUse `recall` to view the extracted records.\n")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolConversationSearch searches past conversations by summary or content.
func (s *MCPServer) toolConversationSearch(id any, args map[string]interface{}) jsonRPCResponse {
	query, _ := args["query"].(string)
	sessionID, _ := args["sessionId"].(string)

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
		if limit > 50 {
			limit = 50
		}
	}

	var conversations []memory.Conversation
	var err error

	if query != "" {
		conversations, err = s.butler.SearchConversations(query, limit)
	} else {
		conversations, err = s.butler.GetConversations(sessionID, "", limit)
	}

	if err != nil {
		return s.toolError(id, fmt.Sprintf("search conversations failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Conversations Found\n\n")

	if len(conversations) == 0 {
		output.WriteString("No conversations found.\n")
	} else {
		fmt.Fprintf(&output, "Found %d conversation(s):\n\n", len(conversations))
		for i := range conversations {
			c := &conversations[i]
			fmt.Fprintf(&output, "## `%s`\n", c.ID)
			fmt.Fprintf(&output, "**Summary:** %s\n", c.Summary)
			fmt.Fprintf(&output, "**Agent:** %s\n", c.AgentType)
			fmt.Fprintf(&output, "**Messages:** %d\n", len(c.Messages))
			fmt.Fprintf(&output, "**Created:** %s\n\n", c.CreatedAt.Format("2006-01-02 15:04"))
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
