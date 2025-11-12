// Package worker provides a generic worker pool for processing queued jobs.
// This package offers concurrent job processing with configurable worker counts per queue.
package worker

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Queue defines the interface for job queue operations
type Queue interface {
	Dequeue(queueName string, timeout time.Duration) (interface{}, error)
	Enqueue(job interface{}) error
	MarkProcessing(jobID string, deadline time.Time) error
	CompleteJob(jobID string) error
	FailJob(jobID string, requeue bool, queueName string, retryCount int) error
}

// JobProcessor defines the interface for processing jobs
type JobProcessor interface {
	Process(ctx context.Context, job interface{}) error
	GetJobID(job interface{}) string
	GetTimeout(job interface{}) time.Duration
}

// Pool manages a pool of workers that process jobs from queues
type Pool struct {
	workers   []*Worker
	queue     Queue
	processor JobProcessor
	stopChan  chan struct{}
}

// Worker represents a single worker that processes jobs from a queue
type Worker struct {
	id        int
	queueName string
	queue     Queue
	processor JobProcessor
	stopChan  chan struct{}
}

// Config configures the worker pool
type Config struct {
	Queues map[string]int // Queue name -> number of workers
}

// DefaultConfig returns the default worker configuration
func DefaultConfig() Config {
	return Config{
		Queues: map[string]int{
			"sequential": 1, // Only 1 worker for sequential processing
			"parallel":   5, // 5 workers for parallel processing
			"priority":   2, // 2 workers for priority queue
		},
	}
}

// NewPool creates a new worker pool
func NewPool(queue Queue, processor JobProcessor, config Config) *Pool {
	pool := &Pool{
		workers:   make([]*Worker, 0),
		queue:     queue,
		processor: processor,
		stopChan:  make(chan struct{}),
	}

	// Create workers for each queue
	for queueName, workerCount := range config.Queues {
		for i := 0; i < workerCount; i++ {
			worker := &Worker{
				id:        i,
				queueName: queueName,
				queue:     queue,
				processor: processor,
				stopChan:  make(chan struct{}),
			}
			pool.workers = append(pool.workers, worker)
		}
	}

	return pool
}

// Start starts all workers in the pool
func (p *Pool) Start() {
	log.Printf("Starting worker pool with %d workers", len(p.workers))

	for _, worker := range p.workers {
		go worker.Start()
		log.Printf("Started worker %d for queue '%s'", worker.id, worker.queueName)
	}
}

// Stop stops all workers in the pool
func (p *Pool) Stop() {
	log.Println("Stopping worker pool...")
	close(p.stopChan)

	for _, worker := range p.workers {
		close(worker.stopChan)
	}

	log.Println("Worker pool stopped")
}

// Start starts a worker processing loop
func (w *Worker) Start() {
	log.Printf("Worker %d (%s queue) started", w.id, w.queueName)

	for {
		select {
		case <-w.stopChan:
			log.Printf("Worker %d (%s queue) stopped", w.id, w.queueName)
			return
		default:
			// Process next job from queue
			if err := w.processNext(); err != nil {
				log.Printf("Worker %d (%s queue) error: %v", w.id, w.queueName, err)
				// Don't exit on error, continue processing
				time.Sleep(1 * time.Second)
			}
		}
	}
}

// processNext fetches and processes the next job from the queue
func (w *Worker) processNext() error {
	// Dequeue next job (blocking with 5s timeout)
	job, err := w.queue.Dequeue(w.queueName, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to dequeue: %w", err)
	}

	if job == nil {
		// Timeout, no job available
		return nil
	}

	jobID := w.processor.GetJobID(job)
	log.Printf("Worker %d (%s queue) processing job %s", w.id, w.queueName, jobID)

	// Get timeout for this job
	timeout := w.processor.GetTimeout(job)
	deadline := time.Now().Add(timeout)

	// Mark as processing
	if err := w.queue.MarkProcessing(jobID, deadline); err != nil {
		log.Printf("Worker %d failed to mark job %s as processing: %v", w.id, jobID, err)
		// Re-enqueue the job
		w.queue.Enqueue(job)
		return nil
	}

	// Create execution context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Process job
	err = w.processor.Process(ctx, job)

	if err != nil {
		log.Printf("Worker %d job %s failed: %v", w.id, jobID, err)

		// Mark as failed - processor should handle retry logic
		if failErr := w.queue.FailJob(jobID, false, w.queueName, 0); failErr != nil {
			log.Printf("Worker %d failed to mark job as failed: %v", w.id, failErr)
		}

		return nil
	}

	// Success
	log.Printf("Worker %d completed job %s", w.id, jobID)

	// Mark as completed
	if err := w.queue.CompleteJob(jobID); err != nil {
		log.Printf("Worker %d failed to mark job as completed: %v", w.id, err)
	}

	return nil
}
