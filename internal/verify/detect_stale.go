package verify

import (
	"mind-palace/internal/config"
	"mind-palace/internal/index"
	"mind-palace/internal/stale"
)

// detectStale determines which files in a directory are stale compared to a stored manifest.
//
// Takes: dir, changedFiles, stored metadata, guardrails, mode, normalizeTimestamps
// Returns: list of stale file paths to be pruned or revalidated.
//
// Note: current implementation delegates to internal/stale.Detect.
// normalizeTimestamps is kept for compatibility (stale.Detect handles timestamp normalization internally).
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
