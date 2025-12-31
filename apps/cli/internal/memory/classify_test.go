package memory

import (
	"os"
	"testing"
)

func TestClassifyDecisions(t *testing.T) {
	tests := []struct {
		input      string
		wantKind   RecordKind
		wantHigher float64 // minimum confidence expected
	}{
		{"We should use PostgreSQL for the database", RecordKindDecision, 0.7},
		{"Let's go with JWT for authentication", RecordKindDecision, 0.7},
		{"I decided to use React for the frontend", RecordKindDecision, 0.7},
		{"Decision: use microservices architecture", RecordKindDecision, 0.7},
		{"We'll implement caching with Redis", RecordKindDecision, 0.7},
		{"Going with TypeScript for this project", RecordKindDecision, 0.7},
		{"Chose to implement rate limiting", RecordKindDecision, 0.7},
		{"The plan is to deploy on Kubernetes", RecordKindDecision, 0.7},
		{"We're going to use GraphQL", RecordKindDecision, 0.7},
		{"Agreed on using bcrypt for passwords", RecordKindDecision, 0.7},
		{"Switching to Postgres from MySQL", RecordKindDecision, 0.7},
		{"Settled on using Docker for deployment", RecordKindDecision, 0.7},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Classify(tt.input)
			if result.Kind != tt.wantKind {
				t.Errorf("Classify(%q) = %v, want %v", tt.input, result.Kind, tt.wantKind)
			}
			if result.Confidence < tt.wantHigher {
				t.Errorf("Classify(%q) confidence = %v, want >= %v", tt.input, result.Confidence, tt.wantHigher)
			}
		})
	}
}

func TestClassifyIdeas(t *testing.T) {
	tests := []struct {
		input      string
		wantKind   RecordKind
		wantHigher float64
	}{
		{"What if we used GraphQL instead of REST?", RecordKindIdea, 0.7},
		{"Maybe we could add a caching layer", RecordKindIdea, 0.7},
		{"How about implementing websockets?", RecordKindIdea, 0.7},
		{"Idea: add dark mode support", RecordKindIdea, 0.7},
		{"Consider using a message queue", RecordKindIdea, 0.7},
		{"Perhaps we should explore microservices", RecordKindIdea, 0.7},
		{"Wouldn't it be nice to have auto-save?", RecordKindIdea, 0.7},
		{"Could we add offline support?", RecordKindIdea, 0.7},
		{"Explore using WebAssembly for performance", RecordKindIdea, 0.7},
		{"Thought: what about using Rust for the CLI?", RecordKindIdea, 0.7},
		{"Suggestion: implement feature flags", RecordKindIdea, 0.7},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Classify(tt.input)
			if result.Kind != tt.wantKind {
				t.Errorf("Classify(%q) = %v, want %v", tt.input, result.Kind, tt.wantKind)
			}
			if result.Confidence < tt.wantHigher {
				t.Errorf("Classify(%q) confidence = %v, want >= %v", tt.input, result.Confidence, tt.wantHigher)
			}
		})
	}
}

func TestClassifyLearnings(t *testing.T) {
	tests := []struct {
		input      string
		wantKind   RecordKind
		wantHigher float64
	}{
		{"TIL SQLite can handle 100k writes per second", RecordKindLearning, 0.7},
		{"Learned that Go channels are not thread-safe by default", RecordKindLearning, 0.7},
		{"Turns out you need to close the response body in Go", RecordKindLearning, 0.7},
		{"Note: always use prepared statements", RecordKindLearning, 0.7},
		{"Realized that indexes slow down writes", RecordKindLearning, 0.7},
		{"Discovered that defer runs in LIFO order", RecordKindLearning, 0.7},
		{"Found out that context cancellation is important", RecordKindLearning, 0.7},
		{"Important: never store passwords in plain text", RecordKindLearning, 0.7},
		{"Key takeaway: test your error paths", RecordKindLearning, 0.7},
		{"Insight: premature optimization is the root of all evil", RecordKindLearning, 0.7},
		{"Figured out how to use generics in Go", RecordKindLearning, 0.7},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Classify(tt.input)
			if result.Kind != tt.wantKind {
				t.Errorf("Classify(%q) = %v, want %v", tt.input, result.Kind, tt.wantKind)
			}
			if result.Confidence < tt.wantHigher {
				t.Errorf("Classify(%q) confidence = %v, want >= %v", tt.input, result.Confidence, tt.wantHigher)
			}
		})
	}
}

func TestClassifyAmbiguous(t *testing.T) {
	// These inputs are ambiguous and should have low confidence
	tests := []struct {
		input string
	}{
		{"PostgreSQL is a relational database"},
		{"The API returns JSON"},
		{"This code is complex"},
		{"Need to fix this bug"},
		{"Something is broken"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Classify(tt.input)
			if result.Confidence >= ConfidenceThreshold {
				t.Errorf("Classify(%q) confidence = %v, want < %v for ambiguous input",
					tt.input, result.Confidence, ConfidenceThreshold)
			}
		})
	}
}

func TestClassifyNeedsConfirmation(t *testing.T) {
	// High confidence - should not need confirmation
	highConf := Classify("We should use PostgreSQL")
	if highConf.NeedsConfirmation() {
		t.Errorf("High confidence classification should not need confirmation")
	}

	// Low confidence - should need confirmation
	lowConf := Classify("PostgreSQL database")
	if !lowConf.NeedsConfirmation() {
		t.Errorf("Low confidence classification should need confirmation")
	}
}

func TestExtractTags(t *testing.T) {
	tests := []struct {
		input    string
		wantTags []string
	}{
		{"Add caching #performance #api", []string{"performance", "api"}},
		{"Fix bug #urgent", []string{"urgent"}},
		{"No tags here", nil},
		{"#API #api", []string{"api"}}, // Should dedupe case-insensitively
		{"Multiple #tags #here #test", []string{"tags", "here", "test"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tags := ExtractTags(tt.input)
			if len(tags) != len(tt.wantTags) {
				t.Errorf("ExtractTags(%q) = %v, want %v", tt.input, tags, tt.wantTags)
				return
			}
			for i, tag := range tags {
				if tag != tt.wantTags[i] {
					t.Errorf("ExtractTags(%q)[%d] = %v, want %v", tt.input, i, tag, tt.wantTags[i])
				}
			}
		})
	}
}

func TestClassificationString(t *testing.T) {
	c := Classification{Kind: RecordKindDecision}
	if c.String() != "decision" {
		t.Errorf("Expected 'decision', got '%s'", c.String())
	}
}

func TestClassifyAndStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "classify-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mem, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	defer mem.Close()

	tests := []struct {
		input    string
		wantKind RecordKind
	}{
		{"We should use Redis for caching #performance", RecordKindDecision},
		{"What if we added real-time updates? #feature", RecordKindIdea},
		{"TIL: SQLite is surprisingly fast #database", RecordKindLearning},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			id, kind, classification, err := mem.ClassifyAndStore(tt.input, "test", "")
			if err != nil {
				t.Fatalf("ClassifyAndStore failed: %v", err)
			}
			if kind != tt.wantKind {
				t.Errorf("ClassifyAndStore(%q) kind = %v, want %v", tt.input, kind, tt.wantKind)
			}
			if id == "" {
				t.Errorf("ClassifyAndStore(%q) returned empty ID", tt.input)
			}
			if classification.Kind != tt.wantKind {
				t.Errorf("Classification kind mismatch")
			}

			// Verify record was stored
			switch kind {
			case RecordKindIdea:
				idea, err := mem.GetIdea(id)
				if err != nil {
					t.Errorf("Failed to get stored idea: %v", err)
				}
				if idea.Content != tt.input {
					t.Errorf("Stored idea content mismatch")
				}
			case RecordKindDecision:
				dec, err := mem.GetDecision(id)
				if err != nil {
					t.Errorf("Failed to get stored decision: %v", err)
				}
				if dec.Content != tt.input {
					t.Errorf("Stored decision content mismatch")
				}
			case RecordKindLearning:
				learning, err := mem.GetLearning(id)
				if err != nil {
					t.Errorf("Failed to get stored learning: %v", err)
				}
				if learning.Content != tt.input {
					t.Errorf("Stored learning content mismatch")
				}
			}
		})
	}
}

func TestClassifyEdgeCases(t *testing.T) {
	// Empty string
	empty := Classify("")
	if empty.Kind != RecordKindIdea {
		t.Errorf("Empty string should default to idea, got %v", empty.Kind)
	}
	if empty.Confidence >= ConfidenceThreshold {
		t.Errorf("Empty string should have low confidence")
	}

	// Whitespace only
	whitespace := Classify("   \t\n  ")
	if whitespace.Kind != RecordKindIdea {
		t.Errorf("Whitespace should default to idea, got %v", whitespace.Kind)
	}

	// Very long input
	longInput := "We should use PostgreSQL " + string(make([]byte, 10000))
	long := Classify(longInput)
	if long.Kind != RecordKindDecision {
		t.Errorf("Long input starting with decision signal should be decision")
	}

	// Mixed signals - first/strongest wins
	mixed := Classify("We should consider what if we used Redis")
	// Should classify based on which signal is stronger/earlier
	if mixed.Kind != RecordKindDecision && mixed.Kind != RecordKindIdea {
		t.Errorf("Mixed signals should still produce a valid classification")
	}
}

func TestClassifyPatternMatching(t *testing.T) {
	// Test regex patterns specifically
	tests := []struct {
		input    string
		wantKind RecordKind
	}{
		{"We should implement caching", RecordKindDecision},
		{"We will deploy tomorrow", RecordKindDecision},
		{"We are going to refactor", RecordKindDecision},
		{"Let's use Go", RecordKindDecision},
		{"Lets start fresh", RecordKindDecision},
		{"I'm going to fix this", RecordKindDecision},
		{"I am going with option A", RecordKindDecision},
		{"The decision is final", RecordKindDecision},
		{"What if we tried something new?", RecordKindIdea},
		{"How about using Docker?", RecordKindIdea},
		{"Maybe we could optimize this?", RecordKindIdea},
		{"Wouldn't it be great to have tests?", RecordKindIdea},
		{"TIL: something interesting", RecordKindLearning},
		{"Learned that this works", RecordKindLearning},
		{"Turns out it was simple", RecordKindLearning},
		{"Note: remember this", RecordKindLearning},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Classify(tt.input)
			if result.Kind != tt.wantKind {
				t.Errorf("Classify(%q) = %v, want %v (confidence: %v)",
					tt.input, result.Kind, tt.wantKind, result.Confidence)
			}
		})
	}
}

func TestClassifySignalsReturned(t *testing.T) {
	result := Classify("We should use PostgreSQL")
	if len(result.Signals) == 0 {
		t.Error("Expected signals to be populated for confident classification")
	}

	ambiguous := Classify("PostgreSQL is good")
	// Ambiguous might have empty signals
	if ambiguous.Confidence >= ConfidenceThreshold && len(ambiguous.Signals) == 0 {
		t.Error("High confidence should have signals")
	}
}

func TestClassifyAndStoreWithSession(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "classify-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Create a session
	session, _ := mem.StartSession("test", "agent", "goal")

	// Store with session
	id, kind, _, err := mem.ClassifyAndStore("We decided to use Go", "cli", session.ID)
	if err != nil {
		t.Fatalf("ClassifyAndStore failed: %v", err)
	}
	if kind != RecordKindDecision {
		t.Fatalf("Expected decision, got %v", kind)
	}

	// Verify session ID was stored
	dec, _ := mem.GetDecision(id)
	if dec.SessionID != session.ID {
		t.Errorf("Session ID not stored: expected %s, got %s", session.ID, dec.SessionID)
	}
}

func TestClassifyAndStoreExtractsTags(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "classify-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Store with tags
	id, kind, _, err := mem.ClassifyAndStore("We should use caching #performance #api", "cli", "")
	if err != nil {
		t.Fatalf("ClassifyAndStore failed: %v", err)
	}
	if kind != RecordKindDecision {
		t.Fatalf("Expected decision, got %v", kind)
	}

	// Verify tags were stored
	tags, _ := mem.GetTags(id, string(RecordKindDecision))
	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d: %v", len(tags), tags)
	}
}
