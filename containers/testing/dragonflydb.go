package testing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// DragonflyDBConfig holds configuration for DragonflyDB testcontainer setup.
type DragonflyDBConfig struct {
	// Image is the Docker image to use (default: "docker.dragonflydb.io/dragonflydb/dragonfly:v1.34.1")
	Image string
	// StartupTimeout is the maximum time to wait for DragonflyDB to be ready (default: 30s)
	StartupTimeout time.Duration
	// Password is the optional password for DragonflyDB authentication (empty = no auth)
	Password string
}

// DefaultDragonflyDBConfig returns the default DragonflyDB configuration for testing.
func DefaultDragonflyDBConfig() DragonflyDBConfig {
	return DragonflyDBConfig{
		Image:          "docker.dragonflydb.io/dragonflydb/dragonfly:v1.34.1",
		StartupTimeout: 30 * time.Second,
		Password:       "", // No authentication by default for testing
	}
}

// SetupDragonflyDB creates a DragonflyDB container for integration testing.
//
// DragonflyDB is a modern Redis-compatible in-memory data store. This function starts a
// DragonflyDB container using testcontainers-go and returns the connection address and
// a cleanup function.
//
// Container Configuration:
//   - Image: docker.dragonflydb.io/dragonflydb/dragonfly:v1.34.1
//   - Port: 6379/tcp (Redis protocol compatible)
//   - Wait Strategy: TCP connection check on port 6379
//   - Memory Lock: Unlimited (required for optimal performance)
//
// Parameters:
//   - ctx: Context for container operations
//   - t: Testing context for requirement checks
//   - config: Optional DragonflyDB configuration (uses defaults if nil)
//
// Returns:
//   - string: DragonflyDB connection address (e.g., "localhost:32770")
//   - ContainerCleanup: Function to terminate the container
//   - error: Container creation or startup errors
//
// Example Usage:
//
//	func TestDragonflyDBIntegration(t *testing.T) {
//	    ctx := context.Background()
//	    dfdbAddr, cleanup, err := SetupDragonflyDB(ctx, t, nil)
//	    require.NoError(t, err)
//	    defer cleanup()
//
//	    // Connect to DragonflyDB using Redis client
//	    client := redis.NewClient(&redis.Options{
//	        Addr: dfdbAddr,
//	    })
//	    defer client.Close()
//
//	    // Use DragonflyDB
//	    err = client.Set(ctx, "key", "value", 0).Err()
//	    require.NoError(t, err)
//	}
//
// Redis Compatibility:
//
//	DragonflyDB is fully compatible with Redis protocol and commands.
//	You can use any Redis client library to connect and interact with it.
//
//	Supported clients:
//	- Go: github.com/redis/go-redis/v9
//	- Python: redis-py
//	- Node.js: ioredis, redis
//	- Java: Jedis, Lettuce
//
// Performance Features:
//
//	DragonflyDB provides significant performance improvements over Redis:
//	- Multi-threaded architecture
//	- Better memory efficiency
//	- Faster snapshot operations
//	- Optimized for modern hardware
//
// Authentication:
//
//	Authentication is optional for testing. Set config.Password to enable:
//
//	config := DefaultDragonflyDBConfig()
//	config.Password = "secret"
//	dfdbAddr, cleanup, err := SetupDragonflyDB(ctx, t, &config)
//
//	Then connect with password:
//	client := redis.NewClient(&redis.Options{
//	    Addr:     dfdbAddr,
//	    Password: "secret",
//	})
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
func SetupDragonflyDB(ctx context.Context, t *testing.T, config *DragonflyDBConfig) (string, ContainerCleanup, error) {
	// Use default config if none provided
	if config == nil {
		defaultConfig := DefaultDragonflyDBConfig()
		config = &defaultConfig
	}

	// Build container request
	req := testcontainers.ContainerRequest{
		Image:        config.Image,
		ExposedPorts: []string{"6379/tcp"},
		// DragonflyDB starts very quickly (< 1 second typically)
		// TCP connection check is sufficient to verify readiness
		WaitingFor: wait.ForListeningPort("6379/tcp").
			WithStartupTimeout(config.StartupTimeout),
	}

	// Add password if configured
	if config.Password != "" {
		req.Env = map[string]string{
			"DRAGONFLY_PASSWORD": config.Password,
		}
	}

	// Start container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", func() {}, fmt.Errorf("failed to start DragonflyDB container: %w", err)
	}

	// Get container connection details
	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return "", func() {}, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		_ = container.Terminate(ctx)
		return "", func() {}, fmt.Errorf("failed to get mapped port: %w", err)
	}

	// Build connection address (host:port format for Redis clients)
	addr := fmt.Sprintf("%s:%s", host, port.Port())

	// Create cleanup function
	cleanup := createCleanupFunc(ctx, container, "DragonflyDB")

	return addr, cleanup, nil
}
