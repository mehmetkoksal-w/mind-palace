package lsp

import (
	"encoding/json"
)

// handleDidOpen handles textDocument/didOpen notification.
func (s *Server) handleDidOpen(req Request) *Response {
	var params DidOpenTextDocumentParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.logger.Printf("Error parsing didOpen params: %v", err)
		return nil // Notifications don't get error responses
	}

	doc := &TextDocument{
		URI:        params.TextDocument.URI,
		LanguageID: params.TextDocument.LanguageID,
		Version:    params.TextDocument.Version,
		Content:    params.TextDocument.Text,
	}

	s.setDocument(doc)
	s.logger.Printf("Document opened: %s (lang=%s, version=%d)", doc.URI, doc.LanguageID, doc.Version)

	// Trigger diagnostics immediately on open (no debounce for initial load)
	go s.computeAndPublishDiagnosticsImmediate(doc.URI)

	return nil // Notification - no response
}

// handleDidChange handles textDocument/didChange notification.
func (s *Server) handleDidChange(req Request) *Response {
	var params DidChangeTextDocumentParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.logger.Printf("Error parsing didChange params: %v", err)
		return nil
	}

	doc := s.getDocument(params.TextDocument.URI)
	if doc == nil {
		s.logger.Printf("Document not found for change: %s", params.TextDocument.URI)
		return nil
	}

	// For full sync, just take the last content change
	if len(params.ContentChanges) > 0 {
		// With TextDocumentSyncKindFull, we get the full text
		doc.Content = params.ContentChanges[len(params.ContentChanges)-1].Text
		doc.Version = params.TextDocument.Version
		s.setDocument(doc)
	}

	s.logger.Printf("Document changed: %s (version=%d)", doc.URI, doc.Version)

	// Trigger diagnostics with debouncing to avoid excessive computation
	s.debounceDiagnostics(doc.URI)

	return nil
}

// handleDidClose handles textDocument/didClose notification.
func (s *Server) handleDidClose(req Request) *Response {
	var params DidCloseTextDocumentParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.logger.Printf("Error parsing didClose params: %v", err)
		return nil
	}

	s.removeDocument(params.TextDocument.URI)
	s.clearDiagnosticsCache(params.TextDocument.URI)
	s.logger.Printf("Document closed: %s", params.TextDocument.URI)

	// Clear diagnostics for closed document
	_ = s.publishDiagnostics(params.TextDocument.URI, []Diagnostic{})

	return nil
}

// handleDidSave handles textDocument/didSave notification.
func (s *Server) handleDidSave(req Request) *Response {
	var params DidSaveTextDocumentParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.logger.Printf("Error parsing didSave params: %v", err)
		return nil
	}

	s.logger.Printf("Document saved: %s", params.TextDocument.URI)

	// Re-compute diagnostics immediately on save (no debounce)
	go s.computeAndPublishDiagnosticsImmediate(params.TextDocument.URI)

	return nil
}
