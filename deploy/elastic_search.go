package deploy

import (
	"context"

	"github.com/docker/docker/api/types/container"
	// "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

func DeployElasticsearch(cli *client.Client, ctx context.Context, net string) error {
	_, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "docker.elastic.co/elasticsearch/elasticsearch:7.10.2",
		Env: []string{
			"discovery.type=single-node",
			"ES_JAVA_OPTS=-Xms512m -Xmx512m",
		},
	}, &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: "always"},
	}, &network.NetworkingConfig{}, nil, "env-zammad-elasticsearch")
	if err != nil {
		return err
	}
	return cli.ContainerStart(ctx, "env-zammad-elasticsearch", container.StartOptions{})
}
