package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// TestReadMessage tests the LSP message reading.
func TestReadMessage(t *testing.T) {
	content := `{"jsonrpc":"2.0","id":1,"method":"initialize"}`
	input := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(content), content)

	server := NewServerWithIO(strings.NewReader(input), &bytes.Buffer{})

	msg, err := server.readMessage()
	if err != nil {
		t.Fatalf("readMessage failed: %v", err)
	}

	var req Request
	if err := json.Unmarshal(msg, &req); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if req.Method != "initialize" {
		t.Errorf("expected method 'initialize', got '%s'", req.Method)
	}
	if req.ID != float64(1) { // JSON numbers are float64
		t.Errorf("expected id 1, got %v", req.ID)
	}
}

// TestWriteMessage tests the LSP message writing.
func TestWriteMessage(t *testing.T) {
	var output bytes.Buffer
	server := NewServerWithIO(strings.NewReader(""), &output)

	resp := &Response{
		JSONRPC: "2.0",
		ID:      1,
		Result:  map[string]string{"test": "value"},
	}

	if err := server.writeMessage(resp); err != nil {
		t.Fatalf("writeMessage failed: %v", err)
	}

	result := output.String()
	if !strings.HasPrefix(result, "Content-Length:") {
		t.Error("expected Content-Length header")
	}
	if !strings.Contains(result, `"test":"value"`) {
		t.Error("expected result in output")
	}
}

// TestInitialize tests the initialize request handling.
func TestInitialize(t *testing.T) {
	params := InitializeParams{
		ProcessID: 1234,
		RootURI:   "file:///test/project",
		Capabilities: ClientCapabilities{},
	}

	paramsJSON, _ := json.Marshal(params)
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  paramsJSON,
	}

	server := NewServerWithIO(strings.NewReader(""), &bytes.Buffer{})
	resp := server.handleInitialize(req)

	if resp.Error != nil {
		t.Fatalf("initialize returned error: %v", resp.Error)
	}

	result, ok := resp.Result.(InitializeResult)
	if !ok {
		t.Fatalf("expected InitializeResult, got %T", resp.Result)
	}

	if result.ServerInfo.Name != "mind-palace-lsp" {
		t.Errorf("expected server name 'mind-palace-lsp', got '%s'", result.ServerInfo.Name)
	}

	if !result.Capabilities.HoverProvider {
		t.Error("expected HoverProvider to be true")
	}
}

// TestDocumentLifecycle tests document open/change/close.
func TestDocumentLifecycle(t *testing.T) {
	server := NewServerWithIO(strings.NewReader(""), &bytes.Buffer{})
	server.initialized = true

	// Open document
	openParams := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        "file:///test/file.go",
			LanguageID: "go",
			Version:    1,
			Text:       "package main",
		},
	}
	openJSON, _ := json.Marshal(openParams)
	server.handleDidOpen(Request{Params: openJSON})

	doc := server.getDocument("file:///test/file.go")
	if doc == nil {
		t.Fatal("document should be open")
	}
	if doc.Content != "package main" {
		t.Errorf("expected content 'package main', got '%s'", doc.Content)
	}

	// Change document
	changeParams := DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: TextDocumentIdentifier{URI: "file:///test/file.go"},
			Version:                2,
		},
		ContentChanges: []TextDocumentContentChangeEvent{
			{Text: "package main\n\nfunc main() {}"},
		},
	}
	changeJSON, _ := json.Marshal(changeParams)
	server.handleDidChange(Request{Params: changeJSON})

	doc = server.getDocument("file:///test/file.go")
	if doc.Version != 2 {
		t.Errorf("expected version 2, got %d", doc.Version)
	}
	if !strings.Contains(doc.Content, "func main()") {
		t.Error("expected content to include 'func main()'")
	}

	// Close document
	closeParams := DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///test/file.go"},
	}
	closeJSON, _ := json.Marshal(closeParams)
	server.handleDidClose(Request{Params: closeJSON})

	doc = server.getDocument("file:///test/file.go")
	if doc != nil {
		t.Error("document should be closed")
	}
}

// TestMethodDispatch tests that methods are correctly dispatched.
func TestMethodDispatch(t *testing.T) {
	server := NewServerWithIO(strings.NewReader(""), &bytes.Buffer{})

	// Before initialization, most methods should fail
	resp := server.handleMessage([]byte(`{"jsonrpc":"2.0","id":1,"method":"textDocument/hover"}`))
	if resp.Error == nil || resp.Error.Code != ErrCodeServerNotInitialized {
		t.Error("expected ServerNotInitialized error before initialization")
	}

	// Initialize
	initParams := InitializeParams{ProcessID: 1, RootURI: "file:///test"}
	paramsJSON, _ := json.Marshal(initParams)
	server.handleMessage([]byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":%s}`, paramsJSON)))
	server.handleMessage([]byte(`{"jsonrpc":"2.0","method":"initialized"}`))

	// After initialization, methods should work
	resp = server.handleMessage([]byte(`{"jsonrpc":"2.0","id":2,"method":"textDocument/hover","params":{"textDocument":{"uri":"file:///test.go"},"position":{"line":0,"character":0}}}`))
	if resp.Error != nil {
		t.Errorf("expected no error after initialization, got: %v", resp.Error)
	}

	// Unknown method should return MethodNotFound
	resp = server.handleMessage([]byte(`{"jsonrpc":"2.0","id":3,"method":"unknown/method"}`))
	if resp.Error == nil || resp.Error.Code != ErrCodeMethodNotFound {
		t.Error("expected MethodNotFound error for unknown method")
	}
}
