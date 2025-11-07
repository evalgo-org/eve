// Package repository provides a unified interface for multi-database storage patterns.
// This package implements the Repository pattern for semantic data management across
// multiple specialized databases (CouchDB, Neo4j, PostgreSQL, Redis/Valkey).
//
// Architecture:
//
//	The repository pattern abstracts database operations into domain-specific interfaces,
//	allowing applications to work with semantic data without coupling to specific database
//	implementations. Each repository type serves a distinct purpose:
//
//	- DocumentRepository: JSON-LD document storage (CouchDB)
//	- GraphRepository: Relationship topology (Neo4j)
//	- MetricsRepository: Time-series execution data (PostgreSQL)
//	- CacheRepository: Ephemeral data and locks (Redis/Valkey)
//
// Design Philosophy:
//
//  1. Semantic Preservation: Full JSON-LD documents stored in document database
//  2. Specialized Storage: Graph relationships in graph database for performance
//  3. Operational Data: Metrics and history in time-series optimized storage
//  4. Ephemeral Data: Locks and cache in fast in-memory storage
//
// Usage Pattern:
//
//	Applications compose these repositories based on their needs:
//	- Workflow systems use all four types
//	- Simple services may only need DocumentRepository
//	- Read-heavy workloads benefit from CacheRepository
//	- Dependency management requires GraphRepository
package repository

import (
	"context"
	"time"

	"eve.evalgo.org/semantic"
)

// DocumentRepository manages JSON-LD document storage in CouchDB.
// This interface provides CRUD operations for semantic documents with
// full Schema.org vocabulary support and real-time change feeds.
//
// Implementation: CouchDB
//   - Native JSON document storage
//   - MVCC for concurrency control
//   - Change feeds for real-time updates
//   - Replication support
//
// Document Structure:
//   - Complete JSON-LD documents with @context and @type
//   - Schema.org vocabulary preserved
//   - Custom properties supported via additionalProperty
//
// Concurrency:
//   - Optimistic locking via document revisions (_rev)
//   - Conflict detection automatic
//   - Last-write-wins by default
type DocumentRepository interface {
	// Workflow operations
	SaveWorkflow(ctx context.Context, workflowID string, workflow map[string]interface{}) error
	GetWorkflow(ctx context.Context, workflowID string) (map[string]interface{}, error)
	ListWorkflows(ctx context.Context) ([]map[string]interface{}, error)
	DeleteWorkflow(ctx context.Context, workflowID string) error

	// Action operations
	SaveAction(ctx context.Context, actionID string, action *semantic.SemanticScheduledAction) error
	GetAction(ctx context.Context, actionID string) (*semantic.SemanticScheduledAction, error)
	ListActions(ctx context.Context, workflowID string) ([]*semantic.SemanticScheduledAction, error)
	DeleteAction(ctx context.Context, actionID string) error

	// Bulk operations for performance
	BulkSaveActions(ctx context.Context, actions []*semantic.SemanticScheduledAction) error

	// Real-time updates
	WatchChanges(ctx context.Context) (<-chan ChangeEvent, error)
}

// GraphRepository manages relationship topology in Neo4j.
// This interface provides graph operations for dependency management,
// cycle detection, and path finding between semantic entities.
//
// Implementation: Neo4j
//   - Property graph model
//   - Cypher query language
//   - Native graph algorithms
//   - Transitive closure support
//
// Graph Structure:
//   - Nodes: Action, Workflow (with identifier property)
//   - Relationships: REQUIRES (dependencies), PART_OF (containment)
//   - Properties: Minimal metadata for performance
//
// Performance:
//   - 10-100x faster than SQL for graph queries
//   - Native cycle detection
//   - Efficient transitive dependencies
type GraphRepository interface {
	// Action graph operations
	StoreActionGraph(ctx context.Context, action *semantic.SemanticScheduledAction) error
	DeleteActionGraph(ctx context.Context, actionID string) error

	// Dependency queries
	GetDependencies(ctx context.Context, actionID string) ([]string, error)            // Direct dependencies
	GetAllDependencies(ctx context.Context, actionID string) ([]string, error)         // Transitive closure
	GetDependents(ctx context.Context, actionID string) ([]string, error)              // Reverse dependencies
	WouldCreateCycle(ctx context.Context, actionID, dependencyID string) (bool, error) // Cycle detection
	FindPath(ctx context.Context, fromAction, toAction string) ([]string, error)       // Shortest path

	// Workflow operations
	GetWorkflowActions(ctx context.Context, workflowID string) ([]string, error)
	LinkActionToWorkflow(ctx context.Context, actionID, workflowID string) error
	DeleteWorkflowGraph(ctx context.Context, workflowID string) error
}

// MetricsRepository manages time-series execution data in PostgreSQL.
// This interface provides operations for recording action executions,
// querying historical data, and computing aggregate metrics.
//
// Implementation: PostgreSQL
//   - JSONB for flexible schema
//   - Time-series indexes
//   - Aggregation functions
//   - Retention policies
//
// Data Model:
//   - ActionRun: Single execution record with timing data
//   - ActionMetrics: Aggregated statistics over time
//   - DataPoint: Time-bucketed metric value
//
// Use Cases:
//   - Execution history and audit trails
//   - Performance monitoring and alerting
//   - Capacity planning and analysis
//   - SLA tracking and reporting
type MetricsRepository interface {
	// Execution recording
	SaveRun(ctx context.Context, run *ActionRun) error
	GetRunHistory(ctx context.Context, actionID string, limit int) ([]*ActionRun, error)

	// Metrics and analytics
	GetMetrics(ctx context.Context, actionID string, from, to time.Time) (*ActionMetrics, error)
	GetAggregatedMetrics(ctx context.Context, actionID string, window time.Duration, aggregation string) ([]DataPoint, error)

	// Maintenance
	DeleteOldRuns(ctx context.Context, before time.Time) (int64, error)
}

// CacheRepository manages ephemeral data in Redis/Valkey.
// This interface provides operations for distributed locks, caching,
// pub/sub messaging, and counters.
//
// Implementation: Redis/Valkey/DragonflyDB
//   - In-memory storage
//   - TTL support
//   - Atomic operations
//   - Pub/sub messaging
//
// Use Cases:
//   - Distributed locking (prevent duplicate execution)
//   - Read-through caching (reduce database load)
//   - Event distribution (pub/sub)
//   - Rate limiting (counters)
//
// Consistency:
//   - Eventually consistent
//   - No durability guarantees
//   - Fast failover
type CacheRepository interface {
	// Distributed locking
	AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, key string) error
	IsLocked(ctx context.Context, key string) (bool, error)

	// Caching
	SetCache(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	GetCache(ctx context.Context, key string, value interface{}) error
	DeleteCache(ctx context.Context, key string) error

	// Pub/sub
	Publish(ctx context.Context, channel string, message interface{}) error
	Subscribe(ctx context.Context, channel string) (<-chan interface{}, error)

	// Counters
	Increment(ctx context.Context, key string) (int64, error)
	Decrement(ctx context.Context, key string) (int64, error)
}

// Data Types

// ActionRun represents a single action execution with timing and result data.
type ActionRun struct {
	RunID      string                 // Unique run identifier
	ActionID   string                 // Action being executed
	WorkflowID string                 // Containing workflow
	StartTime  time.Time              // Execution start
	EndTime    time.Time              // Execution end
	Duration   time.Duration          // Total duration
	Status     string                 // CompletedActionStatus, FailedActionStatus
	Error      string                 // Error message if failed
	Result     map[string]interface{} // Execution result data
	Attempt    int                    // Retry attempt number
}

// ActionMetrics represents aggregated metrics for an action over a time period.
type ActionMetrics struct {
	ActionID       string        // Action identifier
	TotalRuns      int64         // Total number of executions
	SuccessfulRuns int64         // Successful executions
	FailedRuns     int64         // Failed executions
	AvgDuration    time.Duration // Average execution time
	MinDuration    time.Duration // Minimum execution time
	MaxDuration    time.Duration // Maximum execution time
	LastRun        time.Time     // Most recent execution
}

// DataPoint represents a metric value at a specific time (for time-series data).
type DataPoint struct {
	Timestamp time.Time // Time bucket
	Value     float64   // Metric value
}

// ChangeEvent represents a document change notification from CouchDB.
type ChangeEvent struct {
	Type      string                 // "workflow" or "action"
	Operation string                 // "updated" or "deleted"
	ID        string                 // Document ID
	Document  map[string]interface{} // Full document (if include_docs=true)
}
