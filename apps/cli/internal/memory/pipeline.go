package memory

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// EmbeddingPipeline handles background embedding generation for records.
type EmbeddingPipeline struct {
	memory   *Memory
	embedder Embedder
	queue    chan embeddingJob
	workers  int
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	running  bool
	mu       sync.Mutex
}

// embeddingJob represents a single embedding task.
type embeddingJob struct {
	RecordID   string
	RecordKind string
	Content    string
}

// NewEmbeddingPipeline creates a new embedding pipeline.
// workers specifies how many concurrent embedding operations to run.
func NewEmbeddingPipeline(mem *Memory, embedder Embedder, workers int) *EmbeddingPipeline {
	if workers < 1 {
		workers = 1
	}
	if workers > 4 {
		workers = 4 // Cap at 4 to avoid overwhelming the embedding API
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &EmbeddingPipeline{
		memory:   mem,
		embedder: embedder,
		queue:    make(chan embeddingJob, 100), // Buffer up to 100 jobs
		workers:  workers,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins the background workers.
func (p *EmbeddingPipeline) Start() {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.running = true
	p.mu.Unlock()

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Stop gracefully shuts down the pipeline.
func (p *EmbeddingPipeline) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	p.mu.Unlock()

	p.cancel()
	close(p.queue)
	p.wg.Wait()
}

// Enqueue adds a record to the embedding queue.
// This is non-blocking; if the queue is full, the job is dropped.
func (p *EmbeddingPipeline) Enqueue(recordID, kind, content string) {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	job := embeddingJob{
		RecordID:   recordID,
		RecordKind: kind,
		Content:    content,
	}

	select {
	case p.queue <- job:
		// Successfully queued
	default:
		// Queue full, drop the job (can be processed later via ProcessPending)
	}
}

// worker processes embedding jobs from the queue.
func (p *EmbeddingPipeline) worker(_ int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case job, ok := <-p.queue:
			if !ok {
				return
			}
			p.processJob(job)
		}
	}
}

// processJob generates and stores an embedding for a single job.
func (p *EmbeddingPipeline) processJob(job embeddingJob) {
	if job.Content == "" {
		return
	}

	// Generate embedding
	embedding, err := p.embedder.Embed(job.Content)
	if err != nil {
		// Log error but don't fail - embedding is optional
		log.Printf("embedding failed for %s: %v", job.RecordID, err)
		return
	}

	// Store embedding
	if err := p.memory.StoreEmbedding(job.RecordID, job.RecordKind, embedding, p.embedder.Model()); err != nil {
		log.Printf("store embedding failed for %s: %v", job.RecordID, err)
	}
}

// ProcessPending generates embeddings for records that don't have them.
// This is useful for backfilling embeddings after enabling the feature.
func (p *EmbeddingPipeline) ProcessPending(kinds []string, limit int) (int, error) {
	if limit <= 0 {
		limit = 100
	}

	if len(kinds) == 0 {
		kinds = []string{"idea", "decision", "learning"}
	}

	processed := 0

	for _, kind := range kinds {
		records, err := p.memory.GetRecordsWithoutEmbeddings(kind, limit-processed)
		if err != nil {
			return processed, err
		}

		for _, record := range records {
			if processed >= limit {
				break
			}

			select {
			case <-p.ctx.Done():
				return processed, p.ctx.Err()
			default:
			}

			embedding, err := p.embedder.Embed(record.Content)
			if err != nil {
				log.Printf("embedding failed for %s: %v", record.ID, err)
				continue
			}

			if err := p.memory.StoreEmbedding(record.ID, kind, embedding, p.embedder.Model()); err != nil {
				log.Printf("store embedding failed for %s: %v", record.ID, err)
				continue
			}

			processed++
		}
	}

	return processed, nil
}

// QueueSize returns the current number of pending jobs.
func (p *EmbeddingPipeline) QueueSize() int {
	return len(p.queue)
}

// IsRunning returns whether the pipeline is currently running.
func (p *EmbeddingPipeline) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// RecordWithContent is a minimal record type for pending embeddings.
type RecordWithContent struct {
	ID      string
	Content string
}

// GetRecordsWithoutEmbeddings returns records that don't have embeddings.
func (m *Memory) GetRecordsWithoutEmbeddings(kind string, limit int) ([]RecordWithContent, error) {
	var query string
	switch kind {
	case "idea":
		query = `
			SELECT i.id, i.content FROM ideas i
			LEFT JOIN embeddings e ON i.id = e.record_id
			WHERE e.record_id IS NULL
			LIMIT ?`
	case "decision":
		query = `
			SELECT d.id, d.content FROM decisions d
			LEFT JOIN embeddings e ON d.id = e.record_id
			WHERE e.record_id IS NULL
			LIMIT ?`
	case "learning":
		query = `
			SELECT l.id, l.content FROM learnings l
			LEFT JOIN embeddings e ON l.id = e.record_id
			WHERE e.record_id IS NULL
			LIMIT ?`
	default:
		return nil, nil
	}

	rows, err := m.db.QueryContext(context.Background(), query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []RecordWithContent
	for rows.Next() {
		var r RecordWithContent
		if err := rows.Scan(&r.ID, &r.Content); err != nil {
			continue
		}
		records = append(records, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate records: %w", err)
	}

	return records, nil
}

// CountRecordsWithoutEmbeddings returns the count of records without embeddings.
func (m *Memory) CountRecordsWithoutEmbeddings(kind string) (int, error) {
	var query string
	switch kind {
	case "idea":
		query = `SELECT COUNT(*) FROM ideas i LEFT JOIN embeddings e ON i.id = e.record_id WHERE e.record_id IS NULL`
	case "decision":
		query = `SELECT COUNT(*) FROM decisions d LEFT JOIN embeddings e ON d.id = e.record_id WHERE e.record_id IS NULL`
	case "learning":
		query = `SELECT COUNT(*) FROM learnings l LEFT JOIN embeddings e ON l.id = e.record_id WHERE e.record_id IS NULL`
	default:
		return 0, nil
	}

	var count int
	err := m.db.QueryRowContext(context.Background(), query).Scan(&count)
	return count, err
}

// EmbeddingStats holds statistics about embeddings.
type EmbeddingStats struct {
	TotalEmbeddings int            `json:"totalEmbeddings"`
	ByKind          map[string]int `json:"byKind"`
	PendingByKind   map[string]int `json:"pendingByKind"`
	QueueSize       int            `json:"queueSize"`
	PipelineRunning bool           `json:"pipelineRunning"`
}

// GetEmbeddingStats returns statistics about the embedding system.
func (m *Memory) GetEmbeddingStats(pipeline *EmbeddingPipeline) (*EmbeddingStats, error) {
	stats := &EmbeddingStats{
		ByKind:        make(map[string]int),
		PendingByKind: make(map[string]int),
	}

	// Total embeddings
	total, err := m.CountEmbeddings()
	if err != nil {
		return nil, err
	}
	stats.TotalEmbeddings = total

	// By kind
	for _, kind := range []string{"idea", "decision", "learning"} {
		var count int
		err := m.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM embeddings WHERE record_kind = ?`, kind).Scan(&count)
		if err == nil {
			stats.ByKind[kind] = count
		}

		pending, err := m.CountRecordsWithoutEmbeddings(kind)
		if err == nil {
			stats.PendingByKind[kind] = pending
		}
	}

	// Pipeline status
	if pipeline != nil {
		stats.QueueSize = pipeline.QueueSize()
		stats.PipelineRunning = pipeline.IsRunning()
	}

	return stats, nil
}
