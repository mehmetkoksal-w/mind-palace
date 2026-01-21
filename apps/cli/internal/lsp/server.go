package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Server is the LSP server for Mind Palace.
type Server struct {
	// I/O
	reader *bufio.Reader
	writer io.Writer
	mu     sync.Mutex // protects writes

	// State
	initialized bool
	shutdown    bool
	rootURI     string

	// Document management
	documents map[string]*TextDocument
	docMu     sync.RWMutex

	// Logging
	logger *log.Logger

	// Diagnostics provider (set via SetDiagnosticsProvider)
	diagnosticsProvider DiagnosticsProvider

	// Performance: debouncing
	debounceTimers map[string]*time.Timer
	debounceMu     sync.Mutex
	debounceDelay  time.Duration

	// Performance: diagnostics cache
	diagnosticsCache map[string][]Diagnostic
	cacheMu          sync.RWMutex
}

// TextDocument represents an open text document.
type TextDocument struct {
	URI        string
	LanguageID string
	Version    int
	Content    string
}

// DefaultDebounceDelay is the default delay for debouncing diagnostics.
const DefaultDebounceDelay = 300 * time.Millisecond

// NewServer creates a new LSP server.
func NewServer() *Server {
	// Create log file for debugging (don't log to stdout - that's for LSP messages)
	logFile, err := os.OpenFile("palace-lsp.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	var logger *log.Logger
	if err != nil {
		logger = log.New(io.Discard, "", 0)
	} else {
		logger = log.New(logFile, "[LSP] ", log.LstdFlags|log.Lshortfile)
	}

	return &Server{
		reader:           bufio.NewReader(os.Stdin),
		writer:           os.Stdout,
		documents:        make(map[string]*TextDocument),
		logger:           logger,
		debounceTimers:   make(map[string]*time.Timer),
		debounceDelay:    DefaultDebounceDelay,
		diagnosticsCache: make(map[string][]Diagnostic),
	}
}

// NewServerWithIO creates an LSP server with custom I/O (for testing).
func NewServerWithIO(reader io.Reader, writer io.Writer) *Server {
	return &Server{
		reader:           bufio.NewReader(reader),
		writer:           writer,
		documents:        make(map[string]*TextDocument),
		logger:           log.New(io.Discard, "", 0),
		debounceTimers:   make(map[string]*time.Timer),
		debounceDelay:    DefaultDebounceDelay,
		diagnosticsCache: make(map[string][]Diagnostic),
	}
}

// Run starts the LSP server main loop.
func (s *Server) Run(ctx context.Context) error {
	s.logger.Println("LSP server starting")

	for {
		select {
		case <-ctx.Done():
			s.logger.Println("Context cancelled, shutting down")
			return ctx.Err()
		default:
		}

		// Read message
		msg, err := s.readMessage()
		if err != nil {
			if err == io.EOF {
				s.logger.Println("EOF received, shutting down")
				return nil
			}
			s.logger.Printf("Read error: %v", err)
			continue
		}

		// Handle message
		resp := s.handleMessage(msg)
		if resp != nil {
			if err := s.writeMessage(resp); err != nil {
				s.logger.Printf("Write error: %v", err)
				return err
			}
		}

		// Check for shutdown
		if s.shutdown {
			s.logger.Println("Shutdown requested")
			return nil
		}
	}
}

// readMessage reads an LSP message from the input.
// LSP uses Content-Length headers followed by the JSON payload.
func (s *Server) readMessage() ([]byte, error) {
	// Read headers
	var contentLength int
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			// Empty line marks end of headers
			break
		}

		// Parse Content-Length header
		if strings.HasPrefix(line, "Content-Length:") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			contentLength, err = strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length: %w", err)
			}
		}
		// Ignore other headers (like Content-Type)
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}

	// Read content
	content := make([]byte, contentLength)
	_, err := io.ReadFull(s.reader, content)
	if err != nil {
		return nil, fmt.Errorf("read content: %w", err)
	}

	s.logger.Printf("Received: %s", string(content))
	return content, nil
}

// writeMessage writes an LSP message to the output.
func (s *Server) writeMessage(msg interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	content, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}

	s.logger.Printf("Sending: %s", string(content))

	// Write header and content
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(content))
	if _, err := s.writer.Write([]byte(header)); err != nil {
		return err
	}
	if _, err := s.writer.Write(content); err != nil {
		return err
	}

	return nil
}

// handleMessage processes an incoming message and returns a response (if any).
func (s *Server) handleMessage(msg []byte) *Response {
	var req Request
	if err := json.Unmarshal(msg, &req); err != nil {
		return s.errorResponse(nil, ErrCodeParseError, "Parse error", err.Error())
	}

	s.logger.Printf("Handling method: %s", req.Method)

	// Check if initialized (except for initialize/initialized/exit)
	if !s.initialized && req.Method != "initialize" && req.Method != "initialized" && req.Method != "exit" {
		return s.errorResponse(req.ID, ErrCodeServerNotInitialized, "Server not initialized", nil)
	}

	// Dispatch based on method
	switch req.Method {
	// Lifecycle
	case "initialize":
		return s.handleInitialize(req)
	case "initialized":
		return s.handleInitialized(req)
	case "shutdown":
		return s.handleShutdown(req)
	case "exit":
		return s.handleExit(req)

	// Document sync
	case "textDocument/didOpen":
		return s.handleDidOpen(req)
	case "textDocument/didChange":
		return s.handleDidChange(req)
	case "textDocument/didClose":
		return s.handleDidClose(req)
	case "textDocument/didSave":
		return s.handleDidSave(req)

	// Features (to be implemented)
	case "textDocument/hover":
		return s.handleHover(req)
	case "textDocument/codeAction":
		return s.handleCodeAction(req)
	case "textDocument/codeLens":
		return s.handleCodeLens(req)
	case "codeLens/resolve":
		return s.handleCodeLensResolve(req)
	case "textDocument/definition":
		return s.handleDefinition(req)
	case "textDocument/documentSymbol":
		return s.handleDocumentSymbol(req)
	case "textDocument/diagnostic":
		return s.handleDiagnostic(req)

	// Notifications (no response)
	case "$/cancelRequest":
		return nil // Ignored for now
	case "$/setTrace":
		return nil // Ignored

	default:
		s.logger.Printf("Unknown method: %s", req.Method)
		return s.errorResponse(req.ID, ErrCodeMethodNotFound, "Method not found", req.Method)
	}
}

// Helper functions

func (s *Server) successResponse(id any, result interface{}) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

func (s *Server) errorResponse(id any, code int, message string, data any) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// publishDiagnostics sends diagnostics for a document.
func (s *Server) publishDiagnostics(uri string, diagnostics []Diagnostic) error {
	params := PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	}

	// Send as notification (no ID)
	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "textDocument/publishDiagnostics",
		"params":  params,
	}

	return s.writeMessage(notification)
}

// Document helpers

func (s *Server) getDocument(uri string) *TextDocument {
	s.docMu.RLock()
	defer s.docMu.RUnlock()
	return s.documents[uri]
}

func (s *Server) setDocument(doc *TextDocument) {
	s.docMu.Lock()
	defer s.docMu.Unlock()
	s.documents[doc.URI] = doc
}

func (s *Server) removeDocument(uri string) {
	s.docMu.Lock()
	defer s.docMu.Unlock()
	delete(s.documents, uri)
}

// SetLogger sets the logger for the server.
func (s *Server) SetLogger(w io.Writer) {
	if w == nil {
		s.logger = log.New(io.Discard, "", 0)
	} else {
		s.logger = log.New(w, "[LSP] ", log.LstdFlags|log.Lshortfile)
	}
}

// SetDebounceDelay sets the debounce delay for diagnostics.
func (s *Server) SetDebounceDelay(delay time.Duration) {
	s.debounceDelay = delay
}

// debounceDiagnostics schedules diagnostics computation with debouncing.
func (s *Server) debounceDiagnostics(uri string) {
	s.debounceMu.Lock()
	defer s.debounceMu.Unlock()

	// Cancel existing timer for this URI
	if timer, exists := s.debounceTimers[uri]; exists {
		timer.Stop()
	}

	// Schedule new computation
	s.debounceTimers[uri] = time.AfterFunc(s.debounceDelay, func() {
		s.computeAndPublishDiagnosticsImmediate(uri)
	})
}

// computeAndPublishDiagnosticsImmediate computes and publishes diagnostics immediately.
func (s *Server) computeAndPublishDiagnosticsImmediate(uri string) {
	doc := s.getDocument(uri)
	if doc == nil {
		return
	}

	diagnostics := s.computeDiagnostics(doc)

	// Update cache
	s.cacheMu.Lock()
	s.diagnosticsCache[uri] = diagnostics
	s.cacheMu.Unlock()

	if err := s.publishDiagnostics(uri, diagnostics); err != nil {
		s.logger.Printf("Error publishing diagnostics: %v", err)
	}
}

// getCachedDiagnostics returns cached diagnostics for a URI.
func (s *Server) getCachedDiagnostics(uri string) ([]Diagnostic, bool) {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	diags, exists := s.diagnosticsCache[uri]
	return diags, exists
}

// clearDiagnosticsCache clears the diagnostics cache for a URI.
func (s *Server) clearDiagnosticsCache(uri string) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	delete(s.diagnosticsCache, uri)
}
