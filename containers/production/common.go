// Package production provides production-ready container deployment functions.
//
// This package uses EVE's common/docker.go Docker API functions to create
// containers suitable for production environments with proper resource management,
// networking, and persistence.
//
// Key Features:
//   - Named containers for consistent identification
//   - Persistent volumes for data storage
//   - Custom networks for service isolation
//   - Health checks and restart policies
//   - Fixed port mappings for stable access
//
// Differences from Testing:
//   - Uses Docker API directly (not testcontainers-go)
//   - Creates persistent resources (volumes, networks)
//   - Requires manual cleanup (not automatic)
//   - Suitable for long-running deployments
//
// Example Usage:
//
//	ctx, cli := common.CtxCli("unix:///var/run/docker.sock")
//	defer cli.Close()
//
//	config := DefaultBaseXProductionConfig()
//	containerID, err := DeployBaseX(ctx, cli, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("BaseX deployed: %s", containerID)
package production

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"

	"eve.evalgo.org/common"
)

// ProductionConfig defines common configuration for production containers.
type ProductionConfig struct {
	// NetworkName is the Docker network to connect the container to
	NetworkName string
	// CreateNetwork indicates whether to create the network if it doesn't exist
	CreateNetwork bool
	// VolumeName is the Docker volume for persistent data (optional)
	VolumeName string
	// CreateVolume indicates whether to create the volume if it doesn't exist
	CreateVolume bool
}

// DefaultProductionConfig returns default production configuration.
func DefaultProductionConfig() ProductionConfig {
	return ProductionConfig{
		NetworkName:   "app-network",
		CreateNetwork: true,
		VolumeName:    "",
		CreateVolume:  false,
	}
}

// EnsureNetwork creates a Docker network if it doesn't exist.
//
// This function checks if a network exists and creates it if needed.
// Safe to call multiple times - idempotent operation.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker client
//   - networkName: Name of the network to ensure
//
// Returns:
//   - error: Network creation errors (nil if network exists or was created)
//
// Example:
//
//	err := EnsureNetwork(ctx, cli, "app-network")
//	if err != nil {
//	    return fmt.Errorf("failed to ensure network: %w", err)
//	}
func EnsureNetwork(ctx context.Context, cli common.DockerClient, networkName string) error {
	// Check if network already exists
	networks, err := cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	for _, net := range networks {
		if net.Name == networkName {
			return nil // Network already exists
		}
	}

	// Create network
	err = common.CreateNetworkWithClient(ctx, cli, networkName)
	if err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	return nil
}

// EnsureVolume creates a Docker volume if it doesn't exist.
//
// This function checks if a volume exists and creates it if needed.
// Safe to call multiple times - idempotent operation.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker client
//   - volumeName: Name of the volume to ensure
//
// Returns:
//   - error: Volume creation errors (nil if volume exists or was created)
//
// Example:
//
//	err := EnsureVolume(ctx, cli, "basex-data")
//	if err != nil {
//	    return fmt.Errorf("failed to ensure volume: %w", err)
//	}
func EnsureVolume(ctx context.Context, cli common.DockerClient, volumeName string) error {
	// Check if volume already exists
	volumes, err := cli.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list volumes: %w", err)
	}

	for _, vol := range volumes.Volumes {
		if vol.Name == volumeName {
			return nil // Volume already exists
		}
	}

	// Create volume
	err = common.CreateVolumeWithClient(ctx, cli, volumeName)
	if err != nil {
		return fmt.Errorf("failed to create volume: %w", err)
	}

	return nil
}

// PrepareProductionEnvironment ensures network and volume exist for deployment.
//
// This function prepares the production environment by creating the specified
// network and volume if they don't already exist.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker client
//   - config: Production configuration
//
// Returns:
//   - error: Preparation errors
//
// Example:
//
//	config := ProductionConfig{
//	    NetworkName: "app-network",
//	    CreateNetwork: true,
//	    VolumeName: "data-volume",
//	    CreateVolume: true,
//	}
//	err := PrepareProductionEnvironment(ctx, cli, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
func PrepareProductionEnvironment(ctx context.Context, cli common.DockerClient, config ProductionConfig) error {
	// Ensure network exists
	if config.CreateNetwork && config.NetworkName != "" {
		if err := EnsureNetwork(ctx, cli, config.NetworkName); err != nil {
			return fmt.Errorf("failed to prepare network: %w", err)
		}
	}

	// Ensure volume exists
	if config.CreateVolume && config.VolumeName != "" {
		if err := EnsureVolume(ctx, cli, config.VolumeName); err != nil {
			return fmt.Errorf("failed to prepare volume: %w", err)
		}
	}

	return nil
}
