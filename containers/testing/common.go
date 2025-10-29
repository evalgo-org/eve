// Package testing provides testcontainers-based container setup for integration tests.
//
// This package uses testcontainers-go to create ephemeral containers for testing
// purposes. Containers are automatically cleaned up after tests complete.
//
// Key Features:
//   - Ephemeral containers with automatic cleanup
//   - Randomized port allocation to avoid conflicts
//   - Wait strategies for service readiness
//   - Integration test isolation
//
// Build Tags:
//
//	Integration tests using this package should use the integration build tag:
//	//go:build integration
//
// Example Usage:
//
//	func TestMyService(t *testing.T) {
//	    ctx := context.Background()
//	    baseXURL, cleanup, err := SetupBaseX(ctx, t)
//	    require.NoError(t, err)
//	    defer cleanup()
//	    // Use baseXURL for testing...
//	}
package testing

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
)

// ContainerCleanup is a function type for cleaning up test containers.
// Call this function in defer to ensure containers are terminated after tests.
type ContainerCleanup func()

// createCleanupFunc creates a standardized cleanup function for testcontainers.
// This ensures consistent cleanup behavior across all container types.
//
// Parameters:
//   - ctx: Context for container operations
//   - container: The testcontainer to clean up
//   - containerType: Human-readable name for logging (e.g., "BaseX", "CouchDB")
//
// Returns:
//   - ContainerCleanup: Function that terminates the container
//
// Usage:
//
//	cleanup := createCleanupFunc(ctx, container, "BaseX")
//	defer cleanup()
func createCleanupFunc(ctx context.Context, container testcontainers.Container, containerType string) ContainerCleanup {
	return func() {
		if err := container.Terminate(ctx); err != nil {
			// Note: Using fmt.Printf since we can't access testing.T here
			fmt.Printf("Warning: Failed to terminate %s container: %v\n", containerType, err)
		}
	}
}

// getConnectionURL builds a connection URL from container host and port.
// This is a helper function for constructing database connection strings.
//
// Parameters:
//   - protocol: URL protocol (e.g., "http", "https", "postgresql")
//   - host: Container host (usually "localhost" for testcontainers)
//   - port: Mapped container port
//   - path: URL path (optional, can be empty)
//
// Returns:
//   - string: Complete connection URL
//
// Example:
//
//	url := getConnectionURL("http", "localhost", "8984", "/rest")
//	// Returns: "http://localhost:8984/rest"
func getConnectionURL(protocol, host, port, path string) string {
	if path != "" {
		return fmt.Sprintf("%s://%s:%s%s", protocol, host, port, path)
	}
	return fmt.Sprintf("%s://%s:%s", protocol, host, port)
}
