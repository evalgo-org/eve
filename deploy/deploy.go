package deploy

import (
	"context"
	"io"
	// "time"

	// "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	// "github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	// "github.com/docker/go-connections/nat"
)

func CreateVolume(ctx context.Context, cli *client.Client, name string) error {
	_, err := cli.VolumeCreate(ctx, volume.CreateOptions{
		Name: name,
	})
	return err
}

func CreateNetwork(ctx context.Context, cli *client.Client, name string) error {
	_, err := cli.NetworkCreate(ctx, name, network.CreateOptions{
		Driver: "bridge",
	})
	return err
}

// func DeployEJBCA(ctx context.Context, cli *client.Client, volume string, networkName string) {
// 	image := "ejbca-private:latest"
// 	containerName := "ejbca"

// 	// Pull image
// 	// PullImage(cli, ctx, image)

// 	// Create container
// 	resp, err := cli.ContainerCreate(ctx, &container.Config{
// 		Image: image,
// 		Env: []string{
// 			"EJBCA_ADMIN_PASSWORD=supersecret",
// 			"EJBCA_DB=postgres",
// 			"EJBCA_DB_HOST=postgres",
// 			"EJBCA_DB_PORT=5432",
// 			"EJBCA_DB_USER=ejbca",
// 			"EJBCA_DB_PASSWORD=secretpw",
// 			"EJBCA_DB_NAME=ejbca",
// 		},
// 		ExposedPorts: nat.PortSet{
// 			"8080/tcp": {},
// 			"8443/tcp": {},
// 		},
// 	}, &container.HostConfig{
// 		PortBindings: nat.PortMap{
// 			"8080/tcp": {{HostIP: "0.0.0.0", HostPort: "8181"}},
// 			"8443/tcp": {{HostIP: "0.0.0.0", HostPort: "8443"}},
// 		},
// 		Mounts: []mount.Mount{
// 			{
// 				Type:   mount.TypeVolume,
// 				Source: volume,
// 				Target: "/opt/ejbca",
// 			},
// 		},
// 	}, &network.NetworkingConfig{}, nil, containerName)
// 	if err != nil {
// 		eve.Logger.Fatal("Failed to create EJBCA container:", err)
// 	}

// 	// Connect to network
// 	err = cli.NetworkConnect(ctx, networkName, resp.ID, &network.EndpointSettings{
// 		Aliases: []string{"ejbca"},
// 	})
// 	if err != nil {
// 		eve.Logger.Fatal("Failed to connect EJBCA to network:", err)
// 	}

// 	// Start container
// 	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
// 		eve.Logger.Fatal("Failed to start EJBCA container:", err)
// 	}
// }

func PullImage(cli *client.Client, ctx context.Context, imageTag string) error {
	out, err := cli.ImagePull(ctx, imageTag, image.PullOptions{})
	if err != nil {
		return err
	}
	defer out.Close()
	io.Copy(io.Discard, out)
	return nil
}

// func DeployEnvEJBCA(ctx context.Context, cli *client.Client) {
// 	// Volume names
// 	ejbcaVol := "ejbca_data"
// 	pgVol := "ejbca_pgdata"
// 	networkName := "ejbca_net"
// 	// Create volumes
// 	CreateVolume(ctx, cli, ejbcaVol)
// 	CreateVolume(ctx, cli, pgVol)
// 	// Create network
// 	CreateNetwork(ctx, cli, networkName)
// 	// Start PostgreSQL container
// 	DeployPostgres(ctx, cli, pgVol, networkName)
// 	// Wait a moment for DB to be ready
// 	time.Sleep(5 * time.Second)
// 	// Start EJBCA container
// 	DeployEJBCA(ctx, cli, ejbcaVol, networkName)
// }
