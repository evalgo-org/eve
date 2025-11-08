// Package tracing - Async trace export with buffering
package tracing

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AsyncExporter handles buffered, asynchronous trace exports
type AsyncExporter struct {
	tracer      *Tracer
	traceQueue  chan traceRecord
	workers     int
	batchSize   int
	flushPeriod time.Duration
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	stats       *ExporterStats
}

// ExporterStats tracks exporter performance
type ExporterStats struct {
	mu              sync.RWMutex
	TracesQueued    int64
	TracesExported  int64
	TracesFailed    int64
	QueueDropped    int64
	BatchesExported int64
	LastExportTime  time.Time
}

// AsyncExporterConfig configures the async exporter
type AsyncExporterConfig struct {
	Workers     int           // Number of worker goroutines (default: 4)
	QueueSize   int           // Size of trace buffer (default: 10000)
	BatchSize   int           // Max traces per batch (default: 100)
	FlushPeriod time.Duration // Max time to wait before flushing (default: 5s)
}

// NewAsyncExporter creates a new async trace exporter
func NewAsyncExporter(tracer *Tracer, config AsyncExporterConfig) *AsyncExporter {
	// Set defaults
	if config.Workers == 0 {
		config.Workers = 4
	}
	if config.QueueSize == 0 {
		config.QueueSize = 10000
	}
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.FlushPeriod == 0 {
		config.FlushPeriod = 5 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	exporter := &AsyncExporter{
		tracer:      tracer,
		traceQueue:  make(chan traceRecord, config.QueueSize),
		workers:     config.Workers,
		batchSize:   config.BatchSize,
		flushPeriod: config.FlushPeriod,
		ctx:         ctx,
		cancel:      cancel,
		stats:       &ExporterStats{},
	}

	// Start worker pool
	for i := 0; i < config.Workers; i++ {
		exporter.wg.Add(1)
		go exporter.worker(i)
	}

	return exporter
}

// QueueTrace adds a trace to the export queue (non-blocking)
func (e *AsyncExporter) QueueTrace(trace traceRecord) {
	select {
	case e.traceQueue <- trace:
		e.stats.mu.Lock()
		e.stats.TracesQueued++
		e.stats.mu.Unlock()

		// Update Prometheus metrics
		if e.tracer.metrics != nil {
			e.tracer.metrics.ExporterQueueSize.Set(float64(len(e.traceQueue)))
		}
	default:
		// Queue is full, drop the trace
		e.stats.mu.Lock()
		e.stats.QueueDropped++
		e.stats.mu.Unlock()
		e.tracer.logError("Trace queue full, dropping trace", fmt.Errorf("queue_size=%d", len(e.traceQueue)))

		// Update Prometheus metrics
		if e.tracer.metrics != nil {
			e.tracer.metrics.ExporterQueueDropped.WithLabelValues(e.tracer.config.ServiceID).Inc()
		}
	}
}

// worker processes traces from the queue in batches
func (e *AsyncExporter) worker(workerID int) {
	defer e.wg.Done()

	batch := make([]traceRecord, 0, e.batchSize)
	ticker := time.NewTicker(e.flushPeriod)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}

		// Measure batch export latency
		start := time.Now()

		// Export batch
		if err := e.exportBatch(batch); err != nil {
			e.stats.mu.Lock()
			e.stats.TracesFailed += int64(len(batch))
			e.stats.mu.Unlock()
			e.tracer.logError(fmt.Sprintf("Worker %d failed to export batch", workerID), err)

			// Record failure metric
			if e.tracer.metrics != nil {
				e.tracer.metrics.ExporterBatches.WithLabelValues(e.tracer.config.ServiceID, "failure").Inc()
			}
		} else {
			e.stats.mu.Lock()
			e.stats.TracesExported += int64(len(batch))
			e.stats.BatchesExported++
			e.stats.LastExportTime = time.Now()
			e.stats.mu.Unlock()

			// Record success metrics
			if e.tracer.metrics != nil {
				e.tracer.metrics.ExporterBatches.WithLabelValues(e.tracer.config.ServiceID, "success").Inc()
				e.tracer.metrics.ExporterLatency.Observe(time.Since(start).Seconds())
			}
		}

		// Clear batch
		batch = batch[:0]
	}

	for {
		select {
		case <-e.ctx.Done():
			// Drain remaining traces
			flush()
			return

		case trace := <-e.traceQueue:
			batch = append(batch, trace)

			// Flush if batch is full
			if len(batch) >= e.batchSize {
				flush()
			}

		case <-ticker.C:
			// Periodic flush
			flush()
		}
	}
}

// exportBatch exports a batch of traces to PostgreSQL and S3
func (e *AsyncExporter) exportBatch(batch []traceRecord) error {
	// Process each trace in the batch
	for _, trace := range batch {
		// Use the existing recordTrace logic
		e.tracer.recordTrace(trace)
	}

	return nil
}

// Shutdown gracefully stops the async exporter
func (e *AsyncExporter) Shutdown(ctx context.Context) error {
	// Signal workers to stop
	e.cancel()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		close(e.traceQueue)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout: %w", ctx.Err())
	}
}

// Stats returns current exporter statistics
func (e *AsyncExporter) Stats() ExporterStats {
	e.stats.mu.RLock()
	defer e.stats.mu.RUnlock()

	return ExporterStats{
		TracesQueued:    e.stats.TracesQueued,
		TracesExported:  e.stats.TracesExported,
		TracesFailed:    e.stats.TracesFailed,
		QueueDropped:    e.stats.QueueDropped,
		BatchesExported: e.stats.BatchesExported,
		LastExportTime:  e.stats.LastExportTime,
	}
}

// QueueLength returns current queue size
func (e *AsyncExporter) QueueLength() int {
	return len(e.traceQueue)
}

// IsHealthy returns true if exporter is working properly
func (e *AsyncExporter) IsHealthy() bool {
	stats := e.Stats()

	// Check if we're exporting successfully
	if stats.TracesExported == 0 && stats.TracesQueued > 100 {
		return false
	}

	// Check if drop rate is too high (>5%)
	if stats.TracesQueued > 0 {
		dropRate := float64(stats.QueueDropped) / float64(stats.TracesQueued)
		if dropRate > 0.05 {
			return false
		}
	}

	// Check if we've exported recently (within 30s)
	if stats.TracesExported > 0 && time.Since(stats.LastExportTime) > 30*time.Second {
		return false
	}

	return true
}
