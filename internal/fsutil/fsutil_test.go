package fsutil_test

import (
	"path/filepath"
	"testing"

	"github.com/mehmetkoksal-w/mind-palace/internal/config"
	"github.com/mehmetkoksal-w/mind-palace/internal/fsutil"
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
