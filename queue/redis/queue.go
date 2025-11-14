// Package redis provides a Redis-based job queue implementation.
// This package offers distributed queue operations with blocking dequeue and processing tracking.
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// Queue handles job queue operations using Redis
type Queue struct {
	client *redis.Client
	ctx    context.Context
	prefix string // Key prefix for queue keys (e.g., "when:", "eve:")
}

// Job represents a job in the execution queue
type Job struct {
	ActionID   string    `json:"actionID"`
	QueueName  string    `json:"queueName"`
	WorkflowID string    `json:"workflowID"`
	RunID      string    `json:"runID"`
	EnqueuedAt time.Time `json:"enqueuedAt"`
	RetryCount int       `json:"retryCount"`
}

// Config configures the Redis queue
type Config struct {
	RedisURL  string // Redis URL (defaults to WHEN_REDIS_URL or redis://localhost:6379/0)
	KeyPrefix string // Key prefix for queue keys (defaults to "queue:")
}

// NewQueue creates a new Redis queue client
func NewQueue(ctx context.Context, config Config) (*Queue, error) {
	redisURL := config.RedisURL
	if redisURL == "" {
		redisURL = os.Getenv("WHEN_REDIS_URL")
	}
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	prefix := config.KeyPrefix
	if prefix == "" {
		prefix = "queue:"
	}

	return &Queue{
		client: client,
		ctx:    ctx,
		prefix: prefix,
	}, nil
}

// Close closes the Redis connection
func (q *Queue) Close() error {
	return q.client.Close()
}

// Enqueue adds a job to a queue
func (q *Queue) Enqueue(job Job) error {
	jobJSON, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	queueKey := fmt.Sprintf("%s%s", q.prefix, job.QueueName)
	return q.client.RPush(q.ctx, queueKey, string(jobJSON)).Err()
}

// Dequeue removes and returns the next job from a queue (blocking)
func (q *Queue) Dequeue(queueName string, timeout time.Duration) (*Job, error) {
	queueKey := fmt.Sprintf("%s%s", q.prefix, queueName)

	// Use a fresh context with timeout for each dequeue operation
	// This prevents issues with cancelled/expired contexts from init time
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := q.client.BLPop(ctx, timeout, queueKey).Result()
	if err == redis.Nil {
		return nil, nil // Timeout, no job available
	}
	if err != nil {
		return nil, fmt.Errorf("failed to dequeue: %w", err)
	}

	if len(result) < 2 {
		return nil, nil // No job
	}

	var job Job
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// MarkProcessing adds a job to the processing set with a deadline
func (q *Queue) MarkProcessing(actionID string, deadline time.Time) error {
	processingKey := fmt.Sprintf("%sprocessing", q.prefix)
	return q.client.ZAdd(q.ctx, processingKey, redis.Z{
		Score:  float64(deadline.Unix()),
		Member: actionID,
	}).Err()
}

// CompleteJob removes a job from the processing set
func (q *Queue) CompleteJob(actionID string) error {
	processingKey := fmt.Sprintf("%sprocessing", q.prefix)
	return q.client.ZRem(q.ctx, processingKey, actionID).Err()
}

// FailJob marks a job as failed and optionally re-enqueues it
func (q *Queue) FailJob(actionID string, requeue bool, queueName string, retryCount int) error {
	// Remove from processing set
	if err := q.CompleteJob(actionID); err != nil {
		return err
	}

	// Re-enqueue if requested
	if requeue {
		job := Job{
			ActionID:   actionID,
			QueueName:  queueName,
			EnqueuedAt: time.Now(),
			RetryCount: retryCount + 1,
		}
		return q.Enqueue(job)
	}

	return nil
}

// GetQueueDepth returns the number of jobs in a queue
func (q *Queue) GetQueueDepth(queueName string) (int, error) {
	queueKey := fmt.Sprintf("%s%s", q.prefix, queueName)
	depth, err := q.client.LLen(q.ctx, queueKey).Result()
	if err != nil {
		return 0, err
	}
	return int(depth), nil
}

// IsProcessing checks if a job is currently being processed
func (q *Queue) IsProcessing(actionID string) (bool, error) {
	processingKey := fmt.Sprintf("%sprocessing", q.prefix)
	score, err := q.client.ZScore(q.ctx, processingKey, actionID).Result()
	if err == redis.Nil {
		return false, nil // Not in processing set
	}
	if err != nil {
		return false, err
	}
	return score > 0, nil
}

// WaitForJobCompletion waits for a job to complete or timeout
func (q *Queue) WaitForJobCompletion(actionID string, timeout time.Duration, checkStatus func(string) (string, error)) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if action is still in processing set
			inProcessing, err := q.IsProcessing(actionID)
			if err != nil {
				return fmt.Errorf("failed to check processing status: %w", err)
			}

			if !inProcessing {
				// Not in processing set anymore - check if completed or failed
				status, err := checkStatus(actionID)
				if err != nil {
					return fmt.Errorf("failed to get action status: %w", err)
				}

				if status == "CompletedActionStatus" {
					return nil // Success
				} else if status == "FailedActionStatus" {
					return fmt.Errorf("action failed")
				}
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for job completion")
			}
		}
	}
}
