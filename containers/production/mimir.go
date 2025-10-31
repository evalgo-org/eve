package production

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"

	"eve.evalgo.org/common"
)

// MimirProductionConfig holds configuration for production Grafana Mimir deployment.
type MimirProductionConfig struct {
	// ContainerName is the name for the Mimir container
	ContainerName string
	// Image is the Docker image to use (default: "grafana/mimir:2.17.2")
	Image string
	// Port is the host port to expose Mimir HTTP API (default: 9009)
	Port string
	// DataVolume is the volume name for Mimir data persistence
	DataVolume string
	// Production holds common production configuration
	Production ProductionConfig
}

// DefaultMimirProductionConfig returns the default Mimir production configuration.
func DefaultMimirProductionConfig() MimirProductionConfig {
	return MimirProductionConfig{
		ContainerName: "mimir",
		Image:         "grafana/mimir:2.17.2",
		Port:          "9009",
		DataVolume:    "mimir-data",
		Production: ProductionConfig{
			NetworkName:   "app-network",
			CreateNetwork: true,
			VolumeName:    "mimir-data",
			CreateVolume:  true,
		},
	}
}

// DeployMimir deploys a production-ready Grafana Mimir container.
//
// Grafana Mimir is an open-source, horizontally scalable, highly available, multi-tenant
// long-term storage for Prometheus metrics. This function deploys a Mimir container
// suitable for production use with persistent data storage.
//
// Container Features:
//   - Named container for consistent identification
//   - Persistent volume for metrics data and blocks
//   - Custom network connectivity
//   - Fixed port mapping for stable access
//   - Restart policy for high availability
//   - Health checks for monitoring
//   - Demo configuration for quick start
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - config: Mimir production configuration
//
// Returns:
//   - string: Container ID of the deployed Mimir container
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
//	config := DefaultMimirProductionConfig()
//	containerID, err := DeployMimir(ctx, cli, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("Mimir deployed with ID: %s", containerID)
//	log.Printf("Mimir HTTP API: http://localhost:%s", config.Port)
//
// Connection URLs:
//
//	HTTP API:
//	http://localhost:{port}
//	http://{container_name}:{port} (from other containers)
//
//	Ready endpoint:
//	http://localhost:{port}/ready
//
//	Metrics endpoint:
//	http://localhost:{port}/metrics
//
//	Push endpoint (remote_write):
//	http://localhost:{port}/api/v1/push
//
//	Query endpoint (PromQL):
//	http://localhost:{port}/prometheus/api/v1/query
//
// Data Persistence:
//
//	Mimir data is stored in a Docker volume ({config.DataVolume}).
//	This ensures metrics, blocks, and metadata persist across container restarts.
//
//	Volume mount points:
//	- /data - Metrics blocks, compacted data, metadata
//
// Network Configuration:
//
//	The container joins the specified Docker network, enabling
//	communication with other containers on the same network.
//
//	Prometheus and other metrics sources can push to Mimir using:
//	http://{container_name}:9009/api/v1/push
//
//	Grafana can query Mimir metrics using:
//	http://{container_name}:9009/prometheus
//
// Multi-Tenancy:
//
//	Mimir is designed as a multi-tenant system. Each request must include
//	the X-Scope-OrgID header to identify the tenant:
//
//	curl -H "X-Scope-OrgID: tenant1" http://localhost:9009/api/v1/push
//
//	For single-tenant deployments, set a consistent org ID:
//	X-Scope-OrgID: anonymous
//
//	Benefits of multi-tenancy:
//	- Isolated metrics per tenant
//	- Per-tenant rate limiting and quotas
//	- Secure data separation
//	- Cost allocation and tracking
//	- Horizontal scaling per tenant
//
// Prometheus Remote Write:
//
//	Configure Prometheus to write metrics to Mimir:
//
//	remote_write:
//	  - url: http://mimir:9009/api/v1/push
//	    headers:
//	      X-Scope-OrgID: tenant1
//
//	Mimir accepts standard Prometheus remote write protocol:
//	- Snappy-compressed protobuf
//	- Batched samples
//	- Label and metadata support
//	- Out-of-order samples (configurable)
//
// Query with PromQL:
//
//	Mimir provides a Prometheus-compatible query API:
//
//	Instant queries:
//	GET /prometheus/api/v1/query?query=up&time=<timestamp>
//
//	Range queries:
//	GET /prometheus/api/v1/query_range?query=rate(requests[5m])&start=<time>&end=<time>&step=<duration>
//
//	Label queries:
//	GET /prometheus/api/v1/labels
//	GET /prometheus/api/v1/label/<label_name>/values
//
//	Series queries:
//	GET /prometheus/api/v1/series?match[]=up
//
// Architecture Modes:
//
//	1. Monolithic Mode (Default):
//	   - All components in a single process
//	   - Simple deployment and operations
//	   - Suitable for small to medium scale
//	   - Limited horizontal scalability
//	   - Good for development and testing
//
//	2. Microservices Mode:
//	   - Components run as separate services
//	   - Independent scaling of components
//	   - High availability and fault tolerance
//	   - Complex deployment (requires orchestration)
//	   - Suitable for large-scale production
//	   - Components: distributor, ingester, querier, compactor, store-gateway, query-frontend
//
//	This deployment uses monolithic mode with the demo configuration.
//
// Storage:
//
//	Mimir uses a block-based storage model:
//	- Time-series data stored in blocks
//	- Blocks are immutable and compressed
//	- Efficient long-term storage
//	- Support for object storage backends (S3, GCS, Azure, filesystem)
//	- Automatic compaction of blocks
//	- Configurable retention policies
//
//	Default configuration uses local filesystem storage in /data.
//
// Performance Characteristics:
//
//	Write Performance:
//	- Millions of active series
//	- Hundreds of thousands of samples/sec per instance
//	- Low latency ingestion (sub-second)
//	- Horizontal scaling with sharding
//
//	Query Performance:
//	- Sub-second query latency for recent data
//	- Efficient querying across long time ranges
//	- Query result caching
//	- Query parallelization
//	- Deduplication of metrics
//
// Security:
//
//	For production use:
//	- Enable authentication and authorization
//	- Use TLS for transport encryption
//	- Implement network policies and firewall rules
//	- Rotate credentials regularly
//	- Enable audit logging
//	- Configure resource limits
//	- Use secure object storage with encryption
//	- Implement tenant isolation
//
// Restart Policy:
//
//	The container is configured with restart policy "unless-stopped",
//	ensuring it automatically restarts if it crashes.
//
// Health Monitoring:
//
//	Health check runs HTTP GET /ready every 30 seconds:
//	- Timeout: 10 seconds
//	- Retries: 3 consecutive failures before unhealthy
//
//	The /ready endpoint returns:
//	- 200 OK when all components are ready
//	- 503 Service Unavailable when not ready
//
// Performance Tuning:
//
//	For production workloads, consider:
//	- Adequate disk space for blocks and metadata
//	- SSD storage for better I/O performance
//	- Memory sizing based on active series count
//	- CPU allocation for query processing
//	- Network bandwidth for ingestion and queries
//	- Object storage configuration for long-term retention
//	- Compaction settings for storage efficiency
//	- Query limits to prevent resource exhaustion
//
// Grafana Mimir Features:
//
//	Long-term metrics storage for Prometheus:
//	- Horizontally scalable architecture
//	- Multi-tenancy with strong isolation
//	- Prometheus-compatible APIs (remote_write, query)
//	- PromQL query language support
//	- High availability and replication
//	- Efficient block-based storage
//	- Object storage backend support
//	- Automatic data compaction
//	- Query result caching
//	- Cardinality management and limits
//	- Ruler for recording and alerting rules
//	- Alert manager integration
//	- Hash-based sharding
//	- Zone-aware replication
//	- Query federation
//
// Data Model:
//
//	Mimir stores metrics in time-series format:
//	- Metric name (e.g., http_requests_total)
//	- Labels (key-value pairs for dimensions)
//	- Timestamp (millisecond precision)
//	- Value (float64)
//
//	Example:
//	http_requests_total{method="GET", status="200", path="/api"} 42 @1234567890
//
// Ingestion Flow:
//
//	1. Prometheus scrapes targets and buffers samples
//	2. Prometheus sends samples via remote_write to Mimir distributor
//	3. Distributor validates, rate limits, and forwards to ingesters
//	4. Ingesters store samples in memory and WAL
//	5. Ingesters periodically flush blocks to long-term storage
//	6. Compactor merges and deduplicates blocks
//	7. Store-gateway loads blocks for querying
//
// Query Flow:
//
//	1. Query sent to query-frontend (optional)
//	2. Query-frontend splits and caches queries
//	3. Querier executes PromQL query
//	4. Querier fetches recent data from ingesters
//	5. Querier fetches historical data from store-gateway
//	6. Results merged, deduplicated, and returned
//
// Monitoring Mimir:
//
//	Monitor these metrics via /metrics endpoint:
//	- cortex_ingester_active_series - Active time series
//	- cortex_ingester_ingested_samples_total - Ingested samples
//	- cortex_ingester_ingested_samples_failures_total - Failed samples
//	- cortex_query_frontend_queries_total - Total queries
//	- cortex_querier_request_duration_seconds - Query latency
//	- cortex_compactor_runs_completed_total - Compaction runs
//	- cortex_distributor_received_samples_total - Received samples
//
// Grafana Integration:
//
//	Add Mimir as a Prometheus data source in Grafana:
//
//	Data Source Configuration:
//	- Type: Prometheus
//	- URL: http://mimir:9009/prometheus
//	- Custom HTTP Headers:
//	  X-Scope-OrgID: tenant1
//
//	Then create dashboards with PromQL queries.
//
// Alerting:
//
//	Mimir includes a ruler component for alerting:
//	- Evaluate recording rules
//	- Evaluate alerting rules
//	- Send alerts to Alertmanager
//	- Per-tenant rule configuration
//	- Prometheus-compatible rule format
//
// Cardinality Management:
//
//	Control metric cardinality to prevent issues:
//	- Set limits on series per tenant
//	- Set limits on label values
//	- Configure sample rate limits
//	- Monitor cardinality metrics
//	- Use relabeling to reduce labels
//	- Implement metric dropping rules
//
// High Availability:
//
//	For production HA setup:
//	- Run multiple instances of each component
//	- Configure replication factor (typically 3)
//	- Use zone-aware replication
//	- Deploy across multiple availability zones
//	- Use load balancer for distributor
//	- Configure consistent hashing for sharding
//	- Enable metadata replication
//
// Backup Strategy:
//
//	Important: Implement regular backups!
//	- Backup block storage (S3, GCS, Azure)
//	- Backup metadata and configuration
//	- Use object storage versioning
//	- Implement disaster recovery procedures
//	- Test restore procedures regularly
//	- Configure retention policies
//	- Monitor storage usage
//
// Migration from Prometheus:
//
//	Steps to migrate from Prometheus to Mimir:
//	1. Deploy Mimir with appropriate configuration
//	2. Configure Prometheus remote_write to Mimir
//	3. Start dual-writing to both Prometheus and Mimir
//	4. Verify data in Mimir matches Prometheus
//	5. Update Grafana to use Mimir data source
//	6. Optionally import historical Prometheus data
//	7. Decommission old Prometheus instances
//
// Configuration:
//
//	This deployment uses the demo configuration file included in the image.
//	For production, create a custom configuration file and mount it:
//
//	Key configuration sections:
//	- limits: Per-tenant resource limits
//	- storage: Block storage backend settings
//	- ingester: Ingestion settings and WAL
//	- compactor: Compaction settings
//	- query_range: Query caching and splitting
//	- ruler: Alerting and recording rules
//	- server: HTTP and gRPC server settings
//
// API Endpoints:
//
//	Key HTTP API endpoints:
//	- GET  /ready - Readiness check
//	- GET  /metrics - Prometheus metrics
//	- POST /api/v1/push - Remote write endpoint
//	- GET  /prometheus/api/v1/query - PromQL instant query
//	- GET  /prometheus/api/v1/query_range - PromQL range query
//	- GET  /prometheus/api/v1/labels - Get label names
//	- GET  /prometheus/api/v1/label/{name}/values - Get label values
//	- GET  /prometheus/api/v1/series - Get series
//	- GET  /prometheus/api/v1/metadata - Get metadata
//	- POST /prometheus/api/v1/read - Remote read endpoint
//	- GET  /config - Get runtime configuration
//	- GET  /distributor/all_user_stats - Get tenant statistics
//	- GET  /api/v1/rules - Get ruler rules
//	- POST /api/v1/rules/{namespace} - Set ruler rules
//
// Error Handling:
//
//	Returns error if:
//	- Container with same name already exists
//	- Network or volume creation fails
//	- Docker API errors occur
//	- Invalid configuration provided
func DeployMimir(ctx context.Context, cli common.DockerClient, config MimirProductionConfig) (string, error) {
	// Check if container already exists
	exists, err := common.ContainerExistsWithClient(ctx, cli, config.ContainerName)
	if err != nil {
		return "", fmt.Errorf("failed to check container existence: %w", err)
	}
	if exists {
		return "", fmt.Errorf("container %s already exists", config.ContainerName)
	}

	// Prepare production environment (network and volume)
	if err := PrepareProductionEnvironment(ctx, cli, config.Production); err != nil {
		return "", fmt.Errorf("failed to prepare environment: %w", err)
	}

	// Pull Mimir image
	if err := common.ImagePullWithClient(ctx, cli, config.Image, &common.ImagePullOptions{Silent: true}); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Configure port bindings
	portBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: config.Port,
	}
	portMap := nat.PortMap{
		"9009/tcp": []nat.PortBinding{portBinding},
	}

	// Configure volume mounts
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: config.DataVolume,
			Target: "/data",
		},
	}

	// Container configuration
	containerConfig := container.Config{
		Image: config.Image,
		Cmd:   []string{"-config.file=/etc/mimir/demo.yaml"},
		ExposedPorts: nat.PortSet{
			"9009/tcp": struct{}{},
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD-SHELL", "wget --spider --quiet http://localhost:9009/ready || exit 1"},
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

// StopMimir stops a running Mimir container.
//
// Performs graceful shutdown to ensure data integrity and proper flushing of in-memory data.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the Mimir container to stop
//
// Returns:
//   - error: Stop errors
//
// Example:
//
//	err := StopMimir(ctx, cli, "mimir")
//	if err != nil {
//	    log.Printf("Failed to stop Mimir: %v", err)
//	}
func StopMimir(ctx context.Context, cli common.DockerClient, containerName string) error {
	timeout := 60 // 60 seconds for graceful shutdown
	return cli.ContainerStop(ctx, containerName, container.StopOptions{Timeout: &timeout})
}

// RemoveMimir removes a Mimir container and optionally its volume.
//
// WARNING: Removing the volume will DELETE ALL METRICS DATA permanently!
// Always backup data before removing volumes.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the Mimir container to remove
//   - removeVolume: Whether to also remove the data volume (DANGEROUS!)
//   - volumeName: Name of the data volume (required if removeVolume is true)
//
// Returns:
//   - error: Removal errors
//
// Example:
//
//	// Remove container but keep data (safe)
//	err := RemoveMimir(ctx, cli, "mimir", false, "")
//
//	// Remove container and data (DANGEROUS - backup first!)
//	err := RemoveMimir(ctx, cli, "mimir", true, "mimir-data")
func RemoveMimir(ctx context.Context, cli common.DockerClient, containerName string, removeVolume bool, volumeName string) error {
	// Remove container
	if err := cli.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	// Remove volume if requested (DANGEROUS - data loss!)
	if removeVolume && volumeName != "" {
		if err := cli.VolumeRemove(ctx, volumeName, true); err != nil {
			return fmt.Errorf("failed to remove volume: %w", err)
		}
	}

	return nil
}

// GetMimirURL returns the Mimir HTTP endpoint URL for the deployed container.
//
// This is a convenience function that formats the URL for Mimir API.
//
// Parameters:
//   - config: Mimir production configuration
//
// Returns:
//   - string: Mimir HTTP endpoint URL
//
// Example:
//
//	config := DefaultMimirProductionConfig()
//	mimirURL := GetMimirURL(config)
//	// http://localhost:9009
func GetMimirURL(config MimirProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s", config.Port)
}

// GetMimirPushURL returns the Mimir remote write (push) endpoint URL.
//
// This is a convenience function for configuring Prometheus remote_write.
//
// Parameters:
//   - config: Mimir production configuration
//
// Returns:
//   - string: Mimir push endpoint URL
//
// Example:
//
//	config := DefaultMimirProductionConfig()
//	pushURL := GetMimirPushURL(config)
//	// http://localhost:9009/api/v1/push
func GetMimirPushURL(config MimirProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s/api/v1/push", config.Port)
}

// GetMimirQueryURL returns the Mimir PromQL query endpoint URL.
//
// This is a convenience function for querying metrics with PromQL.
//
// Parameters:
//   - config: Mimir production configuration
//
// Returns:
//   - string: Mimir query endpoint URL
//
// Example:
//
//	config := DefaultMimirProductionConfig()
//	queryURL := GetMimirQueryURL(config)
//	// http://localhost:9009/prometheus/api/v1/query
func GetMimirQueryURL(config MimirProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s/prometheus/api/v1/query", config.Port)
}
