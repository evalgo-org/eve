package deploy

import (
	"context"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	eve "eve.evalgo.org/common"
)

func DeployKestraContainer(ctx context.Context, cli *client.Client, imageVersion, containerName, volumeName, postgresLink string, envVars []string) error {
	if err := CreateVolume(ctx, cli, volumeName); err != nil {
		return err
	}
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "docker.io/kestra/kestra:" + imageVersion,
		Env:   envVars,
		ExposedPorts: nat.PortSet{
			"8080/tcp": struct{}{},
		},
		Cmd: []string{"server", "standalone", "--worker-thread=128"},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			"8080/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "8080"}},
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: "/app/storage",
			},
		},
		Links: []string{postgresLink},
	}, nil, nil, "kestra")
	if err != nil {
		return err
	}
	return cli.ContainerStart(ctx, resp.ID, container.StartOptions{})
}

func DeployKestra(ctx context.Context, cli *client.Client) {
	// put both containers in the same network so that they can see each other
	CreateNetwork(ctx, cli, "kestra")
	// Step 1: Create and start PostgreSQL container
	DeployPostgres(ctx, cli, "15", "kestra-postgres", "kestra-postgres-data", []string{"POSTGRES_PASSWORD=kestrapassword", "POSTGRES_USER=kestra", "POSTGRES_DB=kestra"})
	// Step 2: Wait a bit for DB to be ready
	time.Sleep(5 * time.Second)
	eve.AddContainerToNetwork(ctx, cli, "kestra-postgres", "kestra")
	// Step 3: Create and start Kestra container
	config := `
datasources:
  postgres:
    url: jdbc:postgresql://kestra-postgres:5432/kestra
    driverClassName: org.postgresql.Driver
    username: kestra
    password: kestrapassword
server:
  port: 8080
kestra:
  repository:
    type: postgres
  queue:
    type: postgres
  storage:
    type: local
    local:
      basePath: "/app/storage"
`
	envVars := []string{"KESTRA_CONFIGURATION=" + config}
	DeployKestraContainer(ctx, cli, "latest-full", "kestra", "kestra-data", "kestra-postgres:kestra-postgres", envVars)
	eve.AddContainerToNetwork(ctx, cli, "kestra", "kestra")
}
