package production

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultFluentBitProductionConfig(t *testing.T) {
	config := DefaultFluentBitProductionConfig()

	assert.Equal(t, "fluentbit", config.ContainerName)
	assert.Equal(t, "fluent/fluent-bit:4.0.13-amd64", config.Image)
	assert.Equal(t, "2020", config.Port)
	assert.Equal(t, "24224", config.ForwardPort)
	assert.Equal(t, "fluentbit-config", config.ConfigVolume)
	assert.Equal(t, "fluentbit-data", config.DataVolume)
	assert.Equal(t, "app-network", config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)
	assert.Equal(t, "fluentbit-data", config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)
}

func TestFluentBitProductionConfig_CustomConfiguration(t *testing.T) {
	config := DefaultFluentBitProductionConfig()

	// Modify configuration
	config.ContainerName = "my-fluentbit"
	config.Port = "2021"
	config.ForwardPort = "24225"
	config.ConfigVolume = "custom-fluentbit-config"
	config.DataVolume = "custom-fluentbit-data"

	assert.Equal(t, "my-fluentbit", config.ContainerName)
	assert.Equal(t, "2021", config.Port)
	assert.Equal(t, "24225", config.ForwardPort)
	assert.Equal(t, "custom-fluentbit-config", config.ConfigVolume)
	assert.Equal(t, "custom-fluentbit-data", config.DataVolume)
}

func TestGetFluentBitURL(t *testing.T) {
	tests := []struct {
		name     string
		config   FluentBitProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: FluentBitProductionConfig{
				Port: "2020",
			},
			expected: "http://localhost:2020",
		},
		{
			name: "custom port",
			config: FluentBitProductionConfig{
				Port: "2021",
			},
			expected: "http://localhost:2021",
		},
		{
			name: "high port number",
			config: FluentBitProductionConfig{
				Port: "12020",
			},
			expected: "http://localhost:12020",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fluentbitURL := GetFluentBitURL(tt.config)
			assert.Equal(t, tt.expected, fluentbitURL)
		})
	}
}

func TestGetFluentBitHealthURL(t *testing.T) {
	tests := []struct {
		name     string
		config   FluentBitProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: FluentBitProductionConfig{
				Port: "2020",
			},
			expected: "http://localhost:2020/api/v1/health",
		},
		{
			name: "custom port",
			config: FluentBitProductionConfig{
				Port: "2021",
			},
			expected: "http://localhost:2021/api/v1/health",
		},
		{
			name: "high port",
			config: FluentBitProductionConfig{
				Port: "12020",
			},
			expected: "http://localhost:12020/api/v1/health",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healthURL := GetFluentBitHealthURL(tt.config)
			assert.Equal(t, tt.expected, healthURL)
		})
	}
}

func TestGetFluentBitMetricsURL(t *testing.T) {
	tests := []struct {
		name     string
		config   FluentBitProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: FluentBitProductionConfig{
				Port: "2020",
			},
			expected: "http://localhost:2020/api/v1/metrics/prometheus",
		},
		{
			name: "custom port",
			config: FluentBitProductionConfig{
				Port: "2021",
			},
			expected: "http://localhost:2021/api/v1/metrics/prometheus",
		},
		{
			name: "alternative port",
			config: FluentBitProductionConfig{
				Port: "3020",
			},
			expected: "http://localhost:3020/api/v1/metrics/prometheus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metricsURL := GetFluentBitMetricsURL(tt.config)
			assert.Equal(t, tt.expected, metricsURL)
		})
	}
}

func TestFluentBitProductionConfig_DataPersistence(t *testing.T) {
	config := DefaultFluentBitProductionConfig()

	// Verify data volume is configured for persistence
	assert.NotEmpty(t, config.DataVolume, "Data volume should be configured")
	assert.Equal(t, "fluentbit-data", config.DataVolume, "Default data volume name")

	// Test custom volume
	config.DataVolume = "production-fluentbit-volume"
	assert.Equal(t, "production-fluentbit-volume", config.DataVolume)
}

func TestFluentBitProductionConfig_PortConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		port        string
		description string
	}{
		{
			name:        "default monitoring port",
			port:        "2020",
			description: "Fluent Bit default monitoring port",
		},
		{
			name:        "custom monitoring port",
			port:        "2021",
			description: "Alternative port to avoid conflicts",
		},
		{
			name:        "high port",
			port:        "12020",
			description: "High port number for security",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultFluentBitProductionConfig()
			config.Port = tt.port

			assert.Equal(t, tt.port, config.Port, tt.description)
		})
	}
}

func TestFluentBitProductionConfig_ForwardPortConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		port        string
		description string
	}{
		{
			name:        "default forward port",
			port:        "24224",
			description: "Fluent Bit default forward protocol port",
		},
		{
			name:        "custom forward port",
			port:        "24225",
			description: "Alternative forward port to avoid conflicts",
		},
		{
			name:        "high forward port",
			port:        "34224",
			description: "High port number for forward protocol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultFluentBitProductionConfig()
			config.ForwardPort = tt.port

			assert.Equal(t, tt.port, config.ForwardPort, tt.description)
		})
	}
}

func TestFluentBitProductionConfig_ImageVersion(t *testing.T) {
	config := DefaultFluentBitProductionConfig()

	// Verify that Fluent Bit 4.0.13 image is being used
	assert.Contains(t, config.Image, "fluent/fluent-bit:4.0.13", "Should use Fluent Bit 4.0.13 version")
	assert.Equal(t, "fluent/fluent-bit:4.0.13-amd64", config.Image)
}

func TestFluentBitProductionConfig_NetworkIsolation(t *testing.T) {
	config := DefaultFluentBitProductionConfig()

	// Verify network configuration
	assert.NotEmpty(t, config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)

	// Test custom network
	config.Production.NetworkName = "fluentbit-network"
	assert.Equal(t, "fluentbit-network", config.Production.NetworkName)
}

func TestFluentBitProductionConfig_URLFormatting(t *testing.T) {
	config := DefaultFluentBitProductionConfig()

	// Test base URL
	baseURL := GetFluentBitURL(config)
	assert.True(t, len(baseURL) > 0, "Base URL should not be empty")
	assert.Contains(t, baseURL, "http://")
	assert.Contains(t, baseURL, config.Port)

	// Test health URL
	healthURL := GetFluentBitHealthURL(config)
	assert.True(t, len(healthURL) > 0, "Health URL should not be empty")
	assert.Contains(t, healthURL, config.Port)
	assert.Contains(t, healthURL, "/api/v1/health")

	// Test metrics URL
	metricsURL := GetFluentBitMetricsURL(config)
	assert.True(t, len(metricsURL) > 0, "Metrics URL should not be empty")
	assert.Contains(t, metricsURL, config.Port)
	assert.Contains(t, metricsURL, "/api/v1/metrics/prometheus")
}

func TestFluentBitProductionConfig_VolumeConfiguration(t *testing.T) {
	config := DefaultFluentBitProductionConfig()

	// Verify volume configuration
	assert.Equal(t, "fluentbit-config", config.ConfigVolume)
	assert.Equal(t, "fluentbit-data", config.DataVolume)
	assert.Equal(t, "fluentbit-data", config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)

	// Verify data volume and production settings match
	assert.Equal(t, config.DataVolume, config.Production.VolumeName,
		"DataVolume and Production.VolumeName should match")

	// Test both volumes are different
	assert.NotEqual(t, config.ConfigVolume, config.DataVolume,
		"ConfigVolume and DataVolume should be different")
}

func TestFluentBitProductionConfig_ContainerName(t *testing.T) {
	config := DefaultFluentBitProductionConfig()

	// Default container name
	assert.Equal(t, "fluentbit", config.ContainerName)

	// Custom container name
	config.ContainerName = "fluentbit-prod"
	assert.Equal(t, "fluentbit-prod", config.ContainerName)
}

func TestFluentBitProductionConfig_MultipleVolumes(t *testing.T) {
	config := DefaultFluentBitProductionConfig()

	// Verify both config and data volumes exist
	assert.NotEmpty(t, config.ConfigVolume, "Config volume should be configured")
	assert.NotEmpty(t, config.DataVolume, "Data volume should be configured")

	// Verify they are different
	assert.NotEqual(t, config.ConfigVolume, config.DataVolume,
		"Config and data volumes should be separate")

	// Test custom volumes
	config.ConfigVolume = "my-config"
	config.DataVolume = "my-data"
	assert.Equal(t, "my-config", config.ConfigVolume)
	assert.Equal(t, "my-data", config.DataVolume)
}

func TestFluentBitProductionConfig_DefaultPorts(t *testing.T) {
	config := DefaultFluentBitProductionConfig()

	// Verify Fluent Bit default ports
	assert.Equal(t, "2020", config.Port, "Monitoring port should be 2020")
	assert.Equal(t, "24224", config.ForwardPort, "Forward port should be 24224")
}

func TestFluentBitProductionConfig_PortURLConsistency(t *testing.T) {
	config := DefaultFluentBitProductionConfig()
	config.Port = "3020"

	// Verify all URL helpers use the correct port
	baseURL := GetFluentBitURL(config)
	healthURL := GetFluentBitHealthURL(config)
	metricsURL := GetFluentBitMetricsURL(config)

	assert.Contains(t, baseURL, "3020")
	assert.Contains(t, healthURL, "3020")
	assert.Contains(t, metricsURL, "3020")
}

func TestFluentBitProductionConfig_RestartPolicy(t *testing.T) {
	config := DefaultFluentBitProductionConfig()

	// The restart policy is set in DeployFluentBit to "unless-stopped"
	// This test just validates the config structure is ready
	assert.NotEmpty(t, config.ContainerName, "Container name is required for restart policy")
}

func TestFluentBitProductionConfig_ConfigVolumeSeparation(t *testing.T) {
	config := DefaultFluentBitProductionConfig()

	// Config volume should be separate from data volume
	assert.NotEmpty(t, config.ConfigVolume)
	assert.NotEmpty(t, config.DataVolume)
	assert.NotEqual(t, config.ConfigVolume, config.DataVolume,
		"Config and data should use separate volumes for better organization")
}

func TestFluentBitProductionConfig_ImageArchitecture(t *testing.T) {
	config := DefaultFluentBitProductionConfig()

	// Verify architecture-specific image
	assert.Contains(t, config.Image, "amd64", "Should explicitly specify amd64 architecture")
}

func TestFluentBitProductionConfig_MonitoringEndpoints(t *testing.T) {
	config := DefaultFluentBitProductionConfig()

	// Verify monitoring endpoints are properly formatted
	healthURL := GetFluentBitHealthURL(config)
	metricsURL := GetFluentBitMetricsURL(config)

	// Check health endpoint
	assert.Contains(t, healthURL, "/api/v1/health",
		"Health endpoint should follow Fluent Bit API path")

	// Check metrics endpoint
	assert.Contains(t, metricsURL, "/api/v1/metrics/prometheus",
		"Metrics endpoint should follow Fluent Bit API path for Prometheus")
}

func TestFluentBitProductionConfig_AllPortsConfigurable(t *testing.T) {
	config := DefaultFluentBitProductionConfig()

	// Verify both ports are independently configurable
	config.Port = "9999"
	config.ForwardPort = "8888"

	assert.Equal(t, "9999", config.Port)
	assert.Equal(t, "8888", config.ForwardPort)
	assert.NotEqual(t, config.Port, config.ForwardPort,
		"Monitoring and forward ports should be independently configurable")
}

func TestFluentBitProductionConfig_VolumeNames(t *testing.T) {
	tests := []struct {
		name         string
		configVolume string
		dataVolume   string
		valid        bool
	}{
		{
			name:         "default volumes",
			configVolume: "fluentbit-config",
			dataVolume:   "fluentbit-data",
			valid:        true,
		},
		{
			name:         "custom volumes",
			configVolume: "my-config",
			dataVolume:   "my-data",
			valid:        true,
		},
		{
			name:         "environment-specific volumes",
			configVolume: "prod-fluentbit-config",
			dataVolume:   "prod-fluentbit-data",
			valid:        true,
		},
		{
			name:         "versioned volumes",
			configVolume: "fluentbit-config-v1",
			dataVolume:   "fluentbit-data-v1",
			valid:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultFluentBitProductionConfig()
			config.ConfigVolume = tt.configVolume
			config.DataVolume = tt.dataVolume

			assert.Equal(t, tt.configVolume, config.ConfigVolume)
			assert.Equal(t, tt.dataVolume, config.DataVolume)
			if tt.valid {
				assert.NotEmpty(t, config.ConfigVolume)
				assert.NotEmpty(t, config.DataVolume)
			}
		})
	}
}
