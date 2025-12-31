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

// DartLSPClient communicates with the Dart Analysis Server via LSP protocol
type DartLSPClient struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	requestID int64
	responses map[int64]chan json.RawMessage
	mu        sync.Mutex
	rootPath  string
	ready     bool
	ctx       context.Context
	cancel    context.CancelFunc
}

// LSP message types
type lspRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type lspResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *lspError       `json:"error,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type lspError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// LSP types for call hierarchy
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type CallHierarchyItem struct {
	Name           string `json:"name"`
	Kind           int    `json:"kind"`
	Detail         string `json:"detail,omitempty"`
	URI            string `json:"uri"`
	Range          Range  `json:"range"`
	SelectionRange Range  `json:"selectionRange"`
	Data           any    `json:"data,omitempty"`
}

type CallHierarchyIncomingCall struct {
	From       CallHierarchyItem `json:"from"`
	FromRanges []Range           `json:"fromRanges"`
}

type CallHierarchyOutgoingCall struct {
	To         CallHierarchyItem `json:"to"`
	FromRanges []Range           `json:"fromRanges"`
}

// CallInfo represents a simplified call relationship for storage
type CallInfo struct {
	CallerFile   string
	CallerSymbol string
	CallerLine   int
	CalleeFile   string
	CalleeSymbol string
	CalleeLine   int
}

// NewDartLSPClient creates a new Dart LSP client
func NewDartLSPClient(rootPath string) (*DartLSPClient, error) {
	// Find dart executable
	dartPath, err := exec.LookPath("dart")
	if err != nil {
		return nil, fmt.Errorf("dart not found in PATH: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, dartPath, "language-server", "--client-id=mind-palace", "--client-version=1.0.0")
	cmd.Dir = rootPath

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

	// Discard stderr to prevent blocking
	cmd.Stderr = nil

	client := &DartLSPClient{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		responses: make(map[int64]chan json.RawMessage),
		rootPath:  rootPath,
		ctx:       ctx,
		cancel:    cancel,
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start dart language-server: %w", err)
	}

	// Start response reader
	go client.readResponses()

	// Initialize the server
	if err := client.initialize(); err != nil {
		client.Close()
		return nil, fmt.Errorf("initialize LSP: %w", err)
	}

	return client, nil
}

// Close shuts down the LSP server
func (c *DartLSPClient) Close() error {
	c.cancel()

	// Send shutdown request
	c.sendRequest("shutdown", nil)
	c.sendNotification("exit", nil)

	c.stdin.Close()
	c.stdout.Close()

	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		done <- c.cmd.Wait()
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		c.cmd.Process.Kill()
	}

	return nil
}

func (c *DartLSPClient) initialize() error {
	rootURI := "file://" + c.rootPath

	params := map[string]interface{}{
		"processId": os.Getpid(),
		"rootUri":   rootURI,
		"capabilities": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"callHierarchy": map[string]interface{}{
					"dynamicRegistration": false,
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	_, err := c.sendRequestWithContext(ctx, "initialize", params)
	if err != nil {
		return fmt.Errorf("initialize request: %w", err)
	}

	// Send initialized notification
	c.sendNotification("initialized", map[string]interface{}{})

	c.ready = true
	return nil
}

func (c *DartLSPClient) readResponses() {
	reader := bufio.NewReader(c.stdout)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		// Read headers
		var contentLength int
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			if strings.HasPrefix(line, "Content-Length:") {
				lenStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
				contentLength, _ = strconv.Atoi(lenStr)
			}
		}

		if contentLength == 0 {
			continue
		}

		// Read body
		body := make([]byte, contentLength)
		_, err := io.ReadFull(reader, body)
		if err != nil {
			return
		}

		var resp lspResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			continue
		}

		// Route response to waiting request
		if resp.ID != 0 {
			c.mu.Lock()
			if ch, ok := c.responses[resp.ID]; ok {
				ch <- resp.Result
				delete(c.responses, resp.ID)
			}
			c.mu.Unlock()
		}
	}
}

func (c *DartLSPClient) sendRequest(method string, params interface{}) (json.RawMessage, error) {
	return c.sendRequestWithContext(c.ctx, method, params)
}

func (c *DartLSPClient) sendRequestWithContext(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	id := atomic.AddInt64(&c.requestID, 1)

	req := lspRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Create response channel
	respChan := make(chan json.RawMessage, 1)
	c.mu.Lock()
	c.responses[id] = respChan
	c.mu.Unlock()

	// Send with Content-Length header
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	_, err = c.stdin.Write([]byte(header))
	if err != nil {
		return nil, err
	}
	_, err = c.stdin.Write(body)
	if err != nil {
		return nil, err
	}

	// Wait for response
	select {
	case result := <-respChan:
		return result, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.responses, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	}
}

func (c *DartLSPClient) sendNotification(method string, params interface{}) error {
	req := lspRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	_, err = c.stdin.Write([]byte(header))
	if err != nil {
		return err
	}
	_, err = c.stdin.Write(body)
	return err
}

// OpenFile notifies the server about an open file
func (c *DartLSPClient) OpenFile(filePath string, content string) error {
	uri := pathToURI(filePath)

	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        uri,
			"languageId": "dart",
			"version":    1,
			"text":       content,
		},
	}

	return c.sendNotification("textDocument/didOpen", params)
}

// CloseFile notifies the server about a closed file
func (c *DartLSPClient) CloseFile(filePath string) error {
	uri := pathToURI(filePath)

	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": uri,
		},
	}

	return c.sendNotification("textDocument/didClose", params)
}

// PrepareCallHierarchy gets call hierarchy items at a position
func (c *DartLSPClient) PrepareCallHierarchy(filePath string, line, character int) ([]CallHierarchyItem, error) {
	uri := pathToURI(filePath)

	params := TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     Position{Line: line, Character: character},
	}

	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	result, err := c.sendRequestWithContext(ctx, "textDocument/prepareCallHierarchy", params)
	if err != nil {
		return nil, err
	}

	if result == nil || string(result) == "null" {
		return nil, nil
	}

	var items []CallHierarchyItem
	if err := json.Unmarshal(result, &items); err != nil {
		return nil, fmt.Errorf("unmarshal call hierarchy items: %w", err)
	}

	return items, nil
}

// GetIncomingCalls gets all callers of a call hierarchy item
func (c *DartLSPClient) GetIncomingCalls(item CallHierarchyItem) ([]CallHierarchyIncomingCall, error) {
	params := map[string]interface{}{
		"item": item,
	}

	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	result, err := c.sendRequestWithContext(ctx, "callHierarchy/incomingCalls", params)
	if err != nil {
		return nil, err
	}

	if result == nil || string(result) == "null" {
		return nil, nil
	}

	var calls []CallHierarchyIncomingCall
	if err := json.Unmarshal(result, &calls); err != nil {
		return nil, fmt.Errorf("unmarshal incoming calls: %w", err)
	}

	return calls, nil
}

// GetOutgoingCalls gets all callees from a call hierarchy item
func (c *DartLSPClient) GetOutgoingCalls(item CallHierarchyItem) ([]CallHierarchyOutgoingCall, error) {
	params := map[string]interface{}{
		"item": item,
	}

	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	result, err := c.sendRequestWithContext(ctx, "callHierarchy/outgoingCalls", params)
	if err != nil {
		return nil, err
	}

	if result == nil || string(result) == "null" {
		return nil, nil
	}

	var calls []CallHierarchyOutgoingCall
	if err := json.Unmarshal(result, &calls); err != nil {
		return nil, fmt.Errorf("unmarshal outgoing calls: %w", err)
	}

	return calls, nil
}

// ExtractCallsForSymbol extracts all call relationships for a symbol at a position
func (c *DartLSPClient) ExtractCallsForSymbol(filePath string, line, character int) ([]CallInfo, error) {
	items, err := c.PrepareCallHierarchy(filePath, line, character)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, nil
	}

	var calls []CallInfo
	item := items[0]

	// Get incoming calls (who calls this)
	incoming, err := c.GetIncomingCalls(item)
	if err == nil {
		for _, call := range incoming {
			for _, r := range call.FromRanges {
				calls = append(calls, CallInfo{
					CallerFile:   uriToPath(call.From.URI),
					CallerSymbol: call.From.Name,
					CallerLine:   r.Start.Line + 1, // Convert to 1-based
					CalleeFile:   uriToPath(item.URI),
					CalleeSymbol: item.Name,
					CalleeLine:   item.SelectionRange.Start.Line + 1,
				})
			}
		}
	}

	// Get outgoing calls (what this calls)
	outgoing, err := c.GetOutgoingCalls(item)
	if err == nil {
		for _, call := range outgoing {
			for _, r := range call.FromRanges {
				calls = append(calls, CallInfo{
					CallerFile:   uriToPath(item.URI),
					CallerSymbol: item.Name,
					CallerLine:   r.Start.Line + 1,
					CalleeFile:   uriToPath(call.To.URI),
					CalleeSymbol: call.To.Name,
					CalleeLine:   call.To.SelectionRange.Start.Line + 1,
				})
			}
		}
	}

	return calls, nil
}

func pathToURI(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	return "file://" + absPath
}

func uriToPath(uri string) string {
	if strings.HasPrefix(uri, "file://") {
		return strings.TrimPrefix(uri, "file://")
	}
	return uri
}
