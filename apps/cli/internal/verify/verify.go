package verify

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/fsutil"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/signal"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/stale"
)

type Mode string

const (
	ModeFast   Mode = "fast"
	ModeStrict Mode = "strict"
)

type Options struct {
	Root      string
	DiffRange string
	Mode      Mode
}

func Run(db *sql.DB, opts Options) ([]string, bool, string, int, error) {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return nil, false, "", 0, err
	}

	guardrails := config.LoadGuardrails(rootPath)
	stored, err := index.LoadFileMetadata(db)
	if err != nil {
		return nil, false, "", 0, err
	}

	candidates, fullScope, source, err := scopeCandidates(rootPath, guardrails, opts.DiffRange)
	if err != nil {
		return nil, false, "", 0, err
	}

	if fullScope {
		candidates, err = fsutil.ListFiles(rootPath, guardrails)
		if err != nil {
			return nil, false, "", 0, err
		}
	}
	scopeCount := len(candidates)

	mode := stale.ModeFast
	if opts.Mode == ModeStrict {
		mode = stale.ModeStrict
	}

	includeMissing := fullScope
	staleList := stale.Detect(rootPath, candidates, stored, guardrails, mode, includeMissing)

	sort.Strings(staleList)

	return staleList, fullScope, source, scopeCount, nil
}

func scopeCandidates(rootPath string, guardrails config.Guardrails, diffRange string) ([]string, bool, string, error) {
	if strings.TrimSpace(diffRange) == "" {
		return nil, true, "full-scan", nil
	}

	paths, fromSignal, err := signal.Paths(rootPath, diffRange, guardrails)
	if err != nil {
		// Diff mode must be strict: do not widen.
		return nil, false, "", fmt.Errorf("diff unavailable for %q: %w", diffRange, err)
	}
	if len(paths) == 0 {
		return []string{}, false, sourceFrom(fromSignal), nil
	}
	return paths, false, sourceFrom(fromSignal), nil
}

func sourceFrom(fromSignal bool) string {
	if fromSignal {
		return "change-signal"
	}
	return "git-diff"
}
