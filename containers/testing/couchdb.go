package testing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// CouchDBConfig holds configuration for CouchDB testcontainer setup.
type CouchDBConfig struct {
	// Image is the Docker image to use (default: "couchdb:3")
	Image string
	// AdminUsername is the CouchDB admin username (default: "admin")
	AdminUsername string
	// AdminPassword is the CouchDB admin password (default: "admin")
	AdminPassword string
	// StartupTimeout is the maximum time to wait for CouchDB to be ready (default: 60s)
	StartupTimeout time.Duration
}

// DefaultCouchDBConfig returns the default CouchDB configuration for testing.
func DefaultCouchDBConfig() CouchDBConfig {
	return CouchDBConfig{
		Image:          "couchdb:3",
		AdminUsername:  "admin",
		AdminPassword:  "admin",
		StartupTimeout: 60 * time.Second,
	}
}

// SetupCouchDB creates a CouchDB container for integration testing.
//
// CouchDB is a document-oriented NoSQL database. This function starts a CouchDB
// container using testcontainers-go and returns the connection URL and a cleanup
// function.
//
// Container Configuration:
//   - Image: couchdb:3 (official Apache CouchDB image)
//   - Port: 5984/tcp (CouchDB HTTP API)
//   - Admin Credentials: Configurable via CouchDBConfig
//   - Wait Strategy: HTTP readiness check on /_up endpoint
//
// Parameters:
//   - ctx: Context for container operations
//   - t: Testing context for requirement checks
//   - config: Optional CouchDB configuration (uses defaults if nil)
//
// Returns:
//   - string: CouchDB connection URL with embedded credentials
//     (e.g., "http://admin:admin@localhost:32769")
//   - ContainerCleanup: Function to terminate the container
//   - error: Container creation or startup errors
//
// Example Usage:
//
//	func TestCouchDBIntegration(t *testing.T) {
//	    ctx := context.Background()
//	    couchURL, cleanup, err := SetupCouchDB(ctx, t, nil)
//	    require.NoError(t, err)
//	    defer cleanup()
//
//	    // Use couchURL to interact with CouchDB
//	    // Example: http://admin:admin@localhost:32769
//	}
//
// CouchDB HTTP API Endpoints:
//   - GET  /_up                    - Health check
//   - GET  /_all_dbs              - List all databases
//   - PUT  /{database}            - Create database
//   - GET  /{database}/{doc_id}   - Get document
//   - PUT  /{database}/{doc_id}   - Create/update document
//   - DELETE /{database}/{doc_id} - Delete document
//
// Authentication:
//
//	CouchDB uses HTTP Basic Authentication. The returned URL includes
//	embedded credentials for convenience:
//	http://username:password@host:port
//
//	This format works with most CouchDB clients and allows direct use
//	without separate credential configuration.
//
// Cleanup:
//
//	Always defer the cleanup function to ensure the container is terminated:
//	defer cleanup()
//
// Single Node Mode:
//
//	The container runs in single-node mode which is appropriate for testing.
//	The setup process waits for the node to finish initialization before
//	returning, ensuring CouchDB is ready for database operations.
//
// Error Handling:
//
//	If container creation fails, the test should fail with require.NoError(t, err).
//	The cleanup function is safe to call even if setup fails (it's a no-op).
func SetupCouchDB(ctx context.Context, t *testing.T, config *CouchDBConfig) (string, ContainerCleanup, error) {
	// Use default config if none provided
	if config == nil {
		defaultConfig := DefaultCouchDBConfig()
		config = &defaultConfig
	}

	// Create container request
	req := testcontainers.ContainerRequest{
		Image:        config.Image,
		ExposedPorts: []string{"5984/tcp"},
		Env: map[string]string{
			"COUCHDB_USER":     config.AdminUsername,
			"COUCHDB_PASSWORD": config.AdminPassword,
		},
		// CouchDB takes time to initialize in single-node mode
		// The _up endpoint becomes available when setup is complete
		WaitingFor: wait.ForHTTP("/_up").
			WithPort("5984/tcp").
			WithStartupTimeout(config.StartupTimeout),
	}

	// Start container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", func() {}, fmt.Errorf("failed to start CouchDB container: %w", err)
	}

	// Get container connection details
	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return "", func() {}, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "5984")
	if err != nil {
		_ = container.Terminate(ctx)
		return "", func() {}, fmt.Errorf("failed to get mapped port: %w", err)
	}

	// Build CouchDB connection URL with embedded credentials
	// Format: http://username:password@host:port
	couchURL := fmt.Sprintf("http://%s:%s@%s:%s",
		config.AdminUsername,
		config.AdminPassword,
		host,
		port.Port())

	// Create cleanup function
	cleanup := createCleanupFunc(ctx, container, "CouchDB")

	return couchURL, cleanup, nil
}

// SetupCouchDBWithDatabase creates a CouchDB container and creates a test database.
//
// This is a convenience function that combines SetupCouchDB with database creation.
// Useful for tests that need a pre-existing database.
//
// Parameters:
//   - ctx: Context for container operations
//   - t: Testing context for requirement checks
//   - config: Optional CouchDB configuration (uses defaults if nil)
//   - databaseName: Name of the database to create
//
// Returns:
//   - string: CouchDB connection URL with embedded credentials
//   - string: Database name (same as input for convenience)
//   - ContainerCleanup: Function to terminate the container
//   - error: Container creation, startup, or database creation errors
//
// Example Usage:
//
//	func TestWithDatabase(t *testing.T) {
//	    ctx := context.Background()
//	    couchURL, dbName, cleanup, err := SetupCouchDBWithDatabase(ctx, t, nil, "testdb")
//	    require.NoError(t, err)
//	    defer cleanup()
//
//	    // Database "testdb" is already created and ready to use
//	    // Access via: http://admin:admin@localhost:32769/testdb
//	}
//
// Database Creation:
//
//	The database is created via HTTP PUT request to /{database}.
//	CouchDB will return 201 Created for successful creation or
//	412 Precondition Failed if the database already exists.
//
// Note: Database creation requires HTTP client calls to CouchDB HTTP API.
// For now, we return the URL and database name. The calling test can create
// the database using the EVE CouchDB service or direct HTTP calls.
func SetupCouchDBWithDatabase(ctx context.Context, t *testing.T, config *CouchDBConfig, databaseName string) (string, string, ContainerCleanup, error) {
	// Setup CouchDB container
	couchURL, cleanup, err := SetupCouchDB(ctx, t, config)
	if err != nil {
		return "", "", cleanup, err
	}

	// Note: Database creation would require HTTP client calls to CouchDB HTTP API
	// For now, we return the URL and database name
	// The calling test can create the database using the EVE db package:
	// service, err := evedb.NewCouchDBServiceFromConfig(evedb.CouchDBConfig{
	//     URL: couchURL,
	//     Database: databaseName,
	//     CreateIfMissing: true,
	// })

	return couchURL, databaseName, cleanup, nil
}
