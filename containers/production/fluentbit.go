package production

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"

	"eve.evalgo.org/common"
)

// FluentBitProductionConfig holds configuration for production Fluent Bit deployment.
type FluentBitProductionConfig struct {
	// ContainerName is the name for the Fluent Bit container
	ContainerName string
	// Image is the Docker image to use (default: "fluent/fluent-bit:4.0.13-amd64")
	Image string
	// Port is the host port to expose Fluent Bit HTTP monitoring API (default: 2020)
	Port string
	// ForwardPort is the host port to expose Forward input protocol (default: 24224)
	ForwardPort string
	// ConfigVolume is the volume name for Fluent Bit configuration files
	ConfigVolume string
	// DataVolume is the volume name for Fluent Bit data persistence (buffers, logs)
	DataVolume string
	// Production holds common production configuration
	Production ProductionConfig
}

// DefaultFluentBitProductionConfig returns the default Fluent Bit production configuration.
func DefaultFluentBitProductionConfig() FluentBitProductionConfig {
	return FluentBitProductionConfig{
		ContainerName: "fluentbit",
		Image:         "fluent/fluent-bit:4.0.13-amd64",
		Port:          "2020",
		ForwardPort:   "24224",
		ConfigVolume:  "fluentbit-config",
		DataVolume:    "fluentbit-data",
		Production: ProductionConfig{
			NetworkName:   "app-network",
			CreateNetwork: true,
			VolumeName:    "fluentbit-data",
			CreateVolume:  true,
		},
	}
}

// DeployFluentBit deploys a production-ready Fluent Bit container.
//
// Fluent Bit is a lightweight and high-performance log processor and forwarder that allows
// you to collect logs from different sources, enrich them with filters, and send them to
// multiple destinations. This function deploys a Fluent Bit container suitable for production
// use with persistent data storage and configuration.
//
// Container Features:
//   - Named container for consistent identification
//   - Persistent volume for configuration files
//   - Persistent volume for buffering and data storage
//   - Custom network connectivity
//   - Fixed port mapping for stable access
//   - Restart policy for high availability
//   - Health checks for monitoring
//   - HTTP monitoring API enabled
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - config: Fluent Bit production configuration
//
// Returns:
//   - string: Container ID of the deployed Fluent Bit container
//   - error: Deployment errors
//
// Example Usage:
//
//	ctx, cli := common.CtxCli("unix:///var/run/docker.sock")
//	defer cli.Close()
//
//	config := DefaultFluentBitProductionConfig()
//	containerID, err := DeployFluentBit(ctx, cli, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("Fluent Bit deployed with ID: %s", containerID)
//	log.Printf("Fluent Bit monitoring API: http://localhost:%s", config.Port)
//
// Connection URLs:
//
//	HTTP Monitoring API:
//	http://localhost:{port}
//	http://{container_name}:{port} (from other containers)
//
//	Metrics endpoint:
//	http://localhost:{port}/api/v1/metrics/prometheus
//
//	Health endpoint:
//	http://localhost:{port}/api/v1/health
//
//	Forward input (from Fluentd/Fluent Bit clients):
//	tcp://{container_name}:{forward_port}
//
// Data Persistence:
//
//	Fluent Bit data is stored in Docker volumes:
//	- Config volume ({config.ConfigVolume}) - Configuration files
//	- Data volume ({config.DataVolume}) - Buffers, logs, state
//
//	This ensures configuration and buffered data persist across container restarts.
//
//	Volume mount points:
//	- /fluent-bit/etc - Configuration files (fluent-bit.conf, parsers.conf)
//	- /fluent-bit/log - Buffer storage and logs
//
// Network Configuration:
//
//	The container joins the specified Docker network, enabling
//	communication with other containers on the same network.
//
//	Applications can forward logs to Fluent Bit using:
//	tcp://{container_name}:24224 (Forward protocol)
//	http://{container_name}:2020 (HTTP input, if configured)
//
// Architecture:
//
//	Fluent Bit follows a plugin-based architecture:
//
//	1. Input Plugins:
//	   - Collect logs from various sources
//	   - tail, syslog, tcp, forward, http, docker, kubernetes
//	   - Each input generates tagged records
//
//	2. Parser Plugins:
//	   - Parse unstructured log data into structured records
//	   - json, regex, ltsv, logfmt, docker, syslog
//	   - Extract fields and add structure
//
//	3. Filter Plugins:
//	   - Transform and enrich log data
//	   - grep, parser, kubernetes, modify, nest, lua
//	   - Add, remove, or modify fields
//	   - Route by pattern matching
//
//	4. Buffer:
//	   - Memory or filesystem buffering
//	   - Reliability and backpressure handling
//	   - Retry mechanisms
//
//	5. Output Plugins:
//	   - Send logs to destinations
//	   - stdout, forward, http, elasticsearch, kafka, s3, loki
//	   - Multiple outputs per pipeline
//
// Configuration Format:
//
//	Fluent Bit uses a simple INI-style configuration:
//
//	[SERVICE]
//	    Flush        5
//	    Daemon       Off
//	    Log_Level    info
//	    HTTP_Server  On
//	    HTTP_Listen  0.0.0.0
//	    HTTP_Port    2020
//	    storage.path /fluent-bit/log/
//
//	[INPUT]
//	    Name   tail
//	    Path   /var/log/app/*.log
//	    Tag    app.logs
//	    Parser json
//
//	[INPUT]
//	    Name   forward
//	    Listen 0.0.0.0
//	    Port   24224
//
//	[FILTER]
//	    Name   grep
//	    Match  app.logs
//	    Regex  level (error|warning|critical)
//
//	[FILTER]
//	    Name   kubernetes
//	    Match  kube.*
//	    Merge_Log On
//	    K8S-Logging.Parser On
//
//	[FILTER]
//	    Name   record_modifier
//	    Match  *
//	    Record hostname ${HOSTNAME}
//	    Record cluster production
//
//	[OUTPUT]
//	    Name   es
//	    Match  *
//	    Host   elasticsearch
//	    Port   9200
//	    Index  logs
//	    Type   _doc
//
//	[OUTPUT]
//	    Name   s3
//	    Match  *
//	    bucket my-logs
//	    region us-east-1
//	    total_file_size 50M
//	    upload_timeout 10m
//
// Input Plugins:
//
//	Collect logs from various sources:
//	- tail - Read from text files (similar to tail -f)
//	- systemd - Read from systemd journal
//	- syslog - Syslog protocol server (UDP/TCP)
//	- tcp - TCP protocol server
//	- forward - Fluentd forward protocol (compatible)
//	- http - HTTP endpoints (POST logs)
//	- docker - Docker container logs
//	- kubernetes - Kubernetes pod logs (tail + metadata)
//	- mqtt - MQTT protocol
//	- serial - Serial interface
//	- stdin - Standard input
//	- dummy - Generate dummy data (testing)
//	- cpu - CPU metrics
//	- mem - Memory metrics
//	- disk - Disk I/O metrics
//	- netif - Network interface metrics
//
// Parser Plugins:
//
//	Parse and structure log data:
//	- json - Parse JSON logs
//	- regex - Regular expression parsing
//	- ltsv - Labeled Tab Separated Values
//	- logfmt - Logfmt format (key=value)
//	- docker - Docker JSON format
//	- syslog - Syslog format (RFC3164, RFC5424)
//	- apache - Apache access logs
//	- nginx - Nginx access logs
//	- apache_error - Apache error logs
//	- mongodb - MongoDB logs
//	- cri - CRI (Container Runtime Interface) logs
//
// Filter Plugins:
//
//	Transform and enrich log data:
//	- grep - Filter by pattern matching (include/exclude)
//	- parser - Parse fields within records
//	- lua - Lua scripting for custom logic
//	- kubernetes - Enrich with Kubernetes metadata
//	- nest - Nest or lift fields (organize structure)
//	- modify - Modify records (add/remove/rename/copy fields)
//	- record_modifier - Advanced record modification
//	- throttle - Throttle log throughput (rate limiting)
//	- rewrite_tag - Dynamic tag routing
//	- geoip - GeoIP enrichment
//	- aws - AWS metadata enrichment
//	- multiline - Concatenate multiline logs
//	- expect - Wait for records with specific patterns
//
// Output Plugins:
//
//	Send logs to multiple destinations:
//	- stdout - Standard output (debugging)
//	- forward - Forward to Fluentd/Fluent Bit
//	- http - HTTP endpoints (POST)
//	- elasticsearch - Elasticsearch
//	- opensearch - OpenSearch
//	- kafka - Apache Kafka
//	- prometheus - Prometheus metrics exporter
//	- s3 - AWS S3 (with compression and partitioning)
//	- cloudwatch - AWS CloudWatch Logs
//	- kinesis - AWS Kinesis Data Streams
//	- firehose - AWS Kinesis Data Firehose
//	- datadog - Datadog
//	- splunk - Splunk HEC
//	- loki - Grafana Loki
//	- influxdb - InfluxDB
//	- tcp - TCP protocol
//	- syslog - Syslog protocol
//	- gelf - Graylog GELF
//	- stackdriver - Google Cloud Logging
//	- bigquery - Google BigQuery
//	- azure - Azure Log Analytics
//	- file - Local files
//	- null - Discard logs (testing)
//
// Monitoring API:
//
//	Fluent Bit provides an HTTP monitoring API on port 2020:
//
//	Health Check:
//	GET /api/v1/health
//	Returns: {"status": "ok"}
//
//	Metrics (JSON):
//	GET /api/v1/metrics
//	Returns: Fluent Bit internal metrics
//
//	Metrics (Prometheus):
//	GET /api/v1/metrics/prometheus
//	Returns: Prometheus-formatted metrics
//	  - fluentbit_input_bytes_total
//	  - fluentbit_input_records_total
//	  - fluentbit_output_bytes_total
//	  - fluentbit_output_records_total
//	  - fluentbit_output_errors_total
//	  - fluentbit_output_retries_total
//	  - fluentbit_output_retries_failed_total
//
//	Uptime:
//	GET /api/v1/uptime
//	Returns: Service uptime information
//
//	Service Info:
//	GET /
//	Returns: Fluent Bit version and build info
//
// Performance Characteristics:
//
//	Fluent Bit is designed for high performance:
//	- Written in C (low-level, efficient)
//	- Small memory footprint (450KB - 5MB typical)
//	- High throughput (tens of thousands events/sec)
//	- Low CPU usage
//	- Async I/O
//	- Zero-copy operations where possible
//	- Efficient memory management
//
//	Performance tuning:
//	- Adjust Flush interval (balance latency vs throughput)
//	- Configure buffering (memory vs filesystem)
//	- Use multiple workers for outputs
//	- Enable compression for network outputs
//	- Optimize parsers and filters
//	- Monitor backpressure
//
// Buffering and Reliability:
//
//	Fluent Bit provides buffering for reliability:
//
//	Memory Buffering (default):
//	- Fast, low latency
//	- Limited by available RAM
//	- Lost on container restart
//	- Good for non-critical logs
//
//	Filesystem Buffering:
//	- Persistent across restarts
//	- Slower than memory
//	- Disk I/O overhead
//	- Good for critical logs
//
//	Configuration:
//	[SERVICE]
//	    storage.path /fluent-bit/log/
//	    storage.sync normal
//	    storage.max_chunks_up 128
//
//	[INPUT]
//	    Name   forward
//	    storage.type filesystem
//
//	Backpressure:
//	- Fluent Bit pauses inputs when outputs are slow
//	- Prevents memory exhaustion
//	- Configurable with storage limits
//
//	Retry Mechanism:
//	- Automatic retries for failed outputs
//	- Exponential backoff
//	- Configurable retry limits
//
// Security:
//
//	For production use:
//	- Use TLS for network inputs/outputs
//	- Restrict network access (firewall rules)
//	- Use authentication for inputs (if supported)
//	- Encrypt sensitive data in configs
//	- Use secrets management (Docker secrets, Kubernetes secrets)
//	- Run as non-root user (if possible)
//	- Limit container capabilities
//	- Regular security updates
//	- Monitor for security advisories
//
// Kubernetes Integration:
//
//	Fluent Bit is Kubernetes-native:
//	- DaemonSet deployment pattern
//	- Automatic pod log collection
//	- Kubernetes metadata enrichment
//	- Service discovery
//	- CRI log parsing
//	- Multi-format support
//	- Resource limits and requests
//	- RBAC configuration
//
//	Example DaemonSet configuration:
//	- Mount /var/log/containers (pod logs)
//	- Mount /var/log/pods (pod metadata)
//	- Use Kubernetes filter for enrichment
//	- Send to centralized log storage
//
// Use Cases:
//
//	Common Fluent Bit use cases:
//	- Centralized logging (collect logs from multiple sources)
//	- Log aggregation (combine logs from distributed systems)
//	- Log forwarding (send logs to SIEM, log storage)
//	- Log parsing and structuring
//	- Log enrichment (add metadata, GeoIP, etc.)
//	- Log filtering (remove noise, select important logs)
//	- Log routing (send different logs to different destinations)
//	- Metrics collection (system metrics, custom metrics)
//	- Kubernetes logging (pod logs with metadata)
//	- IoT logging (lightweight agent for devices)
//	- Edge computing (process logs at the edge)
//	- Multi-destination logging (send to multiple backends)
//
// Restart Policy:
//
//	The container is configured with restart policy "unless-stopped",
//	ensuring it automatically restarts if it crashes.
//
// Health Monitoring:
//
//	Health check runs HTTP GET http://localhost:2020 every 30 seconds:
//	- Timeout: 10 seconds
//	- Retries: 3 consecutive failures before unhealthy
//
// Performance Tuning:
//
//	For production workloads, consider:
//	- Adjust Flush interval (default: 5 seconds)
//	- Configure buffer limits (prevent memory exhaustion)
//	- Use filesystem buffering for critical logs
//	- Enable compression for network outputs
//	- Optimize parser regex patterns
//	- Use multiline parser for stack traces
//	- Configure retry limits and timeouts
//	- Monitor backpressure metrics
//	- Adjust log levels (info, warning, error)
//	- Use grep filters to reduce volume
//
// Configuration Management:
//
//	Store configuration in version control:
//	- fluent-bit.conf - Main configuration
//	- parsers.conf - Custom parser definitions
//	- plugins.conf - Plugin configurations
//
//	Mount configuration as volume:
//	- Use ConfigMaps in Kubernetes
//	- Use Docker volumes for standalone
//	- Test configuration changes before deployment
//	- Use configuration validation tools
//
// Monitoring:
//
//	Monitor these metrics via Prometheus endpoint:
//	- fluentbit_input_bytes_total - Bytes ingested
//	- fluentbit_input_records_total - Records ingested
//	- fluentbit_output_bytes_total - Bytes sent
//	- fluentbit_output_records_total - Records sent
//	- fluentbit_output_errors_total - Output errors
//	- fluentbit_output_retries_total - Retry attempts
//	- fluentbit_output_retries_failed_total - Failed retries
//
//	Set up alerts for:
//	- High error rates
//	- Failed retries
//	- Backpressure events
//	- High memory usage
//	- Slow outputs
//
// Troubleshooting:
//
//	Common issues and solutions:
//
//	1. Logs not appearing:
//	   - Check input configuration
//	   - Verify file paths exist
//	   - Check permissions
//	   - Review filter rules (grep may be excluding)
//	   - Check output configuration
//
//	2. High memory usage:
//	   - Enable filesystem buffering
//	   - Reduce flush interval
//	   - Configure buffer limits
//	   - Check for slow outputs (backpressure)
//	   - Review filter complexity
//
//	3. Output errors:
//	   - Check network connectivity
//	   - Verify destination is reachable
//	   - Check authentication credentials
//	   - Review retry configuration
//	   - Check destination capacity
//
//	4. Performance issues:
//	   - Optimize parser patterns
//	   - Reduce filter complexity
//	   - Enable compression
//	   - Use multiple workers
//	   - Check system resources
//
// Backup Strategy:
//
//	Important: Implement regular backups!
//	- Backup configuration files
//	- Backup custom parsers
//	- Store configurations in version control
//	- Document configuration changes
//	- Test restore procedures
//	- Use infrastructure as code
//
// Comparison to Fluentd:
//
//	Fluent Bit vs Fluentd:
//	- Fluent Bit: Lightweight, high performance, low memory
//	- Fluentd: Feature-rich, plugin ecosystem, Ruby-based
//	- Fluent Bit: Good for edge, IoT, containers
//	- Fluentd: Good for complex processing, aggregation
//	- Compatible: Can forward between them
//	- Common pattern: Fluent Bit (forwarder) -> Fluentd (aggregator)
//
// Error Handling:
//
//	Returns error if:
//	- Container with same name already exists
//	- Network or volume creation fails
//	- Docker API errors occur
//	- Invalid configuration provided
func DeployFluentBit(ctx context.Context, cli common.DockerClient, config FluentBitProductionConfig) (string, error) {
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

	// Ensure config volume exists
	if config.ConfigVolume != "" {
		if err := EnsureVolume(ctx, cli, config.ConfigVolume); err != nil {
			return "", fmt.Errorf("failed to ensure config volume: %w", err)
		}
	}

	// Pull Fluent Bit image
	if err := common.ImagePullWithClient(ctx, cli, config.Image, &common.ImagePullOptions{Silent: true}); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Configure port bindings
	portMap := nat.PortMap{
		"2020/tcp": []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: config.Port,
			},
		},
		"24224/tcp": []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: config.ForwardPort,
			},
		},
	}

	// Configure volume mounts
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: config.ConfigVolume,
			Target: "/fluent-bit/etc",
		},
		{
			Type:   mount.TypeVolume,
			Source: config.DataVolume,
			Target: "/fluent-bit/log",
		},
	}

	// Container configuration
	containerConfig := container.Config{
		Image: config.Image,
		ExposedPorts: nat.PortSet{
			"2020/tcp":  struct{}{},
			"24224/tcp": struct{}{},
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD-SHELL", "wget --spider --quiet http://localhost:2020 || exit 1"},
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

// StopFluentBit stops a running Fluent Bit container.
//
// Performs graceful shutdown to ensure buffered logs are flushed.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the Fluent Bit container to stop
//
// Returns:
//   - error: Stop errors
//
// Example:
//
//	err := StopFluentBit(ctx, cli, "fluentbit")
//	if err != nil {
//	    log.Printf("Failed to stop Fluent Bit: %v", err)
//	}
func StopFluentBit(ctx context.Context, cli common.DockerClient, containerName string) error {
	timeout := 30 // 30 seconds for graceful shutdown and buffer flush
	return cli.ContainerStop(ctx, containerName, container.StopOptions{Timeout: &timeout})
}

// RemoveFluentBit removes a Fluent Bit container and optionally its volumes.
//
// WARNING: Removing volumes will DELETE configuration and buffered logs permanently!
// Always backup data before removing volumes.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the Fluent Bit container to remove
//   - removeVolumes: Whether to also remove the volumes (DANGEROUS!)
//   - configVolumeName: Name of the config volume (required if removeVolumes is true)
//   - dataVolumeName: Name of the data volume (required if removeVolumes is true)
//
// Returns:
//   - error: Removal errors
//
// Example:
//
//	// Remove container but keep volumes (safe)
//	err := RemoveFluentBit(ctx, cli, "fluentbit", false, "", "")
//
//	// Remove container and volumes (DANGEROUS - backup first!)
//	err := RemoveFluentBit(ctx, cli, "fluentbit", true, "fluentbit-config", "fluentbit-data")
func RemoveFluentBit(ctx context.Context, cli common.DockerClient, containerName string, removeVolumes bool, configVolumeName, dataVolumeName string) error {
	// Remove container
	if err := cli.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	// Remove volumes if requested (DANGEROUS - data loss!)
	if removeVolumes {
		if configVolumeName != "" {
			if err := cli.VolumeRemove(ctx, configVolumeName, true); err != nil {
				return fmt.Errorf("failed to remove config volume: %w", err)
			}
		}
		if dataVolumeName != "" {
			if err := cli.VolumeRemove(ctx, dataVolumeName, true); err != nil {
				return fmt.Errorf("failed to remove data volume: %w", err)
			}
		}
	}

	return nil
}

// GetFluentBitURL returns the Fluent Bit HTTP monitoring endpoint URL for the deployed container.
//
// This is a convenience function that formats the URL for the monitoring API.
//
// Parameters:
//   - config: Fluent Bit production configuration
//
// Returns:
//   - string: Fluent Bit HTTP endpoint URL
//
// Example:
//
//	config := DefaultFluentBitProductionConfig()
//	fluentbitURL := GetFluentBitURL(config)
//	// http://localhost:2020
func GetFluentBitURL(config FluentBitProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s", config.Port)
}

// GetFluentBitHealthURL returns the Fluent Bit health check URL for the deployed container.
//
// This is a convenience function for monitoring and health checks.
//
// Parameters:
//   - config: Fluent Bit production configuration
//
// Returns:
//   - string: Fluent Bit health check URL
//
// Example:
//
//	config := DefaultFluentBitProductionConfig()
//	healthURL := GetFluentBitHealthURL(config)
//	// http://localhost:2020/api/v1/health
func GetFluentBitHealthURL(config FluentBitProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s/api/v1/health", config.Port)
}

// GetFluentBitMetricsURL returns the Fluent Bit Prometheus metrics URL for the deployed container.
//
// This is a convenience function for Prometheus scraping and monitoring.
//
// Parameters:
//   - config: Fluent Bit production configuration
//
// Returns:
//   - string: Fluent Bit Prometheus metrics URL
//
// Example:
//
//	config := DefaultFluentBitProductionConfig()
//	metricsURL := GetFluentBitMetricsURL(config)
//	// http://localhost:2020/api/v1/metrics/prometheus
func GetFluentBitMetricsURL(config FluentBitProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s/api/v1/metrics/prometheus", config.Port)
}
