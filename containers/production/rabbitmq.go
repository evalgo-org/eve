package production

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"

	"eve.evalgo.org/common"
)

// RabbitMQProductionConfig holds configuration for production RabbitMQ deployment.
type RabbitMQProductionConfig struct {
	// ContainerName is the name for the RabbitMQ container
	ContainerName string
	// Image is the Docker image to use (default: "rabbitmq:4.1.0-management")
	Image string
	// AMQPPort is the host port to expose AMQP protocol (default: 5672)
	AMQPPort string
	// ManagementPort is the host port to expose Management UI (default: 15672)
	ManagementPort string
	// Username is the RabbitMQ admin username
	Username string
	// Password is the RabbitMQ admin password
	Password string
	// DataVolume is the volume name for RabbitMQ data persistence
	DataVolume string
	// Production holds common production configuration
	Production ProductionConfig
}

// DefaultRabbitMQProductionConfig returns the default RabbitMQ production configuration.
func DefaultRabbitMQProductionConfig() RabbitMQProductionConfig {
	return RabbitMQProductionConfig{
		ContainerName:  "rabbitmq",
		Image:          "rabbitmq:4.1.0-management",
		AMQPPort:       "5672",
		ManagementPort: "15672",
		Username:       "guest",
		Password:       "changeme",
		DataVolume:     "rabbitmq-data",
		Production: ProductionConfig{
			NetworkName:   "app-network",
			CreateNetwork: true,
			VolumeName:    "rabbitmq-data",
			CreateVolume:  true,
		},
	}
}

// DeployRabbitMQ deploys a production-ready RabbitMQ container.
//
// RabbitMQ is a message broker that implements AMQP protocol. This function deploys a
// RabbitMQ container suitable for production use with persistent data storage.
//
// Container Features:
//   - Named container for consistent identification
//   - Persistent volume for message and configuration data
//   - Custom network connectivity
//   - Fixed port mapping for stable access
//   - Restart policy for high availability
//   - Health checks for monitoring
//   - Management plugin enabled for web UI
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - config: RabbitMQ production configuration
//
// Returns:
//   - string: Container ID of the deployed RabbitMQ container
//   - error: Deployment errors
//
// Example Usage:
//
//	ctx, cli, err := common.CtxCli("unix:///var/run/docker.sock")
//	if err != nil {
//	    return fmt.Errorf("failed to create Docker client: %w", err)
//	}
//	defer cli.Close()
//
//	config := DefaultRabbitMQProductionConfig()
//	config.Username = "admin"
//	config.Password = "secure-password-here"
//
//	containerID, err := DeployRabbitMQ(ctx, cli, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("RabbitMQ deployed with ID: %s", containerID)
//	log.Printf("AMQP: amqp://%s:****@localhost:%s/",
//	    config.Username, config.AMQPPort)
//	log.Printf("Management UI: http://localhost:%s", config.ManagementPort)
//
// Connection URLs:
//
//	AMQP Protocol:
//	amqp://{username}:{password}@localhost:{amqp_port}/
//	amqp://{username}:{password}@{container_name}:{amqp_port}/ (from other containers)
//
//	Management UI:
//	http://localhost:{management_port}
//	Default credentials: {username}/{password}
//
// Data Persistence:
//
//	RabbitMQ data is stored in a Docker volume ({config.DataVolume}).
//	This ensures messages and configuration persist across container restarts.
//
//	Volume mount points:
//	- /var/lib/rabbitmq - Message data, configuration, and mnesia database
//
// Management Plugin:
//
//	The -management image includes the management plugin which provides:
//	- Web-based UI for monitoring and management
//	- HTTP API for automation
//	- Queue, exchange, and binding management
//	- User and vhost administration
//	- Performance metrics and monitoring
//
//	Access: http://localhost:{config.ManagementPort}
//	Credentials: {config.Username}/{config.Password}
//
// Network Configuration:
//
//	The container joins the specified Docker network, enabling
//	communication with other containers on the same network.
//
//	Other containers can connect using the container name:
//	amqp://{username}:{password}@{container_name}:5672/
//
// Security:
//
//	IMPORTANT: Always set a strong password in production!
//	config.Password = "strong-random-password-here"
//
//	Default user "guest" can only connect from localhost in RabbitMQ.
//	For remote connections, create additional users via Management UI or API.
//
// Restart Policy:
//
//	The container is configured with restart policy "unless-stopped",
//	ensuring it automatically restarts if it crashes.
//
// Health Monitoring:
//
//	Health check runs rabbitmq-diagnostics ping every 30 seconds:
//	- Timeout: 10 seconds
//	- Retries: 3 consecutive failures before unhealthy
//
// Performance Tuning:
//
//	For production workloads, consider:
//	- Memory limits (RabbitMQ uses ~400MB base + message storage)
//	- Disk space for persistent messages
//	- Queue mirroring for high availability
//	- Clustering for scalability
//	- Message TTL and dead letter exchanges
//
// Message Patterns:
//
//	RabbitMQ supports various messaging patterns:
//	- Work queues (task distribution)
//	- Publish/Subscribe (fanout)
//	- Routing (direct exchange)
//	- Topics (pattern matching)
//	- RPC (request/reply)
//
// Monitoring:
//
//	Monitor these metrics via Management UI or API:
//	- Queue length and message rates
//	- Connection and channel count
//	- Memory usage
//	- Disk space (for persistent messages)
//	- Consumer utilization
//
// Backup Strategy:
//
//	Important: Implement regular backups!
//	- Export definitions (queues, exchanges, bindings)
//	- Backup message data (if using persistent queues)
//	- Volume snapshots for disaster recovery
//
// Error Handling:
//
//	Returns error if:
//	- Container with same name already exists
//	- Network or volume creation fails
//	- Docker API errors occur
//	- Invalid configuration provided
func DeployRabbitMQ(ctx context.Context, cli common.DockerClient, config RabbitMQProductionConfig) (string, error) {
	// Check if container already exists
	exists, err := common.ContainerExistsWithClient(ctx, cli, config.ContainerName)
	if err != nil {
		return "", fmt.Errorf("failed to check container existence: %w", err)
	}
	if exists {
		return "", fmt.Errorf("container %s already exists", config.ContainerName)
	}

	// Prepare production environment (network and volume)
	if err := PrepareProductionEnvironment(ctx, cli, config.Production); err != nil {
		return "", fmt.Errorf("failed to prepare environment: %w", err)
	}

	// Pull RabbitMQ image
	if err := common.ImagePullWithClient(ctx, cli, config.Image, &common.ImagePullOptions{Silent: true}); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Configure port bindings
	amqpPortBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: config.AMQPPort,
	}
	managementPortBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: config.ManagementPort,
	}
	portMap := nat.PortMap{
		"5672/tcp":  []nat.PortBinding{amqpPortBinding},
		"15672/tcp": []nat.PortBinding{managementPortBinding},
	}

	// Configure volume mounts
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: config.DataVolume,
			Target: "/var/lib/rabbitmq",
		},
	}

	// Container configuration
	containerConfig := container.Config{
		Image: config.Image,
		Env: []string{
			fmt.Sprintf("RABBITMQ_DEFAULT_USER=%s", config.Username),
			fmt.Sprintf("RABBITMQ_DEFAULT_PASS=%s", config.Password),
		},
		ExposedPorts: nat.PortSet{
			"5672/tcp":  struct{}{},
			"15672/tcp": struct{}{},
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD", "rabbitmq-diagnostics", "ping"},
			Interval: 30000000000, // 30 seconds
			Timeout:  10000000000, // 10 seconds
			Retries:  3,
		},
	}

	// Host configuration
	hostConfig := container.HostConfig{
		PortBindings: portMap,
		Mounts:       mounts,
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
	}

	// Deploy container
	err = common.CreateAndStartContainerWithClient(ctx, cli, containerConfig, hostConfig, config.ContainerName, config.Production.NetworkName)
	if err != nil {
		return "", fmt.Errorf("failed to create and start container: %w", err)
	}

	// Get container ID
	containers, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	for _, cont := range containers {
		for _, name := range cont.Names {
			if name == "/"+config.ContainerName {
				return cont.ID, nil
			}
		}
	}

	return "", fmt.Errorf("container created but ID not found")
}

// StopRabbitMQ stops a running RabbitMQ container.
//
// Performs graceful shutdown to ensure messages are not lost.
// RabbitMQ will close connections and persist messages before stopping.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the RabbitMQ container to stop
//
// Returns:
//   - error: Stop errors
//
// Example:
//
//	err := StopRabbitMQ(ctx, cli, "rabbitmq")
//	if err != nil {
//	    log.Printf("Failed to stop RabbitMQ: %v", err)
//	}
func StopRabbitMQ(ctx context.Context, cli common.DockerClient, containerName string) error {
	timeout := 30 // 30 seconds for graceful shutdown
	return cli.ContainerStop(ctx, containerName, container.StopOptions{Timeout: &timeout})
}

// RemoveRabbitMQ removes a RabbitMQ container and optionally its volume.
//
// WARNING: Removing the volume will DELETE ALL MESSAGES and configuration permanently!
// Always backup data before removing volumes.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the RabbitMQ container to remove
//   - removeVolume: Whether to also remove the data volume (DANGEROUS!)
//   - volumeName: Name of the data volume (required if removeVolume is true)
//
// Returns:
//   - error: Removal errors
//
// Example:
//
//	// Remove container but keep data (safe)
//	err := RemoveRabbitMQ(ctx, cli, "rabbitmq", false, "")
//
//	// Remove container and data (DANGEROUS - backup first!)
//	err := RemoveRabbitMQ(ctx, cli, "rabbitmq", true, "rabbitmq-data")
func RemoveRabbitMQ(ctx context.Context, cli common.DockerClient, containerName string, removeVolume bool, volumeName string) error {
	// Remove container
	if err := cli.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	// Remove volume if requested (DANGEROUS - data loss!)
	if removeVolume && volumeName != "" {
		if err := cli.VolumeRemove(ctx, volumeName, true); err != nil {
			return fmt.Errorf("failed to remove volume: %w", err)
		}
	}

	return nil
}

// GetRabbitMQAMQPURL returns the AMQP connection URL for the deployed RabbitMQ container.
//
// This is a convenience function that formats the AMQP URL for RabbitMQ clients.
//
// Parameters:
//   - config: RabbitMQ production configuration
//   - vhost: Virtual host (optional, use "/" for default)
//
// Returns:
//   - string: RabbitMQ AMQP connection URL
//
// Example:
//
//	config := DefaultRabbitMQProductionConfig()
//	config.Password = "secure-password"
//
//	// Default vhost
//	amqpURL := GetRabbitMQAMQPURL(config, "/")
//	// amqp://guest:secure-password@localhost:5672/
//
//	// Custom vhost
//	amqpURL := GetRabbitMQAMQPURL(config, "production")
//	// amqp://guest:secure-password@localhost:5672/production
func GetRabbitMQAMQPURL(config RabbitMQProductionConfig, vhost string) string {
	if vhost == "" {
		vhost = "/"
	}
	// Ensure vhost starts with / if it doesn't already
	if !strings.HasPrefix(vhost, "/") {
		vhost = "/" + vhost
	}
	return fmt.Sprintf("amqp://%s:%s@localhost:%s%s",
		config.Username,
		config.Password,
		config.AMQPPort,
		vhost)
}

// GetRabbitMQManagementURL returns the Management UI URL for the deployed RabbitMQ container.
//
// This is a convenience function that formats the Management UI URL.
//
// Parameters:
//   - config: RabbitMQ production configuration
//
// Returns:
//   - string: RabbitMQ Management UI URL
//
// Example:
//
//	config := DefaultRabbitMQProductionConfig()
//	managementURL := GetRabbitMQManagementURL(config)
//	// http://localhost:15672
func GetRabbitMQManagementURL(config RabbitMQProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s", config.ManagementPort)
}
