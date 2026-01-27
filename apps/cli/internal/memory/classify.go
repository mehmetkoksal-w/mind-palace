package memory

import (
	"regexp"
	"strings"
)

// RecordKind represents the type of record being classified.
type RecordKind string

const (
	// RecordKindIdea represents an idea record.
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
		// Explicit decision markers
		"let's", "we should", "we'll", "decided to", "going with",
		"chose", "choosing", "will use", "switching to", "agreed on",
		"the plan is", "we're going to", "final decision", "decided",
		"decision:", "we decided", "i decided", "going to use",
		"settled on", "picking", "selected", "opting for",
		// Implicit decision patterns - statements about using technologies
		"use ", "using ", "implement ", "add ", "create ", "build ",
		"adopt ", "integrate ", "configure ", "set up ", "enable ",
		// Architectural decisions often use "with" for specifications
		"with ", "for ", "via ", "through ",
	},
	RecordKindIdea: {
		"what if", "maybe we", "could we", "how about", "idea:",
		"wondering if", "might be worth", "consider", "explore",
		"thought:", "perhaps", "possible to", "experiment with",
		"brainstorm", "imagine if", "potentially", "suggestion:",
		"proposal:", "concept:", "what about", "wouldn't it be",
		"try ", "test ", "investigate ", "research ",
	},
	RecordKindLearning: {
		"til", "learned that", "turns out", "realized", "discovered",
		"found out", "note:", "apparently", "insight:", "key takeaway",
		"important:", "remember that", "don't forget", "learning:",
		"lesson:", "figured out", "now i know", "it turns out",
		"fun fact", "did you know", "takeaway:", "gotcha:",
		"always ", "never ", "must ", "should always", "should never",
	},
}

// IntentPatterns are regex patterns for more complex signal matching.
var IntentPatterns = map[RecordKind][]*regexp.Regexp{
	RecordKindDecision: {
		regexp.MustCompile(`(?i)^we\s+(should|will|are going to)`),
		regexp.MustCompile(`(?i)^let'?s\s+`),
		regexp.MustCompile(`(?i)^i('m| am)\s+going\s+(to|with)`),
		regexp.MustCompile(`(?i)^(the\s+)?decision\s+is`),
		regexp.MustCompile(`(?i)^(the\s+)?plan\s+is\s+to`), // "The plan is to..."
		// Technology/architectural decisions
		regexp.MustCompile(`(?i)^use\s+\w+`),                       // "Use JWT", "Use Redis"
		regexp.MustCompile(`(?i)^implement\s+`),                    // "Implement caching"
		regexp.MustCompile(`(?i)^add\s+\w+\s+(to|for|with)`),       // "Add logging to..."
		regexp.MustCompile(`(?i)^(adopt|integrate|enable)\s+`),     // "Adopt GraphQL"
		regexp.MustCompile(`(?i)\bwith\s+\d+\s*(hour|minute|day)`), // "with 24h expiry"
		regexp.MustCompile(`(?i)\bfor\s+(auth|security|performance|caching)`),
		regexp.MustCompile(`(?i)^switch(ing)?\s+(to|from)\s+`), // "Switch to TypeScript", "Switching to..."
	},
	RecordKindIdea: {
		regexp.MustCompile(`(?i)^what\s+if\s+`),
		regexp.MustCompile(`(?i)^how\s+about\s+`),
		regexp.MustCompile(`(?i)^maybe\s+we\s+(could|should)`),
		regexp.MustCompile(`(?i)^wouldn'?t\s+it\s+be`),
		regexp.MustCompile(`(?i)\?$`), // Questions are often ideas
		regexp.MustCompile(`(?i)^(try|test|investigate)\s+`),
		regexp.MustCompile(`(?i)^(consider|explore)\s+(using|implementing|adding)`), // "Consider using...", "Explore using..."
	},
	RecordKindLearning: {
		regexp.MustCompile(`(?i)^til[:\s]`),
		regexp.MustCompile(`(?i)^(i\s+)?learned\s+that`),
		regexp.MustCompile(`(?i)^turns?\s+out`),
		regexp.MustCompile(`(?i)^note[:\s]`),
		regexp.MustCompile(`(?i)^always\s+`),                                             // "Always use..."
		regexp.MustCompile(`(?i)^never\s+`),                                              // "Never do..."
		regexp.MustCompile(`(?i)^(i\s+)?(realized|discovered|found out|figured out)\s+`), // Past-tense discovery
		regexp.MustCompile(`(?i)\s+because\s+.{20,}`),                                    // Explanations are often learnings
	},
}

// Classify analyzes text and returns the most likely record kind with confidence.
func Classify(text string) Classification {
	lower := strings.ToLower(strings.TrimSpace(text))

	// Check for explicit prefixes first - these override all other signals
	explicitPrefixes := map[RecordKind][]string{
		RecordKindDecision: {"decision:", "decided:"},
		RecordKindIdea:     {"idea:", "thought:", "suggestion:", "concept:", "proposal:"},
		RecordKindLearning: {"til:", "note:", "learning:", "lesson:", "insight:", "gotcha:", "takeaway:"},
	}
	for kind, prefixes := range explicitPrefixes {
		for _, prefix := range prefixes {
			if strings.HasPrefix(lower, prefix) {
				return Classification{
					Kind:       kind,
					Confidence: 0.95,
					Signals:    []string{prefix + " (explicit)"},
				}
			}
		}
	}

	// Track scores for each kind (allows accumulating evidence)
	scores := map[RecordKind]float64{
		RecordKindDecision: 0,
		RecordKindIdea:     0,
		RecordKindLearning: 0,
	}
	signals := map[RecordKind][]string{
		RecordKindDecision: {},
		RecordKindIdea:     {},
		RecordKindLearning: {},
	}

	// Check phrase signals
	for kind, phraseSignals := range IntentSignals {
		for _, signal := range phraseSignals {
			if strings.Contains(lower, signal) {
				weight := calculatePhraseWeight(signal, lower)
				scores[kind] += weight
				signals[kind] = append(signals[kind], signal)
			}
		}
	}

	// Check regex patterns (more weight than simple phrases)
	for kind, patterns := range IntentPatterns {
		for _, pattern := range patterns {
			if pattern.MatchString(lower) {
				// Patterns that match at the start of text get higher weight
				match := pattern.FindStringIndex(lower)
				if match != nil && match[0] == 0 {
					scores[kind] += 0.5 // Start-of-text patterns are very strong signals
				} else {
					scores[kind] += 0.3 // Non-start patterns contribute less
				}
				signals[kind] = append(signals[kind], "pattern")
			}
		}
	}

	// Find best kind by score
	var bestKind RecordKind
	var bestScore float64
	for kind, score := range scores {
		if score > bestScore {
			bestKind = kind
			bestScore = score
		}
	}

	// Convert score to confidence (cap at 0.95)
	confidence := min(0.95, bestScore)

	// Default to idea with low confidence if no signals matched
	if bestKind == "" || bestScore < 0.1 {
		return Classification{
			Kind:       RecordKindIdea,
			Confidence: 0.3,
			Signals:    nil,
		}
	}

	return Classification{
		Kind:       bestKind,
		Confidence: confidence,
		Signals:    signals[bestKind],
	}
}

// calculatePhraseWeight returns a weight based on phrase position and specificity.
func calculatePhraseWeight(signal, text string) float64 {
	weight := 0.0

	// Higher weight if signal appears at the start
	if strings.HasPrefix(text, signal) {
		weight += 0.35
	} else {
		weight += 0.20 // Non-start signals still contribute meaningfully
	}

	// Higher weight for longer, more specific signals
	signalLen := len(signal)
	if signalLen >= 10 {
		weight += 0.30
	} else if signalLen >= 6 {
		weight += 0.20
	} else {
		weight += 0.10 // Short signals like "use " get less weight
	}

	return weight
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
func (m *Memory) ClassifyAndStore(text, source, sessionID string) (string, RecordKind, Classification, error) {
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
			_ = m.SetTags(id, string(RecordKindIdea), tags)
		}

	case RecordKindDecision:
		decision := Decision{
			Content:   text,
			Source:    source,
			SessionID: sessionID,
		}
		id, err = m.AddDecision(decision)
		if err == nil && len(tags) > 0 {
			_ = m.SetTags(id, string(RecordKindDecision), tags)
		}

	case RecordKindLearning:
		learning := Learning{
			Content:   text,
			Source:    source,
			SessionID: sessionID,
		}
		id, err = m.AddLearning(learning)
		if err == nil && len(tags) > 0 {
			_ = m.SetTags(id, string(RecordKindLearning), tags)
		}
	}

	return id, classification.Kind, classification, err
}
