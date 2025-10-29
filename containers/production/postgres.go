package production

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"

	"eve.evalgo.org/common"
)

// PostgresProductionConfig holds configuration for production PostgreSQL deployment.
type PostgresProductionConfig struct {
	// ContainerName is the name for the PostgreSQL container
	ContainerName string
	// Image is the Docker image to use (default: "postgres:17")
	Image string
	// Port is the host port to expose PostgreSQL (default: 5432)
	Port string
	// Username is the PostgreSQL superuser username
	Username string
	// Password is the PostgreSQL superuser password
	Password string
	// Database is the default database to create
	Database string
	// DataVolume is the volume name for PostgreSQL data persistence
	DataVolume string
	// Production holds common production configuration
	Production ProductionConfig
}

// DefaultPostgresProductionConfig returns the default PostgreSQL production configuration.
func DefaultPostgresProductionConfig() PostgresProductionConfig {
	return PostgresProductionConfig{
		ContainerName: "postgres",
		Image:         "postgres:17",
		Port:          "5432",
		Username:      "postgres",
		Password:      "changeme",
		Database:      "postgres",
		DataVolume:    "postgres-data",
		Production: ProductionConfig{
			NetworkName:   "app-network",
			CreateNetwork: true,
			VolumeName:    "postgres-data",
			CreateVolume:  true,
		},
	}
}

// DeployPostgres deploys a production-ready PostgreSQL container.
//
// PostgreSQL is a powerful, open-source relational database. This function deploys a
// PostgreSQL container suitable for production use with persistent data storage.
//
// Container Features:
//   - Named container for consistent identification
//   - Persistent volume for database data
//   - Custom network connectivity
//   - Fixed port mapping for stable access
//   - Restart policy for high availability
//   - Health checks for monitoring
//   - SCRAM-SHA-256 authentication (secure)
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - config: PostgreSQL production configuration
//
// Returns:
//   - string: Container ID of the deployed PostgreSQL container
//   - error: Deployment errors
//
// Example Usage:
//
//	ctx, cli := common.CtxCli("unix:///var/run/docker.sock")
//	defer cli.Close()
//
//	config := DefaultPostgresProductionConfig()
//	config.Username = "postgres"
//	config.Password = "secure-password-here"
//	config.Database = "myapp"
//
//	containerID, err := DeployPostgres(ctx, cli, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("PostgreSQL deployed with ID: %s", containerID)
//	log.Printf("Connection: postgresql://%s:****@localhost:%s/%s",
//	    config.Username, config.Port, config.Database)
//
// Connection String:
//
//	After deployment, connect to PostgreSQL using:
//	postgresql://{username}:{password}@localhost:{port}/{database}
//
//	With SSL (production):
//	postgresql://{username}:{password}@localhost:{port}/{database}?sslmode=require
//
//	Without SSL (development):
//	postgresql://{username}:{password}@localhost:{port}/{database}?sslmode=disable
//
// Data Persistence:
//
//	PostgreSQL data is stored in a Docker volume ({config.DataVolume}).
//	This ensures data persists across container restarts and upgrades.
//
//	Volume mount points:
//	- /var/lib/postgresql/data - Database files and configuration
//	- PGDATA=/var/lib/postgresql/data/pgdata - Actual data directory
//
// Authentication:
//
//	Uses SCRAM-SHA-256 for password authentication (PostgreSQL 14+ default).
//	This is more secure than MD5 (legacy) authentication.
//
//	IMPORTANT: Always set a strong password in production!
//	config.Password = "strong-random-password-here"
//
// Network Configuration:
//
//	The container joins the specified Docker network, enabling
//	communication with other containers on the same network.
//
//	Other containers can connect using the container name:
//	postgresql://{username}:{password}@{container_name}:5432/{database}
//
// Restart Policy:
//
//	The container is configured with restart policy "unless-stopped",
//	ensuring it automatically restarts if it crashes.
//
// Health Monitoring:
//
//	Health check runs pg_isready every 30 seconds:
//	- Timeout: 10 seconds
//	- Retries: 3 consecutive failures before unhealthy
//	- Command: pg_isready -U {username}
//
// Performance Tuning:
//
//	For production workloads, consider tuning PostgreSQL settings:
//	- shared_buffers (25% of RAM)
//	- effective_cache_size (75% of RAM)
//	- max_connections (based on workload)
//	- work_mem (based on queries)
//
//	Mount custom postgresql.conf via volume or environment variables.
//
// Backup Strategy:
//
//	Important: Implement regular backups!
//	- pg_dump for logical backups
//	- pg_basebackup for physical backups
//	- Volume snapshots for disaster recovery
//	- Consider PostgreSQL replication for HA
//
// Monitoring:
//
//	Monitor these metrics:
//	- Connection count (max_connections)
//	- Database size and growth
//	- Query performance (slow query log)
//	- Replication lag (if using replicas)
//	- Cache hit ratio
//
// Error Handling:
//
//	Returns error if:
//	- Container with same name already exists
//	- Network or volume creation fails
//	- Docker API errors occur
//	- Invalid configuration provided
func DeployPostgres(ctx context.Context, cli common.DockerClient, config PostgresProductionConfig) (string, error) {
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

	// Pull PostgreSQL image
	if err := common.ImagePullWithClient(ctx, cli, config.Image, &common.ImagePullOptions{Silent: true}); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Configure port bindings
	portBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: config.Port,
	}
	portMap := nat.PortMap{
		"5432/tcp": []nat.PortBinding{portBinding},
	}

	// Configure volume mounts
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: config.DataVolume,
			Target: "/var/lib/postgresql/data",
		},
	}

	// Container configuration
	containerConfig := container.Config{
		Image: config.Image,
		Env: []string{
			fmt.Sprintf("POSTGRES_USER=%s", config.Username),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", config.Password),
			fmt.Sprintf("POSTGRES_DB=%s", config.Database),
			"POSTGRES_INITDB_ARGS=--auth-host=scram-sha-256",
			"PGDATA=/var/lib/postgresql/data/pgdata",
		},
		ExposedPorts: nat.PortSet{
			"5432/tcp": struct{}{},
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD-SHELL", fmt.Sprintf("pg_isready -U %s", config.Username)},
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

// StopPostgres stops a running PostgreSQL container.
//
// Performs graceful shutdown to ensure data integrity.
// PostgreSQL uses smart shutdown mode (waits for active connections).
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the PostgreSQL container to stop
//
// Returns:
//   - error: Stop errors
//
// Example:
//
//	err := StopPostgres(ctx, cli, "postgres")
//	if err != nil {
//	    log.Printf("Failed to stop PostgreSQL: %v", err)
//	}
func StopPostgres(ctx context.Context, cli common.DockerClient, containerName string) error {
	timeout := 60 // 60 seconds for graceful shutdown (waits for connections)
	return cli.ContainerStop(ctx, containerName, container.StopOptions{Timeout: &timeout})
}

// RemovePostgres removes a PostgreSQL container and optionally its volume.
//
// WARNING: Removing the volume will DELETE ALL DATA permanently!
// Always backup data before removing volumes.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the PostgreSQL container to remove
//   - removeVolume: Whether to also remove the data volume (DANGEROUS!)
//   - volumeName: Name of the data volume (required if removeVolume is true)
//
// Returns:
//   - error: Removal errors
//
// Example:
//
//	// Remove container but keep data (safe)
//	err := RemovePostgres(ctx, cli, "postgres", false, "")
//
//	// Remove container and data (DANGEROUS - backup first!)
//	err := RemovePostgres(ctx, cli, "postgres", true, "postgres-data")
func RemovePostgres(ctx context.Context, cli common.DockerClient, containerName string, removeVolume bool, volumeName string) error {
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

// GetPostgresConnectionString builds a connection string for the deployed PostgreSQL container.
//
// This is a convenience function that formats the connection string for PostgreSQL clients.
//
// Parameters:
//   - config: PostgreSQL production configuration
//   - sslmode: SSL mode (disable, allow, prefer, require, verify-ca, verify-full)
//
// Returns:
//   - string: PostgreSQL connection string
//
// Example:
//
//	config := DefaultPostgresProductionConfig()
//	config.Password = "secure-password"
//
//	// For development
//	connStr := GetPostgresConnectionString(config, "disable")
//	// postgresql://postgres:secure-password@localhost:5432/postgres?sslmode=disable
//
//	// For production
//	connStr := GetPostgresConnectionString(config, "require")
//	// postgresql://postgres:secure-password@localhost:5432/postgres?sslmode=require
func GetPostgresConnectionString(config PostgresProductionConfig, sslmode string) string {
	return fmt.Sprintf("postgresql://%s:%s@localhost:%s/%s?sslmode=%s",
		config.Username,
		config.Password,
		config.Port,
		config.Database,
		sslmode)
}
