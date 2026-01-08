package signal

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/fsutil"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/validate"
)

// Paths returns changed paths for a diff range, preferring a matching change signal when available.
// The boolean indicates whether the returned paths came from a change signal.
func Paths(rootPath, diffRange string, guardrails config.Guardrails) ([]string, bool, error) {
	if strings.TrimSpace(diffRange) == "" {
		return nil, false, nil
	}
	if paths, ok, err := pathsFromSignal(rootPath, diffRange, guardrails); err != nil || ok {
		return paths, ok, err
	}
	return gitDiffPaths(rootPath, diffRange, guardrails)
}

func pathsFromSignal(rootPath, diffRange string, guardrails config.Guardrails) ([]string, bool, error) {
	sigPath := filepath.Join(rootPath, ".palace", "outputs", "change-signal.json")
	if _, err := os.Stat(sigPath); err != nil {
		return nil, false, nil
	}
	if err := validate.JSON(sigPath, "change-signal"); err != nil {
		return nil, true, err
	}
	sig, err := model.LoadChangeSignal(sigPath)
	if err != nil {
		return nil, true, err
	}
	if sig.DiffRange != diffRange {
		return nil, false, nil
	}
	var paths []string
	for _, c := range sig.Changes {
		p := filepath.ToSlash(c.Path)
		if fsutil.MatchesGuardrail(p, guardrails) {
			continue
		}
		paths = append(paths, p)
	}
	return paths, true, nil
}

func gitDiffPaths(rootPath, diffRange string, guardrails config.Guardrails) ([]string, bool, error) {
	cmd := exec.CommandContext(context.Background(), "git", "diff", "--name-only", diffRange)
	cmd.Dir = rootPath
	out, err := cmd.Output()
	if err != nil {
		return nil, false, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var paths []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		rel := filepath.ToSlash(l)
		if fsutil.MatchesGuardrail(rel, guardrails) {
			continue
		}
		paths = append(paths, rel)
	}
	return paths, false, nil
}
