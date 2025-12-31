package memory

import (
	"fmt"
	"time"
)

// DecayConfig holds configuration for confidence decay.
type DecayConfig struct {
	Enabled       bool    `json:"enabled"`
	DecayDays     int     `json:"decayDays"`     // Days before decay starts (default: 30)
	DecayRate     float64 `json:"decayRate"`     // Decay amount per period (default: 0.05)
	DecayInterval int     `json:"decayInterval"` // Days between decay applications (default: 7)
	MinConfidence float64 `json:"minConfidence"` // Minimum confidence floor (default: 0.1)
}

// DefaultDecayConfig returns the default decay configuration.
func DefaultDecayConfig() DecayConfig {
	return DecayConfig{
		Enabled:       false,
		DecayDays:     30,
		DecayRate:     0.05,
		DecayInterval: 7,
		MinConfidence: 0.1,
	}
}

// DecayResult contains the results of applying decay.
type DecayResult struct {
	TotalAffected   int                   `json:"totalAffected"`
	TotalDecayed    int                   `json:"totalDecayed"`
	AverageDecay    float64               `json:"averageDecay"`
	DecayedRecords  []DecayedRecordInfo   `json:"decayedRecords,omitempty"`
	AtRiskRecords   []AtRiskRecordInfo    `json:"atRiskRecords,omitempty"`
}

// DecayedRecordInfo contains info about a decayed record.
type DecayedRecordInfo struct {
	ID              string  `json:"id"`
	Content         string  `json:"content"`
	OldConfidence   float64 `json:"oldConfidence"`
	NewConfidence   float64 `json:"newConfidence"`
	DaysSinceAccess int     `json:"daysSinceAccess"`
}

// AtRiskRecordInfo contains info about a record at risk of decay.
type AtRiskRecordInfo struct {
	ID              string  `json:"id"`
	Content         string  `json:"content"`
	Confidence      float64 `json:"confidence"`
	DaysSinceAccess int     `json:"daysSinceAccess"`
	DaysUntilDecay  int     `json:"daysUntilDecay"`
}

// DecayStats contains statistics about decay state.
type DecayStats struct {
	TotalLearnings      int     `json:"totalLearnings"`
	AtRiskCount         int     `json:"atRiskCount"`
	DecayedCount        int     `json:"decayedCount"`
	AverageConfidence   float64 `json:"averageConfidence"`
	OldestInactivedays  int     `json:"oldestInactiveDays"`
	NextDecayEligible   int     `json:"nextDecayEligible"`
}

// GetDecayStats returns statistics about decay state for learnings.
func (m *Memory) GetDecayStats(cfg DecayConfig) (*DecayStats, error) {
	stats := &DecayStats{}

	// Count total learnings
	err := m.db.QueryRow(`SELECT COUNT(*) FROM learnings WHERE status = 'active'`).Scan(&stats.TotalLearnings)
	if err != nil {
		return nil, fmt.Errorf("count learnings: %w", err)
	}

	if stats.TotalLearnings == 0 {
		return stats, nil
	}

	// Average confidence
	err = m.db.QueryRow(`SELECT COALESCE(AVG(confidence), 0) FROM learnings WHERE status = 'active'`).Scan(&stats.AverageConfidence)
	if err != nil {
		return nil, fmt.Errorf("avg confidence: %w", err)
	}

	// Find oldest inactive learning
	now := time.Now()
	cutoffDate := now.AddDate(0, 0, -cfg.DecayDays)

	var oldestAccess time.Time
	err = m.db.QueryRow(`
		SELECT COALESCE(MIN(last_used), created_at)
		FROM learnings
		WHERE status = 'active'
	`).Scan(&oldestAccess)
	if err == nil && !oldestAccess.IsZero() {
		stats.OldestInactivedays = int(now.Sub(oldestAccess).Hours() / 24)
	}

	// Count at-risk (not yet decayed but past threshold)
	err = m.db.QueryRow(`
		SELECT COUNT(*) FROM learnings
		WHERE status = 'active'
		AND COALESCE(last_used, created_at) < ?
		AND confidence > ?
	`, cutoffDate, cfg.MinConfidence).Scan(&stats.AtRiskCount)
	if err != nil {
		return nil, fmt.Errorf("at risk count: %w", err)
	}

	// Count already decayed (below initial confidence)
	err = m.db.QueryRow(`
		SELECT COUNT(*) FROM learnings
		WHERE status = 'active'
		AND confidence <= ?
	`, cfg.MinConfidence+0.01).Scan(&stats.DecayedCount)
	if err != nil {
		return nil, fmt.Errorf("decayed count: %w", err)
	}

	// Count eligible for next decay
	err = m.db.QueryRow(`
		SELECT COUNT(*) FROM learnings
		WHERE status = 'active'
		AND COALESCE(last_used, created_at) < ?
		AND confidence > ?
	`, cutoffDate, cfg.MinConfidence).Scan(&stats.NextDecayEligible)
	if err != nil {
		return nil, fmt.Errorf("next decay eligible: %w", err)
	}

	return stats, nil
}

// PreviewDecay returns what would be affected by applying decay without actually applying it.
func (m *Memory) PreviewDecay(cfg DecayConfig, limit int) (*DecayResult, error) {
	if limit <= 0 {
		limit = 50
	}

	result := &DecayResult{}

	now := time.Now()
	cutoffDate := now.AddDate(0, 0, -cfg.DecayDays)

	// Find learnings that would be decayed
	rows, err := m.db.Query(`
		SELECT id, content, confidence, COALESCE(last_used, created_at) as last_access
		FROM learnings
		WHERE status = 'active'
		AND COALESCE(last_used, created_at) < ?
		AND confidence > ?
		ORDER BY COALESCE(last_used, created_at) ASC
		LIMIT ?
	`, cutoffDate, cfg.MinConfidence, limit)
	if err != nil {
		return nil, fmt.Errorf("query decay candidates: %w", err)
	}
	defer rows.Close()

	var totalDecay float64
	for rows.Next() {
		var id, content string
		var confidence float64
		var lastAccess time.Time

		if err := rows.Scan(&id, &content, &confidence, &lastAccess); err != nil {
			continue
		}

		daysSinceAccess := int(now.Sub(lastAccess).Hours() / 24)
		periodsInactive := (daysSinceAccess - cfg.DecayDays) / cfg.DecayInterval
		if periodsInactive < 1 {
			periodsInactive = 1
		}

		// Calculate decay amount
		decayAmount := float64(periodsInactive) * cfg.DecayRate
		newConfidence := confidence - decayAmount
		if newConfidence < cfg.MinConfidence {
			newConfidence = cfg.MinConfidence
		}

		result.DecayedRecords = append(result.DecayedRecords, DecayedRecordInfo{
			ID:              id,
			Content:         truncateForDisplay(content, 100),
			OldConfidence:   confidence,
			NewConfidence:   newConfidence,
			DaysSinceAccess: daysSinceAccess,
		})

		totalDecay += confidence - newConfidence
		result.TotalAffected++
	}

	if result.TotalAffected > 0 {
		result.AverageDecay = totalDecay / float64(result.TotalAffected)
	}

	// Find at-risk records (approaching decay threshold)
	approachingCutoff := now.AddDate(0, 0, -(cfg.DecayDays - 7)) // Within 7 days of decay
	atRiskRows, err := m.db.Query(`
		SELECT id, content, confidence, COALESCE(last_used, created_at) as last_access
		FROM learnings
		WHERE status = 'active'
		AND COALESCE(last_used, created_at) < ?
		AND COALESCE(last_used, created_at) >= ?
		AND confidence > ?
		ORDER BY COALESCE(last_used, created_at) ASC
		LIMIT ?
	`, approachingCutoff, cutoffDate, cfg.MinConfidence, limit)
	if err == nil {
		defer atRiskRows.Close()
		for atRiskRows.Next() {
			var id, content string
			var confidence float64
			var lastAccess time.Time

			if err := atRiskRows.Scan(&id, &content, &confidence, &lastAccess); err != nil {
				continue
			}

			daysSinceAccess := int(now.Sub(lastAccess).Hours() / 24)
			daysUntilDecay := cfg.DecayDays - daysSinceAccess
			if daysUntilDecay < 0 {
				daysUntilDecay = 0
			}

			result.AtRiskRecords = append(result.AtRiskRecords, AtRiskRecordInfo{
				ID:              id,
				Content:         truncateForDisplay(content, 100),
				Confidence:      confidence,
				DaysSinceAccess: daysSinceAccess,
				DaysUntilDecay:  daysUntilDecay,
			})
		}
	}

	return result, nil
}

// ApplyDecay applies confidence decay to inactive learnings.
func (m *Memory) ApplyDecay(cfg DecayConfig) (*DecayResult, error) {
	if !cfg.Enabled {
		return &DecayResult{}, nil
	}

	result := &DecayResult{}

	now := time.Now()
	cutoffDate := now.AddDate(0, 0, -cfg.DecayDays)

	// Find learnings to decay
	rows, err := m.db.Query(`
		SELECT id, content, confidence, COALESCE(last_used, created_at) as last_access
		FROM learnings
		WHERE status = 'active'
		AND COALESCE(last_used, created_at) < ?
		AND confidence > ?
	`, cutoffDate, cfg.MinConfidence)
	if err != nil {
		return nil, fmt.Errorf("query decay candidates: %w", err)
	}
	defer rows.Close()

	var toUpdate []struct {
		id            string
		oldConfidence float64
		newConfidence float64
	}

	var totalDecay float64
	for rows.Next() {
		var id, content string
		var confidence float64
		var lastAccess time.Time

		if err := rows.Scan(&id, &content, &confidence, &lastAccess); err != nil {
			continue
		}

		daysSinceAccess := int(now.Sub(lastAccess).Hours() / 24)
		periodsInactive := (daysSinceAccess - cfg.DecayDays) / cfg.DecayInterval
		if periodsInactive < 1 {
			periodsInactive = 1
		}

		// Calculate decay amount
		decayAmount := float64(periodsInactive) * cfg.DecayRate
		newConfidence := confidence - decayAmount
		if newConfidence < cfg.MinConfidence {
			newConfidence = cfg.MinConfidence
		}

		// Skip if no actual change
		if newConfidence >= confidence {
			continue
		}

		toUpdate = append(toUpdate, struct {
			id            string
			oldConfidence float64
			newConfidence float64
		}{id, confidence, newConfidence})

		result.DecayedRecords = append(result.DecayedRecords, DecayedRecordInfo{
			ID:              id,
			Content:         truncateForDisplay(content, 100),
			OldConfidence:   confidence,
			NewConfidence:   newConfidence,
			DaysSinceAccess: daysSinceAccess,
		})

		totalDecay += confidence - newConfidence
		result.TotalAffected++
	}

	// Apply updates in a transaction
	if len(toUpdate) > 0 {
		tx, err := m.db.Begin()
		if err != nil {
			return nil, fmt.Errorf("begin transaction: %w", err)
		}

		stmt, err := tx.Prepare(`UPDATE learnings SET confidence = ?, updated_at = ? WHERE id = ?`)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("prepare statement: %w", err)
		}
		defer stmt.Close()

		for _, u := range toUpdate {
			_, err := stmt.Exec(u.newConfidence, now, u.id)
			if err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("update learning %s: %w", u.id, err)
			}
			result.TotalDecayed++
		}

		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit transaction: %w", err)
		}
	}

	if result.TotalAffected > 0 {
		result.AverageDecay = totalDecay / float64(result.TotalAffected)
	}

	return result, nil
}

// BoostConfidence increases confidence for a learning (opposite of decay).
func (m *Memory) BoostConfidence(id string, boost float64, maxConfidence float64) error {
	if maxConfidence <= 0 {
		maxConfidence = 1.0
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := m.db.Exec(`
		UPDATE learnings
		SET confidence = MIN(confidence + ?, ?),
		    last_used = ?,
		    use_count = COALESCE(use_count, 0) + 1
		WHERE id = ?
	`, boost, maxConfidence, now, id)
	return err
}

// truncateForDisplay truncates a string for display purposes.
func truncateForDisplay(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
