package deploy

import (
	"context"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	eve "eve.evalgo.org/common"
)

func DeployInfisical(ctx context.Context, cli *client.Client, imageTag, containerName, volumeName string, envVars []string) error {
	eve.ImagePull(ctx, cli, imageTag, image.PullOptions{})
	port, _ := nat.NewPort("tcp", "8080")
	portBindings := nat.PortMap{
		port: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "8080"}},
	}
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:        imageTag,
		Env:          envVars,
		ExposedPorts: nat.PortSet{port: struct{}{}},
	}, &container.HostConfig{
		PortBindings: portBindings,
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
	}, &network.NetworkingConfig{}, nil, containerName)
	if err != nil {
		return err
	}
	return cli.ContainerStart(ctx, resp.ID, container.StartOptions{})
}
