// Package deploy provides utilities for deploying OpenZiti network components using Docker.
// It includes functions for setting up Ziti controllers, managing volumes with proper permissions,
// and configuring container networking for OpenZiti services.
//
// Features:
//   - Ziti controller deployment with custom configuration
//   - Volume creation and permission management
//   - Network configuration for Ziti services
//   - Container lifecycle management
package deploy

import (
	"context"

	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// zitiRunChownContainer runs a temporary container to set proper permissions on the Ziti controller volume.
// This function creates and starts a BusyBox container that executes a chown command to ensure the
// Ziti controller has the correct permissions (UID 2171) on its data directory.
//
// The container automatically removes itself after completion.
//
// Returns:
//   - error: If container creation, startup, or execution fails
func zitiRunChownContainer(ctx context.Context, cli *client.Client) error {
	config := &containertypes.Config{
		Image: "busybox",
		Cmd:   []string{"chown", "-R", "2171", "/ziti-controller"},
	}
	hostConfig := &containertypes.HostConfig{
		Binds:      []string{"ziti-controller:/ziti-controller"},
		AutoRemove: true,
	}
	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "chown-controller")
	if err != nil {
		return err
	}
	if err := cli.ContainerStart(ctx, resp.ID, containertypes.StartOptions{}); err != nil {
		return err
	}
	// Wait for the container to finish executing the chown command
	_, err = cli.ContainerWait(ctx, resp.ID, containertypes.WaitConditionNotRunning)
	return err
}

// DeployZitiVolume creates a Docker volume for the Ziti controller and sets the appropriate permissions.
// This function:
//  1. Creates a named volume for the Ziti controller
//  2. Runs a temporary container to set the correct ownership (UID 2171) on the volume
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - cli: Docker client for volume and container management
//   - volumeName: Name of the volume to create (default bind mount name is "ziti-controller")
//
// Returns:
//   - error: If volume creation or permission setting fails
func DeployZitiVolume(ctx context.Context, cli *client.Client, volumeName string) error {
	// Create the Docker volume
	if err := CreateVolume(ctx, cli, volumeName); err != nil {
		return err
	}
	// Run container to set permissions on the volume
	return zitiRunChownContainer(ctx, cli)
}

// DeployZitiController deploys an OpenZiti controller container with the specified configuration.
// This function:
//  1. Creates a container using the official OpenZiti controller image
//  2. Configures the container with the provided environment variables
//  3. Exposes port 1280 for controller API access
//  4. Mounts the Ziti controller volume with proper permissions
//  5. Connects the container to the "ziti" network with the alias "ziti-controller"
//  6. Sets a restart policy of "unless-stopped" for resilience
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - cli: Docker client for container management
//   - ctrlName: Name to assign to the controller container
//   - envVars: Environment variables for controller configuration
//
// Returns:
//   - error: If container creation or startup fails
//
// Network Configuration:
//   - The container is connected to the "ziti" network with the alias "ziti-controller"
//   - Port 1280 is exposed and mapped to host port 1280
//
// Volume Requirements:
//   - A volume named "ziti-controller" must exist (typically created by DeployZitiVolume)
//
// Example Environment Variables:
//   - ZITI_CTRL_ADVERTISED_ADDRESS: The advertised address for the controller
//   - ZITI_CTRL_EDGE_IDENTITY_ENROLLMENT_DURATION: Duration for identity enrollment
//   - Other Ziti controller configuration variables as needed
func DeployZitiController(ctx context.Context, cli *client.Client, ctrlName string, envVars []string) error {
	portSet := nat.PortSet{
		"1280/tcp": struct{}{},
	}
	portMap := nat.PortMap{
		"1280/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "1280"}},
	}
	config := &containertypes.Config{
		Image:        "openziti/ziti-controller:1.6.5",
		Cmd:          []string{"run", "config.yml"},
		Env:          envVars,
		ExposedPorts: portSet,
	}
	hostConfig := &containertypes.HostConfig{
		Binds:        []string{"ziti-controller:/ziti-controller"},
		PortBindings: portMap,
		RestartPolicy: containertypes.RestartPolicy{
			Name: "unless-stopped",
		},
	}
	networking := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"ziti": {Aliases: []string{"ziti-controller"}},
		},
	}
	resp, err := cli.ContainerCreate(ctx, config, hostConfig, networking, nil, ctrlName)
	if err != nil {
		return err
	}
	return cli.ContainerStart(ctx, resp.ID, containertypes.StartOptions{})
}
