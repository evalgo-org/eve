// Package deploy provides utilities for deploying PostgreSQL containers using Docker.
// It supports both local deployments (with port mapping) and standard deployments,
// with options for customizing the image version, container name, volume, and environment variables.
//
// Features:
//   - Local PostgreSQL deployment with host port mapping
//   - Standard PostgreSQL deployment for containerized environments
//   - Volume creation and mounting for persistent data storage
//   - Environment variable configuration for PostgreSQL
package deploy

import (
	"context"
	"strconv"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// PostgresDeployOptions contains options for PostgreSQL deployment.
type PostgresDeployOptions struct {
	// LocalPort maps the container's port 5432 to a host port (0 or nil = no port mapping)
	LocalPort *int
	// PullImage determines whether to pull the image before deployment
	PullImage bool
	// CreateVolume determines whether to create the volume if it doesn't exist
	CreateVolume bool
}

// DeployPostgres deploys a PostgreSQL container with flexible configuration options.
// This consolidated function handles both local (with port mapping) and production
// (without port mapping) deployments.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - cli: Docker client for container management
//   - imageVersion: PostgreSQL Docker image version (e.g., "14-alpine")
//   - containerName: Name to assign to the container
//   - volumeName: Name of the Docker volume for persistent data storage
//   - envArgs: Environment variables to pass to the PostgreSQL container (e.g., POSTGRES_PASSWORD)
//   - opts: PostgresDeployOptions (can be nil for defaults)
//
// Default Behavior (opts == nil):
//   - No port mapping (suitable for production/containerized environments)
//   - No image pull (assumes image already exists)
//   - No volume creation (assumes volume already exists)
//
// Example Usage:
//
//	// Production deployment (no port mapping)
//	err := DeployPostgres(ctx, cli, "14-alpine", "pg-prod", "pg-data", envVars, nil)
//
//	// Local deployment with port mapping
//	localPort := 5432
//	err := DeployPostgres(ctx, cli, "14-alpine", "pg-local", "pg-data", envVars, &PostgresDeployOptions{
//	    LocalPort: &localPort,
//	})
//
//	// Full deployment with all options
//	err := DeployPostgres(ctx, cli, "14-alpine", "pg-local", "pg-data", envVars, &PostgresDeployOptions{
//	    LocalPort: &localPort,
//	    PullImage: true,
//	    CreateVolume: true,
//	})
//
// Returns:
//   - error: If any step fails (volume creation, image pull, container creation, or container start)
func DeployPostgres(ctx context.Context, cli *client.Client, imageVersion, containerName, volumeName string, envArgs []string, opts *PostgresDeployOptions) error {
	imageTag := "docker.io/library/postgres:" + imageVersion

	// Create volume if requested
	if opts != nil && opts.CreateVolume {
		if err := CreateVolume(ctx, cli, volumeName); err != nil {
			return err
		}
	}

	// Pull image if requested
	if opts != nil && opts.PullImage {
		if err := PullImage(cli, ctx, imageTag); err != nil {
			return err
		}
	}

	// Prepare host config
	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: "/var/lib/postgresql/data",
			},
		},
	}

	// Add port bindings if local port is specified
	if opts != nil && opts.LocalPort != nil {
		hostConfig.PortBindings = nat.PortMap{
			"5432/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: strconv.Itoa(*opts.LocalPort)}},
		}
	}

	// Create container
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageTag,
		Env:   envArgs,
	}, hostConfig, &network.NetworkingConfig{}, nil, containerName)
	if err != nil {
		return err
	}

	// Start container
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return err
	}

	return nil
}

// DeployPostgresLocal deploys a PostgreSQL container locally, mapping the container's port to a host port.
// Deprecated: Use DeployPostgres with PostgresDeployOptions{LocalPort: &localPort, CreateVolume: true} instead.
// This function is maintained for backward compatibility.
func DeployPostgresLocal(ctx context.Context, cli *client.Client, imageVersion, containerName, volumeName string, envArgs []string, localPort int) error {
	return DeployPostgres(ctx, cli, imageVersion, containerName, volumeName, envArgs, &PostgresDeployOptions{
		LocalPort:    &localPort,
		CreateVolume: true,
	})
}
