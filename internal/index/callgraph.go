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
	Scope         string     `json:"scope"`          // File or symbol name
	IncomingCalls []CallSite `json:"incomingCalls"`  // Who calls this
	OutgoingCalls []CallSite `json:"outgoingCalls"`  // What does this call
}

// GetIncomingCalls returns all locations that call the given symbol
// symbolName can be:
//   - Simple name: "parseConfig"
//   - Qualified name: "config.Parse"
func GetIncomingCalls(db *sql.DB, symbolName string) ([]CallSite, error) {
	// Search for calls where target_symbol matches or ends with the symbol name
	rows, err := db.Query(`
		SELECT r.source_file, r.line, r.target_symbol
		FROM relationships r
		WHERE r.kind = 'call'
		AND (r.target_symbol = ?
		     OR r.target_symbol LIKE ?
		     OR r.target_symbol LIKE ?)
		ORDER BY r.source_file, r.line;
	`, symbolName, "%."+symbolName, "%::"+symbolName)
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
