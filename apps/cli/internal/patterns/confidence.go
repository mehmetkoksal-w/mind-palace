package patterns

import (
	"math"
	"time"
)

// ConfidenceThresholds defines the thresholds for confidence levels.
var ConfidenceThresholds = struct {
	High     float64
	Medium   float64
	Low      float64
	BulkApprove float64
}{
	High:        0.85,
	Medium:      0.70,
	Low:         0.50,
	BulkApprove: 0.95, // Default threshold for bulk approval
}

// CalculateFrequencyScore calculates the frequency score based on occurrence count.
// Uses a logarithmic scale to prevent single high-frequency patterns from dominating.
func CalculateFrequencyScore(occurrences, totalFiles int) float64 {
	if occurrences == 0 || totalFiles == 0 {
		return 0.0
	}

	// Ratio of occurrences to total files
	ratio := float64(occurrences) / float64(totalFiles)

	// Use logarithmic scaling: more occurrences = higher score, but diminishing returns
	// Score approaches 1.0 as occurrences increase
	// At ratio=0.5 (50% of files), score is ~0.85
	// At ratio=1.0 (100% of files), score is ~1.0
	score := math.Log10(1+ratio*9) // log10(1) = 0, log10(10) = 1

	return clamp(score, 0.0, 1.0)
}

// CalculateConsistencyScore calculates how consistent implementations are.
// Takes a variance measure (0 = identical, 1 = completely different).
func CalculateConsistencyScore(variance float64) float64 {
	// Invert variance: low variance = high consistency
	return clamp(1.0-variance, 0.0, 1.0)
}

// CalculateSpreadScore calculates how widely the pattern is spread across files.
func CalculateSpreadScore(filesWithPattern, totalFiles int) float64 {
	if filesWithPattern == 0 || totalFiles == 0 {
		return 0.0
	}

	// Simple ratio with slight logarithmic scaling for larger codebases
	ratio := float64(filesWithPattern) / float64(totalFiles)

	// Boost spread for patterns in multiple files
	// Single file = 0, 2 files = higher, etc.
	if filesWithPattern == 1 {
		return ratio * 0.5 // Penalize single-file patterns
	}

	// Logarithmic scaling to reward spread in larger codebases
	spreadBoost := math.Log10(float64(filesWithPattern)) / math.Log10(float64(max(totalFiles, 10)))
	score := (ratio + spreadBoost) / 2

	return clamp(score, 0.0, 1.0)
}

// CalculateAgeScore calculates the age score based on how long the pattern has existed.
// Older patterns are considered more established.
func CalculateAgeScore(firstSeen, now time.Time) float64 {
	if firstSeen.IsZero() || now.Before(firstSeen) {
		return 0.0
	}

	age := now.Sub(firstSeen)

	// Age thresholds:
	// < 1 day: low score (new pattern)
	// 1-7 days: medium score
	// 7-30 days: high score
	// > 30 days: maximum score

	daysSinceFirst := age.Hours() / 24

	switch {
	case daysSinceFirst < 1:
		return 0.2
	case daysSinceFirst < 7:
		return 0.2 + (daysSinceFirst/7)*0.3 // 0.2 to 0.5
	case daysSinceFirst < 30:
		return 0.5 + ((daysSinceFirst-7)/23)*0.3 // 0.5 to 0.8
	default:
		// Asymptotically approach 1.0
		return clamp(0.8+0.2*(1-math.Exp(-daysSinceFirst/90)), 0.0, 1.0)
	}
}

// CalculateConfidence calculates the overall confidence from factors.
func CalculateConfidence(factors ConfidenceFactors) float64 {
	return factors.Score()
}

// NewConfidenceFactors creates ConfidenceFactors from raw data.
func NewConfidenceFactors(
	occurrences, totalFiles, filesWithPattern int,
	variance float64,
	firstSeen, now time.Time,
) ConfidenceFactors {
	return ConfidenceFactors{
		Frequency:   CalculateFrequencyScore(occurrences, totalFiles),
		Consistency: CalculateConsistencyScore(variance),
		Spread:      CalculateSpreadScore(filesWithPattern, totalFiles),
		Age:         CalculateAgeScore(firstSeen, now),
	}
}

// AdjustConfidenceForOutliers reduces confidence based on outlier ratio.
func AdjustConfidenceForOutliers(baseConfidence float64, matches, outliers int) float64 {
	if matches == 0 {
		return 0.0
	}

	total := matches + outliers
	matchRatio := float64(matches) / float64(total)

	// Reduce confidence proportionally to outlier count
	// If 10% outliers, reduce confidence by ~5%
	// If 50% outliers, reduce confidence significantly
	adjustment := math.Pow(matchRatio, 0.5) // Square root for softer penalty

	return clamp(baseConfidence*adjustment, 0.0, 1.0)
}

// clamp restricts a value to a range.
//
func clamp(value, minVal, maxVal float64) float64 {
	if value < minVal {
		return minVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}

