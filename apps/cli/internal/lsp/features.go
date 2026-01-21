package lsp

import (
	"encoding/json"
	"fmt"
)

// handleHover handles textDocument/hover request.
func (s *Server) handleHover(req Request) *Response {
	var params HoverParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrCodeInvalidParams, "Invalid params", err.Error())
	}

	doc := s.getDocument(params.TextDocument.URI)
	if doc == nil {
		return s.successResponse(req.ID, nil)
	}

	hover := s.getHoverInfo(doc, params.Position)
	if hover == nil {
		return s.successResponse(req.ID, nil)
	}

	return s.successResponse(req.ID, hover)
}

// getHoverInfo returns hover information for a position.
func (s *Server) getHoverInfo(doc *TextDocument, pos Position) *Hover {
	if s.diagnosticsProvider == nil {
		return nil
	}

	filePath := uriToPath(doc.URI)
	line := pos.Line + 1 // Convert to 1-based

	// Check for pattern outliers on this line
	outliers, _ := s.diagnosticsProvider.GetPatternOutliersForFile(filePath)
	for _, outlier := range outliers {
		if outlier.LineStart <= line && line <= outlier.LineEnd {
			return s.patternHover(outlier)
		}
	}

	// Check for contract mismatches on this line
	mismatches, _ := s.diagnosticsProvider.GetContractMismatchesForFile(filePath)
	for _, mismatch := range mismatches {
		if mismatch.Line == line {
			return s.contractHover(mismatch)
		}
	}

	return nil
}

// patternHover creates hover content for a pattern outlier.
func (s *Server) patternHover(outlier PatternOutlier) *Hover {
	content := fmt.Sprintf("## Pattern: %s\n\n", outlier.PatternName)

	if outlier.Description != "" {
		content += fmt.Sprintf("%s\n\n", outlier.Description)
	}

	if outlier.OutlierReason != "" {
		content += fmt.Sprintf("**Issue:** %s\n\n", outlier.OutlierReason)
	}

	content += fmt.Sprintf("**Confidence:** %.0f%%\n", outlier.Confidence*100)
	content += fmt.Sprintf("**Pattern ID:** `%s`\n", outlier.PatternID)

	return &Hover{
		Contents: MarkupContent{
			Kind:  MarkupKindMarkdown,
			Value: content,
		},
	}
}

// contractHover creates hover content for a contract mismatch.
func (s *Server) contractHover(mismatch ContractMismatch) *Hover {
	content := fmt.Sprintf("## Contract: %s %s\n\n", mismatch.Method, mismatch.Endpoint)

	content += fmt.Sprintf("**Mismatch:** %s\n\n", mismatch.Description)

	if mismatch.FieldPath != "" {
		content += fmt.Sprintf("**Field:** `%s`\n", mismatch.FieldPath)
	}

	if mismatch.BackendType != "" && mismatch.FrontendType != "" {
		content += fmt.Sprintf("**Backend type:** `%s`\n", mismatch.BackendType)
		content += fmt.Sprintf("**Frontend type:** `%s`\n", mismatch.FrontendType)
	}

	content += fmt.Sprintf("\n**Severity:** %s\n", mismatch.Severity)
	content += fmt.Sprintf("**Contract ID:** `%s`\n", mismatch.ContractID)

	return &Hover{
		Contents: MarkupContent{
			Kind:  MarkupKindMarkdown,
			Value: content,
		},
	}
}

// handleCodeAction handles textDocument/codeAction request.
func (s *Server) handleCodeAction(req Request) *Response {
	var params CodeActionParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrCodeInvalidParams, "Invalid params", err.Error())
	}

	doc := s.getDocument(params.TextDocument.URI)
	if doc == nil {
		return s.successResponse(req.ID, []CodeAction{})
	}

	actions := s.getCodeActions(doc, params)
	return s.successResponse(req.ID, actions)
}

// getCodeActions returns code actions for the given context.
func (s *Server) getCodeActions(doc *TextDocument, params CodeActionParams) []CodeAction {
	var actions []CodeAction

	for _, diag := range params.Context.Diagnostics {
		if diag.Source != "mind-palace" {
			continue
		}

		// Extract diagnostic data to determine type
		data, _ := diag.Data.(map[string]interface{})
		diagType, _ := data["type"].(string)

		if diagType == "pattern" {
			patternID, _ := data["patternId"].(string)
			if patternID == "" {
				if code, ok := diag.Code.(string); ok {
					patternID = code
				}
			}

			actions = append(actions, CodeAction{
				Title:       fmt.Sprintf("Approve Pattern: %s", patternID),
				Kind:        CodeActionKindQuickFix,
				Diagnostics: []Diagnostic{diag},
				IsPreferred: true,
				Command: &Command{
					Title:     "Approve Pattern",
					Command:   "mindPalace.approvePattern",
					Arguments: []any{patternID},
				},
			})

			actions = append(actions, CodeAction{
				Title:       fmt.Sprintf("Ignore Pattern: %s", patternID),
				Kind:        CodeActionKindQuickFix,
				Diagnostics: []Diagnostic{diag},
				Command: &Command{
					Title:     "Ignore Pattern",
					Command:   "mindPalace.ignorePattern",
					Arguments: []any{patternID},
				},
			})

			actions = append(actions, CodeAction{
				Title:       "Show Pattern Details",
				Kind:        CodeActionKindQuickFix,
				Diagnostics: []Diagnostic{diag},
				Command: &Command{
					Title:     "Show Pattern Details",
					Command:   "mindPalace.showPattern",
					Arguments: []any{patternID},
				},
			})

		} else if diagType == "contract" {
			contractID, _ := data["contractId"].(string)

			actions = append(actions, CodeAction{
				Title:       fmt.Sprintf("Verify Contract: %s", contractID),
				Kind:        CodeActionKindQuickFix,
				Diagnostics: []Diagnostic{diag},
				IsPreferred: true,
				Command: &Command{
					Title:     "Verify Contract",
					Command:   "mindPalace.verifyContract",
					Arguments: []any{contractID},
				},
			})

			actions = append(actions, CodeAction{
				Title:       fmt.Sprintf("Ignore Contract: %s", contractID),
				Kind:        CodeActionKindQuickFix,
				Diagnostics: []Diagnostic{diag},
				Command: &Command{
					Title:     "Ignore Contract",
					Command:   "mindPalace.ignoreContract",
					Arguments: []any{contractID},
				},
			})

			actions = append(actions, CodeAction{
				Title:       "Show Contract Details",
				Kind:        CodeActionKindQuickFix,
				Diagnostics: []Diagnostic{diag},
				Command: &Command{
					Title:     "Show Contract Details",
					Command:   "mindPalace.showContract",
					Arguments: []any{contractID},
				},
			})

		} else {
			// Generic fallback for unknown diagnostic types
			if code, ok := diag.Code.(string); ok {
				actions = append(actions, CodeAction{
					Title:       fmt.Sprintf("Approve: %s", code),
					Kind:        CodeActionKindQuickFix,
					Diagnostics: []Diagnostic{diag},
					Command: &Command{
						Title:     "Approve",
						Command:   "mindPalace.approve",
						Arguments: []any{code},
					},
				})

				actions = append(actions, CodeAction{
					Title:       fmt.Sprintf("Ignore: %s", code),
					Kind:        CodeActionKindQuickFix,
					Diagnostics: []Diagnostic{diag},
					Command: &Command{
						Title:     "Ignore",
						Command:   "mindPalace.ignore",
						Arguments: []any{code},
					},
				})
			}
		}
	}

	return actions
}

// handleCodeLens handles textDocument/codeLens request.
func (s *Server) handleCodeLens(req Request) *Response {
	var params CodeLensParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrCodeInvalidParams, "Invalid params", err.Error())
	}

	doc := s.getDocument(params.TextDocument.URI)
	if doc == nil {
		return s.successResponse(req.ID, []CodeLens{})
	}

	lenses := s.getCodeLenses(doc)
	return s.successResponse(req.ID, lenses)
}

// getCodeLenses returns code lenses for a document.
func (s *Server) getCodeLenses(doc *TextDocument) []CodeLens {
	var lenses []CodeLens

	if s.diagnosticsProvider == nil {
		return lenses
	}

	filePath := uriToPath(doc.URI)

	// Get pattern outliers for this file
	outliers, _ := s.diagnosticsProvider.GetPatternOutliersForFile(filePath)
	patternCount := len(outliers)

	// Get contract mismatches for this file
	mismatches, _ := s.diagnosticsProvider.GetContractMismatchesForFile(filePath)
	contractCount := len(mismatches)

	// Add pattern count lens at file top
	if patternCount > 0 {
		title := fmt.Sprintf("$(warning) %d pattern issue", patternCount)
		if patternCount > 1 {
			title = fmt.Sprintf("$(warning) %d pattern issues", patternCount)
		}

		lenses = append(lenses, CodeLens{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
			Command: &Command{
				Title:     title,
				Command:   "mindPalace.showPatterns",
				Arguments: []any{doc.URI},
			},
			Data: map[string]interface{}{
				"type":  "patterns",
				"uri":   doc.URI,
				"count": patternCount,
			},
		})
	}

	// Add contract count lens at file top
	if contractCount > 0 {
		title := fmt.Sprintf("$(error) %d contract mismatch", contractCount)
		if contractCount > 1 {
			title = fmt.Sprintf("$(error) %d contract mismatches", contractCount)
		}

		lenses = append(lenses, CodeLens{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
			Command: &Command{
				Title:     title,
				Command:   "mindPalace.showContracts",
				Arguments: []any{doc.URI},
			},
			Data: map[string]interface{}{
				"type":  "contracts",
				"uri":   doc.URI,
				"count": contractCount,
			},
		})
	}

	// Add inline lenses for each pattern outlier
	for _, outlier := range outliers {
		lenses = append(lenses, CodeLens{
			Range: Range{
				Start: Position{Line: outlier.LineStart - 1, Character: 0}, // Convert to 0-based
				End:   Position{Line: outlier.LineStart - 1, Character: 0},
			},
			Command: &Command{
				Title:     fmt.Sprintf("Pattern: %s (%.0f%%)", outlier.PatternName, outlier.Confidence*100),
				Command:   "mindPalace.showPattern",
				Arguments: []any{outlier.PatternID},
			},
			Data: map[string]interface{}{
				"type":      "pattern",
				"patternId": outlier.PatternID,
			},
		})
	}

	// Add inline lenses for each contract mismatch
	for _, mismatch := range mismatches {
		lenses = append(lenses, CodeLens{
			Range: Range{
				Start: Position{Line: mismatch.Line - 1, Character: 0}, // Convert to 0-based
				End:   Position{Line: mismatch.Line - 1, Character: 0},
			},
			Command: &Command{
				Title:     fmt.Sprintf("Contract: %s %s", mismatch.Method, mismatch.Endpoint),
				Command:   "mindPalace.showContract",
				Arguments: []any{mismatch.ContractID},
			},
			Data: map[string]interface{}{
				"type":       "contract",
				"contractId": mismatch.ContractID,
			},
		})
	}

	return lenses
}

// handleCodeLensResolve handles codeLens/resolve request.
func (s *Server) handleCodeLensResolve(req Request) *Response {
	var lens CodeLens
	if err := json.Unmarshal(req.Params, &lens); err != nil {
		return s.errorResponse(req.ID, ErrCodeInvalidParams, "Invalid params", err.Error())
	}

	// Code lens is already resolved with command during getCodeLenses
	// This handler is for deferred resolution if needed
	if lens.Command == nil && lens.Data != nil {
		data, _ := lens.Data.(map[string]interface{})
		lensType, _ := data["type"].(string)

		switch lensType {
		case "patterns":
			uri, _ := data["uri"].(string)
			count, _ := data["count"].(float64) // JSON numbers are float64
			title := fmt.Sprintf("$(warning) %d pattern issues", int(count))
			if int(count) == 1 {
				title = "$(warning) 1 pattern issue"
			}
			lens.Command = &Command{
				Title:     title,
				Command:   "mindPalace.showPatterns",
				Arguments: []any{uri},
			}
		case "contracts":
			uri, _ := data["uri"].(string)
			count, _ := data["count"].(float64)
			title := fmt.Sprintf("$(error) %d contract mismatches", int(count))
			if int(count) == 1 {
				title = "$(error) 1 contract mismatch"
			}
			lens.Command = &Command{
				Title:     title,
				Command:   "mindPalace.showContracts",
				Arguments: []any{uri},
			}
		case "pattern":
			patternID, _ := data["patternId"].(string)
			lens.Command = &Command{
				Title:     "Show Pattern",
				Command:   "mindPalace.showPattern",
				Arguments: []any{patternID},
			}
		case "contract":
			contractID, _ := data["contractId"].(string)
			lens.Command = &Command{
				Title:     "Show Contract",
				Command:   "mindPalace.showContract",
				Arguments: []any{contractID},
			}
		}
	}

	return s.successResponse(req.ID, lens)
}

// handleDefinition handles textDocument/definition request.
func (s *Server) handleDefinition(req Request) *Response {
	var params DefinitionParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrCodeInvalidParams, "Invalid params", err.Error())
	}

	doc := s.getDocument(params.TextDocument.URI)
	if doc == nil {
		return s.successResponse(req.ID, nil)
	}

	location := s.getDefinition(doc, params.Position)
	if location == nil {
		return s.successResponse(req.ID, nil)
	}

	return s.successResponse(req.ID, location)
}

// getDefinition returns the definition location for a position.
func (s *Server) getDefinition(doc *TextDocument, pos Position) *Location {
	if s.diagnosticsProvider == nil {
		return nil
	}

	filePath := uriToPath(doc.URI)
	line := pos.Line + 1 // Convert to 1-based

	// Check if we're on a pattern outlier line
	outliers, _ := s.diagnosticsProvider.GetPatternOutliersForFile(filePath)
	for _, outlier := range outliers {
		if outlier.LineStart <= line && line <= outlier.LineEnd {
			// Return the canonical pattern location (first occurrence)
			// In a real implementation, this would query the pattern store
			// for the pattern's first detected location
			return &Location{
				URI: pathToURI(outlier.FilePath),
				Range: Range{
					Start: Position{Line: outlier.LineStart - 1, Character: 0},
					End:   Position{Line: outlier.LineEnd - 1, Character: 0},
				},
			}
		}
	}

	// Check if we're on a contract mismatch line
	mismatches, _ := s.diagnosticsProvider.GetContractMismatchesForFile(filePath)
	for _, mismatch := range mismatches {
		if mismatch.Line == line {
			// Navigate to the related file (backend if on frontend, frontend if on backend)
			// This requires additional context from the diagnostics provider
			// For now, return the current location as a placeholder
			return &Location{
				URI: pathToURI(mismatch.FilePath),
				Range: Range{
					Start: Position{Line: mismatch.Line - 1, Character: 0},
					End:   Position{Line: mismatch.Line - 1, Character: 0},
				},
			}
		}
	}

	return nil
}

// handleDocumentSymbol handles textDocument/documentSymbol request.
func (s *Server) handleDocumentSymbol(req Request) *Response {
	var params DocumentSymbolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrCodeInvalidParams, "Invalid params", err.Error())
	}

	doc := s.getDocument(params.TextDocument.URI)
	if doc == nil {
		return s.successResponse(req.ID, []DocumentSymbol{})
	}

	symbols := s.getDocumentSymbols(doc)
	return s.successResponse(req.ID, symbols)
}

// getDocumentSymbols returns document symbols for a document.
func (s *Server) getDocumentSymbols(doc *TextDocument) []DocumentSymbol {
	var symbols []DocumentSymbol

	if s.diagnosticsProvider == nil {
		return symbols
	}

	filePath := uriToPath(doc.URI)

	// Add pattern outliers as symbols
	outliers, _ := s.diagnosticsProvider.GetPatternOutliersForFile(filePath)
	for _, outlier := range outliers {
		symbol := DocumentSymbol{
			Name:   fmt.Sprintf("Pattern: %s", outlier.PatternName),
			Detail: outlier.OutlierReason,
			Kind:   SymbolKindEvent, // Using Event for patterns
			Range: Range{
				Start: Position{Line: outlier.LineStart - 1, Character: 0},
				End:   Position{Line: outlier.LineEnd - 1, Character: 0},
			},
			SelectionRange: Range{
				Start: Position{Line: outlier.LineStart - 1, Character: 0},
				End:   Position{Line: outlier.LineStart - 1, Character: 0},
			},
		}
		symbols = append(symbols, symbol)
	}

	// Add contract mismatches as symbols
	mismatches, _ := s.diagnosticsProvider.GetContractMismatchesForFile(filePath)
	for _, mismatch := range mismatches {
		symbol := DocumentSymbol{
			Name:   fmt.Sprintf("Contract: %s %s", mismatch.Method, mismatch.Endpoint),
			Detail: mismatch.Description,
			Kind:   SymbolKindInterface, // Using Interface for contracts
			Range: Range{
				Start: Position{Line: mismatch.Line - 1, Character: 0},
				End:   Position{Line: mismatch.Line - 1, Character: 0},
			},
			SelectionRange: Range{
				Start: Position{Line: mismatch.Line - 1, Character: 0},
				End:   Position{Line: mismatch.Line - 1, Character: 0},
			},
		}
		symbols = append(symbols, symbol)
	}

	return symbols
}
