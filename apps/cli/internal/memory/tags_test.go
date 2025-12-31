package memory

import (
	"os"
	"testing"
)

func TestSetTags(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tags-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Create an idea to tag
	id, _ := mem.AddIdea(Idea{Content: "Test idea"})

	// Set tags
	err := mem.SetTags(id, "idea", []string{"performance", "api", "caching"})
	if err != nil {
		t.Fatalf("Failed to set tags: %v", err)
	}

	// Verify tags
	tags, _ := mem.GetTags(id, "idea")
	if len(tags) != 3 {
		t.Errorf("Expected 3 tags, got %d: %v", len(tags), tags)
	}

	// Replace tags
	err = mem.SetTags(id, "idea", []string{"new-tag"})
	if err != nil {
		t.Fatalf("Failed to replace tags: %v", err)
	}

	tags, _ = mem.GetTags(id, "idea")
	if len(tags) != 1 {
		t.Errorf("Expected 1 tag after replace, got %d: %v", len(tags), tags)
	}
	if tags[0] != "new-tag" {
		t.Errorf("Expected 'new-tag', got '%s'", tags[0])
	}
}

func TestSetTagsNormalization(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tags-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id, _ := mem.AddIdea(Idea{Content: "Test"})

	// Tags should be normalized (lowercase, trimmed)
	err := mem.SetTags(id, "idea", []string{"  API  ", "Performance", "CACHING"})
	if err != nil {
		t.Fatalf("Failed to set tags: %v", err)
	}

	tags, _ := mem.GetTags(id, "idea")
	expected := []string{"api", "caching", "performance"} // sorted
	if len(tags) != len(expected) {
		t.Errorf("Expected %d tags, got %d: %v", len(expected), len(tags), tags)
	}
	for i, tag := range tags {
		if tag != expected[i] {
			t.Errorf("Expected tag '%s', got '%s'", expected[i], tag)
		}
	}
}

func TestSetTagsEmpty(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tags-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id, _ := mem.AddIdea(Idea{Content: "Test"})

	// Set some tags first
	mem.SetTags(id, "idea", []string{"tag1", "tag2"})

	// Set empty tags (should clear)
	err := mem.SetTags(id, "idea", []string{})
	if err != nil {
		t.Fatalf("Failed to set empty tags: %v", err)
	}

	tags, _ := mem.GetTags(id, "idea")
	if len(tags) != 0 {
		t.Errorf("Expected 0 tags, got %d: %v", len(tags), tags)
	}
}

func TestAddTag(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tags-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id, _ := mem.AddDecision(Decision{Content: "Test"})

	// Add tags one by one
	mem.AddTag(id, "decision", "api")
	mem.AddTag(id, "decision", "performance")
	mem.AddTag(id, "decision", "api") // Duplicate should be ignored

	tags, _ := mem.GetTags(id, "decision")
	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d: %v", len(tags), tags)
	}
}

func TestAddTagEmpty(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tags-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id, _ := mem.AddIdea(Idea{Content: "Test"})

	// Adding empty tag should fail
	err := mem.AddTag(id, "idea", "")
	if err == nil {
		t.Error("Expected error for empty tag")
	}

	err = mem.AddTag(id, "idea", "   ")
	if err == nil {
		t.Error("Expected error for whitespace-only tag")
	}
}

func TestRemoveTag(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tags-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id, _ := mem.AddIdea(Idea{Content: "Test"})
	mem.SetTags(id, "idea", []string{"tag1", "tag2", "tag3"})

	// Remove one tag
	err := mem.RemoveTag(id, "idea", "tag2")
	if err != nil {
		t.Fatalf("Failed to remove tag: %v", err)
	}

	tags, _ := mem.GetTags(id, "idea")
	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d: %v", len(tags), tags)
	}

	// Verify tag2 is gone
	for _, tag := range tags {
		if tag == "tag2" {
			t.Error("tag2 should have been removed")
		}
	}

	// Remove non-existent tag (should not error)
	err = mem.RemoveTag(id, "idea", "nonexistent")
	if err != nil {
		t.Errorf("Unexpected error removing non-existent tag: %v", err)
	}
}

func TestGetRecordsByTag(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tags-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Create records with tags
	id1, _ := mem.AddIdea(Idea{Content: "Idea 1"})
	id2, _ := mem.AddIdea(Idea{Content: "Idea 2"})
	id3, _ := mem.AddDecision(Decision{Content: "Decision 1"})

	mem.SetTags(id1, "idea", []string{"api", "performance"})
	mem.SetTags(id2, "idea", []string{"api", "security"})
	mem.SetTags(id3, "decision", []string{"api", "database"})

	// Get all records with "api" tag
	allAPI, _ := mem.GetRecordsByTag("api", "")
	if len(allAPI) != 3 {
		t.Errorf("Expected 3 records with 'api' tag, got %d: %v", len(allAPI), allAPI)
	}

	// Get only ideas with "api" tag
	ideasAPI, _ := mem.GetRecordsByTag("api", "idea")
	if len(ideasAPI) != 2 {
		t.Errorf("Expected 2 ideas with 'api' tag, got %d: %v", len(ideasAPI), ideasAPI)
	}

	// Get records with "performance" tag
	perf, _ := mem.GetRecordsByTag("performance", "")
	if len(perf) != 1 {
		t.Errorf("Expected 1 record with 'performance' tag, got %d", len(perf))
	}
}

func TestGetAllTags(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tags-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id1, _ := mem.AddIdea(Idea{Content: "Idea 1"})
	id2, _ := mem.AddDecision(Decision{Content: "Decision 1"})

	mem.SetTags(id1, "idea", []string{"api", "performance"})
	mem.SetTags(id2, "decision", []string{"api", "database"})

	// Get all tags
	allTags, _ := mem.GetAllTags("")
	if len(allTags) != 3 {
		t.Errorf("Expected 3 unique tags, got %d: %v", len(allTags), allTags)
	}

	// Get only idea tags
	ideaTags, _ := mem.GetAllTags("idea")
	if len(ideaTags) != 2 {
		t.Errorf("Expected 2 idea tags, got %d: %v", len(ideaTags), ideaTags)
	}
}

func TestGetTagCounts(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tags-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id1, _ := mem.AddIdea(Idea{Content: "1"})
	id2, _ := mem.AddIdea(Idea{Content: "2"})
	id3, _ := mem.AddIdea(Idea{Content: "3"})

	mem.SetTags(id1, "idea", []string{"api", "performance"})
	mem.SetTags(id2, "idea", []string{"api", "security"})
	mem.SetTags(id3, "idea", []string{"api"})

	// Get counts
	counts, _ := mem.GetTagCounts("idea", 10)
	if counts["api"] != 3 {
		t.Errorf("Expected 'api' count 3, got %d", counts["api"])
	}
	if counts["performance"] != 1 {
		t.Errorf("Expected 'performance' count 1, got %d", counts["performance"])
	}
	if counts["security"] != 1 {
		t.Errorf("Expected 'security' count 1, got %d", counts["security"])
	}

	// Test limit
	limitedCounts, _ := mem.GetTagCounts("idea", 1)
	if len(limitedCounts) != 1 {
		t.Errorf("Expected 1 tag with limit, got %d", len(limitedCounts))
	}
	// Should be "api" since it has highest count
	if _, ok := limitedCounts["api"]; !ok {
		t.Error("Expected 'api' to be the top tag")
	}
}

func TestSearchByTags(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tags-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id1, _ := mem.AddIdea(Idea{Content: "1"})
	id2, _ := mem.AddIdea(Idea{Content: "2"})
	id3, _ := mem.AddIdea(Idea{Content: "3"})

	mem.SetTags(id1, "idea", []string{"api", "performance", "caching"})
	mem.SetTags(id2, "idea", []string{"api", "performance"})
	mem.SetTags(id3, "idea", []string{"api"})

	// Search for records with both "api" AND "performance"
	results, _ := mem.SearchByTags([]string{"api", "performance"}, "idea", 10)
	if len(results) != 2 {
		t.Errorf("Expected 2 records with api+performance, got %d: %v", len(results), results)
	}

	// Search for records with all three tags
	results3, _ := mem.SearchByTags([]string{"api", "performance", "caching"}, "idea", 10)
	if len(results3) != 1 {
		t.Errorf("Expected 1 record with all 3 tags, got %d: %v", len(results3), results3)
	}
	if len(results3) > 0 && results3[0] != id1 {
		t.Errorf("Expected %s, got %s", id1, results3[0])
	}

	// Empty tags should return nil
	empty, _ := mem.SearchByTags([]string{}, "", 10)
	if empty != nil {
		t.Errorf("Expected nil for empty tags, got %v", empty)
	}
}

func TestDeleteTagsForRecord(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tags-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id, _ := mem.AddIdea(Idea{Content: "Test"})
	mem.SetTags(id, "idea", []string{"tag1", "tag2", "tag3"})

	// Delete all tags
	err := mem.DeleteTagsForRecord(id, "idea")
	if err != nil {
		t.Fatalf("Failed to delete tags: %v", err)
	}

	tags, _ := mem.GetTags(id, "idea")
	if len(tags) != 0 {
		t.Errorf("Expected 0 tags after delete, got %d: %v", len(tags), tags)
	}
}

func TestTagsAcrossRecordKinds(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tags-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	ideaID, _ := mem.AddIdea(Idea{Content: "Idea"})
	decID, _ := mem.AddDecision(Decision{Content: "Decision"})

	// Same tag on different record kinds
	mem.AddTag(ideaID, "idea", "shared-tag")
	mem.AddTag(decID, "decision", "shared-tag")

	// Each should have its own tag
	ideaTags, _ := mem.GetTags(ideaID, "idea")
	decTags, _ := mem.GetTags(decID, "decision")

	if len(ideaTags) != 1 || ideaTags[0] != "shared-tag" {
		t.Errorf("Expected idea to have 'shared-tag', got %v", ideaTags)
	}
	if len(decTags) != 1 || decTags[0] != "shared-tag" {
		t.Errorf("Expected decision to have 'shared-tag', got %v", decTags)
	}

	// GetRecordsByTag should find both
	allShared, _ := mem.GetRecordsByTag("shared-tag", "")
	if len(allShared) != 2 {
		t.Errorf("Expected 2 records with 'shared-tag', got %d", len(allShared))
	}

	// But filtered should find only one
	ideasShared, _ := mem.GetRecordsByTag("shared-tag", "idea")
	if len(ideasShared) != 1 {
		t.Errorf("Expected 1 idea with 'shared-tag', got %d", len(ideasShared))
	}
}

func TestTagsCaseSensitivity(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tags-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id, _ := mem.AddIdea(Idea{Content: "Test"})

	// Add tags with different cases
	mem.AddTag(id, "idea", "API")
	mem.AddTag(id, "idea", "api")
	mem.AddTag(id, "idea", "Api")

	// Should only have one tag (normalized)
	tags, _ := mem.GetTags(id, "idea")
	if len(tags) != 1 {
		t.Errorf("Expected 1 tag (case-normalized), got %d: %v", len(tags), tags)
	}
	if tags[0] != "api" {
		t.Errorf("Expected 'api', got '%s'", tags[0])
	}
}
