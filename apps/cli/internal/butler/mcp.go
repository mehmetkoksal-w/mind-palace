package butler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

type MCPServer struct {
	butler *Butler
	reader *bufio.Reader
	writer io.Writer
}

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

func NewMCPServer(butler *Butler) *MCPServer {
	return &MCPServer{
		butler: butler,
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
	}
}

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

func (s *MCPServer) handleToolsList(req jsonRPCRequest) jsonRPCResponse {
	tools := []mcpTool{
		{
			Name:        "search_mind_palace",
			Description: "Search the codebase by intent or keywords. Returns ranked results grouped by 'Rooms' (conceptual areas of the project).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query - can be natural language ('where is auth logic') or code symbols ('handleAuth', 'AuthService').",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results (default: 10, max: 50)",
						"default":     10,
					},
					"room": map[string]interface{}{
						"type":        "string",
						"description": "Optional: filter results to a specific Room name",
					},
					"fuzzy": map[string]interface{}{
						"type":        "boolean",
						"description": "Enable fuzzy matching for typo tolerance (default: false). Useful when unsure of exact spelling.",
						"default":     false,
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "list_rooms",
			Description: "List all available Rooms in the Mind Palace. Rooms are curated conceptual areas of the project with entry points and capabilities.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "get_context",
			Description: "Get complete context for a task. This is the ORACLE query - a single call that returns all relevant files, symbols, imports, architectural decisions, AND relevant learnings from session memory.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task": map[string]interface{}{
						"type":        "string",
						"description": "Description of the task you want to accomplish (e.g., 'add user authentication', 'fix the login bug', 'understand the payment flow').",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of files/symbols to return (default: 20)",
						"default":     20,
					},
					"maxTokens": map[string]interface{}{
						"type":        "integer",
						"description": "Token budget for the response. Context will be truncated to fit within this budget. Use this to ensure the response fits in your context window.",
					},
					"includeTests": map[string]interface{}{
						"type":        "boolean",
						"description": "Include test files in the results (default: false). By default, test files are excluded to reduce noise.",
						"default":     false,
					},
					"includeLearnings": map[string]interface{}{
						"type":        "boolean",
						"description": "Include relevant learnings from session memory (default: true). Learnings provide accumulated knowledge about the codebase.",
						"default":     true,
					},
					"includeFileIntel": map[string]interface{}{
						"type":        "boolean",
						"description": "Include file intelligence (edit history, failure rates) for files in context (default: true).",
						"default":     true,
					},
				},
				"required": []string{"task"},
			},
		},
		{
			Name:        "get_impact",
			Description: "Analyze the impact of changing a file or symbol. Returns what depends on it (dependents) and what it depends on (dependencies).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"target": map[string]interface{}{
						"type":        "string",
						"description": "File path or symbol name to analyze (e.g., 'internal/auth/handler.go', 'AuthService').",
					},
				},
				"required": []string{"target"},
			},
		},
		{
			Name:        "list_symbols",
			Description: "List symbols of a specific kind in the codebase (e.g., all classes, all functions, all interfaces).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"kind": map[string]interface{}{
						"type":        "string",
						"description": "Symbol kind: 'class', 'interface', 'function', 'method', 'constant', 'type', 'enum', 'property', 'constructor'.",
						"enum":        []string{"class", "interface", "function", "method", "constant", "type", "enum", "property", "constructor"},
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of symbols to return (default: 50)",
						"default":     50,
					},
				},
				"required": []string{"kind"},
			},
		},
		{
			Name:        "get_symbol",
			Description: "Get detailed information about a specific symbol (function, class, method, etc.) including its signature, documentation, and location.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the symbol to look up.",
					},
					"file": map[string]interface{}{
						"type":        "string",
						"description": "Optional: file path to narrow down the search if the symbol name is common.",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "get_file_symbols",
			Description: "Get all exported symbols in a specific file.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file": map[string]interface{}{
						"type":        "string",
						"description": "File path to get symbols from.",
					},
				},
				"required": []string{"file"},
			},
		},
		{
			Name:        "get_dependencies",
			Description: "Get the dependency graph for one or more files, showing what they import.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"files": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "List of file paths to analyze.",
					},
				},
				"required": []string{"files"},
			},
		},
		{
			Name:        "get_callers",
			Description: "Find all locations that call a function or method. Answers 'who calls this function?'",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"symbol": map[string]interface{}{
						"type":        "string",
						"description": "Function or method name to find callers of (e.g., 'handleAuth', 'config.Parse', 'User.Save').",
					},
				},
				"required": []string{"symbol"},
			},
		},
		{
			Name:        "get_callees",
			Description: "Find all functions/methods called by a symbol. Answers 'what does this function call?'",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"symbol": map[string]interface{}{
						"type":        "string",
						"description": "Function or method name to analyze.",
					},
					"file": map[string]interface{}{
						"type":        "string",
						"description": "File path where the symbol is defined (required if symbol name is ambiguous).",
					},
				},
				"required": []string{"symbol", "file"},
			},
		},
		{
			Name:        "get_call_graph",
			Description: "Get the complete call graph for a file, showing all incoming and outgoing calls.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file": map[string]interface{}{
						"type":        "string",
						"description": "File path to analyze.",
					},
				},
				"required": []string{"file"},
			},
		},
		// Session Memory Tools
		{
			Name:        "start_session",
			Description: "Start a new agent session. Sessions track activities and learnings within a work context.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"agentType": map[string]interface{}{
						"type":        "string",
						"description": "Type of agent (e.g., 'claude', 'cursor', 'aider').",
					},
					"agentId": map[string]interface{}{
						"type":        "string",
						"description": "Optional unique agent instance ID.",
					},
					"goal": map[string]interface{}{
						"type":        "string",
						"description": "What this session aims to accomplish.",
					},
				},
				"required": []string{"agentType"},
			},
		},
		{
			Name:        "log_activity",
			Description: "Log an activity within a session (file read, edit, search, command).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sessionId": map[string]interface{}{
						"type":        "string",
						"description": "Session ID from start_session.",
					},
					"kind": map[string]interface{}{
						"type":        "string",
						"description": "Activity type: 'file_read', 'file_edit', 'search', 'command'.",
						"enum":        []string{"file_read", "file_edit", "search", "command"},
					},
					"target": map[string]interface{}{
						"type":        "string",
						"description": "File path or search query.",
					},
					"outcome": map[string]interface{}{
						"type":        "string",
						"description": "Result: 'success', 'failure', 'unknown'.",
						"enum":        []string{"success", "failure", "unknown"},
					},
					"details": map[string]interface{}{
						"type":        "string",
						"description": "Optional JSON details about the activity.",
					},
				},
				"required": []string{"sessionId", "kind", "target"},
			},
		},
		{
			Name:        "end_session",
			Description: "End a session and record its outcome.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sessionId": map[string]interface{}{
						"type":        "string",
						"description": "Session ID to end.",
					},
					"outcome": map[string]interface{}{
						"type":        "string",
						"description": "Session outcome: 'success', 'failure', 'partial'.",
						"enum":        []string{"success", "failure", "partial"},
					},
					"summary": map[string]interface{}{
						"type":        "string",
						"description": "Brief summary of what was accomplished.",
					},
				},
				"required": []string{"sessionId"},
			},
		},
		{
			Name:        "add_learning",
			Description: "Store a learning for future reference. Learnings accumulate knowledge about the codebase.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The learning content (what you learned).",
					},
					"scope": map[string]interface{}{
						"type":        "string",
						"description": "Scope: 'palace' (workspace-wide), 'room' (module), 'file' (specific file).",
						"enum":        []string{"palace", "room", "file"},
						"default":     "palace",
					},
					"scopePath": map[string]interface{}{
						"type":        "string",
						"description": "Path for room/file scope (room name or file path).",
					},
					"confidence": map[string]interface{}{
						"type":        "number",
						"description": "Initial confidence 0.0-1.0 (default: 0.5).",
						"default":     0.5,
					},
				},
				"required": []string{"content"},
			},
		},
		{
			Name:        "get_learnings",
			Description: "Retrieve learnings, optionally filtered by scope or search query.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Optional search query to filter learnings.",
					},
					"scope": map[string]interface{}{
						"type":        "string",
						"description": "Filter by scope: 'palace', 'room', 'file'.",
						"enum":        []string{"palace", "room", "file"},
					},
					"scopePath": map[string]interface{}{
						"type":        "string",
						"description": "Filter by scope path.",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum learnings to return (default: 10).",
						"default":     10,
					},
				},
			},
		},
		{
			Name:        "get_file_intel",
			Description: "Get intelligence about a file: edit history, failure rate, associated learnings.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "File path to get intelligence for.",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "get_brief",
			Description: "Get a comprehensive briefing before working. Shows active agents, conflicts, learnings, hotspots.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"filePath": map[string]interface{}{
						"type":        "string",
						"description": "Optional file path for file-specific briefing.",
					},
				},
			},
		},
		{
			Name:        "check_conflict",
			Description: "Check if another agent is working on a file.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "File path to check for conflicts.",
					},
					"sessionId": map[string]interface{}{
						"type":        "string",
						"description": "Optional current session ID to exclude from conflict check.",
					},
				},
				"required": []string{"path"},
			},
		},
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]interface{}{"tools": tools},
	}
}

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
	case "search_mind_palace":
		return s.toolSearchMindPalace(req.ID, params.Arguments)
	case "list_rooms":
		return s.toolListRooms(req.ID)
	case "get_context":
		return s.toolGetContext(req.ID, params.Arguments)
	case "get_impact":
		return s.toolGetImpact(req.ID, params.Arguments)
	case "list_symbols":
		return s.toolListSymbols(req.ID, params.Arguments)
	case "get_symbol":
		return s.toolGetSymbol(req.ID, params.Arguments)
	case "get_file_symbols":
		return s.toolGetFileSymbols(req.ID, params.Arguments)
	case "get_dependencies":
		return s.toolGetDependencies(req.ID, params.Arguments)
	case "get_callers":
		return s.toolGetCallers(req.ID, params.Arguments)
	case "get_callees":
		return s.toolGetCallees(req.ID, params.Arguments)
	case "get_call_graph":
		return s.toolGetCallGraph(req.ID, params.Arguments)
	// Session Memory Tools
	case "start_session":
		return s.toolStartSession(req.ID, params.Arguments)
	case "log_activity":
		return s.toolLogActivity(req.ID, params.Arguments)
	case "end_session":
		return s.toolEndSession(req.ID, params.Arguments)
	case "add_learning":
		return s.toolAddLearning(req.ID, params.Arguments)
	case "get_learnings":
		return s.toolGetLearnings(req.ID, params.Arguments)
	case "get_file_intel":
		return s.toolGetFileIntel(req.ID, params.Arguments)
	case "get_brief":
		return s.toolGetBrief(req.ID, params.Arguments)
	case "check_conflict":
		return s.toolCheckConflict(req.ID, params.Arguments)
	default:
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32602, Message: fmt.Sprintf("Unknown tool: %s", params.Name)},
		}
	}
}

func (s *MCPServer) toolSearchMindPalace(id any, args map[string]interface{}) jsonRPCResponse {
	query, _ := args["query"].(string)
	if query == "" {
		return s.toolError(id, "query is required")
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
		if limit > 50 {
			limit = 50
		}
	}

	roomFilter, _ := args["room"].(string)
	fuzzyMatch, _ := args["fuzzy"].(bool)

	results, err := s.butler.Search(query, SearchOptions{
		Limit:      limit,
		RoomFilter: roomFilter,
		FuzzyMatch: fuzzyMatch,
	})
	if err != nil {
		return s.toolError(id, fmt.Sprintf("search failed: %v", err))
	}

	// Format results as readable text
	var output strings.Builder
	for _, group := range results {
		fmt.Fprintf(&output, "## Room: %s\n", group.Room)
		if group.Summary != "" {
			fmt.Fprintf(&output, "_Summary: %s_\n", group.Summary)
		}
		output.WriteString("\n")

		for _, r := range group.Results {
			entryMark := ""
			if r.IsEntry {
				entryMark = " ⭐ (entry point)"
			}
			fmt.Fprintf(&output, "### %s%s\n", r.Path, entryMark)
			fmt.Fprintf(&output, "Lines %d-%d (score: %.2f)\n", r.StartLine, r.EndLine, r.Score)
			fmt.Fprintf(&output, "```\n%s\n```\n\n", truncateSnippet(r.Snippet, 500))
		}
	}

	if len(results) == 0 {
		output.WriteString("No results found for the query.")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

func (s *MCPServer) toolListRooms(id any) jsonRPCResponse {
	rooms := s.butler.ListRooms()

	var output strings.Builder
	output.WriteString("# Mind Palace Rooms\n\n")

	for _, room := range rooms {
		fmt.Fprintf(&output, "## %s\n", room.Name)
		fmt.Fprintf(&output, "%s\n\n", room.Summary)
		output.WriteString("**Entry Points:**\n")
		for _, ep := range room.EntryPoints {
			fmt.Fprintf(&output, "- `%s`\n", ep)
		}
		if len(room.Capabilities) > 0 {
			output.WriteString("\n**Capabilities:**\n")
			for _, cap := range room.Capabilities {
				fmt.Fprintf(&output, "- %s\n", cap)
			}
		}
		output.WriteString("\n")
	}

	if len(rooms) == 0 {
		output.WriteString("No rooms defined. Run `palace init` to create default rooms.")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

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

func (s *MCPServer) writeResponse(resp jsonRPCResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(s.writer, "%s\n", data)
	return err
}

func (s *MCPServer) writeError(id any, code int, message, data string) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: message, Data: data},
	}
	s.writeResponse(resp)
}

func truncateSnippet(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated)"
}

// New Oracle tool handlers

func (s *MCPServer) toolGetContext(id any, args map[string]interface{}) jsonRPCResponse {
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

	// Use enhanced context with memory data
	opts := EnhancedContextOptions{
		Query:            task,
		Limit:            limit,
		MaxTokens:        maxTokens,
		IncludeTests:     includeTests,
		IncludeLearnings: includeLearnings,
		IncludeFileIntel: includeFileIntel,
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

func (s *MCPServer) toolGetImpact(id any, args map[string]interface{}) jsonRPCResponse {
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

func (s *MCPServer) toolListSymbols(id any, args map[string]interface{}) jsonRPCResponse {
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

func (s *MCPServer) toolGetSymbol(id any, args map[string]interface{}) jsonRPCResponse {
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

func (s *MCPServer) toolGetFileSymbols(id any, args map[string]interface{}) jsonRPCResponse {
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

func (s *MCPServer) toolGetDependencies(id any, args map[string]interface{}) jsonRPCResponse {
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

func (s *MCPServer) toolGetCallers(id any, args map[string]interface{}) jsonRPCResponse {
	symbol, _ := args["symbol"].(string)
	if symbol == "" {
		return s.toolError(id, "symbol is required")
	}

	calls, err := s.butler.GetIncomingCalls(symbol)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get callers failed: %v", err))
	}

	var output strings.Builder
	fmt.Fprintf(&output, "# Callers of `%s`\n\n", symbol)

	if len(calls) == 0 {
		output.WriteString("No callers found. This symbol may not be called anywhere, or call tracking may not be available for this language.\n")
	} else {
		fmt.Fprintf(&output, "Found %d call sites:\n\n", len(calls))
		for _, call := range calls {
			fmt.Fprintf(&output, "- `%s` line %d", call.FilePath, call.Line)
			if call.CallerSymbol != "" {
				fmt.Fprintf(&output, " (in function `%s`)", call.CallerSymbol)
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

func (s *MCPServer) toolGetCallees(id any, args map[string]interface{}) jsonRPCResponse {
	symbol, _ := args["symbol"].(string)
	if symbol == "" {
		return s.toolError(id, "symbol is required")
	}

	file, _ := args["file"].(string)
	if file == "" {
		return s.toolError(id, "file is required to find the symbol's scope")
	}

	calls, err := s.butler.GetOutgoingCalls(symbol, file)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get callees failed: %v", err))
	}

	var output strings.Builder
	fmt.Fprintf(&output, "# Functions called by `%s`\n\n", symbol)
	fmt.Fprintf(&output, "File: `%s`\n\n", file)

	if len(calls) == 0 {
		output.WriteString("No outgoing calls found. This function may not call other functions, or call tracking may not be available.\n")
	} else {
		fmt.Fprintf(&output, "Found %d function calls:\n\n", len(calls))
		for _, call := range calls {
			fmt.Fprintf(&output, "- `%s` (line %d)\n", call.CalleeSymbol, call.Line)
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

func (s *MCPServer) toolGetCallGraph(id any, args map[string]interface{}) jsonRPCResponse {
	file, _ := args["file"].(string)
	if file == "" {
		return s.toolError(id, "file is required")
	}

	graph, err := s.butler.GetCallGraph(file)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get call graph failed: %v", err))
	}

	var output strings.Builder
	fmt.Fprintf(&output, "# Call Graph for `%s`\n\n", file)

	output.WriteString("## Incoming Calls (who calls functions in this file)\n\n")
	if len(graph.IncomingCalls) == 0 {
		output.WriteString("No incoming calls from other files.\n\n")
	} else {
		for _, call := range graph.IncomingCalls {
			fmt.Fprintf(&output, "- `%s` called from `%s` line %d", call.CalleeSymbol, call.FilePath, call.Line)
			if call.CallerSymbol != "" {
				fmt.Fprintf(&output, " (in `%s`)", call.CallerSymbol)
			}
			output.WriteString("\n")
		}
		output.WriteString("\n")
	}

	output.WriteString("## Outgoing Calls (what this file calls)\n\n")
	if len(graph.OutgoingCalls) == 0 {
		output.WriteString("No outgoing calls tracked.\n\n")
	} else {
		for _, call := range graph.OutgoingCalls {
			fmt.Fprintf(&output, "- `%s` at line %d", call.CalleeSymbol, call.Line)
			if call.CallerSymbol != "" {
				fmt.Fprintf(&output, " (from `%s`)", call.CallerSymbol)
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

// ============================================================================
// Session Memory Tool Handlers
// ============================================================================

func (s *MCPServer) toolStartSession(id any, args map[string]interface{}) jsonRPCResponse {
	agentType, _ := args["agentType"].(string)
	if agentType == "" {
		return s.toolError(id, "agentType is required")
	}

	agentID, _ := args["agentId"].(string)
	goal, _ := args["goal"].(string)

	session, err := s.butler.StartSession(agentType, agentID, goal)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("start session failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Session Started\n\n")
	fmt.Fprintf(&output, "**Session ID:** `%s`\n", session.ID)
	fmt.Fprintf(&output, "**Agent Type:** %s\n", session.AgentType)
	if session.Goal != "" {
		fmt.Fprintf(&output, "**Goal:** %s\n", session.Goal)
	}
	fmt.Fprintf(&output, "**Started:** %s\n", session.StartedAt.Format(time.RFC3339))
	output.WriteString("\nUse this session ID to log activities and end the session.")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

func (s *MCPServer) toolLogActivity(id any, args map[string]interface{}) jsonRPCResponse {
	sessionID, _ := args["sessionId"].(string)
	if sessionID == "" {
		return s.toolError(id, "sessionId is required")
	}

	kind, _ := args["kind"].(string)
	if kind == "" {
		return s.toolError(id, "kind is required")
	}

	target, _ := args["target"].(string)
	outcome, _ := args["outcome"].(string)
	details, _ := args["details"].(string)

	if outcome == "" {
		outcome = "unknown"
	}
	if details == "" {
		details = "{}"
	}

	act := memory.Activity{
		Kind:    kind,
		Target:  target,
		Outcome: outcome,
		Details: details,
	}

	err := s.butler.LogActivity(sessionID, act)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("log activity failed: %v", err))
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: fmt.Sprintf("Activity logged: %s on %s (%s)", kind, target, outcome)}},
		},
	}
}

func (s *MCPServer) toolEndSession(id any, args map[string]interface{}) jsonRPCResponse {
	sessionID, _ := args["sessionId"].(string)
	if sessionID == "" {
		return s.toolError(id, "sessionId is required")
	}

	outcome, _ := args["outcome"].(string)
	summary, _ := args["summary"].(string)

	state := "completed"
	if outcome == "failure" {
		state = "abandoned"
	}

	// Record outcome if provided
	if outcome != "" {
		if err := s.butler.RecordOutcome(sessionID, outcome, summary); err != nil {
			return s.toolError(id, fmt.Sprintf("record outcome failed: %v", err))
		}
	}

	if err := s.butler.EndSession(sessionID, state, summary); err != nil {
		return s.toolError(id, fmt.Sprintf("end session failed: %v", err))
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: fmt.Sprintf("Session %s ended (state: %s)", sessionID, state)}},
		},
	}
}

func (s *MCPServer) toolAddLearning(id any, args map[string]interface{}) jsonRPCResponse {
	content, _ := args["content"].(string)
	if content == "" {
		return s.toolError(id, "content is required")
	}

	scope, _ := args["scope"].(string)
	if scope == "" {
		scope = "palace"
	}

	scopePath, _ := args["scopePath"].(string)

	confidence := 0.5
	if c, ok := args["confidence"].(float64); ok {
		confidence = c
	}

	learning := memory.Learning{
		Scope:      scope,
		ScopePath:  scopePath,
		Content:    content,
		Confidence: confidence,
		Source:     "agent",
	}

	learningID, err := s.butler.AddLearning(learning)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("add learning failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Learning Stored\n\n")
	fmt.Fprintf(&output, "**ID:** `%s`\n", learningID)
	fmt.Fprintf(&output, "**Scope:** %s", scope)
	if scopePath != "" {
		fmt.Fprintf(&output, " (%s)", scopePath)
	}
	output.WriteString("\n")
	fmt.Fprintf(&output, "**Confidence:** %.0f%%\n", confidence*100)
	fmt.Fprintf(&output, "**Content:** %s\n", content)

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

func (s *MCPServer) toolGetLearnings(id any, args map[string]interface{}) jsonRPCResponse {
	query, _ := args["query"].(string)
	scope, _ := args["scope"].(string)
	scopePath, _ := args["scopePath"].(string)

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	var learnings []memory.Learning
	var err error

	if query != "" {
		learnings, err = s.butler.SearchLearnings(query, limit)
	} else {
		learnings, err = s.butler.GetLearnings(scope, scopePath, limit)
	}

	if err != nil {
		return s.toolError(id, fmt.Sprintf("get learnings failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Learnings\n\n")

	if len(learnings) == 0 {
		output.WriteString("No learnings found.\n")
	} else {
		for _, l := range learnings {
			scopeInfo := l.Scope
			if l.ScopePath != "" {
				scopeInfo = fmt.Sprintf("%s:%s", l.Scope, l.ScopePath)
			}
			fmt.Fprintf(&output, "## `%s` (%.0f%% confidence)\n", l.ID, l.Confidence*100)
			fmt.Fprintf(&output, "- **Scope:** %s\n", scopeInfo)
			fmt.Fprintf(&output, "- **Source:** %s | Used: %d times\n", l.Source, l.UseCount)
			fmt.Fprintf(&output, "- **Content:** %s\n\n", l.Content)
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

func (s *MCPServer) toolGetFileIntel(id any, args map[string]interface{}) jsonRPCResponse {
	path, _ := args["path"].(string)
	if path == "" {
		return s.toolError(id, "path is required")
	}

	intel, err := s.butler.GetFileIntel(path)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get file intel failed: %v", err))
	}

	var output strings.Builder
	fmt.Fprintf(&output, "# File Intelligence: `%s`\n\n", path)
	fmt.Fprintf(&output, "**Edit Count:** %d\n", intel.EditCount)
	fmt.Fprintf(&output, "**Failure Count:** %d\n", intel.FailureCount)
	if intel.EditCount > 0 {
		failureRate := float64(intel.FailureCount) / float64(intel.EditCount) * 100
		fmt.Fprintf(&output, "**Failure Rate:** %.1f%%\n", failureRate)
	}
	if !intel.LastEdited.IsZero() {
		fmt.Fprintf(&output, "**Last Edited:** %s\n", intel.LastEdited.Format(time.RFC3339))
	}
	if intel.LastEditor != "" {
		fmt.Fprintf(&output, "**Last Editor:** %s\n", intel.LastEditor)
	}

	if len(intel.Learnings) > 0 {
		output.WriteString("\n## Associated Learnings\n\n")
		for _, learningID := range intel.Learnings {
			fmt.Fprintf(&output, "- `%s`\n", learningID)
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

func (s *MCPServer) toolGetBrief(id any, args map[string]interface{}) jsonRPCResponse {
	filePath, _ := args["filePath"].(string)

	brief, err := s.butler.GetBrief(filePath)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get brief failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Briefing")
	if filePath != "" {
		fmt.Fprintf(&output, " for `%s`", filePath)
	}
	output.WriteString("\n\n")

	// Active agents
	if len(brief.ActiveAgents) > 0 {
		output.WriteString("## Active Agents\n\n")
		for _, a := range brief.ActiveAgents {
			currentFile := ""
			if a.CurrentFile != "" {
				currentFile = fmt.Sprintf(" working on `%s`", a.CurrentFile)
			}
			fmt.Fprintf(&output, "- **%s** (session: `%s`)%s\n", a.AgentType, a.SessionID[:12], currentFile)
		}
		output.WriteString("\n")
	}

	// Conflict warning
	if brief.Conflict != nil {
		output.WriteString("## ⚠️ Conflict Warning\n\n")
		fmt.Fprintf(&output, "Another agent (**%s**) touched this file recently.\n", brief.Conflict.OtherAgent)
		fmt.Fprintf(&output, "- Session: `%s`\n", brief.Conflict.OtherSession[:12])
		fmt.Fprintf(&output, "- Last touched: %s\n", brief.Conflict.LastTouched.Format("15:04:05"))
		fmt.Fprintf(&output, "- Severity: %s\n\n", brief.Conflict.Severity)
	}

	// File intel
	if brief.FileIntel != nil && brief.FileIntel.EditCount > 0 {
		output.WriteString("## File History\n\n")
		fmt.Fprintf(&output, "- **Edits:** %d\n", brief.FileIntel.EditCount)
		fmt.Fprintf(&output, "- **Failures:** %d\n", brief.FileIntel.FailureCount)
		if brief.FileIntel.EditCount > 0 {
			failureRate := float64(brief.FileIntel.FailureCount) / float64(brief.FileIntel.EditCount) * 100
			if failureRate > 20 {
				fmt.Fprintf(&output, "- ⚠️ **High failure rate:** %.0f%%\n", failureRate)
			}
		}
		output.WriteString("\n")
	}

	// Relevant learnings
	if len(brief.Learnings) > 0 {
		output.WriteString("## Relevant Learnings\n\n")
		for _, l := range brief.Learnings {
			scopeInfo := ""
			if l.Scope != "palace" {
				scopeInfo = fmt.Sprintf(" [%s]", l.Scope)
			}
			fmt.Fprintf(&output, "- [%.0f%%]%s %s\n", l.Confidence*100, scopeInfo, l.Content)
		}
		output.WriteString("\n")
	}

	// Hotspots
	if len(brief.Hotspots) > 0 {
		output.WriteString("## Hotspots (Most Edited Files)\n\n")
		for _, h := range brief.Hotspots {
			warning := ""
			if h.FailureCount > 0 {
				warning = fmt.Sprintf(" (⚠️ %d failures)", h.FailureCount)
			}
			fmt.Fprintf(&output, "- `%s` (%d edits)%s\n", h.Path, h.EditCount, warning)
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

func (s *MCPServer) toolCheckConflict(id any, args map[string]interface{}) jsonRPCResponse {
	path, _ := args["path"].(string)
	if path == "" {
		return s.toolError(id, "path is required")
	}

	sessionID, _ := args["sessionId"].(string)

	conflict, err := s.butler.CheckConflict(sessionID, path)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("check conflict failed: %v", err))
	}

	var output strings.Builder
	fmt.Fprintf(&output, "# Conflict Check: `%s`\n\n", path)

	if conflict == nil {
		output.WriteString("✅ **No conflicts detected.** Safe to proceed.\n")
	} else {
		output.WriteString("⚠️ **Conflict detected!**\n\n")
		fmt.Fprintf(&output, "- **Other Agent:** %s\n", conflict.OtherAgent)
		fmt.Fprintf(&output, "- **Session:** `%s`\n", conflict.OtherSession[:12])
		fmt.Fprintf(&output, "- **Last Touched:** %s\n", conflict.LastTouched.Format(time.RFC3339))
		fmt.Fprintf(&output, "- **Severity:** %s\n", conflict.Severity)
		output.WriteString("\nConsider coordinating with the other agent before making changes.")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// sanitizePath cleans a file path and prevents path traversal attacks.
// Returns empty string if the path is invalid or attempts to escape workspace.
func sanitizePath(path string) string {
	// Clean the path to normalize . and .. elements
	clean := filepath.Clean(path)

	// Reject absolute paths
	if filepath.IsAbs(clean) {
		return ""
	}

	// Reject paths that try to escape (start with ..)
	if strings.HasPrefix(clean, "..") {
		return ""
	}

	// Reject paths containing .. anywhere (even after clean)
	if strings.Contains(clean, "..") {
		return ""
	}

	return clean
}
