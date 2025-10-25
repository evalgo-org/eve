// Package deploy provides utilities for deploying containerized applications using Docker.
// It includes functions for deploying complex multi-container applications like Zammad,
// with support for custom networks, volumes, environment variables, and OpenZiti integration.
//
// Features:
//   - Multi-container application deployment
//   - Custom Docker network creation
//   - Volume management for persistent data
//   - Environment variable configuration
//   - OpenZiti network integration
//   - Container lifecycle management (creation, startup, cleanup)
package deploy

import (
	"context"
	"time"

	eve "eve.evalgo.org/common"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// DeployZammad deploys a full Zammad helpdesk system as a set of Docker containers.
// This function pulls all required images, creates a custom Docker network,
// and starts all Zammad services (PostgreSQL, Elasticsearch, Redis, Rails server,
// scheduler, websocket, and Nginx) with appropriate configurations and dependencies.
//
// If OpenZiti integration is enabled, the Nginx container is added to the OpenZiti network.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - cli: Docker client for container and network management
//   - prefix: Prefix for container names (e.g., "myzammad" results in "myzammad-postgresql")
//   - net: Name of the Docker network to create for inter-container communication
//   - openziti: If true, adds the Nginx container to the OpenZiti network
//
// Returns:
//   - error: If any step fails (image pull, network creation, container creation, or startup)
//
// Container Services:
//   - PostgreSQL: Database backend for Zammad
//   - Elasticsearch: Search engine for Zammad
//   - Redis: Caching and session store
//   - Rails Server: Main application server
//   - Scheduler: Background job processor
//   - Websocket: Real-time communication server
//   - Nginx: Web server and reverse proxy
//
// Volumes:
//   - postgresql-data: Persistent storage for PostgreSQL
//   - redis-data: Persistent storage for Redis
//   - elasticsearch-data: Persistent storage for Elasticsearch
//   - zammad-storage: Persistent storage for Zammad application data
//   - zammad-backup: Backup storage for Zammad
func DeployZammad(ctx context.Context, cli *client.Client, prefix, net string, openziti bool) error {
	// Pull all required Docker images
	eve.ImagePull(ctx, cli, "postgres:17.5-alpine", image.PullOptions{})
	eve.ImagePull(ctx, cli, "bitnami/elasticsearch:8.18.0", image.PullOptions{})
	eve.ImagePull(ctx, cli, "ghcr.io/zammad/zammad:6.5.0-101", image.PullOptions{})
	eve.ImagePull(ctx, cli, "redis:7.4.4-alpine", image.PullOptions{})

	// Set container names with prefix
	zammad_postgres := prefix + "-postgresql"
	zammad_elasticsearch := prefix + "-elasticsearch"
	zammad_redis := prefix + "-redis"
	zammad_railsserver := prefix + "-railsserver"
	zammad_scheduler := prefix + "-scheduler"
	zammad_websocket := prefix + "-websocket"
	zammad_nginx := prefix + "-nginx"

	// Create Docker network for inter-container communication
	_, err := cli.NetworkCreate(ctx, net, network.CreateOptions{})
	if err != nil {
		eve.Logger.Info(err)
		// Continue even if network already exists
	}

	// Define volume mappings
	volumes := map[string]string{
		"postgresql-data":    "/var/lib/postgresql/data",
		"redis-data":         "/data",
		"elasticsearch-data": "/bitnami/elasticsearch/data",
		"zammad-storage":     "/opt/zammad/storage",
		"zammad-backup":      "/var/tmp/zammad",
	}

	// Start PostgreSQL container
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

	// Wait for PostgreSQL to initialize
	eve.Logger.Info("wait for postgres to boot...")
	time.Sleep(10 * time.Second)

	// Start Elasticsearch container
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

	// Wait for Elasticsearch to initialize
	eve.Logger.Info("wait for elasticsearch to boot...")
	time.Sleep(30 * time.Second)

	// Start Redis container
	eve.Logger.Info(eve.CreateAndStartContainer(ctx, cli, container.Config{
		Image: "redis:7.4.4-alpine",
	}, container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: "always"},
	}, zammad_redis, net))

	// Environment variables for Zammad services
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

	// Run Zammad initialization
	eve.Logger.Info(eve.CreateAndStartContainer(ctx, cli, container.Config{
		Image: "ghcr.io/zammad/zammad:6.5.0-101",
		Env:   railsEnv,
		Cmd:   []string{"zammad-init"},
	}, container.HostConfig{
		AutoRemove:    true,
		Mounts:        []mount.Mount{{Source: "zammad-storage", Target: volumes["zammad-storage"], Type: mount.TypeVolume}},
		RestartPolicy: container.RestartPolicy{},
	}, "tmp-zammad-init", net))

	// Start Rails server
	eve.Logger.Info(eve.CreateAndStartContainer(ctx, cli, container.Config{
		Image: "ghcr.io/zammad/zammad:6.5.0-101",
		Env:   railsEnv,
		Cmd:   []string{"zammad-railsserver"},
	}, container.HostConfig{
		Mounts:        []mount.Mount{{Source: "zammad-storage", Target: volumes["zammad-storage"], Type: mount.TypeVolume}},
		RestartPolicy: container.RestartPolicy{Name: "always"},
	}, zammad_railsserver, net))

	// Start scheduler
	eve.CreateAndStartContainer(ctx, cli, container.Config{
		Image: "ghcr.io/zammad/zammad:6.5.0-101",
		Env:   railsEnv,
		Cmd:   []string{"zammad-scheduler"},
	}, container.HostConfig{
		Mounts:        []mount.Mount{{Source: "zammad-storage", Target: volumes["zammad-storage"], Type: mount.TypeVolume}},
		RestartPolicy: container.RestartPolicy{Name: "always"},
	}, zammad_scheduler, net)

	// Start Websocket server
	eve.CreateAndStartContainer(ctx, cli, container.Config{
		Image: "ghcr.io/zammad/zammad:6.5.0-101",
		Env:   railsEnv,
		Cmd:   []string{"zammad-websocket"},
	}, container.HostConfig{
		Mounts:        []mount.Mount{{Source: "zammad-storage", Target: volumes["zammad-storage"], Type: mount.TypeVolume}},
		RestartPolicy: container.RestartPolicy{Name: "always"},
	}, zammad_websocket, net)

	// Start Nginx server
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

	// Add Nginx container to OpenZiti network if requested
	if openziti {
		return eve.AddContainerToNetwork(ctx, cli, zammad_nginx, "openziti")
	}

	return nil
}
