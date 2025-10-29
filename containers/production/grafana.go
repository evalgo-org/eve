package production

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"

	"eve.evalgo.org/common"
)

// GrafanaProductionConfig holds configuration for production Grafana deployment.
type GrafanaProductionConfig struct {
	// ContainerName is the name for the Grafana container
	ContainerName string
	// Image is the Docker image to use (default: "grafana/grafana:12.3.0-18893060694")
	Image string
	// Port is the host port to expose Grafana HTTP UI and API (default: 3000)
	Port string
	// AdminUser is the admin username (default: "admin")
	AdminUser string
	// AdminPassword is the admin password (default: "admin")
	AdminPassword string
	// DataVolume is the volume name for Grafana data persistence
	DataVolume string
	// Production holds common production configuration
	Production ProductionConfig
}

// DefaultGrafanaProductionConfig returns the default Grafana production configuration.
func DefaultGrafanaProductionConfig() GrafanaProductionConfig {
	return GrafanaProductionConfig{
		ContainerName: "grafana",
		Image:         "grafana/grafana:12.3.0-18893060694",
		Port:          "3000",
		AdminUser:     "admin",
		AdminPassword: "admin",
		DataVolume:    "grafana-data",
		Production: ProductionConfig{
			NetworkName:   "app-network",
			CreateNetwork: true,
			VolumeName:    "grafana-data",
			CreateVolume:  true,
		},
	}
}

// DeployGrafana deploys a production-ready Grafana container.
//
// Grafana is an open-source platform for monitoring and observability with beautiful dashboards.
// This function deploys a Grafana container suitable for production use with persistent data storage.
//
// Container Features:
//   - Named container for consistent identification
//   - Persistent volume for dashboards and data sources
//   - Custom network connectivity
//   - Fixed port mapping for stable access
//   - Restart policy for high availability
//   - Health checks for monitoring
//   - Configurable admin credentials
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - config: Grafana production configuration
//
// Returns:
//   - string: Container ID of the deployed Grafana container
//   - error: Deployment errors
//
// Example Usage:
//
//	ctx, cli := common.CtxCli("unix:///var/run/docker.sock")
//	defer cli.Close()
//
//	config := DefaultGrafanaProductionConfig()
//	config.AdminPassword = "secure_password_here"  // Change default password!
//
//	containerID, err := DeployGrafana(ctx, cli, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("Grafana deployed with ID: %s", containerID)
//	log.Printf("Grafana UI: http://localhost:%s", config.Port)
//
// Connection URLs:
//
//	HTTP UI and API:
//	http://localhost:{port}
//	http://{container_name}:{port} (from other containers)
//
//	Health endpoint:
//	http://localhost:{port}/api/health
//
//	Login:
//	http://localhost:{port}/login
//	Username: {config.AdminUser}
//	Password: {config.AdminPassword}
//
// Data Persistence:
//
//	Grafana data is stored in a Docker volume ({config.DataVolume}).
//	This ensures dashboards, data sources, and settings persist across container restarts.
//
//	Volume mount points:
//	- /var/lib/grafana - Dashboards, data sources, plugins, databases
//
// Network Configuration:
//
//	The container joins the specified Docker network, enabling
//	communication with other containers on the same network.
//
//	Other containers (like Prometheus, Loki) can be accessed by name:
//	http://{container_name}:{port}
//
// Security:
//
//	IMPORTANT: Change the default admin password for production!
//	Default credentials are admin/admin.
//
//	For production use:
//	- Set strong admin password via config.AdminPassword
//	- Configure authentication (LDAP, OAuth, SAML)
//	- Enable HTTPS with TLS certificates
//	- Use reverse proxy (nginx, traefik)
//	- Disable anonymous access
//	- Configure role-based access control (RBAC)
//	- Enable audit logging
//
// Restart Policy:
//
//	The container is configured with restart policy "unless-stopped",
//	ensuring it automatically restarts if it crashes.
//
// Health Monitoring:
//
//	Health check runs HTTP GET /api/health every 30 seconds:
//	- Timeout: 10 seconds
//	- Retries: 3 consecutive failures before unhealthy
//
// Performance Tuning:
//
//	For production workloads, consider:
//	- Adequate disk space for dashboards and databases
//	- SSD storage for better query performance
//	- Configure database backend (SQLite vs PostgreSQL/MySQL)
//	- Enable caching for better performance
//	- Optimize dashboard queries
//	- Set appropriate refresh intervals
//
// Grafana Features:
//
//	Open-source monitoring and observability platform:
//	- Beautiful, customizable dashboards
//	- Multiple data source support (Prometheus, Loki, Tempo, etc.)
//	- Alerting and notifications (email, Slack, PagerDuty)
//	- User management and authentication
//	- Plugin ecosystem (panels, data sources, apps)
//	- Query builder and template variables
//	- Annotations and events
//	- Dashboard sharing and embedding
//	- Reporting (PDF, image)
//	- API for automation
//
// Data Sources:
//
//	Grafana supports many data sources out of the box:
//	- Prometheus - Metrics and monitoring
//	- Loki - Log aggregation
//	- Tempo - Distributed tracing
//	- Jaeger - Distributed tracing
//	- PostgreSQL - SQL database
//	- MySQL - SQL database
//	- InfluxDB - Time series database
//	- Elasticsearch - Full-text search
//	- Graphite - Time series metrics
//	- CloudWatch - AWS monitoring
//	- Azure Monitor - Azure monitoring
//	- Google Cloud Monitoring - GCP monitoring
//
// Plugins:
//
//	Install plugins via environment variable or provisioning:
//	- Panel plugins: Custom visualizations
//	- Data source plugins: New data sources
//	- App plugins: Complete applications
//
// Dashboards:
//
//	Create dashboards via:
//	- Grafana UI (visual editor)
//	- HTTP API (programmatic)
//	- Provisioning (JSON files)
//	- Import from grafana.com
//
// Alerting:
//
//	Configure alerts via:
//	- Alert rules in dashboards
//	- Contact points (email, Slack, etc.)
//	- Notification policies
//	- Silences and mute timings
//
// API Endpoints:
//
//	Key HTTP API endpoints:
//	- GET  /api/health - Health check
//	- GET  /api/datasources - List data sources
//	- POST /api/datasources - Create data source
//	- GET  /api/dashboards/db/:slug - Get dashboard
//	- POST /api/dashboards/db - Create/update dashboard
//	- GET  /api/search - Search dashboards
//	- POST /api/annotations - Create annotation
//	- GET  /api/org - Get current organization
//	- GET  /api/admin/stats - Get server statistics
//	- POST /api/alerts - Create alert
//	- GET  /api/alerting/list - List alerts
//
// Monitoring:
//
//	Monitor these metrics via API or UI:
//	- Dashboard load times
//	- Query performance
//	- Data source health
//	- User activity
//	- Alert firing status
//	- Database size
//
// Backup Strategy:
//
//	Important: Implement regular backups!
//	- Export dashboards (JSON files)
//	- Backup Grafana database (SQLite or external DB)
//	- Backup /var/lib/grafana volume
//	- Volume snapshots for disaster recovery
//	- Store provisioning configurations in version control
//	- Test restore procedures regularly
//
// Database Backend:
//
//	Grafana uses SQLite by default, but supports:
//	- SQLite (default, embedded)
//	- PostgreSQL (recommended for production)
//	- MySQL/MariaDB (production alternative)
//
//	For production, configure PostgreSQL backend via environment:
//	GF_DATABASE_TYPE=postgres
//	GF_DATABASE_HOST=postgres:5432
//	GF_DATABASE_NAME=grafana
//	GF_DATABASE_USER=grafana
//	GF_DATABASE_PASSWORD=password
//
// Error Handling:
//
//	Returns error if:
//	- Container with same name already exists
//	- Network or volume creation fails
//	- Docker API errors occur
//	- Invalid configuration provided
func DeployGrafana(ctx context.Context, cli common.DockerClient, config GrafanaProductionConfig) (string, error) {
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

	// Pull Grafana image
	if err := common.ImagePullWithClient(ctx, cli, config.Image, &common.ImagePullOptions{Silent: true}); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Configure port bindings
	portBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: config.Port,
	}
	portMap := nat.PortMap{
		"3000/tcp": []nat.PortBinding{portBinding},
	}

	// Configure volume mounts
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: config.DataVolume,
			Target: "/var/lib/grafana",
		},
	}

	// Container configuration
	containerConfig := container.Config{
		Image: config.Image,
		Env: []string{
			fmt.Sprintf("GF_SECURITY_ADMIN_USER=%s", config.AdminUser),
			fmt.Sprintf("GF_SECURITY_ADMIN_PASSWORD=%s", config.AdminPassword),
			"GF_USERS_ALLOW_SIGN_UP=false",
		},
		ExposedPorts: nat.PortSet{
			"3000/tcp": struct{}{},
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD-SHELL", "wget --spider --quiet http://localhost:3000/api/health || exit 1"},
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

// StopGrafana stops a running Grafana container.
//
// Performs graceful shutdown to ensure data integrity.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the Grafana container to stop
//
// Returns:
//   - error: Stop errors
//
// Example:
//
//	err := StopGrafana(ctx, cli, "grafana")
//	if err != nil {
//	    log.Printf("Failed to stop Grafana: %v", err)
//	}
func StopGrafana(ctx context.Context, cli common.DockerClient, containerName string) error {
	timeout := 30 // 30 seconds for graceful shutdown
	return cli.ContainerStop(ctx, containerName, container.StopOptions{Timeout: &timeout})
}

// RemoveGrafana removes a Grafana container and optionally its volume.
//
// WARNING: Removing the volume will DELETE ALL DASHBOARDS and DATA SOURCES permanently!
// Always backup data before removing volumes.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the Grafana container to remove
//   - removeVolume: Whether to also remove the data volume (DANGEROUS!)
//   - volumeName: Name of the data volume (required if removeVolume is true)
//
// Returns:
//   - error: Removal errors
//
// Example:
//
//	// Remove container but keep data (safe)
//	err := RemoveGrafana(ctx, cli, "grafana", false, "")
//
//	// Remove container and data (DANGEROUS - backup first!)
//	err := RemoveGrafana(ctx, cli, "grafana", true, "grafana-data")
func RemoveGrafana(ctx context.Context, cli common.DockerClient, containerName string, removeVolume bool, volumeName string) error {
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

// GetGrafanaURL returns the Grafana HTTP endpoint URL for the deployed container.
//
// This is a convenience function that formats the URL for Grafana UI and API.
//
// Parameters:
//   - config: Grafana production configuration
//
// Returns:
//   - string: Grafana HTTP endpoint URL
//
// Example:
//
//	config := DefaultGrafanaProductionConfig()
//	grafanaURL := GetGrafanaURL(config)
//	// http://localhost:3000
func GetGrafanaURL(config GrafanaProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s", config.Port)
}

// GetGrafanaAPIURL returns the Grafana API base URL for the deployed container.
//
// This is a convenience function that formats the API base URL.
//
// Parameters:
//   - config: Grafana production configuration
//
// Returns:
//   - string: Grafana API base URL
//
// Example:
//
//	config := DefaultGrafanaProductionConfig()
//	apiURL := GetGrafanaAPIURL(config)
//	// http://localhost:3000/api
func GetGrafanaAPIURL(config GrafanaProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s/api", config.Port)
}

// GetGrafanaHealthURL returns the Grafana health check URL for the deployed container.
//
// This is a convenience function for monitoring and health checks.
//
// Parameters:
//   - config: Grafana production configuration
//
// Returns:
//   - string: Grafana health check URL
//
// Example:
//
//	config := DefaultGrafanaProductionConfig()
//	healthURL := GetGrafanaHealthURL(config)
//	// http://localhost:3000/api/health
func GetGrafanaHealthURL(config GrafanaProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s/api/health", config.Port)
}
