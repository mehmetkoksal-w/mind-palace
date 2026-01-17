package analysis

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// LSPClient is a generic Language Server Protocol client that can communicate
// with any LSP-compliant language server via stdio.
type LSPClient struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	requestID int64
	responses map[int64]chan json.RawMessage
	mu        sync.Mutex
	rootPath  string
	ready     bool
	ctx       context.Context
	cancel    context.CancelFunc
	timeout   time.Duration
}

// LSPClientConfig holds configuration for creating an LSP client
type LSPClientConfig struct {
	ServerCmd  string        // Command to execute (e.g., "gopls", "typescript-language-server")
	ServerArgs []string      // Arguments to pass (e.g., ["-mode=stdio"])
	RootPath   string        // Workspace root path
	LanguageID string        // Language identifier (e.g., "go", "typescript")
	Timeout    time.Duration // Request timeout (default: 5s)
}

// LSPSymbolKind represents the kind of a symbol (LSP document symbol types).
type LSPSymbolKind int

const (
	LSPSymbolKindFile          LSPSymbolKind = 1
	LSPSymbolKindModule        LSPSymbolKind = 2
	LSPSymbolKindNamespace     LSPSymbolKind = 3
	LSPSymbolKindPackage       LSPSymbolKind = 4
	LSPSymbolKindClass         LSPSymbolKind = 5
	LSPSymbolKindMethod        LSPSymbolKind = 6
	LSPSymbolKindProperty      LSPSymbolKind = 7
	LSPSymbolKindField         LSPSymbolKind = 8
	LSPSymbolKindConstructor   LSPSymbolKind = 9
	LSPSymbolKindEnum          LSPSymbolKind = 10
	LSPSymbolKindInterface     LSPSymbolKind = 11
	LSPSymbolKindFunction      LSPSymbolKind = 12
	LSPSymbolKindVariable      LSPSymbolKind = 13
	LSPSymbolKindConstant      LSPSymbolKind = 14
	LSPSymbolKindString        LSPSymbolKind = 15
	LSPSymbolKindNumber        LSPSymbolKind = 16
	LSPSymbolKindBoolean       LSPSymbolKind = 17
	LSPSymbolKindArray         LSPSymbolKind = 18
	LSPSymbolKindObject        LSPSymbolKind = 19
	LSPSymbolKindKey           LSPSymbolKind = 20
	LSPSymbolKindNull          LSPSymbolKind = 21
	LSPSymbolKindEnumMember    LSPSymbolKind = 22
	LSPSymbolKindStruct        LSPSymbolKind = 23
	LSPSymbolKindEvent         LSPSymbolKind = 24
	LSPSymbolKindOperator      LSPSymbolKind = 25
	LSPSymbolKindTypeParameter LSPSymbolKind = 26
)

type LSPPosition struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type LSPRange struct {
	Start LSPPosition `json:"start"`
	End   LSPPosition `json:"end"`
}

type LSPLocation struct {
	URI   string   `json:"uri"`
	Range LSPRange `json:"range"`
}

type LSPDocumentSymbol struct {
	Name           string              `json:"name"`
	Detail         string              `json:"detail,omitempty"`
	Kind           LSPSymbolKind       `json:"kind"`
	Deprecated     bool                `json:"deprecated,omitempty"`
	Range          LSPRange            `json:"range"`
	SelectionRange LSPRange            `json:"selectionRange"`
	Children       []LSPDocumentSymbol `json:"children,omitempty"`
}

// NewLSPClient creates a new generic LSP client
func NewLSPClient(config LSPClientConfig) (*LSPClient, error) {
	// Validate config
	if config.ServerCmd == "" {
		return nil, fmt.Errorf("server command is required")
	}
	if config.RootPath == "" {
		return nil, fmt.Errorf("root path is required")
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}

	// Check if server is available
	serverPath, err := exec.LookPath(config.ServerCmd)
	if err != nil {
		return nil, fmt.Errorf("%s not found in PATH: %w", config.ServerCmd, err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Build command
	cmd := exec.CommandContext(ctx, serverPath, config.ServerArgs...)
	cmd.Dir = config.RootPath

	// Setup pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create stderr pipe: %w", err)
	}

	client := &LSPClient{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		stderr:    stderr,
		responses: make(map[int64]chan json.RawMessage),
		rootPath:  config.RootPath,
		ctx:       ctx,
		cancel:    cancel,
		timeout:   config.Timeout,
	}

	// Start the server process
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start %s: %w", config.ServerCmd, err)
	}

	// Start response reader
	go client.readResponses()

	// Discard stderr to prevent blocking (can be logged in production)
	go io.Copy(io.Discard, stderr)

	// Initialize the server
	if err := client.initialize(config.LanguageID); err != nil {
		client.Close()
		return nil, fmt.Errorf("initialize LSP: %w", err)
	}

	return client, nil
}

// Close shuts down the LSP server gracefully
func (c *LSPClient) Close() error {
	if c.cancel != nil {
		c.cancel()
	}

	// Send shutdown and exit notifications
	c.sendRequest("shutdown", nil)
	c.sendNotification("exit", nil)

	// Close pipes
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.stdout != nil {
		c.stdout.Close()
	}
	if c.stderr != nil {
		c.stderr.Close()
	}

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		if c.cmd != nil && c.cmd.Process != nil {
			done <- c.cmd.Wait()
		} else {
			done <- nil
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		if c.cmd != nil && c.cmd.Process != nil {
			c.cmd.Process.Kill()
		}
	}

	return nil
}

// initialize sends the LSP initialize request
func (c *LSPClient) initialize(_ string) error {
	absPath, err := filepath.Abs(c.rootPath)
	if err != nil {
		absPath = c.rootPath
	}
	rootURI := pathToURI(absPath)

	params := map[string]interface{}{
		"processId": os.Getpid(),
		"rootUri":   rootURI,
		"capabilities": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"documentSymbol": map[string]interface{}{
					"dynamicRegistration":               false,
					"hierarchicalDocumentSymbolSupport": true,
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	_, err = c.sendRequestWithContext(ctx, "initialize", params)
	if err != nil {
		return fmt.Errorf("initialize request: %w", err)
	}

	// Send initialized notification
	c.sendNotification("initialized", map[string]interface{}{})

	c.ready = true
	return nil
}

// DocumentSymbols retrieves document symbols for a given file
func (c *LSPClient) DocumentSymbols(uri, content string) ([]LSPDocumentSymbol, error) {
	if !c.ready {
		return nil, fmt.Errorf("LSP client not ready")
	}

	// Open the document
	if err := c.openDocument(uri, content); err != nil {
		return nil, fmt.Errorf("open document: %w", err)
	}
	defer c.closeDocument(uri)

	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": uri,
		},
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
	defer cancel()

	result, err := c.sendRequestWithContext(ctx, "textDocument/documentSymbol", params)
	if err != nil {
		return nil, fmt.Errorf("document symbol request: %w", err)
	}

	if result == nil || string(result) == "null" {
		return []LSPDocumentSymbol{}, nil
	}

	var symbols []LSPDocumentSymbol
	if err := json.Unmarshal(result, &symbols); err != nil {
		return nil, fmt.Errorf("unmarshal document symbols: %w", err)
	}

	return symbols, nil
}

// openDocument notifies the server about an opened file
func (c *LSPClient) openDocument(uri, content string) error {
	// Extract language ID from URI (basic heuristic)
	var languageID string
	switch {
	case strings.HasSuffix(uri, ".go"):
		languageID = "go"
	case strings.HasSuffix(uri, ".ts") || strings.HasSuffix(uri, ".tsx"):
		languageID = "typescript"
	case strings.HasSuffix(uri, ".js") || strings.HasSuffix(uri, ".jsx"):
		languageID = "javascript"
	case strings.HasSuffix(uri, ".py"):
		languageID = "python"
	case strings.HasSuffix(uri, ".rs"):
		languageID = "rust"
	default:
		languageID = "plaintext"
	}

	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        uri,
			"languageId": languageID,
			"version":    1,
			"text":       content,
		},
	}

	return c.sendNotification("textDocument/didOpen", params)
}

// closeDocument notifies the server about a closed file
func (c *LSPClient) closeDocument(uri string) error {
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": uri,
		},
	}

	return c.sendNotification("textDocument/didClose", params)
}

// readResponses continuously reads LSP responses from stdout
func (c *LSPClient) readResponses() {
	reader := bufio.NewReader(c.stdout)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		// Read Content-Length header
		var contentLength int
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break // End of headers
			}
			if strings.HasPrefix(line, "Content-Length:") {
				lenStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
				contentLength, _ = strconv.Atoi(lenStr)
			}
		}

		if contentLength == 0 {
			continue
		}

		// Read message body
		body := make([]byte, contentLength)
		_, err := io.ReadFull(reader, body)
		if err != nil {
			return
		}

		// Parse response
		var resp lspResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			continue
		}

		// Check for error in response
		if resp.Error != nil {
			// Log error but continue (in production, use proper logging)
			continue
		}

		// Route response to waiting request
		if resp.ID != 0 {
			c.mu.Lock()
			if ch, ok := c.responses[resp.ID]; ok {
				select {
				case ch <- resp.Result:
				default:
				}
				delete(c.responses, resp.ID)
			}
			c.mu.Unlock()
		}
	}
}

// sendRequest sends an LSP request and waits for response
func (c *LSPClient) sendRequest(method string, params interface{}) (json.RawMessage, error) {
	return c.sendRequestWithContext(c.ctx, method, params)
}

// sendRequestWithContext sends an LSP request with context
func (c *LSPClient) sendRequestWithContext(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	id := atomic.AddInt64(&c.requestID, 1)

	req := lspRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create response channel
	respChan := make(chan json.RawMessage, 1)
	c.mu.Lock()
	c.responses[id] = respChan
	c.mu.Unlock()

	// Send request with Content-Length header
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err = c.stdin.Write([]byte(header)); err != nil {
		c.mu.Lock()
		delete(c.responses, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("write header: %w", err)
	}
	if _, err = c.stdin.Write(body); err != nil {
		c.mu.Lock()
		delete(c.responses, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("write body: %w", err)
	}

	// Wait for response
	select {
	case result := <-respChan:
		return result, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.responses, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("request timeout: %w", ctx.Err())
	}
}

// sendNotification sends an LSP notification (no response expected)
func (c *LSPClient) sendNotification(method string, params interface{}) error {
	req := lspRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err = c.stdin.Write([]byte(header)); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	if _, err = c.stdin.Write(body); err != nil {
		return fmt.Errorf("write body: %w", err)
	}

	return nil
}

// ConvertLSPSymbolKind converts LSP SymbolKind to our internal SymbolKind
func ConvertLSPSymbolKind(kind LSPSymbolKind) SymbolKind {
	switch kind {
	case LSPSymbolKindClass, LSPSymbolKindStruct:
		return KindClass
	case LSPSymbolKindInterface:
		return KindInterface
	case LSPSymbolKindFunction:
		return KindFunction
	case LSPSymbolKindMethod:
		return KindMethod
	case LSPSymbolKindVariable:
		return KindVariable
	case LSPSymbolKindConstant:
		return KindConstant
	case LSPSymbolKindEnum:
		return KindEnum
	case LSPSymbolKindProperty, LSPSymbolKindField:
		return KindProperty
	case LSPSymbolKindConstructor:
		return KindConstructor
	default:
		return KindVariable // Default fallback
	}
}
