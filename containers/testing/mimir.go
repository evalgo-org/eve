package testing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// MimirConfig holds configuration for Grafana Mimir testcontainer setup.
type MimirConfig struct {
	// Image is the Docker image to use (default: "grafana/mimir:2.17.2")
	Image string
	// StartupTimeout is the maximum time to wait for Mimir to be ready (default: 120s)
	StartupTimeout time.Duration
}

// DefaultMimirConfig returns the default Grafana Mimir configuration for testing.
func DefaultMimirConfig() MimirConfig {
	return MimirConfig{
		Image:          "grafana/mimir:2.17.2",
		StartupTimeout: 120 * time.Second,
	}
}

// SetupMimir creates a Grafana Mimir container for integration testing.
//
// Grafana Mimir is an open-source, horizontally scalable, highly available, multi-tenant,
// long-term storage for Prometheus metrics. This function starts a Mimir container using
// testcontainers-go and returns the connection URL and a cleanup function.
//
// Container Configuration:
//   - Image: grafana/mimir:2.17.2 (metrics long-term storage)
//   - Port: 9009/tcp (HTTP API)
//   - Mode: Monolithic (single binary, all components)
//   - Wait Strategy: HTTP GET /ready returning 200 OK
//
// Parameters:
//   - ctx: Context for container operations
//   - t: Testing context for requirement checks
//   - config: Optional Mimir configuration (uses defaults if nil)
//
// Returns:
//   - string: Mimir HTTP endpoint URL
//     (e.g., "http://localhost:32792")
//   - ContainerCleanup: Function to terminate the container
//   - error: Container creation or startup errors
//
// Example Usage:
//
//	func TestMimirIntegration(t *testing.T) {
//	    ctx := context.Background()
//	    mimirURL, cleanup, err := SetupMimir(ctx, t, nil)
//	    require.NoError(t, err)
//	    defer cleanup()
//
//	    // Use Mimir API
//	    resp, err := http.Get(mimirURL + "/ready")
//	    require.NoError(t, err)
//	    defer resp.Body.Close()
//
//	    // Mimir is ready for ingesting metrics
//	}
//
// Grafana Mimir Features:
//
//	Open-source metrics storage with:
//	- Horizontally scalable architecture
//	- Multi-tenancy support
//	- Long-term storage for Prometheus metrics
//	- PromQL query engine
//	- High availability
//	- Object storage backend (S3, GCS, Azure Blob)
//	- Recording and alerting rules
//	- Grafana integration
//	- Prometheus remote_write API
//
// API Endpoints:
//
//	Key endpoints available:
//	- GET  /ready - Readiness check
//	- GET  /metrics - Prometheus metrics
//	- POST /api/v1/push - Push metrics (remote_write)
//	- GET  /prometheus/api/v1/query - Query metrics (PromQL)
//	- GET  /prometheus/api/v1/query_range - Range query
//	- GET  /prometheus/api/v1/series - Series metadata
//	- GET  /prometheus/api/v1/labels - Label names
//	- GET  /prometheus/api/v1/label/{name}/values - Label values
//	- POST /prometheus/api/v1/rules - Configure recording/alerting rules
//
// Prometheus Configuration:
//
//	Configure Prometheus to remote_write to Mimir:
//	remote_write:
//	  - url: http://localhost:9009/api/v1/push
//	    headers:
//	      X-Scope-OrgID: "demo"  # Tenant ID for multi-tenancy
//
// Multi-Tenancy:
//
//	Mimir uses HTTP headers for tenant isolation:
//	X-Scope-OrgID: tenant-id
//
//	For testing, use "demo" or "anonymous" as tenant ID.
//	All requests must include this header.
//
// Performance:
//
//	Mimir container starts in 30-60 seconds typically.
//	The wait strategy ensures the API is fully initialized and
//	ready to accept requests before returning.
//
// Data Storage:
//
//	Mimir stores data in /data inside the container.
//	For testing, this is ephemeral (lost when container stops).
//	This ensures test isolation.
//
// Monolithic Mode:
//
//	For testing, Mimir runs in monolithic mode where all components
//	(distributor, ingester, querier, etc.) run in a single process.
//
//	For production, use microservices mode with separate components.
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
//	- Insufficient memory (Mimir requires ~512MB minimum)
//
// Use Cases:
//
//	Integration testing scenarios:
//	- Testing Prometheus remote_write integration
//	- Testing PromQL queries
//	- Testing long-term metrics storage
//	- Testing alerting rules
//	- Testing Grafana data source connections
//	- Testing multi-tenant scenarios
func SetupMimir(ctx context.Context, t *testing.T, config *MimirConfig) (string, ContainerCleanup, error) {
	// Use default config if none provided
	if config == nil {
		defaultConfig := DefaultMimirConfig()
		config = &defaultConfig
	}

	// Create container request
	req := testcontainers.ContainerRequest{
		Image: config.Image,
		ExposedPorts: []string{
			"9009/tcp", // HTTP API
		},
		Cmd: []string{"-config.file=/etc/mimir/demo.yaml"},
		// Mimir readiness check
		WaitingFor: wait.ForHTTP("/ready").
			WithPort("9009/tcp").
			WithStartupTimeout(config.StartupTimeout),
	}

	// Start container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", func() {}, fmt.Errorf("failed to start Mimir container: %w", err)
	}

	// Get container connection details
	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return "", func() {}, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "9009")
	if err != nil {
		_ = container.Terminate(ctx)
		return "", func() {}, fmt.Errorf("failed to get mapped port: %w", err)
	}

	// Build Mimir HTTP endpoint URL
	// Format: http://host:port
	mimirURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	// Create cleanup function
	cleanup := createCleanupFunc(ctx, container, "Grafana Mimir")

	return mimirURL, cleanup, nil
}

// SetupMimirWithTenant creates a Mimir container and returns URLs for a specific tenant.
//
// This is a convenience function that formats tenant-specific URLs.
//
// Parameters:
//   - ctx: Context for container operations
//   - t: Testing context for requirement checks
//   - config: Optional Mimir configuration (uses defaults if nil)
//   - tenantID: Tenant ID for multi-tenancy (e.g., "demo", "tenant-1")
//
// Returns:
//   - string: Mimir HTTP endpoint URL
//   - string: Tenant ID (same as input for convenience)
//   - ContainerCleanup: Function to terminate the container
//   - error: Container creation or startup errors
//
// Example Usage:
//
//	func TestWithTenant(t *testing.T) {
//	    ctx := context.Background()
//	    mimirURL, tenantID, cleanup, err := SetupMimirWithTenant(
//	        ctx, t, nil, "demo")
//	    require.NoError(t, err)
//	    defer cleanup()
//
//	    // All requests should include X-Scope-OrgID: demo header
//	    req, _ := http.NewRequest("GET", mimirURL+"/ready", nil)
//	    req.Header.Set("X-Scope-OrgID", tenantID)
//	}
//
// Tenant Configuration:
//
//	When making requests to Mimir, always include the tenant header:
//	X-Scope-OrgID: {tenantID}
//
//	This is required for multi-tenancy isolation.
func SetupMimirWithTenant(ctx context.Context, t *testing.T, config *MimirConfig, tenantID string) (string, string, ContainerCleanup, error) {
	// Setup Mimir container
	mimirURL, cleanup, err := SetupMimir(ctx, t, config)
	if err != nil {
		return "", "", cleanup, err
	}

	// Return URL and tenant ID
	return mimirURL, tenantID, cleanup, nil
}
