package testing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// DockerStatsExporterConfig holds configuration for Docker Stats Exporter testcontainer setup.
type DockerStatsExporterConfig struct {
	// Image is the Docker image to use (default: "ghcr.io/grzegorzmika/docker_stats_exporter:latest")
	Image string
	// StartupTimeout is the maximum time to wait for Docker Stats Exporter to be ready (default: 60s)
	StartupTimeout time.Duration
}

// DefaultDockerStatsExporterConfig returns the default Docker Stats Exporter configuration for testing.
func DefaultDockerStatsExporterConfig() DockerStatsExporterConfig {
	return DockerStatsExporterConfig{
		Image:          "ghcr.io/grzegorzmika/docker_stats_exporter:latest",
		StartupTimeout: 60 * time.Second,
	}
}

// SetupDockerStatsExporter creates a Docker Stats Exporter container for integration testing.
//
// Docker Stats Exporter is a Prometheus exporter that exposes Docker container statistics
// (CPU, memory, network, disk I/O) in Prometheus format. This function starts a Docker Stats
// Exporter container using testcontainers-go and returns the metrics endpoint URL and a cleanup function.
//
// Container Configuration:
//   - Image: ghcr.io/grzegorzmika/docker_stats_exporter:latest (Docker container metrics exporter)
//   - Port: 8080/tcp (Prometheus metrics endpoint)
//   - Docker Socket: /var/run/docker.sock mounted read-only for container stats access
//   - Wait Strategy: HTTP GET /metrics returning 200 OK
//
// Parameters:
//   - ctx: Context for container operations
//   - t: Testing context for requirement checks
//   - config: Optional Docker Stats Exporter configuration (uses defaults if nil)
//
// Returns:
//   - string: Docker Stats Exporter metrics endpoint URL
//     (e.g., "http://localhost:32793/metrics")
//   - ContainerCleanup: Function to terminate the container
//   - error: Container creation or startup errors
//
// Example Usage:
//
//	func TestDockerStatsExporterIntegration(t *testing.T) {
//	    ctx := context.Background()
//	    metricsURL, cleanup, err := SetupDockerStatsExporter(ctx, t, nil)
//	    require.NoError(t, err)
//	    defer cleanup()
//
//	    // Use Docker Stats Exporter metrics endpoint
//	    resp, err := http.Get(metricsURL)
//	    require.NoError(t, err)
//	    defer resp.Body.Close()
//
//	    // Docker Stats Exporter is ready for scraping container metrics
//	}
//
// Docker Stats Exporter Features:
//
//	Prometheus exporter for Docker container statistics:
//	- Real-time container CPU usage metrics
//	- Memory usage and limits per container
//	- Network I/O statistics (bytes sent/received)
//	- Disk I/O statistics (read/write operations)
//	- Container state and metadata
//	- Per-container resource consumption
//	- Prometheus exposition format
//	- Low overhead monitoring
//	- Label-based container filtering
//
// Metrics Exposed:
//
//	Key metrics available at /metrics endpoint:
//	- container_cpu_usage_percent - CPU usage percentage per container
//	- container_memory_usage_bytes - Memory usage in bytes
//	- container_memory_limit_bytes - Memory limit in bytes
//	- container_network_receive_bytes_total - Network bytes received
//	- container_network_transmit_bytes_total - Network bytes transmitted
//	- container_block_read_bytes_total - Disk bytes read
//	- container_block_write_bytes_total - Disk bytes written
//	- container_state - Container running state (0=stopped, 1=running)
//	- container_info - Container metadata (name, image, labels)
//
// Prometheus Configuration:
//
//	Configure Prometheus to scrape Docker Stats Exporter:
//	scrape_configs:
//	  - job_name: 'docker-stats'
//	    static_configs:
//	      - targets: ['localhost:8080']
//	    scrape_interval: 15s
//
// Docker Socket Access:
//
//	The exporter requires read access to the Docker socket to collect container statistics:
//	- Mount: /var/run/docker.sock:/var/run/docker.sock:ro
//	- Read-only access for security
//	- Required for accessing Docker API
//	- Collects stats via Docker Engine API
//
// Performance:
//
//	Docker Stats Exporter container starts in 5-15 seconds typically.
//	The wait strategy ensures the metrics endpoint is fully initialized and
//	ready to serve metrics before returning.
//
// Use Cases:
//
//	Integration testing scenarios:
//	- Testing container monitoring infrastructure
//	- Testing Prometheus scraping of Docker metrics
//	- Testing container resource usage tracking
//	- Testing Grafana dashboards for Docker metrics
//	- Testing alerting rules based on container metrics
//	- Testing multi-container resource monitoring
//
// Monitoring Docker Infrastructure:
//
//	Docker Stats Exporter enables monitoring of:
//	- Container resource utilization (CPU, memory, network, disk)
//	- Container health and availability
//	- Resource limits and throttling
//	- Network traffic patterns per container
//	- Disk I/O patterns and bottlenecks
//	- Container lifecycle events
//	- Microservices resource consumption
//
// Security Considerations:
//
//	Docker socket access security:
//	- Read-only mount prevents container manipulation
//	- Exporter can only read container stats, not control containers
//	- No write access to Docker daemon
//	- Suitable for monitoring in production environments
//	- Consider using Docker API over TCP with TLS for remote monitoring
//	- Limit exporter to monitoring role only
//
// Label-Based Filtering:
//
//	The exporter includes container labels in metrics, enabling:
//	- Filtering by container labels
//	- Grouping metrics by label selectors
//	- Service-specific monitoring
//	- Environment-based filtering (dev/staging/prod)
//	- Team or project-based metrics grouping
//
// Integration with Grafana:
//
//	Create Grafana dashboards using Docker Stats Exporter metrics:
//	- Real-time container resource dashboards
//	- Per-container CPU and memory graphs
//	- Network traffic visualization
//	- Disk I/O monitoring panels
//	- Container state overview
//	- Resource usage trends and forecasting
//
// Integration with Prometheus:
//
//	Query Docker container metrics using PromQL:
//	- rate(container_cpu_usage_percent[5m]) - CPU usage rate
//	- container_memory_usage_bytes / container_memory_limit_bytes - Memory utilization
//	- rate(container_network_receive_bytes_total[5m]) - Network receive rate
//	- sum(container_memory_usage_bytes) by (container_name) - Memory by container
//
// Alerting:
//
//	Configure Prometheus alerts for container resources:
//	- High CPU usage per container
//	- Memory limit approaching or exceeded
//	- Container crashes or restarts
//	- Network anomalies
//	- Disk I/O bottlenecks
//	- Container state changes
//
// Data Storage:
//
//	Docker Stats Exporter is stateless and does not store data.
//	All metrics are collected in real-time from the Docker daemon.
//	This ensures test isolation and no cleanup required.
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
//	- Docker socket not accessible or not mounted
//	- Permission denied accessing Docker socket
//
// Comparison with cAdvisor:
//
//	Docker Stats Exporter vs cAdvisor:
//	- Docker Stats Exporter: Lightweight, Docker-specific, simple setup
//	- cAdvisor: More comprehensive, supports multiple container runtimes, heavier
//	- Docker Stats Exporter: Focused on Docker containers only
//	- cAdvisor: Supports Docker, containerd, CRI-O, and more
//	- Choose Docker Stats Exporter for Docker-only environments
//	- Choose cAdvisor for multi-runtime Kubernetes clusters
//
// Limitations:
//
//	Be aware of these limitations:
//	- Requires Docker socket access (security consideration)
//	- Only monitors Docker containers (not processes or host metrics)
//	- Metrics granularity limited by Docker stats API
//	- Historical data requires Prometheus long-term storage
//	- No built-in alerting (use Prometheus Alertmanager)
//
// Best Practices:
//
//	For production monitoring:
//	- Use read-only Docker socket mount
//	- Configure appropriate scrape intervals (15-30s)
//	- Set up alerts for critical resource thresholds
//	- Monitor exporter health and availability
//	- Use persistent Prometheus storage
//	- Create comprehensive Grafana dashboards
//	- Document alert runbooks
//	- Test alerts and dashboards regularly
func SetupDockerStatsExporter(ctx context.Context, t *testing.T, config *DockerStatsExporterConfig) (string, ContainerCleanup, error) {
	// Use default config if none provided
	if config == nil {
		defaultConfig := DefaultDockerStatsExporterConfig()
		config = &defaultConfig
	}

	// Create container request with Docker socket mount
	req := testcontainers.ContainerRequest{
		Image:        config.Image,
		ExposedPorts: []string{"8080/tcp"},
		// Mount Docker socket for container stats access (read-only)
		Binds: []string{"/var/run/docker.sock:/var/run/docker.sock:ro"},
		// Docker Stats Exporter metrics endpoint check
		WaitingFor: wait.ForHTTP("/metrics").
			WithPort("8080/tcp").
			WithStartupTimeout(config.StartupTimeout),
	}

	// Start container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", func() {}, fmt.Errorf("failed to start Docker Stats Exporter container: %w", err)
	}

	// Get container connection details
	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return "", func() {}, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "8080")
	if err != nil {
		_ = container.Terminate(ctx)
		return "", func() {}, fmt.Errorf("failed to get mapped port: %w", err)
	}

	// Build Docker Stats Exporter metrics endpoint URL
	// Format: http://host:port/metrics
	metricsURL := fmt.Sprintf("http://%s:%s/metrics", host, port.Port())

	// Create cleanup function
	cleanup := createCleanupFunc(ctx, container, "Docker Stats Exporter")

	return metricsURL, cleanup, nil
}
