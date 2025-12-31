package index

import (
	"database/sql"
	"fmt"
)

// CallSite represents a location where a function is called
type CallSite struct {
	FilePath     string `json:"filePath"`
	Line         int    `json:"line"`
	CallerSymbol string `json:"callerSymbol,omitempty"` // The function that contains this call
	CalleeSymbol string `json:"calleeSymbol"`           // The function being called
}

// CallGraph represents the complete call graph for a scope
type CallGraph struct {
	Scope         string     `json:"scope"`         // File or symbol name
	IncomingCalls []CallSite `json:"incomingCalls"` // Who calls this
	OutgoingCalls []CallSite `json:"outgoingCalls"` // What does this call
}

// GetIncomingCalls returns all locations that call the given symbol
// symbolName can be:
//   - Simple name: "parseConfig"
//   - Qualified name: "config.Parse"
//   - Dart getter: "get userId" (searches for "userId")
func GetIncomingCalls(db *sql.DB, symbolName string) ([]CallSite, error) {
	// Search for calls where target_symbol matches or ends with the symbol name
	// Also handles Dart patterns like "get foo", "set foo"
	rows, err := db.Query(`
		SELECT r.source_file, r.line, r.target_symbol
		FROM relationships r
		WHERE r.kind = 'call'
		AND (r.target_symbol = ?
		     OR r.target_symbol LIKE ?
		     OR r.target_symbol LIKE ?
		     OR r.target_symbol LIKE ?
		     OR r.target_symbol LIKE ?
		     OR r.target_symbol LIKE ?)
		ORDER BY r.source_file, r.line;
	`, symbolName, "%."+symbolName, "%::"+symbolName, "% "+symbolName, "%/"+symbolName+".dart", symbolName+"%")
	if err != nil {
		return nil, fmt.Errorf("query incoming calls: %w", err)
	}
	defer rows.Close()

	var calls []CallSite
	for rows.Next() {
		var cs CallSite
		if err := rows.Scan(&cs.FilePath, &cs.Line, &cs.CalleeSymbol); err != nil {
			return nil, err
		}

		// Try to find the enclosing function for this call
		cs.CallerSymbol = findEnclosingSymbol(db, cs.FilePath, cs.Line)
		calls = append(calls, cs)
	}

	return calls, rows.Err()
}

// GetOutgoingCalls returns all functions/methods called by the given symbol
func GetOutgoingCalls(db *sql.DB, symbolName string, filePath string) ([]CallSite, error) {
	// First, find the symbol to get its line range
	var startLine, endLine int
	err := db.QueryRow(`
		SELECT line_start, line_end
		FROM symbols
		WHERE name = ? AND (file_path = ? OR ? = '')
		LIMIT 1;
	`, symbolName, filePath, filePath).Scan(&startLine, &endLine)
	if err != nil {
		return nil, fmt.Errorf("symbol not found: %s", symbolName)
	}

	// Find all calls within that line range
	query := `
		SELECT r.source_file, r.line, r.target_symbol
		FROM relationships r
		WHERE r.kind = 'call'
		AND r.source_file = ?
		AND r.line >= ? AND r.line <= ?
		ORDER BY r.line;
	`
	rows, err := db.Query(query, filePath, startLine, endLine)
	if err != nil {
		return nil, fmt.Errorf("query outgoing calls: %w", err)
	}
	defer rows.Close()

	var calls []CallSite
	for rows.Next() {
		var cs CallSite
		cs.CallerSymbol = symbolName
		if err := rows.Scan(&cs.FilePath, &cs.Line, &cs.CalleeSymbol); err != nil {
			return nil, err
		}
		calls = append(calls, cs)
	}

	return calls, rows.Err()
}

// GetCallGraph returns the complete call graph for a file
func GetCallGraph(db *sql.DB, filePath string) (*CallGraph, error) {
	result := &CallGraph{
		Scope: filePath,
	}

	// Get all calls made from this file
	outRows, err := db.Query(`
		SELECT source_file, line, target_symbol
		FROM relationships
		WHERE kind = 'call' AND source_file = ?
		ORDER BY line;
	`, filePath)
	if err != nil {
		return nil, fmt.Errorf("query outgoing: %w", err)
	}
	defer outRows.Close()

	for outRows.Next() {
		var cs CallSite
		if err := outRows.Scan(&cs.FilePath, &cs.Line, &cs.CalleeSymbol); err != nil {
			return nil, err
		}
		cs.CallerSymbol = findEnclosingSymbol(db, cs.FilePath, cs.Line)
		result.OutgoingCalls = append(result.OutgoingCalls, cs)
	}

	// Get all calls to symbols defined in this file
	// First get all symbols in the file
	symRows, err := db.Query(`SELECT name FROM symbols WHERE file_path = ?;`, filePath)
	if err != nil {
		return nil, err
	}
	defer symRows.Close()

	var symbols []string
	for symRows.Next() {
		var name string
		if err := symRows.Scan(&name); err != nil {
			return nil, err
		}
		symbols = append(symbols, name)
	}

	// Find calls to each symbol
	for _, sym := range symbols {
		inCalls, err := GetIncomingCalls(db, sym)
		if err != nil {
			continue
		}
		// Filter out self-calls from the same file
		for _, call := range inCalls {
			if call.FilePath != filePath {
				result.IncomingCalls = append(result.IncomingCalls, call)
			}
		}
	}

	return result, nil
}

// findEnclosingSymbol finds the function/method that contains the given line
func findEnclosingSymbol(db *sql.DB, filePath string, line int) string {
	var name string
	err := db.QueryRow(`
		SELECT name FROM symbols
		WHERE file_path = ?
		AND line_start <= ? AND line_end >= ?
		AND kind IN ('function', 'method')
		ORDER BY (line_end - line_start) ASC
		LIMIT 1;
	`, filePath, line, line).Scan(&name)
	if err != nil {
		return ""
	}
	return name
}

// GetCallersCount returns the number of places a symbol is called
func GetCallersCount(db *sql.DB, symbolName string) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM relationships
		WHERE kind = 'call'
		AND (target_symbol = ? OR target_symbol LIKE ? OR target_symbol LIKE ?)
	`, symbolName, "%."+symbolName, "%::"+symbolName).Scan(&count)
	return count, err
}

// GetMostCalledSymbols returns the most frequently called symbols
func GetMostCalledSymbols(db *sql.DB, limit int) ([]struct {
	Symbol string
	Count  int
}, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := db.Query(`
		SELECT target_symbol, COUNT(*) as call_count
		FROM relationships
		WHERE kind = 'call'
		GROUP BY target_symbol
		ORDER BY call_count DESC
		LIMIT ?;
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []struct {
		Symbol string
		Count  int
	}
	for rows.Next() {
		var r struct {
			Symbol string
			Count  int
		}
		if err := rows.Scan(&r.Symbol, &r.Count); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

// CallChainNode represents a node in the call chain tree
type CallChainNode struct {
	Symbol   string           `json:"symbol"`
	FilePath string           `json:"filePath,omitempty"`
	Line     int              `json:"line,omitempty"`
	Depth    int              `json:"depth"`
	Children []*CallChainNode `json:"children,omitempty"`
}

// CallChainResult represents the result of a call chain trace
type CallChainResult struct {
	Target    string           `json:"target"`
	Direction string           `json:"direction"` // "up", "down", or "both"
	MaxDepth  int              `json:"maxDepth"`
	Chains    []*CallChainNode `json:"chains"`
	TotalPaths int             `json:"totalPaths"`
	Truncated  bool            `json:"truncated,omitempty"`
}

// GetCallChainUp traces callers recursively up to maxDepth
// Returns all paths from entry points down to the target symbol
func GetCallChainUp(db *sql.DB, symbolName string, maxDepth int) (*CallChainResult, error) {
	if maxDepth <= 0 {
		maxDepth = 3
	}
	if maxDepth > 10 {
		maxDepth = 10 // Cap to prevent excessive queries
	}

	result := &CallChainResult{
		Target:    symbolName,
		Direction: "up",
		MaxDepth:  maxDepth,
	}

	// Track visited symbols to detect cycles
	visited := make(map[string]bool)
	totalPaths := 0
	maxPaths := 100 // Limit total paths to prevent explosion

	// Build chains recursively
	var buildChainUp func(symbol string, depth int) []*CallChainNode
	buildChainUp = func(symbol string, depth int) []*CallChainNode {
		if depth > maxDepth || totalPaths >= maxPaths {
			if totalPaths >= maxPaths {
				result.Truncated = true
			}
			return nil
		}

		// Check for cycles
		if visited[symbol] {
			return nil
		}
		visited[symbol] = true
		defer func() { visited[symbol] = false }()

		// Get direct callers
		callers, err := GetIncomingCalls(db, symbol)
		if err != nil || len(callers) == 0 {
			return nil
		}

		// Deduplicate callers by symbol name
		seenCallers := make(map[string]CallSite)
		for _, c := range callers {
			if c.CallerSymbol != "" {
				if existing, ok := seenCallers[c.CallerSymbol]; !ok || c.Line < existing.Line {
					seenCallers[c.CallerSymbol] = c
				}
			}
		}

		var nodes []*CallChainNode
		for callerSym, caller := range seenCallers {
			if totalPaths >= maxPaths {
				result.Truncated = true
				break
			}

			node := &CallChainNode{
				Symbol:   callerSym,
				FilePath: caller.FilePath,
				Line:     caller.Line,
				Depth:    depth,
			}

			// Recursively get callers of this caller
			node.Children = buildChainUp(callerSym, depth+1)

			// If no more callers, this is a root (entry point)
			if len(node.Children) == 0 {
				totalPaths++
			}

			nodes = append(nodes, node)
		}

		return nodes
	}

	result.Chains = buildChainUp(symbolName, 1)
	result.TotalPaths = totalPaths

	return result, nil
}

// GetCallChainDown traces callees recursively down to maxDepth
// Returns all paths from the target symbol to leaf functions
func GetCallChainDown(db *sql.DB, symbolName string, filePath string, maxDepth int) (*CallChainResult, error) {
	if maxDepth <= 0 {
		maxDepth = 3
	}
	if maxDepth > 10 {
		maxDepth = 10
	}

	result := &CallChainResult{
		Target:    symbolName,
		Direction: "down",
		MaxDepth:  maxDepth,
	}

	visited := make(map[string]bool)
	totalPaths := 0
	maxPaths := 100

	var buildChainDown func(symbol string, file string, depth int) []*CallChainNode
	buildChainDown = func(symbol string, file string, depth int) []*CallChainNode {
		if depth > maxDepth || totalPaths >= maxPaths {
			if totalPaths >= maxPaths {
				result.Truncated = true
			}
			return nil
		}

		key := symbol + ":" + file
		if visited[key] {
			return nil
		}
		visited[key] = true
		defer func() { visited[key] = false }()

		// Get direct callees
		callees, err := GetOutgoingCalls(db, symbol, file)
		if err != nil || len(callees) == 0 {
			return nil
		}

		// Deduplicate callees by symbol name
		seenCallees := make(map[string]CallSite)
		for _, c := range callees {
			if c.CalleeSymbol != "" {
				if _, ok := seenCallees[c.CalleeSymbol]; !ok {
					seenCallees[c.CalleeSymbol] = c
				}
			}
		}

		var nodes []*CallChainNode
		for calleeSym, callee := range seenCallees {
			if totalPaths >= maxPaths {
				result.Truncated = true
				break
			}

			node := &CallChainNode{
				Symbol:   calleeSym,
				FilePath: callee.FilePath,
				Line:     callee.Line,
				Depth:    depth,
			}

			// Find file where callee is defined to continue tracing
			calleeFile := findSymbolFile(db, calleeSym)
			if calleeFile != "" {
				node.Children = buildChainDown(calleeSym, calleeFile, depth+1)
			}

			if len(node.Children) == 0 {
				totalPaths++
			}

			nodes = append(nodes, node)
		}

		return nodes
	}

	// If no file provided, try to find it
	if filePath == "" {
		filePath = findSymbolFile(db, symbolName)
	}

	if filePath != "" {
		result.Chains = buildChainDown(symbolName, filePath, 1)
	}
	result.TotalPaths = totalPaths

	return result, nil
}

// findSymbolFile finds the file where a symbol is defined
func findSymbolFile(db *sql.DB, symbolName string) string {
	var filePath string
	err := db.QueryRow(`
		SELECT file_path FROM symbols
		WHERE name = ?
		LIMIT 1;
	`, symbolName).Scan(&filePath)
	if err != nil {
		return ""
	}
	return filePath
}

// GetCallChain traces calls in the specified direction
func GetCallChain(db *sql.DB, symbolName string, filePath string, direction string, maxDepth int) (*CallChainResult, error) {
	switch direction {
	case "up":
		return GetCallChainUp(db, symbolName, maxDepth)
	case "down":
		return GetCallChainDown(db, symbolName, filePath, maxDepth)
	case "both":
		// Get both directions and merge
		upResult, err := GetCallChainUp(db, symbolName, maxDepth)
		if err != nil {
			return nil, err
		}
		downResult, err := GetCallChainDown(db, symbolName, filePath, maxDepth)
		if err != nil {
			return nil, err
		}

		return &CallChainResult{
			Target:     symbolName,
			Direction:  "both",
			MaxDepth:   maxDepth,
			Chains:     append(upResult.Chains, downResult.Chains...),
			TotalPaths: upResult.TotalPaths + downResult.TotalPaths,
			Truncated:  upResult.Truncated || downResult.Truncated,
		}, nil
	default:
		return GetCallChainUp(db, symbolName, maxDepth)
	}
}

// FlattenCallChain converts a tree of call chains into flat paths
// Each path is a slice of symbols from root to leaf
func FlattenCallChain(result *CallChainResult) [][]CallChainNode {
	var paths [][]CallChainNode

	var flatten func(nodes []*CallChainNode, currentPath []CallChainNode)
	flatten = func(nodes []*CallChainNode, currentPath []CallChainNode) {
		for _, node := range nodes {
			newPath := append(currentPath, CallChainNode{
				Symbol:   node.Symbol,
				FilePath: node.FilePath,
				Line:     node.Line,
				Depth:    node.Depth,
			})

			if len(node.Children) == 0 {
				// Leaf node - save the path
				paths = append(paths, newPath)
			} else {
				flatten(node.Children, newPath)
			}
		}
	}

	flatten(result.Chains, nil)
	return paths
}
