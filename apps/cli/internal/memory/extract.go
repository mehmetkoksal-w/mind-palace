package memory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/llm"
)

// Extractor defines the interface for extracting records from conversations.
type Extractor interface {
	// ExtractFromConversation analyzes a conversation and extracts ideas, decisions, and learnings.
	// Returns extracted record IDs after storing them.
	ExtractFromConversation(conv Conversation) ([]string, error)
}

// ExtractedRecord represents a record extracted from a conversation.
type ExtractedRecord struct {
	Kind    RecordKind `json:"kind"`    // "idea", "decision", "learning"
	Content string     `json:"content"` // Extracted content
	Context string     `json:"context"` // Why this was extracted
}

// ExtractionResult holds the structured output from LLM extraction.
type ExtractionResult struct {
	Ideas     []ExtractedRecord `json:"ideas"`
	Decisions []ExtractedRecord `json:"decisions"`
	Learnings []ExtractedRecord `json:"learnings"`
}

// ============================================================================
// LLM-Based Extractor
// ============================================================================

// LLMExtractor uses an LLM to extract records from conversations.
type LLMExtractor struct {
	llm    llm.Client
	memory *Memory
}

// NewLLMExtractor creates a new LLM-based extractor.
func NewLLMExtractor(client llm.Client, mem *Memory) *LLMExtractor {
	return &LLMExtractor{
		llm:    client,
		memory: mem,
	}
}

// extractionPrompt is the prompt template for extracting records.
const extractionPrompt = `Analyze this conversation between a user and an AI assistant. Extract any notable:

1. DECISIONS: Commitments, choices, or agreements made (e.g., "let's use X", "we decided to Y", "going with Z")
2. IDEAS: Proposals, questions for exploration, or creative suggestions (e.g., "what if we", "could we try", "idea:")
3. LEARNINGS: Insights, discoveries, or realizations (e.g., "turns out", "TIL", "realized that", "the issue was")

For each extracted item, provide the content and brief context explaining why it was extracted.

IMPORTANT: Only extract items that are substantive and would be valuable to remember. Skip trivial greetings or acknowledgments.

Return a JSON object with this exact structure:
{
  "ideas": [{"content": "the idea", "context": "why extracted"}],
  "decisions": [{"content": "the decision", "context": "why extracted"}],
  "learnings": [{"content": "the learning", "context": "why extracted"}]
}

If no items are found for a category, use an empty array [].

CONVERSATION:
%s`

// ExtractFromConversation analyzes a conversation using the LLM and stores extracted records.
func (e *LLMExtractor) ExtractFromConversation(conv Conversation) ([]string, error) {
	if len(conv.Messages) == 0 {
		return []string{}, nil
	}

	// Format conversation for the prompt
	var convText strings.Builder
	for _, msg := range conv.Messages {
		role := strings.ToUpper(msg.Role)
		convText.WriteString(fmt.Sprintf("[%s]: %s\n\n", role, msg.Content))
	}

	prompt := fmt.Sprintf(extractionPrompt, convText.String())

	// Call LLM with JSON mode
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	opts := llm.DefaultCompletionOptions()
	opts.SystemPrompt = "You are a knowledge extraction assistant. Your job is to identify and extract valuable insights, decisions, and ideas from conversations. Always respond with valid JSON."

	var result ExtractionResult
	if err := e.llm.CompleteJSON(ctx, prompt, opts, &result); err != nil {
		return nil, fmt.Errorf("LLM extraction failed: %w", err)
	}

	// Store extracted records
	var recordIDs []string

	// Store ideas
	for _, item := range result.Ideas {
		if item.Content == "" {
			continue
		}
		idea := Idea{
			Content:   item.Content,
			Context:   item.Context,
			Status:    IdeaStatusActive,
			Scope:     "palace",
			SessionID: conv.SessionID,
			Source:    "auto-extract",
		}
		id, err := e.memory.AddIdea(idea)
		if err == nil {
			recordIDs = append(recordIDs, id)
		}
	}

	// Store decisions
	for _, item := range result.Decisions {
		if item.Content == "" {
			continue
		}
		dec := Decision{
			Content:   item.Content,
			Context:   item.Context,
			Status:    DecisionStatusActive,
			Outcome:   DecisionOutcomeUnknown,
			Scope:     "palace",
			SessionID: conv.SessionID,
			Source:    "auto-extract",
		}
		id, err := e.memory.AddDecision(dec)
		if err == nil {
			recordIDs = append(recordIDs, id)
		}
	}

	// Store learnings
	for _, item := range result.Learnings {
		if item.Content == "" {
			continue
		}
		learning := Learning{
			Content:    item.Content,
			Scope:      "palace",
			SessionID:  conv.SessionID,
			Source:     "auto-extract",
			Confidence: 0.5, // Start at neutral confidence
		}
		id, err := e.memory.AddLearning(learning)
		if err == nil {
			recordIDs = append(recordIDs, id)
		}
	}

	// Update conversation with extracted record IDs
	if len(recordIDs) > 0 && conv.ID != "" {
		_ = e.memory.UpdateConversationExtracted(conv.ID, recordIDs)
	}

	return recordIDs, nil
}

// ============================================================================
// Stub Extractor (fallback when LLM not configured)
// ============================================================================

// StubExtractor is a no-op extractor that doesn't extract anything.
// Used when LLM is not configured.
type StubExtractor struct{}

// ExtractFromConversation is a stub that returns empty results.
func (e *StubExtractor) ExtractFromConversation(conv Conversation) ([]string, error) {
	return []string{}, nil
}

// NewStubExtractor creates a new stub extractor.
func NewStubExtractor() Extractor {
	return &StubExtractor{}
}

// ============================================================================
// Helper Functions
// ============================================================================

// ExtractRecords analyzes messages and returns potential records to extract.
// This uses keyword-based classification as a fallback.
func ExtractRecords(messages []Message) []ExtractedRecord {
	var records []ExtractedRecord

	for _, msg := range messages {
		if msg.Role != "assistant" {
			continue
		}

		// Use keyword-based classification
		classification := Classify(msg.Content)
		if classification.Confidence >= ConfidenceThreshold {
			records = append(records, ExtractedRecord{
				Kind:    classification.Kind,
				Content: msg.Content,
				Context: fmt.Sprintf("Detected via signals: %v", classification.Signals),
			})
		}
	}

	return records
}

// AutoExtractConfig holds configuration for automatic extraction.
type AutoExtractConfig struct {
	Enabled       bool   `json:"enabled"`       // Whether auto-extraction is enabled
	MinMessages   int    `json:"minMessages"`   // Minimum messages before extraction
	ExtractorType string `json:"extractorType"` // "stub", "llm"
}

// DefaultAutoExtractConfig returns the default auto-extraction configuration.
func DefaultAutoExtractConfig() AutoExtractConfig {
	return AutoExtractConfig{
		Enabled:       false, // Disabled by default
		MinMessages:   5,     // Require at least 5 messages
		ExtractorType: "stub",
	}
}
