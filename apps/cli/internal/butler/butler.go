package butler

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
)

type SearchResult struct {
	Path       string  `json:"path"`
	Room       string  `json:"room,omitempty"`
	ChunkIndex int     `json:"chunkIndex"`
	StartLine  int     `json:"startLine"`
	EndLine    int     `json:"endLine"`
	Snippet    string  `json:"snippet"`
	Score      float64 `json:"score"`
	IsEntry    bool    `json:"isEntry,omitempty"`
}

type GroupedResults struct {
	Room    string         `json:"room"`
	Summary string         `json:"summary,omitempty"`
	Results []SearchResult `json:"results"`
}

type SearchOptions struct {
	Limit      int    // Maximum results (default 20)
	RoomFilter string // Optional: filter to specific room
	FuzzyMatch bool   // Enable fuzzy matching for typo tolerance
}

type Butler struct {
	db          *sql.DB
	root        string
	rooms       map[string]model.Room // cached room manifests
	entryPoints map[string]string     // path -> room name for entry points
	config      *config.PalaceConfig
	memory      *memory.Memory // session memory (optional, may be nil)
}

func New(db *sql.DB, root string) (*Butler, error) {
	b := &Butler{
		db:          db,
		root:        root,
		rooms:       make(map[string]model.Room),
		entryPoints: make(map[string]string),
	}

	cfg, err := config.LoadPalaceConfig(root)
	if err != nil {
		// Not fatal - use defaults
		b.config = nil
	} else {
		b.config = cfg
	}

	if err := b.loadRooms(); err != nil {
		return nil, fmt.Errorf("load rooms: %w", err)
	}

	// Initialize session memory (non-fatal if fails)
	mem, err := memory.Open(root)
	if err == nil {
		b.memory = mem
	}

	return b, nil
}

// Close closes the Butler's resources
func (b *Butler) Close() error {
	if b.memory != nil {
		return b.memory.Close()
	}
	return nil
}

func (b *Butler) loadRooms() error {
	roomsDir := filepath.Join(b.root, ".palace", "rooms")
	entries, err := filepath.Glob(filepath.Join(roomsDir, "*.jsonc"))
	if err != nil {
		return err
	}

	for _, path := range entries {
		var room model.Room
		if err := decodeJSONCFile(path, &room); err != nil {
			continue // Skip invalid room files
		}
		b.rooms[room.Name] = room
		for _, ep := range room.EntryPoints {
			b.entryPoints[ep] = room.Name
		}
	}
	return nil
}

func (b *Butler) Search(query string, opts SearchOptions) ([]GroupedResults, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}

	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	var ftsQuery string
	if opts.FuzzyMatch {
		ftsQuery = preprocessQueryWithFuzzy(query)
	} else {
		ftsQuery = preprocessQuery(query)
	}

	rows, err := b.db.Query(`
		SELECT 
			c.path,
			c.chunk_index,
			c.start_line,
			c.end_line,
			c.content,
			bm25(chunks_fts) as base_score
		FROM chunks_fts
		JOIN chunks c ON c.path = chunks_fts.path AND c.chunk_index = chunks_fts.chunk_index
		WHERE chunks_fts MATCH ?
		ORDER BY base_score
		LIMIT ?;
	`, ftsQuery, opts.Limit*3) // Over-fetch for re-ranking

	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var baseScore float64
		if err := rows.Scan(&r.Path, &r.ChunkIndex, &r.StartLine, &r.EndLine, &r.Snippet, &baseScore); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}

		r.Score = b.calculateScore(baseScore, r.Path, query)

		if roomName, ok := b.entryPoints[r.Path]; ok {
			r.Room = roomName
			r.IsEntry = true
		} else {
			r.Room = b.inferRoom(r.Path)
		}

		if opts.RoomFilter != "" && r.Room != opts.RoomFilter {
			continue
		}

		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return b.groupByRoom(results), nil
}

func (b *Butler) calculateScore(baseScore float64, path, query string) float64 {
	score := -baseScore

	if _, isEntry := b.entryPoints[path]; isEntry {
		score *= 3.0
	}

	queryLower := strings.ToLower(query)
	pathLower := strings.ToLower(path)
	if strings.Contains(pathLower, queryLower) {
		score *= 2.5
	} else {
		for _, word := range strings.Fields(query) {
			if len(word) > 2 && strings.Contains(pathLower, strings.ToLower(word)) {
				score *= 1.5
				break
			}
		}
	}

	ext := strings.ToLower(filepath.Ext(path))
	codeExts := map[string]bool{
		".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true,
		".py": true, ".rs": true, ".java": true, ".c": true, ".cpp": true,
		".rb": true, ".swift": true, ".kt": true,
	}
	if codeExts[ext] {
		score *= 1.2
	}

	return score
}

func (b *Butler) inferRoom(path string) string {
	// Check if path prefix matches any room's entry points' directories
	for name, room := range b.rooms {
		for _, ep := range room.EntryPoints {
			epDir := filepath.Dir(ep)
			if epDir != "." && strings.HasPrefix(path, epDir) {
				return name
			}
		}
	}

	// Default room from config
	if b.config != nil && b.config.DefaultRoom != "" {
		return b.config.DefaultRoom
	}

	return ""
}

func (b *Butler) groupByRoom(results []SearchResult) []GroupedResults {
	groups := make(map[string]*GroupedResults)
	order := []string{}

	for _, r := range results {
		roomName := r.Room
		if roomName == "" {
			roomName = "_ungrouped"
		}

		if _, exists := groups[roomName]; !exists {
			summary := ""
			if room, ok := b.rooms[roomName]; ok {
				summary = room.Summary
			}
			groups[roomName] = &GroupedResults{
				Room:    roomName,
				Summary: summary,
				Results: []SearchResult{},
			}
			order = append(order, roomName)
		}
		groups[roomName].Results = append(groups[roomName].Results, r)
	}

	// Convert to slice maintaining discovery order
	var grouped []GroupedResults
	for _, roomName := range order {
		grouped = append(grouped, *groups[roomName])
	}

	return grouped
}

func (b *Butler) ListRooms() []model.Room {
	rooms := make([]model.Room, 0, len(b.rooms))
	for _, room := range b.rooms {
		rooms = append(rooms, room)
	}
	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].Name < rooms[j].Name
	})
	return rooms
}

func (b *Butler) ReadFile(path string) (string, error) {
	// First, try to read from the chunks table for indexed content
	rows, err := b.db.Query(
		`SELECT content FROM chunks WHERE path = ? ORDER BY chunk_index ASC;`,
		path,
	)
	if err != nil {
		return "", fmt.Errorf("query chunks for %s: %w", path, err)
	}
	defer rows.Close()

	var parts []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return "", err
		}
		parts = append(parts, content)
	}

	if len(parts) == 0 {
		return "", fmt.Errorf("file not found in index: %s", path)
	}

	return strings.Join(parts, "\n"), nil
}

func (b *Butler) ReadRoom(name string) (*model.Room, error) {
	room, ok := b.rooms[name]
	if !ok {
		return nil, fmt.Errorf("room not found: %s", name)
	}
	return &room, nil
}

func preprocessQuery(query string) string {
	return preprocessQueryWithOptions(query, true) // Enable synonyms by default
}

// preprocessQueryWithFuzzy expands query with fuzzy variants
func preprocessQueryWithFuzzy(query string) string {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return ""
	}

	// First expand tokens (CamelCase, snake_case)
	tokens := expandQueryTokens(trimmed)
	if len(tokens) == 0 {
		escaped := strings.ReplaceAll(trimmed, "\"", "\"\"")
		return fmt.Sprintf("\"%s\"", escaped)
	}

	// Expand with synonyms
	tokens = expandWithSynonyms(tokens)

	// Add fuzzy variants for each token
	var allTerms []string
	for _, token := range tokens {
		if len(token) < 2 {
			continue
		}

		// Add the original token
		escaped := strings.ReplaceAll(token, "\"", "\"\"")
		allTerms = append(allTerms, fmt.Sprintf("\"%s\"*", escaped))

		// Add fuzzy variants for longer words
		if len(token) >= 5 {
			fuzzyVariants := ExpandWithFuzzyVariants(token, CommonProgrammingTerms)
			for _, variant := range fuzzyVariants {
				if variant != token {
					escaped := strings.ReplaceAll(variant, "\"", "\"\"")
					allTerms = append(allTerms, fmt.Sprintf("\"%s\"*", escaped))
				}
			}
		}
	}

	if len(allTerms) == 0 {
		escaped := strings.ReplaceAll(trimmed, "\"", "\"\"")
		return fmt.Sprintf("\"%s\"", escaped)
	}

	return strings.Join(allTerms, " OR ")
}

func preprocessQueryWithOptions(query string, useSynonyms bool) string {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return ""
	}

	// Check if query looks like code (exact match needed)
	isExactCodeQuery := strings.ContainsAny(trimmed, ".()[]{}") ||
		strings.Contains(trimmed, "::") ||
		strings.Contains(trimmed, "->")

	if isExactCodeQuery {
		// Exact phrase search with quote escaping
		escaped := strings.ReplaceAll(trimmed, "\"", "\"\"")
		return fmt.Sprintf("\"%s\"", escaped)
	}

	// Expand code identifiers (CamelCase, snake_case) into their parts
	expandedTokens := expandQueryTokens(trimmed)
	if len(expandedTokens) == 0 {
		escaped := strings.ReplaceAll(trimmed, "\"", "\"\"")
		return fmt.Sprintf("\"%s\"", escaped)
	}

	// Optionally expand with programming synonyms
	if useSynonyms {
		expandedTokens = expandWithSynonyms(expandedTokens)
	}

	var terms []string
	for _, word := range expandedTokens {
		if len(word) < 2 {
			continue
		}
		// Escape and add prefix operator
		escaped := strings.ReplaceAll(word, "\"", "\"\"")
		terms = append(terms, fmt.Sprintf("\"%s\"*", escaped))
	}

	if len(terms) == 0 {
		escaped := strings.ReplaceAll(trimmed, "\"", "\"\"")
		return fmt.Sprintf("\"%s\"", escaped)
	}

	return strings.Join(terms, " OR ")
}

func decodeJSONCFile(path string, v interface{}) error {
	// Use the jsonc package from the project
	return jsonCDecode(path, v)
}

var jsonCDecode func(path string, v interface{}) error

func SetJSONCDecoder(fn func(path string, v interface{}) error) {
	jsonCDecode = fn
}

// GetContextForTask returns complete context for a task - the Oracle query
func (b *Butler) GetContextForTask(query string, limit int) (*index.ContextResult, error) {
	return index.GetContextForTask(b.db, query, limit)
}

// GetContextForTaskWithOptions returns context with custom options including token budgeting
func (b *Butler) GetContextForTaskWithOptions(query string, limit int, maxTokens int, includeTests bool) (*index.ContextResult, error) {
	opts := &index.ContextOptions{
		MaxTokens:    maxTokens,
		IncludeTests: includeTests,
	}
	return index.GetContextForTaskWithOptions(b.db, query, limit, opts)
}

// EnhancedContextOptions configures memory-aware context assembly
type EnhancedContextOptions struct {
	Query            string `json:"query"`
	Limit            int    `json:"limit"`
	MaxTokens        int    `json:"maxTokens"`
	IncludeTests     bool   `json:"includeTests"`
	IncludeLearnings bool   `json:"includeLearnings"`
	IncludeFileIntel bool   `json:"includeFileIntel"`
	SessionID        string `json:"sessionId,omitempty"`
}

// EnhancedContextResult includes code context plus memory data
type EnhancedContextResult struct {
	*index.ContextResult

	// Memory-enhanced data
	Learnings []memory.Learning            `json:"learnings,omitempty"`
	FileIntel map[string]*memory.FileIntel `json:"fileIntel,omitempty"`
	Conflict  *memory.Conflict             `json:"conflict,omitempty"`
}

// GetEnhancedContext returns context enriched with learnings and file intel
func (b *Butler) GetEnhancedContext(opts EnhancedContextOptions) (*EnhancedContextResult, error) {
	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	// Get base code context
	codeOpts := &index.ContextOptions{
		MaxTokens:    opts.MaxTokens,
		IncludeTests: opts.IncludeTests,
	}
	codeContext, err := index.GetContextForTaskWithOptions(b.db, opts.Query, opts.Limit, codeOpts)
	if err != nil {
		return nil, fmt.Errorf("get code context: %w", err)
	}

	result := &EnhancedContextResult{
		ContextResult: codeContext,
	}

	// If no memory available, return code context only
	if b.memory == nil {
		return result, nil
	}

	// Get relevant learnings
	if opts.IncludeLearnings {
		learnings, err := b.memory.GetRelevantLearnings("", opts.Query, 10)
		if err == nil && len(learnings) > 0 {
			result.Learnings = learnings
		}
	}

	// Get file intel for files in context
	if opts.IncludeFileIntel && len(codeContext.Files) > 0 {
		result.FileIntel = make(map[string]*memory.FileIntel)
		for _, f := range codeContext.Files {
			intel, err := b.memory.GetFileIntel(f.Path)
			if err == nil && intel != nil && intel.EditCount > 0 {
				result.FileIntel[f.Path] = intel
			}
		}
	}

	return result, nil
}

// GetImpact returns impact analysis for a file or symbol
func (b *Butler) GetImpact(target string) (*index.ImpactResult, error) {
	return index.GetImpact(b.db, target)
}

// ListSymbols lists all symbols of a given kind
func (b *Butler) ListSymbols(kind string, limit int) ([]index.SymbolInfo, error) {
	return index.SearchSymbolsByKind(b.db, kind, limit)
}

// GetSymbol returns a specific symbol by name
func (b *Butler) GetSymbol(name string, filePath string) (*index.SymbolInfo, error) {
	return index.GetSymbol(b.db, name, filePath)
}

// GetFileSymbols returns all symbols in a file
func (b *Butler) GetFileSymbols(filePath string) ([]index.SymbolInfo, error) {
	return index.ListExportedSymbols(b.db, filePath)
}

// GetDependencyGraph returns the import graph for a set of files
func (b *Butler) GetDependencyGraph(rootFiles []string) ([]index.DependencyNode, error) {
	return index.GetDependencyGraph(b.db, rootFiles)
}

// GetIncomingCalls returns all locations that call the given symbol
func (b *Butler) GetIncomingCalls(symbolName string) ([]index.CallSite, error) {
	return index.GetIncomingCalls(b.db, symbolName)
}

// GetOutgoingCalls returns all functions called by the given symbol
func (b *Butler) GetOutgoingCalls(symbolName string, filePath string) ([]index.CallSite, error) {
	return index.GetOutgoingCalls(b.db, symbolName, filePath)
}

// GetCallGraph returns the complete call graph for a file
func (b *Butler) GetCallGraph(filePath string) (*index.CallGraph, error) {
	return index.GetCallGraph(b.db, filePath)
}

// ============================================================================
// Session Memory Methods
// ============================================================================

// HasMemory returns true if session memory is available
func (b *Butler) HasMemory() bool {
	return b.memory != nil
}

// StartSession creates a new session
func (b *Butler) StartSession(agentType, agentID, goal string) (*memory.Session, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.StartSession(agentType, agentID, goal)
}

// EndSession ends a session
func (b *Butler) EndSession(sessionID, state, summary string) error {
	if b.memory == nil {
		return fmt.Errorf("session memory not available")
	}
	return b.memory.EndSession(sessionID, state, summary)
}

// GetSession retrieves a session by ID
func (b *Butler) GetSession(sessionID string) (*memory.Session, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.GetSession(sessionID)
}

// ListSessions lists sessions
func (b *Butler) ListSessions(activeOnly bool, limit int) ([]memory.Session, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.ListSessions(activeOnly, limit)
}

// LogActivity logs an activity
func (b *Butler) LogActivity(sessionID string, act memory.Activity) error {
	if b.memory == nil {
		return fmt.Errorf("session memory not available")
	}
	return b.memory.LogActivity(sessionID, act)
}

// RecordOutcome records session outcome
func (b *Butler) RecordOutcome(sessionID, outcome, summary string) error {
	if b.memory == nil {
		return fmt.Errorf("session memory not available")
	}
	return b.memory.RecordOutcome(sessionID, outcome, summary)
}

// AddLearning adds a new learning
func (b *Butler) AddLearning(l memory.Learning) (string, error) {
	if b.memory == nil {
		return "", fmt.Errorf("session memory not available")
	}
	return b.memory.AddLearning(l)
}

// GetLearnings retrieves learnings
func (b *Butler) GetLearnings(scope, scopePath string, limit int) ([]memory.Learning, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.GetLearnings(scope, scopePath, limit)
}

// SearchLearnings searches learnings by content
func (b *Butler) SearchLearnings(query string, limit int) ([]memory.Learning, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.SearchLearnings(query, limit)
}

// ReinforceLearning increases learning confidence
func (b *Butler) ReinforceLearning(id string) error {
	if b.memory == nil {
		return fmt.Errorf("session memory not available")
	}
	return b.memory.ReinforceLearning(id)
}

// GetFileIntel gets file intelligence
func (b *Butler) GetFileIntel(path string) (*memory.FileIntel, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.GetFileIntel(path)
}

// RecordFileEdit records a file edit
func (b *Butler) RecordFileEdit(path, agentType string) error {
	if b.memory == nil {
		return fmt.Errorf("session memory not available")
	}
	return b.memory.RecordFileEdit(path, agentType)
}

// GetActivities retrieves activities
func (b *Butler) GetActivities(sessionID, filePath string, limit int) ([]memory.Activity, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.GetActivities(sessionID, filePath, limit)
}

// GetRelevantLearnings gets relevant learnings for a context
func (b *Butler) GetRelevantLearnings(filePath, query string, limit int) ([]memory.Learning, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.GetRelevantLearnings(filePath, query, limit)
}

// GetActiveAgents returns currently active agents
func (b *Butler) GetActiveAgents() ([]memory.ActiveAgent, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.GetActiveAgents(5 * time.Minute)
}

// CheckConflict checks if another agent is working on a file
func (b *Butler) CheckConflict(sessionID, path string) (*memory.Conflict, error) {
	if b.memory == nil {
		return nil, nil // No conflict if memory not available
	}
	return b.memory.CheckConflict(sessionID, path)
}

// GetBrief returns a comprehensive briefing
func (b *Butler) GetBrief(filePath string) (*BriefingResult, error) {
	if b.memory == nil {
		return &BriefingResult{}, nil
	}

	result := &BriefingResult{
		FilePath: filePath,
	}

	// Get active agents
	agents, err := b.memory.GetActiveAgents(5 * time.Minute)
	if err == nil {
		result.ActiveAgents = agents
	}

	// Check conflict if file specified
	if filePath != "" {
		conflict, err := b.memory.CheckConflict("", filePath)
		if err == nil && conflict != nil {
			result.Conflict = conflict
		}

		intel, err := b.memory.GetFileIntel(filePath)
		if err == nil {
			result.FileIntel = intel
		}
	}

	// Get relevant learnings
	learnings, err := b.memory.GetRelevantLearnings(filePath, "", 5)
	if err == nil {
		result.Learnings = learnings
	}

	// Get hotspots
	hotspots, err := b.memory.GetFileHotspots(5)
	if err == nil {
		result.Hotspots = hotspots
	}

	return result, nil
}

// BriefingResult contains a comprehensive briefing
type BriefingResult struct {
	FilePath     string               `json:"filePath,omitempty"`
	ActiveAgents []memory.ActiveAgent `json:"activeAgents,omitempty"`
	Conflict     *memory.Conflict     `json:"conflict,omitempty"`
	FileIntel    *memory.FileIntel    `json:"fileIntel,omitempty"`
	Learnings    []memory.Learning    `json:"learnings,omitempty"`
	Hotspots     []memory.FileIntel   `json:"hotspots,omitempty"`
}

// IndexInfo contains information about the code index
type IndexInfo struct {
	FileCount int       `json:"fileCount"`
	LastScan  time.Time `json:"lastScan"`
	Status    string    `json:"status"` // "fresh", "stale", "scanning"
}

// GetIndexInfo returns information about the code index
func (b *Butler) GetIndexInfo() *IndexInfo {
	if b.db == nil {
		return nil
	}

	info := &IndexInfo{
		Status: "fresh",
	}

	// Count files in index
	row := b.db.QueryRow("SELECT COUNT(*) FROM files")
	if err := row.Scan(&info.FileCount); err != nil {
		// If count fails, FileCount remains 0 which is acceptable
		info.FileCount = 0
	}

	// Get last scan time from metadata or file modification
	var lastScan string
	row = b.db.QueryRow("SELECT value FROM metadata WHERE key = 'last_scan'")
	if err := row.Scan(&lastScan); err == nil {
		if t, err := time.Parse(time.RFC3339, lastScan); err == nil {
			info.LastScan = t
			// Check if stale (more than 1 hour old)
			if time.Since(t) > time.Hour {
				info.Status = "stale"
			}
		}
	}

	return info
}
