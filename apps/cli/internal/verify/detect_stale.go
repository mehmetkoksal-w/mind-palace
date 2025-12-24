package verify

import (
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/stale"
)

func detectStale(
	dir string,
	changedFiles []string,
	stored map[string]index.FileMetadata,
	guardrails config.Guardrails,
	mode Mode,
	normalizeTimestamps bool,
) []string {
	_ = normalizeTimestamps // kept for API compatibility / docs

	sMode := stale.ModeFast
	if mode == ModeStrict {
		sMode = stale.ModeStrict
	}

	// In diff scope we only check candidates, so includeMissing=false.
	// Callers who want full-scope missing-file detection should pass the full file list,
	// or use verify.Run which sets includeMissing based on scope.
	includeMissing := false

	return stale.Detect(dir, changedFiles, stored, guardrails, sMode, includeMissing)
}
