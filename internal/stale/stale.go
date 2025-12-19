package stale

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"

	"mind-palace/internal/config"
	"mind-palace/internal/fsutil"
	"mind-palace/internal/index"
)

// Mode defines staleness verification mode.
type Mode string

const (
	ModeFast   Mode = "fast"
	ModeStrict Mode = "strict"
)

// Detect checks a set of candidate files for staleness vs stored metadata.
// - candidates are relative paths (preferred) or will be normalized.
// - includeMissing controls whether files present in stored metadata but not in candidates are reported as missing.
func Detect(rootPath string, candidates []string, stored map[string]index.FileMetadata, guardrails config.Guardrails, mode Mode, includeMissing bool) []string {
	var stale []string
	seen := make(map[string]struct{})

	for _, rel := range candidates {
		rel = filepath.ToSlash(rel)
		if rel == "" {
			continue
		}
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

		// Fast mode: if size + normalized modtime match, accept; otherwise hash.
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

	sort.Strings(stale)
	return stale
}