package patterns

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/analysis"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/memory"
)

// Engine orchestrates pattern detection across a codebase.
type Engine struct {
	registry      *Registry
	memory        *memory.Memory
	parserReg     *analysis.ParserRegistry
	workspaceRoot string
	config        EngineConfig

	// State
	mu       sync.Mutex
	files    []*analysis.FileAnalysis
	fileMap  map[string]*analysis.FileAnalysis
	contents map[string][]byte
}

// EngineConfig contains configuration for the pattern detection engine.
type EngineConfig struct {
	// MaxWorkers is the number of parallel workers for file processing.
	// Defaults to number of CPUs.
	MaxWorkers int

	// Categories to include (empty means all).
	Categories []PatternCategory

	// DetectorIDs to run (empty means all).
	DetectorIDs []string

	// Languages to process (empty means all).
	Languages []string

	// MinConfidence threshold for reporting patterns.
	MinConfidence float64

	// IncludeOutliers controls whether to detect outliers.
	IncludeOutliers bool
}

// DefaultEngineConfig returns the default engine configuration.
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		MaxWorkers:      4,
		MinConfidence:   0.0, // Include all
		IncludeOutliers: true,
	}
}

// NewEngine creates a new pattern detection engine.
func NewEngine(registry *Registry, mem *memory.Memory, workspaceRoot string) *Engine {
	return &Engine{
		registry:      registry,
		memory:        mem,
		parserReg:     analysis.NewParserRegistryWithPath(workspaceRoot),
		workspaceRoot: workspaceRoot,
		config:        DefaultEngineConfig(),
		fileMap:       make(map[string]*analysis.FileAnalysis),
		contents:      make(map[string][]byte),
	}
}

// WithConfig sets the engine configuration.
func (e *Engine) WithConfig(cfg EngineConfig) *Engine {
	e.config = cfg
	return e
}

// ScanResult contains the results of a pattern scan.
type ScanResult struct {
	// Patterns detected during the scan.
	Patterns []Pattern

	// FilesScanned is the number of files processed.
	FilesScanned int

	// DetectorsRun is the number of detectors executed.
	DetectorsRun int

	// Duration is how long the scan took.
	Duration time.Duration

	// Errors encountered during scanning.
	Errors []error
}

// Scan runs pattern detection on the workspace.
func (e *Engine) Scan(ctx context.Context, files []string) (*ScanResult, error) {
	start := time.Now()
	result := &ScanResult{}

	// Parse all files
	if err := e.parseFiles(ctx, files); err != nil {
		return nil, fmt.Errorf("parse files: %w", err)
	}
	result.FilesScanned = len(e.files)

	// Get detectors to run
	detectors := e.getDetectors()
	result.DetectorsRun = len(detectors)

	// Run detection
	patterns, errs := e.runDetectors(ctx, detectors)
	result.Patterns = patterns
	result.Errors = errs
	result.Duration = time.Since(start)

	return result, nil
}

// parseFiles parses all files and builds the context.
func (e *Engine) parseFiles(ctx context.Context, files []string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.files = make([]*analysis.FileAnalysis, 0, len(files))
	e.fileMap = make(map[string]*analysis.FileAnalysis, len(files))
	e.contents = make(map[string][]byte, len(files))

	for _, filePath := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip if language filter doesn't match
		lang := analysis.DetectLanguage(filePath)
		if len(e.config.Languages) > 0 && !e.languageAllowed(string(lang)) {
			continue
		}

		// Read file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue // Skip unreadable files
		}

		// Parse file
		fa, err := e.parserReg.Parse(content, filePath)
		if err != nil {
			continue // Skip unparseable files
		}

		e.files = append(e.files, fa)
		e.fileMap[filePath] = fa
		e.contents[filePath] = content
	}

	return nil
}

// getDetectors returns the detectors to run based on configuration.
func (e *Engine) getDetectors() []Detector {
	all := e.registry.All()

	if len(e.config.DetectorIDs) == 0 && len(e.config.Categories) == 0 {
		return all
	}

	filtered := make([]Detector, 0, len(all))
	for _, d := range all {
		// Check detector ID filter
		if len(e.config.DetectorIDs) > 0 {
			found := false
			for _, id := range e.config.DetectorIDs {
				if d.ID() == id {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check category filter
		if len(e.config.Categories) > 0 {
			found := false
			for _, cat := range e.config.Categories {
				if d.Category() == cat {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		filtered = append(filtered, d)
	}

	return filtered
}

// runDetectors executes all detectors and collects results.
func (e *Engine) runDetectors(ctx context.Context, detectors []Detector) ([]Pattern, []error) {
	var allPatterns []Pattern
	var allErrors []error
	var mu sync.Mutex

	// Create worker pool
	type work struct {
		detector Detector
		file     *analysis.FileAnalysis
	}

	jobs := make(chan work, len(detectors)*len(e.files))
	var wg sync.WaitGroup

	// Start workers
	workerCount := e.config.MaxWorkers
	if workerCount <= 0 {
		workerCount = 4
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Check language support
				if !e.detectorSupportsFile(job.detector, job.file) {
					continue
				}

				// Create detection context
				dctx := &DetectionContext{
					File:           job.file,
					FileContent:    e.contents[job.file.Path],
					AllFiles:       e.files,
					FilesByPath:    e.fileMap,
					WorkspaceRoot:  e.workspaceRoot,
					TotalFileCount: len(e.files),
				}

				// Run detector
				result, err := job.detector.Detect(ctx, dctx)
				if err != nil {
					mu.Lock()
					allErrors = append(allErrors, fmt.Errorf("%s: %w", job.detector.ID(), err))
					mu.Unlock()
					continue
				}

				if result == nil || (len(result.Locations) == 0 && len(result.Outliers) == 0) {
					continue
				}

				// Build pattern
				pattern := e.buildPattern(job.detector, result)

				// Filter by confidence
				if pattern.Confidence < e.config.MinConfidence {
					continue
				}

				mu.Lock()
				allPatterns = append(allPatterns, pattern)
				mu.Unlock()
			}
		}()
	}

	// Queue jobs
	for _, detector := range detectors {
		for _, file := range e.files {
			jobs <- work{detector: detector, file: file}
		}
	}
	close(jobs)

	wg.Wait()

	// Aggregate patterns by detector (combine file-level results)
	aggregated := e.aggregatePatterns(allPatterns)

	return aggregated, allErrors
}

// buildPattern creates a Pattern from detection results.
func (e *Engine) buildPattern(detector Detector, result *DetectionResult) Pattern {
	now := time.Now().UTC()

	// Convert locations
	locations := make([]Location, len(result.Locations))
	copy(locations, result.Locations)

	var outliers []Location
	if e.config.IncludeOutliers {
		outliers = make([]Location, len(result.Outliers))
		for i := range result.Outliers {
			outliers[i] = result.Outliers[i]
			outliers[i].IsOutlier = true
		}
	}

	// Calculate confidence
	confidence := result.Confidence.Score()
	confidence = AdjustConfidenceForOutliers(confidence, len(locations), len(outliers))

	return Pattern{
		Category:         string(detector.Category()),
		Subcategory:      detector.Subcategory(),
		Name:             detector.Name(),
		Description:      detector.Description(),
		DetectorID:       detector.ID(),
		Confidence:       confidence,
		FrequencyScore:   result.Confidence.Frequency,
		ConsistencyScore: result.Confidence.Consistency,
		SpreadScore:      result.Confidence.Spread,
		AgeScore:         result.Confidence.Age,
		Status:           StatusDiscovered,
		Authority:        "proposed",
		Locations:        locations,
		Outliers:         outliers,
		Metadata:         result.Metadata,
		FirstSeen:        now,
		LastSeen:         now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// aggregatePatterns combines patterns from the same detector.
func (e *Engine) aggregatePatterns(patterns []Pattern) []Pattern {
	byDetector := make(map[string]*Pattern)

	for i := range patterns {
		p := &patterns[i]
		existing, ok := byDetector[p.DetectorID]
		if !ok {
			patternCopy := *p
			byDetector[p.DetectorID] = &patternCopy
			continue
		}

		// Merge locations
		existing.Locations = append(existing.Locations, p.Locations...)
		existing.Outliers = append(existing.Outliers, p.Outliers...)

		// Recalculate confidence based on aggregated data
		// This is a simplified approach - real implementation would recalculate factors
		if p.Confidence > existing.Confidence {
			existing.Confidence = p.Confidence
			existing.FrequencyScore = p.FrequencyScore
			existing.ConsistencyScore = p.ConsistencyScore
			existing.SpreadScore = p.SpreadScore
		}
	}

	result := make([]Pattern, 0, len(byDetector))
	for _, p := range byDetector {
		result = append(result, *p)
	}

	return result
}

// detectorSupportsFile checks if a detector supports the file's language.
func (e *Engine) detectorSupportsFile(detector Detector, file *analysis.FileAnalysis) bool {
	langs := detector.Languages()
	if len(langs) == 0 {
		return true // All languages
	}

	for _, lang := range langs {
		if lang == file.Language {
			return true
		}
	}
	return false
}

// languageAllowed checks if a language is in the allowed list.
func (e *Engine) languageAllowed(lang string) bool {
	for _, allowed := range e.config.Languages {
		if allowed == lang {
			return true
		}
	}
	return false
}

// SaveResults persists scan results to the database.
func (e *Engine) SaveResults(_ context.Context, patterns []Pattern) error {
	for i := range patterns {
		p := &patterns[i]
		// Check if pattern already exists
		existing, err := e.findExistingPattern(p.DetectorID)
		if err != nil {
			return err
		}

		if existing != nil { //nolint:nestif // update logic requires nested conditions
			// Update existing pattern
			existing.Confidence = p.Confidence
			existing.FrequencyScore = p.FrequencyScore
			existing.ConsistencyScore = p.ConsistencyScore
			existing.SpreadScore = p.SpreadScore
			existing.AgeScore = p.AgeScore
			existing.LastSeen = time.Now().UTC()
			existing.UpdatedAt = time.Now().UTC()

			if err := e.memory.UpdatePattern(*existing); err != nil {
				return fmt.Errorf("update pattern %s: %w", existing.ID, err)
			}

			// Update locations
			if err := e.memory.DeletePatternLocations(existing.ID); err != nil {
				return err
			}
			if err := e.saveLocations(existing.ID, p.Locations, p.Outliers); err != nil {
				return err
			}
		} else {
			// Create new pattern
			memPattern := memory.Pattern{
				Category:         p.Category,
				Subcategory:      p.Subcategory,
				Name:             p.Name,
				Description:      p.Description,
				DetectorID:       p.DetectorID,
				Confidence:       p.Confidence,
				FrequencyScore:   p.FrequencyScore,
				ConsistencyScore: p.ConsistencyScore,
				SpreadScore:      p.SpreadScore,
				AgeScore:         p.AgeScore,
				Status:           string(p.Status),
				Authority:        p.Authority,
				Metadata:         p.Metadata,
				FirstSeen:        p.FirstSeen,
				LastSeen:         p.LastSeen,
			}

			id, err := e.memory.AddPattern(memPattern)
			if err != nil {
				return fmt.Errorf("add pattern: %w", err)
			}

			if err := e.saveLocations(id, p.Locations, p.Outliers); err != nil {
				return err
			}
		}
	}

	return nil
}

// findExistingPattern looks up a pattern by detector ID.
func (e *Engine) findExistingPattern(detectorID string) (*memory.Pattern, error) {
	patterns, err := e.memory.GetPatterns(memory.PatternFilters{
		DetectorID: detectorID,
		Limit:      1,
	})
	if err != nil {
		return nil, err
	}
	if len(patterns) == 0 {
		return nil, nil
	}
	return &patterns[0], nil
}

// saveLocations persists pattern locations to the database.
func (e *Engine) saveLocations(patternID string, locations, outliers []Location) error {
	for i := range locations {
		memLoc := memory.PatternLocation{
			PatternID: patternID,
			FilePath:  locations[i].FilePath,
			LineStart: locations[i].LineStart,
			LineEnd:   locations[i].LineEnd,
			Snippet:   locations[i].Snippet,
			IsOutlier: false,
		}
		if _, err := e.memory.AddPatternLocation(memLoc); err != nil {
			return err
		}
	}

	for i := range outliers {
		memLoc := memory.PatternLocation{
			PatternID:     patternID,
			FilePath:      outliers[i].FilePath,
			LineStart:     outliers[i].LineStart,
			LineEnd:       outliers[i].LineEnd,
			Snippet:       outliers[i].Snippet,
			IsOutlier:     true,
			OutlierReason: outliers[i].OutlierReason,
		}
		if _, err := e.memory.AddPatternLocation(memLoc); err != nil {
			return err
		}
	}

	return nil
}

// CollectFiles gathers all code files from the workspace.
func CollectFiles(root string, ignorePaths []string) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil //nolint:nilerr // intentionally skip inaccessible files
		}

		// Skip directories
		if info.IsDir() {
			// Skip common ignore directories
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" ||
				name == ".palace" || name == "__pycache__" || name == "dist" ||
				name == "build" || name == ".next" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if path should be ignored
		for _, ignore := range ignorePaths {
			if matched, _ := filepath.Match(ignore, path); matched {
				return nil
			}
		}

		// Only include files with known languages
		lang := analysis.DetectLanguage(path)
		if lang != analysis.LangUnknown {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}
