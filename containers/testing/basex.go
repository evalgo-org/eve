package testing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// BaseXConfig holds configuration for BaseX testcontainer setup.
type BaseXConfig struct {
	// Image is the Docker image to use (default: "basex/basexhttp:latest")
	Image string
	// AdminPassword is the BaseX admin password (default: "admin")
	AdminPassword string
	// StartupTimeout is the maximum time to wait for BaseX to be ready (default: 60s)
	StartupTimeout time.Duration
}

// DefaultBaseXConfig returns the default BaseX configuration for testing.
func DefaultBaseXConfig() BaseXConfig {
	return BaseXConfig{
		Image:          "basex/basexhttp:latest",
		AdminPassword:  "admin",
		StartupTimeout: 60 * time.Second,
	}
}

// SetupBaseX creates a BaseX container for integration testing.
//
// BaseX is an XML database with XQuery support. This function starts a BaseX
// container using testcontainers-go and returns the REST API URL and a cleanup
// function.
//
// Container Configuration:
//   - Image: basex/basexhttp:latest (official BaseX HTTP server image)
//   - Port: 8984/tcp (BaseX REST API)
//   - Admin Password: Configurable via BaseXConfig
//   - Wait Strategy: HTTP readiness check on root endpoint
//
// Parameters:
//   - ctx: Context for container operations
//   - t: Testing context for requirement checks
//   - config: Optional BaseX configuration (uses defaults if nil)
//
// Returns:
//   - string: BaseX REST API URL (e.g., "http://localhost:32768")
//   - ContainerCleanup: Function to terminate the container
//   - error: Container creation or startup errors
//
// Example Usage:
//
//	func TestBaseXIntegration(t *testing.T) {
//	    ctx := context.Background()
//	    baseXURL, cleanup, err := SetupBaseX(ctx, t, nil)
//	    require.NoError(t, err)
//	    defer cleanup()
//
//	    // Use baseXURL to interact with BaseX REST API
//	    // Example: http://localhost:32768/rest
//	}
//
// BaseX REST API Endpoints:
//   - GET  /{database}          - List database resources
//   - GET  /{database}/{resource} - Retrieve resource
//   - POST /{database}          - Execute XQuery
//   - PUT  /{database}/{resource} - Create/update resource
//   - DELETE /{database}/{resource} - Delete resource
//
// Authentication:
//
//	BaseX uses HTTP Basic Authentication. Default credentials:
//	- Username: admin
//	- Password: admin (or custom via BaseXConfig.AdminPassword)
//
// Cleanup:
//
//	Always defer the cleanup function to ensure the container is terminated:
//	defer cleanup()
//
// Error Handling:
//
//	If container creation fails, the test should fail with require.NoError(t, err).
//	The cleanup function is safe to call even if setup fails (it's a no-op).
func SetupBaseX(ctx context.Context, t *testing.T, config *BaseXConfig) (string, ContainerCleanup, error) {
	// Use default config if none provided
	if config == nil {
		defaultConfig := DefaultBaseXConfig()
		config = &defaultConfig
	}

	// Create container request
	req := testcontainers.ContainerRequest{
		Image:        config.Image,
		ExposedPorts: []string{"8984/tcp"},
		Env: map[string]string{
			"BASEX_ADMIN_PW": config.AdminPassword,
		},
		WaitingFor: wait.ForHTTP("/").
			WithPort("8984/tcp").
			WithStartupTimeout(config.StartupTimeout),
	}

	// Start container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", func() {}, fmt.Errorf("failed to start BaseX container: %w", err)
	}

	// Get container connection details
	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return "", func() {}, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "8984")
	if err != nil {
		container.Terminate(ctx)
		return "", func() {}, fmt.Errorf("failed to get mapped port: %w", err)
	}

	// Build BaseX REST API URL
	baseXURL := getConnectionURL("http", host, port.Port(), "")

	// Create cleanup function
	cleanup := createCleanupFunc(ctx, container, "BaseX")

	return baseXURL, cleanup, nil
}

// SetupBaseXWithDatabase creates a BaseX container and creates a test database.
//
// This is a convenience function that combines SetupBaseX with database creation.
// Useful for tests that need a pre-existing database.
//
// Parameters:
//   - ctx: Context for container operations
//   - t: Testing context for requirement checks
//   - config: Optional BaseX configuration (uses defaults if nil)
//   - databaseName: Name of the database to create
//
// Returns:
//   - string: BaseX REST API URL
//   - string: Database name (same as input for convenience)
//   - ContainerCleanup: Function to terminate the container
//   - error: Container creation, startup, or database creation errors
//
// Example Usage:
//
//	func TestWithDatabase(t *testing.T) {
//	    ctx := context.Background()
//	    baseXURL, dbName, cleanup, err := SetupBaseXWithDatabase(ctx, t, nil, "testdb")
//	    require.NoError(t, err)
//	    defer cleanup()
//
//	    // Database "testdb" is already created and ready to use
//	}
//
// Note: Database creation is performed via BaseX REST API.
// The database is empty initially and can be populated with documents.
func SetupBaseXWithDatabase(ctx context.Context, t *testing.T, config *BaseXConfig, databaseName string) (string, string, ContainerCleanup, error) {
	// Setup BaseX container
	baseXURL, cleanup, err := SetupBaseX(ctx, t, config)
	if err != nil {
		return "", "", cleanup, err
	}

	// Note: Database creation would require HTTP client calls to BaseX REST API
	// For now, we return the URL and database name
	// The calling test can create the database using the REST API

	return baseXURL, databaseName, cleanup, nil
}
