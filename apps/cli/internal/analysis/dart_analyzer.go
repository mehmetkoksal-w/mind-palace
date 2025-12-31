package analysis

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// DartAnalyzer provides deep analysis of Dart files using the Dart Analysis Server
type DartAnalyzer struct {
	client   *DartLSPClient
	rootPath string
	mu       sync.Mutex
}

// NewDartAnalyzer creates a new Dart analyzer with LSP support
func NewDartAnalyzer(rootPath string) (*DartAnalyzer, error) {
	client, err := NewDartLSPClient(rootPath)
	if err != nil {
		return nil, fmt.Errorf("create LSP client: %w", err)
	}

	return &DartAnalyzer{
		client:   client,
		rootPath: rootPath,
	}, nil
}

// Close shuts down the analyzer
func (a *DartAnalyzer) Close() error {
	if a.client != nil {
		return a.client.Close()
	}
	return nil
}

// AnalyzeFile performs deep analysis on a Dart file
func (a *DartAnalyzer) AnalyzeFile(filePath string) (*FileAnalysis, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// First, do basic regex parsing to get symbols
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	parser := NewDartParser()
	analysis, err := parser.Parse(content, filePath)
	if err != nil {
		return nil, fmt.Errorf("basic parse: %w", err)
	}

	// Open file in LSP server
	if err := a.client.OpenFile(filePath, string(content)); err != nil {
		// If LSP fails, return basic analysis
		return analysis, nil
	}
	defer a.client.CloseFile(filePath)

	// Extract call relationships for each function/method
	a.extractCallsForSymbols(filePath, analysis.Symbols, analysis)

	return analysis, nil
}

// extractCallsForSymbols recursively extracts call relationships for symbols
func (a *DartAnalyzer) extractCallsForSymbols(filePath string, symbols []Symbol, analysis *FileAnalysis) {
	for _, sym := range symbols {
		// Only analyze functions and methods
		if sym.Kind == KindFunction || sym.Kind == KindMethod || sym.Kind == KindConstructor {
			// Get character position (start of line, adjust if needed)
			calls, err := a.client.ExtractCallsForSymbol(filePath, sym.LineStart-1, 0)
			if err == nil {
				for _, call := range calls {
					// Convert to relative paths
					callerPath := a.toRelativePath(call.CallerFile)
					calleePath := a.toRelativePath(call.CalleeFile)

					analysis.Relationships = append(analysis.Relationships, Relationship{
						SourceSymbol: call.CallerSymbol,
						TargetFile:   calleePath,
						TargetSymbol: call.CalleeSymbol,
						Kind:         RelCall,
						Line:         call.CallerLine,
					})

					// Also add as reference
					if callerPath != calleePath {
						analysis.Relationships = append(analysis.Relationships, Relationship{
							SourceSymbol: call.CallerSymbol,
							TargetFile:   calleePath,
							TargetSymbol: call.CalleeSymbol,
							Kind:         RelReference,
							Line:         call.CallerLine,
						})
					}
				}
			}
		}

		// Recurse into children
		if len(sym.Children) > 0 {
			a.extractCallsForSymbols(filePath, sym.Children, analysis)
		}
	}
}

func (a *DartAnalyzer) toRelativePath(absPath string) string {
	if strings.HasPrefix(absPath, a.rootPath) {
		rel, err := filepath.Rel(a.rootPath, absPath)
		if err == nil {
			return rel
		}
	}
	return absPath
}

// ExtractAllCalls extracts call relationships for all Dart files in a directory
func (a *DartAnalyzer) ExtractAllCalls(files []string, progressFn func(current, total int, file string)) ([]CallInfo, error) {
	var allCalls []CallInfo
	var mu sync.Mutex

	total := len(files)
	for i, file := range files {
		if progressFn != nil {
			progressFn(i+1, total, file)
		}

		if !strings.HasSuffix(file, ".dart") {
			continue
		}

		// Read file
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Parse to get symbols
		parser := NewDartParser()
		analysis, err := parser.Parse(content, file)
		if err != nil {
			continue
		}

		// Open file in LSP
		if err := a.client.OpenFile(file, string(content)); err != nil {
			continue
		}

		// Extract calls for each function/method
		for _, sym := range analysis.Symbols {
			if sym.Kind == KindFunction || sym.Kind == KindMethod || sym.Kind == KindConstructor {
				calls, err := a.client.ExtractCallsForSymbol(file, sym.LineStart-1, 0)
				if err == nil && len(calls) > 0 {
					mu.Lock()
					allCalls = append(allCalls, calls...)
					mu.Unlock()
				}
			}

			// Check children
			for _, child := range sym.Children {
				if child.Kind == KindFunction || child.Kind == KindMethod || child.Kind == KindConstructor {
					calls, err := a.client.ExtractCallsForSymbol(file, child.LineStart-1, 0)
					if err == nil && len(calls) > 0 {
						mu.Lock()
						allCalls = append(allCalls, calls...)
						mu.Unlock()
					}
				}
			}
		}

		a.client.CloseFile(file)
	}

	return allCalls, nil
}

// QuickCallScan performs a faster scan by only analyzing exported symbols
func (a *DartAnalyzer) QuickCallScan(files []string, progressFn func(current, total int, file string)) ([]CallInfo, error) {
	var allCalls []CallInfo

	total := len(files)
	for i, file := range files {
		if progressFn != nil {
			progressFn(i+1, total, file)
		}

		if !strings.HasSuffix(file, ".dart") {
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		parser := NewDartParser()
		analysis, err := parser.Parse(content, file)
		if err != nil {
			continue
		}

		if err := a.client.OpenFile(file, string(content)); err != nil {
			continue
		}

		// Only analyze exported (public) symbols
		for _, sym := range analysis.Symbols {
			if !sym.Exported {
				continue
			}

			if sym.Kind == KindFunction || sym.Kind == KindClass {
				// For classes, find constructors and public methods
				if sym.Kind == KindClass {
					for _, child := range sym.Children {
						if child.Exported && (child.Kind == KindMethod || child.Kind == KindConstructor) {
							calls, _ := a.client.ExtractCallsForSymbol(file, child.LineStart-1, 0)
							allCalls = append(allCalls, calls...)
						}
					}
				} else {
					calls, _ := a.client.ExtractCallsForSymbol(file, sym.LineStart-1, 0)
					allCalls = append(allCalls, calls...)
				}
			}
		}

		a.client.CloseFile(file)
	}

	return allCalls, nil
}
