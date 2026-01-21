package lsp

import (
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/contracts"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/memory"
)

// ButlerAdapter adapts Memory and Contracts to the DiagnosticsProvider interface.
type ButlerAdapter struct {
	memory    *memory.Memory
	contracts *contracts.Store
}

// NewButlerAdapter creates a new adapter for the diagnostics provider.
func NewButlerAdapter(mem *memory.Memory) *ButlerAdapter {
	adapter := &ButlerAdapter{
		memory: mem,
	}

	// Create contracts store if memory is available
	if mem != nil && mem.DB() != nil {
		adapter.contracts = contracts.NewStore(mem.DB())
	}

	return adapter
}

// GetPatternOutliersForFile returns pattern outliers for a file.
func (a *ButlerAdapter) GetPatternOutliersForFile(filePath string) ([]PatternOutlier, error) {
	if a.memory == nil {
		return nil, nil
	}

	// Get outliers from memory
	locations, err := a.memory.GetOutliersForFile(filePath)
	if err != nil {
		return nil, err
	}

	outliers := make([]PatternOutlier, 0, len(locations))
	for _, loc := range locations {
		// Get the pattern for additional info
		pattern, err := a.memory.GetPattern(loc.PatternID)
		if err != nil {
			// Log but continue
			continue
		}

		outlier := PatternOutlier{
			PatternID:     loc.PatternID,
			PatternName:   pattern.Name,
			Description:   pattern.Description,
			FilePath:      loc.FilePath,
			LineStart:     loc.LineStart,
			LineEnd:       loc.LineEnd,
			Snippet:       loc.Snippet,
			OutlierReason: loc.OutlierReason,
			Confidence:    pattern.Confidence,
		}
		outliers = append(outliers, outlier)
	}

	return outliers, nil
}

// GetContractMismatchesForFile returns contract mismatches for a file.
func (a *ButlerAdapter) GetContractMismatchesForFile(filePath string) ([]ContractMismatch, error) {
	if a.contracts == nil {
		return nil, nil
	}

	// Get all contracts with mismatches
	contractList, err := a.contracts.ListContracts(contracts.ContractFilter{
		HasMismatches: true,
		Limit:         1000,
	})
	if err != nil {
		return nil, err
	}

	var mismatches []ContractMismatch
	for _, c := range contractList {
		// Check if this contract involves this file
		isBackendFile := c.Backend.File == filePath
		isFrontendFile := false
		var frontendLine int

		for _, call := range c.FrontendCalls {
			if call.File == filePath {
				isFrontendFile = true
				frontendLine = call.Line
				break
			}
		}

		if !isBackendFile && !isFrontendFile {
			continue
		}

		// Convert mismatches
		for _, m := range c.Mismatches {
			line := c.Backend.Line
			if isFrontendFile && frontendLine > 0 {
				line = frontendLine
			}

			mismatch := ContractMismatch{
				ContractID:   c.ID,
				Method:       c.Method,
				Endpoint:     c.Endpoint,
				FieldPath:    m.FieldPath,
				MismatchType: string(m.Type),
				Severity:     string(m.Severity),
				Description:  m.Description,
				BackendType:  m.BackendType,
				FrontendType: m.FrontendType,
				FilePath:     filePath,
				Line:         line,
			}
			mismatches = append(mismatches, mismatch)
		}
	}

	return mismatches, nil
}
