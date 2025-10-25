// Package cli provides command-line interface functionality for the EVE evaluation system.
// It includes a RabbitMQ message consumer that processes workflow state changes and
// persists them to CouchDB, along with utility functions for file operations and
// command execution.
//
// The package implements a robust message processing pipeline that:
//   - Consumes messages from RabbitMQ queues
//   - Validates and processes state change messages
//   - Maintains process state history in CouchDB
//   - Provides graceful shutdown and error handling
//
// Key Components:
//   - Cobra CLI framework integration for command structure
//   - RabbitMQ consumer with automatic acknowledgment and retry logic
//   - CouchDB client with document versioning and conflict handling
//   - Process state management with full audit trail
//   - Configuration management via Viper and environment variables
package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	eve "eve.evalgo.org/common"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ProcessState represents the possible states of a workflow process.
// These states form a finite state machine that tracks the lifecycle
// of evaluation processes from initiation through completion or failure.
//
// State Transitions:
//
//	StateStarted -> StateRunning -> StateSuccessful|StateFailed
//
// Each state change is recorded in the process history for audit purposes.
type ProcessState string

const (
	StateStarted    ProcessState = "started"    // Process has been initiated
	StateRunning    ProcessState = "running"    // Process is actively executing
	StateSuccessful ProcessState = "successful" // Process completed successfully
	StateFailed     ProcessState = "failed"     // Process terminated with errors
)

// ProcessMessage represents the structure of incoming RabbitMQ messages.
// This message format is used to communicate state changes between different
// components of the evaluation system.
//
// Message Flow:
//
//	Producer -> RabbitMQ Queue -> Consumer -> CouchDB
//
// The message contains all necessary information to update process state
// and maintain a complete audit trail of the process lifecycle.
type ProcessMessage struct {
	ProcessID   string                 `json:"process_id"`              // Unique identifier for the process
	State       ProcessState           `json:"state"`                   // New state of the process
	Timestamp   time.Time              `json:"timestamp"`               // When the state change occurred
	Metadata    map[string]interface{} `json:"metadata,omitempty"`      // Additional process-specific data
	ErrorMsg    string                 `json:"error_message,omitempty"` // Error details for failed states
	Description string                 `json:"description,omitempty"`   // Human-readable description of the change
}

// ProcessDocument represents the complete CouchDB document structure for a process.
// This document maintains the current state and full history of a process,
// enabling both real-time status queries and historical analysis.
//
// CouchDB Integration:
//   - Uses document ID format: "process_{ProcessID}"
//   - Maintains revision history via CouchDB's built-in versioning
//   - Supports optimistic concurrency control through revision IDs
//
// The document grows over time as state changes are appended to the history,
// providing a complete audit trail of process execution.
type ProcessDocument struct {
	ID          string                 `json:"_id"`                     // CouchDB document ID
	Rev         string                 `json:"_rev,omitempty"`          // CouchDB revision ID for versioning
	ProcessID   string                 `json:"process_id"`              // Business process identifier
	State       ProcessState           `json:"state"`                   // Current process state
	CreatedAt   time.Time              `json:"created_at"`              // Document creation timestamp
	UpdatedAt   time.Time              `json:"updated_at"`              // Last modification timestamp
	History     []StateChange          `json:"history"`                 // Complete state change history
	Metadata    map[string]interface{} `json:"metadata,omitempty"`      // Aggregated metadata from all changes
	ErrorMsg    string                 `json:"error_message,omitempty"` // Most recent error message
	Description string                 `json:"description,omitempty"`   // Most recent description
}

// StateChange represents a single state transition in the process history.
// Each state change is immutable and preserves the exact context of when
// the transition occurred, enabling detailed process analysis and debugging.
//
// History Tracking:
//   - Chronologically ordered list of all state changes
//   - Immutable records for audit compliance
//   - Detailed error context for failure analysis
type StateChange struct {
	State     ProcessState `json:"state"`                   // The state that was transitioned to
	Timestamp time.Time    `json:"timestamp"`               // Exact time of the transition
	ErrorMsg  string       `json:"error_message,omitempty"` // Error details if transition was due to failure
}

// CouchDBResponse represents the standard response format from CouchDB operations.
// CouchDB returns this structure for most document operations, providing
// confirmation of successful operations and the new revision ID.
//
// Revision Management:
//
//	The Rev field is crucial for CouchDB's MVCC (Multi-Version Concurrency Control)
//	and must be included in subsequent update operations to prevent conflicts.
type CouchDBResponse struct {
	OK  bool   `json:"ok"`  // Indicates if the operation was successful
	ID  string `json:"id"`  // The document ID that was operated on
	Rev string `json:"rev"` // New revision ID after the operation
}

// CouchDBError represents error responses from CouchDB operations.
// CouchDB returns structured error information that helps diagnose
// issues with document operations, database connectivity, and data validation.
//
// Common Errors:
//   - "not_found": Document or database doesn't exist
//   - "conflict": Document revision conflict (concurrent updates)
//   - "forbidden": Access denied or validation failure
type CouchDBError struct {
	Error  string `json:"error"`  // Error type identifier
	Reason string `json:"reason"` // Human-readable error description
}

// Config holds the complete application configuration for the consumer.
// Configuration is loaded from command-line flags, environment variables,
// and configuration files using Viper's precedence rules.
//
// Configuration Sources (in order of precedence):
//  1. Command-line flags
//  2. Environment variables
//  3. Configuration files
//  4. Default values
type Config struct {
	RabbitMQURL string // RabbitMQ connection URL (amqp://...)
	QueueName   string // Name of the queue to consume from
	CouchDBURL  string // CouchDB server URL (http://...)
	CouchDBName string // CouchDB database name
	ApiKey      string // API key for authentication (if required)
}

// Consumer handles the complete RabbitMQ message consumption workflow.
// It manages connections to both RabbitMQ and CouchDB, processes incoming
// messages, and maintains process state with full error handling and recovery.
//
// Architecture:
//
//	RabbitMQ -> Consumer -> Message Processing -> CouchDB -> Response
//
// The consumer implements reliable message processing with acknowledgments,
// automatic retries for transient failures, and graceful shutdown handling.
type Consumer struct {
	config     Config           // Application configuration
	connection *amqp.Connection // Active RabbitMQ connection
	channel    *amqp.Channel    // RabbitMQ channel for message operations
	httpClient *http.Client     // HTTP client for CouchDB operations
}

// Props represents properties for process operations.
// This structure is used for passing process-related parameters
// to various utility functions within the CLI package.
//
// Usage:
//
//	Primarily used for file processing and process identification
//	in batch operations and utility functions.
type Props struct {
	InFile string // Input file path for processing
	PID    string // Process identifier
}

// init initializes the CLI command structure and configuration bindings.
// This function sets up the Cobra command hierarchy and binds command-line
// flags to Viper configuration keys for flexible configuration management.
//
// Command Structure:
//
//	root
//	└── consume (message consumption command)
//
// Configuration Binding:
//
//	Maps CLI flags to Viper configuration keys, enabling configuration
//	via multiple sources (flags, env vars, config files).
func init() {
	RootCmd.AddCommand(consumeCmd)
	consumeCmd.PersistentFlags().String("rabbitmq-url", "", "RabbitMQ connection URL")
	consumeCmd.PersistentFlags().String("queue-name", "", "RabbitMQ queue name")
	consumeCmd.PersistentFlags().String("couchdb-url", "", "CouchDB connection URL")
	consumeCmd.PersistentFlags().String("database-name", "", "CouchDB database name")

	viper.BindPFlag("rabbitmq.url", consumeCmd.PersistentFlags().Lookup("rabbitmq-url"))
	viper.BindPFlag("rabbitmq.queue_name", consumeCmd.PersistentFlags().Lookup("queue-name"))
	viper.BindPFlag("couchdb.url", consumeCmd.PersistentFlags().Lookup("couchdb-url"))
	viper.BindPFlag("couchdb.database_name", consumeCmd.PersistentFlags().Lookup("database-name"))
}

// consumeCmd defines the CLI command for starting the message consumer.
// This command initializes and starts the RabbitMQ consumer with the provided
// configuration, handling the complete lifecycle from startup to graceful shutdown.
//
// Command Usage:
//
//	eve consume --rabbitmq-url amqp://localhost --queue-name eval_queue --couchdb-url http://localhost:5984 --database-name processes
//
// Configuration Priority:
//  1. Command-line flags (highest priority)
//  2. Viper configuration (env vars, config files)
//  3. Default values (lowest priority)
//
// The command validates all required configuration and starts the consumer
// with proper error handling and logging.
var consumeCmd = &cobra.Command{
	Use:   "consume",
	Short: "consume an rabbit message queue",
	Long: `Consume messages from a RabbitMQ queue and process workflow state changes.

This command starts a persistent consumer that:
- Connects to RabbitMQ and declares the specified queue
- Processes incoming process state change messages
- Updates process documents in CouchDB with state history
- Handles errors with automatic retry and dead letter logic
- Supports graceful shutdown on SIGINT/SIGTERM signals

The consumer maintains reliable message processing with acknowledgments
and provides comprehensive logging for monitoring and debugging.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration with precedence: flags > viper > defaults
		rabbitmq_url, _ := cmd.Flags().GetString("rabbitmq-url")
		if rabbitmq_url == "" {
			rabbitmq_url = viper.GetString("rabbitmq.url")
		}
		queue_name, _ := cmd.Flags().GetString("queue-name")
		if queue_name == "" {
			queue_name = viper.GetString("rabbitmq.queue_name")
		}
		couchdb_url, _ := cmd.Flags().GetString("couchdb-url")
		if couchdb_url == "" {
			couchdb_url = viper.GetString("couchdb.url")
		}
		database_name, _ := cmd.Flags().GetString("database-name")
		if database_name == "" {
			database_name = viper.GetString("couchdb.database_name")
		}

		eve.Logger.Info(rabbitmq_url, queue_name, couchdb_url, database_name)
		ConsumerStart(Config{
			RabbitMQURL: rabbitmq_url,
			QueueName:   queue_name,
			CouchDBURL:  couchdb_url,
			CouchDBName: database_name,
		})
	},
}

// ConsumerStart initializes and starts the message consumer with the provided configuration.
// This function orchestrates the complete consumer lifecycle including connection setup,
// database initialization, message consumption, and graceful shutdown.
//
// Startup Sequence:
//  1. Create consumer instance with configuration
//  2. Initialize CouchDB database (create if not exists)
//  3. Establish RabbitMQ connection and declare queue
//  4. Start message consumption in background goroutine
//  5. Wait for shutdown signals (SIGINT, SIGTERM)
//  6. Perform graceful shutdown and resource cleanup
//
// Error Handling:
//   - Fatal errors during startup terminate the process
//   - Runtime errors are logged and handled per-message
//   - Graceful shutdown ensures proper resource cleanup
//
// Parameters:
//   - config: Complete configuration for RabbitMQ and CouchDB connections
//
// Signals:
//   - SIGINT (Ctrl+C): Triggers graceful shutdown
//   - SIGTERM: Triggers graceful shutdown for container environments
func ConsumerStart(config Config) {
	consumer := NewConsumer(config)
	defer consumer.Close()

	// Create CouchDB database if it doesn't exist
	if err := consumer.createCouchDBDatabase(); err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}

	// Connect to RabbitMQ
	if err := consumer.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	// Start consuming
	if err := consumer.StartConsuming(); err != nil {
		log.Fatalf("Failed to start consuming: %v", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("Consumer is running. Press CTRL+C to exit...")
	<-sigChan

	log.Println("Shutting down consumer...")
}

// NewConsumer creates a new Consumer instance with the provided configuration.
// The consumer is initialized with default HTTP client settings optimized
// for CouchDB operations, including appropriate timeouts for reliable operation.
//
// HTTP Client Configuration:
//   - 10-second timeout for CouchDB operations
//   - Default transport settings for connection pooling
//   - Suitable for production CouchDB interactions
//
// Parameters:
//   - config: Complete configuration including RabbitMQ and CouchDB settings
//
// Returns:
//   - *Consumer: Configured consumer instance ready for connection
//
// Note: The consumer requires explicit Connect() call to establish connections.
func NewConsumer(config Config) *Consumer {
	return &Consumer{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Connect establishes connections to RabbitMQ and prepares the consumer for message processing.
// This method handles the complete RabbitMQ setup including connection, channel creation,
// queue declaration, and QoS configuration for reliable message processing.
//
// Connection Setup:
//  1. Establish connection to RabbitMQ server
//  2. Create channel for message operations
//  3. Declare queue with durability settings
//  4. Configure QoS for message processing control
//
// Queue Configuration:
//   - Durable: Queue survives server restarts
//   - Not auto-delete: Queue persists when consumers disconnect
//   - Not exclusive: Multiple consumers can connect
//   - QoS: Process one message at a time for reliability
//
// Returns:
//   - error: nil on success, error details on failure
//
// Error Conditions:
//   - Network connectivity issues to RabbitMQ
//   - Authentication failures
//   - Queue declaration conflicts
//   - Channel creation failures
func (c *Consumer) Connect() error {
	var err error
	c.connection, err = amqp.Dial(c.config.RabbitMQURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	c.channel, err = c.connection.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare queue
	_, err = c.channel.QueueDeclare(
		c.config.QueueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Set QoS to process one message at a time
	err = c.channel.Qos(1, 0, false)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	return nil
}

// Close gracefully shuts down all consumer connections and resources.
// This method ensures proper cleanup of RabbitMQ connections and channels,
// preventing resource leaks and allowing for clean application shutdown.
//
// Shutdown Sequence:
//  1. Close RabbitMQ channel (stops message consumption)
//  2. Close RabbitMQ connection (releases network resources)
//  3. HTTP client cleanup (handled by garbage collector)
//
// This method is safe to call multiple times and handles nil connections gracefully.
func (c *Consumer) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.connection != nil {
		c.connection.Close()
	}
}

// StartConsuming begins consuming messages from the configured RabbitMQ queue.
// This method sets up the message consumption pipeline with proper acknowledgment
// handling, error recovery, and concurrent processing.
//
// Message Processing Flow:
//  1. Register consumer with RabbitMQ
//  2. Receive messages in background goroutine
//  3. Process each message individually
//  4. Acknowledge successful processing or reject with requeue
//
// Reliability Features:
//   - Manual acknowledgment for guaranteed delivery
//   - Automatic requeue on processing failures
//   - Concurrent processing with QoS limits
//   - Comprehensive error logging
//
// Returns:
//   - error: nil on successful setup, error details on failure
//
// Note: Message processing continues until the consumer is shut down.
func (c *Consumer) StartConsuming() error {
	msgs, err := c.channel.Consume(
		c.config.QueueName,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Printf("Consumer started. Waiting for messages...")

	go func() {
		for msg := range msgs {
			if err := c.processMessage(msg); err != nil {
				log.Printf("Error processing message: %v", err)
				// Reject message and requeue
				msg.Nack(false, true)
			} else {
				// Acknowledge message
				msg.Ack(false)
			}
		}
	}()

	return nil
}

// processMessage handles the complete processing of a single RabbitMQ message.
// This method deserializes the message, validates its contents, and routes it
// to the appropriate processing function based on the process state.
//
// Processing Steps:
//  1. Deserialize JSON message to ProcessMessage struct
//  2. Validate required fields (ProcessID, State)
//  3. Set default timestamp if not provided
//  4. Route to appropriate handler based on state
//
// State Routing:
//   - StateStarted: Creates new process document
//   - StateRunning/StateSuccessful/StateFailed: Updates existing document
//
// Validation Rules:
//   - ProcessID is required and non-empty
//   - State is required and must be valid ProcessState
//   - Timestamp is auto-generated if not provided
//
// Parameters:
//   - msg: RabbitMQ message delivery containing the process state change
//
// Returns:
//   - error: nil on successful processing, error details on failure
//
// Error Conditions:
//   - Invalid JSON format
//   - Missing required fields
//   - Invalid process state
//   - Database operation failures
func (c *Consumer) processMessage(msg amqp.Delivery) error {
	log.Printf("Received message: %s", string(msg.Body))

	var processMsg ProcessMessage
	if err := json.Unmarshal(msg.Body, &processMsg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Validate message
	if processMsg.ProcessID == "" {
		return fmt.Errorf("process_id is required")
	}

	if processMsg.State == "" {
		return fmt.Errorf("state is required")
	}

	if processMsg.Timestamp.IsZero() {
		processMsg.Timestamp = time.Now()
	}

	// Process the message based on state
	switch processMsg.State {
	case StateStarted:
		return c.createProcessDocument(processMsg)
	case StateRunning, StateSuccessful, StateFailed:
		return c.updateProcessDocument(processMsg)
	default:
		return fmt.Errorf("invalid state: %s", processMsg.State)
	}
}

// createProcessDocument creates a new process document in CouchDB for a started process.
// This method initializes a complete ProcessDocument with the initial state and
// creates the first entry in the process history.
//
// Document Structure:
//   - ID: "process_{ProcessID}" format for consistent lookup
//   - Initial state and timestamps set to message values
//   - History initialized with the first state change
//   - Metadata copied from the message
//
// CouchDB Operation:
//   - Uses POST to create new document with auto-generated revision
//   - Returns error if document already exists
//   - Logs successful creation with revision ID
//
// Parameters:
//   - msg: ProcessMessage containing the initial process state
//
// Returns:
//   - error: nil on successful creation, error details on failure
//
// Error Conditions:
//   - Document already exists (process ID collision)
//   - CouchDB connectivity issues
//   - JSON serialization failures
//   - Database write permissions
func (c *Consumer) createProcessDocument(msg ProcessMessage) error {
	docID := fmt.Sprintf("process_%s", msg.ProcessID)

	doc := ProcessDocument{
		ID:          docID,
		ProcessID:   msg.ProcessID,
		State:       msg.State,
		CreatedAt:   msg.Timestamp,
		UpdatedAt:   msg.Timestamp,
		History:     []StateChange{{State: msg.State, Timestamp: msg.Timestamp, ErrorMsg: msg.ErrorMsg}},
		Metadata:    msg.Metadata,
		ErrorMsg:    msg.ErrorMsg,
		Description: msg.Description,
	}

	return c.saveToCouchDB(doc, false)
}

// updateProcessDocument updates an existing process document with new state information.
// This method implements the complete update workflow including document retrieval,
// state merging, history appending, and optimistic concurrency control.
//
// Update Process:
//  1. Retrieve existing document with current revision
//  2. Update current state and timestamp
//  3. Merge metadata (new values override existing)
//  4. Append state change to history
//  5. Save with revision ID for conflict detection
//
// Metadata Merging:
//   - New metadata values override existing keys
//   - Existing keys not in the message are preserved
//   - Nil metadata in message leaves existing metadata unchanged
//
// History Management:
//   - Each state change is appended to history array
//   - History provides complete audit trail
//   - Timestamps preserve exact sequence of events
//
// Parameters:
//   - msg: ProcessMessage containing the state update
//
// Returns:
//   - error: nil on successful update, error details on failure
//
// Error Conditions:
//   - Document not found (process never started)
//   - Revision conflicts (concurrent updates)
//   - CouchDB connectivity issues
//   - JSON serialization failures
func (c *Consumer) updateProcessDocument(msg ProcessMessage) error {
	docID := fmt.Sprintf("process_%s", msg.ProcessID)

	// First, get the existing document
	existingDoc, err := c.getFromCouchDB(docID)
	if err != nil {
		return fmt.Errorf("failed to get existing document: %w", err)
	}

	// Update the document
	existingDoc.State = msg.State
	existingDoc.UpdatedAt = msg.Timestamp
	existingDoc.ErrorMsg = msg.ErrorMsg

	if msg.Description != "" {
		existingDoc.Description = msg.Description
	}

	// Merge metadata
	if msg.Metadata != nil {
		if existingDoc.Metadata == nil {
			existingDoc.Metadata = make(map[string]interface{})
		}
		for k, v := range msg.Metadata {
			existingDoc.Metadata[k] = v
		}
	}

	// Add to history
	stateChange := StateChange{
		State:     msg.State,
		Timestamp: msg.Timestamp,
		ErrorMsg:  msg.ErrorMsg,
	}
	existingDoc.History = append(existingDoc.History, stateChange)

	return c.saveToCouchDB(*existingDoc, true)
}

// getFromCouchDB retrieves a process document from CouchDB by document ID.
// This method handles the complete document retrieval workflow including
// HTTP communication, error parsing, and JSON deserialization.
//
// HTTP Request:
//   - GET /{database}/{docID}
//   - Standard CouchDB document retrieval
//   - Includes revision ID in response
//
// Error Handling:
//   - 404 Not Found: Returns specific "document not found" error
//   - Other HTTP errors: Parses CouchDB error response
//   - Network errors: Returns wrapped connection errors
//
// Parameters:
//   - docID: CouchDB document ID (format: "process_{ProcessID}")
//
// Returns:
//   - *ProcessDocument: Pointer to retrieved document on success
//   - error: nil on success, specific error details on failure
//
// Response Processing:
//   - Validates HTTP status codes
//   - Deserializes JSON to ProcessDocument struct
//   - Preserves CouchDB revision ID for subsequent updates
func (c *Consumer) getFromCouchDB(docID string) (*ProcessDocument, error) {
	url := fmt.Sprintf("%s/%s/%s", c.config.CouchDBURL, c.config.CouchDBName, docID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("document not found: %s", docID)
	}

	if resp.StatusCode != http.StatusOK {
		var couchErr CouchDBError
		if err := json.NewDecoder(resp.Body).Decode(&couchErr); err == nil {
			return nil, fmt.Errorf("CouchDB error: %s - %s", couchErr.Error, couchErr.Reason)
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var doc ProcessDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("failed to decode document: %w", err)
	}

	return &doc, nil
}

// saveToCouchDB saves a process document to CouchDB using appropriate HTTP methods.
// This method handles both document creation (POST) and updates (PUT) with
// proper revision management and error handling.
//
// HTTP Methods:
//   - POST /{database}: Creates new document, CouchDB assigns revision
//   - PUT /{database}/{docID}: Updates existing document with revision check
//
// Revision Control:
//   - New documents: No revision required, CouchDB generates first revision
//   - Updates: Must include current revision ID to prevent conflicts
//   - Conflicts: Returns error if revision doesn't match current
//
// Response Validation:
//   - 200 OK: Document updated successfully
//   - 201 Created: Document created successfully
//   - Other codes: Error condition, parsed from response body
//
// Parameters:
//   - doc: ProcessDocument to save (includes revision for updates)
//   - isUpdate: true for updates (PUT), false for creation (POST)
//
// Returns:
//   - error: nil on success, detailed error information on failure
//
// Success Logging:
//   - Logs document ID and new revision on successful operations
//   - Provides confirmation of database state changes
func (c *Consumer) saveToCouchDB(doc ProcessDocument, isUpdate bool) error {
	jsonData, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	var url string
	var method string

	if isUpdate {
		url = fmt.Sprintf("%s/%s/%s", c.config.CouchDBURL, c.config.CouchDBName, doc.ID)
		method = "PUT"
	} else {
		url = fmt.Sprintf("%s/%s", c.config.CouchDBURL, c.config.CouchDBName)
		method = "POST"
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to save document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var couchErr CouchDBError
		if err := json.NewDecoder(resp.Body).Decode(&couchErr); err == nil {
			return fmt.Errorf("CouchDB error: %s - %s", couchErr.Error, couchErr.Reason)
		}
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response CouchDBResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.OK {
		return fmt.Errorf("CouchDB operation failed")
	}

	log.Printf("Document %s saved successfully with rev: %s", response.ID, response.Rev)
	return nil
}

// createCouchDBDatabase creates the CouchDB database if it doesn't already exist.
// This method ensures the required database is available before starting
// message consumption, handling both creation and existence scenarios.
//
// Database Creation:
//   - Uses HTTP PUT to create database
//   - Returns success if database already exists
//   - Logs creation status for monitoring
//
// HTTP Status Handling:
//   - 201 Created: Database was created successfully
//   - 412 Precondition Failed: Database already exists (acceptable)
//   - Other codes: Error condition requiring investigation
//
// Returns:
//   - error: nil on success or if database exists, error details on failure
//
// Error Conditions:
//   - Network connectivity to CouchDB
//   - Authentication/authorization failures
//   - CouchDB server errors
//   - Invalid database names
//
// This method is called during consumer startup to ensure required
// infrastructure is available before processing begins.
func (c *Consumer) createCouchDBDatabase() error {
	url := fmt.Sprintf("%s/%s", c.config.CouchDBURL, c.config.CouchDBName)

	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		log.Printf("Database %s created successfully", c.config.CouchDBName)
	} else if resp.StatusCode == http.StatusPreconditionFailed {
		log.Printf("Database %s already exists", c.config.CouchDBName)
	} else {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// getEnvOrDefault returns the value of an environment variable or a default value.
// This utility function provides a consistent way to handle optional environment
// variables with fallback defaults throughout the application.
//
// Usage Pattern:
//
//	Used for configuration values that have reasonable defaults but can be
//	overridden via environment variables for different deployment environments.
//
// Parameters:
//   - key: Environment variable name to check
//   - defaultValue: Value to return if environment variable is not set or empty
//
// Returns:
//   - string: Environment variable value if set and non-empty, defaultValue otherwise
//
// Environment Variable Handling:
//   - Empty strings are treated as unset
//   - Leading/trailing whitespace is preserved
//   - Case-sensitive environment variable names
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// listFiles recursively walks a directory and returns all file paths.
// This utility function provides file system traversal for batch processing
// operations and file discovery workflows.
//
// Traversal Behavior:
//   - Recursively processes all subdirectories
//   - Returns paths to regular files only (excludes directories)
//   - Preserves full relative paths from the starting directory
//   - Uses filepath.WalkDir for efficient traversal
//
// Parameters:
//   - dir: Root directory path to start traversal from
//
// Returns:
//   - []string: Slice of file paths relative to the starting directory
//
// Error Handling:
//   - Fatal error on traversal failures (log.Fatal)
//   - Continues on individual file errors
//   - No partial results on failure
//
// Use Cases:
//   - Batch file processing workflows
//   - Archive preparation operations
//   - File discovery for evaluation processes
//
// Note: The commented code shows an example of filtering by file extension
// (e.g., only .md files) which can be uncommented for specific use cases.
func listFiles(dir string) []string {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			files = append(files, path)
		}
		// if !d.IsDir() && filepath.Ext(path) == ".md" {
		// 	files = append(files, path)
		// }
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return files
}

// RunCommand executes an external command with stdout/stderr redirection.
// This utility function provides a standardized way to execute external
// processes with proper output handling and error propagation.
//
// Output Handling:
//   - Redirects command stdout to current process stdout
//   - Redirects command stderr to current process stderr
//   - Enables real-time output streaming for long-running commands
//
// Error Handling:
//   - Returns wrapped error from command execution
//   - Preserves exit codes and error messages
//   - Suitable for command success/failure validation
//
// Parameters:
//   - cmd: Configured exec.Cmd ready for execution
//
// Returns:
//   - error: nil on successful execution (exit code 0), error details on failure
//
// Usage Examples:
//
//	cmd := exec.Command("git", "clone", repoURL)
//	if err := RunCommand(cmd); err != nil {
//	    log.Printf("Git clone failed: %v", err)
//	}
//
//	cmd := exec.Command("make", "build")
//	err := RunCommand(cmd)
//
// Security Considerations:
//   - Command arguments should be validated to prevent injection
//   - Consider using absolute paths for executables
//   - Be cautious with user-provided arguments
func RunCommand(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
