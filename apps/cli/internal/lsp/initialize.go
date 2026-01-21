package lsp

import "encoding/json"

// handleInitialize handles the initialize request.
func (s *Server) handleInitialize(req Request) *Response {
	var params InitializeParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrCodeInvalidParams, "Invalid params", err.Error())
	}

	s.logger.Printf("Initialize: rootURI=%s, processId=%d", params.RootURI, params.ProcessID)

	s.rootURI = params.RootURI

	result := InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: &TextDocumentSyncOptions{
				OpenClose: true,
				Change:    TextDocumentSyncKindFull, // Start with full sync for simplicity
				Save: &SaveOptions{
					IncludeText: false,
				},
			},
			HoverProvider: true,
			CodeActionProvider: &CodeActionOptions{
				CodeActionKinds: []CodeActionKind{
					CodeActionKindQuickFix,
				},
				ResolveProvider: false,
			},
			CodeLensProvider: &CodeLensOptions{
				ResolveProvider: true,
			},
			DefinitionProvider:     true,
			DocumentSymbolProvider: true,
			DiagnosticProvider: &DiagnosticOptions{
				Identifier:            "mind-palace",
				InterFileDependencies: false,
				WorkspaceDiagnostics:  false,
			},
		},
		ServerInfo: &ServerInfo{
			Name:    "mind-palace-lsp",
			Version: "0.1.0",
		},
	}

	return s.successResponse(req.ID, result)
}

// handleInitialized handles the initialized notification.
func (s *Server) handleInitialized(req Request) *Response {
	s.initialized = true
	s.logger.Println("Server initialized")
	// Notification - no response
	return nil
}

// handleShutdown handles the shutdown request.
func (s *Server) handleShutdown(req Request) *Response {
	s.logger.Println("Shutdown requested")
	s.shutdown = true
	return s.successResponse(req.ID, nil)
}

// handleExit handles the exit notification.
func (s *Server) handleExit(req Request) *Response {
	s.logger.Println("Exit requested")
	// The server loop will check s.shutdown and exit
	s.shutdown = true
	return nil
}
