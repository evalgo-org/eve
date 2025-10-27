package db

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// CouchDBConfig provides generic CouchDB connection configuration.
// This configuration structure supports advanced connection options including
// TLS security, connection pooling, and automatic database creation.
//
// Configuration Options:
//   - URL: CouchDB server URL (e.g., "http://localhost:5984")
//   - Database: Target database name for operations
//   - Username: Authentication username for CouchDB access
//   - Password: Authentication password for secure connections
//   - MaxConnections: Connection pool size for concurrent operations
//   - Timeout: Request timeout in milliseconds
//   - CreateIfMissing: Automatically create database if it doesn't exist
//   - TLS: Optional TLS/SSL configuration for secure connections
//
// Example Usage:
//
//	config := &CouchDBConfig{
//	    URL:             "https://couchdb.example.com:6984",
//	    Database:        "graphium",
//	    Username:        "admin",
//	    Password:        "secure-password",
//	    MaxConnections:  100,
//	    Timeout:         30000,
//	    CreateIfMissing: true,
//	    TLS: &TLSConfig{
//	        Enabled:  true,
//	        CAFile:   "/path/to/ca.crt",
//	        CertFile: "/path/to/client.crt",
//	        KeyFile:  "/path/to/client.key",
//	    },
//	}
type CouchDBConfig struct {
	URL             string     // CouchDB server URL
	Database        string     // Database name
	Username        string     // Authentication username
	Password        string     // Authentication password
	MaxConnections  int        // Maximum number of concurrent connections
	Timeout         int        // Request timeout in milliseconds
	CreateIfMissing bool       // Create database if it doesn't exist
	TLS             *TLSConfig // Optional TLS configuration
}

// TLSConfig provides TLS/SSL configuration for secure CouchDB connections.
// This configuration enables encrypted communication between the client and
// CouchDB server with optional client certificate authentication.
//
// Security Options:
//   - Enabled: Enable TLS/SSL for the connection
//   - CertFile: Client certificate file for mutual TLS authentication
//   - KeyFile: Client private key file for certificate authentication
//   - CAFile: Certificate Authority file for server verification
//   - InsecureSkipVerify: Skip server certificate verification (not recommended)
//
// Example Usage:
//
//	tlsConfig := &TLSConfig{
//	    Enabled:  true,
//	    CAFile:   "/etc/ssl/certs/ca-bundle.crt",
//	    CertFile: "/etc/ssl/certs/client.crt",
//	    KeyFile:  "/etc/ssl/private/client.key",
//	    InsecureSkipVerify: false,
//	}
type TLSConfig struct {
	Enabled            bool   // Enable TLS/SSL
	CertFile           string // Client certificate file path
	KeyFile            string // Client private key file path
	CAFile             string // Certificate Authority file path
	InsecureSkipVerify bool   // Skip certificate verification (development only)
}

// CouchDBError represents a CouchDB-specific error with HTTP status information.
// This error type provides structured error handling with helper methods for
// common CouchDB error conditions like conflicts, not found, and authorization.
//
// Error Fields:
//   - StatusCode: HTTP status code from CouchDB response
//   - ErrorType: Error type identifier (e.g., "conflict", "not_found")
//   - Reason: Human-readable error description
//
// Common Error Types:
//   - 404 Not Found: Document or database doesn't exist
//   - 409 Conflict: Document revision conflict (MVCC)
//   - 401 Unauthorized: Authentication required or failed
//   - 403 Forbidden: Insufficient permissions
//   - 412 Precondition Failed: Missing or invalid revision
//
// Example Usage:
//
//	err := service.GetDocument("missing-doc")
//	if err != nil {
//	    if couchErr, ok := err.(*CouchDBError); ok {
//	        if couchErr.IsNotFound() {
//	            fmt.Println("Document not found")
//	        } else if couchErr.IsConflict() {
//	            fmt.Println("Revision conflict - retry needed")
//	        }
//	    }
//	}
type CouchDBError struct {
	StatusCode int    `json:"status_code"` // HTTP status code
	ErrorType  string `json:"error"`       // Error type identifier
	Reason     string `json:"reason"`      // Human-readable error description
}

// Error implements the error interface for CouchDBError.
// Returns a formatted error message containing status code, error type, and reason.
func (e *CouchDBError) Error() string {
	return fmt.Sprintf("CouchDB error (status %d): %s - %s", e.StatusCode, e.ErrorType, e.Reason)
}

// IsConflict checks if the error is a document conflict error (HTTP 409).
// Conflicts occur when attempting to update a document with an outdated revision,
// indicating that another process has modified the document since it was retrieved.
//
// Returns:
//   - bool: true if this is a revision conflict error, false otherwise
//
// Usage:
//
//	if couchErr.IsConflict() {
//	    // Retrieve latest version and retry
//	    latestDoc, _ := service.GetDocument(docID)
//	    // Merge changes and retry save
//	}
func (e *CouchDBError) IsConflict() bool {
	return e.StatusCode == http.StatusConflict
}

// IsNotFound checks if the error is a not found error (HTTP 404).
// Not found errors occur when attempting to access a document or database
// that doesn't exist in CouchDB.
//
// Returns:
//   - bool: true if this is a not found error, false otherwise
//
// Usage:
//
//	if couchErr.IsNotFound() {
//	    // Create new document instead
//	    service.SaveDocument(newDoc)
//	}
func (e *CouchDBError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

// IsUnauthorized checks if the error is an authorization error (HTTP 401 or 403).
// Authorization errors occur when authentication fails or the authenticated user
// lacks sufficient permissions for the requested operation.
//
// Returns:
//   - bool: true if this is an authorization error, false otherwise
//
// Usage:
//
//	if couchErr.IsUnauthorized() {
//	    // Check credentials or request elevated permissions
//	    log.Println("Authentication or authorization failed")
//	}
func (e *CouchDBError) IsUnauthorized() bool {
	return e.StatusCode == http.StatusUnauthorized || e.StatusCode == http.StatusForbidden
}

// ViewOptions configures parameters for querying CouchDB MapReduce views.
// This structure provides comprehensive control over view query behavior including
// key ranges, document inclusion, pagination, sorting, and reduce function usage.
//
// Query Parameters:
//   - Key: Exact key match for view results
//   - StartKey: Starting key for range queries (inclusive)
//   - EndKey: Ending key for range queries (inclusive)
//   - IncludeDocs: Include full document content with view results
//   - Limit: Maximum number of results to return
//   - Skip: Number of results to skip for pagination
//   - Descending: Reverse result order
//   - Group: Group results by key when using reduce
//   - GroupLevel: Group by key array prefix (for array keys)
//   - Reduce: Execute reduce function (if defined in view)
//
// Example Usage:
//
//	// Query containers by host with full documents
//	opts := ViewOptions{
//	    Key:         "host-123",
//	    IncludeDocs: true,
//	    Limit:       50,
//	}
//	results, _ := service.QueryView("graphium", "containers_by_host", opts)
//
//	// Range query with pagination
//	opts := ViewOptions{
//	    StartKey:   "2024-01-01",
//	    EndKey:     "2024-12-31",
//	    Skip:       100,
//	    Limit:      50,
//	    Descending: false,
//	}
type ViewOptions struct {
	Key         interface{} // Exact key to query
	StartKey    interface{} // Range query start key (inclusive)
	EndKey      interface{} // Range query end key (inclusive)
	IncludeDocs bool        // Include full documents in results
	Limit       int         // Maximum results to return
	Skip        int         // Number of results to skip
	Descending  bool        // Reverse sort order
	Group       bool        // Group results by key
	GroupLevel  int         // Group by key array prefix level
	Reduce      bool        // Execute reduce function
}

// ViewResult contains the results from a CouchDB view query.
// This structure provides metadata about the query results along with
// the actual row data returned from the view.
//
// Result Fields:
//   - TotalRows: Total number of rows in the view (before limit/skip)
//   - Offset: Starting offset for the returned results
//   - Rows: Array of view rows containing key/value/document data
//
// Example Usage:
//
//	result, _ := service.QueryView("graphium", "containers_by_host", opts)
//	fmt.Printf("Found %d total rows, showing %d\n", result.TotalRows, len(result.Rows))
//
//	for _, row := range result.Rows {
//	    fmt.Printf("Key: %v, Value: %v\n", row.Key, row.Value)
//	    if opts.IncludeDocs {
//	        var container Container
//	        json.Unmarshal(row.Doc, &container)
//	    }
//	}
type ViewResult struct {
	TotalRows int       `json:"total_rows"` // Total rows in view
	Offset    int       `json:"offset"`     // Starting offset
	Rows      []ViewRow `json:"rows"`       // Result rows
}

// ViewRow represents a single row from a CouchDB view query result.
// Each row contains the document ID, the emitted key and value from the
// map function, and optionally the full document content.
//
// Row Fields:
//   - ID: Document identifier that emitted this row
//   - Key: Key emitted by the map function
//   - Value: Value emitted by the map function
//   - Doc: Full document content (if IncludeDocs=true)
//
// Map Function Behavior:
//
//	In a MapReduce view, the map function calls emit(key, value):
//	- Key: Determines sort order and enables range queries
//	- Value: Can be the document, a field, or computed value
//	- Multiple emit() calls create multiple rows per document
//
// Example Usage:
//
//	for _, row := range result.Rows {
//	    fmt.Printf("Document %s has key %v\n", row.ID, row.Key)
//
//	    if row.Doc != nil {
//	        var doc map[string]interface{}
//	        json.Unmarshal(row.Doc, &doc)
//	        fmt.Printf("Document content: %+v\n", doc)
//	    }
//	}
type ViewRow struct {
	ID    string          `json:"id"`            // Document ID
	Key   interface{}     `json:"key"`           // Emitted key
	Value interface{}     `json:"value"`         // Emitted value
	Doc   json.RawMessage `json:"doc,omitempty"` // Full document (if IncludeDocs=true)
}

// View represents a CouchDB MapReduce view definition.
// Views enable efficient querying and aggregation of documents through
// JavaScript map and reduce functions.
//
// View Components:
//   - Name: View identifier within the design document
//   - Map: JavaScript function that emits key/value pairs
//   - Reduce: Optional JavaScript function for aggregation
//
// Map Function:
//
//	The map function processes each document and emits key/value pairs:
//	function(doc) {
//	    if (doc.type === 'container') {
//	        emit(doc.hostId, doc);
//	    }
//	}
//
// Reduce Function:
//
//	The reduce function aggregates values for each key:
//	function(keys, values, rereduce) {
//	    return sum(values);
//	}
//
// Built-in Reduce Functions:
//   - _count: Count number of values
//   - _sum: Sum numeric values
//   - _stats: Calculate statistics (sum, count, min, max, etc.)
//
// Example Usage:
//
//	view := View{
//	    Name: "containers_by_host",
//	    Map: `function(doc) {
//	        if (doc['@type'] === 'SoftwareApplication' && doc.hostedOn) {
//	            emit(doc.hostedOn, {name: doc.name, status: doc.status});
//	        }
//	    }`,
//	    Reduce: "_count",
//	}
type View struct {
	Name   string `json:"-"`                // View name (not in JSON)
	Map    string `json:"map"`              // JavaScript map function
	Reduce string `json:"reduce,omitempty"` // JavaScript reduce function (optional)
}

// DesignDoc represents a CouchDB design document containing views.
// Design documents are special documents that contain application logic
// including MapReduce views, validation functions, and show/list functions.
//
// Design Document Structure:
//   - ID: Design document identifier (must start with "_design/")
//   - Language: Programming language for functions (default: "javascript")
//   - Views: Map of view names to view definitions
//
// Naming Convention:
//
//	Design document IDs must have the "_design/" prefix:
//	- "_design/graphium" (correct)
//	- "graphium" (incorrect - will be rejected)
//
// Example Usage:
//
//	designDoc := DesignDoc{
//	    ID:       "_design/graphium",
//	    Language: "javascript",
//	    Views: map[string]View{
//	        "containers_by_host": {
//	            Map: `function(doc) {
//	                if (doc['@type'] === 'SoftwareApplication') {
//	                    emit(doc.hostedOn, doc);
//	                }
//	            }`,
//	        },
//	        "host_container_count": {
//	            Map: `function(doc) {
//	                if (doc['@type'] === 'SoftwareApplication') {
//	                    emit(doc.hostedOn, 1);
//	                }
//	            }`,
//	            Reduce: "_sum",
//	        },
//	    },
//	}
//	service.CreateDesignDoc(designDoc)
type DesignDoc struct {
	ID       string          `json:"_id"`            // Design document ID (must start with "_design/")
	Rev      string          `json:"_rev,omitempty"` // Document revision (for updates)
	Language string          `json:"language"`       // Programming language (typically "javascript")
	Views    map[string]View `json:"views"`          // Map of view names to definitions
}

// MangoQuery represents a CouchDB Mango query (MongoDB-style queries).
// Mango queries provide a declarative JSON-based query language for filtering
// documents without writing MapReduce views.
//
// Query Components:
//   - Selector: MongoDB-style selector with operators ($eq, $gt, $and, etc.)
//   - Fields: Array of field names to return (projection)
//   - Sort: Array of sort specifications
//   - Limit: Maximum number of results
//   - Skip: Number of results to skip for pagination
//   - UseIndex: Hint for which index to use
//
// Selector Operators:
//   - $eq: Equal to
//   - $ne: Not equal to
//   - $gt, $gte: Greater than (or equal)
//   - $lt, $lte: Less than (or equal)
//   - $and, $or, $not: Logical operators
//   - $in, $nin: In array / not in array
//   - $regex: Regular expression match
//   - $exists: Field exists check
//
// Example Usage:
//
//	// Find running containers in us-east datacenter
//	query := MangoQuery{
//	    Selector: map[string]interface{}{
//	        "$and": []interface{}{
//	            map[string]interface{}{"status": "running"},
//	            map[string]interface{}{"location": map[string]interface{}{
//	                "$regex": "^us-east",
//	            }},
//	        },
//	    },
//	    Fields: []string{"_id", "name", "status", "hostedOn"},
//	    Sort: []map[string]string{
//	        {"name": "asc"},
//	    },
//	    Limit: 100,
//	}
//	results, _ := service.Find(query)
type MangoQuery struct {
	Selector map[string]interface{} `json:"selector"`            // MongoDB-style selector
	Fields   []string               `json:"fields,omitempty"`    // Fields to return
	Sort     []map[string]string    `json:"sort,omitempty"`      // Sort specifications
	Limit    int                    `json:"limit,omitempty"`     // Maximum results
	Skip     int                    `json:"skip,omitempty"`      // Pagination offset
	UseIndex string                 `json:"use_index,omitempty"` // Index hint
}

// Index represents a CouchDB index for query optimization.
// Indexes improve query performance by maintaining sorted data structures
// for frequently queried fields.
//
// Index Types:
//   - "json": Standard JSON index for Mango queries (default)
//   - "text": Full-text search index (requires special queries)
//
// Index Components:
//   - Name: Index identifier for management and hints
//   - Fields: Array of field names to index
//   - Type: Index type ("json" or "text")
//
// Index Usage:
//
//	Indexes are automatically used by Mango queries when the query
//	selector matches the indexed fields. Explicit index selection
//	is possible via the UseIndex field in MangoQuery.
//
// Example Usage:
//
//	// Create compound index for common query pattern
//	index := Index{
//	    Name:   "status-location-index",
//	    Fields: []string{"status", "location"},
//	    Type:   "json",
//	}
//	service.CreateIndex(index)
//
//	// Query will automatically use the index
//	query := MangoQuery{
//	    Selector: map[string]interface{}{
//	        "status": "running",
//	        "location": "us-east-1",
//	    },
//	}
type Index struct {
	Name   string   `json:"name"`   // Index name
	Fields []string `json:"fields"` // Fields to index
	Type   string   `json:"type"`   // Index type: "json" or "text"
}

// BulkResult represents the result of a single document operation in a bulk request.
// Bulk operations return an array of results, one for each document processed.
//
// Result Fields:
//   - ID: Document identifier
//   - Rev: New document revision (on success)
//   - Error: Error type if operation failed
//   - Reason: Error description if operation failed
//   - OK: Boolean indicating success or failure
//
// Success Indicators:
//   - OK=true: Operation succeeded, Rev contains new revision
//   - OK=false: Operation failed, Error and Reason explain why
//
// Common Errors:
//   - "conflict": Document revision conflict
//   - "forbidden": Insufficient permissions
//   - "not_found": Document doesn't exist (for updates)
//
// Example Usage:
//
//	results, _ := service.BulkSaveDocuments(docs)
//	for _, result := range results {
//	    if result.OK {
//	        fmt.Printf("Saved %s with rev %s\n", result.ID, result.Rev)
//	    } else {
//	        fmt.Printf("Failed %s: %s - %s\n", result.ID, result.Error, result.Reason)
//	    }
//	}
type BulkResult struct {
	ID     string `json:"id"`               // Document ID
	Rev    string `json:"rev,omitempty"`    // New revision (success)
	Error  string `json:"error,omitempty"`  // Error type (failure)
	Reason string `json:"reason,omitempty"` // Error description (failure)
	OK     bool   `json:"ok"`               // Success indicator
}

// BulkDeleteDoc represents a document to be deleted in a bulk operation.
// Bulk deletions require the document ID, current revision, and deleted flag.
//
// Fields:
//   - ID: Document identifier to delete
//   - Rev: Current document revision for conflict detection
//   - Deleted: Must be true to indicate deletion
//
// Example Usage:
//
//	deleteOps := []BulkDeleteDoc{
//	    {ID: "doc1", Rev: "1-abc", Deleted: true},
//	    {ID: "doc2", Rev: "2-def", Deleted: true},
//	}
//	results, _ := service.BulkDeleteDocuments(deleteOps)
type BulkDeleteDoc struct {
	ID      string `json:"_id"`      // Document ID
	Rev     string `json:"_rev"`     // Current revision
	Deleted bool   `json:"_deleted"` // Deletion flag (must be true)
}

// ChangesFeedOptions configures the CouchDB changes feed for real-time updates.
// The changes feed provides notification of all document modifications in the database.
//
// Feed Types:
//   - "normal": Return all changes since sequence and close
//   - "longpoll": Wait for changes, return when available, close
//   - "continuous": Keep connection open, stream changes indefinitely
//
// Configuration Options:
//   - Since: Starting sequence ("now", "0", or specific sequence ID)
//   - Feed: Feed type for change delivery
//   - Filter: Server-side filter function name
//   - IncludeDocs: Include full document content with changes
//   - Heartbeat: Milliseconds between heartbeat signals
//   - Timeout: Request timeout in milliseconds
//   - Limit: Maximum number of changes to return
//   - Descending: Reverse chronological order
//   - Selector: Mango selector for filtering changes
//
// Example Usage:
//
//	// Continuous monitoring from current point
//	opts := ChangesFeedOptions{
//	    Since:       "now",
//	    Feed:        "continuous",
//	    IncludeDocs: true,
//	    Heartbeat:   60000,
//	    Selector: map[string]interface{}{
//	        "type": "container",
//	    },
//	}
//	service.ListenChanges(opts, func(change Change) {
//	    fmt.Printf("Document %s changed\n", change.ID)
//	})
type ChangesFeedOptions struct {
	Since       string                 `json:"since,omitempty"`        // Starting sequence
	Feed        string                 `json:"feed,omitempty"`         // Feed type
	Filter      string                 `json:"filter,omitempty"`       // Filter function
	IncludeDocs bool                   `json:"include_docs,omitempty"` // Include documents
	Heartbeat   int                    `json:"heartbeat,omitempty"`    // Heartbeat interval (ms)
	Timeout     int                    `json:"timeout,omitempty"`      // Request timeout (ms)
	Limit       int                    `json:"limit,omitempty"`        // Maximum changes
	Descending  bool                   `json:"descending,omitempty"`   // Reverse order
	Selector    map[string]interface{} `json:"selector,omitempty"`     // Mango selector filter
}

// Change represents a single change notification from the changes feed.
// Each change indicates a document modification (create, update, or delete).
//
// Change Fields:
//   - Seq: Change sequence identifier (for resuming)
//   - ID: Document identifier that changed
//   - Changes: Array of revision changes
//   - Deleted: True if document was deleted
//   - Doc: Full document content (if IncludeDocs=true)
//
// Sequence Numbers:
//
//	Sequence IDs are opaque strings that uniquely identify each change:
//	- Used to resume changes feed after interruption
//	- Monotonically increasing (newer changes have higher sequences)
//	- Format varies by CouchDB version and configuration
//
// Example Usage:
//
//	service.ListenChanges(opts, func(change Change) {
//	    if change.Deleted {
//	        fmt.Printf("Document %s was deleted\n", change.ID)
//	    } else {
//	        fmt.Printf("Document %s updated to rev %s\n",
//	            change.ID, change.Changes[0].Rev)
//
//	        if change.Doc != nil {
//	            var doc map[string]interface{}
//	            json.Unmarshal(change.Doc, &doc)
//	            // Process document
//	        }
//	    }
//	})
type Change struct {
	Seq     string          `json:"seq"`               // Sequence ID
	ID      string          `json:"id"`                // Document ID
	Changes []ChangeRev     `json:"changes"`           // Revision changes
	Deleted bool            `json:"deleted,omitempty"` // Deletion flag
	Doc     json.RawMessage `json:"doc,omitempty"`     // Document content
}

// ChangeRev represents a document revision in a change notification.
// Each change can include multiple revisions in conflict scenarios.
//
// Fields:
//   - Rev: Document revision identifier
//
// Example Usage:
//
//	for _, change := range changes {
//	    for _, rev := range change.Changes {
//	        fmt.Printf("Revision: %s\n", rev.Rev)
//	    }
//	}
type ChangeRev struct {
	Rev string `json:"rev"` // Revision ID
}

// TraversalOptions configures graph traversal operations for following relationships.
// Traversal allows navigation through document relationships like container → host → datacenter.
//
// Configuration Options:
//   - StartID: Document ID to begin traversal from
//   - Depth: Maximum relationship hops to traverse
//   - RelationField: Document field containing relationship reference
//   - Direction: "forward" (follow refs) or "reverse" (find referring docs)
//   - Filter: Additional filters for traversed documents
//
// Traversal Directions:
//
//	Forward: Follow relationship references in documents
//	- Start: Container document
//	- Field: "hostedOn"
//	- Result: Host documents referenced by containers
//
//	Reverse: Find documents that reference this one
//	- Start: Host document
//	- Field: "hostedOn"
//	- Result: Container documents that reference this host
//
// Example Usage:
//
//	// Find all containers on a host and their networks
//	opts := TraversalOptions{
//	    StartID:       "host-123",
//	    Depth:         2,
//	    RelationField: "hostedOn",
//	    Direction:     "reverse",
//	    Filter: map[string]interface{}{
//	        "status": "running",
//	    },
//	}
//	results, _ := service.Traverse(opts)
type TraversalOptions struct {
	StartID       string                 `json:"start_id"`         // Starting document ID
	Depth         int                    `json:"depth"`            // Traversal depth
	RelationField string                 `json:"relation_field"`   // Relationship field name
	Direction     string                 `json:"direction"`        // "forward" or "reverse"
	Filter        map[string]interface{} `json:"filter,omitempty"` // Optional filters
}

// DatabaseInfo contains metadata about a CouchDB database.
// This structure provides statistics and status information for database
// monitoring, capacity planning, and administrative operations.
//
// Information Fields:
//   - DBName: Database name
//   - DocCount: Number of non-deleted documents
//   - DocDelCount: Number of deleted documents
//   - UpdateSeq: Current update sequence
//   - PurgeSeq: Current purge sequence
//   - CompactRunning: True if compaction is in progress
//   - DiskSize: Total disk space used (bytes)
//   - DataSize: Actual data size excluding overhead (bytes)
//   - InstanceStartTime: Database instance start timestamp
//
// Example Usage:
//
//	info, _ := service.GetDatabaseInfo()
//	fmt.Printf("Database: %s\n", info.DBName)
//	fmt.Printf("Documents: %d (+ %d deleted)\n", info.DocCount, info.DocDelCount)
//	fmt.Printf("Disk usage: %.2f MB\n", float64(info.DiskSize)/1024/1024)
//	fmt.Printf("Data size: %.2f MB\n", float64(info.DataSize)/1024/1024)
//
//	if info.CompactRunning {
//	    fmt.Println("Compaction in progress")
//	}
type DatabaseInfo struct {
	DBName            string `json:"db_name"`             // Database name
	DocCount          int64  `json:"doc_count"`           // Non-deleted document count
	DocDelCount       int64  `json:"doc_del_count"`       // Deleted document count
	UpdateSeq         string `json:"update_seq"`          // Current update sequence
	PurgeSeq          int64  `json:"purge_seq"`           // Purge sequence
	CompactRunning    bool   `json:"compact_running"`     // Compaction status
	DiskSize          int64  `json:"disk_size"`           // Disk space used (bytes)
	DataSize          int64  `json:"data_size"`           // Data size (bytes)
	InstanceStartTime string `json:"instance_start_time"` // Instance start time
}
