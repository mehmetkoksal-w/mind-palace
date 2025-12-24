package fsutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/fsutil"
)

func TestMatchesGuardrailEdgeCases(t *testing.T) {
	guardrails := config.Guardrails{
		DoNotTouchGlobs: []string{
			".git/**",
			"**/.git/**",
			"**/.env",
			"**/.hidden/**",
		},
		ReadOnlyGlobs: []string{
			"**/.DS_Store",
		},
	}

	cases := []struct {
		path string
		want bool
	}{
		{path: ".git/config", want: true},
		{path: filepath.Join("nested", ".git", "config"), want: true},
		{path: filepath.Join("config", ".env"), want: true},
		{path: filepath.Join("app", ".hidden", "secret.txt"), want: true},
		{path: filepath.Join("app", ".DS_Store"), want: true},
		{path: filepath.Join("app", "visible.txt"), want: false},
	}

	for _, tc := range cases {
		if got := fsutil.MatchesGuardrail(tc.path, guardrails); got != tc.want {
			t.Fatalf("MatchesGuardrail(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestHashFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")

	content := "Hello, World!"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	hash, err := fsutil.HashFile(path)
	if err != nil {
		t.Fatalf("HashFile failed: %v", err)
	}

	if hash == "" {
		t.Error("hash should not be empty")
	}

	// Same content should produce same hash
	path2 := filepath.Join(tmpDir, "test2.txt")
	if err := os.WriteFile(path2, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	hash2, err := fsutil.HashFile(path2)
	if err != nil {
		t.Fatalf("HashFile failed: %v", err)
	}

	if hash != hash2 {
		t.Errorf("same content should produce same hash: got %s and %s", hash, hash2)
	}

	// Different content should produce different hash
	path3 := filepath.Join(tmpDir, "test3.txt")
	if err := os.WriteFile(path3, []byte("Different content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	hash3, err := fsutil.HashFile(path3)
	if err != nil {
		t.Fatalf("HashFile failed: %v", err)
	}

	if hash == hash3 {
		t.Error("different content should produce different hash")
	}
}

func TestHashFileNotFound(t *testing.T) {
	_, err := fsutil.HashFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestMatchesGuardrailExclude(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		guardrails config.Guardrails
		want       bool
	}{
		{
			name:       "no guardrails",
			path:       "src/main.go",
			guardrails: config.Guardrails{},
			want:       false,
		},
		{
			name: "matches DoNotTouchGlobs pattern",
			path: "node_modules/package/index.js",
			guardrails: config.Guardrails{
				DoNotTouchGlobs: []string{"node_modules/**"},
			},
			want: true,
		},
		{
			name: "matches vendor pattern",
			path: "vendor/pkg/file.go",
			guardrails: config.Guardrails{
				DoNotTouchGlobs: []string{"vendor/**"},
			},
			want: true,
		},
		{
			name: "does not match pattern",
			path: "src/app.go",
			guardrails: config.Guardrails{
				DoNotTouchGlobs: []string{"vendor/**", "node_modules/**"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fsutil.MatchesGuardrail(tt.path, tt.guardrails)
			if got != tt.want {
				t.Errorf("MatchesGuardrail(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestListFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file structure
	dirs := []string{
		"src",
		"src/lib",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, d), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}

	files := []string{
		"src/main.go",
		"src/lib/util.go",
		"README.md",
	}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	// Test without guardrails
	listed, err := fsutil.ListFiles(tmpDir, config.Guardrails{})
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	if len(listed) == 0 {
		t.Error("expected some files to be listed")
	}
}

func TestChunkContent(t *testing.T) {
	// Create content with multiple lines
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, "This is line "+string(rune('0'+i%10)))
	}
	content := strings.Join(lines, "\n")

	// Chunk with max 20 lines
	chunks := fsutil.ChunkContent(content, 20, 10000)

	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}

	// Each chunk should have at most 20 lines
	for _, chunk := range chunks {
		lineCount := chunk.EndLine - chunk.StartLine + 1
		if lineCount > 20 {
			t.Errorf("chunk has %d lines, expected max 20", lineCount)
		}
	}

	// First chunk should start at line 1
	if chunks[0].StartLine != 1 {
		t.Errorf("first chunk should start at line 1, got %d", chunks[0].StartLine)
	}
}

func TestChunkContentSmall(t *testing.T) {
	content := "line1\nline2\nline3"
	chunks := fsutil.ChunkContent(content, 100, 10000)

	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk for small content, got %d", len(chunks))
	}

	if chunks[0].StartLine != 1 || chunks[0].EndLine != 3 {
		t.Errorf("chunk bounds incorrect: got %d-%d, want 1-3", chunks[0].StartLine, chunks[0].EndLine)
	}
}

func TestChunkContentSmart(t *testing.T) {
	content := `func main() {
    fmt.Println("Hello")
}

func helper() {
    fmt.Println("Helper")
}

func another() {
    fmt.Println("Another")
}`

	symbols := []fsutil.SymbolBoundary{
		{Name: "main", Kind: "function", StartLine: 1, EndLine: 3},
		{Name: "helper", Kind: "function", StartLine: 5, EndLine: 7},
		{Name: "another", Kind: "function", StartLine: 9, EndLine: 11},
	}

	chunks := fsutil.ChunkContentSmart(content, symbols, 5, 10000)

	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}

	// Verify chunks have content
	for _, chunk := range chunks {
		if chunk.Content == "" {
			t.Error("chunk content should not be empty")
		}
	}
}

func TestChunkContentSmartNoSymbols(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5"

	// With no symbols, should fall back to line-based chunking
	chunks := fsutil.ChunkContentSmart(content, nil, 2, 10000)

	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}
}

func TestStatFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")

	content := "Test content here"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	stat, err := fsutil.StatFile(path)
	if err != nil {
		t.Fatalf("StatFile failed: %v", err)
	}

	if stat.Size != int64(len(content)) {
		t.Errorf("size mismatch: got %d, want %d", stat.Size, len(content))
	}

	if stat.ModTime.IsZero() {
		t.Error("mod time should not be zero")
	}
}

func TestStatFileNotFound(t *testing.T) {
	_, err := fsutil.StatFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestNormalizeModTime(t *testing.T) {
	now := time.Now()
	normalized := fsutil.NormalizeModTime(now)

	// Nanoseconds should be zero after normalization
	if normalized.Nanosecond() != 0 {
		t.Errorf("expected nanoseconds to be 0, got %d", normalized.Nanosecond())
	}

	// Should preserve the second
	if normalized.Second() != now.Second() {
		t.Errorf("second mismatch: got %d, want %d", normalized.Second(), now.Second())
	}
}

func TestSymbolBoundary(t *testing.T) {
	sb := fsutil.SymbolBoundary{
		Name:      "TestFunc",
		Kind:      "function",
		StartLine: 10,
		EndLine:   20,
	}

	if sb.Name != "TestFunc" {
		t.Error("name not set correctly")
	}
	if sb.Kind != "function" {
		t.Error("kind not set correctly")
	}
	if sb.StartLine != 10 {
		t.Error("startLine not set correctly")
	}
	if sb.EndLine != 20 {
		t.Error("endLine not set correctly")
	}
}

func TestChunkStruct(t *testing.T) {
	chunk := fsutil.Chunk{
		Index:     0,
		StartLine: 1,
		EndLine:   10,
		Content:   "test content",
	}

	if chunk.Index != 0 {
		t.Error("index not set correctly")
	}
	if chunk.StartLine != 1 {
		t.Error("startLine not set correctly")
	}
	if chunk.EndLine != 10 {
		t.Error("endLine not set correctly")
	}
	if chunk.Content != "test content" {
		t.Error("content not set correctly")
	}
}

func TestFileStat(t *testing.T) {
	fs := fsutil.FileStat{
		Size:    1024,
		ModTime: time.Now(),
		Hash:    "abc123",
	}

	if fs.Size != 1024 {
		t.Error("size not set correctly")
	}
	if fs.Hash != "abc123" {
		t.Error("hash not set correctly")
	}
}
