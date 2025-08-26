package deploy

import (
	"context"

	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

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
	// wait for it to finish
	cli.ContainerWait(ctx, resp.ID, containertypes.WaitConditionNotRunning)
	return nil
}

func DeployZitiVolume(ctx context.Context, cli *client.Client, volumeName string) error {
	// create volume
	if err := CreateVolume(ctx, cli, volumeName); err != nil {
		return err
	}
	// run chown-controller (to set permissions)
	return zitiRunChownContainer(ctx, cli)
}

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
