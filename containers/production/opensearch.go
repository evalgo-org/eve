package production

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"

	"eve.evalgo.org/common"
)

// OpenSearchProductionConfig holds configuration for production OpenSearch deployment.
type OpenSearchProductionConfig struct {
	// ContainerName is the name for the OpenSearch container
	ContainerName string
	// Image is the Docker image to use (default: "opensearchproject/opensearch:3.0.0")
	Image string
	// Port is the host port to expose OpenSearch HTTP API (default: 9200)
	Port string
	// JavaOpts are JVM options for memory configuration (default: "-Xms2g -Xmx2g")
	JavaOpts string
	// DisableSecurity disables OpenSearch security plugin (default: true for testing)
	DisableSecurity bool
	// DataVolume is the volume name for OpenSearch data persistence
	DataVolume string
	// Production holds common production configuration
	Production ProductionConfig
}

// DefaultOpenSearchProductionConfig returns the default OpenSearch production configuration.
func DefaultOpenSearchProductionConfig() OpenSearchProductionConfig {
	return OpenSearchProductionConfig{
		ContainerName:   "opensearch",
		Image:           "opensearchproject/opensearch:3.0.0",
		Port:            "9200",
		JavaOpts:        "-Xms2g -Xmx2g",
		DisableSecurity: true,
		DataVolume:      "opensearch-data",
		Production: ProductionConfig{
			NetworkName:   "app-network",
			CreateNetwork: true,
			VolumeName:    "opensearch-data",
			CreateVolume:  true,
		},
	}
}

// DeployOpenSearch deploys a production-ready OpenSearch container.
//
// OpenSearch is a community-driven, open-source search and analytics suite. This function
// deploys an OpenSearch container suitable for production use with persistent data storage.
//
// Container Features:
//   - Named container for consistent identification
//   - Persistent volume for search indices and data
//   - Custom network connectivity
//   - Fixed port mapping for stable access
//   - Restart policy for high availability
//   - Health checks for monitoring
//   - Configurable JVM memory settings
//   - Optional security plugin (disabled by default for testing)
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - config: OpenSearch production configuration
//
// Returns:
//   - string: Container ID of the deployed OpenSearch container
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
//	config := DefaultOpenSearchProductionConfig()
//	config.JavaOpts = "-Xms4g -Xmx4g"  // Increase memory for production
//	config.DisableSecurity = false      // Enable security for production
//
//	containerID, err := DeployOpenSearch(ctx, cli, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("OpenSearch deployed with ID: %s", containerID)
//	log.Printf("HTTP API: http://localhost:%s", config.Port)
//
// Connection URLs:
//
//	HTTP REST API:
//	http://localhost:{port}
//	http://{container_name}:{port} (from other containers)
//
//	Cluster Information:
//	http://localhost:{port}/
//
//	Cluster Health:
//	http://localhost:{port}/_cluster/health
//
//	Index Operations:
//	http://localhost:{port}/{index_name}
//
// Data Persistence:
//
//	OpenSearch data is stored in a Docker volume ({config.DataVolume}).
//	This ensures indices and documents persist across container restarts.
//
//	Volume mount points:
//	- /usr/share/opensearch/data - Search indices, documents, and cluster state
//
// Memory Configuration:
//
//	OpenSearch is a Java application requiring proper JVM tuning:
//	- Development: -Xms512m -Xmx512m
//	- Production: -Xms2g -Xmx2g (default)
//	- Large datasets: -Xms4g -Xmx4g or higher
//
//	Memory requirements depend on:
//	- Dataset size (number of documents)
//	- Index size and structure
//	- Query complexity and aggregations
//	- Number of concurrent users
//	- Number of shards and replicas
//
//	Important: Set Xms == Xmx to avoid heap resizing overhead
//
// Network Configuration:
//
//	The container joins the specified Docker network, enabling
//	communication with other containers on the same network.
//
//	Other containers can connect using the container name:
//	http://{container_name}:9200
//
// Security:
//
//	IMPORTANT: Security is disabled by default for testing!
//	For production use:
//	- Set config.DisableSecurity = false
//	- Configure authentication (basic, SAML, OIDC, etc.)
//	- Enable TLS/SSL encryption
//	- Use role-based access control (RBAC)
//	- Enable audit logging
//	- Configure fine-grained access control
//
// Restart Policy:
//
//	The container is configured with restart policy "unless-stopped",
//	ensuring it automatically restarts if it crashes.
//
// Health Monitoring:
//
//	Health check runs HTTP GET / every 30 seconds:
//	- Timeout: 10 seconds
//	- Retries: 3 consecutive failures before unhealthy
//
// Performance Tuning:
//
//	For production workloads, consider:
//	- Adequate JVM heap size (50% of available RAM, up to 32GB)
//	- SSD storage for better I/O performance
//	- Proper shard sizing (20-50GB per shard)
//	- Replica configuration for high availability
//	- Index lifecycle management (ILM)
//	- Query caching optimization
//	- Segment merging tuning
//
// OpenSearch Features:
//
//	Search and analytics engine with:
//	- Full-text search with Lucene
//	- Real-time indexing
//	- Distributed architecture
//	- RESTful API
//	- JSON document storage
//	- Aggregations and analytics
//	- Machine learning capabilities
//	- Alerting and notifications
//	- SQL query support
//	- Index State Management (ISM)
//	- Performance Analyzer
//
// Monitoring:
//
//	Monitor these metrics via REST API or Performance Analyzer:
//	- Cluster health (green, yellow, red)
//	- Node status and resources
//	- Index size and document count
//	- Query performance and latency
//	- JVM heap usage
//	- Cache hit ratio
//	- Search and indexing rate
//
// Backup Strategy:
//
//	Important: Implement regular backups!
//	- Snapshot and restore API
//	- Repository configuration (S3, filesystem, etc.)
//	- Automated snapshot policies
//	- Cross-cluster replication for HA
//	- Index State Management for retention
//
// Error Handling:
//
//	Returns error if:
//	- Container with same name already exists
//	- Network or volume creation fails
//	- Docker API errors occur
//	- Invalid configuration provided
func DeployOpenSearch(ctx context.Context, cli common.DockerClient, config OpenSearchProductionConfig) (string, error) {
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

	// Pull OpenSearch image
	if err := common.ImagePullWithClient(ctx, cli, config.Image, &common.ImagePullOptions{Silent: true}); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Configure port bindings
	portBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: config.Port,
	}
	portMap := nat.PortMap{
		"9200/tcp": []nat.PortBinding{portBinding},
	}

	// Configure volume mounts
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: config.DataVolume,
			Target: "/usr/share/opensearch/data",
		},
	}

	// Build environment variables
	env := []string{
		fmt.Sprintf("OPENSEARCH_JAVA_OPTS=%s", config.JavaOpts),
		"discovery.type=single-node",
	}

	// Add security settings
	if config.DisableSecurity {
		env = append(env, "DISABLE_SECURITY_PLUGIN=true")
		env = append(env, "DISABLE_INSTALL_DEMO_CONFIG=true")
	}

	// Container configuration
	containerConfig := container.Config{
		Image: config.Image,
		Env:   env,
		ExposedPorts: nat.PortSet{
			"9200/tcp": struct{}{},
			"9600/tcp": struct{}{}, // Performance Analyzer
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD-SHELL", "curl -f http://localhost:9200/ || exit 1"},
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

// StopOpenSearch stops a running OpenSearch container.
//
// Performs graceful shutdown to ensure data integrity.
// OpenSearch will complete active indexing operations and close indices before stopping.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the OpenSearch container to stop
//
// Returns:
//   - error: Stop errors
//
// Example:
//
//	err := StopOpenSearch(ctx, cli, "opensearch")
//	if err != nil {
//	    log.Printf("Failed to stop OpenSearch: %v", err)
//	}
func StopOpenSearch(ctx context.Context, cli common.DockerClient, containerName string) error {
	timeout := 60 // 60 seconds for graceful shutdown
	return cli.ContainerStop(ctx, containerName, container.StopOptions{Timeout: &timeout})
}

// RemoveOpenSearch removes an OpenSearch container and optionally its volume.
//
// WARNING: Removing the volume will DELETE ALL INDICES and DOCUMENTS permanently!
// Always backup data before removing volumes.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the OpenSearch container to remove
//   - removeVolume: Whether to also remove the data volume (DANGEROUS!)
//   - volumeName: Name of the data volume (required if removeVolume is true)
//
// Returns:
//   - error: Removal errors
//
// Example:
//
//	// Remove container but keep data (safe)
//	err := RemoveOpenSearch(ctx, cli, "opensearch", false, "")
//
//	// Remove container and data (DANGEROUS - backup first!)
//	err := RemoveOpenSearch(ctx, cli, "opensearch", true, "opensearch-data")
func RemoveOpenSearch(ctx context.Context, cli common.DockerClient, containerName string, removeVolume bool, volumeName string) error {
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

// GetOpenSearchURL returns the OpenSearch HTTP endpoint URL for the deployed container.
//
// This is a convenience function that formats the URL for OpenSearch REST API.
//
// Parameters:
//   - config: OpenSearch production configuration
//
// Returns:
//   - string: OpenSearch HTTP endpoint URL
//
// Example:
//
//	config := DefaultOpenSearchProductionConfig()
//	opensearchURL := GetOpenSearchURL(config)
//	// http://localhost:9200
func GetOpenSearchURL(config OpenSearchProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s", config.Port)
}

// GetOpenSearchIndexURL returns the URL for a specific index.
//
// This is a convenience function that formats the index-specific endpoint URL.
//
// Parameters:
//   - config: OpenSearch production configuration
//   - indexName: Name of the index
//
// Returns:
//   - string: Index endpoint URL
//
// Example:
//
//	config := DefaultOpenSearchProductionConfig()
//	indexURL := GetOpenSearchIndexURL(config, "my-index")
//	// http://localhost:9200/my-index
func GetOpenSearchIndexURL(config OpenSearchProductionConfig, indexName string) string {
	return fmt.Sprintf("http://localhost:%s/%s", config.Port, indexName)
}
