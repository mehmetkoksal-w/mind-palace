package butler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
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

	results, err := s.butler.Search(query, SearchOptions{
		Limit:      limit,
		RoomFilter: roomFilter,
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
