package production

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"

	"eve.evalgo.org/common"
)

// OpenSearchDashboardsProductionConfig holds configuration for production OpenSearch Dashboards deployment.
type OpenSearchDashboardsProductionConfig struct {
	// ContainerName is the name for the OpenSearch Dashboards container
	ContainerName string
	// Image is the Docker image to use (default: "opensearchproject/opensearch-dashboards:3.0.0")
	Image string
	// Port is the host port to expose OpenSearch Dashboards UI (default: 5601)
	Port string
	// OpenSearchURL is the URL to the OpenSearch instance (required)
	OpenSearchURL string
	// DisableSecurity disables OpenSearch Dashboards security plugin (default: true for testing)
	DisableSecurity bool
	// DataVolume is the volume name for OpenSearch Dashboards data persistence
	DataVolume string
	// Production holds common production configuration
	Production ProductionConfig
}

// DefaultOpenSearchDashboardsProductionConfig returns the default OpenSearch Dashboards production configuration.
func DefaultOpenSearchDashboardsProductionConfig(opensearchURL string) OpenSearchDashboardsProductionConfig {
	return OpenSearchDashboardsProductionConfig{
		ContainerName:   "opensearch-dashboards",
		Image:           "opensearchproject/opensearch-dashboards:3.0.0",
		Port:            "5601",
		OpenSearchURL:   opensearchURL,
		DisableSecurity: true,
		DataVolume:      "opensearch-dashboards-data",
		Production: ProductionConfig{
			NetworkName:   "app-network",
			CreateNetwork: true,
			VolumeName:    "opensearch-dashboards-data",
			CreateVolume:  true,
		},
	}
}

// DeployOpenSearchDashboards deploys a production-ready OpenSearch Dashboards container.
//
// OpenSearch Dashboards is the visualization and user interface for OpenSearch. This function
// deploys an OpenSearch Dashboards container suitable for production use with persistent data storage.
//
// Container Features:
//   - Named container for consistent identification
//   - Persistent volume for saved objects and configuration
//   - Custom network connectivity
//   - Fixed port mapping for stable access
//   - Restart policy for high availability
//   - Health checks for monitoring
//   - Connection to OpenSearch cluster
//   - Optional security plugin (disabled by default for testing)
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - config: OpenSearch Dashboards production configuration
//
// Returns:
//   - string: Container ID of the deployed OpenSearch Dashboards container
//   - error: Deployment errors
//
// Example Usage:
//
//	ctx, cli := common.CtxCli("unix:///var/run/docker.sock")
//	defer cli.Close()
//
//	// First, deploy OpenSearch
//	osConfig := DefaultOpenSearchProductionConfig()
//	_, err := DeployOpenSearch(ctx, cli, osConfig)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Then, deploy OpenSearch Dashboards
//	config := DefaultOpenSearchDashboardsProductionConfig("http://opensearch:9200")
//	config.DisableSecurity = false  // Enable security for production
//
//	containerID, err := DeployOpenSearchDashboards(ctx, cli, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("OpenSearch Dashboards deployed with ID: %s", containerID)
//	log.Printf("UI: http://localhost:%s", config.Port)
//
// Connection URL:
//
//	Access the OpenSearch Dashboards UI:
//	http://localhost:{port}
//
//	From other containers on the same network:
//	http://{container_name}:{port}
//
// UI Features:
//
//	OpenSearch Dashboards provides:
//	- Discover: Explore and search data
//	- Visualize: Create charts, graphs, and visualizations
//	- Dashboards: Combine visualizations into dashboards
//	- Dev Tools: Console for running queries
//	- Management: Index patterns, saved objects, settings
//	- Alerting: Create and manage alerts
//	- Reports: Generate and schedule reports
//	- Notebooks: Interactive analysis notebooks
//	- Observability: Logs, traces, and metrics
//	- Security: User management and permissions
//
// Data Persistence:
//
//	OpenSearch Dashboards data is stored in a Docker volume ({config.DataVolume}).
//	This ensures saved objects and configuration persist across container restarts.
//
//	Volume mount points:
//	- /usr/share/opensearch-dashboards/data - Saved objects, index patterns, visualizations
//
// Network Configuration:
//
//	The container joins the specified Docker network, enabling
//	communication with OpenSearch and other containers.
//
//	IMPORTANT: The OpenSearchURL should use the container name if both
//	are on the same Docker network:
//	- http://opensearch:9200 (same network, recommended)
//	- http://host.docker.internal:9200 (from container to host)
//	- http://<host-ip>:9200 (external OpenSearch)
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
//	- Configure session timeouts
//	- Enable multi-tenancy if needed
//
// Restart Policy:
//
//	The container is configured with restart policy "unless-stopped",
//	ensuring it automatically restarts if it crashes.
//
// Health Monitoring:
//
//	Health check runs HTTP GET /api/status every 30 seconds:
//	- Timeout: 10 seconds
//	- Retries: 3 consecutive failures before unhealthy
//
// Performance Tuning:
//
//	For production workloads, consider:
//	- Adequate memory for Node.js (default is usually sufficient)
//	- Fast storage for saved objects
//	- Connection pooling to OpenSearch
//	- Browser caching configuration
//	- Load balancing for multiple instances
//	- CDN for static assets (if applicable)
//
// API Endpoints:
//
//	REST API for automation:
//	- GET    /api/status - Application status
//	- GET    /api/saved_objects - List saved objects
//	- POST   /api/saved_objects/{type} - Create saved object
//	- PUT    /api/saved_objects/{type}/{id} - Update saved object
//	- DELETE /api/saved_objects/{type}/{id} - Delete saved object
//	- POST   /api/console/proxy - Dev Tools console proxy
//
// Monitoring:
//
//	Monitor these metrics:
//	- Application health (/api/status)
//	- Response time
//	- Active user sessions
//	- Saved object count
//	- Connection to OpenSearch
//	- Browser performance (client-side)
//
// Backup Strategy:
//
//	Important: Implement regular backups!
//	- Export saved objects via API
//	- Backup configuration files
//	- Volume snapshots for disaster recovery
//	- Version control for custom plugins
//
// Customization:
//
//	Customize Dashboards via:
//	- Custom branding (logo, colors)
//	- Custom plugins
//	- Configuration files (opensearch_dashboards.yml)
//	- Environment variables
//	- Custom visualizations
//
// Error Handling:
//
//	Returns error if:
//	- Container with same name already exists
//	- Network or volume creation fails
//	- Docker API errors occur
//	- Invalid configuration provided
//	- OpenSearchURL is empty
func DeployOpenSearchDashboards(ctx context.Context, cli common.DockerClient, config OpenSearchDashboardsProductionConfig) (string, error) {
	// Validate OpenSearchURL
	if config.OpenSearchURL == "" {
		return "", fmt.Errorf("OpenSearchURL is required")
	}

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

	// Pull OpenSearch Dashboards image
	if err := common.ImagePullWithClient(ctx, cli, config.Image, &common.ImagePullOptions{Silent: true}); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Configure port bindings
	portBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: config.Port,
	}
	portMap := nat.PortMap{
		"5601/tcp": []nat.PortBinding{portBinding},
	}

	// Configure volume mounts
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: config.DataVolume,
			Target: "/usr/share/opensearch-dashboards/data",
		},
	}

	// Build environment variables
	env := []string{
		fmt.Sprintf("OPENSEARCH_HOSTS=%s", config.OpenSearchURL),
	}

	// Add security settings
	if config.DisableSecurity {
		env = append(env, "DISABLE_SECURITY_DASHBOARDS_PLUGIN=true")
	}

	// Container configuration
	containerConfig := container.Config{
		Image: config.Image,
		Env:   env,
		ExposedPorts: nat.PortSet{
			"5601/tcp": struct{}{},
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD-SHELL", "curl -f http://localhost:5601/api/status || exit 1"},
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

// StopOpenSearchDashboards stops a running OpenSearch Dashboards container.
//
// Performs graceful shutdown to ensure saved objects are properly persisted.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the OpenSearch Dashboards container to stop
//
// Returns:
//   - error: Stop errors
//
// Example:
//
//	err := StopOpenSearchDashboards(ctx, cli, "opensearch-dashboards")
//	if err != nil {
//	    log.Printf("Failed to stop OpenSearch Dashboards: %v", err)
//	}
func StopOpenSearchDashboards(ctx context.Context, cli common.DockerClient, containerName string) error {
	timeout := 30 // 30 seconds for graceful shutdown
	return cli.ContainerStop(ctx, containerName, container.StopOptions{Timeout: &timeout})
}

// RemoveOpenSearchDashboards removes an OpenSearch Dashboards container and optionally its volume.
//
// WARNING: Removing the volume will DELETE ALL SAVED OBJECTS permanently!
// Always backup saved objects before removing volumes.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the OpenSearch Dashboards container to remove
//   - removeVolume: Whether to also remove the data volume (DANGEROUS!)
//   - volumeName: Name of the data volume (required if removeVolume is true)
//
// Returns:
//   - error: Removal errors
//
// Example:
//
//	// Remove container but keep data (safe)
//	err := RemoveOpenSearchDashboards(ctx, cli, "opensearch-dashboards", false, "")
//
//	// Remove container and data (DANGEROUS - backup first!)
//	err := RemoveOpenSearchDashboards(ctx, cli, "opensearch-dashboards", true, "opensearch-dashboards-data")
func RemoveOpenSearchDashboards(ctx context.Context, cli common.DockerClient, containerName string, removeVolume bool, volumeName string) error {
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

// GetOpenSearchDashboardsURL returns the OpenSearch Dashboards HTTP endpoint URL for the deployed container.
//
// This is a convenience function that formats the URL for OpenSearch Dashboards UI.
//
// Parameters:
//   - config: OpenSearch Dashboards production configuration
//
// Returns:
//   - string: OpenSearch Dashboards HTTP endpoint URL
//
// Example:
//
//	config := DefaultOpenSearchDashboardsProductionConfig("http://opensearch:9200")
//	dashboardsURL := GetOpenSearchDashboardsURL(config)
//	// http://localhost:5601
func GetOpenSearchDashboardsURL(config OpenSearchDashboardsProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s", config.Port)
}

// GetOpenSearchDashboardsAppURL returns the URL for a specific Dashboards application.
//
// This is a convenience function that formats the app-specific URL.
//
// Parameters:
//   - config: OpenSearch Dashboards production configuration
//   - appName: Name of the application (e.g., "discover", "dashboards", "visualize", "dev_tools")
//
// Returns:
//   - string: Application URL
//
// Example:
//
//	config := DefaultOpenSearchDashboardsProductionConfig("http://opensearch:9200")
//	discoverURL := GetOpenSearchDashboardsAppURL(config, "discover")
//	// http://localhost:5601/app/discover
func GetOpenSearchDashboardsAppURL(config OpenSearchDashboardsProductionConfig, appName string) string {
	return fmt.Sprintf("http://localhost:%s/app/%s", config.Port, appName)
}
