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

// DeployPostgresLocal deploys a PostgreSQL container locally, mapping the container's port to a host port.
// This function is intended for development and testing environments where direct access to the database is required.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - cli: Docker client for container management
//   - imageVersion: PostgreSQL Docker image version (e.g., "14-alpine")
//   - containerName: Name to assign to the container
//   - volumeName: Name of the Docker volume for persistent data storage
//   - envArgs: Environment variables to pass to the PostgreSQL container (e.g., POSTGRES_PASSWORD)
//   - localPort: Host port to map to the container's PostgreSQL port (5432)
//
// Returns:
//   - error: If any step fails (volume creation, container creation, or container start)
func DeployPostgresLocal(ctx context.Context, cli *client.Client, imageVersion, containerName, volumeName string, envArgs []string, localPort int) error {
	if err := CreateVolume(ctx, cli, volumeName); err != nil {
		return err
	}
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "docker.io/library/postgres:" + imageVersion,
		Env:   envArgs,
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			"5432/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: strconv.Itoa(localPort)}},
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: "/var/lib/postgresql/data",
			},
		},
	}, nil, nil, containerName)
	if err != nil {
		return err
	}
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return err
	}
	return nil
}

// DeployPostgres deploys a PostgreSQL container without exposing ports to the host.
// This function is suitable for production or containerized environments where networking is managed externally.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - cli: Docker client for container management
//   - imageVersion: PostgreSQL Docker image version (e.g., "14-alpine")
//   - containerName: Name to assign to the container
//   - volumeName: Name of the Docker volume for persistent data storage
//   - envArgs: Environment variables to pass to the PostgreSQL container (e.g., POSTGRES_PASSWORD)
//
// Returns:
//   - error: If any step fails (image pull, container creation, or container start)
func DeployPostgres(ctx context.Context, cli *client.Client, imageVersion, containerName, volumeName string, envArgs []string) error {
	imageTag := "docker.io/library/postgres:" + imageVersion
	// Pull image
	if err := PullImage(cli, ctx, imageTag); err != nil {
		return err
	}
	// Create container
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageTag,
		Env:   envArgs,
	}, &container.HostConfig{
		Mounts: []mount.Mount{
			{Type: mount.TypeVolume, Source: volumeName, Target: "/var/lib/postgresql/data"},
		},
	}, &network.NetworkingConfig{}, nil, containerName)
	if err != nil {
		return err
	}
	// Start container
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return err
	}
	return nil
}
