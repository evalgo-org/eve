package deploy

import (
	"context"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	eve "eve.evalgo.org/common"
)

func DeployNexus3(ctx context.Context, cli *client.Client, imageTag, containerName, volumeName string) {
	// Pull image if not already available
	eve.Logger.Info("Pulling image:", imageTag)
	eve.ImagePull(ctx, cli, imageTag, image.PullOptions{})
	// Port bindings
	port, _ := nat.NewPort("tcp", "8081")
	portBinding := nat.PortMap{
		port: []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: "8081",
			},
		},
	}
	// Container configuration
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageTag,
		ExposedPorts: nat.PortSet{
			port: struct{}{},
		},
	}, &container.HostConfig{
		PortBindings: portBinding,
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: "/nexus-data",
			},
		},
	}, &network.NetworkingConfig{}, nil, containerName)
	if err != nil {
		eve.Logger.Fatal("Error creating container:", err)
	}
	// Start container
	err = cli.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		eve.Logger.Fatal("Error starting container:", err)
	}
}
