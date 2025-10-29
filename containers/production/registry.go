package production

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"

	"eve.evalgo.org/common"
)

// RegistryProductionConfig holds configuration for production Docker Registry deployment.
type RegistryProductionConfig struct {
	// ContainerName is the name for the Docker Registry container
	ContainerName string
	// Image is the Docker image to use (default: "registry:3")
	Image string
	// Port is the host port to expose Registry HTTP API (default: 5000)
	Port string
	// DataVolume is the volume name for Registry data persistence
	DataVolume string
	// Production holds common production configuration
	Production ProductionConfig
}

// DefaultRegistryProductionConfig returns the default Docker Registry production configuration.
func DefaultRegistryProductionConfig() RegistryProductionConfig {
	return RegistryProductionConfig{
		ContainerName: "registry",
		Image:         "registry:3",
		Port:          "5000",
		DataVolume:    "registry-data",
		Production: ProductionConfig{
			NetworkName:   "app-network",
			CreateNetwork: true,
			VolumeName:    "registry-data",
			CreateVolume:  true,
		},
	}
}

// DeployRegistry deploys a production-ready Docker Registry container.
//
// Docker Registry is the open-source server-side application that stores and distributes
// Docker images. This function deploys a Registry container suitable for production use
// with persistent data storage.
//
// Container Features:
//   - Named container for consistent identification
//   - Persistent volume for image storage
//   - Custom network connectivity
//   - Fixed port mapping for stable access
//   - Restart policy for high availability
//   - Health checks for monitoring
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - config: Docker Registry production configuration
//
// Returns:
//   - string: Container ID of the deployed Registry container
//   - error: Deployment errors
//
// Example Usage:
//
//	ctx, cli := common.CtxCli("unix:///var/run/docker.sock")
//	defer cli.Close()
//
//	config := DefaultRegistryProductionConfig()
//
//	containerID, err := DeployRegistry(ctx, cli, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("Docker Registry deployed with ID: %s", containerID)
//	log.Printf("Registry URL: http://localhost:%s", config.Port)
//
// Connection URLs:
//
//	HTTP API V2:
//	http://localhost:{port}
//	http://{container_name}:{port} (from other containers)
//
//	Docker push/pull:
//	docker tag myimage localhost:{port}/myimage:tag
//	docker push localhost:{port}/myimage:tag
//	docker pull localhost:{port}/myimage:tag
//
// Data Persistence:
//
//	Registry data is stored in a Docker volume ({config.DataVolume}).
//	This ensures images and manifests persist across container restarts.
//
//	Volume mount points:
//	- /var/lib/registry - Image layers and manifests
//
// Network Configuration:
//
//	The container joins the specified Docker network, enabling
//	communication with other containers on the same network.
//
//	Other containers can pull images using the container name:
//	docker pull {container_name}:{port}/myimage:tag
//
// Security:
//
//	IMPORTANT: The default registry has NO AUTHENTICATION!
//	For production use:
//	- Enable basic authentication with htpasswd
//	- Use TLS/SSL certificates
//	- Place behind reverse proxy (nginx, traefik)
//	- Use network-level security (firewall rules)
//	- Consider Docker Registry with authentication plugins
//
// Restart Policy:
//
//	The container is configured with restart policy "unless-stopped",
//	ensuring it automatically restarts if it crashes.
//
// Health Monitoring:
//
//	Health check runs HTTP GET /v2/ every 30 seconds:
//	- Timeout: 10 seconds
//	- Retries: 3 consecutive failures before unhealthy
//
// Performance Tuning:
//
//	For production workloads, consider:
//	- Adequate disk space for image storage
//	- SSD storage for better I/O performance
//	- Enable garbage collection for cleanup
//	- Configure storage drivers (filesystem, S3, etc.)
//	- Set up CDN for faster image distribution
//
// Docker Registry Features:
//
//	Open-source registry implementation:
//	- Store and distribute Docker images
//	- Docker Registry HTTP API V2
//	- Content addressable storage
//	- Image manifest management
//	- Layer deduplication
//	- Garbage collection
//	- Webhook notifications
//	- Token-based authentication support
//
// Storage Drivers:
//
//	Registry supports various storage backends:
//	- filesystem (default) - local disk storage
//	- s3 - Amazon S3
//	- azure - Azure Blob Storage
//	- gcs - Google Cloud Storage
//	- swift - OpenStack Swift
//
// API Endpoints:
//
//	Key HTTP API V2 endpoints:
//	- GET  /v2/ - Check API version
//	- GET  /v2/_catalog - List repositories
//	- GET  /v2/{name}/tags/list - List tags
//	- GET  /v2/{name}/manifests/{reference} - Get manifest
//	- PUT  /v2/{name}/manifests/{reference} - Push manifest
//	- GET  /v2/{name}/blobs/{digest} - Get layer
//	- DELETE /v2/{name}/manifests/{reference} - Delete image
//
// Monitoring:
//
//	Monitor these metrics:
//	- Disk usage (image storage)
//	- Request rate and latency
//	- Push/pull operations
//	- Error rates
//	- Storage backend performance
//
// Backup Strategy:
//
//	Important: Implement regular backups!
//	- Backup /var/lib/registry volume
//	- Volume snapshots for disaster recovery
//	- Consider S3 or object storage for durability
//	- Test restore procedures regularly
//
// Error Handling:
//
//	Returns error if:
//	- Container with same name already exists
//	- Network or volume creation fails
//	- Docker API errors occur
//	- Invalid configuration provided
func DeployRegistry(ctx context.Context, cli common.DockerClient, config RegistryProductionConfig) (string, error) {
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

	// Pull Docker Registry image
	if err := common.ImagePullWithClient(ctx, cli, config.Image, &common.ImagePullOptions{Silent: true}); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Configure port bindings
	portBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: config.Port,
	}
	portMap := nat.PortMap{
		"5000/tcp": []nat.PortBinding{portBinding},
	}

	// Configure volume mounts
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: config.DataVolume,
			Target: "/var/lib/registry",
		},
	}

	// Container configuration
	containerConfig := container.Config{
		Image: config.Image,
		ExposedPorts: nat.PortSet{
			"5000/tcp": struct{}{},
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD-SHELL", "wget --spider --quiet http://localhost:5000/v2/ || exit 1"},
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

// StopRegistry stops a running Docker Registry container.
//
// Performs graceful shutdown to ensure data integrity.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the Registry container to stop
//
// Returns:
//   - error: Stop errors
//
// Example:
//
//	err := StopRegistry(ctx, cli, "registry")
//	if err != nil {
//	    log.Printf("Failed to stop Docker Registry: %v", err)
//	}
func StopRegistry(ctx context.Context, cli common.DockerClient, containerName string) error {
	timeout := 30 // 30 seconds for graceful shutdown
	return cli.ContainerStop(ctx, containerName, container.StopOptions{Timeout: &timeout})
}

// RemoveRegistry removes a Docker Registry container and optionally its volume.
//
// WARNING: Removing the volume will DELETE ALL IMAGES permanently!
// Always backup data before removing volumes.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the Registry container to remove
//   - removeVolume: Whether to also remove the data volume (DANGEROUS!)
//   - volumeName: Name of the data volume (required if removeVolume is true)
//
// Returns:
//   - error: Removal errors
//
// Example:
//
//	// Remove container but keep data (safe)
//	err := RemoveRegistry(ctx, cli, "registry", false, "")
//
//	// Remove container and data (DANGEROUS - backup first!)
//	err := RemoveRegistry(ctx, cli, "registry", true, "registry-data")
func RemoveRegistry(ctx context.Context, cli common.DockerClient, containerName string, removeVolume bool, volumeName string) error {
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

// GetRegistryURL returns the Docker Registry HTTP endpoint URL for the deployed container.
//
// This is a convenience function that formats the URL for the Registry HTTP API.
//
// Parameters:
//   - config: Docker Registry production configuration
//
// Returns:
//   - string: Docker Registry HTTP endpoint URL
//
// Example:
//
//	config := DefaultRegistryProductionConfig()
//	registryURL := GetRegistryURL(config)
//	// http://localhost:5000
func GetRegistryURL(config RegistryProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s", config.Port)
}

// GetRegistryImageURL returns the full URL for pushing/pulling a specific image.
//
// This is a convenience function that formats the image URL for Docker operations.
//
// Parameters:
//   - config: Docker Registry production configuration
//   - imageName: Name of the image (e.g., "myapp")
//   - tag: Tag of the image (e.g., "latest", "v1.0.0")
//
// Returns:
//   - string: Full image URL for Docker operations
//
// Example:
//
//	config := DefaultRegistryProductionConfig()
//	imageURL := GetRegistryImageURL(config, "myapp", "latest")
//	// localhost:5000/myapp:latest
//
//	// Use with Docker commands:
//	// docker tag myapp:latest localhost:5000/myapp:latest
//	// docker push localhost:5000/myapp:latest
//	// docker pull localhost:5000/myapp:latest
func GetRegistryImageURL(config RegistryProductionConfig, imageName, tag string) string {
	return fmt.Sprintf("localhost:%s/%s:%s", config.Port, imageName, tag)
}
