package analysis

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

// LSP protocol message types
// These types are shared across all LSP client implementations
// as they represent the standardized LSP protocol structures.

// lspRequest represents an LSP request message
type lspRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// lspResponse represents an LSP response or notification message
type lspResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *lspError       `json:"error,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// lspError represents an LSP error object
type lspError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// pathToURI converts a file path to a file:// URI
// Handles Windows path separators correctly
func pathToURI(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	// Convert Windows path separators to forward slashes
	absPath = filepath.ToSlash(absPath)
	return "file:///" + strings.TrimPrefix(absPath, "/")
}

// uriToPath converts a file:// URI to a file path
// Handles both file:// and file:/// prefixes and converts to OS-specific separators
func uriToPath(uri string) string {
	path := strings.TrimPrefix(uri, "file:///")
	path = strings.TrimPrefix(path, "file://")
	// Convert forward slashes back to OS-specific separators
	return filepath.FromSlash(path)
}
