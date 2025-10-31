package production

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"

	"eve.evalgo.org/common"
)

// OTelCollectorProductionConfig holds configuration for production OpenTelemetry Collector deployment.
type OTelCollectorProductionConfig struct {
	// ContainerName is the name for the OTel Collector container
	ContainerName string
	// Image is the Docker image to use (default: "otel/opentelemetry-collector:nightly")
	Image string
	// HTTPPort is the host port to expose OTLP HTTP receiver (default: 4318)
	HTTPPort string
	// GRPCPort is the host port to expose OTLP gRPC receiver (default: 4317)
	GRPCPort string
	// HealthPort is the host port to expose health check extension (default: 13133)
	HealthPort string
	// ConfigVolume is the volume name for collector configuration
	ConfigVolume string
	// DataVolume is the volume name for collector data persistence
	DataVolume string
	// Production holds common production configuration
	Production ProductionConfig
}

// DefaultOTelCollectorProductionConfig returns the default OTel Collector production configuration.
func DefaultOTelCollectorProductionConfig() OTelCollectorProductionConfig {
	return OTelCollectorProductionConfig{
		ContainerName: "otelcollector",
		Image:         "otel/opentelemetry-collector:nightly",
		HTTPPort:      "4318",
		GRPCPort:      "4317",
		HealthPort:    "13133",
		ConfigVolume:  "otelcollector-config",
		DataVolume:    "otelcollector-data",
		Production: ProductionConfig{
			NetworkName:   "app-network",
			CreateNetwork: true,
			VolumeName:    "otelcollector-data",
			CreateVolume:  true,
		},
	}
}

// DeployOTelCollector deploys a production-ready OpenTelemetry Collector container.
//
// OpenTelemetry Collector is a vendor-agnostic agent for receiving, processing, and exporting
// telemetry data (traces, metrics, and logs). This function deploys an OTel Collector container
// suitable for production use with persistent configuration and data storage.
//
// Container Features:
//   - Named container for consistent identification
//   - Persistent volumes for configuration and data
//   - Custom network connectivity
//   - Fixed port mappings for stable access
//   - Restart policy for high availability
//   - Health checks for monitoring
//   - Support for all three signals: traces, metrics, logs
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - config: OTel Collector production configuration
//
// Returns:
//   - string: Container ID of the deployed OTel Collector container
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
//	config := DefaultOTelCollectorProductionConfig()
//	containerID, err := DeployOTelCollector(ctx, cli, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("OTel Collector deployed with ID: %s", containerID)
//	log.Printf("OTLP HTTP endpoint: http://localhost:%s", config.HTTPPort)
//	log.Printf("OTLP gRPC endpoint: localhost:%s", config.GRPCPort)
//
// Connection URLs:
//
//	OTLP HTTP receiver:
//	http://localhost:{httpPort}
//	http://{container_name}:{httpPort} (from other containers)
//
//	OTLP gRPC receiver:
//	localhost:{grpcPort}
//	{container_name}:{grpcPort} (from other containers)
//
//	Health check:
//	http://localhost:{healthPort}/
//
// Data Persistence:
//
//	OTel Collector data is stored in Docker volumes:
//	- Configuration: {config.ConfigVolume} mounted at /etc/otelcol
//	- Data: {config.DataVolume} mounted at /var/lib/otelcol
//
//	This ensures configuration and state persist across container restarts.
//
//	Volume mount points:
//	- /etc/otelcol - Collector configuration files
//	- /var/lib/otelcol - Collector data and state
//
// Network Configuration:
//
//	The container joins the specified Docker network, enabling
//	communication with other containers on the same network.
//
//	Applications can send telemetry using:
//	http://{container_name}:4318/v1/traces (HTTP)
//	{container_name}:4317 (gRPC)
//
//	Grafana and other backends can receive exported telemetry.
//
// OpenTelemetry:
//
//	OpenTelemetry is a collection of tools, APIs, and SDKs for:
//	- Instrumenting applications to generate telemetry
//	- Collecting telemetry data (traces, metrics, logs)
//	- Processing and transforming telemetry
//	- Exporting telemetry to backends
//
//	Key concepts:
//	- Traces: Distributed traces across services
//	- Metrics: Time-series measurements
//	- Logs: Structured log data
//	- Context propagation: Correlation across services
//	- Semantic conventions: Standard attribute names
//
// OTLP Protocol:
//
//	OpenTelemetry Protocol (OTLP) is the native wire protocol:
//
//	OTLP/HTTP:
//	- Uses HTTP/1.1 or HTTP/2
//	- Content-Type: application/json or application/x-protobuf
//	- Endpoints: /v1/traces, /v1/metrics, /v1/logs
//	- Default port: 4318
//
//	OTLP/gRPC:
//	- Uses HTTP/2 and gRPC
//	- Protocol Buffers serialization
//	- Services: TraceService, MetricsService, LogsService
//	- Default port: 4317
//	- Better performance than HTTP for high-throughput scenarios
//
// Architecture:
//
//	The collector follows a pipeline architecture:
//
//	┌──────────┐   ┌────────────┐   ┌──────────┐
//	│Receivers │ → │ Processors │ → │Exporters │
//	└──────────┘   └────────────┘   └──────────┘
//
//	1. Receivers: Ingest telemetry from sources
//	2. Processors: Transform and process telemetry
//	3. Exporters: Send telemetry to backends
//
//	All three components operate within signal pipelines:
//	- Traces pipeline
//	- Metrics pipeline
//	- Logs pipeline
//
// Receivers:
//
//	Built-in receivers for various protocols:
//
//	OTLP Receiver:
//	- Native OpenTelemetry protocol
//	- Supports HTTP and gRPC
//	- Handles traces, metrics, and logs
//	- Default receiver for OTel SDK
//
//	Jaeger Receiver:
//	- Accepts Jaeger format traces
//	- Multiple protocols (Thrift, gRPC, Protobuf)
//	- Migration path from Jaeger
//
//	Zipkin Receiver:
//	- Accepts Zipkin format traces
//	- JSON and Protobuf formats
//	- Migration path from Zipkin
//
//	Prometheus Receiver:
//	- Scrapes Prometheus metrics
//	- Compatible with Prometheus exporters
//	- Service discovery support
//
//	Host Metrics Receiver:
//	- Collects system metrics (CPU, memory, disk, network)
//	- Cross-platform support
//	- Process metrics
//
//	Kubernetes Receiver:
//	- Collects Kubernetes cluster metrics
//	- Pod and node metrics
//	- Event collection
//
//	File Receiver:
//	- Reads logs from files
//	- Tail file changes
//	- Log rotation support
//
//	Syslog Receiver:
//	- Receives syslog messages
//	- TCP and UDP protocols
//	- RFC 3164 and RFC 5424 formats
//
// Processors:
//
//	Transform and filter telemetry data:
//
//	Batch Processor:
//	- Batches telemetry for efficiency
//	- Reduces network overhead
//	- Configurable batch size and timeout
//	- Recommended for all pipelines
//
//	Memory Limiter Processor:
//	- Prevents out-of-memory errors
//	- Monitors memory usage
//	- Back-pressure mechanism
//	- Essential for production
//
//	Resource Processor:
//	- Add, modify, or delete resource attributes
//	- Enrich telemetry with metadata
//	- Environment-specific attributes
//
//	Attributes Processor:
//	- Add, modify, or delete span/metric attributes
//	- Filter sensitive data
//	- Rename attributes
//
//	Filter Processor:
//	- Filter telemetry based on conditions
//	- Include or exclude signals
//	- Regex and exact matching
//
//	Probabilistic Sampler:
//	- Sample traces based on probability
//	- Reduce data volume
//	- Consistent sampling decisions
//
//	Span Processor:
//	- Modify span names and attributes
//	- Extract attributes from span names
//	- Generate metrics from spans
//
//	Tail Sampling Processor:
//	- Sample based on complete traces
//	- Policy-based decisions (error, latency, etc.)
//	- More intelligent than head-based sampling
//
//	Transform Processor:
//	- Transform telemetry using OTTL
//	- Complex data manipulation
//	- Conditional transformations
//
// Exporters:
//
//	Send telemetry to various backends:
//
//	OTLP Exporter:
//	- Send to OTLP-compatible backends
//	- Supports HTTP and gRPC
//	- Works with Jaeger, Grafana Tempo, etc.
//
//	Jaeger Exporter:
//	- Send traces to Jaeger
//	- gRPC and Thrift protocols
//	- Native Jaeger format
//
//	Zipkin Exporter:
//	- Send traces to Zipkin
//	- JSON format
//	- HTTP transport
//
//	Prometheus Exporter:
//	- Expose metrics for Prometheus scraping
//	- HTTP endpoint
//	- Metric translation
//
//	Prometheus Remote Write Exporter:
//	- Push metrics to Prometheus-compatible backends
//	- Works with Grafana Mimir, Cortex, Thanos
//	- Remote write protocol
//
//	Logging Exporter:
//	- Log telemetry to stdout/stderr
//	- Debugging and development
//	- Multiple log levels
//
//	File Exporter:
//	- Write telemetry to files
//	- JSON or Protobuf format
//	- File rotation support
//
//	Kafka Exporter:
//	- Send telemetry to Kafka
//	- Partitioning support
//	- Compression options
//
//	OpenSearch Exporter:
//	- Send telemetry to OpenSearch
//	- Index management
//	- Bulk indexing
//
//	Loki Exporter:
//	- Send logs to Grafana Loki
//	- Label extraction
//	- Log streaming
//
// Configuration:
//
//	The collector is configured via YAML file at /etc/otelcol/config.yaml:
//
//	receivers:
//	  otlp:
//	    protocols:
//	      http:
//	        endpoint: 0.0.0.0:4318
//	      grpc:
//	        endpoint: 0.0.0.0:4317
//
//	processors:
//	  batch:
//	    timeout: 10s
//	    send_batch_size: 1024
//	  memory_limiter:
//	    check_interval: 1s
//	    limit_mib: 512
//
//	exporters:
//	  otlp/jaeger:
//	    endpoint: jaeger:4317
//	    tls:
//	      insecure: true
//	  prometheus:
//	    endpoint: 0.0.0.0:8889
//	  logging:
//	    loglevel: info
//
//	extensions:
//	  health_check:
//	    endpoint: 0.0.0.0:13133
//	  pprof:
//	    endpoint: 0.0.0.0:1777
//
//	service:
//	  extensions: [health_check, pprof]
//	  pipelines:
//	    traces:
//	      receivers: [otlp]
//	      processors: [memory_limiter, batch]
//	      exporters: [otlp/jaeger, logging]
//	    metrics:
//	      receivers: [otlp]
//	      processors: [memory_limiter, batch]
//	      exporters: [prometheus, logging]
//	    logs:
//	      receivers: [otlp]
//	      processors: [memory_limiter, batch]
//	      exporters: [logging]
//
// Signal Pipelines:
//
//	Traces Pipeline:
//	- Receives distributed traces
//	- Processes spans and trace context
//	- Exports to trace backends (Jaeger, Tempo, etc.)
//	- Enables distributed tracing across microservices
//
//	Metrics Pipeline:
//	- Receives time-series metrics
//	- Processes counters, gauges, histograms
//	- Exports to metrics backends (Prometheus, Mimir, etc.)
//	- Enables metrics monitoring and alerting
//
//	Logs Pipeline:
//	- Receives structured log data
//	- Processes log records
//	- Exports to log backends (Loki, OpenSearch, etc.)
//	- Enables centralized logging
//
// Extensions:
//
//	Optional components for operational features:
//
//	Health Check Extension:
//	- HTTP health endpoint
//	- Readiness and liveness checks
//	- Kubernetes integration
//
//	PPof Extension:
//	- Runtime profiling
//	- CPU and memory profiling
//	- Performance debugging
//
//	zPages Extension:
//	- In-process diagnostics
//	- Pipeline visualization
//	- Trace debugging
//
// Deployment Modes:
//
//	Agent Mode:
//	- Deployed alongside applications
//	- One per host/pod
//	- Lightweight forwarding
//	- Local processing
//
//	Gateway Mode:
//	- Centralized deployment
//	- Receives from multiple agents
//	- Advanced processing
//	- Backend consolidation
//
//	This deployment supports both modes.
//
// Sending Telemetry:
//
//	Applications can send telemetry using OpenTelemetry SDKs:
//
//	Go SDK:
//	exporter, _ := otlptracehttp.New(ctx,
//	    otlptracehttp.WithEndpoint("localhost:4318"),
//	    otlptracehttp.WithInsecure(),
//	)
//
//	Python SDK:
//	exporter = OTLPSpanExporter(
//	    endpoint="http://localhost:4318/v1/traces"
//	)
//
//	Java SDK:
//	OtlpHttpSpanExporter exporter = OtlpHttpSpanExporter.builder()
//	    .setEndpoint("http://localhost:4318/v1/traces")
//	    .build();
//
//	JavaScript SDK:
//	const exporter = new OTLPTraceExporter({
//	    url: 'http://localhost:4318/v1/traces'
//	});
//
// Security:
//
//	For production use:
//	- Enable TLS for receivers (HTTPS, gRPC with TLS)
//	- Configure authentication (API keys, mTLS)
//	- Use secure exporters with credentials
//	- Implement network policies
//	- Filter sensitive attributes
//	- Enable audit logging
//	- Set resource limits
//	- Use least privilege principles
//
// Restart Policy:
//
//	The container is configured with restart policy "unless-stopped",
//	ensuring it automatically restarts if it crashes.
//
// Health Monitoring:
//
//	Health check runs HTTP GET / on port 13133 every 30 seconds:
//	- Timeout: 10 seconds
//	- Retries: 3 consecutive failures before unhealthy
//
//	The health endpoint returns:
//	- 200 OK when collector is healthy
//	- 503 Service Unavailable when not ready
//
// Performance Tuning:
//
//	For production workloads, consider:
//	- Adequate memory for batching and processing
//	- CPU allocation for processing throughput
//	- Network bandwidth for ingestion and export
//	- Batch processor configuration (size and timeout)
//	- Memory limiter to prevent OOM
//	- Concurrent processing settings
//	- Queue size configuration
//	- Compression for network efficiency
//
// Monitoring the Collector:
//
//	The collector exposes its own metrics:
//	- otelcol_receiver_accepted_spans - Spans received
//	- otelcol_receiver_refused_spans - Spans rejected
//	- otelcol_exporter_sent_spans - Spans exported
//	- otelcol_exporter_failed_spans - Export failures
//	- otelcol_processor_batch_size - Batch sizes
//	- otelcol_process_runtime_* - Runtime metrics
//
//	Access metrics at: http://localhost:8888/metrics
//
// Grafana Integration:
//
//	Integrate with Grafana observability stack:
//
//	Traces → Grafana Tempo:
//	exporters:
//	  otlp/tempo:
//	    endpoint: tempo:4317
//
//	Metrics → Grafana Mimir:
//	exporters:
//	  prometheusremotewrite:
//	    endpoint: http://mimir:9009/api/v1/push
//
//	Logs → Grafana Loki:
//	exporters:
//	  loki:
//	    endpoint: http://loki:3100/loki/api/v1/push
//
// High Availability:
//
//	For production HA setup:
//	- Deploy multiple collector instances
//	- Use load balancer for distribution
//	- Configure agent mode on each host
//	- Use gateway mode for centralization
//	- Enable health checks
//	- Monitor collector metrics
//	- Set up alerting for failures
//
// Backup Strategy:
//
//	Important: Backup configuration!
//	- Store configuration in version control
//	- Backup /etc/otelcol volume
//	- Document custom processors and exporters
//	- Test configuration changes in staging
//	- Keep rollback configurations
//
// Migration:
//
//	Migrating to OpenTelemetry:
//
//	From Jaeger:
//	- Configure Jaeger receiver
//	- Update Jaeger clients to OTLP
//	- Use OTLP exporter to Jaeger backend
//	- Gradual migration per service
//
//	From Zipkin:
//	- Configure Zipkin receiver
//	- Update Zipkin clients to OTLP
//	- Use OTLP exporter to Zipkin backend
//	- Gradual migration per service
//
//	From Prometheus:
//	- Configure Prometheus receiver
//	- Scrape existing exporters
//	- Push metrics via OTLP
//	- Use Prometheus Remote Write exporter
//
// Troubleshooting:
//
//	Common issues and solutions:
//
//	Telemetry not received:
//	- Check receiver configuration and ports
//	- Verify network connectivity
//	- Check client endpoint configuration
//	- Review firewall rules
//
//	High memory usage:
//	- Enable memory_limiter processor
//	- Reduce batch sizes
//	- Increase export frequency
//	- Add sampling
//
//	Export failures:
//	- Check backend availability
//	- Verify authentication credentials
//	- Review network connectivity
//	- Check backend capacity
//
//	Performance issues:
//	- Scale collector instances
//	- Optimize processor configuration
//	- Use compression
//	- Enable batching
//
// API Endpoints:
//
//	Key HTTP API endpoints:
//	- POST /v1/traces - Receive trace data (OTLP/HTTP)
//	- POST /v1/metrics - Receive metrics data (OTLP/HTTP)
//	- POST /v1/logs - Receive logs data (OTLP/HTTP)
//	- GET  / - Health check (port 13133)
//	- GET  /metrics - Collector metrics (port 8888, if enabled)
//
// Error Handling:
//
//	Returns error if:
//	- Container with same name already exists
//	- Network or volume creation fails
//	- Docker API errors occur
//	- Invalid configuration provided
func DeployOTelCollector(ctx context.Context, cli common.DockerClient, config OTelCollectorProductionConfig) (string, error) {
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

	// Create config volume if it doesn't exist
	if config.ConfigVolume != "" {
		if err := EnsureVolume(ctx, cli, config.ConfigVolume); err != nil {
			return "", fmt.Errorf("failed to ensure config volume: %w", err)
		}
	}

	// Pull OTel Collector image
	if err := common.ImagePullWithClient(ctx, cli, config.Image, &common.ImagePullOptions{Silent: true}); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Configure port bindings
	portMap := nat.PortMap{
		"4318/tcp": []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: config.HTTPPort,
			},
		},
		"4317/tcp": []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: config.GRPCPort,
			},
		},
		"13133/tcp": []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: config.HealthPort,
			},
		},
	}

	// Configure volume mounts
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: config.DataVolume,
			Target: "/var/lib/otelcol",
		},
	}

	// Add config volume if specified
	if config.ConfigVolume != "" {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeVolume,
			Source: config.ConfigVolume,
			Target: "/etc/otelcol",
		})
	}

	// Container configuration
	containerConfig := container.Config{
		Image: config.Image,
		ExposedPorts: nat.PortSet{
			"4318/tcp":  struct{}{}, // OTLP HTTP
			"4317/tcp":  struct{}{}, // OTLP gRPC
			"13133/tcp": struct{}{}, // Health check
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD-SHELL", "wget --spider --quiet http://localhost:13133 || exit 1"},
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

// StopOTelCollector stops a running OpenTelemetry Collector container.
//
// Performs graceful shutdown to ensure telemetry is flushed.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the OTel Collector container to stop
//
// Returns:
//   - error: Stop errors
//
// Example:
//
//	err := StopOTelCollector(ctx, cli, "otelcollector")
//	if err != nil {
//	    log.Printf("Failed to stop OTel Collector: %v", err)
//	}
func StopOTelCollector(ctx context.Context, cli common.DockerClient, containerName string) error {
	timeout := 30 // 30 seconds for graceful shutdown
	return cli.ContainerStop(ctx, containerName, container.StopOptions{Timeout: &timeout})
}

// RemoveOTelCollector removes an OTel Collector container and optionally its volumes.
//
// WARNING: Removing volumes will DELETE ALL CONFIGURATION and DATA permanently!
// Always backup data before removing volumes.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the OTel Collector container to remove
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
//	err := RemoveOTelCollector(ctx, cli, "otelcollector", false, "", "")
//
//	// Remove container and volumes (DANGEROUS - backup first!)
//	err := RemoveOTelCollector(ctx, cli, "otelcollector", true,
//	    "otelcollector-config", "otelcollector-data")
func RemoveOTelCollector(ctx context.Context, cli common.DockerClient, containerName string, removeVolumes bool, configVolumeName, dataVolumeName string) error {
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

// GetOTelCollectorHTTPURL returns the OTLP HTTP endpoint URL for the deployed container.
//
// This is a convenience function that formats the URL for sending telemetry via HTTP.
//
// Parameters:
//   - config: OTel Collector production configuration
//
// Returns:
//   - string: OTLP HTTP endpoint URL
//
// Example:
//
//	config := DefaultOTelCollectorProductionConfig()
//	httpURL := GetOTelCollectorHTTPURL(config)
//	// http://localhost:4318
func GetOTelCollectorHTTPURL(config OTelCollectorProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s", config.HTTPPort)
}

// GetOTelCollectorGRPCURL returns the OTLP gRPC endpoint URL for the deployed container.
//
// This is a convenience function that formats the URL for sending telemetry via gRPC.
//
// Parameters:
//   - config: OTel Collector production configuration
//
// Returns:
//   - string: OTLP gRPC endpoint URL
//
// Example:
//
//	config := DefaultOTelCollectorProductionConfig()
//	grpcURL := GetOTelCollectorGRPCURL(config)
//	// localhost:4317
func GetOTelCollectorGRPCURL(config OTelCollectorProductionConfig) string {
	return fmt.Sprintf("localhost:%s", config.GRPCPort)
}

// GetOTelCollectorHealthURL returns the health check endpoint URL for the deployed container.
//
// This is a convenience function for monitoring and health checks.
//
// Parameters:
//   - config: OTel Collector production configuration
//
// Returns:
//   - string: Health check endpoint URL
//
// Example:
//
//	config := DefaultOTelCollectorProductionConfig()
//	healthURL := GetOTelCollectorHealthURL(config)
//	// http://localhost:13133
func GetOTelCollectorHealthURL(config OTelCollectorProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s", config.HealthPort)
}
