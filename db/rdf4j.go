// Package db provides comprehensive RDF4J server integration for semantic data management.
// This package implements a complete client library for interacting with RDF4J triple stores,
// supporting repository management, RDF data import/export, and SPARQL query operations.
//
// RDF4J Integration:
//
//	RDF4J is a powerful Java framework for processing RDF data, providing:
//	- Multiple storage backends (memory, native, LMDB)
//	- SPARQL 1.1 query and update support
//	- RDF serialization format support (RDF/XML, Turtle, JSON-LD, N-Triples)
//	- Repository management and configuration
//	- Transaction support and concurrent access
//
// Semantic Data Management:
//
//	The package enables working with semantic web technologies:
//	- RDF triple storage and retrieval
//	- Ontology and schema management
//	- Knowledge graph construction and querying
//	- Linked data publishing and consumption
//	- Reasoning and inference capabilities
//
// Repository Types:
//
//	Supports multiple RDF4J repository configurations:
//	- Memory stores for fast, temporary data processing
//	- Native stores for persistent, file-based storage
//	- LMDB stores for high-performance persistent storage
//	- Remote repository connections for distributed setups
//
// Authentication and Security:
//
//	All operations support HTTP Basic Authentication for secure access
//	to RDF4J servers, enabling integration with enterprise authentication
//	systems and access control policies.
//
// Data Format Support:
//   - RDF/XML for W3C standard compatibility
//   - Turtle for human-readable RDF serialization
//   - JSON-LD for web-friendly linked data
//   - N-Triples for simple triple streaming
//   - SPARQL Results JSON for query responses
package db

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"unicode/utf8"
)

// sparqlValue represents a single value in a SPARQL query result.
// This structure encapsulates the type and value information for
// RDF literals, URIs, and blank nodes returned by SPARQL queries.
//
// SPARQL Value Types:
//   - "uri": Resource identifiers (IRIs)
//   - "literal": String literals with optional language tags or datatypes
//   - "bnode": Blank nodes (anonymous resources)
//   - "typed-literal": Literals with explicit datatype URIs
//
// The Type field indicates the RDF term type while Value contains
// the actual string representation of the term.
type sparqlValue struct {
	Type  string `json:"type"`  // RDF term type (uri, literal, bnode)
	Value string `json:"value"` // String representation of the value
}

// sparqlResult contains the bindings section of a SPARQL SELECT query response.
// Each binding maps variable names to their corresponding values, enabling
// structured access to query results with proper type information.
//
// Structure:
//
//	The Bindings array contains one map per result row, where each map
//	associates SPARQL variable names (without the ? prefix) with their
//	corresponding sparqlValue structures.
//
// Example:
//
//	For a query "SELECT ?subject ?predicate WHERE { ... }", a binding might be:
//	{
//	  "subject": {"type": "uri", "value": "http://example.org/person1"},
//	  "predicate": {"type": "uri", "value": "http://xmlns.com/foaf/0.1/name"}
//	}
type sparqlResult struct {
	Bindings []map[string]sparqlValue `json:"bindings"` // Query result bindings
}

// sparqlResponse represents the complete response from a SPARQL SELECT query.
// This structure follows the W3C SPARQL Query Results JSON Format specification,
// providing both metadata about the query variables and the actual result data.
//
// Components:
//   - Head: Contains metadata including variable names used in the query
//   - Results: Contains the actual query results with variable bindings
//
// The Head section provides information about the query structure while
// Results contains the data returned by the query execution.
type sparqlResponse struct {
	Head    map[string][]string `json:"head"`    // Query metadata (variables, etc.)
	Results sparqlResult        `json:"results"` // Query results with bindings
}

// Repository represents an RDF4J repository configuration and metadata.
// This structure provides essential information about repositories available
// on an RDF4J server, including identification, description, and storage type.
//
// Repository Attributes:
//   - ID: Unique identifier used in API endpoints and configuration
//   - Title: Human-readable description for management interfaces
//   - Type: Storage backend type (memory, native, LMDB, etc.)
//
// The repository serves as the primary container for RDF data and provides
// the context for all SPARQL operations and data management functions.
//
// Repository Types:
//   - "memory": In-memory storage (fast, non-persistent)
//   - "native": File-based persistent storage
//   - "lmdb": LMDB-based high-performance storage
//   - "remote": Proxy to remote repositories
//   - "federation": Federated access to multiple repositories
type Repository struct {
	ID    string // Unique repository identifier
	Title string // Human-readable repository name
	Type  string // Repository storage type
}

// ListRepositories retrieves all available repositories from an RDF4J server.
// This function queries the server's repository management API to discover
// available data stores and their configurations.
//
// Server Discovery:
//
//	The function connects to the RDF4J server's repository endpoint to
//	retrieve metadata about all configured repositories, including both
//	system repositories and user-created data stores.
//
// Parameters:
//   - serverURL: Base URL of the RDF4J server (e.g., "http://localhost:8080/rdf4j-server")
//   - username: Username for HTTP Basic Authentication
//   - password: Password for HTTP Basic Authentication
//
// Returns:
//   - []Repository: Array of repository metadata structures
//   - error: HTTP communication, authentication, or parsing errors
//
// Authentication:
//
//	Uses HTTP Basic Authentication to access the repository listing endpoint.
//	The credentials must have appropriate permissions to read repository
//	metadata from the RDF4J server.
//
// Response Format:
//
//	The function expects SPARQL JSON results format from the server,
//	containing repository information with id, title, and type bindings
//	for each available repository.
//
// Error Conditions:
//   - Network connectivity issues to the RDF4J server
//   - Authentication failures with provided credentials
//   - Server errors or invalid responses
//   - JSON parsing errors in response data
//   - Repository access permission denied
//
// Example Usage:
//
//	repos, err := ListRepositories(
//	    "http://localhost:8080/rdf4j-server",
//	    "admin",
//	    "password",
//	)
//	if err != nil {
//	    log.Fatal("Failed to list repositories:", err)
//	}
//
//	for _, repo := range repos {
//	    fmt.Printf("Repository: %s (%s) - %s\n", repo.ID, repo.Type, repo.Title)
//	}
//
// Repository Management:
//
//	The returned repository list can be used for:
//	- Dynamic repository selection in applications
//	- Repository health monitoring and status checking
//	- Administrative interfaces for repository management
//	- Automated repository discovery and configuration
func ListRepositories(serverURL, username, password string) ([]Repository, error) {
	client := &http.Client{}
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/repositories", serverURL),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.SetBasicAuth(username, password)
	req.Header.Set("Accept", "application/sparql-results+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories. Status: %s, read error: %v", resp.Status, err)
		}
		return nil, fmt.Errorf("failed to list repositories. Status: %s, Body: %s", resp.Status, string(body))
	}

	var data sparqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var repos []Repository
	for _, binding := range data.Results.Bindings {
		repos = append(repos, Repository{
			ID:    binding["id"].Value,
			Title: binding["title"].Value,
			Type:  binding["type"].Value,
		})
	}

	return repos, nil
}

// stripBOM removes UTF-8 Byte Order Mark (BOM) from byte data.
// This utility function handles files that may contain UTF-8 BOM sequences,
// which can interfere with RDF parsing and cause validation errors.
//
// BOM Detection:
//
//	UTF-8 BOM consists of the byte sequence 0xEF, 0xBB, 0xBF at the
//	beginning of a file. This function checks for this sequence and
//	removes it if present.
//
// Parameters:
//   - data: Input byte array that may contain BOM
//
// Returns:
//   - []byte: Data with BOM removed if it was present, otherwise original data
//
// Use Cases:
//   - Processing RDF files created by Windows text editors
//   - Handling RDF data from web services that add BOM
//   - Ensuring clean UTF-8 data for RDF parsers
//   - Preprocessing files before RDF validation
//
// RDF Parser Compatibility:
//
//	Many RDF parsers are sensitive to BOM sequences and may fail
//	to parse otherwise valid RDF data. This function ensures
//	compatibility by removing problematic BOM sequences.
func stripBOM(data []byte) []byte {
	// UTF-8 BOM is 0xEF,0xBB,0xBF
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
}

// ImportRDF imports RDF data from a file into an RDF4J repository.
// This function uploads RDF data in various serialization formats to
// a specified repository, handling encoding validation and content negotiation.
//
// Import Process:
//  1. Reads RDF data from the specified file path
//  2. Removes UTF-8 BOM if present to ensure parser compatibility
//  3. Validates UTF-8 encoding to prevent parser errors
//  4. Uploads data to the repository via HTTP POST
//  5. Returns server response for status verification
//
// Parameters:
//   - serverURL: Base URL of the RDF4J server
//   - repositoryID: Target repository identifier
//   - username: Username for HTTP Basic Authentication
//   - password: Password for HTTP Basic Authentication
//   - rdfFilePath: File system path to the RDF data file
//   - contentType: MIME type for the RDF serialization format
//
// Returns:
//   - []byte: Raw response body from the server
//   - error: File reading, encoding validation, or HTTP errors
//
// Supported Content Types:
//   - "application/rdf+xml": RDF/XML format
//   - "text/turtle": Turtle format
//   - "application/ld+json": JSON-LD format
//   - "application/n-triples": N-Triples format
//   - "application/n-quads": N-Quads format
//
// File Validation:
//
//	The function performs UTF-8 validation to ensure the RDF file
//	contains valid Unicode text. Invalid UTF-8 sequences will cause
//	the function to return an error before attempting upload.
//
// Error Conditions:
//   - File not found or permission denied
//   - Invalid UTF-8 encoding in the file
//   - Network connectivity issues
//   - Authentication failures
//   - Repository not found or access denied
//   - Invalid RDF syntax (server-side validation)
//
// Example Usage:
//
//	response, err := ImportRDF(
//	    "http://localhost:8080/rdf4j-server",
//	    "my-repository",
//	    "admin",
//	    "password",
//	    "/path/to/data.rdf",
//	    "application/rdf+xml",
//	)
//	if err != nil {
//	    log.Fatal("Import failed:", err)
//	}
//
//	fmt.Printf("Server response: %s\n", string(response))
//
// Transaction Behavior:
//
//	The import operation is typically atomic at the repository level,
//	meaning either all triples are imported successfully or none are
//	added in case of errors during processing.
//
// Performance Considerations:
//   - Large files are uploaded entirely into memory before sending
//   - Consider chunking or streaming for very large datasets
//   - Server timeout settings may affect large imports
//   - Repository performance depends on storage backend type
func ImportRDF(serverURL, repositoryID, username, password, rdfFilePath, contentType string) ([]byte, error) {
	// Read the RDF file from filesystem
	rdfData, err := os.ReadFile(rdfFilePath)
	if err != nil {
		return nil, err
	}

	// Strip BOM if present to ensure parser compatibility
	rdfData = stripBOM(rdfData)

	// Validate UTF-8 encoding to prevent parser errors
	if !utf8.Valid(rdfData) {
		return nil, errors.New("invalid UTF-8 file")
	}

	// Create HTTP client and request for data upload
	client := &http.Client{}
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/repositories/%s/statements", serverURL, repositoryID),
		bytes.NewReader(rdfData),
	)
	if err != nil {
		return nil, err
	}

	// Set content type for RDF serialization format
	req.Header.Set("Content-Type", contentType)

	// Configure HTTP Basic Authentication
	req.SetBasicAuth(username, password)

	// Send the import request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read and return server response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}

// ExportRDFXml exports all RDF data from a repository to a file.
// This function retrieves complete repository contents and saves them
// in the specified RDF serialization format for backup, transfer, or analysis.
//
// Export Process:
//  1. Connects to the repository statements endpoint
//  2. Requests data in the specified serialization format
//  3. Downloads all triples from the repository
//  4. Writes the data to the specified output file
//
// Parameters:
//   - serverURL: Base URL of the RDF4J server
//   - repositoryID: Source repository identifier
//   - username: Username for HTTP Basic Authentication
//   - password: Password for HTTP Basic Authentication
//   - outputFilePath: File system path for the exported data
//   - contentType: MIME type for the desired output format
//
// Returns:
//   - error: Network, authentication, or file writing errors
//
// Supported Export Formats:
//   - "application/rdf+xml": Standard RDF/XML serialization
//   - "text/turtle": Turtle format (human-readable)
//   - "application/ld+json": JSON-LD format (web-friendly)
//   - "application/n-triples": N-Triples format (streaming-friendly)
//   - "text/plain": N-Triples in plain text format
//
// Data Scope:
//
//	The export includes all triples stored in the repository across
//	all named graphs, providing a complete dump of repository contents.
//	Named graph information is preserved in formats that support it.
//
// File Handling:
//
//	The exported file is written with standard permissions (0644),
//	making it readable by the owner and group while preventing
//	unauthorized modifications.
//
// Error Conditions:
//   - Repository not found or access denied
//   - Network connectivity issues during download
//   - Authentication failures
//   - Insufficient disk space for export file
//   - File system permission errors
//   - Server-side errors during data serialization
//
// Example Usage:
//
//	err := ExportRDFXml(
//	    "http://localhost:8080/rdf4j-server",
//	    "my-repository",
//	    "admin",
//	    "password",
//	    "/backup/repository-export.rdf",
//	    "application/rdf+xml",
//	)
//	if err != nil {
//	    log.Fatal("Export failed:", err)
//	}
//
// Backup and Recovery:
//
//	Exported files can be used for:
//	- Repository backup and disaster recovery
//	- Data migration between different RDF stores
//	- Data analysis and processing with external tools
//	- Version control and change tracking
//
// Performance Considerations:
//   - Large repositories may take significant time to export
//   - Memory usage depends on repository size and export format
//   - Network bandwidth affects download time
//   - Consider compression for large export files
func ExportRDFXml(serverURL, repositoryID, username, password, outputFilePath, contentType string) error {
	client := &http.Client{}
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/repositories/%s/statements", serverURL, repositoryID),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Accept", contentType)
	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to export data. Status: %s, read error: %v", resp.Status, err)
		}
		return fmt.Errorf("failed to export data. Status: %s, Body: %s", resp.Status, string(body))
	}

	rdfData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if err := os.WriteFile(outputFilePath, rdfData, 0644); err != nil {
		return fmt.Errorf("failed to write RDF data to file: %w", err)
	}

	return nil
}

// DeleteRepository removes a repository and all its data from an RDF4J server.
// This function permanently deletes a repository configuration and all
// stored RDF data, providing a clean removal mechanism for repository management.
//
// Deletion Process:
//  1. Sends HTTP DELETE request to the repository endpoint
//  2. Authenticates using provided credentials
//  3. Verifies successful deletion via HTTP status codes
//  4. Returns error information for failed deletions
//
// Parameters:
//   - serverURL: Base URL of the RDF4J server
//   - repositoryID: Identifier of the repository to delete
//   - username: Username for HTTP Basic Authentication
//   - password: Password for HTTP Basic Authentication
//
// Returns:
//   - error: Authentication, permission, or deletion errors
//
// Data Loss Warning:
//
//	This operation is irreversible and will permanently destroy:
//	- All RDF triples stored in the repository
//	- Repository configuration and metadata
//	- Any custom indexes or optimization data
//	- Transaction logs and backup information
//
// Success Conditions:
//
//	The function considers deletion successful when the server returns:
//	- HTTP 204 No Content (standard success response)
//	- HTTP 200 OK (alternative success response)
//
// Security Considerations:
//   - Requires appropriate authentication credentials
//   - User must have repository deletion permissions
//   - Consider implementing confirmation mechanisms in applications
//   - Audit logging recommended for deletion operations
//
// Error Conditions:
//   - Repository not found (may already be deleted)
//   - Authentication failures or insufficient permissions
//   - Server-side errors during deletion process
//   - Network connectivity issues
//   - Repository currently in use by other operations
//
// Example Usage:
//
//	err := DeleteRepository(
//	    "http://localhost:8080/rdf4j-server",
//	    "temporary-repository",
//	    "admin",
//	    "password",
//	)
//	if err != nil {
//	    log.Fatal("Deletion failed:", err)
//	}
//
//	log.Println("Repository deleted successfully")
//
// Repository Management:
//
//	Use this function as part of:
//	- Automated testing cleanup procedures
//	- Repository lifecycle management
//	- Data migration and reorganization
//	- Administrative maintenance tasks
//
// Best Practices:
//   - Always backup important data before deletion
//   - Verify repository ID to prevent accidental deletions
//   - Implement proper access controls and audit logs
//   - Consider soft deletion patterns for critical systems
func DeleteRepository(serverURL, repositoryID, username, password string) error {
	client := &http.Client{}
	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/repositories/%s", serverURL, repositoryID),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("HTTP error. Status: %s, read error: %v", resp.Status, err)
		}
		return fmt.Errorf("failed to delete repository. Status: %s, Body: %s", resp.Status, string(body))
	}

	return nil
}

// CreateRepository creates a new in-memory RDF4J repository.
// This function dynamically creates a repository configuration using
// Turtle syntax and deploys it to the RDF4J server for immediate use.
//
// Repository Configuration:
//
//	Creates an in-memory repository with the following characteristics:
//	- Memory-based storage (non-persistent)
//	- Fast read/write operations
//	- Automatic cleanup on server restart
//	- Suitable for temporary data processing and testing
//
// Parameters:
//   - serverURL: Base URL of the RDF4J server
//   - repositoryID: Unique identifier for the new repository
//   - username: Username for HTTP Basic Authentication
//   - password: Password for HTTP Basic Authentication
//
// Returns:
//   - error: Repository creation, authentication, or configuration errors
//
// Configuration Format:
//
//	The function generates a Turtle configuration that defines:
//	- Repository type as SailRepository with MemoryStore
//	- Repository ID and human-readable label
//	- Storage backend configuration and parameters
//
// Memory Store Characteristics:
//   - All data stored in server memory (RAM)
//   - No persistence across server restarts
//   - Excellent performance for read/write operations
//   - Limited by available server memory
//   - Immediate data loss on server failure
//
// Success Conditions:
//
//	Repository creation is successful when the server returns:
//	- HTTP 204 No Content (creation successful)
//	- HTTP 200 OK (alternative success response)
//
// Error Conditions:
//   - Repository ID already exists on the server
//   - Authentication failures or insufficient permissions
//   - Invalid repository configuration syntax
//   - Server-side errors during repository initialization
//   - Network connectivity issues
//
// Example Usage:
//
//	err := CreateRepository(
//	    "http://localhost:8080/rdf4j-server",
//	    "test-memory-repo",
//	    "admin",
//	    "password",
//	)
//	if err != nil {
//	    log.Fatal("Repository creation failed:", err)
//	}
//
//	log.Println("Memory repository created successfully")
//
// Use Cases:
//
//	Memory repositories are ideal for:
//	- Unit testing and integration testing
//	- Temporary data processing and analysis
//	- Development and prototyping
//	- Cache-like storage for computed results
//	- Session-based data storage
//
// Repository Lifecycle:
//   - Created empty and ready for data import
//   - Destroyed automatically on server restart
//   - Can be explicitly deleted using DeleteRepository
//   - Performance degrades as data size approaches memory limits
func CreateRepository(serverURL, repositoryID, username, password string) error {
	client := &http.Client{}

	repoConfigTurtle := fmt.Sprintf(`
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#>.
@prefix rep: <http://www.openrdf.org/config/repository#>.
@prefix sr: <http://www.openrdf.org/config/repository/sail#>.
@prefix sail: <http://www.openrdf.org/config/sail#>.
@prefix mem: <http://www.openrdf.org/config/sail/memory#>.

[] a rep:Repository ;
   rep:repositoryID "%s" ;
   rdfs:label "Memory Store for %s" ;
   rep:repositoryImpl [
      rep:repositoryType "openrdf:SailRepository" ;
      sr:sailImpl [
         sail:sailType "openrdf:MemoryStore"
      ]
   ].`, repositoryID, repositoryID)

	url := fmt.Sprintf("%s/repositories/%s", serverURL, repositoryID)

	req, err := http.NewRequest("PUT", url, bytes.NewBufferString(repoConfigTurtle))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/turtle")
	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("failed to create repository. Status: %d, read error: %v", resp.StatusCode, readErr)
		}
		return fmt.Errorf("failed to create repository. Status: %d , Body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// CreateLMDBRepository creates a new LMDB-based RDF4J repository.
// This function creates a high-performance persistent repository using
// Lightning Memory-Mapped Database (LMDB) as the storage backend.
//
// LMDB Storage Benefits:
//   - High-performance read and write operations
//   - Memory-mapped file access for efficiency
//   - ACID transaction support with durability
//   - Crash-safe persistence with automatic recovery
//   - Excellent scalability for large datasets
//
// Repository Configuration:
//
//	Creates an LMDB repository with the following characteristics:
//	- File-based persistent storage
//	- Standard query evaluation mode
//	- Automatic indexing and optimization
//	- Full SPARQL 1.1 support
//
// Parameters:
//   - serverURL: Base URL of the RDF4J server
//   - repositoryID: Unique identifier for the new repository
//   - username: Username for HTTP Basic Authentication
//   - password: Password for HTTP Basic Authentication
//
// Returns:
//   - error: Repository creation, authentication, or configuration errors
//
// LMDB Characteristics:
//   - Memory-mapped files for efficient access
//   - Copy-on-write semantics for consistent reads
//   - No write-ahead logging overhead
//   - Automatic file growth and management
//   - Operating system page cache integration
//
// Performance Profile:
//   - Excellent read performance through memory mapping
//   - Good write performance with batch operations
//   - Efficient range queries and indexing
//   - Low memory overhead for large datasets
//   - Scales well with available system memory
//
// Success Conditions:
//
//	Repository creation is successful when the server returns:
//	- HTTP 200 OK (creation successful)
//	- HTTP 201 Created (alternative success response)
//	- HTTP 204 No Content (creation without response body)
//
// Error Conditions:
//   - Repository ID already exists on the server
//   - Authentication failures or insufficient permissions
//   - Invalid LMDB configuration parameters
//   - Insufficient disk space for database files
//   - File system permission errors
//   - Server-side LMDB initialization failures
//
// Example Usage:
//
//	err := CreateLMDBRepository(
//	    "http://localhost:8080/rdf4j-server",
//	    "production-lmdb-repo",
//	    "admin",
//	    "password",
//	)
//	if err != nil {
//	    log.Fatal("LMDB repository creation failed:", err)
//	}
//
//	log.Println("LMDB repository created successfully")
//
// Use Cases:
//
//	LMDB repositories are ideal for:
//	- Production systems requiring persistence
//	- Large-scale data processing and analytics
//	- High-throughput read-heavy workloads
//	- Systems requiring fast startup times
//	- Applications with strict consistency requirements
//
// Configuration Details:
//
//	The function uses RDF4J's modern configuration vocabulary
//	(tag:rdf4j.org,2023:config/) rather than legacy OpenRDF namespaces
//	for future compatibility and enhanced functionality.
//
// Storage Considerations:
//   - Database files created in RDF4J server data directory
//   - Automatic file size management and growth
//   - Backup requires file system level copying
//   - Consider disk I/O performance for optimal results
func CreateLMDBRepository(serverURL, repositoryID, username, password string) error {
	// Turtle configuration for LMDB repository
	config := fmt.Sprintf(`
		@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#>.
		@prefix config: <tag:rdf4j.org,2023:config/>.

		[] a config:Repository ;
		config:rep.id "%s" ;
		rdfs:label "LMDB store" ;
		config:rep.impl [
			config:rep.type "openrdf:SailRepository" ;
			config:sail.impl [
				config:sail.type "rdf4j:LmdbStore" ;
				config:sail.defaultQueryEvaluationMode "STANDARD"
			]
		].
	`, repositoryID)

	client := &http.Client{}
	req, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("%s/repositories/%s", serverURL, repositoryID),
		bytes.NewReader([]byte(config)),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "text/turtle")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("HTTP error. Status: %s, read error: %v", resp.Status, err)
		}
		return fmt.Errorf("failed to create LMDB repository. Status: %s, Body: %s", resp.Status, string(body))
	}

	return nil
}
