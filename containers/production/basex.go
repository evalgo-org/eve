package production

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"

	"eve.evalgo.org/common"
)

// BaseXProductionConfig holds configuration for production BaseX deployment.
type BaseXProductionConfig struct {
	// ContainerName is the name for the BaseX container
	ContainerName string
	// Image is the Docker image to use (default: "basex/basexhttp:latest")
	Image string
	// Port is the host port to expose BaseX REST API (default: 8984)
	Port string
	// AdminPassword is the BaseX admin password
	AdminPassword string
	// DataVolume is the volume name for BaseX data persistence
	DataVolume string
	// Production holds common production configuration
	Production ProductionConfig
}

// DefaultBaseXProductionConfig returns the default BaseX production configuration.
func DefaultBaseXProductionConfig() BaseXProductionConfig {
	return BaseXProductionConfig{
		ContainerName: "basex",
		Image:         "basex/basexhttp:latest",
		Port:          "8984",
		AdminPassword: "changeme",
		DataVolume:    "basex-data",
		Production: ProductionConfig{
			NetworkName:   "app-network",
			CreateNetwork: true,
			VolumeName:    "basex-data",
			CreateVolume:  true,
		},
	}
}

// DeployBaseX deploys a production-ready BaseX container.
//
// BaseX is an XML database with XQuery support. This function deploys a BaseX
// container suitable for production use with persistent data storage.
//
// Container Features:
//   - Named container for consistent identification
//   - Persistent volume for BaseX data
//   - Custom network connectivity
//   - Fixed port mapping for stable access
//   - Restart policy for high availability
//   - Health checks for monitoring
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - config: BaseX production configuration
//
// Returns:
//   - string: Container ID of the deployed BaseX container
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
//	config := DefaultBaseXProductionConfig()
//	config.AdminPassword = "secure-password"
//
//	containerID, err := DeployBaseX(ctx, cli, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("BaseX deployed with ID: %s", containerID)
//	log.Printf("Access at: http://localhost:%s", config.Port)
//
// BaseX REST API:
//
//	After deployment, BaseX REST API is available at:
//	http://localhost:{config.Port}
//
//	Default credentials:
//	- Username: admin
//	- Password: {config.AdminPassword}
//
// Data Persistence:
//
//	BaseX data is stored in a Docker volume ({config.DataVolume}).
//	This ensures data persists across container restarts and upgrades.
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
// Error Handling:
//
//	Returns error if:
//	- Container with same name already exists
//	- Network or volume creation fails
//	- Docker API errors occur
func DeployBaseX(ctx context.Context, cli common.DockerClient, config BaseXProductionConfig) (string, error) {
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

	// Pull BaseX image
	if err := common.ImagePullWithClient(ctx, cli, config.Image, &common.ImagePullOptions{Silent: true}); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Configure port bindings
	portBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: config.Port,
	}
	portMap := nat.PortMap{
		"8984/tcp": []nat.PortBinding{portBinding},
	}

	// Configure volume mounts
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: config.DataVolume,
			Target: "/srv/basex/data",
		},
	}

	// Container configuration
	containerConfig := container.Config{
		Image: config.Image,
		Env: []string{
			fmt.Sprintf("BASEX_ADMIN_PW=%s", config.AdminPassword),
		},
		ExposedPorts: nat.PortSet{
			"8984/tcp": struct{}{},
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD", "curl", "-f", "http://localhost:8984/"},
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

// StopBaseX stops a running BaseX container.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the BaseX container to stop
//
// Returns:
//   - error: Stop errors
//
// Example:
//
//	err := StopBaseX(ctx, cli, "basex")
//	if err != nil {
//	    log.Printf("Failed to stop BaseX: %v", err)
//	}
func StopBaseX(ctx context.Context, cli common.DockerClient, containerName string) error {
	timeout := 30 // 30 seconds graceful shutdown
	return cli.ContainerStop(ctx, containerName, container.StopOptions{Timeout: &timeout})
}

// RemoveBaseX removes a BaseX container and optionally its volume.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the BaseX container to remove
//   - removeVolume: Whether to also remove the data volume
//   - volumeName: Name of the data volume (required if removeVolume is true)
//
// Returns:
//   - error: Removal errors
//
// Example:
//
//	// Remove container but keep data
//	err := RemoveBaseX(ctx, cli, "basex", false, "")
//
//	// Remove container and data
//	err := RemoveBaseX(ctx, cli, "basex", true, "basex-data")
func RemoveBaseX(ctx context.Context, cli common.DockerClient, containerName string, removeVolume bool, volumeName string) error {
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
