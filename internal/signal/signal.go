package signal

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"mind-palace/internal/config"
	"mind-palace/internal/fsutil"
	"mind-palace/internal/model"
	"mind-palace/internal/validate"
)

// Generate produces a change signal for the given git diff range and writes it to outputs.
func Generate(root string, diffRange string) (model.ChangeSignal, error) {
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
	cmd := exec.Command("git", "diff", "--name-status", diffRange)
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
		status := string(parts[0])
		path := filepath.ToSlash(string(parts[1]))
		if fsutil.MatchesGuardrail(path, guardrails) {
			continue
		}
		change := model.Change{Path: path}
		switch status {
		case "A":
			change.Status = "added"
		case "M":
			change.Status = "modified"
		case "D":
			change.Status = "deleted"
		default:
			change.Status = "modified"
		}
		if change.Status != "deleted" {
			abs := filepath.Join(rootPath, path)
			h, err := fsutil.HashFile(abs)
			if err != nil {
				return nil, fmt.Errorf("hash %s: %w", path, err)
			}
			change.Hash = h
		}
		changes = append(changes, change)
	}
	return changes, nil
}
