package testing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// OpenSearchDashboardsConfig holds configuration for OpenSearch Dashboards testcontainer setup.
type OpenSearchDashboardsConfig struct {
	// Image is the Docker image to use (default: "opensearchproject/opensearch-dashboards:3.0.0")
	Image string
	// OpenSearchURL is the URL to the OpenSearch instance
	OpenSearchURL string
	// DisableSecurity disables OpenSearch Dashboards security for testing (default: true)
	DisableSecurity bool
	// StartupTimeout is the maximum time to wait for Dashboards to be ready (default: 120s)
	StartupTimeout time.Duration
}

// DefaultOpenSearchDashboardsConfig returns the default OpenSearch Dashboards configuration for testing.
func DefaultOpenSearchDashboardsConfig(opensearchURL string) OpenSearchDashboardsConfig {
	return OpenSearchDashboardsConfig{
		Image:           "opensearchproject/opensearch-dashboards:3.0.0",
		OpenSearchURL:   opensearchURL,
		DisableSecurity: true,
		StartupTimeout:  120 * time.Second,
	}
}

// SetupOpenSearchDashboards creates an OpenSearch Dashboards container for integration testing.
//
// OpenSearch Dashboards is the visualization and user interface for OpenSearch. This function
// starts an OpenSearch Dashboards container using testcontainers-go and returns the connection URL
// and a cleanup function.
//
// Container Configuration:
//   - Image: opensearchproject/opensearch-dashboards:3.0.0 (visualization UI)
//   - Port: 5601/tcp (HTTP UI)
//   - Connection: Links to OpenSearch instance
//   - Security: Disabled by default for testing
//   - Wait Strategy: HTTP GET /api/status returning 200 OK
//
// Parameters:
//   - ctx: Context for container operations
//   - t: Testing context for requirement checks
//   - config: Optional OpenSearch Dashboards configuration (uses defaults if nil)
//
// Returns:
//   - string: OpenSearch Dashboards HTTP endpoint URL
//            (e.g., "http://localhost:32801")
//   - ContainerCleanup: Function to terminate the container
//   - error: Container creation or startup errors
//
// Example Usage:
//
//	func TestOpenSearchDashboardsIntegration(t *testing.T) {
//	    ctx := context.Background()
//
//	    // First, start OpenSearch
//	    opensearchURL, cleanupOS, err := SetupOpenSearch(ctx, t, nil)
//	    require.NoError(t, err)
//	    defer cleanupOS()
//
//	    // Then, start OpenSearch Dashboards
//	    config := DefaultOpenSearchDashboardsConfig(opensearchURL)
//	    dashboardsURL, cleanup, err := SetupOpenSearchDashboards(ctx, t, &config)
//	    require.NoError(t, err)
//	    defer cleanup()
//
//	    // Open Dashboards UI in browser or test via API
//	    resp, err := http.Get(dashboardsURL + "/api/status")
//	    require.NoError(t, err)
//	    defer resp.Body.Close()
//	}
//
// OpenSearch Dashboards Features:
//
//	Visualization and management interface:
//	- Discover: Explore and search data
//	- Visualize: Create charts, graphs, and visualizations
//	- Dashboards: Combine visualizations into dashboards
//	- Dev Tools: Console for running queries
//	- Management: Index patterns, saved objects, settings
//	- Alerting: Create and manage alerts
//	- Reports: Generate and schedule reports
//	- Notebooks: Interactive analysis notebooks
//
// UI Endpoints:
//
//	Key UI paths available:
//	- GET  / - Home page
//	- GET  /app/home - Application home
//	- GET  /app/discover - Data discovery
//	- GET  /app/dashboards - Dashboard viewer
//	- GET  /app/visualize - Visualization editor
//	- GET  /app/dev_tools - Developer tools console
//	- GET  /api/status - Health status API
//
// API Endpoints:
//
//	REST API for automation:
//	- GET    /api/status - Application status
//	- GET    /api/saved_objects - List saved objects
//	- POST   /api/saved_objects/{type} - Create saved object
//	- PUT    /api/saved_objects/{type}/{id} - Update saved object
//	- DELETE /api/saved_objects/{type}/{id} - Delete saved object
//
// Security:
//
//	For testing, security is disabled by default:
//	- No authentication required
//	- No TLS/SSL
//	- Open access to all features
//
//	For production, enable security plugin:
//	- Authentication (basic, SAML, OIDC, etc.)
//	- TLS/SSL encryption
//	- Role-based access control (RBAC)
//	- Multi-tenancy support
//
// Connection to OpenSearch:
//
//	OpenSearch Dashboards requires a running OpenSearch instance.
//	Configure the connection via OPENSEARCH_HOSTS environment variable.
//
//	The config.OpenSearchURL should point to the OpenSearch REST API:
//	- http://localhost:9200 (from host)
//	- http://opensearch:9200 (from Docker network)
//
// Performance:
//
//	OpenSearch Dashboards container starts in 30-60 seconds typically.
//	The wait strategy ensures the UI is fully initialized and ready
//	to accept requests before returning.
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
//	- OpenSearch not accessible (check URL)
//	- Connection timeout (increase StartupTimeout)
func SetupOpenSearchDashboards(ctx context.Context, t *testing.T, config *OpenSearchDashboardsConfig) (string, ContainerCleanup, error) {
	// Use default config if none provided
	if config == nil {
		return "", func() {}, fmt.Errorf("config is required with OpenSearchURL")
	}

	if config.OpenSearchURL == "" {
		return "", func() {}, fmt.Errorf("OpenSearchURL is required in config")
	}

	// Build environment variables
	env := map[string]string{
		"OPENSEARCH_HOSTS": config.OpenSearchURL,
	}

	// Disable security for testing if requested
	if config.DisableSecurity {
		env["DISABLE_SECURITY_DASHBOARDS_PLUGIN"] = "true"
	}

	// Create container request
	req := testcontainers.ContainerRequest{
		Image: config.Image,
		ExposedPorts: []string{
			"5601/tcp", // HTTP UI
		},
		Env: env,
		// OpenSearch Dashboards UI readiness check
		WaitingFor: wait.ForHTTP("/api/status").
			WithPort("5601/tcp").
			WithStartupTimeout(config.StartupTimeout),
	}

	// Start container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", func() {}, fmt.Errorf("failed to start OpenSearch Dashboards container: %w", err)
	}

	// Get container connection details
	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return "", func() {}, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "5601")
	if err != nil {
		container.Terminate(ctx)
		return "", func() {}, fmt.Errorf("failed to get mapped port: %w", err)
	}

	// Build OpenSearch Dashboards HTTP endpoint URL
	// Format: http://host:port
	dashboardsURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	// Create cleanup function
	cleanup := createCleanupFunc(ctx, container, "OpenSearch Dashboards")

	return dashboardsURL, cleanup, nil
}

// SetupOpenSearchWithDashboards creates both OpenSearch and OpenSearch Dashboards containers.
//
// This is a convenience function that combines SetupOpenSearch with SetupOpenSearchDashboards.
// Useful for tests that need a complete search stack.
//
// Parameters:
//   - ctx: Context for container operations
//   - t: Testing context for requirement checks
//   - opensearchConfig: Optional OpenSearch configuration (uses defaults if nil)
//   - dashboardsConfig: Optional Dashboards configuration (OpenSearchURL will be set automatically)
//
// Returns:
//   - string: OpenSearch HTTP endpoint URL
//   - string: OpenSearch Dashboards HTTP endpoint URL
//   - ContainerCleanup: Function to terminate both containers
//   - error: Container creation or startup errors
//
// Example Usage:
//
//	func TestFullStack(t *testing.T) {
//	    ctx := context.Background()
//	    opensearchURL, dashboardsURL, cleanup, err := SetupOpenSearchWithDashboards(
//	        ctx, t, nil, nil)
//	    require.NoError(t, err)
//	    defer cleanup()
//
//	    // Both OpenSearch and Dashboards are ready to use
//	    // Test via API or UI
//	}
//
// Cleanup:
//
//	The returned cleanup function will terminate both containers.
//	Always defer the cleanup function to ensure proper cleanup.
//
// Use Cases:
//   - End-to-end testing with complete stack
//   - UI automation testing
//   - Integration tests requiring visualization
//   - Testing saved objects and dashboards
func SetupOpenSearchWithDashboards(ctx context.Context, t *testing.T, opensearchConfig *OpenSearchConfig, dashboardsConfig *OpenSearchDashboardsConfig) (string, string, ContainerCleanup, error) {
	// Setup OpenSearch container
	opensearchURL, cleanupOS, err := SetupOpenSearch(ctx, t, opensearchConfig)
	if err != nil {
		return "", "", cleanupOS, err
	}

	// Setup Dashboards configuration
	if dashboardsConfig == nil {
		defaultDashboardsConfig := DefaultOpenSearchDashboardsConfig(opensearchURL)
		dashboardsConfig = &defaultDashboardsConfig
	} else {
		// Override OpenSearchURL with the actual container URL
		dashboardsConfig.OpenSearchURL = opensearchURL
	}

	// Setup OpenSearch Dashboards container
	dashboardsURL, cleanupDashboards, err := SetupOpenSearchDashboards(ctx, t, dashboardsConfig)
	if err != nil {
		cleanupOS() // Clean up OpenSearch if Dashboards setup fails
		return "", "", func() {}, err
	}

	// Create combined cleanup function
	combinedCleanup := func() {
		cleanupDashboards()
		cleanupOS()
	}

	return opensearchURL, dashboardsURL, combinedCleanup, nil
}
