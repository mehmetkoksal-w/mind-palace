package model_test

import (
	"encoding/json"
	"testing"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
	"github.com/koksalmehmet/mind-palace/apps/cli/schemas"
)

func TestContextPackJSONSchemaRoundTrip(t *testing.T) {
	cp := model.NewContextPack("Ship the release")
	cp.ScanID = "scan-42"
	cp.ScanHash = "hash-123"
	cp.ScanTime = "2025-12-19T12:00:00Z"
	cp.RoomsVisited = []string{"project-overview"}
	cp.FilesReferenced = []string{"notes.txt"}
	cp.SymbolsReferenced = []string{"main"}
	cp.Findings = []model.Finding{{Summary: "found", Severity: "info"}}
	cp.Plan = []model.PlanStep{{Step: "do it", Status: "pending"}}
	cp.Verification = []model.VerificationResult{{Name: "verify", Status: "pass"}}
	cp.Scope = &model.ScopeInfo{Mode: "full", Source: "full-scan", FileCount: 1}
	cp.Provenance.CreatedBy = "palace"
	cp.Provenance.CreatedAt = "2025-12-19T12:00:00Z"
	cp.Provenance.Generator = "palace"
	cp.Provenance.GeneratorVersion = "0.0.1"

	data, err := json.Marshal(cp)
	if err != nil {
		t.Fatalf("marshal context pack: %v", err)
	}

	validateSchema(t, schemas.ContextPack, data)

	var decoded model.ContextPack
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal context pack: %v", err)
	}
	if decoded.ScanHash != cp.ScanHash {
		t.Fatalf("scanHash mismatch: got %q want %q", decoded.ScanHash, cp.ScanHash)
	}
	if decoded.Scope == nil || decoded.Scope.Mode != "full" {
		t.Fatalf("scope not decoded as expected: %+v", decoded.Scope)
	}

	var instance map[string]any
	if err := json.Unmarshal(data, &instance); err != nil {
		t.Fatalf("decode context pack json: %v", err)
	}
	instance["extra"] = "nope"
	assertSchemaInvalid(t, schemas.ContextPack, instance)
	delete(instance, "goal")
	assertSchemaInvalid(t, schemas.ContextPack, instance)
}

func TestScanSummaryJSONSchemaRoundTrip(t *testing.T) {
	summary := model.ScanSummary{
		SchemaVersion: "1.0.0",
		Kind:          "palace/scan",
		ScanID:        "scan-uuid",
		DBScanID:      1,
		StartedAt:     "2025-12-19T12:00:00Z",
		CompletedAt:   "2025-12-19T12:01:00Z",
		FileCount:     2,
		ChunkCount:    3,
		ScanHash:      "hash-abc",
		Provenance: model.Provenance{
			CreatedBy: "palace scan",
			CreatedAt: "2025-12-19T12:01:00Z",
		},
	}

	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("marshal scan summary: %v", err)
	}

	validateSchema(t, schemas.ScanSummary, data)

	var decoded model.ScanSummary
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal scan summary: %v", err)
	}
	if decoded.ScanID != summary.ScanID {
		t.Fatalf("scanId mismatch: got %q want %q", decoded.ScanID, summary.ScanID)
	}
	if decoded.DBScanID != summary.DBScanID {
		t.Fatalf("dbScanId mismatch: got %d want %d", decoded.DBScanID, summary.DBScanID)
	}

	var instance map[string]any
	if err := json.Unmarshal(data, &instance); err != nil {
		t.Fatalf("decode scan summary json: %v", err)
	}
	instance["extra"] = "nope"
	assertSchemaInvalid(t, schemas.ScanSummary, instance)
	delete(instance, "scanHash")
	assertSchemaInvalid(t, schemas.ScanSummary, instance)
}

func validateSchema(t *testing.T, schemaName string, data []byte) {
	t.Helper()
	schema, err := schemas.Compile(schemaName)
	if err != nil {
		t.Fatalf("compile schema %s: %v", schemaName, err)
	}
	var instance any
	if err := json.Unmarshal(data, &instance); err != nil {
		t.Fatalf("unmarshal instance for %s: %v", schemaName, err)
	}
	if err := schema.Validate(instance); err != nil {
		t.Fatalf("schema %s validation failed: %v", schemaName, err)
	}
}

func assertSchemaInvalid(t *testing.T, schemaName string, instance any) {
	t.Helper()
	schema, err := schemas.Compile(schemaName)
	if err != nil {
		t.Fatalf("compile schema %s: %v", schemaName, err)
	}
	if err := schema.Validate(instance); err == nil {
		t.Fatalf("expected schema %s validation error", schemaName)
	}
}
