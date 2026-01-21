package lsp

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
)

// DiagnosticsProvider provides diagnostics data to the LSP server.
// This interface allows decoupling from concrete Butler/memory types.
type DiagnosticsProvider interface {
	// GetPatternOutliersForFile returns pattern outliers for a file.
	GetPatternOutliersForFile(filePath string) ([]PatternOutlier, error)
	// GetContractMismatchesForFile returns contract mismatches for a file.
	GetContractMismatchesForFile(filePath string) ([]ContractMismatch, error)
}

// PatternOutlier represents a pattern outlier at a specific location.
type PatternOutlier struct {
	PatternID     string
	PatternName   string
	Description   string
	FilePath      string
	LineStart     int
	LineEnd       int
	Snippet       string
	OutlierReason string
	Confidence    float64
}

// ContractMismatch represents a contract type mismatch at a specific location.
type ContractMismatch struct {
	ContractID   string
	Method       string
	Endpoint     string
	FieldPath    string
	MismatchType string
	Severity     string
	Description  string
	BackendType  string
	FrontendType string
	FilePath     string
	Line         int
}

// SetDiagnosticsProvider sets the diagnostics provider for the server.
func (s *Server) SetDiagnosticsProvider(provider DiagnosticsProvider) {
	s.docMu.Lock()
	defer s.docMu.Unlock()
	s.diagnosticsProvider = provider
}

// computeDiagnostics computes diagnostics for a document.
func (s *Server) computeDiagnostics(doc *TextDocument) []Diagnostic {
	var diagnostics []Diagnostic

	// Convert URI to file path
	filePath := uriToPath(doc.URI)

	// Get pattern outliers
	if s.diagnosticsProvider != nil {
		outliers, err := s.diagnosticsProvider.GetPatternOutliersForFile(filePath)
		if err != nil {
			s.logger.Printf("Error getting pattern outliers: %v", err)
		} else {
			for _, outlier := range outliers {
				diag := s.patternOutlierToDiagnostic(outlier)
				diagnostics = append(diagnostics, diag)
			}
		}

		// Get contract mismatches
		mismatches, err := s.diagnosticsProvider.GetContractMismatchesForFile(filePath)
		if err != nil {
			s.logger.Printf("Error getting contract mismatches: %v", err)
		} else {
			for _, mismatch := range mismatches {
				diag := s.contractMismatchToDiagnostic(mismatch)
				diagnostics = append(diagnostics, diag)
			}
		}
	}

	return diagnostics
}

// patternOutlierToDiagnostic converts a pattern outlier to an LSP diagnostic.
func (s *Server) patternOutlierToDiagnostic(outlier PatternOutlier) Diagnostic {
	// Convert 1-based line numbers to 0-based
	startLine := outlier.LineStart - 1
	if startLine < 0 {
		startLine = 0
	}
	endLine := outlier.LineEnd - 1
	if endLine < startLine {
		endLine = startLine
	}

	message := outlier.Description
	if outlier.OutlierReason != "" {
		message = outlier.OutlierReason
	}
	if message == "" {
		message = "Deviates from pattern: " + outlier.PatternName
	}

	return Diagnostic{
		Range: Range{
			Start: Position{Line: startLine, Character: 0},
			End:   Position{Line: endLine, Character: 0},
		},
		Severity: DiagnosticSeverityWarning,
		Code:     outlier.PatternID,
		Source:   "mind-palace",
		Message:  message,
		Data: map[string]interface{}{
			"type":       "pattern",
			"patternId":  outlier.PatternID,
			"confidence": outlier.Confidence,
		},
	}
}

// contractMismatchToDiagnostic converts a contract mismatch to an LSP diagnostic.
func (s *Server) contractMismatchToDiagnostic(mismatch ContractMismatch) Diagnostic {
	// Convert 1-based line number to 0-based
	line := mismatch.Line - 1
	if line < 0 {
		line = 0
	}

	// Determine severity
	severity := DiagnosticSeverityWarning
	if mismatch.Severity == "error" {
		severity = DiagnosticSeverityError
	}

	message := mismatch.Description
	if mismatch.BackendType != "" && mismatch.FrontendType != "" {
		message += " (backend: " + mismatch.BackendType + ", frontend: " + mismatch.FrontendType + ")"
	}

	return Diagnostic{
		Range: Range{
			Start: Position{Line: line, Character: 0},
			End:   Position{Line: line, Character: 0},
		},
		Severity: severity,
		Code:     mismatch.MismatchType,
		Source:   "mind-palace",
		Message:  message,
		Data: map[string]interface{}{
			"type":       "contract",
			"contractId": mismatch.ContractID,
			"method":     mismatch.Method,
			"endpoint":   mismatch.Endpoint,
			"fieldPath":  mismatch.FieldPath,
		},
	}
}

// URI/Path conversion utilities

// uriToPath converts a file:// URI to a file path.
func uriToPath(uri string) string {
	path := strings.TrimPrefix(uri, "file:///")
	path = strings.TrimPrefix(path, "file://")
	// Handle Windows drive letters
	if len(path) >= 3 && path[1] == ':' {
		// Already a Windows path
		return filepath.FromSlash(path)
	}
	if len(path) >= 2 && path[0] != '/' && path[1] == ':' {
		// Windows path without leading slash
		return filepath.FromSlash(path)
	}
	// Unix path - add leading slash if missing
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return filepath.FromSlash(path)
}

// pathToURI converts a file path to a file:// URI.
func pathToURI(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	absPath = filepath.ToSlash(absPath)
	// Ensure proper file URI format
	if !strings.HasPrefix(absPath, "/") {
		// Windows path
		return "file:///" + absPath
	}
	return "file://" + absPath
}

// handleDiagnostic handles the textDocument/diagnostic request (pull diagnostics).
// This is part of the LSP 3.17 diagnostic pull model.
func (s *Server) handleDiagnostic(req Request) *Response {
	var params DocumentDiagnosticParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrCodeInvalidParams, "Invalid params", err.Error())
	}

	s.logger.Printf("Diagnostic request for: %s", params.TextDocument.URI)

	// Get document from cache
	doc := s.getDocument(params.TextDocument.URI)
	if doc == nil {
		// Document not open - return empty diagnostics
		return s.successResponse(req.ID, DocumentDiagnosticReport{
			Kind:  "full",
			Items: []Diagnostic{},
		})
	}

	// Check if we have cached diagnostics and result hasn't changed
	cachedDiags, hasCached := s.getCachedDiagnostics(params.TextDocument.URI)
	if hasCached && params.PreviousResultID != "" {
		// Check if document version matches (simplified check)
		currentResultID := s.getResultID(params.TextDocument.URI)
		if params.PreviousResultID == currentResultID {
			return s.successResponse(req.ID, DocumentDiagnosticReport{
				Kind:     "unchanged",
				ResultID: currentResultID,
			})
		}
	}

	// Compute fresh diagnostics
	var diagnostics []Diagnostic
	if cachedDiags != nil {
		diagnostics = cachedDiags
	} else {
		diagnostics = s.computeDiagnostics(doc)
	}

	resultID := s.getResultID(params.TextDocument.URI)

	return s.successResponse(req.ID, DocumentDiagnosticReport{
		Kind:     "full",
		ResultID: resultID,
		Items:    diagnostics,
	})
}

// getResultID generates a result ID for diagnostic caching.
func (s *Server) getResultID(uri string) string {
	doc := s.getDocument(uri)
	if doc == nil {
		return ""
	}
	// Use document version as result ID
	return uri + ":" + strconv.Itoa(doc.Version)
}
