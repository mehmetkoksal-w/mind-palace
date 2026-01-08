package signal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/fsutil"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/validate"
)

func Generate(root, diffRange string) (model.ChangeSignal, error) {
	if strings.TrimSpace(diffRange) == "" {
		return model.ChangeSignal{}, errors.New("signal requires --diff range")
	}
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return model.ChangeSignal{}, err
	}
	if _, err := config.EnsureLayout(rootPath); err != nil {
		return model.ChangeSignal{}, err
	}

	guardrails := config.LoadGuardrails(rootPath)
	changes, err := gitChanges(rootPath, diffRange, guardrails)
	if err != nil {
		return model.ChangeSignal{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	sig := model.ChangeSignal{
		SchemaVersion: "1.0.0",
		Kind:          "palace/change-signal",
		DiffRange:     diffRange,
		GeneratedAt:   now,
		Changes:       changes,
		Provenance: model.Provenance{
			CreatedBy: "palace signal",
			CreatedAt: now,
		},
	}

	sort.Slice(sig.Changes, func(i, j int) bool { return sig.Changes[i].Path < sig.Changes[j].Path })

	outPath := filepath.Join(rootPath, ".palace", "outputs", "change-signal.json")
	if err := model.WriteChangeSignal(outPath, sig); err != nil {
		return model.ChangeSignal{}, err
	}
	if err := validate.JSON(outPath, "change-signal"); err != nil {
		return model.ChangeSignal{}, err
	}
	return sig, nil
}

func gitChanges(rootPath, diffRange string, guardrails config.Guardrails) ([]model.Change, error) {
	cmd := exec.CommandContext(context.Background(), "git", "diff", "--name-status", diffRange)
	cmd.Dir = rootPath
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	var changes []model.Change
	lines := bytes.Split(bytes.TrimSpace(out), []byte("\n"))
	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		parts := bytes.Fields(line)
		if len(parts) < 2 {
			continue
		}
		statusToken := string(parts[0])
		status := parseStatus(statusToken)
		pathIdx := 1
		if strings.HasPrefix(statusToken, "R") || strings.HasPrefix(statusToken, "C") {
			if len(parts) >= 3 {
				pathIdx = 2
			}
		}
		if pathIdx >= len(parts) {
			continue
		}
		path := filepath.ToSlash(string(parts[pathIdx]))
		if fsutil.MatchesGuardrail(path, guardrails) {
			continue
		}
		change := model.Change{Path: path, Status: status}
		if change.Status != "deleted" {
			abs := filepath.Join(rootPath, path)
			h, err := fsutil.HashFile(abs)
			if err != nil {
				if errors.Is(err, fsutil.ErrNotFound) || os.IsNotExist(err) {
					return nil, fmt.Errorf("hash %s: file not found for diff range %s", path, diffRange)
				}
				return nil, fmt.Errorf("hash %s: %w", path, err)
			}
			change.Hash = h
		}
		changes = append(changes, change)
	}
	return changes, nil
}

func parseStatus(token string) string {
	switch token {
	case "A":
		return "added"
	case "M":
		return "modified"
	case "D":
		return "deleted"
	}
	if strings.HasPrefix(token, "R") || strings.HasPrefix(token, "C") {
		return "modified"
	}
	return "modified"
}
