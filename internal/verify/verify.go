package verify

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"mind-palace/internal/config"
	"mind-palace/internal/fsutil"
	"mind-palace/internal/index"
	"mind-palace/internal/signal"
)

// Mode defines staleness verification mode.
type Mode string

const (
	ModeFast   Mode = "fast"
	ModeStrict Mode = "strict"
)

// Options controls verification.
type Options struct {
	Root      string
	DiffRange string
	Mode      Mode
}

// Run performs staleness verification according to options. It returns whether the diff range was ignored.
func Run(db *sql.DB, opts Options) ([]string, bool, error) {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return nil, false, err
	}

	guardrails := config.LoadGuardrails(rootPath)
	stored, err := index.LoadFileMetadata(db)
	if err != nil {
		return nil, false, err
	}

	candidates, fallbackAll, err := diffCandidates(rootPath, guardrails, opts.DiffRange)
	if err != nil {
		return nil, false, err
	}

	if fallbackAll {
		candidates, err = fsutil.ListFiles(rootPath, guardrails)
		if err != nil {
			return nil, false, err
		}
	}

	stale := detectStale(rootPath, candidates, stored, guardrails, opts.Mode, fallbackAll)

	sort.Strings(stale)
	return stale, fallbackAll, nil
}

func detectStale(rootPath string, candidates []string, stored map[string]index.FileMetadata, guardrails config.Guardrails, mode Mode, includeMissing bool) []string {
	var stale []string
	seen := make(map[string]struct{})
	for _, rel := range candidates {
		rel = filepath.ToSlash(rel)
		if fsutil.MatchesGuardrail(rel, guardrails) {
			continue
		}
		seen[rel] = struct{}{}
		abs := filepath.Join(rootPath, rel)
		stat, err := fsutil.StatFile(abs)
		if err != nil {
			if errors.Is(err, fsutil.ErrNotFound) {
				stale = append(stale, fmt.Sprintf("missing file %s", rel))
				continue
			}
			stale = append(stale, fmt.Sprintf("error reading %s: %v", rel, err))
			continue
		}
		storedMeta, ok := stored[rel]
		if !ok {
			stale = append(stale, fmt.Sprintf("new file %s", rel))
			continue
		}

		if mode == ModeStrict {
			h, err := fsutil.HashFile(abs)
			if err != nil {
				stale = append(stale, fmt.Sprintf("hash %s: %v", rel, err))
				continue
			}
			if h != storedMeta.Hash {
				stale = append(stale, fmt.Sprintf("changed file %s", rel))
			}
			continue
		}

		if stat.Size == storedMeta.Size && stat.ModTime.Equal(storedMeta.ModTime) {
			continue
		}

		h, err := fsutil.HashFile(abs)
		if err != nil {
			stale = append(stale, fmt.Sprintf("hash %s: %v", rel, err))
			continue
		}
		if h != storedMeta.Hash {
			stale = append(stale, fmt.Sprintf("changed file %s", rel))
		}
	}

	if includeMissing {
		for rel := range stored {
			if _, ok := seen[rel]; ok {
				continue
			}
			stale = append(stale, fmt.Sprintf("missing file %s", rel))
		}
	}

	return stale
}

func diffCandidates(root string, guardrails config.Guardrails, diffRange string) ([]string, bool, error) {
	if strings.TrimSpace(diffRange) == "" {
		return nil, true, nil
	}
	paths, fromSignal, err := signal.Paths(root, diffRange, guardrails)
	if err != nil && fromSignal {
		return nil, false, err
	}
	if err != nil && !fromSignal {
		return nil, true, nil
	}
	if len(paths) == 0 {
		return nil, true, nil
	}
	return paths, false, nil
}
