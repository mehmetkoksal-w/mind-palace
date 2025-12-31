package butler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// MCPServer handles Model Context Protocol communication.
type MCPServer struct {
	butler *Butler
	reader *bufio.Reader
	writer io.Writer
}

// JSON-RPC types
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      any         `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// MCP protocol types
type mcpServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type mcpCapabilities struct {
	Tools     *mcpToolsCap     `json:"tools,omitempty"`
	Resources *mcpResourcesCap `json:"resources,omitempty"`
}

type mcpToolsCap struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type mcpResourcesCap struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

type mcpInitializeResult struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    mcpCapabilities `json:"capabilities"`
	ServerInfo      mcpServerInfo   `json:"serverInfo"`
}

type mcpTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type mcpResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type mcpToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type mcpToolResult struct {
	Content []mcpContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

type mcpContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type mcpResourceReadParams struct {
	URI string `json:"uri"`
}

type mcpResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

// NewMCPServer creates a new MCP server backed by the given Butler.
func NewMCPServer(butler *Butler) *MCPServer {
	return &MCPServer{
		butler: butler,
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
	}
}

// Serve runs the MCP server, reading JSON-RPC requests from stdin.
func (s *MCPServer) Serve() error {
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil // Clean shutdown
			}
			return fmt.Errorf("read stdin: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.writeError(nil, -32700, "Parse error", err.Error())
			continue
		}

		resp := s.handleRequest(req)
		if err := s.writeResponse(resp); err != nil {
			return fmt.Errorf("write response: %w", err)
		}
	}
}

// handleRequest dispatches a JSON-RPC request to the appropriate handler.
func (s *MCPServer) handleRequest(req jsonRPCRequest) jsonRPCResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "initialized":
		// Notification, no response required
		return jsonRPCResponse{JSONRPC: "2.0", ID: req.ID}
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	case "resources/list":
		return s.handleResourcesList(req)
	case "resources/read":
		return s.handleResourcesRead(req)
	case "ping":
		return jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]string{}}
	default:
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", req.Method)},
		}
	}
}

// handleInitialize handles the MCP initialize request.
func (s *MCPServer) handleInitialize(req jsonRPCRequest) jsonRPCResponse {
	result := mcpInitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: mcpCapabilities{
			Tools:     &mcpToolsCap{},
			Resources: &mcpResourcesCap{},
		},
		ServerInfo: mcpServerInfo{
			Name:    "mind-palace",
			Version: "0.1.0",
		},
	}
	return jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
}

// handleToolsList returns the list of available tools.
func (s *MCPServer) handleToolsList(req jsonRPCRequest) jsonRPCResponse {
	tools := buildToolsList()
	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]interface{}{"tools": tools},
	}
}

// handleToolsCall dispatches a tool call to the appropriate handler.
func (s *MCPServer) handleToolsCall(req jsonRPCRequest) jsonRPCResponse {
	var params mcpToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32602, Message: "Invalid params", Data: err.Error()},
		}
	}

	switch params.Name {
	// Explore tools - search, context, symbols, graphs
	case "explore":
		return s.toolExplore(req.ID, params.Arguments)
	case "explore_rooms":
		return s.toolExploreRooms(req.ID)
	case "explore_context":
		return s.toolExploreContext(req.ID, params.Arguments)
	case "explore_impact":
		return s.toolExploreImpact(req.ID, params.Arguments)
	case "explore_symbols":
		return s.toolExploreSymbols(req.ID, params.Arguments)
	case "explore_symbol":
		return s.toolExploreSymbol(req.ID, params.Arguments)
	case "explore_file":
		return s.toolExploreFile(req.ID, params.Arguments)
	case "explore_deps":
		return s.toolExploreDeps(req.ID, params.Arguments)
	case "explore_callers":
		return s.toolExploreCallers(req.ID, params.Arguments)
	case "explore_callees":
		return s.toolExploreCallees(req.ID, params.Arguments)
	case "explore_graph":
		return s.toolExploreGraph(req.ID, params.Arguments)

	// Store tools - store ideas, decisions, learnings
	case "store":
		return s.toolStore(req.ID, params.Arguments)

	// Recall tools - retrieve knowledge and manage relationships
	case "recall":
		return s.toolRecall(req.ID, params.Arguments)
	case "recall_decisions":
		return s.toolRecallDecisions(req.ID, params.Arguments)
	case "recall_ideas":
		return s.toolRecallIdeas(req.ID, params.Arguments)
	case "recall_outcome":
		return s.toolRecallOutcome(req.ID, params.Arguments)
	case "recall_link":
		return s.toolRecallLink(req.ID, params.Arguments)
	case "recall_links":
		return s.toolRecallLinks(req.ID, params.Arguments)
	case "recall_unlink":
		return s.toolRecallUnlink(req.ID, params.Arguments)

	// Brief tools - get briefings and file intel
	case "brief":
		return s.toolBrief(req.ID, params.Arguments)
	case "brief_file":
		return s.toolBriefFile(req.ID, params.Arguments)
	case "briefing_smart":
		return s.toolBriefingSmart(req.ID, params.Arguments)

	// Session tools - manage agent sessions
	case "session_start":
		return s.toolSessionStart(req.ID, params.Arguments)
	case "session_log":
		return s.toolSessionLog(req.ID, params.Arguments)
	case "session_end":
		return s.toolSessionEnd(req.ID, params.Arguments)
	case "session_conflict":
		return s.toolSessionConflict(req.ID, params.Arguments)
	case "session_list":
		return s.toolSessionList(req.ID, params.Arguments)

	// Conversation tools - store and search conversations
	case "conversation_store":
		return s.toolConversationStore(req.ID, params.Arguments)
	case "conversation_search":
		return s.toolConversationSearch(req.ID, params.Arguments)
	case "conversation_extract":
		return s.toolConversationExtract(req.ID, params.Arguments)

	// Corridor tools - personal cross-workspace learnings
	case "corridor_learnings":
		return s.toolCorridorLearnings(req.ID, params.Arguments)
	case "corridor_links":
		return s.toolCorridorLinks(req.ID, params.Arguments)
	case "corridor_stats":
		return s.toolCorridorStats(req.ID, params.Arguments)
	case "corridor_promote":
		return s.toolCorridorPromote(req.ID, params.Arguments)
	case "corridor_reinforce":
		return s.toolCorridorReinforce(req.ID, params.Arguments)

	// Semantic Search Tools
	case "search_semantic":
		return s.toolSearchSemantic(req.ID, params.Arguments)
	case "search_hybrid":
		return s.toolSearchHybrid(req.ID, params.Arguments)
	case "search_similar":
		return s.toolSearchSimilar(req.ID, params.Arguments)

	// Embedding Management Tools
	case "embedding_sync":
		return s.toolEmbeddingSync(req.ID, params.Arguments)
	case "embedding_stats":
		return s.toolEmbeddingStats(req.ID, params.Arguments)

	// Learning Lifecycle Tools
	case "recall_learning_link":
		return s.toolRecallLearningLink(req.ID, params.Arguments)
	case "recall_obsolete":
		return s.toolRecallObsolete(req.ID, params.Arguments)
	case "recall_archive":
		return s.toolRecallArchive(req.ID, params.Arguments)
	case "recall_learnings_by_status":
		return s.toolRecallLearningsByStatus(req.ID, params.Arguments)

	// Contradiction Detection Tools
	case "recall_contradictions":
		return s.toolRecallContradictions(req.ID, params.Arguments)
	case "recall_contradiction_check":
		return s.toolRecallContradictionCheck(req.ID, params.Arguments)
	case "recall_contradiction_summary":
		return s.toolRecallContradictionSummary(req.ID, params.Arguments)

	// Decay Tools
	case "decay_stats":
		return s.toolDecayStats(req.ID, params.Arguments)
	case "decay_preview":
		return s.toolDecayPreview(req.ID, params.Arguments)
	case "decay_apply":
		return s.toolDecayApply(req.ID, params.Arguments)
	case "decay_reinforce":
		return s.toolDecayReinforce(req.ID, params.Arguments)
	case "decay_boost":
		return s.toolDecayBoost(req.ID, params.Arguments)

	// Context & Scope Tools
	case "context_auto_inject":
		return s.toolContextAutoInject(req.ID, params.Arguments)
	case "scope_explain":
		return s.toolScopeExplain(req.ID, params.Arguments)

	// Postmortem Tools
	case "store_postmortem":
		return s.toolStorePostmortem(req.ID, params.Arguments)
	case "get_postmortems":
		return s.toolGetPostmortems(req.ID, params.Arguments)
	case "get_postmortem":
		return s.toolGetPostmortem(req.ID, params.Arguments)
	case "resolve_postmortem":
		return s.toolResolvePostmortem(req.ID, params.Arguments)
	case "postmortem_stats":
		return s.toolPostmortemStats(req.ID, params.Arguments)
	case "postmortem_to_learnings":
		return s.toolPostmortemToLearnings(req.ID, params.Arguments)

	default:
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32602, Message: fmt.Sprintf("Unknown tool: %s", params.Name)},
		}
	}
}

// handleResourcesList returns the list of available resources.
func (s *MCPServer) handleResourcesList(req jsonRPCRequest) jsonRPCResponse {
	resources := []mcpResource{
		{
			URI:         "palace://files",
			Name:        "Indexed Files",
			Description: "Read files from the Mind Palace index. Use palace://files/{path} to read specific files.",
			MimeType:    "text/plain",
		},
		{
			URI:         "palace://rooms",
			Name:        "Room Manifests",
			Description: "Read room configuration. Use palace://rooms/{name} to read specific room manifests.",
			MimeType:    "application/json",
		},
	}

	// Add specific room resources
	for _, room := range s.butler.ListRooms() {
		resources = append(resources, mcpResource{
			URI:         fmt.Sprintf("palace://rooms/%s", room.Name),
			Name:        fmt.Sprintf("Room: %s", room.Name),
			Description: room.Summary,
			MimeType:    "application/json",
		})
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]interface{}{"resources": resources},
	}
}

// handleResourcesRead reads a resource by URI.
func (s *MCPServer) handleResourcesRead(req jsonRPCRequest) jsonRPCResponse {
	var params mcpResourceReadParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32602, Message: "Invalid params", Data: err.Error()},
		}
	}

	uri := params.URI

	// Parse URI: palace://files/{path} or palace://rooms/{name}
	if strings.HasPrefix(uri, "palace://files/") {
		path := strings.TrimPrefix(uri, "palace://files/")
		// Sanitize path to prevent path traversal attacks
		path = sanitizePath(path)
		if path == "" {
			return jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &rpcError{Code: -32602, Message: "Invalid file path"},
			}
		}
		content, err := s.butler.ReadFile(path)
		if err != nil {
			return jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &rpcError{Code: -32602, Message: err.Error()},
			}
		}
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"contents": []mcpResourceContent{{
					URI:      uri,
					MimeType: "text/plain",
					Text:     content,
				}},
			},
		}
	}

	if strings.HasPrefix(uri, "palace://rooms/") {
		name := strings.TrimPrefix(uri, "palace://rooms/")
		room, err := s.butler.ReadRoom(name)
		if err != nil {
			return jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &rpcError{Code: -32602, Message: err.Error()},
			}
		}
		roomJSON, _ := json.MarshalIndent(room, "", "  ")
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"contents": []mcpResourceContent{{
					URI:      uri,
					MimeType: "application/json",
					Text:     string(roomJSON),
				}},
			},
		}
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Error:   &rpcError{Code: -32602, Message: fmt.Sprintf("Unknown resource URI: %s", uri)},
	}
}

// writeResponse writes a JSON-RPC response to stdout.
func (s *MCPServer) writeResponse(resp jsonRPCResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(s.writer, "%s\n", data)
	return err
}

// writeError writes a JSON-RPC error response to stdout.
func (s *MCPServer) writeError(id any, code int, message, data string) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: message, Data: data},
	}
	s.writeResponse(resp)
}

// toolError creates a tool error response.
func (s *MCPServer) toolError(id any, msg string) jsonRPCResponse {
	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: fmt.Sprintf("Error: %s", msg)}},
			IsError: true,
		},
	}
}
