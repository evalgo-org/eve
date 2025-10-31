package production

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"

	"eve.evalgo.org/common"
)

// GraphDBProductionConfig holds configuration for production GraphDB deployment.
type GraphDBProductionConfig struct {
	// ContainerName is the name for the GraphDB container
	ContainerName string
	// Image is the Docker image to use (default: "ontotext/graphdb:10.8.1")
	Image string
	// Port is the host port to expose GraphDB HTTP API (default: 7200)
	Port string
	// JavaOpts are JVM options for memory configuration (default: "-Xms2g -Xmx4g")
	JavaOpts string
	// DataVolume is the volume name for GraphDB data persistence
	DataVolume string
	// Production holds common production configuration
	Production ProductionConfig
}

// DefaultGraphDBProductionConfig returns the default GraphDB production configuration.
func DefaultGraphDBProductionConfig() GraphDBProductionConfig {
	return GraphDBProductionConfig{
		ContainerName: "graphdb",
		Image:         "ontotext/graphdb:10.8.1",
		Port:          "7200",
		JavaOpts:      "-Xms2g -Xmx4g",
		DataVolume:    "graphdb-data",
		Production: ProductionConfig{
			NetworkName:   "app-network",
			CreateNetwork: true,
			VolumeName:    "graphdb-data",
			CreateVolume:  true,
		},
	}
}

// DeployGraphDB deploys a production-ready GraphDB container.
//
// GraphDB is a semantic graph database (RDF triple store) from Ontotext. This function
// deploys a GraphDB container suitable for production use with persistent data storage.
//
// Container Features:
//   - Named container for consistent identification
//   - Persistent volume for RDF data and repositories
//   - Custom network connectivity
//   - Fixed port mapping for stable access
//   - Restart policy for high availability
//   - Health checks for monitoring
//   - Configurable JVM memory settings
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - config: GraphDB production configuration
//
// Returns:
//   - string: Container ID of the deployed GraphDB container
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
//	config := DefaultGraphDBProductionConfig()
//	config.JavaOpts = "-Xms4g -Xmx8g"  // Increase memory for production
//
//	containerID, err := DeployGraphDB(ctx, cli, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("GraphDB deployed with ID: %s", containerID)
//	log.Printf("Workbench UI: http://localhost:%s", config.Port)
//
// Connection URLs:
//
//	HTTP REST API and Workbench UI:
//	http://localhost:{port}
//	http://{container_name}:{port} (from other containers)
//
//	SPARQL Endpoint (per repository):
//	http://localhost:{port}/repositories/{repository_id}
//
//	REST API Base:
//	http://localhost:{port}/rest
//
// Data Persistence:
//
//	GraphDB data is stored in a Docker volume ({config.DataVolume}).
//	This ensures repositories and RDF data persist across container restarts.
//
//	Volume mount points:
//	- /opt/graphdb/home - Repository data, configuration, and logs
//
// Memory Configuration:
//
//	GraphDB is a Java application requiring proper JVM tuning:
//	- Development: -Xms1g -Xmx2g
//	- Production: -Xms2g -Xmx4g (default)
//	- Large datasets: -Xms4g -Xmx8g or higher
//
//	Memory requirements depend on:
//	- Dataset size (number of triples)
//	- Query complexity
//	- Reasoning enabled (increases memory usage)
//	- Number of concurrent users
//
// Network Configuration:
//
//	The container joins the specified Docker network, enabling
//	communication with other containers on the same network.
//
//	Other containers can connect using the container name:
//	http://{container_name}:7200/repositories/{repository_id}
//
// Security:
//
//	IMPORTANT: GraphDB Free edition has no built-in authentication!
//	For production use:
//	- Use GraphDB Enterprise for security features
//	- Place behind reverse proxy with authentication
//	- Use firewall rules to restrict access
//	- Consider network-level security
//
// Restart Policy:
//
//	The container is configured with restart policy "unless-stopped",
//	ensuring it automatically restarts if it crashes.
//
// Health Monitoring:
//
//	Health check runs HTTP GET /protocol every 30 seconds:
//	- Timeout: 10 seconds
//	- Retries: 3 consecutive failures before unhealthy
//
// Performance Tuning:
//
//	For production workloads, consider:
//	- Adequate JVM heap size (rule of thumb: 50% of available RAM)
//	- SSD storage for better I/O performance
//	- Repository optimization (ruleset selection)
//	- Query optimization (use LIMIT, avoid Cartesian products)
//	- Enable query caching
//
// GraphDB Features:
//
//	Semantic graph database with:
//	- SPARQL 1.1 Query and Update
//	- RDF storage (triples and quads)
//	- Reasoning (RDFS, OWL-Horst, OWL-Max)
//	- Full-text search (Lucene-based)
//	- GeoSPARQL for geographic queries
//	- REST API for programmatic access
//	- Workbench UI for visual management
//
// Repository Management:
//
//	Create repositories via REST API or Workbench UI:
//	- Free repositories (no reasoning)
//	- RDFS repositories (RDF Schema reasoning)
//	- OWL-Horst (OWL subset reasoning)
//	- OWL-Max (extended OWL reasoning)
//
// Monitoring:
//
//	Monitor these metrics via Workbench or API:
//	- Repository size (number of triples)
//	- Query performance
//	- Memory usage (JVM heap)
//	- Active queries and connections
//	- Cache hit ratio
//
// Backup Strategy:
//
//	Important: Implement regular backups!
//	- Export repositories (RDF dump files)
//	- Backup repository configuration
//	- Volume snapshots for disaster recovery
//	- Consider GraphDB clustering for HA (Enterprise)
//
// Error Handling:
//
//	Returns error if:
//	- Container with same name already exists
//	- Network or volume creation fails
//	- Docker API errors occur
//	- Invalid configuration provided
func DeployGraphDB(ctx context.Context, cli common.DockerClient, config GraphDBProductionConfig) (string, error) {
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

	// Pull GraphDB image
	if err := common.ImagePullWithClient(ctx, cli, config.Image, &common.ImagePullOptions{Silent: true}); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Configure port bindings
	portBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: config.Port,
	}
	portMap := nat.PortMap{
		"7200/tcp": []nat.PortBinding{portBinding},
	}

	// Configure volume mounts
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: config.DataVolume,
			Target: "/opt/graphdb/home",
		},
	}

	// Container configuration
	containerConfig := container.Config{
		Image: config.Image,
		Env: []string{
			fmt.Sprintf("GDB_JAVA_OPTS=%s", config.JavaOpts),
		},
		ExposedPorts: nat.PortSet{
			"7200/tcp": struct{}{},
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD-SHELL", "curl -f http://localhost:7200/protocol || exit 1"},
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

// StopGraphDB stops a running GraphDB container.
//
// Performs graceful shutdown to ensure data integrity.
// GraphDB will complete active transactions and close repositories before stopping.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the GraphDB container to stop
//
// Returns:
//   - error: Stop errors
//
// Example:
//
//	err := StopGraphDB(ctx, cli, "graphdb")
//	if err != nil {
//	    log.Printf("Failed to stop GraphDB: %v", err)
//	}
func StopGraphDB(ctx context.Context, cli common.DockerClient, containerName string) error {
	timeout := 60 // 60 seconds for graceful shutdown
	return cli.ContainerStop(ctx, containerName, container.StopOptions{Timeout: &timeout})
}

// RemoveGraphDB removes a GraphDB container and optionally its volume.
//
// WARNING: Removing the volume will DELETE ALL REPOSITORIES and RDF DATA permanently!
// Always backup data before removing volumes.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the GraphDB container to remove
//   - removeVolume: Whether to also remove the data volume (DANGEROUS!)
//   - volumeName: Name of the data volume (required if removeVolume is true)
//
// Returns:
//   - error: Removal errors
//
// Example:
//
//	// Remove container but keep data (safe)
//	err := RemoveGraphDB(ctx, cli, "graphdb", false, "")
//
//	// Remove container and data (DANGEROUS - backup first!)
//	err := RemoveGraphDB(ctx, cli, "graphdb", true, "graphdb-data")
func RemoveGraphDB(ctx context.Context, cli common.DockerClient, containerName string, removeVolume bool, volumeName string) error {
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

// GetGraphDBURL returns the GraphDB HTTP endpoint URL for the deployed container.
//
// This is a convenience function that formats the URL for GraphDB REST API and Workbench.
//
// Parameters:
//   - config: GraphDB production configuration
//
// Returns:
//   - string: GraphDB HTTP endpoint URL
//
// Example:
//
//	config := DefaultGraphDBProductionConfig()
//	graphdbURL := GetGraphDBURL(config)
//	// http://localhost:7200
func GetGraphDBURL(config GraphDBProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s", config.Port)
}

// GetGraphDBRepositoryURL returns the SPARQL endpoint URL for a specific repository.
//
// This is a convenience function that formats the repository-specific SPARQL endpoint.
//
// Parameters:
//   - config: GraphDB production configuration
//   - repositoryID: ID of the repository
//
// Returns:
//   - string: Repository SPARQL endpoint URL
//
// Example:
//
//	config := DefaultGraphDBProductionConfig()
//	sparqlEndpoint := GetGraphDBRepositoryURL(config, "my-repo")
//	// http://localhost:7200/repositories/my-repo
func GetGraphDBRepositoryURL(config GraphDBProductionConfig, repositoryID string) string {
	return fmt.Sprintf("http://localhost:%s/repositories/%s", config.Port, repositoryID)
}
