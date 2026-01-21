package butler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/memory"
)

// MCPMode represents the operational mode of the MCP server.
// Mode determines which tools are available and what security restrictions apply.
type MCPMode string

const (
	// MCPModeAgent is restricted mode - no admin tools, no direct write.
	// Agents can only create proposals; they cannot bypass governance.
	MCPModeAgent MCPMode = "agent"

	// MCPModeHuman is full access mode - admin tools and direct write available.
	// Only use when the MCP client is a human-operated interface.
	MCPModeHuman MCPMode = "human"
)

// ValidMCPModes returns all valid MCP mode values.
func ValidMCPModes() []MCPMode {
	return []MCPMode{MCPModeAgent, MCPModeHuman}
}

// IsValidMCPMode returns true if the mode is valid.
func IsValidMCPMode(mode string) bool {
	for _, m := range ValidMCPModes() {
		if string(m) == mode {
			return true
		}
	}
	return false
}

// adminOnlyTools lists tools that are only available in human mode.
// These tools can bypass the proposal system, perform privileged operations,
// or mutate memory state (outcomes, links, archival, obsolescence).
var adminOnlyTools = map[string]bool{
	"store_direct":    true, // Bypasses proposal system
	"approve":         true, // Approves proposals
	"reject":          true, // Rejects proposals
	"recall_outcome":  true, // Marks decisions with outcomes
	"recall_link":     true, // Links ideas/decisions/learnings
	"recall_unlink":   true, // Removes links
	"recall_obsolete": true, // Marks learnings obsolete
	"recall_archive":  true, // Archives learnings
}

// IsAdminOnlyTool returns true if the tool requires human mode.
func IsAdminOnlyTool(toolName string) bool {
	return adminOnlyTools[toolName]
}

// GetAdminOnlyTools returns a copy of the admin-only tools map.
func GetAdminOnlyTools() map[string]bool {
	result := make(map[string]bool, len(adminOnlyTools))
	for k, v := range adminOnlyTools {
		result[k] = v
	}
	return result
}

// MCPServer handles Model Context Protocol communication.
type MCPServer struct {
	butler *Butler
	reader *bufio.Reader
	writer io.Writer
	mode   MCPMode // Operational mode (agent or human)

	// Session tracking for autonomy features
	currentSessionID string // Active session ID for this connection
	autoSessionUsed  bool   // True if current session was auto-created

	// Session timeout tracking
	lastActivity        time.Time // Time of last tool call
	sessionAutoEnded    bool      // True if session was auto-ended due to timeout
	sessionAutoEndedMsg string    // Message to show on next tool call

	// Background goroutine management
	stopTimeout chan struct{} // Channel to stop timeout checker

	// Proactive intelligence: conflict monitoring
	trackedFiles map[string]time.Time // Files accessed by this session with last access time

	// Proactive intelligence: briefing updates
	lastBriefingTime time.Time // Time of last briefing update shown

	// Smart context management
	currentTaskFocus  string   // Current task/goal for context prioritization
	focusKeywords     []string // Keywords extracted from current focus
	contextPriorityUp []string // Record IDs to prioritize (pinned)
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

// mcpToolAutonomy provides metadata for agent autonomy guidance.
// This helps agents understand when and how to use tools without explicit instructions.
type mcpToolAutonomy struct {
	// Level indicates how critical this tool is: "required", "recommended", or "optional"
	Level string `json:"level"`
	// Prerequisites lists tools that should be called before this one
	Prerequisites []string `json:"prerequisites,omitempty"`
	// Triggers describes when this tool should be called
	Triggers []string `json:"triggers,omitempty"`
	// Frequency indicates how often: "once_per_session", "per_file", "as_needed"
	Frequency string `json:"frequency,omitempty"`
}

type mcpTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Autonomy    *mcpToolAutonomy       `json:"autonomy,omitempty"`
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
// Defaults to agent mode (restricted) for security.
func NewMCPServer(butler *Butler) *MCPServer {
	return &MCPServer{
		butler: butler,
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
		mode:   MCPModeAgent, // Default to restricted mode
	}
}

// NewMCPServerWithMode creates a new MCP server with the specified mode.
func NewMCPServerWithMode(butler *Butler, mode MCPMode) *MCPServer {
	return &MCPServer{
		butler: butler,
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
		mode:   mode,
	}
}

// NewMCPServerWithIO creates a new MCP server with custom reader/writer (for testing).
func NewMCPServerWithIO(butler *Butler, mode MCPMode, reader *bufio.Reader, writer io.Writer) *MCPServer {
	return &MCPServer{
		butler: butler,
		reader: reader,
		writer: writer,
		mode:   mode,
	}
}

// Mode returns the current operational mode of the server.
func (s *MCPServer) Mode() MCPMode {
	return s.mode
}

// CurrentSessionID returns the active session ID for this connection.
func (s *MCPServer) CurrentSessionID() string {
	return s.currentSessionID
}

// SetCurrentSessionID sets the active session ID.
func (s *MCPServer) SetCurrentSessionID(sessionID string) {
	s.currentSessionID = sessionID
	s.autoSessionUsed = false
}

// IsAutoSessionUsed returns true if the current session was auto-created.
func (s *MCPServer) IsAutoSessionUsed() bool {
	return s.autoSessionUsed
}

// startSessionTimeoutChecker starts a background goroutine that checks for stale sessions.
func (s *MCPServer) startSessionTimeoutChecker() {
	cfg := s.butler.Config()
	if cfg == nil || cfg.Autonomy == nil || cfg.Autonomy.SessionTimeoutMinutes <= 0 {
		return // Timeout checking disabled
	}

	s.stopTimeout = make(chan struct{})
	timeout := time.Duration(cfg.Autonomy.SessionTimeoutMinutes) * time.Minute
	checkInterval := timeout / 6 // Check every 1/6 of timeout period (min 1 min)
	if checkInterval < time.Minute {
		checkInterval = time.Minute
	}

	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopTimeout:
				return
			case <-ticker.C:
				s.checkSessionTimeout(timeout)
			}
		}
	}()
}

// stopSessionTimeoutChecker stops the background timeout checker.
func (s *MCPServer) stopSessionTimeoutChecker() {
	if s.stopTimeout != nil {
		close(s.stopTimeout)
		s.stopTimeout = nil
	}
}

// checkSessionTimeout checks if the current session has timed out.
func (s *MCPServer) checkSessionTimeout(timeout time.Duration) {
	if s.currentSessionID == "" || s.lastActivity.IsZero() {
		return
	}

	if time.Since(s.lastActivity) > timeout {
		// Auto-end the session
		summary := "Auto-ended due to inactivity"
		if err := s.butler.EndSession(s.currentSessionID, "timeout", summary); err == nil {
			s.sessionAutoEndedMsg = fmt.Sprintf("⚠️ **Session Auto-Ended** (ID: `%s`)\n\nThe previous session was automatically ended due to inactivity (%d minutes).\nCall `session_init` to start a new session.\n\n---\n\n",
				s.currentSessionID, int(timeout.Minutes()))
			s.sessionAutoEnded = true
			s.currentSessionID = ""
			s.autoSessionUsed = false
		}
	}
}

// updateLastActivity records the time of the last tool call.
func (s *MCPServer) updateLastActivity() {
	s.lastActivity = time.Now()
}

// trackFileAccess records that a file was accessed by this session.
func (s *MCPServer) trackFileAccess(filePath string) {
	if filePath == "" {
		return
	}
	if s.trackedFiles == nil {
		s.trackedFiles = make(map[string]time.Time)
	}
	s.trackedFiles[filePath] = time.Now()
}

// checkFileConflicts checks if any tracked files have been modified by other agents.
// Returns a list of conflict warnings to show in the response.
func (s *MCPServer) checkFileConflicts() []string {
	// Check if conflict monitoring is enabled
	cfg := s.butler.Config()
	if cfg == nil || cfg.Autonomy == nil || !cfg.Autonomy.ConflictMonitoring {
		return nil
	}

	if len(s.trackedFiles) == 0 || s.currentSessionID == "" {
		return nil
	}

	var warnings []string
	mem := s.butler.Memory()

	for filePath, lastAccess := range s.trackedFiles {
		// Check if another agent has modified this file since we last accessed it
		conflict, err := mem.CheckConflict(filePath, s.currentSessionID)
		if err != nil || conflict == nil {
			continue
		}

		// Only warn if the conflict is newer than our last access
		if conflict.LastTouched.After(lastAccess) {
			warnings = append(warnings, fmt.Sprintf("⚠️ File `%s` was modified by **%s** since you last accessed it (at %s)",
				filePath, conflict.OtherAgent, conflict.LastTouched.Format("15:04:05")))
		}
	}

	return warnings
}

// formatConflictWarnings formats conflict warnings for inclusion in response.
func (s *MCPServer) formatConflictWarnings(warnings []string) string {
	if len(warnings) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## ⚠️ Conflict Warnings\n\n")
	for _, w := range warnings {
		sb.WriteString(w)
		sb.WriteString("\n")
	}
	sb.WriteString("\n---\n\n")
	return sb.String()
}

// checkBriefingUpdates checks for new learnings, decisions, or postmortems since last briefing.
// Returns a formatted update string or empty if no updates or rate limited.
func (s *MCPServer) checkBriefingUpdates() string {
	// Check if proactive briefing is enabled
	cfg := s.butler.Config()
	if cfg == nil || cfg.Autonomy == nil || !cfg.Autonomy.ProactiveBriefing {
		return ""
	}

	// Rate limit: max 1 update per 5 minutes
	if !s.lastBriefingTime.IsZero() && time.Since(s.lastBriefingTime) < 5*time.Minute {
		return ""
	}

	// If no session, no briefing updates
	if s.currentSessionID == "" {
		return ""
	}

	// Get counts of new items since last briefing
	mem := s.butler.Memory()
	since := s.lastBriefingTime
	if since.IsZero() {
		// First time - use session start or 1 hour ago
		since = time.Now().Add(-1 * time.Hour)
	}

	var updates []string

	// Check for new learnings
	if learnings, err := mem.GetLearningsSince(since); err == nil && len(learnings) > 0 {
		updates = append(updates, fmt.Sprintf("📚 **%d new learning(s)** added to the knowledge base", len(learnings)))
	}

	// Check for new decisions
	if decisions, err := mem.GetDecisionsSince(since, 10); err == nil && len(decisions) > 0 {
		updates = append(updates, fmt.Sprintf("📋 **%d new decision(s)** recorded", len(decisions)))
	}

	// Check for new postmortems
	if postmortems, err := mem.GetPostmortemsSince(since); err == nil && len(postmortems) > 0 {
		updates = append(updates, fmt.Sprintf("🔥 **%d new postmortem(s)** - review for critical learnings!", len(postmortems)))
	}

	if len(updates) == 0 {
		return ""
	}

	// Update last briefing time
	s.lastBriefingTime = time.Now()

	var sb strings.Builder
	sb.WriteString("## 📰 Context Updates\n\n")
	for _, u := range updates {
		sb.WriteString("- ")
		sb.WriteString(u)
		sb.WriteString("\n")
	}
	sb.WriteString("\nUse `recall` or `recall_decisions` to explore these updates.\n\n---\n\n")
	return sb.String()
}

// cleanupOnDisconnect ends any active session when the MCP connection is closed.
func (s *MCPServer) cleanupOnDisconnect() {
	if s.currentSessionID == "" {
		return
	}

	// End the session with "disconnected" outcome
	summary := "MCP connection closed"
	if err := s.butler.EndSession(s.currentSessionID, "disconnected", summary); err != nil {
		// Log the error but don't fail - we're already shutting down
		fmt.Fprintf(os.Stderr, "Warning: failed to cleanup session %s on disconnect: %v\n", s.currentSessionID, err)
		return
	}

	s.currentSessionID = ""
	s.autoSessionUsed = false
}

// ensureSession auto-creates a session if needed and returns the session ID.
// Returns empty string if auto-session is disabled or if creation fails.
func (s *MCPServer) ensureSession(agentName string) (string, bool, error) {
	// If we already have a session, return it
	if s.currentSessionID != "" {
		return s.currentSessionID, false, nil
	}

	// Check if auto-session is enabled
	cfg := s.butler.Config()
	if cfg == nil || cfg.Autonomy == nil || !cfg.Autonomy.AutoSession {
		return "", false, nil
	}

	// Auto-create session
	if agentName == "" {
		agentName = "unknown"
	}
	// StartSession takes (agentType, agentID, goal)
	agentID := fmt.Sprintf("auto-%s-%d", agentName, time.Now().UnixNano())
	session, err := s.butler.Memory().StartSession(agentName, agentID, "auto-created session")
	if err != nil {
		return "", false, err
	}

	s.currentSessionID = session.ID
	s.autoSessionUsed = true
	return session.ID, true, nil
}

// Serve runs the MCP server, reading JSON-RPC requests from stdin.
func (s *MCPServer) Serve() error {
	// Start session timeout checker (if configured)
	s.startSessionTimeoutChecker()
	defer s.stopSessionTimeoutChecker()
	defer s.cleanupOnDisconnect() // Cleanup sessions on exit

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

// handleToolsList returns the list of available tools filtered by mode.
func (s *MCPServer) handleToolsList(req jsonRPCRequest) jsonRPCResponse {
	allTools := buildToolsList()

	// Filter tools based on mode
	var tools []mcpTool
	for _, tool := range allTools {
		// Skip admin-only tools in agent mode
		if s.mode == MCPModeAgent && IsAdminOnlyTool(tool.Name) {
			continue
		}
		tools = append(tools, tool)
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]interface{}{"tools": tools},
	}
}

// toolsRequiringSession lists tools that benefit from session tracking.
// These are tools that perform actions (not just queries) and need context.
var toolsRequiringSession = map[string]bool{
	"file_context":        true,
	"store":               true,
	"store_direct":        true,
	"recall_outcome":      true,
	"recall_link":         true,
	"recall_unlink":       true,
	"recall_obsolete":     true,
	"recall_archive":      true,
	"session_log":         true,
	"context_auto_inject": true,
}

// toolsCreatingSession lists tools that create their own sessions.
var toolsCreatingSession = map[string]bool{
	"session_init":  true,
	"session_start": true,
}

// autoActivityMapping maps tool names to their auto-log activity kinds.
var autoActivityMapping = map[string]struct {
	kind   string
	target func(args map[string]interface{}) string
}{
	"file_context": {
		kind:   "file_focus",
		target: func(args map[string]interface{}) string { v, _ := args["file_path"].(string); return v },
	},
	"context_auto_inject": {
		kind:   "file_focus",
		target: func(args map[string]interface{}) string { v, _ := args["file_path"].(string); return v },
	},
	"explore": {
		kind:   "search",
		target: func(args map[string]interface{}) string { v, _ := args["query"].(string); return v },
	},
	"explore_context": {
		kind:   "search",
		target: func(args map[string]interface{}) string { v, _ := args["task"].(string); return v },
	},
	"store": {
		kind:   "knowledge_create",
		target: func(args map[string]interface{}) string { v, _ := args["as"].(string); return v },
	},
	"recall": {
		kind:   "knowledge_query",
		target: func(args map[string]interface{}) string { v, _ := args["query"].(string); return v },
	},
	"recall_decisions": {
		kind:   "knowledge_query",
		target: func(args map[string]interface{}) string { v, _ := args["query"].(string); return v },
	},
}

// handleToolsCall dispatches a tool call to the appropriate handler.
func (s *MCPServer) handleToolsCall(req jsonRPCRequest) jsonRPCResponse {
	// Update activity timestamp for session timeout tracking
	s.updateLastActivity()

	var params mcpToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32602, Message: "Invalid params", Data: err.Error()},
		}
	}

	// Enforce mode restrictions - reject admin-only tools in agent mode
	if s.mode == MCPModeAgent && IsAdminOnlyTool(params.Name) {
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32602, Message: fmt.Sprintf("Tool %q not available in agent mode", params.Name)},
		}
	}

	// Check if session was auto-ended due to timeout and capture the message
	var sessionTimeoutMsg string
	if s.sessionAutoEnded {
		sessionTimeoutMsg = s.sessionAutoEndedMsg
		s.sessionAutoEnded = false
		s.sessionAutoEndedMsg = ""
	}

	// Auto-session: create session if needed for tools that benefit from tracking
	var autoSessionWarning string
	if toolsRequiringSession[params.Name] && !toolsCreatingSession[params.Name] {
		agentName := "unknown"
		if name, ok := params.Arguments["agent_name"].(string); ok && name != "" {
			agentName = name
		} else if name, ok := params.Arguments["agentType"].(string); ok && name != "" {
			agentName = name
		}
		sessionID, wasCreated, err := s.ensureSession(agentName)
		if err != nil {
			// Log but don't fail - auto-session is a convenience feature
			autoSessionWarning = fmt.Sprintf("⚠️ Warning: Could not auto-create session: %v\n\n", err)
		} else if wasCreated {
			autoSessionWarning = fmt.Sprintf("⚠️ **Auto-Session Created** (ID: `%s`)\n\nFor better tracking, call `session_init` at the start of your work.\n\n---\n\n", sessionID)
		}
	}

	// Proactive conflict monitoring: check for conflicts on tracked files
	conflictWarnings := s.checkFileConflicts()

	// Dispatch to tool handler
	resp := s.dispatchTool(req.ID, params)

	// Track file access for file-related tools (after successful dispatch)
	if resp.Error == nil {
		if filePath := s.extractFilePath(params.Name, params.Arguments); filePath != "" {
			s.trackFileAccess(filePath)
		}
	}

	// Auto-activity logging: log activities transparently
	s.autoLogActivity(params.Name, params.Arguments, resp.Error == nil)

	// Proactive briefing: check for knowledge updates
	briefingUpdates := s.checkBriefingUpdates()

	// Prepend any notification messages to the response
	if resp.Error == nil {
		if result, ok := resp.Result.(mcpToolResult); ok && len(result.Content) > 0 {
			var prefix strings.Builder
			if sessionTimeoutMsg != "" {
				prefix.WriteString(sessionTimeoutMsg)
			}
			if conflictWarningsStr := s.formatConflictWarnings(conflictWarnings); conflictWarningsStr != "" {
				prefix.WriteString(conflictWarningsStr)
			}
			if briefingUpdates != "" {
				prefix.WriteString(briefingUpdates)
			}
			if autoSessionWarning != "" {
				prefix.WriteString(autoSessionWarning)
			}
			if prefix.Len() > 0 {
				result.Content[0].Text = prefix.String() + result.Content[0].Text
				resp.Result = result
			}
		}
	}

	return resp
}

// extractFilePath extracts a file path from tool arguments if applicable.
func (s *MCPServer) extractFilePath(toolName string, args map[string]interface{}) string {
	switch toolName {
	case "file_context", "context_auto_inject", "brief_file":
		if path, ok := args["file_path"].(string); ok {
			return path
		}
		if path, ok := args["path"].(string); ok {
			return path
		}
	case "explore_file", "explore_impact":
		if path, ok := args["file"].(string); ok {
			return path
		}
		if path, ok := args["target"].(string); ok {
			return path
		}
	}
	return ""
}

// autoLogActivity logs an activity if auto-logging is enabled for this tool.
func (s *MCPServer) autoLogActivity(toolName string, args map[string]interface{}, success bool) {
	// Check if we have a session to log to
	if s.currentSessionID == "" {
		return
	}

	// Check if auto-logging is enabled
	cfg := s.butler.Config()
	if cfg == nil || cfg.Autonomy == nil || !cfg.Autonomy.AutoActivityLog {
		return
	}

	// Check if this tool has auto-logging mapping
	mapping, ok := autoActivityMapping[toolName]
	if !ok {
		return
	}

	// Get target from the mapping function
	target := mapping.target(args)
	if target == "" {
		target = toolName // fallback to tool name
	}

	// Determine outcome
	outcome := "success"
	if !success {
		outcome = "failure"
	}

	// Log the activity (fire and forget - don't affect the response)
	act := memory.Activity{
		Kind:    mapping.kind,
		Target:  target,
		Outcome: outcome,
	}
	_ = s.butler.Memory().LogActivity(s.currentSessionID, act)
}

// dispatchTool routes the tool call to the appropriate handler.
func (s *MCPServer) dispatchTool(id any, params mcpToolCallParams) jsonRPCResponse {
	switch params.Name {
	// Composite tools - streamlined workflows
	case "session_init":
		return s.toolSessionInit(id, params.Arguments)
	case "file_context":
		return s.toolFileContext(id, params.Arguments)

	// Explore tools - search, context, symbols, graphs
	case "explore":
		return s.toolExplore(id, params.Arguments)
	case "explore_rooms":
		return s.toolExploreRooms(id)
	case "explore_context":
		return s.toolExploreContext(id, params.Arguments)
	case "explore_impact":
		return s.toolExploreImpact(id, params.Arguments)
	case "explore_symbols":
		return s.toolExploreSymbols(id, params.Arguments)
	case "explore_symbol":
		return s.toolExploreSymbol(id, params.Arguments)
	case "explore_file":
		return s.toolExploreFile(id, params.Arguments)
	case "explore_deps":
		return s.toolExploreDeps(id, params.Arguments)
	case "explore_callers":
		return s.toolExploreCallers(id, params.Arguments)
	case "explore_callees":
		return s.toolExploreCallees(id, params.Arguments)
	case "explore_graph":
		return s.toolExploreGraph(id, params.Arguments)
	case "get_route":
		return s.toolGetRoute(id, params.Arguments)

	// Store tools - store ideas, decisions, learnings
	case "store":
		return s.toolStore(id, params.Arguments)
	case "store_direct":
		return s.toolStoreDirect(id, params.Arguments)

	// Governance tools - approve/reject proposals (human mode only)
	case "approve":
		return s.toolApprove(id, params.Arguments)
	case "reject":
		return s.toolReject(id, params.Arguments)

	// Recall tools - retrieve knowledge and manage relationships
	case "recall":
		return s.toolRecall(id, params.Arguments)
	case "recall_decisions":
		return s.toolRecallDecisions(id, params.Arguments)
	case "recall_ideas":
		return s.toolRecallIdeas(id, params.Arguments)
	case "recall_outcome":
		return s.toolRecallOutcome(id, params.Arguments)
	case "recall_link":
		return s.toolRecallLink(id, params.Arguments)
	case "recall_links":
		return s.toolRecallLinks(id, params.Arguments)
	case "recall_unlink":
		return s.toolRecallUnlink(id, params.Arguments)

	// Brief tools - get briefings and file intel
	case "brief":
		return s.toolBrief(id, params.Arguments)
	case "brief_file":
		return s.toolBriefFile(id, params.Arguments)
	case "briefing_smart":
		return s.toolBriefingSmart(id, params.Arguments)

	// Session tools - manage agent sessions
	case "session_start":
		return s.toolSessionStart(id, params.Arguments)
	case "session_log":
		return s.toolSessionLog(id, params.Arguments)
	case "session_end":
		return s.toolSessionEnd(id, params.Arguments)
	case "session_conflict":
		return s.toolSessionConflict(id, params.Arguments)
	case "session_list":
		return s.toolSessionList(id, params.Arguments)
	case "session_resume":
		return s.toolSessionResume(id, params.Arguments)
	case "session_status":
		return s.toolSessionStatus(id, params.Arguments)

	// Handoff tools - multi-agent task handoff
	case "handoff_create":
		return s.toolHandoffCreate(id, params.Arguments)
	case "handoff_list":
		return s.toolHandoffList(id, params.Arguments)
	case "handoff_accept":
		return s.toolHandoffAccept(id, params.Arguments)
	case "handoff_complete":
		return s.toolHandoffComplete(id, params.Arguments)

	// Conversation tools - store and search conversations
	case "conversation_store":
		return s.toolConversationStore(id, params.Arguments)
	case "conversation_search":
		return s.toolConversationSearch(id, params.Arguments)
	case "conversation_extract":
		return s.toolConversationExtract(id, params.Arguments)

	// Corridor tools - personal cross-workspace learnings
	case "corridor_learnings":
		return s.toolCorridorLearnings(id, params.Arguments)
	case "corridor_links":
		return s.toolCorridorLinks(id, params.Arguments)
	case "corridor_stats":
		return s.toolCorridorStats(id, params.Arguments)
	case "corridor_promote":
		return s.toolCorridorPromote(id, params.Arguments)
	case "corridor_reinforce":
		return s.toolCorridorReinforce(id, params.Arguments)

	// Semantic Search Tools
	case "search_semantic":
		return s.toolSearchSemantic(id, params.Arguments)
	case "search_hybrid":
		return s.toolSearchHybrid(id, params.Arguments)
	case "search_similar":
		return s.toolSearchSimilar(id, params.Arguments)

	// Embedding Management Tools
	case "embedding_sync":
		return s.toolEmbeddingSync(id, params.Arguments)
	case "embedding_stats":
		return s.toolEmbeddingStats(id, params.Arguments)

	// Learning Lifecycle Tools
	case "recall_learning_link":
		return s.toolRecallLearningLink(id, params.Arguments)
	case "recall_obsolete":
		return s.toolRecallObsolete(id, params.Arguments)
	case "recall_archive":
		return s.toolRecallArchive(id, params.Arguments)
	case "recall_learnings_by_status":
		return s.toolRecallLearningsByStatus(id, params.Arguments)

	// Contradiction Detection Tools
	case "recall_contradictions":
		return s.toolRecallContradictions(id, params.Arguments)
	case "recall_contradiction_check":
		return s.toolRecallContradictionCheck(id, params.Arguments)
	case "recall_contradiction_summary":
		return s.toolRecallContradictionSummary(id, params.Arguments)

	// Decay Tools
	case "decay_stats":
		return s.toolDecayStats(id, params.Arguments)
	case "decay_preview":
		return s.toolDecayPreview(id, params.Arguments)
	case "decay_apply":
		return s.toolDecayApply(id, params.Arguments)
	case "decay_reinforce":
		return s.toolDecayReinforce(id, params.Arguments)
	case "decay_boost":
		return s.toolDecayBoost(id, params.Arguments)

	// Context & Scope Tools
	case "context_auto_inject":
		return s.toolContextAutoInject(id, params.Arguments)
	case "context_focus":
		return s.toolContextFocus(id, params.Arguments)
	case "context_get":
		return s.toolContextGet(id, params.Arguments)
	case "context_pin":
		return s.toolContextPin(id, params.Arguments)
	case "scope_explain":
		return s.toolScopeExplain(id, params.Arguments)

	// Postmortem Tools
	case "store_postmortem":
		return s.toolStorePostmortem(id, params.Arguments)
	case "get_postmortems":
		return s.toolGetPostmortems(id, params.Arguments)
	case "get_postmortem":
		return s.toolGetPostmortem(id, params.Arguments)
	case "resolve_postmortem":
		return s.toolResolvePostmortem(id, params.Arguments)
	case "postmortem_stats":
		return s.toolPostmortemStats(id, params.Arguments)
	case "postmortem_to_learnings":
		return s.toolPostmortemToLearnings(id, params.Arguments)

	// Analytics tools - workspace insights
	case "analytics_sessions":
		return s.toolSessionAnalytics(id, params.Arguments)
	case "analytics_learnings":
		return s.toolLearningEffectiveness(id, params.Arguments)
	case "analytics_health":
		return s.toolWorkspaceHealth(id, params.Arguments)

	// Pattern tools - detected code pattern management
	case "patterns_get":
		return s.toolPatternsGet(id, params.Arguments)
	case "pattern_show":
		return s.toolPatternShow(id, params.Arguments)
	case "pattern_approve":
		return s.toolPatternApprove(id, params.Arguments)
	case "pattern_ignore":
		return s.toolPatternIgnore(id, params.Arguments)
	case "pattern_stats":
		return s.toolPatternStats(id, params.Arguments)

	// Contract tools - FE-BE API contract management
	case "contracts_get":
		return s.toolContractsGet(id, params.Arguments)
	case "contract_show":
		return s.toolContractShow(id, params.Arguments)
	case "contract_verify":
		return s.toolContractVerify(id, params.Arguments)
	case "contract_ignore":
		return s.toolContractIgnore(id, params.Arguments)
	case "contract_stats":
		return s.toolContractStats(id, params.Arguments)
	case "contract_mismatches":
		return s.toolContractMismatches(id, params.Arguments)

	default:
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
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
	rooms := s.butler.ListRooms()
	for i := range rooms {
		room := &rooms[i]
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
