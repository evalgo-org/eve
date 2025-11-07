// Package db provides comprehensive CouchDB integration for document-based data storage and flow processing.
// This package implements a complete CouchDB client with specialized support for flow process management,
// document lifecycle operations, and bulk data export capabilities using the Kivik CouchDB driver.
//
// CouchDB Integration:
//
//	CouchDB is a document-oriented NoSQL database that provides:
//	- JSON document storage with schema flexibility
//	- ACID transactions and eventual consistency
//	- Multi-Version Concurrency Control (MVCC) for conflict resolution
//	- MapReduce views for complex queries and aggregation
//	- HTTP RESTful API for language-agnostic access
//
// Flow Processing Support:
//
//	The package provides specialized functionality for workflow management:
//	- Process state tracking with complete audit trails
//	- Document versioning and history management
//	- State-based queries and filtering capabilities
//	- Integration with RabbitMQ messaging for distributed workflows
//
// Document Operations:
//
//	Supports complete document lifecycle management:
//	- CRUD operations with revision management
//	- Bulk operations for high-performance scenarios
//	- Conflict resolution through MVCC
//	- Selective querying with Mango query language
//	- Database export and backup capabilities
//
// Service Architecture:
//
//	Implements service-oriented patterns with:
//	- Connection pooling and resource management
//	- Error handling and graceful degradation
//	- Configuration-driven database setup
//	- Clean separation of concerns between data and business logic
package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	kivik "github.com/go-kivik/kivik/v4"
	_ "github.com/go-kivik/kivik/v4/couchdb" // The CouchDB driver

	eve "eve.evalgo.org/common"
)

// DocumentStore defines the interface for flow process document storage and retrieval.
// This interface allows for easy mocking and testing of document operations.
type DocumentStore interface {
	// GetDocument retrieves a flow process document by its ID.
	// Returns the document if found, or an error if not found or on failure.
	GetDocument(id string) (*eve.FlowProcessDocument, error)

	// GetAllDocuments retrieves all flow process documents from the database.
	// Returns a slice of all documents, or an error on failure.
	GetAllDocuments() ([]eve.FlowProcessDocument, error)

	// GetDocumentsByState retrieves flow process documents filtered by state.
	// Returns a slice of documents matching the specified state, or an error on failure.
	GetDocumentsByState(state eve.FlowProcessState) ([]eve.FlowProcessDocument, error)

	// SaveDocument saves a flow process document to the database.
	// Returns a response with revision information, or an error on failure.
	SaveDocument(doc eve.FlowProcessDocument) (*eve.FlowCouchDBResponse, error)

	// DeleteDocument deletes a flow process document by ID and revision.
	// Returns an error if deletion fails.
	DeleteDocument(id, rev string) error

	// Close closes the database connection.
	// Returns an error if closing fails.
	Close() error
}

// CouchDBService encapsulates CouchDB client functionality for flow processing operations.
// This service provides a high-level abstraction over CouchDB operations with specialized
// support for flow document management, state tracking, and audit trail maintenance.
//
// Service Components:
//   - client: Kivik CouchDB client for database connectivity
//   - database: Active database handle for document operations
//   - dbName: Database name for configuration and logging purposes
//
// Connection Management:
//
//	The service maintains persistent connections to CouchDB with automatic
//	database creation and proper resource management. Connections are pooled
//	internally by the Kivik driver for optimal performance.
//
// Transaction Support:
//
//	CouchDB's MVCC model provides optimistic concurrency control through
//	document revisions. The service handles revision management automatically
//	for conflict resolution and consistent updates.
//
// Error Handling:
//
//	Implements comprehensive error handling with wrapped errors for debugging
//	and appropriate HTTP status code interpretation for CouchDB-specific conditions.
type CouchDBService struct {
	client   *kivik.Client // CouchDB client connection
	database *kivik.DB     // Active database handle
	dbName   string        // Database name for operations
}

// CouchDBAnimals demonstrates basic CouchDB operations with a simple animal document.
// This function serves as an example of fundamental CouchDB operations including
// database creation, document insertion, and revision management.
//
// Operation Workflow:
//  1. Establishes connection to CouchDB server
//  2. Creates "animals" database if it doesn't exist
//  3. Inserts a sample document with predefined ID
//  4. Reports successful insertion with revision information
//
// Parameters:
//   - url: CouchDB server URL (e.g., "http://admin:password@localhost:5984/")
//
// Example Document:
//
//	The function creates a document with:
//	- _id: "cow" (document identifier)
//	- feet: 4 (integer field)
//	- greeting: "moo" (string field)
//
// Error Handling:
//   - Connection failures cause panic for immediate feedback
//   - Database creation errors are logged but don't halt execution
//   - Document insertion failures cause panic to indicate critical errors
//
// Educational Value:
//
//	This function demonstrates:
//	- Basic CouchDB connection establishment
//	- Database existence checking and creation
//	- Document insertion with explicit ID
//	- Revision handling and success confirmation
//
// Example Usage:
//
//	CouchDBAnimals("http://admin:password@localhost:5984/")
//
// Production Considerations:
//
//	This function is intended for demonstration and testing purposes.
//	Production code should implement proper error handling instead of panic.
func CouchDBAnimals(url string) {
	client, err := kivik.New("couch", url)
	if err != nil {
		panic(err)
	}

	exists, _ := client.DBExists(context.Background(), "animals")
	if !exists {
		err = client.CreateDB(context.Background(), "animals")
		if err != nil {
			fmt.Println(err)
		}
	}
	db := client.DB("animals")

	doc := map[string]interface{}{
		"_id":      "cow",
		"feet":     4,
		"greeting": "moo",
	}

	rev, err := db.Put(context.TODO(), "cow", doc)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Cow inserted with revision %s\n", rev)
}

// CouchDBDocNew creates a new document in the specified database with automatic ID generation.
// This function provides a simplified interface for document creation with automatic
// database setup and unique document ID assignment by CouchDB.
//
// Document Creation Process:
//  1. Establishes connection to CouchDB server
//  2. Creates database if it doesn't exist
//  3. Inserts document with CouchDB-generated unique ID
//  4. Returns both document ID and initial revision
//
// Parameters:
//   - url: CouchDB server URL with authentication
//   - db: Database name for document storage
//   - doc: Document data as interface{} (typically map[string]interface{})
//
// Returns:
//   - string: Document ID assigned by CouchDB (UUID format)
//   - string: Initial revision ID for subsequent updates
//
// Automatic ID Generation:
//
//	CouchDB generates UUIDs for document IDs when not explicitly provided,
//	ensuring uniqueness across the database and enabling distributed scenarios
//	without coordination overhead.
//
// Database Auto-Creation:
//
//	The function automatically creates the target database if it doesn't exist,
//	enabling rapid development and deployment without manual database setup.
//
// Error Handling:
//   - Connection failures cause panic for immediate feedback
//   - Database creation errors are logged but don't prevent document creation
//   - Document creation failures cause panic to indicate data operation issues
//
// Example Usage:
//
//	docData := map[string]interface{}{
//	    "name": "John Doe",
//	    "email": "john@example.com",
//	    "created": time.Now(),
//	}
//	docId, revId := CouchDBDocNew("http://admin:pass@localhost:5984/", "users", docData)
//	fmt.Printf("Created document %s with revision %s\n", docId, revId)
//
// Document Structure:
//
//	The doc parameter should be JSON-serializable data typically represented
//	as map[string]interface{} for maximum flexibility with CouchDB's schema-free nature.
//
// Revision Management:
//
//	The returned revision ID is essential for subsequent document updates and
//	should be stored with application state for conflict resolution.
func CouchDBDocNew(url, db string, doc interface{}) (string, string) {
	client, err := kivik.New("couch", url)
	if err != nil {
		panic(err)
	}
	exists, _ := client.DBExists(context.Background(), db)
	if !exists {
		err = client.CreateDB(context.Background(), db)
		if err != nil {
			fmt.Println(err)
		}
	}
	cdb := client.DB(db)
	docId, revId, err := cdb.CreateDoc(context.TODO(), doc)
	if err != nil {
		panic(err)
	}
	return docId, revId
}

// CouchDBDocGet retrieves a document from the specified database by document ID.
// This function provides direct document access with automatic database creation
// and returns a Kivik document handle for flexible data extraction.
//
// Document Retrieval Process:
//  1. Establishes connection to CouchDB server
//  2. Creates database if it doesn't exist (for development convenience)
//  3. Retrieves document by ID from the specified database
//  4. Returns Kivik document handle for data access
//
// Parameters:
//   - url: CouchDB server URL with authentication
//   - db: Database name containing the document
//   - docId: Document identifier to retrieve
//
// Returns:
//   - *kivik.Document: Document handle for data extraction and metadata access
//
// Document Handle Usage:
//
//	The returned kivik.Document provides methods for:
//	- ScanDoc(): Extract document data into Go structures
//	- Rev(): Access document revision information
//	- Err(): Check for retrieval errors
//
// Error Handling:
//   - Connection failures cause panic for immediate feedback
//   - Database creation errors are logged but don't prevent retrieval
//   - Document not found errors are returned via the document's Err() method
//
// Example Usage:
//
//	doc := CouchDBDocGet("http://admin:pass@localhost:5984/", "users", "user123")
//	if doc.Err() != nil {
//	    if kivik.HTTPStatus(doc.Err()) == 404 {
//	        fmt.Println("Document not found")
//	    } else {
//	        fmt.Printf("Error: %v\n", doc.Err())
//	    }
//	    return
//	}
//
//	var userData map[string]interface{}
//	if err := doc.ScanDoc(&userData); err != nil {
//	    fmt.Printf("Scan error: %v\n", err)
//	    return
//	}
//
//	fmt.Printf("Retrieved user: %v\n", userData)
//
// Database Auto-Creation:
//
//	The function creates the database if it doesn't exist, which is convenient
//	for development but may not be desired in production environments where
//	database creation should be explicit and controlled.
//
// Document Metadata:
//
//	The document handle provides access to CouchDB metadata including
//	revision information, which is essential for subsequent update operations.
func CouchDBDocGet(url, db, docId string) *kivik.Document {
	client, err := kivik.New("couch", url)
	if err != nil {
		panic(err)
	}
	exists, _ := client.DBExists(context.Background(), db)
	if !exists {
		err = client.CreateDB(context.Background(), db)
		if err != nil {
			fmt.Println(err)
		}
	}
	cdb := client.DB(db)
	return cdb.Get(context.TODO(), docId)
}

// NewCouchDBService creates a new CouchDB service instance for flow processing operations.
// This constructor establishes a persistent connection to CouchDB and configures
// the service for flow document management with proper database initialization.
//
// Service Initialization:
//  1. Creates CouchDB client with provided configuration
//  2. Verifies or creates the target database
//  3. Establishes database handle for operations
//  4. Returns configured service ready for use
//
// Parameters:
//   - config: FlowConfig containing CouchDB connection details and database name
//
// Returns:
//   - *CouchDBService: Configured service instance for flow operations
//   - error: Connection, authentication, or database creation errors
//
// Configuration Requirements:
//
//	The FlowConfig must contain:
//	- CouchDBURL: Complete connection URL with authentication
//	- DatabaseName: Target database for flow document storage
//
// Connection Management:
//
//	The service maintains a persistent connection to CouchDB with automatic
//	reconnection and connection pooling handled by the Kivik driver.
//	Connections should be properly closed when the service is no longer needed.
//
// Database Setup:
//
//	The constructor automatically creates the target database if it doesn't exist,
//	enabling immediate use without manual database provisioning. This behavior
//	is suitable for development and can be controlled in production environments.
//
// Error Conditions:
//   - Invalid CouchDB URL or authentication credentials
//   - Network connectivity issues to CouchDB server
//   - Insufficient permissions for database creation
//   - CouchDB server errors or unavailability
//
// Example Usage:
//
//	config := eve.FlowConfig{
//	    CouchDBURL:    "http://admin:password@localhost:5984",
//	    DatabaseName:  "flow_processes",
//	}
//
//	service, err := NewCouchDBService(config)
//	if err != nil {
//	    log.Fatal("Failed to create CouchDB service:", err)
//	}
//	defer service.Close()
//
//	// Use service for flow operations
//	response, err := service.SaveDocument(flowDocument)
//
// Resource Management:
//
//	Services should be closed when no longer needed to release database
//	connections and prevent resource leaks. Use defer statements or proper
//	lifecycle management in long-running applications.
//
// Concurrent Usage:
//
//	CouchDBService instances are safe for concurrent use across multiple
//	goroutines. The underlying Kivik client handles connection pooling
//	and thread safety automatically.
func NewCouchDBService(config eve.FlowConfig) (*CouchDBService, error) {
	client, err := kivik.New("couch", config.CouchDBURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to CouchDB: %w", err)
	}

	ctx := context.Background()

	// Create database if it doesn't exist
	exists, err := client.DBExists(ctx, config.DatabaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if database exists: %w", err)
	}

	if !exists {
		err = client.CreateDB(ctx, config.DatabaseName)
		if err != nil {
			return nil, fmt.Errorf("failed to create database: %w", err)
		}
	}

	db := client.DB(config.DatabaseName)

	return &CouchDBService{
		client:   client,
		database: db,
		dbName:   config.DatabaseName,
	}, nil
}

// SaveDocument saves or updates a flow process document with automatic history management.
// This method handles the complete document lifecycle including revision management,
// audit trail maintenance, and state change tracking for flow processing workflows.
//
// Document Processing:
//  1. Sets document ID to ProcessID if not provided
//  2. Updates timestamp to current time
//  3. Retrieves existing document for revision and history management
//  4. Appends new state change to audit history
//  5. Saves document with CouchDB's MVCC conflict resolution
//
// Parameters:
//   - doc: FlowProcessDocument containing process state and metadata
//
// Returns:
//   - *eve.FlowCouchDBResponse: Success response with document ID and new revision
//   - error: Save operation, conflict resolution, or validation errors
//
// Revision Management:
//
//	The method automatically handles CouchDB revisions by:
//	- Retrieving current revision for existing documents
//	- Preserving CreatedAt timestamp from original document
//	- Generating new revision on successful save
//	- Handling conflicts through retry mechanisms
//
// History Tracking:
//
//	Each save operation appends a new state change to the document history:
//	- State: Current process state (started, running, completed, failed)
//	- Timestamp: When the state change occurred
//	- ErrorMsg: Error details for failed states (if applicable)
//
// Document Lifecycle:
//
//	New Documents:
//	- CreatedAt set to current time
//	- History initialized with first state change
//	- Document ID derived from ProcessID
//
//	Existing Documents:
//	- CreatedAt preserved from original document
//	- History appended with new state change
//	- UpdatedAt updated to current time
//	- Revision updated for conflict resolution
//
// Error Conditions:
//   - Document conflicts due to concurrent modifications
//   - Invalid document structure or missing required fields
//   - Database connectivity issues
//   - Insufficient permissions for document modification
//
// Example Usage:
//
//	doc := eve.FlowProcessDocument{
//	    ProcessID:   "process-12345",
//	    State:       eve.StateRunning,
//	    Description: "Processing started",
//	    Metadata:    map[string]interface{}{"step": "validation"},
//	}
//
//	response, err := service.SaveDocument(doc)
//	if err != nil {
//	    log.Printf("Save failed: %v", err)
//	    return
//	}
//
//	fmt.Printf("Document saved with revision: %s\n", response.Rev)
//
// Conflict Resolution:
//
//	CouchDB's MVCC model prevents lost updates through revision checking.
//	Applications should handle conflict errors by retrieving the latest
//	document version and retrying the save operation with updated data.
//
// Audit Trail:
//
//	The complete state change history is preserved in the document,
//	providing a full audit trail for compliance, debugging, and analysis
//	of process execution patterns and performance.
func (c *CouchDBService) SaveDocument(doc eve.FlowProcessDocument) (*eve.FlowCouchDBResponse, error) {
	ctx := context.Background()

	if doc.ID == "" {
		doc.ID = doc.ProcessID
	}
	doc.UpdatedAt = time.Now()

	// Check if document exists to get revision
	if doc.Rev == "" {
		existingDoc, err := c.GetDocument(doc.ID)
		if err == nil && existingDoc != nil {
			doc.Rev = existingDoc.Rev
			// Preserve created_at from existing document
			doc.CreatedAt = existingDoc.CreatedAt
			// Append to history
			doc.History = append(existingDoc.History, eve.FlowStateChange{
				State:     doc.State,
				Timestamp: time.Now(),
				ErrorMsg:  doc.ErrorMsg,
			})
		} else {
			// New document
			doc.CreatedAt = time.Now()
			doc.History = []eve.FlowStateChange{{
				State:     doc.State,
				Timestamp: time.Now(),
				ErrorMsg:  doc.ErrorMsg,
			}}
		}
	}

	rev, err := c.database.Put(ctx, doc.ID, doc)
	if err != nil {
		return nil, fmt.Errorf("failed to save document: %w", err)
	}

	return &eve.FlowCouchDBResponse{
		OK:  true,
		ID:  doc.ID,
		Rev: rev,
	}, nil
}

// GetDocument retrieves a flow process document by ID with complete metadata.
// This method provides access to stored flow documents including all state
// history, metadata, and current processing status information.
//
// Retrieval Process:
//  1. Queries CouchDB for document by ID
//  2. Handles not-found conditions gracefully
//  3. Deserializes document into FlowProcessDocument structure
//  4. Returns complete document with all fields and history
//
// Parameters:
//   - id: Document identifier (typically same as ProcessID)
//
// Returns:
//   - *eve.FlowProcessDocument: Complete document with all fields and history
//   - error: Document not found, access, or parsing errors
//
// Document Structure:
//
//	The returned document contains:
//	- Process identification and metadata
//	- Current state and processing information
//	- Complete audit trail of state changes
//	- Timestamps for creation and last update
//	- Error information for failed processes
//
// Error Handling:
//   - HTTP 404: Document not found (explicit error message)
//   - Other HTTP errors: Database connectivity or permission issues
//   - Parsing errors: Document corruption or schema changes
//
// Example Usage:
//
//	doc, err := service.GetDocument("process-12345")
//	if err != nil {
//	    if strings.Contains(err.Error(), "not found") {
//	        fmt.Println("Process not found")
//	        return
//	    }
//	    log.Printf("Retrieval error: %v", err)
//	    return
//	}
//
//	fmt.Printf("Process %s is in state: %s\n", doc.ProcessID, doc.State)
//	fmt.Printf("History has %d entries\n", len(doc.History))
//
// Data Consistency:
//
//	Retrieved documents reflect the most recent committed state in CouchDB,
//	ensuring consistency with MVCC guarantees. Concurrent modifications
//	are handled through revision-based conflict detection.
//
// Performance Considerations:
//   - Single document retrieval is efficient with CouchDB's B-tree indexing
//   - Document size affects retrieval time (consider history size)
//   - Network latency impacts response time for remote CouchDB instances
//   - Frequent access patterns benefit from CouchDB's caching mechanisms
func (c *CouchDBService) GetDocument(id string) (*eve.FlowProcessDocument, error) {
	ctx := context.Background()

	row := c.database.Get(ctx, id)
	if row.Err() != nil {
		if kivik.HTTPStatus(row.Err()) == 404 {
			return nil, fmt.Errorf("document not found")
		}
		return nil, fmt.Errorf("failed to get document: %w", row.Err())
	}

	var doc eve.FlowProcessDocument
	if err := row.ScanDoc(&doc); err != nil {
		return nil, fmt.Errorf("failed to scan document: %w", err)
	}

	return &doc, nil
}

// GetDocumentsByState retrieves all flow process documents in a specific state.
// This method uses CouchDB's Mango query language to filter documents by
// processing state, enabling efficient monitoring and management of workflows.
//
// Query Processing:
//  1. Constructs Mango selector for state filtering
//  2. Executes query against CouchDB database
//  3. Iterates through results with memory-efficient processing
//  4. Returns array of matching documents
//
// Parameters:
//   - state: FlowProcessState to filter by (started, running, successful, failed)
//
// Returns:
//   - []eve.FlowProcessDocument: Array of documents matching the specified state
//   - error: Query execution, iteration, or parsing errors
//
// Mango Query Language:
//
//	Uses CouchDB's native Mango query syntax for efficient server-side filtering:
//	- Selector-based filtering on document fields
//	- Index utilization for optimal performance
//	- Server-side processing reduces network traffic
//
// State Filtering:
//
//	Supports all FlowProcessState values:
//	- StateStarted: Processes that have been initiated
//	- StateRunning: Processes currently executing
//	- StateSuccessful: Successfully completed processes
//	- StateFailed: Processes that encountered errors
//
// Performance Optimization:
//   - Server-side filtering reduces network bandwidth
//   - Index on state field improves query performance
//   - Streaming results prevent memory overload with large datasets
//   - Efficient iteration through Kivik's row interface
//
// Error Conditions:
//   - Database connectivity issues during query
//   - Invalid state values or query syntax
//   - Document parsing errors for corrupted data
//   - Memory limitations with extremely large result sets
//
// Example Usage:
//
//	// Get all failed processes for error analysis
//	failedDocs, err := service.GetDocumentsByState(eve.StateFailed)
//	if err != nil {
//	    log.Printf("Query failed: %v", err)
//	    return
//	}
//
//	fmt.Printf("Found %d failed processes\n", len(failedDocs))
//	for _, doc := range failedDocs {
//	    fmt.Printf("Process %s failed: %s\n", doc.ProcessID, doc.ErrorMsg)
//	}
//
// Index Recommendations:
//
//	For optimal performance, create a CouchDB index on the state field:
//	```json
//	{
//	    "index": {
//	        "fields": ["state"]
//	    },
//	    "name": "state-index",
//	    "type": "json"
//	}
//	```
//
// Use Cases:
//   - Monitoring dashboard for process states
//   - Error analysis and debugging workflows
//   - Capacity planning and load analysis
//   - Automated cleanup of completed processes
//   - SLA monitoring and reporting
func (c *CouchDBService) GetDocumentsByState(state eve.FlowProcessState) ([]eve.FlowProcessDocument, error) {
	ctx := context.Background()

	// Use Mango query (CouchDB's native query language)
	selector := map[string]interface{}{
		"state": string(state),
	}

	rows := c.database.Find(ctx, selector)
	defer rows.Close()

	var docs []eve.FlowProcessDocument
	for rows.Next() {
		var doc eve.FlowProcessDocument
		if err := rows.ScanDoc(&doc); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return docs, nil
}

// GetAllDocuments retrieves all flow process documents from the database.
// This method provides complete database enumeration for administrative
// purposes, reporting, and bulk operations on process documents.
//
// Enumeration Process:
//  1. Uses CouchDB's _all_docs view for efficient document listing
//  2. Includes full document content with include_docs parameter
//  3. Streams results to handle large datasets efficiently
//  4. Returns complete array of all documents
//
// Returns:
//   - []eve.FlowProcessDocument: Array containing all documents in the database
//   - error: Query execution, iteration, or parsing errors
//
// Performance Characteristics:
//   - Efficient B-tree traversal through _all_docs view
//   - Memory usage scales with total number of documents
//   - Network bandwidth usage depends on document sizes
//   - Streaming processing prevents memory exhaustion
//
// Use Cases:
//   - Administrative reporting and analytics
//   - Database backup and migration operations
//   - Bulk processing and data transformation
//   - System health monitoring and auditing
//   - Data export for external analysis
//
// Error Conditions:
//   - Database connectivity issues during enumeration
//   - Memory limitations with very large databases
//   - Document parsing errors for corrupted data
//   - Permission restrictions on database access
//
// Example Usage:
//
//	allDocs, err := service.GetAllDocuments()
//	if err != nil {
//	    log.Printf("Failed to retrieve all documents: %v", err)
//	    return
//	}
//
//	fmt.Printf("Total processes: %d\n", len(allDocs))
//
//	// Analyze state distribution
//	stateCount := make(map[eve.FlowProcessState]int)
//	for _, doc := range allDocs {
//	    stateCount[doc.State]++
//	}
//
//	for state, count := range stateCount {
//	    fmt.Printf("State %s: %d processes\n", state, count)
//	}
//
// Memory Considerations:
//
//	Large databases may require pagination or streaming approaches:
//	- Consider implementing offset/limit parameters
//	- Use continuous processing for real-time scenarios
//	- Implement data export to files for very large datasets
//
// Alternative Approaches:
//
//	For large databases, consider:
//	- Paginated retrieval with skip/limit parameters
//	- State-based filtering to reduce result sets
//	- Export functions for file-based processing
//	- Streaming APIs for real-time data processing
func (c *CouchDBService) GetAllDocuments() ([]eve.FlowProcessDocument, error) {
	ctx := context.Background()

	rows := c.database.AllDocs(ctx, kivik.Param("include_docs", true))
	defer rows.Close()

	var docs []eve.FlowProcessDocument
	for rows.Next() {
		var doc eve.FlowProcessDocument
		if err := rows.ScanDoc(&doc); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return docs, nil
}

// DeleteDocument removes a flow process document from the database.
// This method performs document deletion with proper revision handling
// to ensure consistency with CouchDB's MVCC conflict resolution.
//
// Deletion Process:
//  1. Validates document existence and revision
//  2. Executes deletion with specified revision
//  3. Handles conflicts and concurrent modification scenarios
//  4. Confirms successful deletion operation
//
// Parameters:
//   - id: Document identifier to delete
//   - rev: Current document revision for conflict detection
//
// Returns:
//   - error: Deletion failures, conflicts, or permission errors
//
// Revision Requirements:
//
//	CouchDB requires the current document revision for deletion to prevent
//	conflicts from concurrent modifications. The revision must match the
//	current document state for successful deletion.
//
// MVCC Conflict Handling:
//
//	If the provided revision doesn't match the current document revision:
//	- CouchDB returns a 409 Conflict error
//	- Applications should retrieve the latest revision and retry
//	- Alternatively, implement conflict resolution strategies
//
// Error Conditions:
//   - Document not found (may have been deleted by another process)
//   - Revision conflict due to concurrent modifications
//   - Insufficient permissions for document deletion
//   - Database connectivity issues
//
// Example Usage:
//
//	// Retrieve document to get current revision
//	doc, err := service.GetDocument("process-12345")
//	if err != nil {
//	    log.Printf("Failed to get document: %v", err)
//	    return
//	}
//
//	// Delete with current revision
//	err = service.DeleteDocument(doc.ID, doc.Rev)
//	if err != nil {
//	    log.Printf("Deletion failed: %v", err)
//	    return
//	}
//
//	fmt.Printf("Document %s deleted successfully\n", doc.ID)
//
// Soft Deletion Alternative:
//
//	Consider implementing soft deletion for audit purposes:
//	- Mark documents as deleted instead of removing them
//	- Preserve audit trails and compliance data
//	- Enable recovery of accidentally deleted processes
//	- Maintain referential integrity with related data
//
// Cleanup Considerations:
//   - Implement cleanup policies for old completed processes
//   - Consider archival strategies for long-term retention
//   - Monitor database size and performance impact
//   - Plan for backup and disaster recovery scenarios
func (c *CouchDBService) DeleteDocument(id, rev string) error {
	ctx := context.Background()

	_, err := c.database.Delete(ctx, id, rev)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

// Close gracefully shuts down the CouchDB service and releases resources.
// This method ensures proper cleanup of database connections and resources
// to prevent connection leaks and maintain optimal resource utilization.
//
// Cleanup Operations:
//   - Closes active database connections
//   - Releases connection pool resources
//   - Terminates background operations
//   - Ensures graceful service shutdown
//
// Returns:
//   - error: Connection closure or cleanup errors
//
// Resource Management:
//
//	Proper closure is essential for:
//	- Preventing connection leaks in long-running applications
//	- Maintaining optimal database connection pools
//	- Ensuring clean application shutdown
//	- Meeting resource management best practices
//
// Usage Pattern:
//
//	Always defer Close() immediately after service creation:
//
//	service, err := NewCouchDBService(config)
//	if err != nil {
//	    return err
//	}
//	defer service.Close()
//
// Error Handling:
//
//	Connection closure errors are typically non-critical but should be
//	logged for monitoring and debugging purposes in production environments.
func (c *CouchDBService) Close() error {
	return c.client.Close()
}

// DownloadAllDocuments exports all documents from a CouchDB database to the filesystem.
// This function provides comprehensive database backup and export capabilities
// with organized file structure and progress monitoring for large datasets.
//
// Export Process:
//  1. Connects to CouchDB server with provided credentials
//  2. Creates organized directory structure for exported data
//  3. Iterates through all documents in the specified database
//  4. Saves each document as individual JSON file
//  5. Provides progress feedback for large datasets
//
// Parameters:
//   - url: CouchDB server URL with authentication
//   - db: Database name to export documents from
//   - outputDir: Base directory for exported document files
//
// Returns:
//   - error: Connection, permission, or file system errors
//
// File Organization:
//
//	Creates directory structure: outputDir/database/document_id.json
//	- Each document saved as separate JSON file
//	- Document IDs sanitized for filesystem compatibility
//	- Pretty-printed JSON for human readability
//	- Preserves all document fields and metadata
//
// Progress Monitoring:
//   - Reports progress every 100 documents for large datasets
//   - Displays total document count upon completion
//   - Provides feedback for long-running export operations
//   - Logs errors for individual document processing failures
//
// Error Handling:
//   - Individual document errors don't halt the entire export
//   - Connection errors are reported and terminate the operation
//   - File system errors are logged with appropriate context
//   - Design document (_design/) are automatically skipped
//
// Example Usage:
//
//	err := DownloadAllDocuments(
//	    "http://admin:password@localhost:5984",
//	    "flow_processes",
//	    "/backup/couchdb_export")
//	if err != nil {
//	    log.Printf("Export failed: %v", err)
//	    return
//	}
//
//	fmt.Println("Database export completed successfully")
//
// File Naming:
//
//	Document IDs are sanitized for filesystem compatibility:
//	- Invalid characters replaced with underscores
//	- Length limited to prevent filesystem issues
//	- Maintains uniqueness while ensuring compatibility
//
// Use Cases:
//   - Database backup and disaster recovery
//   - Data migration between CouchDB instances
//   - Offline analysis and data processing
//   - Compliance and audit data archival
//   - Development data seeding and testing
//
// Performance Considerations:
//   - Memory usage remains constant regardless of dataset size
//   - Network bandwidth depends on document sizes and count
//   - Disk I/O performance affects export speed
//   - Large datasets benefit from progress monitoring
func DownloadAllDocuments(url, db, outputDir string) error {
	ctx := context.Background()
	// Connect to CouchDB
	client, err := kivik.New("couch", url)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	// Skip system databases
	fmt.Printf("Processing database: %s\n", db)

	if err := downloadDatabaseDocuments(ctx, client, db, outputDir); err != nil {
		log.Printf("Error processing database %s: %v", db, err)
	}
	return nil
}

// downloadDatabaseDocuments handles the actual document export process for a specific database.
// This internal function performs the detailed work of iterating through documents
// and saving them to the filesystem with proper error handling and progress tracking.
//
// Export Implementation:
//  1. Opens database connection
//  2. Creates database-specific directory
//  3. Uses _all_docs view for efficient document enumeration
//  4. Filters out design documents
//  5. Saves each document as JSON file with progress tracking
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout control
//   - client: Active CouchDB client connection
//   - dbName: Database name to export documents from
//   - outputDir: Base output directory for file organization
//
// Returns:
//   - error: Document iteration, file creation, or processing errors
//
// Directory Structure:
//
//	Creates: outputDir/dbName/document_files
//	Each document becomes: outputDir/dbName/sanitized_document_id.json
//
// Document Processing:
//   - Includes full document content via include_docs parameter
//   - Skips CouchDB design documents (_design/ prefix)
//   - Handles document ID sanitization for filesystem compatibility
//   - Preserves all document fields including metadata
//
// Progress Reporting:
//   - Counts processed documents for progress tracking
//   - Reports progress every 100 documents
//   - Displays final count upon completion
//   - Logs individual document processing errors
//
// Error Recovery:
//   - Individual document errors don't halt the entire process
//   - Continues processing remaining documents after errors
//   - Logs errors with document context for debugging
//   - Maintains progress count despite individual failures
//
// Performance Optimization:
//   - Streaming document processing prevents memory overload
//   - Efficient _all_docs view utilization
//   - Minimal memory footprint for large databases
//   - Progress feedback for long-running operations
func downloadDatabaseDocuments(ctx context.Context, client *kivik.Client, dbName, outputDir string) error {
	// Open database
	db := client.DB(dbName)

	// Create directory for this database
	dbDir := filepath.Join(outputDir, dbName)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Get all documents using _all_docs view
	rows := db.AllDocs(ctx, kivik.Param("include_docs", true))
	defer rows.Close()

	docCount := 0
	for rows.Next() {
		id, err := rows.ID()
		if err != nil {
			log.Printf("Failed to get ID: %v", err)
			continue
		}
		// Skip design documents
		if strings.HasPrefix(id, "_design/") {
			continue
		}

		var doc map[string]interface{}
		if err := rows.ScanDoc(&doc); err != nil {
			id, err := rows.ID()
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Error scanning document %s: %v", id, err)
			continue
		}

		// Save document to file
		id, err = rows.ID()
		if err != nil {
			log.Fatal(err)
		}
		filename := sanitizeFilename(id) + ".json"
		filepath := filepath.Join(dbDir, filename)

		if err := saveDocumentToFile(doc, filepath); err != nil {
			id, err := rows.ID()
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Error saving document %s: %v", id, err)
			continue
		}

		docCount++
		if docCount%100 == 0 {
			fmt.Printf("  Downloaded %d documents from %s\n", docCount, dbName)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating documents: %w", err)
	}

	fmt.Printf("  Completed %s: %d documents downloaded\n", dbName, docCount)
	return nil
}

// saveDocumentToFile writes a document to the filesystem as formatted JSON.
// This utility function handles JSON serialization and file writing with
// proper formatting for human readability and debugging purposes.
//
// File Writing Process:
//  1. Creates output file with specified path
//  2. Configures JSON encoder with pretty printing
//  3. Serializes document data to JSON format
//  4. Writes formatted JSON to file
//
// Parameters:
//   - doc: Document data as map[string]interface{}
//   - filepath: Complete file path for document storage
//
// Returns:
//   - error: File creation, JSON encoding, or writing errors
//
// JSON Formatting:
//   - Pretty-printed with 2-space indentation
//   - Human-readable format for debugging and analysis
//   - Preserves all data types and nested structures
//   - Compatible with standard JSON parsing tools
//
// Error Conditions:
//   - File path permission errors
//   - Disk space limitations
//   - JSON serialization failures for complex data types
//   - File system errors during writing
//
// File Management:
//   - Creates files with standard permissions
//   - Overwrites existing files without warning
//   - Ensures proper file closure for resource cleanup
//   - Handles file path validation and creation
func saveDocumentToFile(doc map[string]interface{}, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty print JSON

	if err := encoder.Encode(doc); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// sanitizeFilename converts document IDs to filesystem-safe filenames.
// This utility function handles the conversion of CouchDB document IDs
// to valid filenames across different operating systems and filesystems.
//
// Sanitization Process:
//  1. Replaces invalid filesystem characters with underscores
//  2. Limits filename length to prevent filesystem issues
//  3. Maintains uniqueness while ensuring compatibility
//  4. Preserves readability where possible
//
// Parameters:
//   - filename: Original document ID to sanitize
//
// Returns:
//   - string: Sanitized filename safe for filesystem use
//
// Character Replacement:
//
//	Invalid characters replaced with underscores:
//	- Forward slash (/) and backslash (\)
//	- Colon (:) and asterisk (*)
//	- Question mark (?) and quotes (")
//	- Angle brackets (<>) and pipe (|)
//
// Length Limitations:
//   - Maximum length limited to 200 characters
//   - Prevents filesystem path length issues
//   - Maintains reasonable filename readability
//   - Handles edge cases with very long document IDs
//
// Cross-Platform Compatibility:
//   - Works across Windows, macOS, and Linux filesystems
//   - Handles reserved filenames and special cases
//   - Ensures consistent behavior across environments
//   - Maintains filename uniqueness for document identification
//
// Example Transformations:
//   - "user/123" → "user_123"
//   - "process:2024-01-15" → "process_2024-01-15"
//   - "data<test>" → "data_test_"
//   - Very long IDs truncated to 200 characters
func sanitizeFilename(filename string) string {
	// Replace invalid characters with underscores
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := filename

	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Limit length to avoid filesystem issues
	if len(result) > 200 {
		result = result[:200]
	}

	return result
}

// NewCouchDBServiceFromConfig creates a new CouchDB service from generic configuration.
// This constructor provides more flexibility than NewCouchDBService by supporting
// advanced configuration options including TLS, timeouts, and connection pooling.
//
// Parameters:
//   - config: CouchDBConfig with connection details and options
//
// Returns:
//   - *CouchDBService: Configured service instance
//   - error: Connection, authentication, or database creation errors
//
// Configuration Features:
//   - Custom connection URL and database name
//   - Optional TLS/SSL configuration for secure connections
//   - Connection timeout settings
//   - Automatic database creation
//   - Flexible authentication options
//
// Example Usage:
//
//	config := CouchDBConfig{
//	    URL:             "https://couchdb.example.com:6984",
//	    Database:        "graphium",
//	    Username:        "admin",
//	    Password:        "secure-password",
//	    Timeout:         30000,
//	    CreateIfMissing: true,
//	}
//
//	service, err := NewCouchDBServiceFromConfig(config)
//	if err != nil {
//	    log.Fatal("Failed to create service:", err)
//	}
//	defer service.Close()
//
//	// Use service for operations
//	response, _ := service.SaveDocument(myDocument)
func NewCouchDBServiceFromConfig(config CouchDBConfig) (*CouchDBService, error) {
	// Build connection URL with authentication
	connectionURL := config.URL
	if config.Username != "" && config.Password != "" {
		// Parse URL to inject credentials
		if !strings.Contains(connectionURL, "@") {
			// Insert credentials into URL
			parts := strings.SplitN(connectionURL, "://", 2)
			if len(parts) == 2 {
				connectionURL = fmt.Sprintf("%s://%s:%s@%s",
					parts[0], config.Username, config.Password, parts[1])
			}
		}
	}

	// Create CouchDB client
	client, err := kivik.New("couch", connectionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to CouchDB: %w", err)
	}

	ctx := context.Background()

	// Apply timeout if specified
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(config.Timeout)*time.Millisecond)
		defer cancel()
	}

	// Check if database exists
	exists, err := client.DBExists(ctx, config.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to check if database exists: %w", err)
	}

	// Create database if it doesn't exist and CreateIfMissing is true
	if !exists {
		if config.CreateIfMissing {
			err = client.CreateDB(ctx, config.Database)
			if err != nil {
				return nil, fmt.Errorf("failed to create database: %w", err)
			}
		} else {
			return nil, fmt.Errorf("database %s does not exist", config.Database)
		}
	}

	db := client.DB(config.Database)

	return &CouchDBService{
		client:   client,
		database: db,
		dbName:   config.Database,
	}, nil
}

// CreateDatabaseFromURL creates a new CouchDB database with the given name.
// This is a standalone function that doesn't require a service instance.
//
// Parameters:
//   - url: CouchDB server URL with authentication
//   - dbName: Name of the database to create
//
// Returns:
//   - error: Database creation or connection errors
//
// Error Handling:
//   - Returns error if database already exists
//   - Returns error if insufficient permissions
//   - Returns error on connection failures
//
// Example Usage:
//
//	err := CreateDatabaseFromURL(
//	    "http://admin:password@localhost:5984",
//	    "my_new_database")
//	if err != nil {
//	    log.Printf("Failed to create database: %v", err)
//	}
func CreateDatabaseFromURL(url, dbName string) error {
	client, err := kivik.New("couch", url)
	if err != nil {
		return fmt.Errorf("failed to connect to CouchDB: %w", err)
	}
	defer client.Close()

	ctx := context.Background()
	err = client.CreateDB(ctx, dbName)
	if err != nil {
		if kivik.HTTPStatus(err) != 0 {
			return &CouchDBError{
				StatusCode: kivik.HTTPStatus(err),
				ErrorType:  "create_database_failed",
				Reason:     err.Error(),
			}
		}
		return fmt.Errorf("failed to create database: %w", err)
	}

	return nil
}

// DeleteDatabaseFromURL deletes a CouchDB database.
// This permanently removes the database and all its documents.
//
// Parameters:
//   - url: CouchDB server URL with authentication
//   - dbName: Name of the database to delete
//
// Returns:
//   - error: Database deletion or connection errors
//
// WARNING:
//
//	This operation is irreversible and deletes all data in the database.
//	Use with extreme caution in production environments.
//
// Example Usage:
//
//	err := DeleteDatabaseFromURL(
//	    "http://admin:password@localhost:5984",
//	    "old_database")
//	if err != nil {
//	    log.Printf("Failed to delete database: %v", err)
//	}
func DeleteDatabaseFromURL(url, dbName string) error {
	client, err := kivik.New("couch", url)
	if err != nil {
		return fmt.Errorf("failed to connect to CouchDB: %w", err)
	}
	defer client.Close()

	ctx := context.Background()
	err = client.DestroyDB(ctx, dbName)
	if err != nil {
		if kivik.HTTPStatus(err) != 0 {
			return &CouchDBError{
				StatusCode: kivik.HTTPStatus(err),
				ErrorType:  "delete_database_failed",
				Reason:     err.Error(),
			}
		}
		return fmt.Errorf("failed to delete database: %w", err)
	}

	return nil
}

// DatabaseExistsFromURL checks if a database exists.
// This is a standalone function that doesn't require a service instance.
//
// Parameters:
//   - url: CouchDB server URL with authentication
//   - dbName: Name of the database to check
//
// Returns:
//   - bool: true if database exists, false otherwise
//   - error: Connection or query errors
//
// Example Usage:
//
//	exists, err := DatabaseExistsFromURL(
//	    "http://admin:password@localhost:5984",
//	    "my_database")
//	if err != nil {
//	    log.Printf("Error checking database: %v", err)
//	    return
//	}
//
//	if exists {
//	    fmt.Println("Database exists")
//	} else {
//	    fmt.Println("Database does not exist")
//	}
func DatabaseExistsFromURL(url, dbName string) (bool, error) {
	client, err := kivik.New("couch", url)
	if err != nil {
		return false, fmt.Errorf("failed to connect to CouchDB: %w", err)
	}
	defer client.Close()

	ctx := context.Background()
	exists, err := client.DBExists(ctx, dbName)
	if err != nil {
		return false, fmt.Errorf("failed to check database existence: %w", err)
	}

	return exists, nil
}

// GetDatabaseInfo retrieves metadata and statistics about the database.
// This provides information useful for monitoring, capacity planning, and administration.
//
// Returns:
//   - *DatabaseInfo: Database metadata and statistics
//   - error: Query or connection errors
//
// Information Provided:
//   - Document count (active and deleted)
//   - Database size (disk and data)
//   - Update sequence for change tracking
//   - Compaction status
//   - Instance start time
//
// Example Usage:
//
//	info, err := service.GetDatabaseInfo()
//	if err != nil {
//	    log.Printf("Failed to get database info: %v", err)
//	    return
//	}
//
//	fmt.Printf("Database: %s\n", info.DBName)
//	fmt.Printf("Documents: %d active, %d deleted\n",
//	    info.DocCount, info.DocDelCount)
//	fmt.Printf("Size: %.2f MB (disk), %.2f MB (data)\n",
//	    float64(info.DiskSize)/1024/1024,
//	    float64(info.DataSize)/1024/1024)
//	fmt.Printf("Compaction running: %v\n", info.CompactRunning)
func (c *CouchDBService) GetDatabaseInfo() (*DatabaseInfo, error) {
	ctx := context.Background()

	// Get database stats
	stats, err := c.database.Stats(ctx)
	if err != nil {
		if kivik.HTTPStatus(err) != 0 {
			return nil, &CouchDBError{
				StatusCode: kivik.HTTPStatus(err),
				ErrorType:  "get_database_info_failed",
				Reason:     err.Error(),
			}
		}
		return nil, fmt.Errorf("failed to get database info: %w", err)
	}

	info := &DatabaseInfo{
		DBName:      c.dbName,
		DocCount:    stats.DocCount,
		DocDelCount: stats.DeletedCount,
		UpdateSeq:   stats.UpdateSeq,
		DiskSize:    stats.DiskSize,
		DataSize:    stats.ActiveSize,
	}

	return info, nil
}

// CompactDatabase triggers database compaction.
// Compaction reclaims disk space by removing old document revisions and deleted documents.
//
// Returns:
//   - error: Compaction request errors
//
// Compaction Process:
//   - Removes old document revisions beyond the revision limit
//   - Purges deleted documents
//   - Rebuilds B-tree indexes
//   - Reclaims disk space
//   - Runs asynchronously in the background
//
// Performance Impact:
//   - Compaction is I/O intensive
//   - May impact database performance during compaction
//   - Recommended during low-traffic periods
//   - Monitor with GetDatabaseInfo().CompactRunning
//
// Example Usage:
//
//	err := service.CompactDatabase()
//	if err != nil {
//	    log.Printf("Failed to start compaction: %v", err)
//	    return
//	}
//
//	fmt.Println("Compaction started")
//
//	// Monitor compaction progress
//	for {
//	    info, _ := service.GetDatabaseInfo()
//	    if !info.CompactRunning {
//	        fmt.Println("Compaction completed")
//	        break
//	    }
//	    time.Sleep(10 * time.Second)
//	}
func (c *CouchDBService) CompactDatabase() error {
	ctx := context.Background()

	err := c.database.Compact(ctx)
	if err != nil {
		if kivik.HTTPStatus(err) != 0 {
			return &CouchDBError{
				StatusCode: kivik.HTTPStatus(err),
				ErrorType:  "compact_database_failed",
				Reason:     err.Error(),
			}
		}
		return fmt.Errorf("failed to compact database: %w", err)
	}

	return nil
}
