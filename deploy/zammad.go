package deploy

import (
	"context"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	eve "eve.evalgo.org/common"
)

func DeployZammad(ctx context.Context, cli *client.Client, prefix, net string, openziti bool) error {
	// get all required images
	eve.ImagePull(ctx, cli, "postgres:17.5-alpine", image.PullOptions{})
	eve.ImagePull(ctx, cli, "bitnami/elasticsearch:8.18.0", image.PullOptions{})
	eve.ImagePull(ctx, cli, "ghcr.io/zammad/zammad:6.5.0-101", image.PullOptions{})
	eve.ImagePull(ctx, cli, "redis:7.4.4-alpine", image.PullOptions{})
	// set all container names
	zammad_postgres := prefix + "-postgresql"
	zammad_elasticsearch := prefix + "-elasticsearch"
	zammad_redis := prefix + "-redis"
	zammad_railsserver := prefix + "-railsserver"
	zammad_scheduler := prefix + "-scheduler"
	zammad_websocket := prefix + "-websocket"
	zammad_nginx := prefix + "-nginx"
	_, err := cli.NetworkCreate(ctx, net, network.CreateOptions{})
	if err != nil {
		eve.Logger.Info(err)
		// return err
	}
	volumes := map[string]string{
		"postgresql-data":    "/var/lib/postgresql/data",
		"redis-data":         "/data",
		"elasticsearch-data": "/bitnami/elasticsearch/data",
		"zammad-storage":     "/opt/zammad/storage",
		"zammad-backup":      "/var/tmp/zammad",
	}
	eve.Logger.Info(eve.CreateAndStartContainer(ctx, cli, container.Config{
		Image: "postgres:17.5-alpine",
		Env: []string{
			"POSTGRES_USER=zammad",
			"POSTGRES_PASSWORD=zammad",
			"POSTGRES_DB=zammad_production",
		},
	}, container.HostConfig{
		Mounts:        []mount.Mount{{Source: "postgresql-data", Target: volumes["postgresql-data"], Type: mount.TypeVolume}},
		RestartPolicy: container.RestartPolicy{Name: "always"},
	}, zammad_postgres, net))

	eve.Logger.Info("wait for postgres to boot...")
	time.Sleep(10 * time.Second)

	eve.Logger.Info(eve.CreateAndStartContainer(ctx, cli, container.Config{
		Image: "bitnami/elasticsearch:8.18.0",
		Env: []string{
			"ELASTICSEARCH_ENABLE_SECURITY=true",
			"ELASTICSEARCH_SKIP_TRANSPORT_TLS=true",
			"ELASTICSEARCH_ENABLE_REST_TLS=false",
			"ELASTICSEARCH_PASSWORD=zammad",
		},
	}, container.HostConfig{
		Mounts:        []mount.Mount{{Source: "elasticsearch-data", Target: volumes["elasticsearch-data"], Type: mount.TypeVolume}},
		RestartPolicy: container.RestartPolicy{Name: "always"},
	}, zammad_elasticsearch, net))

	eve.Logger.Info("wait for elasticsearch to boot...")
	time.Sleep(30 * time.Second)

	eve.Logger.Info(eve.CreateAndStartContainer(ctx, cli, container.Config{
		Image: "redis:7.4.4-alpine",
	}, container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: "always"},
	}, zammad_redis, net))

	railsEnv := []string{
		"POSTGRESQL_DB=zammad_production",
		"POSTGRESQL_HOST=zammad-postgresql",
		"POSTGRESQL_USER=zammad",
		"POSTGRESQL_PASS=zammad",
		"POSTGRESQL_PORT=5432",
		"POSTGRESQL_DB_CREATE=true",
		"ELASTICSEARCH_HOST=zammad-elasticsearch",
		"POSTGRESQL_DB=zammad_production",
		"POSTGRESQL_USER=zammad",
		"POSTGRESQL_PASS=zammad",
		"REDIS_URL=redis://zammad-redis:6379",
		"ELASTICSEARCH_SCHEMA=http",
		"ELASTICSEARCH_ENABLE_SECURITY=true",
		"ELASTICSEARCH_SKIP_TRANSPORT_TLS=true",
		"ELASTICSEARCH_ENABLE_REST_TLS=false",
		"ELASTICSEARCH_PASSWORD=zammad",
		"ELASTICSEARCH_PASS=zammad",
		"ELASTICSEARCH_USER=elastic",
		"TZ=Europe/Berlin",
	}

	eve.Logger.Info(eve.CreateAndStartContainer(ctx, cli, container.Config{
		Image: "ghcr.io/zammad/zammad:6.5.0-101",
		Env:   railsEnv,
		Cmd:   []string{"zammad-init"},
	}, container.HostConfig{
		AutoRemove:    true,
		Mounts:        []mount.Mount{{Source: "zammad-storage", Target: volumes["zammad-storage"], Type: mount.TypeVolume}},
		RestartPolicy: container.RestartPolicy{},
	}, "tmp-zammad-init", net))

	eve.Logger.Info(eve.CreateAndStartContainer(ctx, cli, container.Config{
		Image: "ghcr.io/zammad/zammad:6.5.0-101",
		Env:   railsEnv,
		Cmd:   []string{"zammad-railsserver"},
	}, container.HostConfig{
		Mounts:        []mount.Mount{{Source: "zammad-storage", Target: volumes["zammad-storage"], Type: mount.TypeVolume}},
		RestartPolicy: container.RestartPolicy{Name: "always"},
	}, zammad_railsserver, net))

	eve.CreateAndStartContainer(ctx, cli, container.Config{
		Image: "ghcr.io/zammad/zammad:6.5.0-101",
		Env:   railsEnv,
		Cmd:   []string{"zammad-scheduler"},
	}, container.HostConfig{
		Mounts:        []mount.Mount{{Source: "zammad-storage", Target: volumes["zammad-storage"], Type: mount.TypeVolume}},
		RestartPolicy: container.RestartPolicy{Name: "always"},
	}, zammad_scheduler, net)

	eve.CreateAndStartContainer(ctx, cli, container.Config{
		Image: "ghcr.io/zammad/zammad:6.5.0-101",
		Env:   railsEnv,
		Cmd:   []string{"zammad-websocket"},
	}, container.HostConfig{
		Mounts:        []mount.Mount{{Source: "zammad-storage", Target: volumes["zammad-storage"], Type: mount.TypeVolume}},
		RestartPolicy: container.RestartPolicy{Name: "always"},
	}, zammad_websocket, net)

	portBindings := nat.PortMap{}
	eve.CreateAndStartContainer(ctx, cli, container.Config{
		Image:        "ghcr.io/zammad/zammad:6.5.0-101",
		Env:          railsEnv,
		ExposedPorts: nat.PortSet{"8080/tcp": {}},
		Cmd:          []string{"zammad-nginx"},
	}, container.HostConfig{
		PortBindings:  portBindings,
		Mounts:        []mount.Mount{{Source: "zammad-backup", Target: volumes["zammad-backup"], Type: mount.TypeVolume}},
		RestartPolicy: container.RestartPolicy{Name: "always"},
	}, zammad_nginx, net)
	if openziti {
		return eve.AddContainerToNetwork(ctx, cli, zammad_nginx, "openziti")
	}
	return nil
}
