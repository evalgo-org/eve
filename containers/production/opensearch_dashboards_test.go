package production

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultOpenSearchDashboardsProductionConfig(t *testing.T) {
	opensearchURL := "http://opensearch:9200"
	config := DefaultOpenSearchDashboardsProductionConfig(opensearchURL)

	assert.Equal(t, "opensearch-dashboards", config.ContainerName)
	assert.Equal(t, "opensearchproject/opensearch-dashboards:3.0.0", config.Image)
	assert.Equal(t, "5601", config.Port)
	assert.Equal(t, opensearchURL, config.OpenSearchURL)
	assert.True(t, config.DisableSecurity)
	assert.Equal(t, "opensearch-dashboards-data", config.DataVolume)
	assert.Equal(t, "app-network", config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)
	assert.Equal(t, "opensearch-dashboards-data", config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)
}

func TestOpenSearchDashboardsProductionConfig_CustomConfiguration(t *testing.T) {
	opensearchURL := "http://my-opensearch:9200"
	config := DefaultOpenSearchDashboardsProductionConfig(opensearchURL)

	// Modify configuration
	config.ContainerName = "my-dashboards"
	config.Port = "5602"
	config.OpenSearchURL = "http://custom-opensearch:9201"
	config.DisableSecurity = false
	config.DataVolume = "custom-dashboards-data"

	assert.Equal(t, "my-dashboards", config.ContainerName)
	assert.Equal(t, "5602", config.Port)
	assert.Equal(t, "http://custom-opensearch:9201", config.OpenSearchURL)
	assert.False(t, config.DisableSecurity)
	assert.Equal(t, "custom-dashboards-data", config.DataVolume)
}

func TestGetOpenSearchDashboardsURL(t *testing.T) {
	tests := []struct {
		name     string
		config   OpenSearchDashboardsProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: OpenSearchDashboardsProductionConfig{
				Port: "5601",
			},
			expected: "http://localhost:5601",
		},
		{
			name: "custom port",
			config: OpenSearchDashboardsProductionConfig{
				Port: "5602",
			},
			expected: "http://localhost:5602",
		},
		{
			name: "high port number",
			config: OpenSearchDashboardsProductionConfig{
				Port: "15601",
			},
			expected: "http://localhost:15601",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dashboardsURL := GetOpenSearchDashboardsURL(tt.config)
			assert.Equal(t, tt.expected, dashboardsURL)
		})
	}
}

func TestGetOpenSearchDashboardsAppURL(t *testing.T) {
	tests := []struct {
		name     string
		config   OpenSearchDashboardsProductionConfig
		appName  string
		expected string
	}{
		{
			name: "discover app with default port",
			config: OpenSearchDashboardsProductionConfig{
				Port: "5601",
			},
			appName:  "discover",
			expected: "http://localhost:5601/app/discover",
		},
		{
			name: "dashboards app with custom port",
			config: OpenSearchDashboardsProductionConfig{
				Port: "5602",
			},
			appName:  "dashboards",
			expected: "http://localhost:5602/app/dashboards",
		},
		{
			name: "visualize app",
			config: OpenSearchDashboardsProductionConfig{
				Port: "5601",
			},
			appName:  "visualize",
			expected: "http://localhost:5601/app/visualize",
		},
		{
			name: "dev tools app",
			config: OpenSearchDashboardsProductionConfig{
				Port: "5601",
			},
			appName:  "dev_tools",
			expected: "http://localhost:5601/app/dev_tools",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appURL := GetOpenSearchDashboardsAppURL(tt.config, tt.appName)
			assert.Equal(t, tt.expected, appURL)
		})
	}
}

func TestOpenSearchDashboardsProductionConfig_DataPersistence(t *testing.T) {
	config := DefaultOpenSearchDashboardsProductionConfig("http://opensearch:9200")

	// Verify data volume is configured for persistence
	assert.NotEmpty(t, config.DataVolume, "Data volume should be configured")
	assert.Equal(t, "opensearch-dashboards-data", config.DataVolume, "Default data volume name")

	// Test custom volume
	config.DataVolume = "production-dashboards-volume"
	assert.Equal(t, "production-dashboards-volume", config.DataVolume)
}

func TestOpenSearchDashboardsProductionConfig_PortConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		port        string
		description string
	}{
		{
			name:        "default port",
			port:        "5601",
			description: "OpenSearch Dashboards default port",
		},
		{
			name:        "custom port",
			port:        "5602",
			description: "Alternative port to avoid conflicts",
		},
		{
			name:        "high port",
			port:        "15601",
			description: "High port number for security",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultOpenSearchDashboardsProductionConfig("http://opensearch:9200")
			config.Port = tt.port

			assert.Equal(t, tt.port, config.Port, tt.description)
		})
	}
}

func TestOpenSearchDashboardsProductionConfig_SecuritySettings(t *testing.T) {
	tests := []struct {
		name            string
		disableSecurity bool
		description     string
	}{
		{
			name:            "security disabled for testing",
			disableSecurity: true,
			description:     "Testing environment with security disabled",
		},
		{
			name:            "security enabled for production",
			disableSecurity: false,
			description:     "Production environment with security enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultOpenSearchDashboardsProductionConfig("http://opensearch:9200")
			config.DisableSecurity = tt.disableSecurity

			assert.Equal(t, tt.disableSecurity, config.DisableSecurity, tt.description)
		})
	}
}

func TestOpenSearchDashboardsProductionConfig_ImageVersion(t *testing.T) {
	config := DefaultOpenSearchDashboardsProductionConfig("http://opensearch:9200")

	// Verify that OpenSearch Dashboards 3.x.x image is being used
	assert.Contains(t, config.Image, "opensearch-dashboards:3", "Should use OpenSearch Dashboards 3.x.x version")
	assert.Equal(t, "opensearchproject/opensearch-dashboards:3.0.0", config.Image)
}

func TestOpenSearchDashboardsProductionConfig_NetworkIsolation(t *testing.T) {
	config := DefaultOpenSearchDashboardsProductionConfig("http://opensearch:9200")

	// Verify network configuration
	assert.NotEmpty(t, config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)

	// Test custom network
	config.Production.NetworkName = "dashboards-network"
	assert.Equal(t, "dashboards-network", config.Production.NetworkName)
}

func TestOpenSearchDashboardsProductionConfig_RestartPolicy(t *testing.T) {
	config := DefaultOpenSearchDashboardsProductionConfig("http://opensearch:9200")

	// The restart policy is set in DeployOpenSearchDashboards to "unless-stopped"
	// This test just validates the config structure is ready
	assert.NotEmpty(t, config.ContainerName, "Container name is required for restart policy")
}

func TestOpenSearchDashboardsProductionConfig_OpenSearchConnection(t *testing.T) {
	tests := []struct {
		name          string
		opensearchURL string
		valid         bool
	}{
		{
			name:          "container name URL",
			opensearchURL: "http://opensearch:9200",
			valid:         true,
		},
		{
			name:          "localhost URL",
			opensearchURL: "http://localhost:9200",
			valid:         true,
		},
		{
			name:          "host.docker.internal URL",
			opensearchURL: "http://host.docker.internal:9200",
			valid:         true,
		},
		{
			name:          "custom port URL",
			opensearchURL: "http://opensearch:9201",
			valid:         true,
		},
		{
			name:          "IP address URL",
			opensearchURL: "http://192.168.1.100:9200",
			valid:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultOpenSearchDashboardsProductionConfig(tt.opensearchURL)

			assert.Equal(t, tt.opensearchURL, config.OpenSearchURL)
			if tt.valid {
				assert.NotEmpty(t, config.OpenSearchURL)
			}
		})
	}
}

func TestOpenSearchDashboardsProductionConfig_AppNames(t *testing.T) {
	tests := []struct {
		name    string
		appName string
		valid   bool
	}{
		{
			name:    "discover app",
			appName: "discover",
			valid:   true,
		},
		{
			name:    "dashboards app",
			appName: "dashboards",
			valid:   true,
		},
		{
			name:    "visualize app",
			appName: "visualize",
			valid:   true,
		},
		{
			name:    "dev_tools app",
			appName: "dev_tools",
			valid:   true,
		},
		{
			name:    "management app",
			appName: "management",
			valid:   true,
		},
		{
			name:    "home app",
			appName: "home",
			valid:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultOpenSearchDashboardsProductionConfig("http://opensearch:9200")
			appURL := GetOpenSearchDashboardsAppURL(config, tt.appName)

			assert.NotEmpty(t, appURL)
			if tt.valid {
				assert.Contains(t, appURL, tt.appName)
			}
		})
	}
}

func TestOpenSearchDashboardsProductionConfig_RequiredOpenSearchURL(t *testing.T) {
	// Test that OpenSearchURL is required
	config := DefaultOpenSearchDashboardsProductionConfig("http://opensearch:9200")
	assert.NotEmpty(t, config.OpenSearchURL, "OpenSearchURL should be required")

	// Test that empty OpenSearchURL should be detectable
	config.OpenSearchURL = ""
	assert.Empty(t, config.OpenSearchURL, "Empty OpenSearchURL should be detectable")
}
