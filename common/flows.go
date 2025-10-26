// Package common provides core data structures and types for the EVE flow process management system.
// This package defines the fundamental types used throughout the EVE evaluation service
// for managing workflow processes, state transitions, and inter-service communication.
//
// The package serves as the foundation for a distributed workflow engine that tracks
// process execution across multiple services using RabbitMQ for messaging and CouchDB
// for persistent state storage. It implements a robust state machine pattern with
// comprehensive audit trails and error handling.
//
// Core Components:
//   - Process state management with defined state transitions
//   - Message structures for inter-service communication
//   - Document models for persistent storage in CouchDB
//   - Configuration management for service coordination
//   - Consumer framework for RabbitMQ message processing
//
// State Management Philosophy:
//
//	The system implements a finite state machine where processes progress through
//	well-defined states with complete audit trails. Each state transition is
//	recorded with timestamps, metadata, and error information for comprehensive
//	process tracking and debugging.
//
// Distributed Architecture:
//
//	Designed for microservices architectures where multiple services coordinate
//	through message passing and shared state storage. The types support both
//	synchronous and asynchronous processing patterns with reliable delivery
//	guarantees through RabbitMQ message acknowledgment.
//
// Data Consistency:
//
//	Uses CouchDB's eventual consistency model with document versioning to handle
//	concurrent updates and maintain data integrity across distributed services.
//	All operations include proper revision management for conflict resolution.
package common

import (
	"net/http"
	"time"

	"github.com/streadway/amqp"
)

// FlowProcessState represents the possible states of a workflow process in the EVE system.
// This enumeration defines a finite state machine that governs process lifecycle
// management with clear state transitions and business logic rules.
//
// State Transition Rules:
//
//	StateStarted    → StateRunning (process begins execution)
//	StateRunning    → StateSuccessful (normal completion)
//	StateRunning    → StateFailed (error termination)
//	StateSuccessful → (terminal state, no further transitions)
//	StateFailed     → (terminal state, no further transitions)
//
// State Semantics:
//   - StateStarted: Process has been initiated and queued for execution
//   - StateRunning: Process is actively executing (may have multiple updates)
//   - StateSuccessful: Process completed successfully with expected outcomes
//   - StateFailed: Process terminated due to errors or validation failures
//
// Usage in System:
//
//	These states are used consistently across all services to ensure uniform
//	process tracking and enable cross-service coordination. State changes
//	trigger various system behaviors including notifications, cleanup, and
//	dependent process initiation.
//
// Audit and Monitoring:
//
//	Each state transition is logged with full context including timestamps,
//	error messages, and metadata for comprehensive audit trails and system
//	monitoring capabilities.
type FlowProcessState string

const (
	StateStarted    FlowProcessState = "started"    // Process initiated and queued
	StateRunning    FlowProcessState = "running"    // Process actively executing
	StateSuccessful FlowProcessState = "successful" // Process completed successfully
	StateFailed     FlowProcessState = "failed"     // Process terminated with errors
)

// FlowProcessMessage represents the structure of messages exchanged between services
// for communicating process state changes and coordination events. This message
// format serves as the primary communication protocol for the distributed workflow engine.
//
// Message Flow Pattern:
//
//	Producer Service → RabbitMQ Queue → Consumer Service → State Update → CouchDB
//
// The message contains all necessary information to process state transitions,
// update persistent storage, and trigger downstream actions without requiring
// additional service calls or data lookups.
//
// Field Descriptions:
//   - ProcessID: Unique identifier for the workflow process (UUID recommended)
//   - State: Target state for this transition (must follow state machine rules)
//   - Timestamp: When the state change occurred (for ordering and audit)
//   - Metadata: Process-specific data and configuration (extensible JSON object)
//   - ErrorMsg: Error details for failed states (required for StateFailed)
//   - Description: Human-readable description of the state change
//
// Message Validation:
//   - ProcessID must be non-empty and unique across the system
//   - State must be a valid FlowProcessState value
//   - Timestamp should be set by the producing service (auto-generated if missing)
//   - ErrorMsg is required when State is StateFailed
//
// Serialization:
//
//	Messages are JSON-serialized for RabbitMQ transport with proper field
//	mapping and optional field handling. The structure supports both compact
//	and verbose message formats depending on use case requirements.
//
// Error Handling:
//
//	Invalid messages are rejected at the consumer level with appropriate
//	error logging and optional dead letter queue routing for investigation.
//
// Example Message:
//
//	{
//	  "process_id": "eval-2024-001",
//	  "state": "running",
//	  "timestamp": "2024-01-15T10:30:00Z",
//	  "metadata": {"step": "validation", "progress": 0.3},
//	  "description": "Starting validation phase"
//	}
type FlowProcessMessage struct {
	ProcessID   string                 `json:"process_id"`              // Unique process identifier
	State       FlowProcessState       `json:"state"`                   // Target state for transition
	Timestamp   time.Time              `json:"timestamp"`               // When state change occurred
	Metadata    map[string]interface{} `json:"metadata,omitempty"`      // Process-specific data
	ErrorMsg    string                 `json:"error_message,omitempty"` // Error details for failures
	Description string                 `json:"description,omitempty"`   // Human-readable description
}

// FlowProcessDocument represents the complete process state document stored in CouchDB.
// This document serves as the authoritative record of process state with full
// audit history and supports the system's eventual consistency model.
//
// CouchDB Integration:
//   - Uses CouchDB's document-based storage with automatic revision management
//   - Document ID format: "flow_process_{ProcessID}" for consistent retrieval
//   - Revision field (_rev) enables optimistic concurrency control
//   - Supports CouchDB replication and conflict resolution mechanisms
//
// Document Evolution:
//
//	Documents grow over time as state changes are appended to the history array.
//	This provides a complete audit trail while maintaining current state in
//	top-level fields for efficient querying and indexing.
//
// Field Descriptions:
//   - ID: CouchDB document identifier (includes "_id" prefix)
//   - Rev: CouchDB revision for concurrency control and conflict resolution
//   - ProcessID: Business process identifier (extracted from document ID)
//   - State: Current process state (updated with each transition)
//   - CreatedAt: Document creation timestamp (process initiation time)
//   - UpdatedAt: Last modification timestamp (most recent state change)
//   - History: Complete chronological list of all state transitions
//   - Metadata: Merged metadata from all state changes (latest values win)
//   - ErrorMsg: Most recent error message (from latest failed state)
//   - Description: Most recent description (from latest state change)
//
// Querying Patterns:
//   - By ProcessID: Direct document retrieval using computed document ID
//   - By State: Secondary index on current state for bulk operations
//   - By Timestamp: Range queries on CreatedAt/UpdatedAt for time-based analysis
//   - By History: Complex queries on state transition patterns and durations
//
// Conflict Resolution:
//
//	CouchDB's MVCC handles concurrent updates through revision conflicts.
//	Applications must handle 409 Conflict responses by retrieving current
//	document, merging changes, and retrying the update operation.
//
// Data Retention:
//
//	Documents persist indefinitely for audit and compliance purposes.
//	Consider implementing archival strategies for completed processes
//	older than retention policy requirements.
//
// Example Document:
//
//	{
//	  "_id": "flow_process_eval-2024-001",
//	  "_rev": "3-a1b2c3d4e5f6",
//	  "process_id": "eval-2024-001",
//	  "state": "successful",
//	  "created_at": "2024-01-15T10:00:00Z",
//	  "updated_at": "2024-01-15T10:45:00Z",
//	  "history": [
//	    {"state": "started", "timestamp": "2024-01-15T10:00:00Z"},
//	    {"state": "running", "timestamp": "2024-01-15T10:15:00Z"},
//	    {"state": "successful", "timestamp": "2024-01-15T10:45:00Z"}
//	  ],
//	  "metadata": {"total_steps": 5, "completion_time": "00:45:00"}
//	}
type FlowProcessDocument struct {
	ID          string                 `json:"_id"`                     // CouchDB document ID
	Rev         string                 `json:"_rev,omitempty"`          // CouchDB revision for MVCC
	ProcessID   string                 `json:"process_id"`              // Business process identifier
	State       FlowProcessState       `json:"state"`                   // Current process state
	CreatedAt   time.Time              `json:"created_at"`              // Process creation timestamp
	UpdatedAt   time.Time              `json:"updated_at"`              // Last update timestamp
	History     []FlowStateChange      `json:"history"`                 // Complete state change history
	Metadata    map[string]interface{} `json:"metadata,omitempty"`      // Aggregated process metadata
	ErrorMsg    string                 `json:"error_message,omitempty"` // Most recent error message
	Description string                 `json:"description,omitempty"`   // Most recent description
}

// FlowStateChange represents a single state transition in the process history.
// This immutable record captures the exact context of each state change for
// comprehensive audit trails and process analysis.
//
// Immutability Principle:
//
//	Once created, state change records are never modified. This ensures
//	audit trail integrity and enables reliable process analysis, debugging,
//	and compliance reporting.
//
// Field Descriptions:
//   - State: The process state that was transitioned to
//   - Timestamp: Exact time when the transition occurred (UTC recommended)
//   - ErrorMsg: Error context if the transition was due to a failure
//
// Ordering and Consistency:
//
//	State changes are appended to the history array in chronological order.
//	Services must ensure proper timestamp ordering when processing messages
//	to maintain logical consistency in the audit trail.
//
// Error Documentation:
//
//	When a process transitions to StateFailed, the ErrorMsg field provides
//	detailed error information for debugging and analysis. This information
//	is preserved in the history even if subsequent operations succeed.
//
// Analysis Applications:
//
//	History records enable various analytical capabilities:
//	- Process duration calculation (time between started and terminal states)
//	- Failure pattern analysis (common error conditions and timing)
//	- Performance optimization (identifying bottlenecks and slow processes)
//	- SLA monitoring (tracking process completion times)
//
// Storage Efficiency:
//
//	While each state change is a separate record, the lightweight structure
//	minimizes storage overhead while maximizing analytical value. Consider
//	implementing history compression for very long-running processes.
//
// Example State Change:
//
//	{
//	  "state": "failed",
//	  "timestamp": "2024-01-15T10:30:00Z",
//	  "error_message": "Validation failed: missing required field 'customer_id'"
//	}
type FlowStateChange struct {
	State     FlowProcessState `json:"state"`                   // Target state of transition
	Timestamp time.Time        `json:"timestamp"`               // When transition occurred
	ErrorMsg  string           `json:"error_message,omitempty"` // Error context for failures
}

// FlowCouchDBResponse represents the standard response format from CouchDB operations.
// This structure captures the essential information returned by CouchDB for
// successful document operations including creation, updates, and deletions.
//
// CouchDB Operation Context:
//
//	CouchDB returns this response format for most document modification
//	operations (PUT, POST, DELETE) to confirm success and provide the
//	new document revision information required for subsequent operations.
//
// Field Descriptions:
//   - OK: Boolean flag indicating operation success (should always be true in success responses)
//   - ID: Document identifier that was operated on (confirms target document)
//   - Rev: New revision ID after the operation (required for future updates)
//
// Revision Management:
//
//	The Rev field is critical for CouchDB's MVCC (Multi-Version Concurrency Control).
//	Applications must store and provide this revision ID for subsequent update
//	operations to prevent conflicts and ensure data consistency.
//
// Error Handling:
//
//	When OK is false or missing, check for CouchDB error responses instead
//	of this structure. HTTP status codes provide additional error context.
//
// Usage Patterns:
//   - Store Rev field for future document updates
//   - Verify ID matches expected document identifier
//   - Log successful operations for audit and monitoring
//
// Example Response:
//
//	{
//	  "ok": true,
//	  "id": "flow_process_eval-2024-001",
//	  "rev": "2-b7c8d9e0f1a2"
//	}
type FlowCouchDBResponse struct {
	OK  bool   `json:"ok"`  // Operation success indicator
	ID  string `json:"id"`  // Document ID that was operated on
	Rev string `json:"rev"` // New document revision after operation
}

// FlowCouchDBError represents error responses from CouchDB operations.
// This structure provides structured error information for diagnosing and
// handling various failure conditions in CouchDB interactions.
//
// CouchDB Error Model:
//
//	CouchDB returns structured error information with both machine-readable
//	error codes and human-readable reason descriptions. This enables both
//	programmatic error handling and useful error reporting.
//
// Common Error Types:
//   - "not_found": Document or database doesn't exist
//   - "conflict": Document revision conflict (concurrent modification)
//   - "forbidden": Access denied or validation failure
//   - "bad_request": Invalid request format or parameters
//   - "internal_server_error": CouchDB server-side errors
//
// Field Descriptions:
//   - Error: Machine-readable error type for programmatic handling
//   - Reason: Human-readable error description for logging and debugging
//
// Error Handling Strategies:
//   - "not_found": Check document/database existence and create if needed
//   - "conflict": Retrieve latest document revision and retry operation
//   - "forbidden": Check permissions and document validation rules
//   - "bad_request": Validate request parameters and document structure
//
// Retry Logic:
//
//	Some errors (like "conflict") are transient and should trigger retry
//	logic with exponential backoff. Others (like "forbidden") indicate
//	permanent failures requiring different handling approaches.
//
// Logging and Monitoring:
//
//	Error information should be logged for operational monitoring and
//	debugging purposes. Track error frequencies to identify systemic issues.
//
// Example Error Response:
//
//	{
//	  "error": "conflict",
//	  "reason": "Document update conflict. Document was modified by another process."
//	}
type FlowCouchDBError struct {
	Error  string `json:"error"`  // Machine-readable error type
	Reason string `json:"reason"` // Human-readable error description
}

// FlowConfig holds the complete configuration for the EVE flow processing system.
// This structure centralizes all configuration parameters required for service
// coordination, message processing, and data persistence across the distributed system.
//
// Configuration Management:
//
//	This structure supports various configuration sources including:
//	- Command-line flags and arguments
//	- Environment variables
//	- Configuration files (YAML, JSON, TOML)
//	- Remote configuration services
//	- Default values for development environments
//
// Field Descriptions:
//   - RabbitMQURL: Complete connection URL for RabbitMQ message broker
//   - QueueName: Name of the queue for process state messages
//   - CouchDBURL: Base URL for CouchDB server (including protocol and port)
//   - DatabaseName: CouchDB database name for storing process documents
//   - ApiKey: Authentication key for API access and JWT token generation
//
// Connection Strings:
//   - RabbitMQURL format: "amqp://username:password@hostname:port/vhost"
//   - CouchDBURL format: "http://username:password@hostname:port"
//   - Support for both local and remote service connections
//
// Security Considerations:
//   - Store sensitive configuration (passwords, API keys) securely
//   - Use environment variables or secret management systems
//   - Avoid hardcoding credentials in source code
//   - Consider certificate-based authentication for production
//
// Environment-Specific Configuration:
//
//	Different environments (development, staging, production) should use
//	separate configuration instances with appropriate service endpoints,
//	credentials, and resource names to prevent cross-environment interference.
//
// Validation Requirements:
//
//	All fields should be validated for proper format and connectivity
//	during service initialization to fail fast on configuration errors.
//
// Example Configuration:
//
//	{
//	  "RabbitMQURL": "amqp://guest:guest@localhost:5672/",
//	  "QueueName": "flow_process_queue",
//	  "CouchDBURL": "http://admin:password@localhost:5984",
//	  "DatabaseName": "flow_processes",
//	  "ApiKey": "your-secret-api-key"
//	}
type FlowConfig struct {
	RabbitMQURL  string // RabbitMQ connection URL with credentials
	QueueName    string // RabbitMQ queue name for process messages
	CouchDBURL   string // CouchDB server URL with credentials
	DatabaseName string // CouchDB database name for process storage
	ApiKey       string // API key for authentication and JWT signing
}

// FlowConsumer handles RabbitMQ message consumption and processing for the flow system.
// This structure encapsulates all the state and dependencies required for reliable
// message processing including connection management, HTTP client configuration,
// and process state coordination.
//
// Consumer Architecture:
//
//	The consumer implements a robust message processing pattern with:
//	- Reliable message delivery through acknowledgments
//	- Automatic reconnection on connection failures
//	- Dead letter queue support for message failures
//	- Graceful shutdown with proper resource cleanup
//
// Field Descriptions:
//   - config: Complete system configuration including service endpoints
//   - connection: Active RabbitMQ connection for message operations
//   - channel: RabbitMQ channel for message consumption and acknowledgment
//   - httpClient: HTTP client for CouchDB operations with appropriate timeouts
//
// Connection Management:
//   - Maintains persistent connection to RabbitMQ for efficient message processing
//   - Implements connection recovery and retry logic for network resilience
//   - Configures appropriate timeouts and heartbeat intervals
//   - Supports both local and remote RabbitMQ deployments
//
// Message Processing Flow:
//  1. Receive message from RabbitMQ queue
//  2. Deserialize and validate message structure
//  3. Process state transition and update CouchDB document
//  4. Acknowledge successful processing or reject with requeue
//  5. Log operation results for monitoring and debugging
//
// Error Handling Strategy:
//   - Transient errors: Reject message with requeue for retry
//   - Permanent errors: Reject message without requeue (dead letter)
//   - Processing errors: Log detailed error information for debugging
//   - Connection errors: Implement automatic reconnection with backoff
//
// Performance Considerations:
//   - Configurable prefetch count for message batching
//   - HTTP connection pooling for CouchDB operations
//   - Concurrent message processing with appropriate limits
//   - Resource monitoring and cleanup for long-running operations
//
// Monitoring and Observability:
//   - Message processing metrics (rate, latency, errors)
//   - Connection health monitoring and alerting
//   - Resource utilization tracking (memory, connections)
//   - Business metrics (process completion rates, state distributions)
//
// Lifecycle Management:
//   - Initialize: Establish connections and validate configuration
//   - Start: Begin message consumption with proper error handling
//   - Stop: Graceful shutdown with message acknowledgment completion
//   - Cleanup: Release resources and close connections properly
//
// Example Usage:
//
//	consumer := &FlowConsumer{
//	    config: flowConfig,
//	    httpClient: &http.Client{Timeout: 30 * time.Second},
//	}
//	err := consumer.Connect()
//	if err != nil {
//	    log.Fatal("Failed to connect:", err)
//	}
//	defer consumer.Close()
//	consumer.StartConsuming()
//
//nolint:unused // FlowConsumer fields are reserved for future use
type FlowConsumer struct {
	config     FlowConfig       // System configuration
	connection *amqp.Connection // RabbitMQ connection
	channel    *amqp.Channel    // RabbitMQ channel for operations
	httpClient *http.Client     // HTTP client for CouchDB operations
}
