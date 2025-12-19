package collect

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"mind-palace/internal/config"
	"mind-palace/internal/fsutil"
	"mind-palace/internal/index"
	"mind-palace/internal/jsonc"
	"mind-palace/internal/model"
	"mind-palace/internal/signal"
	"mind-palace/internal/stale"
	"mind-palace/internal/validate"
)

// Options controls collect behavior.
type Options struct {
	AllowStale bool
}

// Run assembles a context pack using the latest index and curated manifests.
func Run(root string, diffRange string, opts Options) (model.ContextPack, error) {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return model.ContextPack{}, err
	}
	if _, err := config.EnsureLayout(rootPath); err != nil {
		return model.ContextPack{}, err
	}

	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	db, err := index.Open(dbPath)
	if err != nil {
		return model.ContextPack{}, fmt.Errorf("open index: %w", err)
	}
	defer db.Close()

	summary, err := index.LatestScan(db)
	if err != nil {
		return model.ContextPack{}, err
	}
	if summary.ID == 0 {
		return model.ContextPack{}, errors.New("no scan records found; run palace scan")
	}

	palaceCfg, err := config.LoadPalaceConfig(rootPath)
	if err != nil {
		return model.ContextPack{}, fmt.Errorf("load palace config: %w", err)
	}

	cpPath := filepath.Join(rootPath, ".palace", "outputs", "context-pack.json")
	cp, err := model.LoadContextPack(cpPath)
	if err != nil {
		cp = model.NewContextPack("unspecified")
		cp.Provenance.CreatedBy = "palace collect"
	}

	cp.ScanHash = summary.ScanHash
	cp.ScanTime = summary.CompletedAt.Format(time.RFC3339)
	cp.ScanID = fmt.Sprintf("scan-%d", summary.ID)
	cp.Provenance.UpdatedBy = "palace collect"
	cp.Provenance.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	guardrails := config.LoadGuardrails(rootPath)
	storedMeta, err := index.LoadFileMetadata(db)
	if err != nil {
		return model.ContextPack{}, err
	}

	// Determine scope candidates.
	fullScope := strings.TrimSpace(diffRange) == ""
	scopeSource := "full-scan"
	var candidates []string

	if fullScope {
		candidates, err = fsutil.ListFiles(rootPath, guardrails)
		if err != nil {
			return model.ContextPack{}, err
		}

		// Freshness enforcement in full scope unless allow-stale.
		if !opts.AllowStale {
			staleList := stale.Detect(rootPath, candidates, storedMeta, guardrails, stale.ModeFast, true)
			if len(staleList) > 0 {
				msg := "index is stale; run palace scan"
				// Provide a bounded preview.
				preview := staleList
				if len(preview) > 20 {
					preview = preview[:20]
				}
				return model.ContextPack{}, fmt.Errorf("%s\nstale artifacts detected (showing %d/%d):\n- %s",
					msg, len(preview), len(staleList), strings.Join(preview, "\n- "))
			}
		}
	} else {
		paths, fromSignal, err := signal.Paths(rootPath, diffRange, guardrails)
		if err != nil {
			return model.ContextPack{}, fmt.Errorf("diff unavailable for %q: %w", diffRange, err)
		}
		candidates = paths
		if fromSignal {
			scopeSource = "change-signal"
		} else {
			scopeSource = "git-diff"
		}
	}

	cp.Scope = &model.ScopeInfo{
		Mode:      map[bool]string{true: "full", false: "diff"}[fullScope],
		Source:    scopeSource,
		FileCount: len(candidates),
		DiffRange: strings.TrimSpace(diffRange),
	}
	if fullScope {
		cp.Scope.DiffRange = ""
	}

	// changedPaths are used for FilesReferenced ordering priority.
	changedPaths := []string{}
	if !fullScope {
		changedPaths = candidates
	}

	roomEntries := collectEntryPoints(rootPath, palaceCfg.DefaultRoom)
	cp.RoomsVisited = nil
	if palaceCfg.DefaultRoom != "" {
		cp.RoomsVisited = []string{palaceCfg.DefaultRoom}
	}

	// In full scope, keep FilesReferenced deterministic: entry points first, then (optionally) diff candidates.
	// In diff scope, prioritize changed paths, then room entrypoints that exist in the index.
	cp.FilesReferenced = mergeOrderedUnique(changedPaths, filterExisting(roomEntries, storedMeta))

	if strings.TrimSpace(cp.Goal) == "" {
		cp.Goal = "unspecified"
	}

	cp.Findings = nil
	if goal := cp.Goal; goal != "" {
		hits, err := index.SearchChunks(db, goal, 20)
		if err == nil {
			ordered := prioritizeHits(hits, changedPaths)
			for _, h := range ordered {
				cp.Findings = append(cp.Findings, model.Finding{
					Summary:  fmt.Sprintf("content match for goal in %s", h.Path),
					Detail:   fmt.Sprintf("lines %d-%d", h.StartLine, h.EndLine),
					Severity: "info",
					File:     h.Path,
				})
				if len(cp.Findings) >= 5 {
					break
				}
			}
		}
	}

	if err := model.WriteContextPack(cpPath, cp); err != nil {
		return model.ContextPack{}, err
	}
	if err := validate.JSON(cpPath, "context-pack"); err != nil {
		return model.ContextPack{}, err
	}
	return cp, nil
}

func collectEntryPoints(rootPath, roomName string) []string {
	if roomName == "" {
		return nil
	}
	roomPath := filepath.Join(rootPath, ".palace", "rooms", fmt.Sprintf("%s.jsonc", roomName))
	var room model.Room
	if err := validate.JSONC(roomPath, "room"); err != nil {
		return nil
	}
	if err := jsonc.DecodeFile(roomPath, &room); err != nil {
		return nil
	}
	var entries []string
	for _, ep := range room.EntryPoints {
		entries = append(entries, filepath.ToSlash(ep))
	}
	return entries
}

func filterExisting(paths []string, stored map[string]index.FileMetadata) []string {
	var out []string
	for _, p := range paths {
		if _, ok := stored[p]; ok {
			out = append(out, p)
		}
	}
	return out
}

func mergeOrderedUnique(primary, secondary []string) []string {
	seen := make(map[string]struct{})
	var out []string
	appendList := func(list []string) {
		for _, v := range list {
			if v == "" {
				continue
			}
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	appendList(primary)
	appendList(secondary)
	return out
}

func prioritizeHits(hits []index.ChunkHit, changedPaths []string) []index.ChunkHit {
	if len(changedPaths) == 0 {
		return hits
	}
	set := make(map[string]struct{}, len(changedPaths))
	for _, p := range changedPaths {
		set[p] = struct{}{}
	}
	var prioritized, remainder []index.ChunkHit
	for _, h := range hits {
		if _, ok := set[h.Path]; ok {
			prioritized = append(prioritized, h)
		} else {
			remainder = append(remainder, h)
		}
	}
	return append(prioritized, remainder...)
}