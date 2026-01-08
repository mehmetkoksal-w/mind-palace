package memory

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAddLink(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "link-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Create source and target records
	sourceID, _ := mem.AddIdea(Idea{Content: "Source idea"})
	targetID, _ := mem.AddDecision(Decision{Content: "Target decision"})

	// Add a link
	link := Link{
		SourceID:   sourceID,
		SourceKind: "idea",
		TargetID:   targetID,
		TargetKind: "decision",
		Relation:   RelationInspiredBy,
	}

	id, err := mem.AddLink(link)
	if err != nil {
		t.Fatalf("Failed to add link: %v", err)
	}

	if id == "" {
		t.Error("Expected non-empty ID")
	}
	if id[:2] != "l_" {
		t.Errorf("Expected ID to start with 'l_', got %s", id)
	}

	// Retrieve the link
	retrieved, err := mem.GetLink(id)
	if err != nil {
		t.Fatalf("Failed to get link: %v", err)
	}

	if retrieved.SourceID != sourceID {
		t.Errorf("Expected source ID %s, got %s", sourceID, retrieved.SourceID)
	}
	if retrieved.Relation != RelationInspiredBy {
		t.Errorf("Expected relation %s, got %s", RelationInspiredBy, retrieved.Relation)
	}
}

func TestAddLinkValidation(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "link-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	tests := []struct {
		name    string
		link    Link
		wantErr bool
	}{
		{
			name:    "missing source_id",
			link:    Link{SourceKind: "idea", TargetID: "x", TargetKind: "decision", Relation: "related"},
			wantErr: true,
		},
		{
			name:    "missing source_kind",
			link:    Link{SourceID: "x", TargetID: "y", TargetKind: "decision", Relation: "related"},
			wantErr: true,
		},
		{
			name:    "missing target_id",
			link:    Link{SourceID: "x", SourceKind: "idea", TargetKind: "decision", Relation: "related"},
			wantErr: true,
		},
		{
			name:    "missing target_kind",
			link:    Link{SourceID: "x", SourceKind: "idea", TargetID: "y", Relation: "related"},
			wantErr: true,
		},
		{
			name:    "missing relation",
			link:    Link{SourceID: "x", SourceKind: "idea", TargetID: "y", TargetKind: "decision"},
			wantErr: true,
		},
		{
			name:    "invalid relation",
			link:    Link{SourceID: "x", SourceKind: "idea", TargetID: "y", TargetKind: "decision", Relation: "invalid"},
			wantErr: true,
		},
		{
			name:    "valid link",
			link:    Link{SourceID: "x", SourceKind: "idea", TargetID: "y", TargetKind: "decision", Relation: "related"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mem.AddLink(tt.link)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddLink() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetLinksForSource(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "link-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	sourceID, _ := mem.AddIdea(Idea{Content: "Source"})
	target1, _ := mem.AddDecision(Decision{Content: "Target 1"})
	target2, _ := mem.AddDecision(Decision{Content: "Target 2"})
	target3, _ := mem.AddIdea(Idea{Content: "Target 3"})

	// Add links from source to targets
	mem.AddLink(Link{SourceID: sourceID, SourceKind: "idea", TargetID: target1, TargetKind: "decision", Relation: RelationInspiredBy})
	mem.AddLink(Link{SourceID: sourceID, SourceKind: "idea", TargetID: target2, TargetKind: "decision", Relation: RelationSupports})
	mem.AddLink(Link{SourceID: sourceID, SourceKind: "idea", TargetID: target3, TargetKind: "idea", Relation: RelationRelated})

	// Add a link from different source
	otherSource, _ := mem.AddIdea(Idea{Content: "Other"})
	mem.AddLink(Link{SourceID: otherSource, SourceKind: "idea", TargetID: target1, TargetKind: "decision", Relation: RelationRelated})

	// Get links for source
	links, err := mem.GetLinksForSource(sourceID)
	if err != nil {
		t.Fatalf("Failed to get links: %v", err)
	}

	if len(links) != 3 {
		t.Errorf("Expected 3 links, got %d", len(links))
	}
}

func TestGetLinksForTarget(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "link-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	targetID, _ := mem.AddDecision(Decision{Content: "Target"})
	source1, _ := mem.AddIdea(Idea{Content: "Source 1"})
	source2, _ := mem.AddIdea(Idea{Content: "Source 2"})

	// Add links to target
	mem.AddLink(Link{SourceID: source1, SourceKind: "idea", TargetID: targetID, TargetKind: "decision", Relation: RelationSupports})
	mem.AddLink(Link{SourceID: source2, SourceKind: "idea", TargetID: targetID, TargetKind: "decision", Relation: RelationContradicts})

	// Get links for target
	links, err := mem.GetLinksForTarget(targetID)
	if err != nil {
		t.Fatalf("Failed to get links: %v", err)
	}

	if len(links) != 2 {
		t.Errorf("Expected 2 links, got %d", len(links))
	}
}

func TestGetAllLinksFor(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "link-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id1, _ := mem.AddIdea(Idea{Content: "Record 1"})
	id2, _ := mem.AddDecision(Decision{Content: "Record 2"})
	id3, _ := mem.AddIdea(Idea{Content: "Record 3"})

	// id1 is source for id2, id3 is source for id1
	mem.AddLink(Link{SourceID: id1, SourceKind: "idea", TargetID: id2, TargetKind: "decision", Relation: RelationSupports})
	mem.AddLink(Link{SourceID: id3, SourceKind: "idea", TargetID: id1, TargetKind: "idea", Relation: RelationRelated})

	// Get all links for id1 (should be 2: one as source, one as target)
	links, err := mem.GetAllLinksFor(id1)
	if err != nil {
		t.Fatalf("Failed to get links: %v", err)
	}

	if len(links) != 2 {
		t.Errorf("Expected 2 links, got %d", len(links))
	}
}

func TestDeleteLink(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "link-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id, _ := mem.AddLink(Link{SourceID: "x", SourceKind: "idea", TargetID: "y", TargetKind: "decision", Relation: RelationRelated})

	// Delete the link
	err := mem.DeleteLink(id)
	if err != nil {
		t.Fatalf("Failed to delete link: %v", err)
	}

	// Verify it's gone
	_, err = mem.GetLink(id)
	if err == nil {
		t.Error("Expected error getting deleted link")
	}

	// Delete non-existent link
	err = mem.DeleteLink("nonexistent")
	if err == nil {
		t.Error("Expected error deleting non-existent link")
	}
}

func TestDeleteLinksForRecord(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "link-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	recordID := "test_record"

	// Create links where record is source and target
	mem.AddLink(Link{SourceID: recordID, SourceKind: "idea", TargetID: "other1", TargetKind: "decision", Relation: RelationSupports})
	mem.AddLink(Link{SourceID: recordID, SourceKind: "idea", TargetID: "other2", TargetKind: "idea", Relation: RelationRelated})
	mem.AddLink(Link{SourceID: "other3", SourceKind: "decision", TargetID: recordID, TargetKind: "idea", Relation: RelationContradicts})

	// Verify links exist
	links, _ := mem.GetAllLinksFor(recordID)
	if len(links) != 3 {
		t.Errorf("Expected 3 links before delete, got %d", len(links))
	}

	// Delete all links for record
	err := mem.DeleteLinksForRecord(recordID)
	if err != nil {
		t.Fatalf("Failed to delete links: %v", err)
	}

	// Verify links are gone
	links, _ = mem.GetAllLinksFor(recordID)
	if len(links) != 0 {
		t.Errorf("Expected 0 links after delete, got %d", len(links))
	}
}

func TestGetLinksByRelation(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "link-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add links with different relations
	mem.AddLink(Link{SourceID: "a", SourceKind: "idea", TargetID: "b", TargetKind: "decision", Relation: RelationSupports})
	mem.AddLink(Link{SourceID: "c", SourceKind: "idea", TargetID: "d", TargetKind: "decision", Relation: RelationSupports})
	mem.AddLink(Link{SourceID: "e", SourceKind: "decision", TargetID: "f", TargetKind: "decision", Relation: RelationSupersedes})
	mem.AddLink(Link{SourceID: "g", SourceKind: "idea", TargetID: "h", TargetKind: "idea", Relation: RelationContradicts})

	// Get supports links
	supports, _ := mem.GetLinksByRelation(RelationSupports, 10)
	if len(supports) != 2 {
		t.Errorf("Expected 2 supports links, got %d", len(supports))
	}

	// Get supersedes links
	supersedes, _ := mem.GetLinksByRelation(RelationSupersedes, 10)
	if len(supersedes) != 1 {
		t.Errorf("Expected 1 supersedes link, got %d", len(supersedes))
	}
}

func TestLinkStaleness(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "link-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add a link that's not stale
	id, _ := mem.AddLink(Link{
		SourceID:    "idea1",
		SourceKind:  "idea",
		TargetID:    "code/file.go:10-20",
		TargetKind:  TargetKindCode,
		Relation:    RelationImplements,
		TargetMtime: time.Now().UTC(),
		IsStale:     false,
	})

	// Mark as stale
	err := mem.MarkLinkStale(id, true)
	if err != nil {
		t.Fatalf("Failed to mark link stale: %v", err)
	}

	// Verify it's stale
	link, _ := mem.GetLink(id)
	if !link.IsStale {
		t.Error("Expected link to be stale")
	}

	// Get stale links
	staleLinks, _ := mem.GetStaleLinks()
	if len(staleLinks) != 1 {
		t.Errorf("Expected 1 stale link, got %d", len(staleLinks))
	}

	// Mark as not stale
	mem.MarkLinkStale(id, false)
	link, _ = mem.GetLink(id)
	if link.IsStale {
		t.Error("Expected link to not be stale")
	}
}

func TestParseCodeTarget(t *testing.T) {
	tests := []struct {
		input     string
		filePath  string
		startLine int
		endLine   int
	}{
		{"auth/jwt.go", "auth/jwt.go", 0, 0},
		{"auth/jwt.go:15", "auth/jwt.go", 15, 15},
		{"auth/jwt.go:15-45", "auth/jwt.go", 15, 45},
		{"src/api/handler.go:100-200", "src/api/handler.go", 100, 200},
		{"./relative/path.go:1", "relative/path.go", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			parsed, err := ParseCodeTarget(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}
			if parsed.FilePath != tt.filePath {
				t.Errorf("Expected file path %q, got %q", tt.filePath, parsed.FilePath)
			}
			if parsed.StartLine != tt.startLine {
				t.Errorf("Expected start line %d, got %d", tt.startLine, parsed.StartLine)
			}
			if parsed.EndLine != tt.endLine {
				t.Errorf("Expected end line %d, got %d", tt.endLine, parsed.EndLine)
			}
		})
	}
}

func TestValidateCodeTarget(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "link-test-*")
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.go")
	content := "line1\nline2\nline3\nline4\nline5\n"
	os.WriteFile(testFile, []byte(content), 0o644)

	tests := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{"valid file only", "test.go", false},
		{"valid file with line", "test.go:3", false},
		{"valid file with range", "test.go:2-4", false},
		{"nonexistent file", "nonexistent.go", true},
		{"line exceeds file", "test.go:100", true},
		{"end line exceeds file", "test.go:1-100", true},
		{"start after end", "test.go:5-2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ValidateCodeTarget(tmpDir, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCodeTarget() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckAndUpdateStaleness(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "link-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.go")
	os.WriteFile(testFile, []byte("content\n"), 0o644)

	// Get file mtime
	info, _ := os.Stat(testFile)
	oldMtime := info.ModTime()

	// Add a code link with old mtime
	mem.AddLink(Link{
		SourceID:    "idea1",
		SourceKind:  "idea",
		TargetID:    "test.go:1",
		TargetKind:  TargetKindCode,
		Relation:    RelationImplements,
		TargetMtime: oldMtime.Add(-1 * time.Hour), // Pretend link was created before file was modified
		IsStale:     false,
	})

	// Check staleness - should mark as stale since file mtime > link mtime
	staleCount, err := mem.CheckAndUpdateStaleness(tmpDir)
	if err != nil {
		t.Fatalf("Failed to check staleness: %v", err)
	}

	if staleCount != 1 {
		t.Errorf("Expected 1 stale link, got %d", staleCount)
	}
}
