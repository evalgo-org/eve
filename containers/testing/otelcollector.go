package testing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// OTelCollectorConfig holds configuration for OpenTelemetry Collector testcontainer setup.
type OTelCollectorConfig struct {
	// Image is the Docker image to use (default: "otel/opentelemetry-collector:nightly")
	Image string
	// StartupTimeout is the maximum time to wait for OTel Collector to be ready (default: 60s)
	StartupTimeout time.Duration
}

// DefaultOTelCollectorConfig returns the default OpenTelemetry Collector configuration for testing.
func DefaultOTelCollectorConfig() OTelCollectorConfig {
	return OTelCollectorConfig{
		Image:          "otel/opentelemetry-collector:nightly",
		StartupTimeout: 60 * time.Second,
	}
}

// SetupOTelCollector creates an OpenTelemetry Collector container for integration testing.
//
// OpenTelemetry Collector is a vendor-agnostic agent for receiving, processing, and exporting
// telemetry data (traces, metrics, and logs). This function starts an OTel Collector container
// using testcontainers-go and returns the connection URLs and a cleanup function.
//
// Container Configuration:
//   - Image: otel/opentelemetry-collector:nightly (latest OpenTelemetry Collector)
//   - Port: 4318/tcp (OTLP HTTP receiver)
//   - Port: 4317/tcp (OTLP gRPC receiver)
//   - Port: 13133/tcp (health check extension)
//   - Wait Strategy: HTTP GET / returning 200 OK on port 13133
//
// Parameters:
//   - ctx: Context for container operations
//   - t: Testing context for requirement checks
//   - config: Optional OTel Collector configuration (uses defaults if nil)
//
// Returns:
//   - string: OTel Collector HTTP endpoint URL (OTLP HTTP receiver)
//            (e.g., "http://localhost:32793")
//   - string: OTel Collector gRPC endpoint URL (OTLP gRPC receiver)
//            (e.g., "localhost:32794")
//   - ContainerCleanup: Function to terminate the container
//   - error: Container creation or startup errors
//
// Example Usage:
//
//	func TestOTelCollectorIntegration(t *testing.T) {
//	    ctx := context.Background()
//	    httpURL, grpcURL, cleanup, err := SetupOTelCollector(ctx, t, nil)
//	    require.NoError(t, err)
//	    defer cleanup()
//
//	    // Use OTel Collector endpoints
//	    // Send traces via HTTP: POST to httpURL + "/v1/traces"
//	    // Send metrics via gRPC: connect to grpcURL
//	}
//
// OpenTelemetry Collector Features:
//
//	Vendor-agnostic telemetry collection and processing:
//	- Receive telemetry data in multiple formats (OTLP, Jaeger, Zipkin, Prometheus)
//	- Process telemetry data (filtering, sampling, batching, enrichment)
//	- Export telemetry data to multiple backends (Jaeger, Zipkin, Prometheus, etc.)
//	- Support for traces, metrics, and logs
//	- Pluggable architecture with receivers, processors, and exporters
//	- Configuration via YAML
//	- Low resource consumption
//	- High performance and scalability
//
// OTLP Protocol:
//
//	OpenTelemetry Protocol (OTLP) is the native protocol:
//	- OTLP/HTTP: HTTP/1.1 with JSON or Protobuf
//	- OTLP/gRPC: gRPC with Protobuf
//
//	OTLP supports three signal types:
//	- Traces: Distributed tracing data
//	- Metrics: Time-series metrics
//	- Logs: Structured log data
//
// Receivers:
//
//	The collector can receive telemetry from various sources:
//	- OTLP receiver: Native OpenTelemetry protocol (HTTP/gRPC)
//	- Jaeger receiver: Jaeger format traces
//	- Zipkin receiver: Zipkin format traces
//	- Prometheus receiver: Scrapes Prometheus metrics
//	- Host metrics receiver: System metrics
//	- Kubernetes receiver: Kubernetes cluster metrics
//	- File receiver: Read from log files
//	- Syslog receiver: Syslog protocol
//
// Processors:
//
//	Process telemetry data before exporting:
//	- Batch processor: Batches telemetry for efficiency
//	- Memory limiter: Prevents OOM by limiting memory usage
//	- Resource processor: Add/modify resource attributes
//	- Attributes processor: Add/modify span attributes
//	- Filter processor: Filter telemetry based on conditions
//	- Probabilistic sampler: Sample traces based on probability
//	- Span processor: Modify span properties
//	- Tail sampling: Sample based on complete trace
//	- Transform processor: Transform telemetry data
//
// Exporters:
//
//	Export telemetry to various backends:
//	- OTLP exporter: Send to OTLP-compatible backends
//	- Jaeger exporter: Send traces to Jaeger
//	- Zipkin exporter: Send traces to Zipkin
//	- Prometheus exporter: Expose metrics for Prometheus scraping
//	- Logging exporter: Log telemetry (for debugging)
//	- File exporter: Write to files
//	- Kafka exporter: Send to Kafka
//	- OpenSearch exporter: Send to OpenSearch
//	- Loki exporter: Send logs to Loki
//
// API Endpoints:
//
//	Key HTTP endpoints available:
//	- POST /v1/traces - Receive trace data (OTLP/HTTP)
//	- POST /v1/metrics - Receive metrics data (OTLP/HTTP)
//	- POST /v1/logs - Receive logs data (OTLP/HTTP)
//	- GET  / - Health check (on port 13133)
//	- GET  /metrics - Collector's own metrics (Prometheus format)
//
// gRPC Services:
//
//	Key gRPC services available:
//	- opentelemetry.proto.collector.trace.v1.TraceService/Export
//	- opentelemetry.proto.collector.metrics.v1.MetricsService/Export
//	- opentelemetry.proto.collector.logs.v1.LogsService/Export
//
// Configuration:
//
//	The default configuration includes:
//	- OTLP receivers (HTTP on 4318, gRPC on 4317)
//	- Batch processor for efficiency
//	- Logging exporter for debugging
//
//	For custom configuration, use SetupOTelCollectorWithConfig.
//
// Performance:
//
//	OTel Collector container starts in 5-15 seconds typically.
//	The wait strategy ensures the health check endpoint is ready
//	before returning.
//
// Data Storage:
//
//	The default configuration exports telemetry to stdout (logging exporter).
//	For testing, this is ephemeral (lost when container stops).
//	This ensures test isolation.
//
// Sending Telemetry:
//
//	Send traces via HTTP:
//	POST http://localhost:{httpPort}/v1/traces
//	Content-Type: application/json or application/x-protobuf
//
//	Send metrics via HTTP:
//	POST http://localhost:{httpPort}/v1/metrics
//	Content-Type: application/json or application/x-protobuf
//
//	Send logs via HTTP:
//	POST http://localhost:{httpPort}/v1/logs
//	Content-Type: application/json or application/x-protobuf
//
//	Send via gRPC:
//	Connect to grpcURL and call the appropriate service method.
//
// Cleanup:
//
//	Always defer the cleanup function to ensure the container is terminated:
//	defer cleanup()
//
//	The cleanup function is safe to call even if setup fails (it's a no-op).
//
// Error Handling:
//
//	If container creation fails, the test should fail with require.NoError(t, err).
//	Common errors:
//	- Docker daemon not running
//	- Image pull failures (network issues)
//	- Port conflicts (rare with random ports)
//
// Use Cases:
//
//	Integration testing scenarios:
//	- Testing OpenTelemetry instrumentation
//	- Testing trace collection and export
//	- Testing metrics collection and export
//	- Testing log collection and export
//	- Testing custom processors and exporters
//	- Testing collector configuration
//	- Testing multi-signal pipelines
//	- Testing sampling strategies
func SetupOTelCollector(ctx context.Context, t *testing.T, config *OTelCollectorConfig) (string, string, ContainerCleanup, error) {
	// Use default config if none provided
	if config == nil {
		defaultConfig := DefaultOTelCollectorConfig()
		config = &defaultConfig
	}

	// Create container request
	req := testcontainers.ContainerRequest{
		Image: config.Image,
		ExposedPorts: []string{
			"4318/tcp",  // OTLP HTTP receiver
			"4317/tcp",  // OTLP gRPC receiver
			"13133/tcp", // Health check extension
		},
		// Health check on health extension port
		WaitingFor: wait.ForHTTP("/").
			WithPort("13133/tcp").
			WithStartupTimeout(config.StartupTimeout),
	}

	// Start container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", "", func() {}, fmt.Errorf("failed to start OTel Collector container: %w", err)
	}

	// Get container connection details
	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return "", "", func() {}, fmt.Errorf("failed to get container host: %w", err)
	}

	// Get HTTP port
	httpPort, err := container.MappedPort(ctx, "4318")
	if err != nil {
		container.Terminate(ctx)
		return "", "", func() {}, fmt.Errorf("failed to get HTTP port: %w", err)
	}

	// Get gRPC port
	grpcPort, err := container.MappedPort(ctx, "4317")
	if err != nil {
		container.Terminate(ctx)
		return "", "", func() {}, fmt.Errorf("failed to get gRPC port: %w", err)
	}

	// Build OTel Collector endpoint URLs
	// HTTP endpoint: http://host:port
	httpURL := fmt.Sprintf("http://%s:%s", host, httpPort.Port())
	// gRPC endpoint: host:port (no protocol)
	grpcURL := fmt.Sprintf("%s:%s", host, grpcPort.Port())

	// Create cleanup function
	cleanup := createCleanupFunc(ctx, container, "OpenTelemetry Collector")

	return httpURL, grpcURL, cleanup, nil
}

// SetupOTelCollectorWithConfig creates an OTel Collector container with custom configuration.
//
// This function allows you to provide a custom collector configuration file.
// Useful for testing specific receiver, processor, or exporter configurations.
//
// Parameters:
//   - ctx: Context for container operations
//   - t: Testing context for requirement checks
//   - config: Optional OTel Collector configuration (uses defaults if nil)
//   - configContent: YAML configuration content for the collector
//
// Returns:
//   - string: OTel Collector HTTP endpoint URL
//   - string: OTel Collector gRPC endpoint URL
//   - ContainerCleanup: Function to terminate the container
//   - error: Container creation or startup errors
//
// Example Usage:
//
//	configYAML := `
//	receivers:
//	  otlp:
//	    protocols:
//	      http:
//	        endpoint: 0.0.0.0:4318
//	      grpc:
//	        endpoint: 0.0.0.0:4317
//	processors:
//	  batch:
//	    timeout: 1s
//	    send_batch_size: 1024
//	exporters:
//	  logging:
//	    loglevel: debug
//	service:
//	  pipelines:
//	    traces:
//	      receivers: [otlp]
//	      processors: [batch]
//	      exporters: [logging]
//	`
//
//	httpURL, grpcURL, cleanup, err := SetupOTelCollectorWithConfig(
//	    ctx, t, nil, configYAML)
//	require.NoError(t, err)
//	defer cleanup()
//
// Configuration Format:
//
//	The configuration must be valid YAML following the OTel Collector schema:
//	- receivers: Define how to receive telemetry
//	- processors: Define how to process telemetry
//	- exporters: Define where to send telemetry
//	- service.pipelines: Define signal pipelines
//	- extensions: Optional extensions (health check, pprof, etc.)
//
// Custom Receivers:
//
//	Configure custom receivers in the configuration:
//	receivers:
//	  jaeger:
//	    protocols:
//	      thrift_http:
//	        endpoint: 0.0.0.0:14268
//	  prometheus:
//	    config:
//	      scrape_configs:
//	        - job_name: 'otel-collector'
//	          scrape_interval: 10s
//
// Custom Processors:
//
//	Configure custom processors:
//	processors:
//	  attributes:
//	    actions:
//	      - key: environment
//	        value: testing
//	        action: insert
//	  probabilistic_sampler:
//	    sampling_percentage: 10
//
// Custom Exporters:
//
//	Configure custom exporters:
//	exporters:
//	  otlp/jaeger:
//	    endpoint: jaeger:4317
//	    tls:
//	      insecure: true
//	  prometheus:
//	    endpoint: 0.0.0.0:8889
//
// Pipelines:
//
//	Define signal pipelines:
//	service:
//	  pipelines:
//	    traces:
//	      receivers: [otlp, jaeger]
//	      processors: [batch, attributes]
//	      exporters: [otlp/jaeger, logging]
//	    metrics:
//	      receivers: [otlp, prometheus]
//	      processors: [batch]
//	      exporters: [prometheus, logging]
//	    logs:
//	      receivers: [otlp]
//	      processors: [batch]
//	      exporters: [logging]
func SetupOTelCollectorWithConfig(ctx context.Context, t *testing.T, config *OTelCollectorConfig, configContent string) (string, string, ContainerCleanup, error) {
	// Use default config if none provided
	if config == nil {
		defaultConfig := DefaultOTelCollectorConfig()
		config = &defaultConfig
	}

	// Create container request with custom configuration
	req := testcontainers.ContainerRequest{
		Image: config.Image,
		ExposedPorts: []string{
			"4318/tcp",  // OTLP HTTP receiver
			"4317/tcp",  // OTLP gRPC receiver
			"13133/tcp", // Health check extension
		},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      configContent,
				ContainerFilePath: "/etc/otelcol/config.yaml",
				FileMode:          0644,
			},
		},
		Cmd: []string{"--config=/etc/otelcol/config.yaml"},
		// Health check on health extension port
		WaitingFor: wait.ForHTTP("/").
			WithPort("13133/tcp").
			WithStartupTimeout(config.StartupTimeout),
	}

	// Start container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", "", func() {}, fmt.Errorf("failed to start OTel Collector container: %w", err)
	}

	// Get container connection details
	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return "", "", func() {}, fmt.Errorf("failed to get container host: %w", err)
	}

	// Get HTTP port
	httpPort, err := container.MappedPort(ctx, "4318")
	if err != nil {
		container.Terminate(ctx)
		return "", "", func() {}, fmt.Errorf("failed to get HTTP port: %w", err)
	}

	// Get gRPC port
	grpcPort, err := container.MappedPort(ctx, "4317")
	if err != nil {
		container.Terminate(ctx)
		return "", "", func() {}, fmt.Errorf("failed to get gRPC port: %w", err)
	}

	// Build OTel Collector endpoint URLs
	httpURL := fmt.Sprintf("http://%s:%s", host, httpPort.Port())
	grpcURL := fmt.Sprintf("%s:%s", host, grpcPort.Port())

	// Create cleanup function
	cleanup := createCleanupFunc(ctx, container, "OpenTelemetry Collector")

	return httpURL, grpcURL, cleanup, nil
}
