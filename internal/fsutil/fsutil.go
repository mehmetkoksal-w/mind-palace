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

	"github.com/koksalmehmet/mind-palace/internal/config"
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
