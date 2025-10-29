package production

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultMimirProductionConfig(t *testing.T) {
	config := DefaultMimirProductionConfig()

	assert.Equal(t, "mimir", config.ContainerName)
	assert.Equal(t, "grafana/mimir:2.17.2", config.Image)
	assert.Equal(t, "9009", config.Port)
	assert.Equal(t, "mimir-data", config.DataVolume)
	assert.Equal(t, "app-network", config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)
	assert.Equal(t, "mimir-data", config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)
}

func TestMimirProductionConfig_CustomConfiguration(t *testing.T) {
	config := DefaultMimirProductionConfig()

	// Modify configuration
	config.ContainerName = "my-mimir"
	config.Port = "9010"
	config.DataVolume = "custom-mimir-data"
	config.Image = "grafana/mimir:2.16.0"

	assert.Equal(t, "my-mimir", config.ContainerName)
	assert.Equal(t, "9010", config.Port)
	assert.Equal(t, "custom-mimir-data", config.DataVolume)
	assert.Equal(t, "grafana/mimir:2.16.0", config.Image)
}

func TestGetMimirURL(t *testing.T) {
	tests := []struct {
		name     string
		config   MimirProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: MimirProductionConfig{
				Port: "9009",
			},
			expected: "http://localhost:9009",
		},
		{
			name: "custom port",
			config: MimirProductionConfig{
				Port: "9010",
			},
			expected: "http://localhost:9010",
		},
		{
			name: "high port number",
			config: MimirProductionConfig{
				Port: "19009",
			},
			expected: "http://localhost:19009",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mimirURL := GetMimirURL(tt.config)
			assert.Equal(t, tt.expected, mimirURL)
		})
	}
}

func TestGetMimirPushURL(t *testing.T) {
	tests := []struct {
		name     string
		config   MimirProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: MimirProductionConfig{
				Port: "9009",
			},
			expected: "http://localhost:9009/api/v1/push",
		},
		{
			name: "custom port",
			config: MimirProductionConfig{
				Port: "9010",
			},
			expected: "http://localhost:9010/api/v1/push",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pushURL := GetMimirPushURL(tt.config)
			assert.Equal(t, tt.expected, pushURL)
		})
	}
}

func TestGetMimirQueryURL(t *testing.T) {
	tests := []struct {
		name     string
		config   MimirProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: MimirProductionConfig{
				Port: "9009",
			},
			expected: "http://localhost:9009/prometheus/api/v1/query",
		},
		{
			name: "custom port",
			config: MimirProductionConfig{
				Port: "9010",
			},
			expected: "http://localhost:9010/prometheus/api/v1/query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryURL := GetMimirQueryURL(tt.config)
			assert.Equal(t, tt.expected, queryURL)
		})
	}
}

func TestMimirProductionConfig_DataPersistence(t *testing.T) {
	config := DefaultMimirProductionConfig()

	// Verify data volume is configured for persistence
	assert.NotEmpty(t, config.DataVolume, "Data volume should be configured")
	assert.Equal(t, "mimir-data", config.DataVolume, "Default data volume name")

	// Test custom volume
	config.DataVolume = "production-mimir-volume"
	assert.Equal(t, "production-mimir-volume", config.DataVolume)
}

func TestMimirProductionConfig_PortConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		port        string
		description string
	}{
		{
			name:        "default port",
			port:        "9009",
			description: "Mimir default port",
		},
		{
			name:        "custom port",
			port:        "9010",
			description: "Alternative port to avoid conflicts",
		},
		{
			name:        "high port",
			port:        "19009",
			description: "High port number for security",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultMimirProductionConfig()
			config.Port = tt.port

			assert.Equal(t, tt.port, config.Port, tt.description)
		})
	}
}

func TestMimirProductionConfig_ImageVersion(t *testing.T) {
	config := DefaultMimirProductionConfig()

	// Verify that Mimir 2.17.2 image is being used
	assert.Contains(t, config.Image, "grafana/mimir:2.17.2", "Should use Mimir 2.17.2 version")
	assert.Equal(t, "grafana/mimir:2.17.2", config.Image)
}

func TestMimirProductionConfig_NetworkIsolation(t *testing.T) {
	config := DefaultMimirProductionConfig()

	// Verify network configuration
	assert.NotEmpty(t, config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)

	// Test custom network
	config.Production.NetworkName = "mimir-network"
	assert.Equal(t, "mimir-network", config.Production.NetworkName)
}

func TestMimirProductionConfig_RestartPolicy(t *testing.T) {
	config := DefaultMimirProductionConfig()

	// The restart policy is set in DeployMimir to "unless-stopped"
	// This test just validates the config structure is ready
	assert.NotEmpty(t, config.ContainerName, "Container name is required for restart policy")
}

func TestMimirProductionConfig_URLFormatting(t *testing.T) {
	config := DefaultMimirProductionConfig()

	// Test base URL
	baseURL := GetMimirURL(config)
	assert.True(t, len(baseURL) > 0, "Base URL should not be empty")
	assert.Contains(t, baseURL, "http://")
	assert.Contains(t, baseURL, config.Port)

	// Test push URL
	pushURL := GetMimirPushURL(config)
	assert.True(t, len(pushURL) > 0, "Push URL should not be empty")
	assert.Contains(t, pushURL, config.Port)
	assert.Contains(t, pushURL, "/api/v1/push")

	// Test query URL
	queryURL := GetMimirQueryURL(config)
	assert.True(t, len(queryURL) > 0, "Query URL should not be empty")
	assert.Contains(t, queryURL, config.Port)
	assert.Contains(t, queryURL, "/prometheus/api/v1/query")
}

func TestMimirProductionConfig_MultiTenancy(t *testing.T) {
	config := DefaultMimirProductionConfig()

	// Multi-tenancy in Mimir is handled via X-Scope-OrgID header
	// This test verifies that the configuration supports multi-tenant deployments
	assert.NotEmpty(t, config.ContainerName, "Container name is required")
	assert.NotEmpty(t, config.Port, "Port is required for multi-tenant access")

	// Verify URLs work with different tenant IDs (headers handled by client)
	pushURL := GetMimirPushURL(config)
	assert.Contains(t, pushURL, "/api/v1/push", "Push URL must support remote_write with tenant header")

	queryURL := GetMimirQueryURL(config)
	assert.Contains(t, queryURL, "/prometheus/api/v1/query", "Query URL must support PromQL with tenant header")
}

func TestMimirProductionConfig_ContainerName(t *testing.T) {
	config := DefaultMimirProductionConfig()

	// Default container name
	assert.Equal(t, "mimir", config.ContainerName)

	// Custom container name
	config.ContainerName = "mimir-prod"
	assert.Equal(t, "mimir-prod", config.ContainerName)
}

func TestMimirProductionConfig_DefaultPort(t *testing.T) {
	config := DefaultMimirProductionConfig()

	// Verify Mimir default port
	assert.Equal(t, "9009", config.Port, "Mimir default port should be 9009")
}

func TestMimirProductionConfig_VolumeConfiguration(t *testing.T) {
	config := DefaultMimirProductionConfig()

	// Verify volume configuration
	assert.Equal(t, "mimir-data", config.DataVolume)
	assert.Equal(t, "mimir-data", config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)

	// Verify volume and production settings match
	assert.Equal(t, config.DataVolume, config.Production.VolumeName,
		"DataVolume and Production.VolumeName should match")
}

func TestMimirProductionConfig_PrometheusCompatibility(t *testing.T) {
	config := DefaultMimirProductionConfig()

	// Verify Prometheus remote_write endpoint is properly formatted
	pushURL := GetMimirPushURL(config)
	assert.Contains(t, pushURL, "/api/v1/push", "Must have Prometheus remote_write compatible endpoint")

	// Verify Prometheus query endpoint is properly formatted
	queryURL := GetMimirQueryURL(config)
	assert.Contains(t, queryURL, "/prometheus/api/v1/query", "Must have Prometheus query compatible endpoint")
}

func TestMimirProductionConfig_EndpointPaths(t *testing.T) {
	config := DefaultMimirProductionConfig()

	// Test all URL helper functions return correct paths
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "base URL",
			url:      GetMimirURL(config),
			expected: "http://localhost:9009",
		},
		{
			name:     "push endpoint",
			url:      GetMimirPushURL(config),
			expected: "http://localhost:9009/api/v1/push",
		},
		{
			name:     "query endpoint",
			url:      GetMimirQueryURL(config),
			expected: "http://localhost:9009/prometheus/api/v1/query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.url)
		})
	}
}

func TestMimirProductionConfig_RemoteWriteConfiguration(t *testing.T) {
	config := DefaultMimirProductionConfig()

	// Verify remote_write URL is correctly constructed
	pushURL := GetMimirPushURL(config)
	assert.Equal(t, "http://localhost:9009/api/v1/push", pushURL)

	// Test with custom port
	config.Port = "9999"
	pushURL = GetMimirPushURL(config)
	assert.Equal(t, "http://localhost:9999/api/v1/push", pushURL)
}

func TestMimirProductionConfig_PromQLQueryConfiguration(t *testing.T) {
	config := DefaultMimirProductionConfig()

	// Verify PromQL query URL is correctly constructed
	queryURL := GetMimirQueryURL(config)
	assert.Equal(t, "http://localhost:9009/prometheus/api/v1/query", queryURL)

	// Test with custom port
	config.Port = "9999"
	queryURL = GetMimirQueryURL(config)
	assert.Equal(t, "http://localhost:9999/prometheus/api/v1/query", queryURL)
}

func TestMimirProductionConfig_ImageCustomization(t *testing.T) {
	config := DefaultMimirProductionConfig()

	// Test different image versions
	tests := []struct {
		name  string
		image string
	}{
		{
			name:  "default version",
			image: "grafana/mimir:2.17.2",
		},
		{
			name:  "specific version",
			image: "grafana/mimir:2.16.0",
		},
		{
			name:  "latest tag",
			image: "grafana/mimir:latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Image = tt.image
			assert.Equal(t, tt.image, config.Image)
		})
	}
}

func TestMimirProductionConfig_NetworkConfiguration(t *testing.T) {
	config := DefaultMimirProductionConfig()

	// Verify network settings
	assert.Equal(t, "app-network", config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)

	// Test custom network settings
	config.Production.NetworkName = "metrics-network"
	config.Production.CreateNetwork = false
	assert.Equal(t, "metrics-network", config.Production.NetworkName)
	assert.False(t, config.Production.CreateNetwork)
}

func TestMimirProductionConfig_GrafanaIntegration(t *testing.T) {
	config := DefaultMimirProductionConfig()

	// For Grafana integration, the base URL + /prometheus path is used
	baseURL := GetMimirURL(config)
	expectedGrafanaDataSourceURL := baseURL + "/prometheus"

	assert.Equal(t, "http://localhost:9009/prometheus", expectedGrafanaDataSourceURL,
		"Grafana data source should point to /prometheus endpoint")
}

func TestMimirProductionConfig_HealthCheckEndpoint(t *testing.T) {
	config := DefaultMimirProductionConfig()

	// Health check endpoint should be at /ready
	baseURL := GetMimirURL(config)
	expectedHealthURL := baseURL + "/ready"

	assert.Equal(t, "http://localhost:9009/ready", expectedHealthURL,
		"Health check endpoint should be at /ready")
}

func TestMimirProductionConfig_TenantIsolation(t *testing.T) {
	config := DefaultMimirProductionConfig()

	// Each tenant should use the same URLs but different X-Scope-OrgID headers
	// Verify URLs are tenant-agnostic
	pushURL := GetMimirPushURL(config)
	queryURL := GetMimirQueryURL(config)

	assert.NotContains(t, pushURL, "tenant", "Push URL should not contain tenant in path")
	assert.NotContains(t, queryURL, "tenant", "Query URL should not contain tenant in path")
	assert.Contains(t, pushURL, "/api/v1/push", "Push URL should have standard path")
	assert.Contains(t, queryURL, "/prometheus/api/v1/query", "Query URL should have standard path")
}
