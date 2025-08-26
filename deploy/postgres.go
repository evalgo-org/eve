package deploy

import (
	"context"
	"strconv"

	"github.com/docker/docker/api/types/container"
	// "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

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
