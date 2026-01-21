package contracts

import (
	"fmt"
	"time"
)

// Analyzer detects mismatches between backend endpoints and frontend calls.
type Analyzer struct {
	matcher *Matcher
}

// NewAnalyzer creates a new contract analyzer.
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		matcher: NewMatcher(),
	}
}

// AnalysisResult contains the result of contract analysis.
type AnalysisResult struct {
	Contracts         []*Contract
	UnmatchedBackend  []UnmatchedEndpoint
	UnmatchedFrontend []UnmatchedCall
	TotalMismatches   int
	AnalyzedAt        time.Time
}

// UnmatchedEndpoint represents a backend endpoint with no frontend calls.
type UnmatchedEndpoint struct {
	Method  string
	Path    string
	File    string
	Line    int
	Handler string
}

// UnmatchedCall represents a frontend call with no matching backend endpoint.
type UnmatchedCall struct {
	Method string
	URL    string
	File   string
	Line   int
}

// AnalysisInput contains the input for contract analysis.
type AnalysisInput struct {
	Endpoints []EndpointInput
	Calls     []CallInput
}

// EndpointInput represents a backend endpoint for analysis.
type EndpointInput struct {
	Method         string
	Path           string
	File           string
	Line           int
	Handler        string
	Framework      string
	RequestSchema  *TypeSchema
	ResponseSchema *TypeSchema
}

// CallInput represents a frontend API call for analysis.
type CallInput struct {
	Method         string
	URL            string
	File           string
	Line           int
	IsDynamic      bool
	ExpectedSchema *TypeSchema
}

// Analyze performs contract analysis on the given input.
func (a *Analyzer) Analyze(input *AnalysisInput) *AnalysisResult {
	result := &AnalysisResult{
		Contracts:  make([]*Contract, 0),
		AnalyzedAt: time.Now(),
	}

	// Build matcher with all endpoints
	for _, ep := range input.Endpoints {
		a.matcher.AddEndpoint(ep.Method, ep.Path)
	}

	// Create endpoint lookup map
	endpointMap := make(map[string]*EndpointInput)
	for i := range input.Endpoints {
		ep := &input.Endpoints[i]
		key := ep.Method + ":" + NormalizePath(ep.Path)
		endpointMap[key] = ep
	}

	// Track matched endpoints
	matchedEndpoints := make(map[string]bool)

	// Process each frontend call
	contractMap := make(map[string]*Contract)

	for _, call := range input.Calls {
		match := a.matcher.Match(call.Method, call.URL)
		if match == nil {
			// No matching endpoint
			result.UnmatchedFrontend = append(result.UnmatchedFrontend, UnmatchedCall{
				Method: call.Method,
				URL:    call.URL,
				File:   call.File,
				Line:   call.Line,
			})
			continue
		}

		key := match.Method + ":" + match.BackendEndpoint
		matchedEndpoints[key] = true

		// Get or create contract
		contract, exists := contractMap[key]
		if !exists {
			ep := endpointMap[key]
			if ep == nil {
				continue
			}

			contract = &Contract{
				ID:              GenerateID("ct"),
				Method:          ep.Method,
				Endpoint:        ep.Path,
				EndpointPattern: PathToPattern(ep.Path),
				Backend: BackendEndpoint{
					File:           ep.File,
					Line:           ep.Line,
					Framework:      ep.Framework,
					Handler:        ep.Handler,
					RequestSchema:  ep.RequestSchema,
					ResponseSchema: ep.ResponseSchema,
				},
				FrontendCalls: make([]FrontendCall, 0),
				Mismatches:    make([]FieldMismatch, 0),
				Status:        ContractDiscovered,
				FirstSeen:     time.Now(),
				LastSeen:      time.Now(),
			}
			contractMap[key] = contract
		}

		// Add frontend call
		frontendCall := FrontendCall{
			ID:       GenerateID("ct"),
			File:     call.File,
			Line:     call.Line,
			CallType: "fetch", // Default, could be detected from extractor
		}

		// Detect type mismatches
		if call.ExpectedSchema != nil && contract.Backend.ResponseSchema != nil {
			mismatches := contract.Backend.ResponseSchema.Compare(call.ExpectedSchema, "")
			contract.Mismatches = append(contract.Mismatches, mismatches...)
		}

		contract.FrontendCalls = append(contract.FrontendCalls, frontendCall)
		contract.LastSeen = time.Now()
	}

	// Find unmatched backend endpoints
	for _, ep := range input.Endpoints {
		key := ep.Method + ":" + NormalizePath(ep.Path)
		if !matchedEndpoints[key] {
			result.UnmatchedBackend = append(result.UnmatchedBackend, UnmatchedEndpoint{
				Method:  ep.Method,
				Path:    ep.Path,
				File:    ep.File,
				Line:    ep.Line,
				Handler: ep.Handler,
			})
		}
	}

	// Finalize contracts
	for _, contract := range contractMap {
		// Update status based on mismatches
		if len(contract.Mismatches) > 0 {
			contract.Status = ContractMismatch
			result.TotalMismatches += len(contract.Mismatches)
		}

		// Calculate confidence
		contract.Confidence = a.calculateContractConfidence(contract)

		result.Contracts = append(result.Contracts, contract)
	}

	return result
}

func (a *Analyzer) calculateContractConfidence(contract *Contract) float64 {
	confidence := 0.5 // Base confidence

	// More frontend calls = higher confidence
	callCount := len(contract.FrontendCalls)
	switch {
	case callCount >= 5:
		confidence += 0.3
	case callCount >= 2:
		confidence += 0.2
	case callCount >= 1:
		confidence += 0.1
	}

	// Schema presence increases confidence
	if contract.Backend.ResponseSchema != nil {
		confidence += 0.1
	}
	if contract.Backend.RequestSchema != nil {
		confidence += 0.1
	}

	// Mismatches decrease confidence (or indicate it's well-analyzed)
	// This is debatable - mismatches could mean the contract is well-understood

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// DetectMismatches compares backend and frontend schemas and returns mismatches.
func DetectMismatches(backendSchema, frontendSchema *TypeSchema, fieldPath string) []FieldMismatch {
	if backendSchema == nil || frontendSchema == nil {
		return nil
	}
	return backendSchema.Compare(frontendSchema, fieldPath)
}

// SummarizeMismatches generates human-readable summaries for mismatches.
func SummarizeMismatches(mismatches []FieldMismatch) []string {
	summaries := make([]string, 0, len(mismatches))

	for _, m := range mismatches {
		var summary string
		switch m.Type {
		case MismatchMissingInFrontend:
			summary = fmt.Sprintf("Field '%s' exists in backend (%s) but not expected by frontend",
				m.FieldPath, m.BackendType)
		case MismatchMissingInBackend:
			summary = fmt.Sprintf("Frontend expects field '%s' (%s) but backend doesn't provide it",
				m.FieldPath, m.FrontendType)
		case MismatchTypeMismatch:
			summary = fmt.Sprintf("Type mismatch at '%s': backend sends %s, frontend expects %s",
				m.FieldPath, m.BackendType, m.FrontendType)
		case MismatchOptionalityMismatch:
			summary = fmt.Sprintf("Optionality mismatch at '%s': %s", m.FieldPath, m.Description)
		case MismatchNullabilityMismatch:
			summary = fmt.Sprintf("Nullability mismatch at '%s': %s", m.FieldPath, m.Description)
		default:
			summary = m.Description
		}
		summaries = append(summaries, summary)
	}

	return summaries
}

// GetMismatchSeverity returns the severity level for a mismatch type.
func GetMismatchSeverity(mType MismatchType) string {
	switch mType {
	case MismatchTypeMismatch:
		return "error"
	case MismatchMissingInBackend:
		return "error"
	case MismatchMissingInFrontend:
		return "warning"
	case MismatchOptionalityMismatch:
		return "warning"
	case MismatchNullabilityMismatch:
		return "warning"
	default:
		return "info"
	}
}
