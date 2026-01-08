package index

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
)

// ContextResult represents the complete context for a task
type ContextResult struct {
	Query      string        `json:"query"`
	Files      []FileContext `json:"files"`
	Symbols    []SymbolInfo  `json:"symbols"`
	Imports    []ImportInfo  `json:"imports"`
	Decisions  []Decision    `json:"decisions,omitempty"`
	Warnings   []string      `json:"warnings,omitempty"`
	TotalFiles int           `json:"totalFiles"`
	TokenStats *TokenStats   `json:"tokenStats,omitempty"`
}

// TokenStats reports token usage in the context result
type TokenStats struct {
	TotalTokens  int  `json:"totalTokens"`
	SymbolTokens int  `json:"symbolTokens"`
	FileTokens   int  `json:"fileTokens"`
	ImportTokens int  `json:"importTokens"`
	Budget       int  `json:"budget,omitempty"`
	Truncated    bool `json:"truncated,omitempty"`
}

// FileContext represents a file with relevant context
type FileContext struct {
	Path       string       `json:"path"`
	Language   string       `json:"language"`
	Relevance  float64      `json:"relevance"`
	Symbols    []SymbolInfo `json:"symbols,omitempty"`
	ChunkStart int          `json:"chunkStart,omitempty"`
	ChunkEnd   int          `json:"chunkEnd,omitempty"`
	Snippet    string       `json:"snippet,omitempty"`
}

// SymbolInfo represents a symbol with its metadata
type SymbolInfo struct {
	Name       string       `json:"name"`
	Kind       string       `json:"kind"`
	FilePath   string       `json:"filePath"`
	LineStart  int          `json:"lineStart"`
	LineEnd    int          `json:"lineEnd"`
	Signature  string       `json:"signature,omitempty"`
	DocComment string       `json:"docComment,omitempty"`
	Exported   bool         `json:"exported"`
	Children   []SymbolInfo `json:"children,omitempty"`
}

// ImportInfo represents an import relationship
type ImportInfo struct {
	SourceFile   string `json:"sourceFile"`
	TargetFile   string `json:"targetFile"`
	TargetSymbol string `json:"targetSymbol,omitempty"`
	Kind         string `json:"kind"`
	Line         int    `json:"line"`
}

// Decision represents an architectural decision
type Decision struct {
	ID            string   `json:"id"`
	Room          string   `json:"room,omitempty"`
	Title         string   `json:"title"`
	Summary       string   `json:"summary"`
	Rationale     string   `json:"rationale,omitempty"`
	AffectedFiles []string `json:"affectedFiles,omitempty"`
	CreatedAt     string   `json:"createdAt"`
}

// ContextOptions configures how context is assembled
type ContextOptions struct {
	ExcludePatterns []string `json:"excludePatterns,omitempty"` // Glob patterns to exclude (e.g., "*_test.go")
	ExcludeKinds    []string `json:"excludeKinds,omitempty"`    // Symbol kinds to exclude (e.g., "test")
	MaxTokens       int      `json:"maxTokens,omitempty"`       // Token budget (0 = no limit)
	IncludeTests    bool     `json:"includeTests,omitempty"`    // Include test files (default: false)
}

// DefaultExcludePatterns returns patterns for files typically excluded from AI context
var DefaultExcludePatterns = []string{
	// Test files
	"*_test.go",
	"*_test.ts",
	"*_test.tsx",
	"*_test.js",
	"*_test.jsx",
	"*.test.ts",
	"*.test.tsx",
	"*.test.js",
	"*.test.jsx",
	"*.spec.ts",
	"*.spec.tsx",
	"*.spec.js",
	"*.spec.jsx",
	"test_*.py",
	"*_test.py",
	"*_test.rb",
	"*_spec.rb",
	// Generated files
	"*.pb.go",
	"*.pb.ts",
	"*.generated.go",
	"*.generated.ts",
	"*.gen.go",
	"*.g.dart",
	// Dependencies and lock files
	"package-lock.json",
	"yarn.lock",
	"pnpm-lock.yaml",
	"go.sum",
	"Cargo.lock",
	"Gemfile.lock",
	"poetry.lock",
	"composer.lock",
	// Vendor/dependencies
	"vendor/*",
	"node_modules/*",
	".venv/*",
	"__pycache__/*",
	// Config/metadata
	".git/*",
	".palace/*",
	".vscode/*",
	".idea/*",
}

// shouldExcludeFile checks if a file should be excluded based on patterns
func shouldExcludeFile(path string, patterns []string) bool {
	baseName := filepath.Base(path)
	for _, pattern := range patterns {
		// Check against full path
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		// Check against basename
		if matched, _ := filepath.Match(pattern, baseName); matched {
			return true
		}
		// Check for directory patterns (e.g., "vendor/*")
		if strings.HasSuffix(pattern, "/*") {
			dir := strings.TrimSuffix(pattern, "/*")
			if strings.Contains(path, "/"+dir+"/") || strings.HasPrefix(path, dir+"/") {
				return true
			}
		}
	}
	return false
}

// GetContextForTask returns complete context for a given task description
func GetContextForTask(db *sql.DB, query string, limit int) (*ContextResult, error) {
	return GetContextForTaskWithOptions(db, query, limit, nil)
}

// GetContextForTaskWithOptions returns context with custom filtering options
func GetContextForTaskWithOptions(db *sql.DB, query string, limit int, opts *ContextOptions) (*ContextResult, error) {
	if limit <= 0 {
		limit = 20
	}

	// Set up exclusion patterns
	var excludePatterns []string
	if opts != nil && len(opts.ExcludePatterns) > 0 {
		excludePatterns = opts.ExcludePatterns
	} else if opts == nil || !opts.IncludeTests {
		// Use defaults unless explicitly including tests
		excludePatterns = DefaultExcludePatterns
	}

	result := &ContextResult{
		Query: query,
	}

	// Search symbols by name and doc comments
	symbols, err := searchSymbols(db, query, limit*2) // Fetch more to account for filtering
	if err != nil {
		return nil, fmt.Errorf("search symbols: %w", err)
	}

	// Filter symbols based on exclusion patterns
	var filteredSymbols []SymbolInfo
	for i := range symbols {
		sym := &symbols[i]
		if !shouldExcludeFile(sym.FilePath, excludePatterns) {
			filteredSymbols = append(filteredSymbols, *sym)
			if len(filteredSymbols) >= limit {
				break
			}
		}
	}
	result.Symbols = filteredSymbols

	// Get files containing matched symbols
	fileSet := make(map[string]bool)
	for i := range filteredSymbols {
		fileSet[filteredSymbols[i].FilePath] = true
	}

	// Also search content for query terms
	chunks, err := SearchChunks(db, query, limit*2) // Fetch more to account for filtering
	if err != nil {
		return nil, fmt.Errorf("search chunks: %w", err)
	}

	// Build file contexts (with filtering)
	fileContexts := make(map[string]*FileContext)
	chunkCount := 0
	for _, chunk := range chunks {
		// Skip excluded files
		if shouldExcludeFile(chunk.Path, excludePatterns) {
			continue
		}

		fc, exists := fileContexts[chunk.Path]
		if !exists {
			if chunkCount >= limit {
				continue
			}
			lang, _ := getFileLanguage(db, chunk.Path)
			fc = &FileContext{
				Path:       chunk.Path,
				Language:   lang,
				Relevance:  1.0,
				ChunkStart: chunk.StartLine,
				ChunkEnd:   chunk.EndLine,
				Snippet:    truncateSnippet(chunk.Content, 500),
			}
			fileContexts[chunk.Path] = fc
			chunkCount++
		} else {
			// Extend range if this chunk is adjacent
			if chunk.StartLine < fc.ChunkStart {
				fc.ChunkStart = chunk.StartLine
			}
			if chunk.EndLine > fc.ChunkEnd {
				fc.ChunkEnd = chunk.EndLine
			}
		}
		fileSet[chunk.Path] = true
	}

	// Get symbols for each relevant file
	for path := range fileSet {
		// Skip excluded files
		if shouldExcludeFile(path, excludePatterns) {
			continue
		}

		fc, exists := fileContexts[path]
		if !exists {
			lang, _ := getFileLanguage(db, path)
			fc = &FileContext{
				Path:      path,
				Language:  lang,
				Relevance: 0.8, // Lower relevance for files found only via symbols
			}
			fileContexts[path] = fc
		}

		fileSymbols, err := getSymbolsForFile(db, path)
		if err == nil {
			fc.Symbols = fileSymbols
		}
	}

	// Convert map to slice
	for _, fc := range fileContexts {
		result.Files = append(result.Files, *fc)
	}

	// Get imports for relevant files
	for path := range fileSet {
		imports, err := getImportsForFile(db, path)
		if err == nil {
			result.Imports = append(result.Imports, imports...)
		}
	}

	// Get decisions if any match
	decisions, err := searchDecisions(db, query)
	if err == nil {
		result.Decisions = decisions
	}

	result.TotalFiles = len(result.Files)

	// Apply token budgeting if configured
	if opts != nil && opts.MaxTokens > 0 {
		result = applyTokenBudget(result, opts.MaxTokens)
	}

	return result, nil
}

// applyTokenBudget truncates context to fit within a token budget
func applyTokenBudget(result *ContextResult, budget int) *ContextResult {
	stats := &TokenStats{Budget: budget}

	// Estimate current token usage
	symbolTokens := 0
	for i := range result.Symbols {
		symbolTokens += estimateSymbolTokens(result.Symbols[i])
	}

	fileTokens := 0
	for _, fc := range result.Files {
		fileTokens += estimateFileContextTokens(fc)
	}

	importTokens := 0
	for _, imp := range result.Imports {
		importTokens += EstimateTokens(imp.SourceFile + imp.TargetFile + imp.Kind)
	}

	totalTokens := symbolTokens + fileTokens + importTokens

	// If within budget, just report stats
	if totalTokens <= budget {
		stats.TotalTokens = totalTokens
		stats.SymbolTokens = symbolTokens
		stats.FileTokens = fileTokens
		stats.ImportTokens = importTokens
		result.TokenStats = stats
		return result
	}

	// Need to truncate - prioritize symbols and files over imports
	stats.Truncated = true

	// Allocate budget: 50% files, 40% symbols, 10% imports
	fileBudget := budget * 50 / 100
	symbolBudget := budget * 40 / 100
	importBudget := budget * 10 / 100

	// Truncate files
	truncatedFiles := truncateFileContexts(result.Files, fileBudget)
	result.Files = truncatedFiles
	fileTokens = 0
	for i := range result.Files {
		fileTokens += estimateFileContextTokens(result.Files[i])
	}

	// Truncate symbols
	result.Symbols = TruncateSymbols(result.Symbols, symbolBudget)
	symbolTokens = 0
	for i := range result.Symbols {
		symbolTokens += estimateSymbolTokens(result.Symbols[i])
	}

	// Truncate imports
	result.Imports = truncateImports(result.Imports, importBudget)
	importTokens = 0
	for _, imp := range result.Imports {
		importTokens += EstimateTokens(imp.SourceFile + imp.TargetFile + imp.Kind)
	}

	stats.TotalTokens = symbolTokens + fileTokens + importTokens
	stats.SymbolTokens = symbolTokens
	stats.FileTokens = fileTokens
	stats.ImportTokens = importTokens
	result.TokenStats = stats
	result.TotalFiles = len(result.Files)

	if stats.TotalTokens > budget {
		result.Warnings = append(result.Warnings, "Context truncated to fit token budget")
	}

	return result
}

// estimateSymbolTokens estimates tokens for a symbol
func estimateSymbolTokens(sym SymbolInfo) int {
	text := sym.Name + " " + sym.Kind + " " + sym.FilePath
	if sym.Signature != "" {
		text += " " + sym.Signature
	}
	if sym.DocComment != "" {
		text += " " + sym.DocComment
	}
	return EstimateTokens(text)
}

// estimateFileContextTokens estimates tokens for a file context
func estimateFileContextTokens(fc FileContext) int {
	tokens := EstimateTokens(fc.Path + " " + fc.Language)
	if fc.Snippet != "" {
		tokens += EstimateTokens(fc.Snippet)
	}
	for i := range fc.Symbols {
		tokens += estimateSymbolTokens(fc.Symbols[i])
	}
	return tokens
}

// truncateFileContexts truncates file contexts to fit within budget
func truncateFileContexts(files []FileContext, budget int) []FileContext {
	if len(files) == 0 {
		return files
	}

	budgeted := make([]BudgetedItem, len(files))
	for i, fc := range files {
		budgeted[i] = BudgetedItem{
			Item:       fc,
			TokenCount: estimateFileContextTokens(fc),
			Priority:   fc.Relevance,
		}
	}

	truncated := TruncateToTokenBudget(budgeted, budget)
	result := make([]FileContext, len(truncated))
	for i, b := range truncated {
		result[i] = b.Item.(FileContext)
	}
	return result
}

// truncateImports truncates imports to fit within budget
func truncateImports(imports []ImportInfo, budget int) []ImportInfo {
	if len(imports) == 0 || budget <= 0 {
		return imports
	}

	used := 0
	var result []ImportInfo
	for _, imp := range imports {
		tokens := EstimateTokens(imp.SourceFile + imp.TargetFile + imp.Kind)
		if used+tokens <= budget {
			result = append(result, imp)
			used += tokens
		}
	}
	return result
}

// searchSymbols searches for symbols matching the query
func searchSymbols(db *sql.DB, query string, limit int) ([]SymbolInfo, error) {
	escaped := sanitizeFTSQuery(query)
	rows, err := db.QueryContext(context.Background(), `
		SELECT s.name, s.kind, s.file_path, s.line_start, s.line_end, s.signature, s.doc_comment, s.exported
		FROM symbols_fts
		JOIN symbols s ON s.name = symbols_fts.name AND s.file_path = symbols_fts.file_path
		WHERE symbols_fts MATCH ?
		ORDER BY s.file_path, s.line_start
		LIMIT ?;
	`, escaped, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []SymbolInfo
	for rows.Next() {
		var sym SymbolInfo
		var exported int
		if err := rows.Scan(&sym.Name, &sym.Kind, &sym.FilePath, &sym.LineStart, &sym.LineEnd, &sym.Signature, &sym.DocComment, &exported); err != nil {
			return nil, err
		}
		sym.Exported = exported == 1
		symbols = append(symbols, sym)
	}
	return symbols, rows.Err()
}

// getSymbolsForFile returns all symbols in a file
func getSymbolsForFile(db *sql.DB, path string) ([]SymbolInfo, error) {
	rows, err := db.QueryContext(context.Background(), `
		SELECT id, name, kind, line_start, line_end, signature, doc_comment, parent_id, exported
		FROM symbols
		WHERE file_path = ?
		ORDER BY line_start;
	`, path)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type rawSymbol struct {
		ID         int64
		ParentID   sql.NullInt64
		SymbolInfo SymbolInfo
	}

	var rawSymbols []rawSymbol
	for rows.Next() {
		var rs rawSymbol
		var exported int
		if err := rows.Scan(&rs.ID, &rs.SymbolInfo.Name, &rs.SymbolInfo.Kind, &rs.SymbolInfo.LineStart, &rs.SymbolInfo.LineEnd, &rs.SymbolInfo.Signature, &rs.SymbolInfo.DocComment, &rs.ParentID, &exported); err != nil {
			return nil, err
		}
		rs.SymbolInfo.FilePath = path
		rs.SymbolInfo.Exported = exported == 1
		rawSymbols = append(rawSymbols, rs)
	}

	// Build hierarchy
	symbolMap := make(map[int64]*SymbolInfo)
	var topLevel []SymbolInfo

	for i := range rawSymbols {
		sym := &rawSymbols[i].SymbolInfo
		symbolMap[rawSymbols[i].ID] = sym
	}

	for i := range rawSymbols {
		if rawSymbols[i].ParentID.Valid {
			parent, ok := symbolMap[rawSymbols[i].ParentID.Int64]
			if ok {
				parent.Children = append(parent.Children, rawSymbols[i].SymbolInfo)
			}
		} else {
			topLevel = append(topLevel, rawSymbols[i].SymbolInfo)
		}
	}

	return topLevel, rows.Err()
}

// getImportsForFile returns all imports for a file
func getImportsForFile(db *sql.DB, path string) ([]ImportInfo, error) {
	rows, err := db.QueryContext(context.Background(), `
		SELECT source_file, target_file, target_symbol, kind, line
		FROM relationships
		WHERE source_file = ? AND kind = 'import'
		ORDER BY line;
	`, path)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var imports []ImportInfo
	for rows.Next() {
		var imp ImportInfo
		var targetSymbol sql.NullString
		if err := rows.Scan(&imp.SourceFile, &imp.TargetFile, &targetSymbol, &imp.Kind, &imp.Line); err != nil {
			return nil, err
		}
		if targetSymbol.Valid {
			imp.TargetSymbol = targetSymbol.String
		}
		imports = append(imports, imp)
	}
	return imports, rows.Err()
}

// searchDecisions searches for decisions matching the query
func searchDecisions(db *sql.DB, query string) ([]Decision, error) {
	// Simple LIKE search for decisions
	pattern := "%" + query + "%"
	rows, err := db.QueryContext(context.Background(), `
		SELECT id, room, title, summary, rationale, affected_files, created_at
		FROM decisions
		WHERE title LIKE ? OR summary LIKE ? OR rationale LIKE ?
		LIMIT 10;
	`, pattern, pattern, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []Decision
	for rows.Next() {
		var d Decision
		var room, rationale, affectedFiles sql.NullString
		if err := rows.Scan(&d.ID, &room, &d.Title, &d.Summary, &rationale, &affectedFiles, &d.CreatedAt); err != nil {
			return nil, err
		}
		if room.Valid {
			d.Room = room.String
		}
		if rationale.Valid {
			d.Rationale = rationale.String
		}
		decisions = append(decisions, d)
	}
	return decisions, rows.Err()
}

// getFileLanguage returns the language of a file
func getFileLanguage(db *sql.DB, path string) (string, error) {
	var lang string
	err := db.QueryRowContext(context.Background(), `SELECT language FROM files WHERE path = ?;`, path).Scan(&lang)
	return lang, err
}

// truncateSnippet truncates a snippet to maxLen chars
func truncateSnippet(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ImpactResult contains the impact analysis for a file or symbol.
type ImpactResult struct {
	Target       string       `json:"target"`
	Dependents   []string     `json:"dependents"`
	Dependencies []string     `json:"dependencies"`
	Symbols      []SymbolInfo `json:"symbols"`
}

// GetImpact analyzes what would be affected by changing a file or symbol
func GetImpact(db *sql.DB, target string) (*ImpactResult, error) {
	result := &ImpactResult{
		Target: target,
	}

	// Find files that import this target
	rows, err := db.QueryContext(context.Background(), `
		SELECT DISTINCT source_file
		FROM relationships
		WHERE target_file LIKE ? AND kind = 'import';
	`, "%"+target+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var sourceFile string
		if err := rows.Scan(&sourceFile); err != nil {
			return nil, err
		}
		result.Dependents = append(result.Dependents, sourceFile)
	}

	// Find files that this target imports
	rows2, err := db.QueryContext(context.Background(), `
		SELECT DISTINCT target_file
		FROM relationships
		WHERE source_file = ? AND kind = 'import';
	`, target)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	for rows2.Next() {
		var targetFile string
		if err := rows2.Scan(&targetFile); err != nil {
			return nil, err
		}
		result.Dependencies = append(result.Dependencies, targetFile)
	}

	// Get symbols in the target file
	symbols, err := getSymbolsForFile(db, target)
	if err == nil {
		result.Symbols = symbols
	}

	return result, nil
}

// SearchSymbolsByKind searches for symbols of a specific kind
func SearchSymbolsByKind(db *sql.DB, kind string, limit int) ([]SymbolInfo, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.QueryContext(context.Background(), `
		SELECT name, kind, file_path, line_start, line_end, signature, doc_comment, exported
		FROM symbols
		WHERE kind = ?
		ORDER BY file_path, line_start
		LIMIT ?;
	`, kind, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []SymbolInfo
	for rows.Next() {
		var sym SymbolInfo
		var exported int
		if err := rows.Scan(&sym.Name, &sym.Kind, &sym.FilePath, &sym.LineStart, &sym.LineEnd, &sym.Signature, &sym.DocComment, &exported); err != nil {
			return nil, err
		}
		sym.Exported = exported == 1
		symbols = append(symbols, sym)
	}
	return symbols, rows.Err()
}

// GetSymbol returns a specific symbol by name and file
func GetSymbol(db *sql.DB, name, filePath string) (*SymbolInfo, error) {
	var sym SymbolInfo
	var exported int

	query := `
		SELECT name, kind, file_path, line_start, line_end, signature, doc_comment, exported
		FROM symbols
		WHERE name = ?`
	args := []any{name}

	if filePath != "" {
		query += ` AND file_path = ?`
		args = append(args, filePath)
	}
	query += ` LIMIT 1;`

	err := db.QueryRowContext(context.Background(), query, args...).Scan(&sym.Name, &sym.Kind, &sym.FilePath, &sym.LineStart, &sym.LineEnd, &sym.Signature, &sym.DocComment, &exported)
	if err != nil {
		return nil, err
	}
	sym.Exported = exported == 1
	return &sym, nil
}

// ListExportedSymbols returns all exported symbols for a file
func ListExportedSymbols(db *sql.DB, filePath string) ([]SymbolInfo, error) {
	rows, err := db.QueryContext(context.Background(), `
		SELECT name, kind, file_path, line_start, line_end, signature, doc_comment, exported
		FROM symbols
		WHERE file_path = ? AND exported = 1
		ORDER BY line_start;
	`, filePath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []SymbolInfo
	for rows.Next() {
		var sym SymbolInfo
		var exported int
		if err := rows.Scan(&sym.Name, &sym.Kind, &sym.FilePath, &sym.LineStart, &sym.LineEnd, &sym.Signature, &sym.DocComment, &exported); err != nil {
			return nil, err
		}
		sym.Exported = exported == 1
		symbols = append(symbols, sym)
	}
	return symbols, rows.Err()
}

// DependencyNode represents a node in the import dependency graph.
type DependencyNode struct {
	File     string   `json:"file"`
	Language string   `json:"language"`
	Imports  []string `json:"imports"`
}

func GetDependencyGraph(db *sql.DB, rootFiles []string) ([]DependencyNode, error) {
	visited := make(map[string]bool)
	var nodes []DependencyNode

	visit := func(file string) error {
		if visited[file] {
			return nil
		}
		visited[file] = true

		lang, _ := getFileLanguage(db, file)
		node := DependencyNode{
			File:     file,
			Language: lang,
		}

		imports, err := getImportsForFile(db, file)
		if err != nil {
			return err
		}

		for _, imp := range imports {
			node.Imports = append(node.Imports, imp.TargetFile)
		}

		nodes = append(nodes, node)
		return nil
	}

	for _, f := range rootFiles {
		if err := visit(f); err != nil {
			return nil, err
		}
	}

	return nodes, nil
}

// RecordDecision stores an architectural decision
func RecordDecision(db *sql.DB, decision Decision) error {
	_, err := db.ExecContext(context.Background(), `
		INSERT OR REPLACE INTO decisions (id, room, title, summary, rationale, affected_files, created_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?);
	`, decision.ID, decision.Room, decision.Title, decision.Summary, decision.Rationale, strings.Join(decision.AffectedFiles, ","), decision.CreatedAt, "")
	return err
}
