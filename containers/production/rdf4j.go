package production

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"

	"eve.evalgo.org/common"
)

// RDF4JProductionConfig holds configuration for production RDF4J deployment.
type RDF4JProductionConfig struct {
	// ContainerName is the name for the RDF4J container
	ContainerName string
	// Image is the Docker image to use (default: "eclipse/rdf4j-workbench:5.2.0-jetty")
	Image string
	// Port is the host port to expose RDF4J HTTP API (default: 8080)
	Port string
	// JavaOpts are JVM options for memory configuration (default: "-Xms2g -Xmx4g")
	JavaOpts string
	// DataVolume is the volume name for RDF4J data persistence
	DataVolume string
	// LogsVolume is the volume name for Tomcat/Jetty logs
	LogsVolume string
	// Production holds common production configuration
	Production ProductionConfig
}

// DefaultRDF4JProductionConfig returns the default RDF4J production configuration.
func DefaultRDF4JProductionConfig() RDF4JProductionConfig {
	return RDF4JProductionConfig{
		ContainerName: "rdf4j",
		Image:         "eclipse/rdf4j-workbench:5.2.0-jetty",
		Port:          "8080",
		JavaOpts:      "-Xms2g -Xmx4g",
		DataVolume:    "rdf4j-data",
		LogsVolume:    "rdf4j-logs",
		Production: ProductionConfig{
			NetworkName:   "app-network",
			CreateNetwork: true,
			VolumeName:    "rdf4j-data",
			CreateVolume:  true,
		},
	}
}

// DeployRDF4J deploys a production-ready RDF4J container.
//
// RDF4J is an open-source framework for working with RDF data. This function deploys
// an RDF4J Workbench container suitable for production use with persistent data storage.
//
// Container Features:
//   - Named container for consistent identification
//   - Persistent volumes for RDF data and logs
//   - Custom network connectivity
//   - Fixed port mapping for stable access
//   - Restart policy for high availability
//   - Health checks for monitoring
//   - Configurable JVM memory settings
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - config: RDF4J production configuration
//
// Returns:
//   - string: Container ID of the deployed RDF4J container
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
//	config := DefaultRDF4JProductionConfig()
//	config.JavaOpts = "-Xms4g -Xmx8g"  // Increase memory for production
//
//	containerID, err := DeployRDF4J(ctx, cli, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("RDF4J deployed with ID: %s", containerID)
//	log.Printf("Workbench UI: http://localhost:%s/rdf4j-workbench", config.Port)
//
// Connection URLs:
//
//	HTTP REST API and Workbench UI:
//	http://localhost:{port}
//	http://{container_name}:{port} (from other containers)
//
//	Workbench UI:
//	http://localhost:{port}/rdf4j-workbench
//
//	Server API:
//	http://localhost:{port}/rdf4j-server
//
//	SPARQL Endpoint (per repository):
//	http://localhost:{port}/rdf4j-server/repositories/{repository_id}
//
// Data Persistence:
//
//	RDF4J data is stored in Docker volumes ({config.DataVolume}, {config.LogsVolume}).
//	This ensures repositories and RDF data persist across container restarts.
//
//	Volume mount points:
//	- /var/rdf4j - Repository data and configuration
//	- /usr/local/tomcat/logs - Server logs (Jetty/Tomcat)
//
// Memory Configuration:
//
//	RDF4J is a Java application requiring proper JVM tuning:
//	- Development: -Xms1g -Xmx2g
//	- Production: -Xms2g -Xmx4g (default)
//	- Large datasets: -Xms4g -Xmx8g or higher
//
//	Memory requirements depend on:
//	- Dataset size (number of triples)
//	- Query complexity
//	- Number of concurrent users
//	- Repository type (Memory vs Native)
//
// Network Configuration:
//
//	The container joins the specified Docker network, enabling
//	communication with other containers on the same network.
//
//	Other containers can connect using the container name:
//	http://{container_name}:8080/rdf4j-server/repositories/{repository_id}
//
// Security:
//
//	IMPORTANT: RDF4J Workbench has no built-in authentication by default!
//	For production use:
//	- Enable authentication in RDF4J Server configuration
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
//	Health check runs HTTP GET / every 30 seconds:
//	- Timeout: 10 seconds
//	- Retries: 3 consecutive failures before unhealthy
//
// Performance Tuning:
//
//	For production workloads, consider:
//	- Adequate JVM heap size (rule of thumb: 50% of available RAM)
//	- SSD storage for better I/O performance
//	- Repository type selection (Native for persistence, Memory for speed)
//	- Query optimization (use LIMIT, avoid Cartesian products)
//	- Connection pooling
//
// RDF4J Features:
//
//	Open-source RDF framework with:
//	- SPARQL 1.1 Query and Update
//	- RDF storage (Memory Store and Native Store)
//	- REST API for programmatic access
//	- Workbench UI for visual management
//	- Support for various RDF serialization formats
//	- Repository federation
//	- Transaction support
//	- Inference and reasoning (RDFS, custom rules)
//
// Repository Management:
//
//	Create repositories via REST API or Workbench UI:
//	- Memory Store (in-memory, fast, non-persistent)
//	- Native Store (disk-based, persistent)
//	- SPARQL Repository (federation to remote endpoints)
//	- HTTP Repository (proxy to remote RDF4J server)
//
// Monitoring:
//
//	Monitor these metrics via Workbench or API:
//	- Repository size (number of triples)
//	- Query performance
//	- Memory usage (JVM heap)
//	- Active connections
//	- Request latency
//
// Backup Strategy:
//
//	Important: Implement regular backups!
//	- Export repositories (RDF dump files)
//	- Backup repository configuration
//	- Volume snapshots for disaster recovery
//	- Regular backups of /var/rdf4j volume
//
// Error Handling:
//
//	Returns error if:
//	- Container with same name already exists
//	- Network or volume creation fails
//	- Docker API errors occur
//	- Invalid configuration provided
func DeployRDF4J(ctx context.Context, cli common.DockerClient, config RDF4JProductionConfig) (string, error) {
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

	// Pull RDF4J image
	if err := common.ImagePullWithClient(ctx, cli, config.Image, &common.ImagePullOptions{Silent: true}); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Create logs volume if it doesn't exist
	if config.LogsVolume != "" {
		if err := EnsureVolume(ctx, cli, config.LogsVolume); err != nil {
			return "", fmt.Errorf("failed to ensure logs volume: %w", err)
		}
	}

	// Configure port bindings
	portBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: config.Port,
	}
	portMap := nat.PortMap{
		"8080/tcp": []nat.PortBinding{portBinding},
	}

	// Configure volume mounts
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: config.DataVolume,
			Target: "/var/rdf4j",
		},
	}

	// Add logs volume if configured
	if config.LogsVolume != "" {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeVolume,
			Source: config.LogsVolume,
			Target: "/usr/local/tomcat/logs",
		})
	}

	// Container configuration
	containerConfig := container.Config{
		Image: config.Image,
		Env: []string{
			fmt.Sprintf("JAVA_OPTS=%s", config.JavaOpts),
		},
		ExposedPorts: nat.PortSet{
			"8080/tcp": struct{}{},
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD-SHELL", "curl -f http://localhost:8080/ || exit 1"},
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

// StopRDF4J stops a running RDF4J container.
//
// Performs graceful shutdown to ensure data integrity.
// RDF4J will complete active transactions and close repositories before stopping.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the RDF4J container to stop
//
// Returns:
//   - error: Stop errors
//
// Example:
//
//	err := StopRDF4J(ctx, cli, "rdf4j")
//	if err != nil {
//	    log.Printf("Failed to stop RDF4J: %v", err)
//	}
func StopRDF4J(ctx context.Context, cli common.DockerClient, containerName string) error {
	timeout := 60 // 60 seconds for graceful shutdown
	return cli.ContainerStop(ctx, containerName, container.StopOptions{Timeout: &timeout})
}

// RemoveRDF4J removes an RDF4J container and optionally its volumes.
//
// WARNING: Removing volumes will DELETE ALL REPOSITORIES and RDF DATA permanently!
// Always backup data before removing volumes.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the RDF4J container to remove
//   - removeVolumes: Whether to also remove the data volumes (DANGEROUS!)
//   - dataVolumeName: Name of the data volume (required if removeVolumes is true)
//   - logsVolumeName: Name of the logs volume (optional)
//
// Returns:
//   - error: Removal errors
//
// Example:
//
//	// Remove container but keep data (safe)
//	err := RemoveRDF4J(ctx, cli, "rdf4j", false, "", "")
//
//	// Remove container and all data (DANGEROUS - backup first!)
//	err := RemoveRDF4J(ctx, cli, "rdf4j", true, "rdf4j-data", "rdf4j-logs")
func RemoveRDF4J(ctx context.Context, cli common.DockerClient, containerName string, removeVolumes bool, dataVolumeName, logsVolumeName string) error {
	// Remove container
	if err := cli.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	// Remove volumes if requested (DANGEROUS - data loss!)
	if removeVolumes {
		if dataVolumeName != "" {
			if err := cli.VolumeRemove(ctx, dataVolumeName, true); err != nil {
				return fmt.Errorf("failed to remove data volume: %w", err)
			}
		}
		if logsVolumeName != "" {
			if err := cli.VolumeRemove(ctx, logsVolumeName, true); err != nil {
				return fmt.Errorf("failed to remove logs volume: %w", err)
			}
		}
	}

	return nil
}

// GetRDF4JURL returns the RDF4J HTTP endpoint URL for the deployed container.
//
// This is a convenience function that formats the URL for RDF4J REST API and Workbench.
//
// Parameters:
//   - config: RDF4J production configuration
//
// Returns:
//   - string: RDF4J HTTP endpoint URL
//
// Example:
//
//	config := DefaultRDF4JProductionConfig()
//	rdf4jURL := GetRDF4JURL(config)
//	// http://localhost:8080
func GetRDF4JURL(config RDF4JProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s", config.Port)
}

// GetRDF4JWorkbenchURL returns the RDF4J Workbench UI URL for the deployed container.
//
// This is a convenience function that formats the Workbench UI URL.
//
// Parameters:
//   - config: RDF4J production configuration
//
// Returns:
//   - string: RDF4J Workbench UI URL
//
// Example:
//
//	config := DefaultRDF4JProductionConfig()
//	workbenchURL := GetRDF4JWorkbenchURL(config)
//	// http://localhost:8080/rdf4j-workbench
func GetRDF4JWorkbenchURL(config RDF4JProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s/rdf4j-workbench", config.Port)
}

// GetRDF4JServerURL returns the RDF4J Server API URL for the deployed container.
//
// This is a convenience function that formats the Server API URL.
//
// Parameters:
//   - config: RDF4J production configuration
//
// Returns:
//   - string: RDF4J Server API URL
//
// Example:
//
//	config := DefaultRDF4JProductionConfig()
//	serverURL := GetRDF4JServerURL(config)
//	// http://localhost:8080/rdf4j-server
func GetRDF4JServerURL(config RDF4JProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s/rdf4j-server", config.Port)
}

// GetRDF4JRepositoryURL returns the SPARQL endpoint URL for a specific repository.
//
// This is a convenience function that formats the repository-specific SPARQL endpoint.
//
// Parameters:
//   - config: RDF4J production configuration
//   - repositoryID: ID of the repository
//
// Returns:
//   - string: Repository SPARQL endpoint URL
//
// Example:
//
//	config := DefaultRDF4JProductionConfig()
//	sparqlEndpoint := GetRDF4JRepositoryURL(config, "my-repo")
//	// http://localhost:8080/rdf4j-server/repositories/my-repo
func GetRDF4JRepositoryURL(config RDF4JProductionConfig, repositoryID string) string {
	return fmt.Sprintf("http://localhost:%s/rdf4j-server/repositories/%s", config.Port, repositoryID)
}
