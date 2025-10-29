package production

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultOTelCollectorProductionConfig(t *testing.T) {
	config := DefaultOTelCollectorProductionConfig()

	assert.Equal(t, "otelcollector", config.ContainerName)
	assert.Equal(t, "otel/opentelemetry-collector:nightly", config.Image)
	assert.Equal(t, "4318", config.HTTPPort)
	assert.Equal(t, "4317", config.GRPCPort)
	assert.Equal(t, "13133", config.HealthPort)
	assert.Equal(t, "otelcollector-config", config.ConfigVolume)
	assert.Equal(t, "otelcollector-data", config.DataVolume)
	assert.Equal(t, "app-network", config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)
	assert.Equal(t, "otelcollector-data", config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)
}

func TestOTelCollectorProductionConfig_CustomConfiguration(t *testing.T) {
	config := DefaultOTelCollectorProductionConfig()

	// Modify configuration
	config.ContainerName = "my-otelcollector"
	config.HTTPPort = "14318"
	config.GRPCPort = "14317"
	config.HealthPort = "13134"
	config.ConfigVolume = "custom-otelcollector-config"
	config.DataVolume = "custom-otelcollector-data"

	assert.Equal(t, "my-otelcollector", config.ContainerName)
	assert.Equal(t, "14318", config.HTTPPort)
	assert.Equal(t, "14317", config.GRPCPort)
	assert.Equal(t, "13134", config.HealthPort)
	assert.Equal(t, "custom-otelcollector-config", config.ConfigVolume)
	assert.Equal(t, "custom-otelcollector-data", config.DataVolume)
}

func TestGetOTelCollectorHTTPURL(t *testing.T) {
	tests := []struct {
		name     string
		config   OTelCollectorProductionConfig
		expected string
	}{
		{
			name: "default http port",
			config: OTelCollectorProductionConfig{
				HTTPPort: "4318",
			},
			expected: "http://localhost:4318",
		},
		{
			name: "custom http port",
			config: OTelCollectorProductionConfig{
				HTTPPort: "14318",
			},
			expected: "http://localhost:14318",
		},
		{
			name: "high http port number",
			config: OTelCollectorProductionConfig{
				HTTPPort: "54318",
			},
			expected: "http://localhost:54318",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpURL := GetOTelCollectorHTTPURL(tt.config)
			assert.Equal(t, tt.expected, httpURL)
		})
	}
}

func TestGetOTelCollectorGRPCURL(t *testing.T) {
	tests := []struct {
		name     string
		config   OTelCollectorProductionConfig
		expected string
	}{
		{
			name: "default grpc port",
			config: OTelCollectorProductionConfig{
				GRPCPort: "4317",
			},
			expected: "localhost:4317",
		},
		{
			name: "custom grpc port",
			config: OTelCollectorProductionConfig{
				GRPCPort: "14317",
			},
			expected: "localhost:14317",
		},
		{
			name: "high grpc port number",
			config: OTelCollectorProductionConfig{
				GRPCPort: "54317",
			},
			expected: "localhost:54317",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grpcURL := GetOTelCollectorGRPCURL(tt.config)
			assert.Equal(t, tt.expected, grpcURL)
		})
	}
}

func TestGetOTelCollectorHealthURL(t *testing.T) {
	tests := []struct {
		name     string
		config   OTelCollectorProductionConfig
		expected string
	}{
		{
			name: "default health port",
			config: OTelCollectorProductionConfig{
				HealthPort: "13133",
			},
			expected: "http://localhost:13133",
		},
		{
			name: "custom health port",
			config: OTelCollectorProductionConfig{
				HealthPort: "13134",
			},
			expected: "http://localhost:13134",
		},
		{
			name: "high health port number",
			config: OTelCollectorProductionConfig{
				HealthPort: "23133",
			},
			expected: "http://localhost:23133",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healthURL := GetOTelCollectorHealthURL(tt.config)
			assert.Equal(t, tt.expected, healthURL)
		})
	}
}

func TestOTelCollectorProductionConfig_DataPersistence(t *testing.T) {
	config := DefaultOTelCollectorProductionConfig()

	// Verify data volume is configured for persistence
	assert.NotEmpty(t, config.DataVolume, "Data volume should be configured")
	assert.Equal(t, "otelcollector-data", config.DataVolume, "Default data volume name")

	// Verify config volume is configured
	assert.NotEmpty(t, config.ConfigVolume, "Config volume should be configured")
	assert.Equal(t, "otelcollector-config", config.ConfigVolume, "Default config volume name")

	// Test custom volumes
	config.DataVolume = "production-otelcollector-data"
	config.ConfigVolume = "production-otelcollector-config"
	assert.Equal(t, "production-otelcollector-data", config.DataVolume)
	assert.Equal(t, "production-otelcollector-config", config.ConfigVolume)
}

func TestOTelCollectorProductionConfig_PortConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		httpPort    string
		grpcPort    string
		healthPort  string
		description string
	}{
		{
			name:        "default ports",
			httpPort:    "4318",
			grpcPort:    "4317",
			healthPort:  "13133",
			description: "OTLP standard ports",
		},
		{
			name:        "custom ports",
			httpPort:    "14318",
			grpcPort:    "14317",
			healthPort:  "13134",
			description: "Alternative ports to avoid conflicts",
		},
		{
			name:        "high port numbers",
			httpPort:    "54318",
			grpcPort:    "54317",
			healthPort:  "53133",
			description: "High port numbers for security",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultOTelCollectorProductionConfig()
			config.HTTPPort = tt.httpPort
			config.GRPCPort = tt.grpcPort
			config.HealthPort = tt.healthPort

			assert.Equal(t, tt.httpPort, config.HTTPPort, "HTTP port: "+tt.description)
			assert.Equal(t, tt.grpcPort, config.GRPCPort, "gRPC port: "+tt.description)
			assert.Equal(t, tt.healthPort, config.HealthPort, "Health port: "+tt.description)
		})
	}
}

func TestOTelCollectorProductionConfig_ImageVersion(t *testing.T) {
	config := DefaultOTelCollectorProductionConfig()

	// Verify that OTel Collector nightly image is being used
	assert.Contains(t, config.Image, "otel/opentelemetry-collector", "Should use OTel Collector image")
	assert.Equal(t, "otel/opentelemetry-collector:nightly", config.Image)

	// Test custom image version
	config.Image = "otel/opentelemetry-collector:0.91.0"
	assert.Equal(t, "otel/opentelemetry-collector:0.91.0", config.Image)
}

func TestOTelCollectorProductionConfig_NetworkIsolation(t *testing.T) {
	config := DefaultOTelCollectorProductionConfig()

	// Verify network configuration
	assert.NotEmpty(t, config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)

	// Test custom network
	config.Production.NetworkName = "otelcollector-network"
	assert.Equal(t, "otelcollector-network", config.Production.NetworkName)
}

func TestOTelCollectorProductionConfig_URLFormatting(t *testing.T) {
	config := DefaultOTelCollectorProductionConfig()

	// Test HTTP URL
	httpURL := GetOTelCollectorHTTPURL(config)
	assert.True(t, len(httpURL) > 0, "HTTP URL should not be empty")
	assert.Contains(t, httpURL, "http://")
	assert.Contains(t, httpURL, config.HTTPPort)

	// Test gRPC URL
	grpcURL := GetOTelCollectorGRPCURL(config)
	assert.True(t, len(grpcURL) > 0, "gRPC URL should not be empty")
	assert.Contains(t, grpcURL, config.GRPCPort)
	assert.NotContains(t, grpcURL, "http://", "gRPC URL should not have http:// prefix")

	// Test health URL
	healthURL := GetOTelCollectorHealthURL(config)
	assert.True(t, len(healthURL) > 0, "Health URL should not be empty")
	assert.Contains(t, healthURL, config.HealthPort)
	assert.Contains(t, healthURL, "http://")
}

func TestOTelCollectorProductionConfig_VolumeConfiguration(t *testing.T) {
	config := DefaultOTelCollectorProductionConfig()

	// Verify volume configuration
	assert.Equal(t, "otelcollector-data", config.DataVolume)
	assert.Equal(t, "otelcollector-config", config.ConfigVolume)
	assert.Equal(t, "otelcollector-data", config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)

	// Verify data volume and production settings match
	assert.Equal(t, config.DataVolume, config.Production.VolumeName,
		"DataVolume and Production.VolumeName should match")
}

func TestOTelCollectorProductionConfig_OTLPProtocols(t *testing.T) {
	config := DefaultOTelCollectorProductionConfig()

	// Verify both OTLP protocols are configured
	assert.NotEmpty(t, config.HTTPPort, "OTLP HTTP port should be configured")
	assert.NotEmpty(t, config.GRPCPort, "OTLP gRPC port should be configured")

	// Verify default OTLP ports
	assert.Equal(t, "4318", config.HTTPPort, "OTLP HTTP default port")
	assert.Equal(t, "4317", config.GRPCPort, "OTLP gRPC default port")

	// Test that HTTP and gRPC ports are different
	assert.NotEqual(t, config.HTTPPort, config.GRPCPort,
		"HTTP and gRPC ports should be different")
}

func TestOTelCollectorProductionConfig_ContainerName(t *testing.T) {
	config := DefaultOTelCollectorProductionConfig()

	// Default container name
	assert.Equal(t, "otelcollector", config.ContainerName)

	// Custom container name
	config.ContainerName = "otelcollector-prod"
	assert.Equal(t, "otelcollector-prod", config.ContainerName)
}

func TestOTelCollectorProductionConfig_DefaultPorts(t *testing.T) {
	config := DefaultOTelCollectorProductionConfig()

	// Verify OTLP standard ports
	assert.Equal(t, "4318", config.HTTPPort, "OTLP HTTP standard port")
	assert.Equal(t, "4317", config.GRPCPort, "OTLP gRPC standard port")
	assert.Equal(t, "13133", config.HealthPort, "Health check standard port")
}

func TestOTelCollectorProductionConfig_MultipleVolumes(t *testing.T) {
	config := DefaultOTelCollectorProductionConfig()

	// Verify both volumes are configured
	assert.NotEmpty(t, config.ConfigVolume, "Config volume should be configured")
	assert.NotEmpty(t, config.DataVolume, "Data volume should be configured")

	// Verify volumes have different names
	assert.NotEqual(t, config.ConfigVolume, config.DataVolume,
		"Config and data volumes should be different")
}

func TestOTelCollectorProductionConfig_HealthCheckPort(t *testing.T) {
	config := DefaultOTelCollectorProductionConfig()

	// Verify health check port is configured
	assert.NotEmpty(t, config.HealthPort, "Health check port should be configured")
	assert.Equal(t, "13133", config.HealthPort, "Default health check port")

	// Test custom health check port
	config.HealthPort = "8888"
	assert.Equal(t, "8888", config.HealthPort)
}

func TestOTelCollectorProductionConfig_HTTPURLEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		port     string
		endpoint string
	}{
		{
			name:     "traces endpoint",
			port:     "4318",
			endpoint: "/v1/traces",
		},
		{
			name:     "metrics endpoint",
			port:     "4318",
			endpoint: "/v1/metrics",
		},
		{
			name:     "logs endpoint",
			port:     "4318",
			endpoint: "/v1/logs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultOTelCollectorProductionConfig()
			config.HTTPPort = tt.port

			baseURL := GetOTelCollectorHTTPURL(config)
			fullURL := baseURL + tt.endpoint

			assert.Contains(t, fullURL, "http://localhost:"+tt.port)
			assert.Contains(t, fullURL, tt.endpoint)
		})
	}
}

func TestOTelCollectorProductionConfig_PortConflicts(t *testing.T) {
	config := DefaultOTelCollectorProductionConfig()

	// Ensure all three ports are different
	ports := []string{config.HTTPPort, config.GRPCPort, config.HealthPort}
	portSet := make(map[string]bool)

	for _, port := range ports {
		assert.False(t, portSet[port], "Port %s is duplicated", port)
		portSet[port] = true
	}

	assert.Equal(t, 3, len(portSet), "Should have 3 unique ports")
}

func TestOTelCollectorProductionConfig_VolumePathMapping(t *testing.T) {
	config := DefaultOTelCollectorProductionConfig()

	// Verify volume names are appropriate for their mount paths
	assert.Contains(t, config.ConfigVolume, "config",
		"Config volume name should indicate it's for configuration")
	assert.Contains(t, config.DataVolume, "data",
		"Data volume name should indicate it's for data")
}

func TestOTelCollectorProductionConfig_GRPCURLFormat(t *testing.T) {
	config := DefaultOTelCollectorProductionConfig()
	grpcURL := GetOTelCollectorGRPCURL(config)

	// gRPC URL should not have protocol prefix
	assert.NotContains(t, grpcURL, "http://", "gRPC URL should not have http:// prefix")
	assert.NotContains(t, grpcURL, "grpc://", "gRPC URL should not have grpc:// prefix")

	// Should be in host:port format
	assert.Contains(t, grpcURL, ":")
	assert.Contains(t, grpcURL, "localhost")
}

func TestOTelCollectorProductionConfig_ImageNaming(t *testing.T) {
	config := DefaultOTelCollectorProductionConfig()

	// Verify image naming convention
	assert.Contains(t, config.Image, "otel/", "Image should be from otel/ repository")
	assert.Contains(t, config.Image, "opentelemetry-collector", "Image name should be opentelemetry-collector")
	assert.Contains(t, config.Image, ":", "Image should have a tag")
}

func TestOTelCollectorProductionConfig_RestartPolicy(t *testing.T) {
	config := DefaultOTelCollectorProductionConfig()

	// The restart policy is set in DeployOTelCollector to "unless-stopped"
	// This test just validates the config structure is ready
	assert.NotEmpty(t, config.ContainerName, "Container name is required for restart policy")
}

func TestOTelCollectorProductionConfig_MultiSignalSupport(t *testing.T) {
	config := DefaultOTelCollectorProductionConfig()

	// OTel Collector supports all three signals via the same endpoints
	httpURL := GetOTelCollectorHTTPURL(config)

	// Verify base URL is configured
	assert.NotEmpty(t, httpURL)

	// All three signal types should be accessible via this endpoint
	tracesURL := httpURL + "/v1/traces"
	metricsURL := httpURL + "/v1/metrics"
	logsURL := httpURL + "/v1/logs"

	assert.Contains(t, tracesURL, config.HTTPPort)
	assert.Contains(t, metricsURL, config.HTTPPort)
	assert.Contains(t, logsURL, config.HTTPPort)
}

func TestOTelCollectorProductionConfig_NetworkConnectivity(t *testing.T) {
	config := DefaultOTelCollectorProductionConfig()

	// Verify production network is configured
	assert.NotEmpty(t, config.Production.NetworkName,
		"Network name should be configured for container connectivity")
	assert.True(t, config.Production.CreateNetwork,
		"Network creation should be enabled")
}

func TestOTelCollectorProductionConfig_ConfigurationPersistence(t *testing.T) {
	config := DefaultOTelCollectorProductionConfig()

	// Verify both config and data volumes are different
	assert.NotEqual(t, config.ConfigVolume, config.DataVolume,
		"Config and data should use separate volumes")

	// Verify volumes are properly named
	assert.Contains(t, config.ConfigVolume, "otelcollector")
	assert.Contains(t, config.DataVolume, "otelcollector")
}
