package util

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
)

func TestMustAbs(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"current dir", "."},
		{"relative path", "./foo/bar"},
		{"absolute path", "/tmp/test"},
		{"empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// MustAbs should not panic and should return a string
			result := MustAbs(tt.input)
			if result == "" && tt.input != "" {
				t.Errorf("MustAbs(%q) returned empty string", tt.input)
			}
		})
	}
}

func TestScopeFileCount(t *testing.T) {
	tests := []struct {
		name string
		cp   model.ContextPack
		want int
	}{
		{
			name: "nil scope returns 0",
			cp:   model.ContextPack{Scope: nil},
			want: 0,
		},
		{
			name: "scope with count",
			cp: model.ContextPack{
				Scope: &model.ScopeInfo{FileCount: 42},
			},
			want: 42,
		},
		{
			name: "scope with zero count",
			cp: model.ContextPack{
				Scope: &model.ScopeInfo{FileCount: 0},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScopeFileCount(tt.cp)
			if got != tt.want {
				t.Errorf("ScopeFileCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTruncateLine(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a longer line", 10, "this is a ..."},
		{"", 5, ""},
	}

	for _, tt := range tests {
		got := TruncateLine(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("TruncateLine(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestPrintScope(t *testing.T) {
	tests := []struct {
		name      string
		cmd       string
		fullScope bool
		source    string
		diffRange string
		fileCount int
		rootPath  string
		wantParts []string
		dontWant  []string
	}{
		{
			name:      "diff mode with source",
			cmd:       "verify",
			fullScope: false,
			source:    "git-diff",
			diffRange: "HEAD~1..HEAD",
			fileCount: 10,
			rootPath:  "/project",
			wantParts: []string{
				"Scope (verify):",
				"root: /project",
				"mode: diff",
				"source: git-diff",
				"fileCount: 10",
				"diffRange: HEAD~1..HEAD",
			},
		},
		{
			name:      "full mode with source",
			cmd:       "check",
			fullScope: true,
			source:    "manual",
			diffRange: "",
			fileCount: 100,
			rootPath:  "/home/user/project",
			wantParts: []string{
				"Scope (check):",
				"root: /home/user/project",
				"mode: full",
				"source: manual",
				"fileCount: 100",
			},
			dontWant: []string{"diffRange"},
		},
		{
			name:      "diff mode without source uses default",
			cmd:       "ci",
			fullScope: false,
			source:    "",
			diffRange: "main..feature",
			fileCount: 5,
			rootPath:  "/app",
			wantParts: []string{
				"source: git-diff/change-signal",
				"diffRange: main..feature",
			},
		},
		{
			name:      "full mode without source uses default",
			cmd:       "scan",
			fullScope: true,
			source:    "",
			diffRange: "",
			fileCount: 50,
			rootPath:  "/code",
			wantParts: []string{
				"source: full-scan",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			PrintScope(tt.cmd, tt.fullScope, tt.source, tt.diffRange, tt.fileCount, tt.rootPath)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			for _, part := range tt.wantParts {
				if !strings.Contains(output, part) {
					t.Errorf("PrintScope() output missing %q\nGot: %s", part, output)
				}
			}

			for _, part := range tt.dontWant {
				if strings.Contains(output, part) {
					t.Errorf("PrintScope() output should not contain %q\nGot: %s", part, output)
				}
			}
		})
	}
}
