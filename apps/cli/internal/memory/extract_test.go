package memory

import (
	"testing"
	"time"
)

func TestStubExtractor(t *testing.T) {
	extractor := NewStubExtractor()

	conv := Conversation{
		ID:      "c_test123",
		Summary: "Test conversation",
		Messages: []Message{
			{Role: "user", Content: "Let's use JWT for authentication", Timestamp: time.Now()},
			{Role: "assistant", Content: "Good idea, JWT is stateless and scalable", Timestamp: time.Now()},
		},
	}

	extracted, err := extractor.ExtractFromConversation(conv)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Stub should return empty slice
	if len(extracted) != 0 {
		t.Errorf("Expected 0 extracted records from stub, got %d", len(extracted))
	}
}

func TestExtractRecords(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "We should use PostgreSQL", Timestamp: time.Now()},
		{Role: "assistant", Content: "Agreed, I learned that PostgreSQL handles JSON well", Timestamp: time.Now()},
	}

	records := ExtractRecords(messages)

	// ExtractRecords now uses keyword-based classification
	// The assistant message contains "learned" which triggers learning extraction
	if len(records) != 1 {
		t.Errorf("Expected 1 extracted record, got %d", len(records))
	}
	if len(records) > 0 && records[0].Kind != "learning" {
		t.Errorf("Expected learning kind, got %s", records[0].Kind)
	}
}

func TestDefaultAutoExtractConfig(t *testing.T) {
	config := DefaultAutoExtractConfig()

	if config.Enabled {
		t.Error("Expected auto-extract to be disabled by default")
	}
	if config.MinMessages != 5 {
		t.Errorf("Expected MinMessages to be 5, got %d", config.MinMessages)
	}
	if config.ExtractorType != "stub" {
		t.Errorf("Expected ExtractorType to be 'stub', got %s", config.ExtractorType)
	}
}

func TestExtractedRecordStruct(t *testing.T) {
	record := ExtractedRecord{
		Kind:    RecordKindDecision,
		Content: "Use JWT for authentication",
		Context: "Discussed in authentication planning session",
	}

	if record.Kind != RecordKindDecision {
		t.Errorf("Expected kind decision, got %s", record.Kind)
	}
	if record.Content == "" {
		t.Error("Expected non-empty content")
	}
	if record.Context != "Discussed in authentication planning session" {
		t.Error("Context mismatch")
	}
}
