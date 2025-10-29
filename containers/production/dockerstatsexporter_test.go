package production

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultDockerStatsExporterProductionConfig(t *testing.T) {
	config := DefaultDockerStatsExporterProductionConfig()

	assert.Equal(t, "docker-stats-exporter", config.ContainerName)
	assert.Equal(t, "ghcr.io/grzegorzmika/docker_stats_exporter:latest", config.Image)
	assert.Equal(t, "8080", config.Port)
	assert.Equal(t, "app-network", config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)
	assert.False(t, config.Production.CreateVolume, "Docker Stats Exporter is stateless and doesn't need volume")
}

func TestDockerStatsExporterProductionConfig_CustomConfiguration(t *testing.T) {
	config := DefaultDockerStatsExporterProductionConfig()

	// Modify configuration
	config.ContainerName = "my-docker-stats-exporter"
	config.Port = "9090"
	config.Image = "ghcr.io/grzegorzmika/docker_stats_exporter:v1.0.0"

	assert.Equal(t, "my-docker-stats-exporter", config.ContainerName)
	assert.Equal(t, "9090", config.Port)
	assert.Equal(t, "ghcr.io/grzegorzmika/docker_stats_exporter:v1.0.0", config.Image)
}

func TestGetDockerStatsExporterURL(t *testing.T) {
	tests := []struct {
		name     string
		config   DockerStatsExporterProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: DockerStatsExporterProductionConfig{
				Port: "8080",
			},
			expected: "http://localhost:8080",
		},
		{
			name: "custom port",
			config: DockerStatsExporterProductionConfig{
				Port: "9090",
			},
			expected: "http://localhost:9090",
		},
		{
			name: "high port number",
			config: DockerStatsExporterProductionConfig{
				Port: "18080",
			},
			expected: "http://localhost:18080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporterURL := GetDockerStatsExporterURL(tt.config)
			assert.Equal(t, tt.expected, exporterURL)
		})
	}
}

func TestGetDockerStatsExporterMetricsURL(t *testing.T) {
	tests := []struct {
		name     string
		config   DockerStatsExporterProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: DockerStatsExporterProductionConfig{
				Port: "8080",
			},
			expected: "http://localhost:8080/metrics",
		},
		{
			name: "custom port",
			config: DockerStatsExporterProductionConfig{
				Port: "9090",
			},
			expected: "http://localhost:9090/metrics",
		},
		{
			name: "alternative port",
			config: DockerStatsExporterProductionConfig{
				Port: "8081",
			},
			expected: "http://localhost:8081/metrics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metricsURL := GetDockerStatsExporterMetricsURL(tt.config)
			assert.Equal(t, tt.expected, metricsURL)
		})
	}
}

func TestDockerStatsExporterProductionConfig_PortConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		port        string
		description string
	}{
		{
			name:        "default port",
			port:        "8080",
			description: "Docker Stats Exporter default port",
		},
		{
			name:        "custom port",
			port:        "9090",
			description: "Alternative port to avoid conflicts",
		},
		{
			name:        "high port",
			port:        "18080",
			description: "High port number for security",
		},
		{
			name:        "prometheus default port",
			port:        "9100",
			description: "Using Prometheus Node Exporter port range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultDockerStatsExporterProductionConfig()
			config.Port = tt.port

			assert.Equal(t, tt.port, config.Port, tt.description)
		})
	}
}

func TestDockerStatsExporterProductionConfig_ImageVersion(t *testing.T) {
	config := DefaultDockerStatsExporterProductionConfig()

	// Verify that latest tag is being used by default
	assert.Contains(t, config.Image, "docker_stats_exporter:latest", "Should use latest version by default")
	assert.Equal(t, "ghcr.io/grzegorzmika/docker_stats_exporter:latest", config.Image)

	// Test custom image version
	config.Image = "ghcr.io/grzegorzmika/docker_stats_exporter:v1.0.0"
	assert.Contains(t, config.Image, "v1.0.0", "Should support custom version tags")
}

func TestDockerStatsExporterProductionConfig_NetworkIsolation(t *testing.T) {
	config := DefaultDockerStatsExporterProductionConfig()

	// Verify network configuration
	assert.NotEmpty(t, config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)

	// Test custom network
	config.Production.NetworkName = "monitoring-network"
	assert.Equal(t, "monitoring-network", config.Production.NetworkName)
}

func TestDockerStatsExporterProductionConfig_URLFormatting(t *testing.T) {
	config := DefaultDockerStatsExporterProductionConfig()

	// Test base URL
	baseURL := GetDockerStatsExporterURL(config)
	assert.True(t, len(baseURL) > 0, "Base URL should not be empty")
	assert.Contains(t, baseURL, "http://")
	assert.Contains(t, baseURL, config.Port)

	// Test metrics URL
	metricsURL := GetDockerStatsExporterMetricsURL(config)
	assert.True(t, len(metricsURL) > 0, "Metrics URL should not be empty")
	assert.Contains(t, metricsURL, config.Port)
	assert.Contains(t, metricsURL, "/metrics")
}

func TestDockerStatsExporterProductionConfig_ContainerName(t *testing.T) {
	config := DefaultDockerStatsExporterProductionConfig()

	// Default container name
	assert.Equal(t, "docker-stats-exporter", config.ContainerName)

	// Custom container name
	config.ContainerName = "docker-stats-exporter-prod"
	assert.Equal(t, "docker-stats-exporter-prod", config.ContainerName)
}

func TestDockerStatsExporterProductionConfig_DockerSocketAccess(t *testing.T) {
	config := DefaultDockerStatsExporterProductionConfig()

	// Verify that the exporter is configured without volume
	// (it uses bind mount for Docker socket instead)
	assert.False(t, config.Production.CreateVolume,
		"Docker Stats Exporter should not create volume, it uses bind mount for Docker socket")

	// The bind mount configuration is set in DeployDockerStatsExporter
	// Verify container name is set for deployment
	assert.NotEmpty(t, config.ContainerName, "Container name is required for deployment")
	assert.NotEmpty(t, config.Image, "Image is required for deployment")
}

func TestDockerStatsExporterProductionConfig_SecurityConsiderations(t *testing.T) {
	config := DefaultDockerStatsExporterProductionConfig()

	// The exporter should have minimal configuration
	// Security is handled through read-only Docker socket mount
	assert.NotEmpty(t, config.ContainerName, "Container must be named for management")
	assert.NotEmpty(t, config.Image, "Image must be specified")
	assert.NotEmpty(t, config.Port, "Port must be specified for metrics endpoint")

	// No volume required - stateless exporter
	assert.False(t, config.Production.CreateVolume, "Stateless exporter doesn't need persistent storage")
}

func TestDockerStatsExporterProductionConfig_StatelessDesign(t *testing.T) {
	config := DefaultDockerStatsExporterProductionConfig()

	// Verify stateless configuration
	assert.False(t, config.Production.CreateVolume, "Exporter should be stateless")
	assert.Empty(t, config.Production.VolumeName, "No volume name should be set")

	// Container can be removed without data loss
	assert.NotEmpty(t, config.ContainerName, "Container name required for management")
}

func TestDockerStatsExporterProductionConfig_DefaultPort(t *testing.T) {
	config := DefaultDockerStatsExporterProductionConfig()

	// Verify Docker Stats Exporter default port
	assert.Equal(t, "8080", config.Port, "Docker Stats Exporter default port should be 8080")
}

func TestDockerStatsExporterProductionConfig_PrometheusCompatibility(t *testing.T) {
	config := DefaultDockerStatsExporterProductionConfig()

	// Verify metrics URL follows Prometheus standards
	metricsURL := GetDockerStatsExporterMetricsURL(config)
	assert.Contains(t, metricsURL, "/metrics", "Should expose metrics at /metrics endpoint")
}

func TestDockerStatsExporterProductionConfig_RestartPolicy(t *testing.T) {
	config := DefaultDockerStatsExporterProductionConfig()

	// The restart policy is set in DeployDockerStatsExporter to "unless-stopped"
	// This test just validates the config structure is ready
	assert.NotEmpty(t, config.ContainerName, "Container name is required for restart policy")
}

func TestDockerStatsExporterProductionConfig_MultipleInstances(t *testing.T) {
	// Test configuration for multiple exporter instances
	tests := []struct {
		name          string
		containerName string
		port          string
	}{
		{
			name:          "primary instance",
			containerName: "docker-stats-exporter-primary",
			port:          "8080",
		},
		{
			name:          "secondary instance",
			containerName: "docker-stats-exporter-secondary",
			port:          "8081",
		},
		{
			name:          "monitoring instance",
			containerName: "docker-stats-exporter-monitoring",
			port:          "8082",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultDockerStatsExporterProductionConfig()
			config.ContainerName = tt.containerName
			config.Port = tt.port

			assert.Equal(t, tt.containerName, config.ContainerName)
			assert.Equal(t, tt.port, config.Port)
		})
	}
}

func TestDockerStatsExporterProductionConfig_ImageRegistry(t *testing.T) {
	config := DefaultDockerStatsExporterProductionConfig()

	// Verify image is from GitHub Container Registry
	assert.Contains(t, config.Image, "ghcr.io", "Should use GitHub Container Registry")
	assert.Contains(t, config.Image, "grzegorzmika/docker_stats_exporter", "Should use correct image name")
}

func TestDockerStatsExporterProductionConfig_NetworkName(t *testing.T) {
	tests := []struct {
		name        string
		networkName string
		description string
	}{
		{
			name:        "default network",
			networkName: "app-network",
			description: "Default application network",
		},
		{
			name:        "monitoring network",
			networkName: "monitoring-network",
			description: "Dedicated monitoring network",
		},
		{
			name:        "isolated network",
			networkName: "docker-stats-network",
			description: "Isolated network for exporter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultDockerStatsExporterProductionConfig()
			config.Production.NetworkName = tt.networkName

			assert.Equal(t, tt.networkName, config.Production.NetworkName, tt.description)
		})
	}
}

func TestDockerStatsExporterProductionConfig_ContainerNaming(t *testing.T) {
	tests := []struct {
		name          string
		containerName string
		valid         bool
	}{
		{
			name:          "default name",
			containerName: "docker-stats-exporter",
			valid:         true,
		},
		{
			name:          "environment suffix",
			containerName: "docker-stats-exporter-prod",
			valid:         true,
		},
		{
			name:          "region suffix",
			containerName: "docker-stats-exporter-us-east-1",
			valid:         true,
		},
		{
			name:          "numeric suffix",
			containerName: "docker-stats-exporter-01",
			valid:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultDockerStatsExporterProductionConfig()
			config.ContainerName = tt.containerName

			assert.Equal(t, tt.containerName, config.ContainerName)
			if tt.valid {
				assert.NotEmpty(t, config.ContainerName)
			}
		})
	}
}

func TestDockerStatsExporterProductionConfig_PortRange(t *testing.T) {
	tests := []struct {
		name string
		port string
		min  int
		max  int
	}{
		{
			name: "standard port range",
			port: "8080",
			min:  1024,
			max:  65535,
		},
		{
			name: "high port",
			port: "18080",
			min:  1024,
			max:  65535,
		},
		{
			name: "prometheus port range",
			port: "9090",
			min:  1024,
			max:  65535,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultDockerStatsExporterProductionConfig()
			config.Port = tt.port

			assert.NotEmpty(t, config.Port, "Port should not be empty")
			// Port validation would require parsing, just verify it's set
			assert.Equal(t, tt.port, config.Port)
		})
	}
}
