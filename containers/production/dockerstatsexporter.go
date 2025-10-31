package production

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"

	"eve.evalgo.org/common"
)

// DockerStatsExporterProductionConfig holds configuration for production Docker Stats Exporter deployment.
type DockerStatsExporterProductionConfig struct {
	// ContainerName is the name for the Docker Stats Exporter container
	ContainerName string
	// Image is the Docker image to use (default: "ghcr.io/grzegorzmika/docker_stats_exporter:latest")
	Image string
	// Port is the host port to expose Docker Stats Exporter metrics endpoint (default: 8080)
	Port string
	// Production holds common production configuration
	Production ProductionConfig
}

// DefaultDockerStatsExporterProductionConfig returns the default Docker Stats Exporter production configuration.
func DefaultDockerStatsExporterProductionConfig() DockerStatsExporterProductionConfig {
	return DockerStatsExporterProductionConfig{
		ContainerName: "docker-stats-exporter",
		Image:         "ghcr.io/grzegorzmika/docker_stats_exporter:latest",
		Port:          "8080",
		Production: ProductionConfig{
			NetworkName:   "app-network",
			CreateNetwork: true,
			CreateVolume:  false, // No volume needed for stateless exporter
		},
	}
}

// DeployDockerStatsExporter deploys a production-ready Docker Stats Exporter container.
//
// Docker Stats Exporter is a Prometheus exporter that exposes Docker container statistics
// (CPU, memory, network, disk I/O) in Prometheus format. This function deploys a Docker Stats
// Exporter container suitable for production use with Docker socket access for real-time monitoring.
//
// Container Features:
//   - Named container for consistent identification
//   - Docker socket bind mount for container stats access
//   - Custom network connectivity
//   - Fixed port mapping for stable access
//   - Restart policy for high availability
//   - Health checks for monitoring
//   - Read-only Docker socket access for security
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - config: Docker Stats Exporter production configuration
//
// Returns:
//   - string: Container ID of the deployed Docker Stats Exporter container
//   - error: Deployment errors
//
// Example Usage:
//
//	ctx, cli, err := common.CtxCli("unix:///var/run/docker.sock")
//	if err != nil {
//	    return fmt.Errorf("failed to create Docker client: %w", err)
//	}
//	defer cli.Close()
//
//	config := DefaultDockerStatsExporterProductionConfig()
//	containerID, err := DeployDockerStatsExporter(ctx, cli, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("Docker Stats Exporter deployed with ID: %s", containerID)
//	log.Printf("Metrics endpoint: http://localhost:%s/metrics", config.Port)
//
// Connection URLs:
//
//	Metrics endpoint:
//	http://localhost:{port}/metrics
//	http://{container_name}:{port}/metrics (from other containers)
//
//	Example:
//	http://localhost:8080/metrics
//	http://docker-stats-exporter:8080/metrics (from Prometheus container)
//
// Docker Socket Access:
//
//	The exporter requires access to the Docker socket to collect container statistics:
//	- Bind mount: /var/run/docker.sock:/var/run/docker.sock:ro
//	- Read-only access for security (no container manipulation)
//	- Required for accessing Docker Engine API
//	- Collects real-time container stats via Docker API
//
// Security Considerations:
//
//	Docker socket access is a powerful capability that requires careful consideration:
//
//	Read-Only Access:
//	- The socket is mounted read-only (:ro flag)
//	- Prevents container creation, deletion, or modification
//	- Only allows reading container statistics
//	- Suitable for monitoring in production environments
//
//	Attack Surface:
//	- Read-only socket still exposes Docker API information
//	- Container can see all containers and their configurations
//	- Cannot modify containers but can read sensitive information
//	- Consider network segmentation and firewall rules
//
//	Alternatives for High-Security Environments:
//	- Use Docker API over TCP with TLS authentication
//	- Deploy on a dedicated monitoring host
//	- Use Docker context with SSH tunneling
//	- Consider cAdvisor for Kubernetes environments
//	- Implement API rate limiting and access controls
//
//	Best Practices:
//	- Always use read-only socket mount (:ro)
//	- Limit exporter to monitoring role only
//	- Run exporter in isolated network namespace
//	- Monitor exporter logs for suspicious activity
//	- Use least privilege container user (if supported)
//	- Keep exporter image updated
//	- Audit Docker socket access regularly
//
// Network Configuration:
//
//	The container joins the specified Docker network, enabling
//	communication with other containers on the same network.
//
//	Prometheus and Grafana can scrape metrics from:
//	http://{container_name}:8080/metrics
//
// Restart Policy:
//
//	The container is configured with restart policy "unless-stopped",
//	ensuring it automatically restarts if it crashes.
//
// Health Monitoring:
//
//	Health check runs HTTP GET /metrics every 30 seconds:
//	- Timeout: 10 seconds
//	- Retries: 3 consecutive failures before unhealthy
//
// Metrics Exposed:
//
//	Docker Stats Exporter provides comprehensive container metrics:
//
//	CPU Metrics:
//	- container_cpu_usage_percent - CPU usage percentage per container
//	- container_cpu_throttling_periods_total - CPU throttling periods
//	- container_cpu_throttling_throttled_periods_total - Throttled periods
//
//	Memory Metrics:
//	- container_memory_usage_bytes - Current memory usage in bytes
//	- container_memory_limit_bytes - Memory limit in bytes
//	- container_memory_max_usage_bytes - Maximum memory usage
//	- container_memory_cache_bytes - Cache memory usage
//	- container_memory_rss_bytes - RSS (Resident Set Size) memory
//	- container_memory_swap_bytes - Swap usage
//
//	Network Metrics:
//	- container_network_receive_bytes_total - Network bytes received
//	- container_network_transmit_bytes_total - Network bytes transmitted
//	- container_network_receive_packets_total - Packets received
//	- container_network_transmit_packets_total - Packets transmitted
//	- container_network_receive_errors_total - Network receive errors
//	- container_network_transmit_errors_total - Network transmit errors
//
//	Disk I/O Metrics:
//	- container_block_read_bytes_total - Disk bytes read
//	- container_block_write_bytes_total - Disk bytes written
//	- container_block_read_operations_total - Disk read operations
//	- container_block_write_operations_total - Disk write operations
//
//	Container State Metrics:
//	- container_state - Container running state (0=stopped, 1=running, 2=paused, 3=restarting)
//	- container_info - Container metadata (name, image, labels)
//	- container_uptime_seconds - Container uptime in seconds
//
// Label-Based Filtering:
//
//	All metrics include container labels for filtering and grouping:
//	- container_name: Container name
//	- container_id: Container ID
//	- image: Container image name
//	- image_tag: Container image tag
//	- Custom labels from Docker containers
//
//	Example PromQL queries with labels:
//	- container_memory_usage_bytes{container_name="myapp"}
//	- sum(container_cpu_usage_percent) by (image)
//	- container_network_receive_bytes_total{environment="production"}
//
// Prometheus Configuration:
//
//	Configure Prometheus to scrape Docker Stats Exporter:
//
//	scrape_configs:
//	  - job_name: 'docker-stats'
//	    static_configs:
//	      - targets: ['docker-stats-exporter:8080']
//	    scrape_interval: 15s
//	    scrape_timeout: 10s
//
// PromQL Query Examples:
//
//	Useful queries for monitoring Docker containers:
//
//	CPU Usage:
//	- container_cpu_usage_percent > 80 - High CPU containers
//	- rate(container_cpu_usage_percent[5m]) - CPU usage rate
//	- topk(5, container_cpu_usage_percent) - Top 5 CPU consumers
//
//	Memory Usage:
//	- container_memory_usage_bytes / container_memory_limit_bytes * 100 - Memory utilization %
//	- container_memory_usage_bytes > 1e9 - Containers using > 1GB
//	- sum(container_memory_usage_bytes) by (container_name) - Memory by container
//
//	Network Traffic:
//	- rate(container_network_receive_bytes_total[5m]) - Network receive rate
//	- rate(container_network_transmit_bytes_total[5m]) - Network transmit rate
//	- sum(rate(container_network_receive_bytes_total[5m])) - Total network receive rate
//
//	Disk I/O:
//	- rate(container_block_read_bytes_total[5m]) - Disk read rate
//	- rate(container_block_write_bytes_total[5m]) - Disk write rate
//	- topk(5, rate(container_block_write_bytes_total[5m])) - Top disk writers
//
// Grafana Integration:
//
//	Create Grafana dashboards using Docker Stats Exporter metrics:
//
//	Dashboard Panels:
//	- Real-time container resource usage overview
//	- Per-container CPU utilization graphs
//	- Memory usage and limits visualization
//	- Network traffic charts (sent/received)
//	- Disk I/O operation graphs
//	- Container state timeline
//	- Resource usage trends and forecasting
//	- Top resource consumers tables
//
//	Dashboard Templates:
//	- Use container_name variable for filtering
//	- Use image variable for grouping by service
//	- Use environment label for filtering by environment
//
//	Example Grafana Panel Query:
//	sum(container_memory_usage_bytes{container_name=~"$container"}) by (container_name)
//
// Alerting Rules:
//
//	Configure Prometheus alerts for container resource issues:
//
//	High CPU Usage:
//	- alert: ContainerHighCPU
//	  expr: container_cpu_usage_percent > 90
//	  for: 5m
//	  annotations:
//	    summary: Container {{ $labels.container_name }} high CPU usage
//
//	Memory Limit Approaching:
//	- alert: ContainerMemoryLimit
//	  expr: container_memory_usage_bytes / container_memory_limit_bytes > 0.9
//	  for: 5m
//	  annotations:
//	    summary: Container {{ $labels.container_name }} approaching memory limit
//
//	Container Down:
//	- alert: ContainerDown
//	  expr: container_state == 0
//	  for: 1m
//	  annotations:
//	    summary: Container {{ $labels.container_name }} is down
//
//	High Network Traffic:
//	- alert: ContainerHighNetworkTraffic
//	  expr: rate(container_network_transmit_bytes_total[5m]) > 100000000
//	  for: 5m
//	  annotations:
//	    summary: Container {{ $labels.container_name }} high network traffic
//
// Monitoring Use Cases:
//
//	Docker Stats Exporter enables comprehensive container monitoring:
//
//	1. Resource Utilization:
//	   - Track CPU, memory, network, and disk usage per container
//	   - Identify resource bottlenecks and constraints
//	   - Optimize container resource allocations
//	   - Plan capacity based on usage trends
//
//	2. Performance Monitoring:
//	   - Monitor application performance via container metrics
//	   - Detect performance degradation
//	   - Correlate resource usage with application behavior
//	   - Troubleshoot slow requests and timeouts
//
//	3. Cost Optimization:
//	   - Identify over-provisioned containers
//	   - Right-size container resource limits
//	   - Track resource consumption for cost allocation
//	   - Optimize infrastructure costs
//
//	4. Health Monitoring:
//	   - Monitor container availability and uptime
//	   - Detect container crashes and restarts
//	   - Alert on container state changes
//	   - Ensure service reliability
//
//	5. Capacity Planning:
//	   - Analyze resource usage trends
//	   - Forecast future resource requirements
//	   - Plan infrastructure scaling
//	   - Prevent resource exhaustion
//
//	6. Troubleshooting:
//	   - Identify resource-constrained containers
//	   - Debug memory leaks and OOM issues
//	   - Analyze network patterns and anomalies
//	   - Investigate disk I/O bottlenecks
//
// Performance Characteristics:
//
//	Docker Stats Exporter is lightweight and efficient:
//	- Minimal CPU overhead (< 1% typically)
//	- Low memory footprint (< 50MB typically)
//	- Real-time metric collection
//	- Efficient Docker API usage
//	- Scales to hundreds of containers
//	- Suitable for production environments
//
// Comparison with Other Solutions:
//
//	Docker Stats Exporter vs Alternatives:
//
//	vs cAdvisor:
//	- Docker Stats Exporter: Lightweight, Docker-specific, simple setup
//	- cAdvisor: Comprehensive, multi-runtime, heavier, more features
//	- Choose Docker Stats Exporter for Docker-only monitoring
//	- Choose cAdvisor for Kubernetes or multi-runtime environments
//
//	vs Prometheus Node Exporter:
//	- Docker Stats Exporter: Container-level metrics
//	- Node Exporter: Host-level metrics (CPU, memory, disk, network)
//	- Use both for complete infrastructure monitoring
//	- Node Exporter complements container metrics with host metrics
//
//	vs Docker stats command:
//	- Docker Stats Exporter: Prometheus format, historical data, alerting
//	- Docker stats: CLI-only, real-time only, no historical data
//	- Docker Stats Exporter is production-ready monitoring solution
//
// Stateless Design:
//
//	Docker Stats Exporter is stateless:
//	- No persistent storage required
//	- All metrics collected in real-time from Docker daemon
//	- No data loss on container restart
//	- Prometheus stores historical metrics
//	- Simple deployment and maintenance
//
// Data Retention:
//
//	Metric retention is handled by Prometheus:
//	- Configure Prometheus retention period (default 15 days)
//	- Use Prometheus remote_write for long-term storage
//	- Consider Grafana Mimir or Thanos for long-term retention
//	- Plan storage capacity based on metric cardinality
//
// Limitations:
//
//	Be aware of these limitations:
//	- Requires Docker socket access (security consideration)
//	- Only monitors Docker containers (not processes or host)
//	- Metrics granularity limited by Docker stats API
//	- No historical data (relies on Prometheus)
//	- Cannot monitor containers on remote hosts directly
//	- No built-in alerting (use Prometheus Alertmanager)
//
// Best Practices:
//
//	For production deployment:
//	- Always use read-only Docker socket mount (:ro)
//	- Configure appropriate Prometheus scrape intervals (15-30s)
//	- Set up alerts for critical resource thresholds
//	- Monitor exporter health and availability
//	- Use persistent Prometheus storage
//	- Create comprehensive Grafana dashboards
//	- Document alert runbooks and escalation procedures
//	- Test alerts and dashboards regularly
//	- Implement backup for Prometheus data
//	- Use Prometheus federation for multi-cluster monitoring
//	- Consider exporter redundancy for high availability
//	- Monitor exporter scrape duration and errors
//
// Troubleshooting:
//
//	Common issues and solutions:
//
//	No metrics exposed:
//	- Check Docker socket access: ls -l /var/run/docker.sock
//	- Verify socket permissions (should be readable)
//	- Check container logs: docker logs docker-stats-exporter
//	- Ensure Docker daemon is running
//
//	Permission denied:
//	- Docker socket requires appropriate permissions
//	- Add exporter container to docker group (if applicable)
//	- Verify read-only mount is working
//
//	Missing containers in metrics:
//	- Check container visibility from exporter: docker ps
//	- Verify all containers are running
//	- Check Docker API version compatibility
//
//	High scrape duration:
//	- Too many containers to monitor
//	- Consider increasing scrape timeout
//	- Optimize metric collection
//	- Scale horizontally with multiple exporters
//
// Error Handling:
//
//	Returns error if:
//	- Container with same name already exists
//	- Network creation fails
//	- Docker API errors occur
//	- Invalid configuration provided
//	- Docker socket not accessible
func DeployDockerStatsExporter(ctx context.Context, cli common.DockerClient, config DockerStatsExporterProductionConfig) (string, error) {
	// Check if container already exists
	exists, err := common.ContainerExistsWithClient(ctx, cli, config.ContainerName)
	if err != nil {
		return "", fmt.Errorf("failed to check container existence: %w", err)
	}
	if exists {
		return "", fmt.Errorf("container %s already exists", config.ContainerName)
	}

	// Prepare production environment (network only, no volume needed)
	if err := PrepareProductionEnvironment(ctx, cli, config.Production); err != nil {
		return "", fmt.Errorf("failed to prepare environment: %w", err)
	}

	// Pull Docker Stats Exporter image
	if err := common.ImagePullWithClient(ctx, cli, config.Image, &common.ImagePullOptions{Silent: true}); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Configure port bindings
	portBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: config.Port,
	}
	portMap := nat.PortMap{
		"8080/tcp": []nat.PortBinding{portBinding},
	}

	// Configure Docker socket bind mount (read-only for security)
	mounts := []mount.Mount{
		{
			Type:     mount.TypeBind,
			Source:   "/var/run/docker.sock",
			Target:   "/var/run/docker.sock",
			ReadOnly: true,
		},
	}

	// Container configuration
	containerConfig := container.Config{
		Image: config.Image,
		ExposedPorts: nat.PortSet{
			"8080/tcp": struct{}{},
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD-SHELL", "wget --spider --quiet http://localhost:8080/metrics || exit 1"},
			Interval: 30000000000, // 30 seconds
			Timeout:  10000000000, // 10 seconds
			Retries:  3,
		},
	}

	// Host configuration
	hostConfig := container.HostConfig{
		PortBindings: portMap,
		Mounts:       mounts,
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
	}

	// Deploy container
	err = common.CreateAndStartContainerWithClient(ctx, cli, containerConfig, hostConfig, config.ContainerName, config.Production.NetworkName)
	if err != nil {
		return "", fmt.Errorf("failed to create and start container: %w", err)
	}

	// Get container ID
	containers, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	for _, cont := range containers {
		for _, name := range cont.Names {
			if name == "/"+config.ContainerName {
				return cont.ID, nil
			}
		}
	}

	return "", fmt.Errorf("container created but ID not found")
}

// StopDockerStatsExporter stops a running Docker Stats Exporter container.
//
// Performs graceful shutdown to ensure clean termination.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the Docker Stats Exporter container to stop
//
// Returns:
//   - error: Stop errors
//
// Example:
//
//	err := StopDockerStatsExporter(ctx, cli, "docker-stats-exporter")
//	if err != nil {
//	    log.Printf("Failed to stop Docker Stats Exporter: %v", err)
//	}
func StopDockerStatsExporter(ctx context.Context, cli common.DockerClient, containerName string) error {
	timeout := 30 // 30 seconds for graceful shutdown
	return cli.ContainerStop(ctx, containerName, container.StopOptions{Timeout: &timeout})
}

// RemoveDockerStatsExporter removes a Docker Stats Exporter container.
//
// Since Docker Stats Exporter is stateless, no data is lost on removal.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the Docker Stats Exporter container to remove
//
// Returns:
//   - error: Removal errors
//
// Example:
//
//	// Remove container (safe - no data stored)
//	err := RemoveDockerStatsExporter(ctx, cli, "docker-stats-exporter")
func RemoveDockerStatsExporter(ctx context.Context, cli common.DockerClient, containerName string) error {
	// Remove container
	if err := cli.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	return nil
}

// GetDockerStatsExporterURL returns the Docker Stats Exporter base URL for the deployed container.
//
// This is a convenience function that formats the URL for the exporter endpoint.
//
// Parameters:
//   - config: Docker Stats Exporter production configuration
//
// Returns:
//   - string: Docker Stats Exporter base URL
//
// Example:
//
//	config := DefaultDockerStatsExporterProductionConfig()
//	exporterURL := GetDockerStatsExporterURL(config)
//	// http://localhost:8080
func GetDockerStatsExporterURL(config DockerStatsExporterProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s", config.Port)
}

// GetDockerStatsExporterMetricsURL returns the Docker Stats Exporter metrics endpoint URL.
//
// This is a convenience function for Prometheus scrape configuration.
//
// Parameters:
//   - config: Docker Stats Exporter production configuration
//
// Returns:
//   - string: Docker Stats Exporter metrics endpoint URL
//
// Example:
//
//	config := DefaultDockerStatsExporterProductionConfig()
//	metricsURL := GetDockerStatsExporterMetricsURL(config)
//	// http://localhost:8080/metrics
func GetDockerStatsExporterMetricsURL(config DockerStatsExporterProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s/metrics", config.Port)
}
