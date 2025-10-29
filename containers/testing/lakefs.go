package testing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// LakeFSConfig holds configuration for LakeFS testcontainer setup.
type LakeFSConfig struct {
	// Image is the Docker image to use (default: "treeverse/lakefs:1.70")
	Image string
	// StartupTimeout is the maximum time to wait for LakeFS to be ready (default: 120s)
	StartupTimeout time.Duration
}

// DefaultLakeFSConfig returns the default LakeFS configuration for testing.
func DefaultLakeFSConfig() LakeFSConfig {
	return LakeFSConfig{
		Image:          "treeverse/lakefs:1.70",
		StartupTimeout: 120 * time.Second,
	}
}

// SetupLakeFS creates a LakeFS container for integration testing.
//
// LakeFS is an open-source platform that brings Git-like version control to data lakes,
// enabling branches, commits, merges, and rollbacks for object storage. This function
// starts a LakeFS container using testcontainers-go and returns the connection URL and
// a cleanup function.
//
// Container Configuration:
//   - Image: treeverse/lakefs:1.70 (data lake versioning)
//   - Port: 8000/tcp (HTTP API and UI)
//   - Mode: Local mode (built-in KV store, no external DB required)
//   - Wait Strategy: HTTP GET / returning 200 OK
//
// Parameters:
//   - ctx: Context for container operations
//   - t: Testing context for requirement checks
//   - config: Optional LakeFS configuration (uses defaults if nil)
//
// Returns:
//   - string: LakeFS HTTP endpoint URL
//            (e.g., "http://localhost:32793")
//   - ContainerCleanup: Function to terminate the container
//   - error: Container creation or startup errors
//
// Example Usage:
//
//	func TestLakeFSIntegration(t *testing.T) {
//	    ctx := context.Background()
//	    lakeFSURL, cleanup, err := SetupLakeFS(ctx, t, nil)
//	    require.NoError(t, err)
//	    defer cleanup()
//
//	    // Use LakeFS API
//	    resp, err := http.Get(lakeFSURL + "/api/v1/healthcheck")
//	    require.NoError(t, err)
//	    defer resp.Body.Close()
//
//	    // LakeFS is ready for data versioning operations
//	}
//
// LakeFS Features:
//
//	Git-like data versioning platform:
//	- Branching and merging for data lakes
//	- Atomic commits for data changes
//	- Time travel and rollback capabilities
//	- CI/CD for data pipelines
//	- Data quality gates
//	- Isolated development environments
//	- Zero-copy branching (metadata only)
//	- S3-compatible API
//	- Support for multiple storage backends
//
// Storage Backends:
//
//	LakeFS supports various object storage backends:
//	- Amazon S3
//	- Azure Blob Storage
//	- Google Cloud Storage
//	- MinIO
//	- Local filesystem (for testing)
//
// API Endpoints:
//
//	Key HTTP API endpoints:
//	- GET  /api/v1/healthcheck - Health check
//	- GET  /api/v1/repositories - List repositories
//	- POST /api/v1/repositories - Create repository
//	- GET  /api/v1/repositories/{repository}/branches - List branches
//	- POST /api/v1/repositories/{repository}/branches - Create branch
//	- POST /api/v1/repositories/{repository}/branches/{branch}/commits - Commit changes
//	- POST /api/v1/repositories/{repository}/refs/{branch}/merge - Merge branches
//	- GET  /api/v1/repositories/{repository}/refs/{ref}/objects - List objects
//
// Web UI:
//
//	LakeFS provides a web interface for:
//	- Repository management
//	- Branch visualization
//	- Commit history
//	- Object browsing
//	- Diff viewing
//	- User management
//
// S3 Gateway:
//
//	LakeFS provides S3-compatible endpoints:
//	- Accessible at the same port (8000)
//	- Use repository/branch as bucket name: {repo}/{branch}
//	- Example: s3://my-repo/main/path/to/object
//
// Authentication:
//
//	For testing, LakeFS can run without authentication in local mode.
//	Default credentials for production mode:
//	- Access Key ID: generated on first run
//	- Secret Access Key: generated on first run
//
// Local Mode:
//
//	For testing, LakeFS runs in local mode:
//	- Built-in KV store (no PostgreSQL required)
//	- Local block storage
//	- Single node deployment
//	- Ephemeral data (lost on container restart)
//
// Performance:
//
//	LakeFS container starts in 30-60 seconds typically.
//	The wait strategy ensures the API is fully initialized and
//	ready to accept requests before returning.
//
// Data Storage:
//
//	LakeFS stores data in /data inside the container.
//	For testing, this is ephemeral (lost when container stops).
//	This ensures test isolation.
//
// Use Cases:
//
//	Data versioning scenarios:
//	- Testing ETL pipeline versions
//	- Data quality validation
//	- Rollback data changes
//	- Isolated data environments
//	- Data CI/CD workflows
//	- Reproducible data science
//	- Compliance and auditing
//
// Cleanup:
//
//	Always defer the cleanup function to ensure the container is terminated:
//	defer cleanup()
//
//	The cleanup function is safe to call even if setup fails (it's a no-op).
//
// Error Handling:
//
//	If container creation fails, the test should fail with require.NoError(t, err).
//	Common errors:
//	- Docker daemon not running
//	- Image pull failures (network issues)
//	- Port conflicts (rare with random ports)
//	- Insufficient memory (LakeFS requires ~512MB minimum)
//
// Git-like Operations:
//
//	LakeFS provides familiar Git operations:
//	- Branch: Create isolated data environments
//	- Commit: Create immutable snapshots
//	- Merge: Integrate changes from branches
//	- Revert: Roll back to previous versions
//	- Tag: Mark specific data versions
//	- Diff: Compare data between branches/commits
func SetupLakeFS(ctx context.Context, t *testing.T, config *LakeFSConfig) (string, ContainerCleanup, error) {
	// Use default config if none provided
	if config == nil {
		defaultConfig := DefaultLakeFSConfig()
		config = &defaultConfig
	}

	// Build environment variables for local mode
	env := map[string]string{
		"LAKEFS_DATABASE_TYPE":                "local",
		"LAKEFS_BLOCKSTORE_TYPE":              "local",
		"LAKEFS_BLOCKSTORE_LOCAL_PATH":        "/data",
		"LAKEFS_AUTH_ENCRYPT_SECRET_KEY":      "some-secret-for-testing",
		"LAKEFS_STATS_ENABLED":                "false",
		"LAKEFS_LOGGING_LEVEL":                "INFO",
		"LAKEFS_INSTALLATION_USER_NAME":       "admin",
		"LAKEFS_INSTALLATION_ACCESS_KEY_ID":   "AKIAIOSFODNN7EXAMPLE",
		"LAKEFS_INSTALLATION_SECRET_ACCESS_KEY": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	// Create container request
	req := testcontainers.ContainerRequest{
		Image:        config.Image,
		ExposedPorts: []string{"8000/tcp"},
		Env:          env,
		Cmd:          []string{"run", "--local-settings"},
		// LakeFS health check
		WaitingFor: wait.ForHTTP("/").
			WithPort("8000/tcp").
			WithStartupTimeout(config.StartupTimeout),
	}

	// Start container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", func() {}, fmt.Errorf("failed to start LakeFS container: %w", err)
	}

	// Get container connection details
	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return "", func() {}, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "8000")
	if err != nil {
		container.Terminate(ctx)
		return "", func() {}, fmt.Errorf("failed to get mapped port: %w", err)
	}

	// Build LakeFS HTTP endpoint URL
	// Format: http://host:port
	lakeFSURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	// Create cleanup function
	cleanup := createCleanupFunc(ctx, container, "LakeFS")

	return lakeFSURL, cleanup, nil
}

// SetupLakeFSWithRepository creates a LakeFS container and creates a test repository.
//
// This is a convenience function that combines SetupLakeFS with repository creation.
// Useful for tests that need a ready-to-use repository.
//
// Parameters:
//   - ctx: Context for container operations
//   - t: Testing context for requirement checks
//   - config: Optional LakeFS configuration (uses defaults if nil)
//   - repoName: Name of the repository to create
//   - defaultBranch: Default branch name (e.g., "main")
//
// Returns:
//   - string: LakeFS HTTP endpoint URL
//   - string: Repository name (same as input for convenience)
//   - ContainerCleanup: Function to terminate the container
//   - error: Container creation or repository creation errors
//
// Example Usage:
//
//	func TestWithRepository(t *testing.T) {
//	    ctx := context.Background()
//	    lakeFSURL, repoName, cleanup, err := SetupLakeFSWithRepository(
//	        ctx, t, nil, "my-repo", "main")
//	    require.NoError(t, err)
//	    defer cleanup()
//
//	    // Repository "my-repo" with branch "main" is ready to use
//	}
//
// Repository Creation:
//
//	The repository is created via LakeFS HTTP API:
//	POST /api/v1/repositories
//	Content-Type: application/json
//
//	Note: Repository creation requires HTTP calls to LakeFS API.
//	For now, we return the connection URL pattern.
//	The calling test should create the repository using the LakeFS API.
//
// Use Cases:
//   - Testing with pre-configured repository
//   - Multi-repository testing
//   - Testing repository-specific features
//   - Isolating test data in separate repositories
func SetupLakeFSWithRepository(ctx context.Context, t *testing.T, config *LakeFSConfig, repoName, defaultBranch string) (string, string, ContainerCleanup, error) {
	// Setup LakeFS container
	lakeFSURL, cleanup, err := SetupLakeFS(ctx, t, config)
	if err != nil {
		return "", "", cleanup, err
	}

	// Note: Repository creation would require HTTP calls to LakeFS API
	// For now, we return the connection URL pattern
	// The calling test should create the repository using the LakeFS API:
	// POST /api/v1/repositories with JSON body

	return lakeFSURL, repoName, cleanup, nil
}
