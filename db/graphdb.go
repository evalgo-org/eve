// Package db provides comprehensive GraphDB integration for RDF graph database operations.
// This package implements a complete client library for interacting with GraphDB servers,
// supporting repository management, RDF graph operations, data import/export, and
// secure connectivity through Ziti zero-trust networking.
//
// GraphDB Integration:
//
//	GraphDB is an enterprise RDF graph database that provides:
//	- SPARQL 1.1 query and update support
//	- Named graph management for data organization
//	- High-performance RDF storage and retrieval
//	- RESTful API for administration and data operations
//	- Binary RDF format (BRF) for efficient data transfer
//
// Graph Database Operations:
//
//	The package supports complete graph database lifecycle:
//	- Repository discovery and configuration management
//	- Named graph creation, import, and deletion
//	- RDF data export in multiple formats (RDF/XML, Turtle, BRF)
//	- SPARQL query execution and result processing
//	- Backup and restore operations for data persistence
//
// Security and Connectivity:
//
//	Integrates with Ziti zero-trust networking for secure database access:
//	- HTTP Basic Authentication for traditional security
//	- Ziti overlay networking for network invisibility
//	- Configurable HTTP client with timeout management
//	- Support for both public and private network deployments
//
// Data Format Support:
//   - RDF/XML for W3C standard compatibility
//   - Turtle (TTL) for human-readable configuration
//   - Binary RDF (BRF) for high-performance data transfer
//   - JSON for API responses and metadata
//   - SPARQL Update for graph modifications
package db

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	eve "eve.evalgo.org/common"
)

// HttpClient provides the global HTTP client for GraphDB operations.
// This client can be customized for different connectivity patterns including
// Ziti zero-trust networking, custom timeouts, and proxy configurations.
//
// Default Configuration:
//
//	Uses http.DefaultClient with standard Go HTTP client settings.
//	Can be replaced with custom clients for specific networking requirements
//	such as Ziti overlay networks or enterprise proxy configurations.
//
// Customization Examples:
//   - Ziti client via GraphDBZitiClient() function
//   - Custom timeouts for long-running operations
//   - Proxy configuration for corporate networks
//   - Certificate-based authentication for secure environments
var (
	HttpClient *http.Client = http.DefaultClient
)

// ContextID represents a context identifier in GraphDB responses.
// This structure captures context information for RDF graph operations,
// providing type and value information for graph context management.
//
// Context Types:
//   - "uri": Named graph URI contexts
//   - "literal": Literal value contexts
//   - "bnode": Blank node contexts
//
// Usage in GraphDB:
//
//	Context IDs are used to identify and manage named graphs within
//	GraphDB repositories, enabling graph-level operations and queries.
type ContextID struct {
	Type  string `json:"type"`  // Context type (uri, literal, bnode)
	Value string `json:"value"` // Context value or identifier
}

// GraphDBBinding represents a single binding result from GraphDB SPARQL queries.
// This structure captures the complex binding format returned by GraphDB API
// responses, including repository metadata and graph information.
//
// Binding Fields:
//   - Readable: Access permissions for read operations
//   - Id: Repository or graph identifier information
//   - Title: Human-readable titles and descriptions
//   - Uri: URI references for resources
//   - Writable: Access permissions for write operations
//   - ContextID: Graph context information
//
// Data Access:
//
//	Each field is a map[string]string containing type and value information
//	similar to standard SPARQL JSON results format but with GraphDB-specific
//	extensions for repository and graph management.
type GraphDBBinding struct {
	Readable  map[string]string `json:"readable"`  // Read access information
	Id        map[string]string `json:"id"`        // Resource identifier
	Title     map[string]string `json:"title"`     // Human-readable title
	Uri       map[string]string `json:"uri"`       // Resource URI
	Writable  map[string]string `json:"writable"`  // Write access information
	ContextID ContextID         `json:"contextID"` // Graph context identifier
}

// GraphDBResults contains an array of binding results from GraphDB queries.
// This structure represents the results section of GraphDB API responses,
// containing multiple binding objects for repository lists, graph lists,
// and other query results.
//
// Result Processing:
//
//	The Bindings array contains one entry per result item, where each
//	binding provides detailed information about repositories, graphs,
//	or other GraphDB resources depending on the query type.
type GraphDBResults struct {
	Bindings []GraphDBBinding `json:"bindings"` // Array of query result bindings
}

// GraphDBResponse represents the complete response structure from GraphDB API calls.
// This structure follows GraphDB's JSON response format for repository listing,
// graph management, and other administrative operations.
//
// Response Structure:
//   - Head: Contains metadata about the response (variables, etc.)
//   - Results: Contains the actual query results with detailed bindings
//
// API Compatibility:
//
//	The structure is designed to handle GraphDB's specific JSON response
//	format which may differ from standard SPARQL JSON results in some
//	administrative operations.
type GraphDBResponse struct {
	Head    []interface{}  `json:"head>vars"` // Response metadata
	Results GraphDBResults `json:"results"`   // Query results with bindings
}

// GraphDBZitiClient creates an HTTP client configured for Ziti zero-trust networking.
// This function enables secure GraphDB access through Ziti overlay networks,
// providing network invisibility and strong identity-based authentication.
//
// Ziti Integration:
//
//	Uses the ZitiSetup function to create a transport that routes all
//	GraphDB traffic through the Ziti network overlay, ensuring:
//	- End-to-end encryption for all database communications
//	- Network invisibility (no exposed ports or network discovery)
//	- Identity-based access control and policy enforcement
//	- Automatic service discovery within the Ziti network
//
// Parameters:
//   - identityFile: Path to Ziti identity file for authentication
//   - serviceName: Name of the Ziti service for GraphDB access
//
// Returns:
//   - *http.Client: HTTP client configured with Ziti transport
//   - error: Ziti setup or configuration errors
//
// Client Configuration:
//
//	The returned client includes:
//	- Custom transport for Ziti network routing
//	- 30-second timeout for database operations
//	- Standard HTTP client features for GraphDB API calls
//
// Error Conditions:
//   - Invalid Ziti identity file or credentials
//   - Ziti service not found or accessible
//   - Network connectivity issues to Ziti controllers
//   - Invalid service name or configuration
//
// Example Usage:
//
//	client, err := GraphDBZitiClient("/path/to/identity.json", "graphdb-service")
//	if err != nil {
//	    log.Fatal("Failed to create Ziti client:", err)
//	}
//	HttpClient = client // Use Ziti client for all GraphDB operations
//
// Security Benefits:
//   - GraphDB server becomes invisible to traditional networks
//   - Strong cryptographic identity for all connections
//   - Dynamic policy enforcement and access control
//   - Protection against network-based attacks and reconnaissance
func GraphDBZitiClient(identityFile, serviceName string) (*http.Client, error) {
	zitiTransport, err := ZitiSetup(identityFile, serviceName)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Transport: zitiTransport,
		Timeout:   30 * time.Second,
	}, nil
}

// GraphDBRepositories retrieves a list of all repositories from a GraphDB server.
// This function queries the GraphDB management API to discover available
// repositories and their metadata for administration and data access purposes.
//
// Repository Discovery:
//
//	Connects to the GraphDB repositories endpoint to retrieve:
//	- Repository identifiers and names
//	- Access permissions (readable/writable status)
//	- Repository types and configurations
//	- Context information for graph management
//
// Parameters:
//   - url: Base URL of the GraphDB server (e.g., "http://localhost:7200")
//   - user: Username for HTTP Basic Authentication (empty string for no auth)
//   - pass: Password for HTTP Basic Authentication (empty string for no auth)
//
// Returns:
//   - *GraphDBResponse: Structured response with repository information
//   - error: HTTP communication, authentication, or parsing errors
//
// Authentication:
//
//	Supports HTTP Basic Authentication when credentials are provided.
//	Empty username and password values skip authentication for open servers.
//
// Response Format:
//
//	Returns GraphDB's JSON format with bindings containing:
//	- Repository IDs and human-readable titles
//	- URI references for API access
//	- Access permissions for read/write operations
//	- Context information for graph-aware operations
//
// Error Conditions:
//   - Network connectivity issues to GraphDB server
//   - Authentication failures with provided credentials
//   - Server errors or invalid responses
//   - JSON parsing errors in response data
//
// Example Usage:
//
//	repos, err := GraphDBRepositories("http://localhost:7200", "admin", "password")
//	if err != nil {
//	    log.Fatal("Failed to list repositories:", err)
//	}
//
//	for _, binding := range repos.Results.Bindings {
//	    fmt.Printf("Repository: %s - %s\n",
//	               binding.Id["value"], binding.Title["value"])
//	}
//
// Administrative Use:
//
//	This function is typically used for:
//	- Repository discovery and inventory
//	- Administrative dashboard implementations
//	- Automated repository management scripts
//	- Health monitoring and status checking
func GraphDBRepositories(url string, user string, pass string) (*GraphDBResponse, error) {
	tgt_url := url + "/repositories"
	req, err := http.NewRequest("GET", tgt_url, nil)
	if err != nil {
		return nil, err
	}
	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Add("Accept", "application/json")
	res, err := HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		response := GraphDBResponse{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, err
		}
		return &response, nil
	}
	return nil, fmt.Errorf("could not return repositories because of status code: %d", res.StatusCode)
}

// GraphDBRepositoryConf downloads the configuration of a GraphDB repository in Turtle format.
// This function retrieves the complete repository configuration for backup,
// analysis, or recreation purposes in a human-readable Turtle serialization.
//
// Configuration Export:
//
//	Downloads the repository configuration from GraphDB's REST API endpoint,
//	saving it as a Turtle (.ttl) file that contains:
//	- Repository type and storage backend settings
//	- Index configurations and performance tuning
//	- Security and access control settings
//	- Plugin and extension configurations
//
// Parameters:
//   - url: Base URL of the GraphDB server
//   - user: Username for HTTP Basic Authentication
//   - pass: Password for HTTP Basic Authentication
//   - repo: Repository identifier to download configuration for
//
// Returns:
//   - string: Filename of the downloaded configuration file (repo.ttl)
//
// File Output:
//
//	Creates a file named "{repo}.ttl" in the current directory containing
//	the complete repository configuration in Turtle format, suitable for:
//	- Repository backup and disaster recovery
//	- Configuration analysis and documentation
//	- Repository recreation on different servers
//	- Version control of repository settings
//
// Error Handling:
//   - Network errors are logged via eve.Logger.Error
//   - HTTP errors cause fatal logging and program termination
//   - File creation errors are logged but don't terminate execution
//
// Example Usage:
//
//	configFile := GraphDBRepositoryConf("http://localhost:7200", "admin", "password", "my-repo")
//	fmt.Printf("Repository configuration saved to: %s\n", configFile)
//
// Turtle Format Benefits:
//   - Human-readable RDF configuration
//   - Easy editing and version control
//   - Standard RDF serialization format
//   - Compatible with RDF tools and editors
func GraphDBRepositoryConf(url string, user string, pass string, repo string) string {
	tgt_url := url + "/rest/repositories/" + repo + "/download-ttl"
	req, _ := http.NewRequest("GET", tgt_url, nil)
	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Add("Accept", "text/turtle")
	res, err := HttpClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		out, err := os.Create(repo + ".ttl")
		if err != nil {
			eve.Logger.Info(err)
		}
		defer out.Close()
		io.Copy(out, res.Body)
		return repo + ".ttl"
	}
	eve.Logger.Fatal(res.StatusCode, http.StatusText(res.StatusCode))
	return ""
}

// GraphDBRepositoryBrf exports all RDF data from a repository in Binary RDF format.
// This function downloads the complete repository content in GraphDB's efficient
// Binary RDF (BRF) format for high-performance backup and data transfer operations.
//
// Binary RDF Format:
//
//	BRF is GraphDB's proprietary binary serialization that provides:
//	- Compact data representation with reduced file sizes
//	- Fast serialization and deserialization performance
//	- Preservation of all RDF data including named graphs
//	- Optimized format for large dataset operations
//
// Parameters:
//   - url: Base URL of the GraphDB server
//   - user: Username for HTTP Basic Authentication
//   - pass: Password for HTTP Basic Authentication
//   - repo: Repository identifier to export data from
//
// Returns:
//   - string: Filename of the exported data file (repo.brf)
//
// Export Process:
//  1. Connects to the repository statements endpoint
//  2. Requests data in Binary RDF format
//  3. Downloads all triples and named graphs
//  4. Saves to a .brf file for future restoration
//
// File Output:
//
//	Creates a file named "{repo}.brf" containing all repository data
//	in binary format, suitable for:
//	- High-performance backup operations
//	- Data migration between GraphDB instances
//	- Bulk data transfer for analytics
//	- Repository cloning and replication
//
// Performance Benefits:
//   - Significantly smaller file sizes compared to RDF/XML or Turtle
//   - Faster download and upload operations
//   - Reduced network bandwidth usage
//   - Optimized for GraphDB's internal data structures
//
// Error Handling:
//   - Network errors are logged and may cause termination
//   - HTTP errors result in fatal logging and program exit
//   - File creation errors are logged as informational
//
// Example Usage:
//
//	backupFile := GraphDBRepositoryBrf("http://localhost:7200", "admin", "password", "production-data")
//	fmt.Printf("Repository backup saved to: %s\n", backupFile)
//
// Restore Compatibility:
//
//	The exported BRF file can be restored using GraphDBRestoreBrf()
//	function to recreate the repository with identical data content.
func GraphDBRepositoryBrf(url string, user string, pass string, repo string) string {
	tgt_url := url + "/repositories/" + repo + "/statements"
	req, _ := http.NewRequest("GET", tgt_url, nil)
	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Add("Accept", "application/x-binary-rdf")
	res, err := HttpClient.Do(req)
	if err != nil {
		eve.Logger.Error(err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		out, err := os.Create(repo + ".brf")
		if err != nil {
			eve.Logger.Info(err)
		}
		defer out.Close()
		io.Copy(out, res.Body)
		return repo + ".brf"
	}
	eve.Logger.Fatal(res.StatusCode, http.StatusText(res.StatusCode))
	return ""
}

// GraphDBRestoreConf restores a repository configuration from a Turtle file.
// This function uploads a repository configuration file to create a new
// repository with the settings defined in the Turtle configuration.
//
// Configuration Restoration:
//
//	Uses multipart form upload to send the Turtle configuration file
//	to GraphDB's repository creation endpoint, enabling:
//	- Repository recreation from backup configurations
//	- Deployment automation with predefined settings
//	- Configuration migration between environments
//	- Template-based repository creation
//
// Parameters:
//   - url: Base URL of the GraphDB server
//   - user: Username for HTTP Basic Authentication
//   - pass: Password for HTTP Basic Authentication
//   - restoreFile: Path to the Turtle configuration file (.ttl)
//
// Returns:
//   - error: File reading, upload, or server errors
//
// Upload Process:
//  1. Reads the Turtle configuration file from disk
//  2. Creates multipart form data with the file content
//  3. Uploads to GraphDB's REST repositories endpoint
//  4. Validates successful repository creation response
//
// File Format Requirements:
//
//	The restore file must be a valid Turtle configuration containing:
//	- Repository type and identifier
//	- Storage backend configuration
//	- Index and performance settings
//	- Security and access control definitions
//
// Success Conditions:
//
//	Repository creation is successful when the server returns:
//	- HTTP 201 Created status code
//	- Confirmation message in response body
//
// Error Conditions:
//   - Configuration file not found or unreadable
//   - Invalid Turtle syntax in configuration
//   - Repository ID conflicts with existing repositories
//   - Authentication failures or insufficient permissions
//   - Server-side configuration validation errors
//
// Example Usage:
//
//	err := GraphDBRestoreConf("http://localhost:7200", "admin", "password", "backup-repo.ttl")
//	if err != nil {
//	    log.Fatal("Configuration restore failed:", err)
//	}
//	log.Println("Repository restored successfully")
//
// Best Practices:
//   - Validate configuration files before restoration
//   - Ensure repository IDs don't conflict with existing ones
//   - Test configurations in development before production use
//   - Backup existing repositories before restoration operations
func GraphDBRestoreConf(url string, user string, pass string, restoreFile string) error {
	var (
		buf = new(bytes.Buffer)
		w   = multipart.NewWriter(buf)
	)
	part, err := w.CreateFormFile("config", filepath.Base(restoreFile))
	if err != nil {
		return err
	}
	fData, err := ioutil.ReadFile(restoreFile)
	if err != nil {
		return err
	}
	_, err = part.Write(fData)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	tgt_url := url + "/rest/repositories"
	req, _ := http.NewRequest("POST", tgt_url, buf)
	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", w.FormDataContentType())
	res, _ := HttpClient.Do(req)
	body, _ := io.ReadAll(res.Body)
	defer res.Body.Close()
	if res.StatusCode == http.StatusCreated {
		eve.Logger.Info(string(body))
		return nil
	}
	eve.Logger.Fatal(res.StatusCode, http.StatusText(res.StatusCode))
	return errors.New("could not run GraphDBRestoreConf")
}

// GraphDBRestoreBrf restores RDF data from a Binary RDF file into a repository.
// This function uploads BRF data to recreate repository content with high
// performance and complete data fidelity preservation.
//
// Binary RDF Restoration:
//
//	Uploads Binary RDF data directly to the repository statements endpoint,
//	restoring all triples and named graphs with:
//	- High-performance data upload using binary format
//	- Complete preservation of graph structure and content
//	- Efficient processing of large datasets
//	- Atomic restoration operation
//
// Parameters:
//   - url: Base URL of the GraphDB server
//   - user: Username for HTTP Basic Authentication
//   - pass: Password for HTTP Basic Authentication
//   - restoreFile: Path to the Binary RDF file (.brf)
//
// Returns:
//   - error: File reading, upload, or server errors
//
// Repository Inference:
//
//	The target repository name is automatically inferred from the BRF
//	filename (excluding the .brf extension), enabling automated
//	restoration workflows with consistent naming patterns.
//
// Data Upload Process:
//  1. Reads the complete BRF file into memory
//  2. Extracts repository name from filename
//  3. Uploads data to the repository statements endpoint
//  4. Validates successful data import response
//
// Performance Characteristics:
//   - Binary format provides fastest upload speeds
//   - Single HTTP request for complete data transfer
//   - Minimal CPU overhead during upload
//   - Efficient memory usage for large datasets
//
// Success Conditions:
//
//	Data restoration is successful when the server returns:
//	- HTTP 204 No Content status code
//	- Empty response body (data loaded successfully)
//
// Error Conditions:
//   - BRF file not found or unreadable
//   - Repository does not exist (must be created first)
//   - Authentication failures or insufficient permissions
//   - Server-side data processing errors
//   - Memory limitations with very large datasets
//
// Example Usage:
//
//	err := GraphDBRestoreBrf("http://localhost:7200", "admin", "password", "production-data.brf")
//	if err != nil {
//	    log.Fatal("Data restore failed:", err)
//	}
//	log.Println("Repository data restored successfully")
//
// Workflow Integration:
//
//	Typically used after GraphDBRestoreConf() to complete repository restoration:
//	1. Restore repository configuration
//	2. Restore repository data from BRF file
//	3. Verify data integrity and accessibility
func GraphDBRestoreBrf(url string, user string, pass string, restoreFile string) error {
	fData, err := ioutil.ReadFile(restoreFile)
	if err != nil {
		return err
	}
	repo := strings.TrimSuffix(filepath.Base(restoreFile), filepath.Ext(restoreFile))
	tgt_url := url + "/repositories/" + repo + "/statements"
	req, _ := http.NewRequest("POST", tgt_url, bytes.NewBuffer(fData))
	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-binary-rdf")
	res, _ := HttpClient.Do(req)
	body, _ := io.ReadAll(res.Body)
	defer res.Body.Close()
	if res.StatusCode == http.StatusNoContent {
		eve.Logger.Info(string(body))
		return nil
	}
	eve.Logger.Fatal(res.StatusCode, http.StatusText(res.StatusCode))
	return errors.New("could not run GraphDBRestoreBrf")
}

// GraphDBImportGraphRdf imports RDF data into a specific named graph within a repository.
// This function enables graph-level data management by importing RDF/XML content
// into designated named graphs for organized data storage and querying.
//
// Named Graph Import:
//
//	Loads RDF data into a specific named graph context within the repository,
//	enabling:
//	- Data organization by source, domain, or purpose
//	- Graph-level access control and permissions
//	- Selective querying and data management
//	- Logical separation of different datasets
//
// Parameters:
//   - url: Base URL of the GraphDB server
//   - user: Username for HTTP Basic Authentication
//   - pass: Password for HTTP Basic Authentication
//   - repo: Repository identifier for data import
//   - graph: Named graph URI for data context
//   - restoreFile: Path to RDF/XML file containing data
//
// Returns:
//   - error: File reading, upload, or server errors
//
// Graph Context Management:
//
//	The graph parameter specifies the named graph URI where data will be
//	stored, enabling graph-aware operations and queries. The URI typically
//	follows standard IRI format for global uniqueness.
//
// RDF/XML Format:
//
//	The function expects RDF/XML formatted data files containing:
//	- Valid RDF triples with subject, predicate, object
//	- Namespace declarations for vocabulary terms
//	- Proper XML structure and RDF syntax
//	- Compatible encoding (UTF-8 recommended)
//
// Import Process:
//  1. Reads RDF/XML file from filesystem
//  2. Constructs URL with graph parameter
//  3. Uploads data via HTTP PUT to RDF graphs service
//  4. Validates successful import response
//
// Success Conditions:
//
//	Data import is successful when the server returns:
//	- HTTP 204 No Content status code
//	- Empty response body indicating successful processing
//
// Error Conditions:
//   - RDF file not found or unreadable
//   - Invalid RDF/XML syntax or structure
//   - Repository does not exist
//   - Graph URI format errors
//   - Authentication failures or insufficient permissions
//
// Example Usage:
//
//	err := GraphDBImportGraphRdf(
//	    "http://localhost:7200", "admin", "password",
//	    "knowledge-base", "http://example.org/graph/publications",
//	    "/data/publications.rdf")
//	if err != nil {
//	    log.Fatal("Graph import failed:", err)
//	}
//
// Graph Organization Strategies:
//   - Domain-based: separate graphs for different knowledge domains
//   - Source-based: separate graphs for different data sources
//   - Time-based: separate graphs for different time periods
//   - Access-based: separate graphs for different security levels
func GraphDBImportGraphRdf(url, user, pass, repo, graph, restoreFile string) error {
	fData, err := ioutil.ReadFile(restoreFile)
	if err != nil {
		return err
	}
	tgt_url := url + "/repositories/" + repo + "/rdf-graphs/service"
	req, err := http.NewRequest("PUT", tgt_url, bytes.NewBuffer(fData))
	if err != nil {
		return err
	}
	values := req.URL.Query()
	values.Add("graph", graph)
	req.URL.RawQuery = values.Encode()
	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/rdf+xml")
	res, err := HttpClient.Do(req)
	if err != nil {
		return err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNoContent {
		eve.Logger.Info(string(body))
		return nil
	}
	return errors.New("could not run GraphDBImportGraphRdf " + http.StatusText(res.StatusCode))
}

// GraphDBDeleteRepository removes a repository and all its data from GraphDB server.
// This function permanently deletes a repository configuration and all stored
// RDF data, providing a clean removal mechanism for repository lifecycle management.
//
// Repository Deletion:
//
//	Sends a DELETE request to GraphDB's REST repository endpoint to:
//	- Remove repository configuration and metadata
//	- Delete all stored RDF triples and named graphs
//	- Clean up associated indexes and cached data
//	- Free storage space and system resources
//
// Parameters:
//   - URL: Base URL of the GraphDB server
//   - user: Username for HTTP Basic Authentication
//   - pass: Password for HTTP Basic Authentication
//   - repo: Repository identifier to delete
//
// Returns:
//   - error: Authentication, permission, or deletion errors
//
// Data Loss Warning:
//
//	This operation is irreversible and will permanently destroy:
//	- All RDF triples and named graphs in the repository
//	- Repository configuration and settings
//	- Custom indexes and optimization data
//	- Query result caches and temporary data
//
// Success Conditions:
//
//	Repository deletion is successful when the server returns:
//	- HTTP 200 OK (deletion completed successfully)
//	- HTTP 204 No Content (deletion completed without response body)
//
// Authentication Requirements:
//   - Valid credentials with repository deletion permissions
//   - Administrative access to the GraphDB server
//   - Appropriate role-based access controls
//
// Error Conditions:
//   - Repository not found (may already be deleted)
//   - Authentication failures or insufficient permissions
//   - Repository currently in use by active connections
//   - Server-side errors during deletion process
//   - Network connectivity issues
//
// Example Usage:
//
//	err := GraphDBDeleteRepository("http://localhost:7200", "admin", "password", "test-repository")
//	if err != nil {
//	    log.Fatal("Repository deletion failed:", err)
//	}
//	log.Println("Repository deleted successfully")
//
// Safety Recommendations:
//   - Always backup important repositories before deletion
//   - Verify repository name to prevent accidental deletions
//   - Check for dependent applications or integrations
//   - Implement proper access controls and audit logging
//   - Consider soft deletion patterns for critical systems
//
// Administrative Use:
//
//	This function is typically used for:
//	- Development environment cleanup
//	- Automated testing teardown procedures
//	- Repository lifecycle management
//	- Data migration and reorganization
func GraphDBDeleteRepository(URL, user, pass, repo string) error {
	tgt_url := URL + "/rest/repositories/" + repo
	eve.Logger.Info(tgt_url)

	req, err := http.NewRequest("DELETE", tgt_url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}

	req.Header.Add("Accept", "application/json")

	res, err := HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNoContent {
		eve.Logger.Info("Repository deleted successfully:", string(body))
		return nil
	}

	eve.Logger.Error("Failed to delete repository:", res.StatusCode, http.StatusText(res.StatusCode), string(body))
	return fmt.Errorf("could not delete repository: %s (%d)", http.StatusText(res.StatusCode), res.StatusCode)
}

// GraphDBDeleteGraph removes a specific named graph from a repository using SPARQL UPDATE.
// This function executes a DROP GRAPH operation to permanently delete all triples
// in the specified named graph while preserving other graphs in the repository.
//
// Named Graph Deletion:
//
//	Uses SPARQL UPDATE "DROP GRAPH" command to:
//	- Remove all triples from the specified named graph
//	- Preserve other named graphs and default graph content
//	- Execute atomic deletion operation
//	- Update graph metadata and indexes
//
// Parameters:
//   - URL: Base URL of the GraphDB server
//   - user: Username for HTTP Basic Authentication
//   - pass: Password for HTTP Basic Authentication
//   - repo: Repository identifier containing the graph
//   - graph: Named graph URI to delete
//
// Returns:
//   - error: Authentication, execution, or server errors
//
// SPARQL Operation:
//
//	Constructs and executes: DROP GRAPH <graph-uri>
//	This standard SPARQL UPDATE operation removes all triples where
//	the named graph matches the specified URI.
//
// Graph URI Format:
//
//	The graph parameter should be a valid IRI (Internationalized Resource
//	Identifier) that uniquely identifies the named graph within the repository.
//
// Success Conditions:
//
//	Graph deletion is successful when the server returns:
//	- HTTP 204 No Content status code
//	- Empty response body indicating successful execution
//
// Error Conditions:
//   - Named graph does not exist (may be silent success)
//   - Invalid graph URI format
//   - Authentication failures or insufficient permissions
//   - Repository not found or inaccessible
//   - SPARQL syntax or execution errors
//
// Data Impact:
//   - Only affects triples in the specified named graph
//   - Default graph and other named graphs remain unchanged
//   - Graph metadata and context information is removed
//   - Indexes are updated to reflect graph deletion
//
// Example Usage:
//
//	err := GraphDBDeleteGraph(
//	    "http://localhost:7200", "admin", "password",
//	    "knowledge-base", "http://example.org/graph/outdated-data")
//	if err != nil {
//	    log.Fatal("Graph deletion failed:", err)
//	}
//	log.Println("Named graph deleted successfully")
//
// Safety Considerations:
//   - Verify graph URI to prevent accidental deletions
//   - Backup important graph data before deletion
//   - Check for applications depending on the graph
//   - Consider graph dependencies and relationships
func GraphDBDeleteGraph(URL, user, pass, repo, graph string) error {
	tgt_url := URL + "/repositories/" + repo + "/statements"
	eve.Logger.Info(tgt_url)
	fData := []byte("DROP GRAPH <" + graph + ">")
	req, _ := http.NewRequest("POST", tgt_url, bytes.NewBuffer(fData))
	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Add("Content-Type", "application/sparql-update")
	res, _ := HttpClient.Do(req)
	body, _ := io.ReadAll(res.Body)
	defer res.Body.Close()
	if res.StatusCode == http.StatusNoContent {
		eve.Logger.Info(string(body))
		return nil
	}
	eve.Logger.Fatal(res.StatusCode, http.StatusText(res.StatusCode))
	return errors.New("could not run GraphDBDeleteGraph")
}

// GraphDBListGraphs retrieves a list of all named graphs in a repository.
// This function queries GraphDB's RDF graphs endpoint to discover named
// graphs and their metadata for graph management and administration.
//
// Named Graph Discovery:
//
//	Connects to the repository's RDF graphs endpoint to retrieve:
//	- Named graph URIs and identifiers
//	- Graph metadata and context information
//	- Access permissions and properties
//	- Graph statistics and summary data
//
// Parameters:
//   - url: Base URL of the GraphDB server
//   - user: Username for HTTP Basic Authentication
//   - pass: Password for HTTP Basic Authentication
//   - repo: Repository identifier to list graphs from
//
// Returns:
//   - *GraphDBResponse: Structured response with graph information
//   - error: HTTP communication, authentication, or parsing errors
//
// Response Structure:
//
//	Returns GraphDBResponse containing bindings with graph information:
//	- Graph URIs and identifiers
//	- Context metadata and properties
//	- Access control information
//	- Graph statistics and summary data
//
// Graph Management:
//
//	The returned information enables:
//	- Graph inventory and discovery
//	- Access control verification
//	- Graph-aware query planning
//	- Administrative monitoring and reporting
//
// Success Conditions:
//
//	Graph listing is successful when the server returns:
//	- HTTP 200 OK status code
//	- Valid JSON response with graph bindings
//
// Error Conditions:
//   - Repository not found or inaccessible
//   - Authentication failures or insufficient permissions
//   - Server errors during graph enumeration
//   - JSON parsing errors in response data
//   - Network connectivity issues
//
// Example Usage:
//
//	graphs, err := GraphDBListGraphs("http://localhost:7200", "admin", "password", "knowledge-base")
//	if err != nil {
//	    log.Fatal("Failed to list graphs:", err)
//	}
//
//	for _, binding := range graphs.Results.Bindings {
//	    if uri, ok := binding.Uri["value"]; ok {
//	        fmt.Printf("Named graph: %s\n", uri)
//	    }
//	}
//
// Administrative Applications:
//   - Graph discovery for query optimization
//   - Access control auditing and verification
//   - Graph-based data organization analysis
//   - Repository content inventory and documentation
func GraphDBListGraphs(url, user, pass, repo string) (*GraphDBResponse, error) {
	tgt_url := url + "/repositories/" + repo + "/rdf-graphs"
	req, err := http.NewRequest("GET", tgt_url, nil)
	if err != nil {
		return nil, err
	}
	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Add("Accept", "application/json")
	res, err := HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	fmt.Println(res.StatusCode)
	if res.StatusCode == http.StatusOK {
		response := GraphDBResponse{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, err
		}
		return &response, nil
	}
	return nil, errors.New("could not run GraphDBListGraphs on " + repo)
}

// GraphDBExportGraphRdf exports a specific named graph from a repository to an RDF/XML file.
// This function retrieves all triples from a named graph and saves them in
// RDF/XML format for backup, analysis, or data transfer purposes.
//
// Named Graph Export:
//
//	Downloads all RDF triples from the specified named graph using
//	GraphDB's RDF graphs service endpoint, providing:
//	- Graph-specific data extraction
//	- Complete triple preservation with context
//	- Standard RDF/XML serialization format
//	- File-based output for external processing
//
// Parameters:
//   - url: Base URL of the GraphDB server
//   - user: Username for HTTP Basic Authentication
//   - pass: Password for HTTP Basic Authentication
//   - repo: Repository identifier containing the graph
//   - graph: Named graph URI to export
//   - exportFile: Output filename for the exported RDF/XML data
//
// Returns:
//   - error: File creation, network, or server errors
//
// Export Process:
//  1. Constructs request URL with graph parameter
//  2. Requests RDF/XML representation of the named graph
//  3. Downloads all triples from the specified graph context
//  4. Writes RDF/XML data to the specified output file
//
// RDF/XML Output:
//
//	The exported file contains standard RDF/XML with:
//	- All triples from the named graph
//	- Proper namespace declarations
//	- XML structure following RDF/XML specification
//	- UTF-8 encoding for international character support
//
// Graph Context Preservation:
//
//	While the exported RDF/XML doesn't explicitly contain graph context
//	information, all triples that belonged to the named graph are
//	preserved for reimport into the same or different graph contexts.
//
// Success Conditions:
//
//	Export is successful when the server returns:
//	- HTTP 200 OK status code
//	- Valid RDF/XML content in response body
//	- Successful file creation and writing
//
// Error Conditions:
//   - Named graph does not exist or is empty
//   - Authentication failures or insufficient permissions
//   - File creation or writing permissions errors
//   - Network connectivity issues during download
//   - Server errors during RDF serialization
//
// Example Usage:
//
//	err := GraphDBExportGraphRdf(
//	    "http://localhost:7200", "admin", "password",
//	    "knowledge-base", "http://example.org/graph/publications",
//	    "/backup/publications-export.rdf")
//	if err != nil {
//	    log.Fatal("Graph export failed:", err)
//	}
//	log.Println("Named graph exported successfully")
//
// Use Cases:
//   - Graph-specific backup and archival
//   - Data migration between repositories
//   - Selective data analysis and processing
//   - Graph-based data distribution and sharing
//   - Integration with external RDF tools and systems
func GraphDBExportGraphRdf(url, user, pass, repo, graph, exportFile string) error {
	tgt_url := url + "/repositories/" + repo + "/rdf-graphs/service"
	req, _ := http.NewRequest("GET", tgt_url, nil)
	values := req.URL.Query()
	values.Add("graph", graph)
	req.URL.RawQuery = values.Encode()
	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Add("Accept", "application/rdf+xml")
	res, err := HttpClient.Do(req)
	if err != nil {
		eve.Logger.Info("Failed to create file:", err)
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		outFile, err := os.Create(exportFile)
		if err != nil {
			eve.Logger.Info("Failed to create file:", err)
			return err
		}
		defer outFile.Close()
		_, err = io.Copy(outFile, res.Body)
		if err != nil {
			eve.Logger.Info("Error writing response to file:", err)
			return err
		}
		return nil
	}
	eve.Logger.Fatal(res.StatusCode, http.StatusText(res.StatusCode))
	return errors.New("could not run GraphDBExportGraphRdf")
}
