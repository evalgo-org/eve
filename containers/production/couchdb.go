package production

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"eve.evalgo.org/common"
)

// CouchDBProductionConfig holds configuration for production CouchDB deployment.
type CouchDBProductionConfig struct {
	// ContainerName is the name for the CouchDB container
	ContainerName string
	// Image is the Docker image to use (default: "couchdb:3")
	Image string
	// Port is the host port to expose CouchDB HTTP API (default: 5984)
	Port string
	// AdminUsername is the CouchDB admin username
	AdminUsername string
	// AdminPassword is the CouchDB admin password
	AdminPassword string
	// DataVolume is the volume name for CouchDB data persistence
	DataVolume string
	// Production holds common production configuration
	Production ProductionConfig
}

// DefaultCouchDBProductionConfig returns the default CouchDB production configuration.
func DefaultCouchDBProductionConfig() CouchDBProductionConfig {
	return CouchDBProductionConfig{
		ContainerName: "couchdb",
		Image:         "couchdb:3",
		Port:          "5984",
		AdminUsername: "admin",
		AdminPassword: "changeme",
		DataVolume:    "couchdb-data",
		Production: ProductionConfig{
			NetworkName:   "app-network",
			CreateNetwork: true,
			VolumeName:    "couchdb-data",
			CreateVolume:  true,
		},
	}
}

// DeployCouchDB deploys a production-ready CouchDB container.
//
// CouchDB is a document-oriented NoSQL database. This function deploys a CouchDB
// container suitable for production use with persistent data storage.
//
// Container Features:
//   - Named container for consistent identification
//   - Persistent volume for CouchDB data
//   - Custom network connectivity
//   - Fixed port mapping for stable access
//   - Restart policy for high availability
//   - Health checks for monitoring
//   - Single-node mode (suitable for most deployments)
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - config: CouchDB production configuration
//
// Returns:
//   - string: Container ID of the deployed CouchDB container
//   - error: Deployment errors
//
// Example Usage:
//
//	ctx, cli := common.CtxCli("unix:///var/run/docker.sock")
//	defer cli.Close()
//
//	config := DefaultCouchDBProductionConfig()
//	config.AdminUsername = "admin"
//	config.AdminPassword = "secure-password"
//
//	containerID, err := DeployCouchDB(ctx, cli, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("CouchDB deployed with ID: %s", containerID)
//	log.Printf("Access at: http://localhost:%s", config.Port)
//
// CouchDB HTTP API:
//
//	After deployment, CouchDB HTTP API is available at:
//	http://localhost:{config.Port}
//
//	Credentials:
//	- Username: {config.AdminUsername}
//	- Password: {config.AdminPassword}
//
//	Connection URL format:
//	http://{username}:{password}@localhost:{port}
//
// Data Persistence:
//
//	CouchDB data is stored in a Docker volume ({config.DataVolume}).
//	This ensures data persists across container restarts and upgrades.
//
//	Volume mount points:
//	- /opt/couchdb/data - Database files
//
// Network Configuration:
//
//	The container joins the specified Docker network, enabling
//	communication with other containers on the same network.
//
// Restart Policy:
//
//	The container is configured with restart policy "unless-stopped",
//	ensuring it automatically restarts if it crashes.
//
// Single Node Mode:
//
//	The container runs in single-node mode which is appropriate for
//	most deployments. For multi-node clustering, additional configuration
//	and containers are required.
//
// Error Handling:
//
//	Returns error if:
//	- Container with same name already exists
//	- Network or volume creation fails
//	- Docker API errors occur
func DeployCouchDB(ctx context.Context, cli *client.Client, config CouchDBProductionConfig) (string, error) {
	// Check if container already exists
	exists, err := common.ContainerExists(ctx, cli, config.ContainerName)
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

	// Pull CouchDB image
	if err := common.ImagePull(ctx, cli, config.Image, &common.ImagePullOptions{Silent: true}); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Configure port bindings
	portBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: config.Port,
	}
	portMap := nat.PortMap{
		"5984/tcp": []nat.PortBinding{portBinding},
	}

	// Configure volume mounts
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: config.DataVolume,
			Target: "/opt/couchdb/data",
		},
	}

	// Container configuration
	containerConfig := container.Config{
		Image: config.Image,
		Env: []string{
			fmt.Sprintf("COUCHDB_USER=%s", config.AdminUsername),
			fmt.Sprintf("COUCHDB_PASSWORD=%s", config.AdminPassword),
		},
		ExposedPorts: nat.PortSet{
			"5984/tcp": struct{}{},
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD", "curl", "-f", "http://localhost:5984/_up"},
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
	err = common.CreateAndStartContainer(ctx, cli, containerConfig, hostConfig, config.ContainerName, config.Production.NetworkName)
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

// StopCouchDB stops a running CouchDB container.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the CouchDB container to stop
//
// Returns:
//   - error: Stop errors
//
// Example:
//
//	err := StopCouchDB(ctx, cli, "couchdb")
//	if err != nil {
//	    log.Printf("Failed to stop CouchDB: %v", err)
//	}
func StopCouchDB(ctx context.Context, cli *client.Client, containerName string) error {
	timeout := 30 // 30 seconds graceful shutdown
	return cli.ContainerStop(ctx, containerName, container.StopOptions{Timeout: &timeout})
}

// RemoveCouchDB removes a CouchDB container and optionally its volume.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the CouchDB container to remove
//   - removeVolume: Whether to also remove the data volume
//   - volumeName: Name of the data volume (required if removeVolume is true)
//
// Returns:
//   - error: Removal errors
//
// Example:
//
//	// Remove container but keep data
//	err := RemoveCouchDB(ctx, cli, "couchdb", false, "")
//
//	// Remove container and data
//	err := RemoveCouchDB(ctx, cli, "couchdb", true, "couchdb-data")
func RemoveCouchDB(ctx context.Context, cli *client.Client, containerName string, removeVolume bool, volumeName string) error {
	// Remove container
	if err := cli.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	// Remove volume if requested
	if removeVolume && volumeName != "" {
		if err := cli.VolumeRemove(ctx, volumeName, true); err != nil {
			return fmt.Errorf("failed to remove volume: %w", err)
		}
	}

	return nil
}

// GetCouchDBConnectionURL builds a connection URL for the deployed CouchDB container.
//
// This is a convenience function that formats the connection URL with embedded credentials.
//
// Parameters:
//   - config: CouchDB production configuration
//
// Returns:
//   - string: CouchDB connection URL (e.g., "http://admin:password@localhost:5984")
//
// Example:
//
//	config := DefaultCouchDBProductionConfig()
//	url := GetCouchDBConnectionURL(config)
//	// Use url with CouchDB client or EVE db package
func GetCouchDBConnectionURL(config CouchDBProductionConfig) string {
	return fmt.Sprintf("http://%s:%s@localhost:%s",
		config.AdminUsername,
		config.AdminPassword,
		config.Port)
}
