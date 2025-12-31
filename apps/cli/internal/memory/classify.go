package memory

import (
	"regexp"
	"strings"
)

// RecordKind represents the type of record being classified.
type RecordKind string

const (
	RecordKindIdea     RecordKind = "idea"
	RecordKindDecision RecordKind = "decision"
	RecordKindLearning RecordKind = "learning"
)

// Classification represents the result of classifying natural language input.
type Classification struct {
	Kind       RecordKind `json:"kind"`       // "decision", "idea", "learning"
	Confidence float64    `json:"confidence"` // 0.0-1.0
	Signals    []string   `json:"signals"`    // Which signals triggered this classification
}

// ConfidenceThreshold is the minimum confidence for auto-storing without user confirmation.
const ConfidenceThreshold = 0.7

// IntentSignals maps record kinds to phrases that indicate that kind.
var IntentSignals = map[RecordKind][]string{
	RecordKindDecision: {
		"let's", "we should", "we'll", "decided to", "going with",
		"chose", "choosing", "will use", "switching to", "agreed on",
		"the plan is", "we're going to", "final decision", "decided",
		"decision:", "we decided", "i decided", "going to use",
		"settled on", "picking", "selected", "opting for",
	},
	RecordKindIdea: {
		"what if", "maybe we", "could we", "how about", "idea:",
		"wondering if", "might be worth", "consider", "explore",
		"thought:", "perhaps", "possible to", "experiment with",
		"brainstorm", "imagine if", "potentially", "suggestion:",
		"proposal:", "concept:", "what about", "wouldn't it be",
	},
	RecordKindLearning: {
		"til", "learned that", "turns out", "realized", "discovered",
		"found out", "note:", "apparently", "insight:", "key takeaway",
		"important:", "remember that", "don't forget", "learning:",
		"lesson:", "figured out", "now i know", "it turns out",
		"fun fact", "did you know", "takeaway:", "gotcha:",
	},
}

// IntentPatterns are regex patterns for more complex signal matching.
var IntentPatterns = map[RecordKind][]*regexp.Regexp{
	RecordKindDecision: {
		regexp.MustCompile(`(?i)^we\s+(should|will|are going to)`),
		regexp.MustCompile(`(?i)^let'?s\s+`),
		regexp.MustCompile(`(?i)^i('m| am)\s+going\s+(to|with)`),
		regexp.MustCompile(`(?i)^(the\s+)?decision\s+is`),
	},
	RecordKindIdea: {
		regexp.MustCompile(`(?i)^what\s+if\s+`),
		regexp.MustCompile(`(?i)^how\s+about\s+`),
		regexp.MustCompile(`(?i)^maybe\s+we\s+(could|should)`),
		regexp.MustCompile(`(?i)^wouldn'?t\s+it\s+be`),
		regexp.MustCompile(`(?i)\?$`), // Questions are often ideas
	},
	RecordKindLearning: {
		regexp.MustCompile(`(?i)^til[:\s]`),
		regexp.MustCompile(`(?i)^(i\s+)?learned\s+that`),
		regexp.MustCompile(`(?i)^turns?\s+out`),
		regexp.MustCompile(`(?i)^note[:\s]`),
	},
}

// Classify analyzes text and returns the most likely record kind with confidence.
func Classify(text string) Classification {
	lower := strings.ToLower(strings.TrimSpace(text))

	var bestKind RecordKind
	var bestConfidence float64
	var matchedSignals []string

	// Check phrase signals
	for kind, signals := range IntentSignals {
		for _, signal := range signals {
			if strings.Contains(lower, signal) {
				confidence := calculatePhraseConfidence(signal, lower)
				if confidence > bestConfidence {
					bestKind = kind
					bestConfidence = confidence
					matchedSignals = []string{signal}
				} else if confidence == bestConfidence && kind == bestKind {
					matchedSignals = append(matchedSignals, signal)
				}
			}
		}
	}

	// Check regex patterns (can boost confidence)
	for kind, patterns := range IntentPatterns {
		for _, pattern := range patterns {
			if pattern.MatchString(lower) {
				patternConfidence := 0.85 // Patterns are more specific
				if kind == bestKind {
					// Boost confidence if pattern matches same kind
					bestConfidence = min(1.0, bestConfidence+0.1)
				} else if patternConfidence > bestConfidence {
					bestKind = kind
					bestConfidence = patternConfidence
					matchedSignals = []string{pattern.String()}
				}
			}
		}
	}

	// Default to idea with low confidence if no signals matched
	if bestKind == "" {
		return Classification{
			Kind:       RecordKindIdea,
			Confidence: 0.3,
			Signals:    nil,
		}
	}

	return Classification{
		Kind:       bestKind,
		Confidence: bestConfidence,
		Signals:    matchedSignals,
	}
}

// calculatePhraseConfidence returns confidence based on phrase position and specificity.
func calculatePhraseConfidence(signal, text string) float64 {
	// Higher confidence if signal appears at the start
	if strings.HasPrefix(text, signal) {
		return 0.85
	}

	// Higher confidence for longer, more specific signals
	signalLen := len(signal)
	if signalLen >= 10 {
		return 0.8
	}
	if signalLen >= 6 {
		return 0.75
	}

	return 0.7
}

// NeedsConfirmation returns true if the classification confidence is below threshold.
func (c Classification) NeedsConfirmation() bool {
	return c.Confidence < ConfidenceThreshold
}

// String returns a human-readable description of the classification.
func (c Classification) String() string {
	return string(c.Kind)
}

// ExtractTags attempts to extract tags from the text (hashtags or keywords).
func ExtractTags(text string) []string {
	var tags []string
	seen := make(map[string]bool)

	// Extract hashtags
	hashtagPattern := regexp.MustCompile(`#(\w+)`)
	matches := hashtagPattern.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) > 1 {
			tag := strings.ToLower(match[1])
			if !seen[tag] {
				tags = append(tags, tag)
				seen[tag] = true
			}
		}
	}

	return tags
}

// ClassifyAndStore classifies text and stores it as the appropriate record type.
// Returns the record ID, kind, classification, and any error.
func (m *Memory) ClassifyAndStore(text string, source string, sessionID string) (string, RecordKind, Classification, error) {
	classification := Classify(text)
	tags := ExtractTags(text)

	var id string
	var err error

	switch classification.Kind {
	case RecordKindIdea:
		idea := Idea{
			Content:   text,
			Source:    source,
			SessionID: sessionID,
		}
		id, err = m.AddIdea(idea)
		if err == nil && len(tags) > 0 {
			m.SetTags(id, string(RecordKindIdea), tags)
		}

	case RecordKindDecision:
		decision := Decision{
			Content:   text,
			Source:    source,
			SessionID: sessionID,
		}
		id, err = m.AddDecision(decision)
		if err == nil && len(tags) > 0 {
			m.SetTags(id, string(RecordKindDecision), tags)
		}

	case RecordKindLearning:
		learning := Learning{
			Content:   text,
			Source:    source,
			SessionID: sessionID,
		}
		id, err = m.AddLearning(learning)
		if err == nil && len(tags) > 0 {
			m.SetTags(id, string(RecordKindLearning), tags)
		}
	}

	return id, classification.Kind, classification, err
}

// min returns the minimum of two float64 values.
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
