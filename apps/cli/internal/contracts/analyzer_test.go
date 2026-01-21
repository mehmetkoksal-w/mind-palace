package contracts

import (
	"testing"
)

func TestAnalyzer_BasicAnalysis(t *testing.T) {
	analyzer := NewAnalyzer()

	input := &AnalysisInput{
		Endpoints: []EndpointInput{
			{
				Method:    "GET",
				Path:      "/api/users",
				File:      "handler.go",
				Line:      10,
				Handler:   "listUsers",
				Framework: "gin",
			},
			{
				Method:    "POST",
				Path:      "/api/users",
				File:      "handler.go",
				Line:      20,
				Handler:   "createUser",
				Framework: "gin",
			},
		},
		Calls: []CallInput{
			{
				Method: "GET",
				URL:    "/api/users",
				File:   "api.ts",
				Line:   5,
			},
		},
	}

	result := analyzer.Analyze(input)

	if len(result.Contracts) != 1 {
		t.Errorf("expected 1 contract, got %d", len(result.Contracts))
	}

	// POST /api/users should be unmatched
	if len(result.UnmatchedBackend) != 1 {
		t.Errorf("expected 1 unmatched backend, got %d", len(result.UnmatchedBackend))
	}
}

func TestAnalyzer_TypeMismatch(t *testing.T) {
	analyzer := NewAnalyzer()

	backendSchema := NewObjectSchema()
	backendSchema.AddProperty("id", NewPrimitiveSchema(SchemaTypeString), true)
	backendSchema.AddProperty("name", NewPrimitiveSchema(SchemaTypeString), true)
	backendSchema.AddProperty("age", NewPrimitiveSchema(SchemaTypeInteger), true)

	frontendSchema := NewObjectSchema()
	frontendSchema.AddProperty("id", NewPrimitiveSchema(SchemaTypeString), true)
	frontendSchema.AddProperty("name", NewPrimitiveSchema(SchemaTypeString), true)
	frontendSchema.AddProperty("age", NewPrimitiveSchema(SchemaTypeString), true) // Wrong type!

	input := &AnalysisInput{
		Endpoints: []EndpointInput{
			{
				Method:         "GET",
				Path:           "/api/user",
				File:           "handler.go",
				Line:           10,
				Handler:        "getUser",
				ResponseSchema: backendSchema,
			},
		},
		Calls: []CallInput{
			{
				Method:         "GET",
				URL:            "/api/user",
				File:           "api.ts",
				Line:           5,
				ExpectedSchema: frontendSchema,
			},
		},
	}

	result := analyzer.Analyze(input)

	if len(result.Contracts) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(result.Contracts))
	}

	contract := result.Contracts[0]
	if len(contract.Mismatches) != 1 {
		t.Errorf("expected 1 mismatch, got %d", len(contract.Mismatches))
		for _, m := range contract.Mismatches {
			t.Logf("  - %s: %s", m.FieldPath, m.Description)
		}
	}

	if contract.Status != ContractMismatch {
		t.Errorf("expected status mismatch, got %s", contract.Status)
	}
}

func TestAnalyzer_MissingField(t *testing.T) {
	analyzer := NewAnalyzer()

	backendSchema := NewObjectSchema()
	backendSchema.AddProperty("id", NewPrimitiveSchema(SchemaTypeString), true)
	backendSchema.AddProperty("name", NewPrimitiveSchema(SchemaTypeString), true)
	// Backend has extra field

	frontendSchema := NewObjectSchema()
	frontendSchema.AddProperty("id", NewPrimitiveSchema(SchemaTypeString), true)
	// Frontend doesn't expect "name"

	input := &AnalysisInput{
		Endpoints: []EndpointInput{
			{
				Method:         "GET",
				Path:           "/api/user",
				ResponseSchema: backendSchema,
			},
		},
		Calls: []CallInput{
			{
				Method:         "GET",
				URL:            "/api/user",
				ExpectedSchema: frontendSchema,
			},
		},
	}

	result := analyzer.Analyze(input)

	if len(result.Contracts) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(result.Contracts))
	}

	contract := result.Contracts[0]
	// Should detect "name" as missing in frontend (field path includes $. prefix)
	foundMissing := false
	for _, m := range contract.Mismatches {
		if m.Type == MismatchMissingInFrontend && (m.FieldPath == "name" || m.FieldPath == "$.name") {
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Error("expected to find 'name' as missing in frontend")
		for _, m := range contract.Mismatches {
			t.Logf("  - %s: %s (%s)", m.FieldPath, m.Type, m.Description)
		}
	}
}

func TestAnalyzer_UnmatchedCalls(t *testing.T) {
	analyzer := NewAnalyzer()

	input := &AnalysisInput{
		Endpoints: []EndpointInput{
			{
				Method: "GET",
				Path:   "/api/users",
			},
		},
		Calls: []CallInput{
			{
				Method: "GET",
				URL:    "/api/users",
				File:   "api.ts",
				Line:   5,
			},
			{
				Method: "GET",
				URL:    "/api/unknown",
				File:   "api.ts",
				Line:   10,
			},
			{
				Method: "POST",
				URL:    "/api/other",
				File:   "api.ts",
				Line:   15,
			},
		},
	}

	result := analyzer.Analyze(input)

	if len(result.UnmatchedFrontend) != 2 {
		t.Errorf("expected 2 unmatched frontend calls, got %d", len(result.UnmatchedFrontend))
	}
}

func TestAnalyzer_MultipleFrontendCalls(t *testing.T) {
	analyzer := NewAnalyzer()

	input := &AnalysisInput{
		Endpoints: []EndpointInput{
			{
				Method:  "GET",
				Path:    "/api/users",
				Handler: "listUsers",
			},
		},
		Calls: []CallInput{
			{Method: "GET", URL: "/api/users", File: "page1.ts", Line: 5},
			{Method: "GET", URL: "/api/users", File: "page2.ts", Line: 10},
			{Method: "GET", URL: "/api/users", File: "page3.ts", Line: 15},
		},
	}

	result := analyzer.Analyze(input)

	if len(result.Contracts) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(result.Contracts))
	}

	contract := result.Contracts[0]
	if len(contract.FrontendCalls) != 3 {
		t.Errorf("expected 3 frontend calls, got %d", len(contract.FrontendCalls))
	}

	// More calls should increase confidence
	if contract.Confidence < 0.7 {
		t.Errorf("expected higher confidence with multiple calls, got %f", contract.Confidence)
	}
}

func TestAnalyzer_PathParams(t *testing.T) {
	analyzer := NewAnalyzer()

	input := &AnalysisInput{
		Endpoints: []EndpointInput{
			{
				Method:  "GET",
				Path:    "/api/users/:id",
				Handler: "getUser",
			},
		},
		Calls: []CallInput{
			{Method: "GET", URL: "/api/users/123"},
			{Method: "GET", URL: "/api/users/456"},
		},
	}

	result := analyzer.Analyze(input)

	if len(result.Contracts) != 1 {
		t.Errorf("expected 1 contract, got %d", len(result.Contracts))
	}

	if len(result.UnmatchedFrontend) != 0 {
		t.Errorf("expected no unmatched calls, got %d", len(result.UnmatchedFrontend))
	}
}

func TestSummarizeMismatches(t *testing.T) {
	mismatches := []FieldMismatch{
		{
			Type:        MismatchTypeMismatch,
			FieldPath:   "user.age",
			BackendType: "integer",
			FrontendType: "string",
		},
		{
			Type:        MismatchMissingInFrontend,
			FieldPath:   "user.email",
			BackendType: "string",
		},
	}

	summaries := SummarizeMismatches(mismatches)

	if len(summaries) != 2 {
		t.Errorf("expected 2 summaries, got %d", len(summaries))
	}

	// Check summaries contain relevant info
	if summaries[0] == "" {
		t.Error("expected non-empty summary")
	}
}

func TestGetMismatchSeverity(t *testing.T) {
	tests := []struct {
		mType    MismatchType
		expected string
	}{
		{MismatchTypeMismatch, "error"},
		{MismatchMissingInBackend, "error"},
		{MismatchMissingInFrontend, "warning"},
		{MismatchOptionalityMismatch, "warning"},
		{MismatchNullabilityMismatch, "warning"},
	}

	for _, tt := range tests {
		severity := GetMismatchSeverity(tt.mType)
		if severity != tt.expected {
			t.Errorf("GetMismatchSeverity(%s) = %s, want %s", tt.mType, severity, tt.expected)
		}
	}
}
