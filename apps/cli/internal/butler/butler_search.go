package butler

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// Search performs a full-text search across the codebase.
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

// calculateScore computes the final relevance score for a search result.
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

// inferRoom determines which room a file belongs to based on its path.
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

// groupByRoom organizes search results by their room.
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
