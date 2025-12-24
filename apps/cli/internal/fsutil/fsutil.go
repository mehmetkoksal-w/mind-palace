package fsutil

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
)

type Chunk struct {
	Index     int
	StartLine int
	EndLine   int
	Content   string
}

// MatchesGuardrail returns true if the path matches any guardrail glob.
func MatchesGuardrail(path string, guardrails config.Guardrails) bool {
	normalized := filepath.ToSlash(path)
	for _, g := range guardrails.DoNotTouchGlobs {
		if g == "" {
			continue
		}
		ok, err := doublestar.Match(g, normalized)
		if err == nil && ok {
			return true
		}
	}
	for _, g := range guardrails.ReadOnlyGlobs {
		if g == "" {
			continue
		}
		ok, err := doublestar.Match(g, normalized)
		if err == nil && ok {
			return true
		}
	}
	return false
}

func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func ListFiles(root string, guardrails config.Guardrails) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Skip permission errors and other access issues gracefully
			if os.IsPermission(err) {
				return filepath.SkipDir
			}
			return err
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if MatchesGuardrail(rel, guardrails) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if entry is a symlink
		if d.Type()&os.ModeSymlink != 0 {
			// Resolve symlink to check what it points to
			target, err := os.Stat(path)
			if err != nil {
				// Skip broken symlinks or inaccessible targets
				return nil
			}
			if target.IsDir() {
				// Skip symlinked directories to avoid following them
				return filepath.SkipDir
			}
			// Symlink to file - include it
			files = append(files, rel)
			return nil
		}

		if d.IsDir() {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func ChunkContent(content string, maxLines int, maxBytes int) []Chunk {
	if maxLines <= 0 {
		maxLines = 120
	}
	if maxBytes <= 0 {
		maxBytes = 8 * 1024
	}
	lines := strings.Split(content, "\n")
	var chunks []Chunk
	var buffer []string
	currentBytes := 0
	startLine := 1

	flush := func(endLine int) {
		if len(buffer) == 0 {
			return
		}
		chunkContent := strings.Join(buffer, "\n")
		chunks = append(chunks, Chunk{
			Index:     len(chunks),
			StartLine: startLine,
			EndLine:   endLine,
			Content:   chunkContent,
		})
		buffer = buffer[:0]
		currentBytes = 0
		startLine = endLine + 1
	}

	for i, line := range lines {
		lineBytes := len(line)
		// Add 1 for the newline except for the final line.
		if i < len(lines)-1 {
			lineBytes++
		}
		if len(buffer) >= maxLines || currentBytes+lineBytes > maxBytes {
			flush(startLine + len(buffer) - 1)
		}
		buffer = append(buffer, line)
		currentBytes += lineBytes
	}
	flush(startLine + len(buffer) - 1)
	return chunks
}

// SymbolBoundary represents a symbol's line range for AST-aware chunking
type SymbolBoundary struct {
	Name      string
	Kind      string
	StartLine int
	EndLine   int
}

// ChunkContentSmart creates chunks that respect symbol boundaries (AST-aware).
// It never splits in the middle of a function, class, or method.
// Falls back to line-based chunking if no symbols are provided.
func ChunkContentSmart(content string, symbols []SymbolBoundary, maxLines int, maxBytes int) []Chunk {
	if len(symbols) == 0 {
		// Fall back to line-based chunking
		return ChunkContent(content, maxLines, maxBytes)
	}

	if maxLines <= 0 {
		maxLines = 120
	}
	if maxBytes <= 0 {
		maxBytes = 8 * 1024
	}

	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// Sort symbols by start line
	sortedSymbols := make([]SymbolBoundary, len(symbols))
	copy(sortedSymbols, symbols)
	sortSymbolsByLine(sortedSymbols)

	// Find natural break points (lines between symbols)
	breakPoints := findBreakPoints(sortedSymbols, totalLines)

	var chunks []Chunk
	currentStart := 1

	for _, bp := range breakPoints {
		// Calculate chunk from currentStart to bp
		chunkLines := bp - currentStart + 1
		chunkContent := extractLines(lines, currentStart, bp)
		chunkBytes := len(chunkContent)

		// If this chunk is too large, we need to split it
		if chunkLines > maxLines || chunkBytes > maxBytes {
			// Split at symbol boundaries within this range
			subChunks := splitLargeChunk(lines, currentStart, bp, sortedSymbols, maxLines, maxBytes)
			for _, sc := range subChunks {
				sc.Index = len(chunks)
				chunks = append(chunks, sc)
			}
		} else if chunkLines > 0 {
			chunks = append(chunks, Chunk{
				Index:     len(chunks),
				StartLine: currentStart,
				EndLine:   bp,
				Content:   chunkContent,
			})
		}

		currentStart = bp + 1
	}

	// Handle remaining content
	if currentStart <= totalLines {
		chunkContent := extractLines(lines, currentStart, totalLines)
		if len(chunkContent) > 0 {
			chunks = append(chunks, Chunk{
				Index:     len(chunks),
				StartLine: currentStart,
				EndLine:   totalLines,
				Content:   chunkContent,
			})
		}
	}

	return chunks
}

// sortSymbolsByLine sorts symbols by their start line (simple insertion sort)
func sortSymbolsByLine(symbols []SymbolBoundary) {
	for i := 1; i < len(symbols); i++ {
		j := i
		for j > 0 && symbols[j-1].StartLine > symbols[j].StartLine {
			symbols[j-1], symbols[j] = symbols[j], symbols[j-1]
			j--
		}
	}
}

// findBreakPoints identifies safe lines to split between symbols
func findBreakPoints(symbols []SymbolBoundary, totalLines int) []int {
	if len(symbols) == 0 {
		return []int{totalLines}
	}

	var breakPoints []int
	for i := 0; i < len(symbols)-1; i++ {
		// Break point is the line just before the next symbol starts
		// (i.e., after current symbol ends)
		endOfCurrent := symbols[i].EndLine
		startOfNext := symbols[i+1].StartLine

		if startOfNext > endOfCurrent+1 {
			// There's a gap between symbols - break in the gap
			breakPoints = append(breakPoints, startOfNext-1)
		} else {
			// Symbols are adjacent or overlapping - break at end of current
			breakPoints = append(breakPoints, endOfCurrent)
		}
	}

	// Add final break point at end of last symbol or file
	lastSymbol := symbols[len(symbols)-1]
	if lastSymbol.EndLine < totalLines {
		breakPoints = append(breakPoints, totalLines)
	} else {
		breakPoints = append(breakPoints, lastSymbol.EndLine)
	}

	return breakPoints
}

// extractLines extracts lines from start to end (1-indexed, inclusive)
func extractLines(lines []string, start, end int) string {
	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start > end || start > len(lines) {
		return ""
	}
	return strings.Join(lines[start-1:end], "\n")
}

// splitLargeChunk splits a chunk that's too large, respecting symbol boundaries
func splitLargeChunk(lines []string, start, end int, symbols []SymbolBoundary, maxLines, maxBytes int) []Chunk {
	var chunks []Chunk

	// Find symbols within this range
	var relevantSymbols []SymbolBoundary
	for _, sym := range symbols {
		if sym.StartLine >= start && sym.EndLine <= end {
			relevantSymbols = append(relevantSymbols, sym)
		}
	}

	if len(relevantSymbols) == 0 {
		// No symbols in range - fall back to line-based chunking for this section
		content := extractLines(lines, start, end)
		lineChunks := ChunkContent(content, maxLines, maxBytes)
		for _, lc := range lineChunks {
			lc.StartLine += start - 1
			lc.EndLine += start - 1
			chunks = append(chunks, lc)
		}
		return chunks
	}

	// Group consecutive symbols that fit within limits
	currentStart := start
	var currentSymbols []SymbolBoundary
	currentBytes := 0

	flushGroup := func() {
		if len(currentSymbols) == 0 {
			return
		}
		groupEnd := currentSymbols[len(currentSymbols)-1].EndLine
		content := extractLines(lines, currentStart, groupEnd)
		chunks = append(chunks, Chunk{
			StartLine: currentStart,
			EndLine:   groupEnd,
			Content:   content,
		})
		currentStart = groupEnd + 1
		currentSymbols = nil
		currentBytes = 0
	}

	for _, sym := range relevantSymbols {
		symContent := extractLines(lines, sym.StartLine, sym.EndLine)
		symBytes := len(symContent)
		symLines := sym.EndLine - sym.StartLine + 1

		// Check if adding this symbol would exceed limits
		groupLines := 0
		if len(currentSymbols) > 0 {
			groupLines = sym.EndLine - currentStart + 1
		} else {
			groupLines = symLines
		}

		if len(currentSymbols) > 0 && (groupLines > maxLines || currentBytes+symBytes > maxBytes) {
			flushGroup()
		}

		// Handle symbols that are individually too large
		if symLines > maxLines || symBytes > maxBytes {
			if len(currentSymbols) > 0 {
				flushGroup()
			}
			// Add the large symbol as its own chunk (don't split functions)
			chunks = append(chunks, Chunk{
				StartLine: sym.StartLine,
				EndLine:   sym.EndLine,
				Content:   symContent,
			})
			currentStart = sym.EndLine + 1
			continue
		}

		currentSymbols = append(currentSymbols, sym)
		currentBytes += symBytes
	}

	flushGroup()

	return chunks
}

var ErrNotFound = os.ErrNotExist

type FileStat struct {
	Size    int64
	ModTime time.Time
	Hash    string
}

// StatFile returns size and mod time for a path.
func StatFile(path string) (FileStat, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return FileStat{}, ErrNotFound
		}
		return FileStat{}, err
	}
	return FileStat{
		Size:    info.Size(),
		ModTime: NormalizeModTime(info.ModTime()),
	}, nil
}

// NormalizeModTime truncates mod time to second precision for deterministic comparisons.
func NormalizeModTime(t time.Time) time.Time {
	return t.UTC().Truncate(time.Second)
}
