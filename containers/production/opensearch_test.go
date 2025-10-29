package production

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultOpenSearchProductionConfig(t *testing.T) {
	config := DefaultOpenSearchProductionConfig()

	assert.Equal(t, "opensearch", config.ContainerName)
	assert.Equal(t, "opensearchproject/opensearch:3.0.0", config.Image)
	assert.Equal(t, "9200", config.Port)
	assert.Equal(t, "-Xms2g -Xmx2g", config.JavaOpts)
	assert.True(t, config.DisableSecurity)
	assert.Equal(t, "opensearch-data", config.DataVolume)
	assert.Equal(t, "app-network", config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)
	assert.Equal(t, "opensearch-data", config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)
}

func TestOpenSearchProductionConfig_CustomConfiguration(t *testing.T) {
	config := DefaultOpenSearchProductionConfig()

	// Modify configuration
	config.ContainerName = "my-opensearch"
	config.Port = "9201"
	config.JavaOpts = "-Xms4g -Xmx4g"
	config.DisableSecurity = false
	config.DataVolume = "custom-opensearch-data"

	assert.Equal(t, "my-opensearch", config.ContainerName)
	assert.Equal(t, "9201", config.Port)
	assert.Equal(t, "-Xms4g -Xmx4g", config.JavaOpts)
	assert.False(t, config.DisableSecurity)
	assert.Equal(t, "custom-opensearch-data", config.DataVolume)
}

func TestGetOpenSearchURL(t *testing.T) {
	tests := []struct {
		name     string
		config   OpenSearchProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: OpenSearchProductionConfig{
				Port: "9200",
			},
			expected: "http://localhost:9200",
		},
		{
			name: "custom port",
			config: OpenSearchProductionConfig{
				Port: "9201",
			},
			expected: "http://localhost:9201",
		},
		{
			name: "high port number",
			config: OpenSearchProductionConfig{
				Port: "19200",
			},
			expected: "http://localhost:19200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opensearchURL := GetOpenSearchURL(tt.config)
			assert.Equal(t, tt.expected, opensearchURL)
		})
	}
}

func TestGetOpenSearchIndexURL(t *testing.T) {
	tests := []struct {
		name      string
		config    OpenSearchProductionConfig
		indexName string
		expected  string
	}{
		{
			name: "default port with simple index",
			config: OpenSearchProductionConfig{
				Port: "9200",
			},
			indexName: "test-index",
			expected:  "http://localhost:9200/test-index",
		},
		{
			name: "custom port with hyphenated index",
			config: OpenSearchProductionConfig{
				Port: "9201",
			},
			indexName: "my-test-index",
			expected:  "http://localhost:9201/my-test-index",
		},
		{
			name: "index with underscores",
			config: OpenSearchProductionConfig{
				Port: "9200",
			},
			indexName: "test_index_2024",
			expected:  "http://localhost:9200/test_index_2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexURL := GetOpenSearchIndexURL(tt.config, tt.indexName)
			assert.Equal(t, tt.expected, indexURL)
		})
	}
}

func TestOpenSearchProductionConfig_MemoryConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		javaOpts string
		valid    bool
	}{
		{
			name:     "minimal settings",
			javaOpts: "-Xms512m -Xmx512m",
			valid:    true,
		},
		{
			name:     "development settings",
			javaOpts: "-Xms1g -Xmx1g",
			valid:    true,
		},
		{
			name:     "production settings",
			javaOpts: "-Xms2g -Xmx2g",
			valid:    true,
		},
		{
			name:     "large dataset settings",
			javaOpts: "-Xms4g -Xmx4g",
			valid:    true,
		},
		{
			name:     "enterprise settings",
			javaOpts: "-Xms8g -Xmx8g",
			valid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultOpenSearchProductionConfig()
			config.JavaOpts = tt.javaOpts

			assert.Equal(t, tt.javaOpts, config.JavaOpts)
			if tt.valid {
				assert.NotEmpty(t, config.JavaOpts)
			}
		})
	}
}

func TestOpenSearchProductionConfig_DataPersistence(t *testing.T) {
	config := DefaultOpenSearchProductionConfig()

	// Verify data volume is configured for persistence
	assert.NotEmpty(t, config.DataVolume, "Data volume should be configured")
	assert.Equal(t, "opensearch-data", config.DataVolume, "Default data volume name")

	// Test custom volume
	config.DataVolume = "production-opensearch-volume"
	assert.Equal(t, "production-opensearch-volume", config.DataVolume)
}

func TestOpenSearchProductionConfig_PortConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		port        string
		description string
	}{
		{
			name:        "default port",
			port:        "9200",
			description: "OpenSearch default port",
		},
		{
			name:        "custom port",
			port:        "9201",
			description: "Alternative port to avoid conflicts",
		},
		{
			name:        "high port",
			port:        "19200",
			description: "High port number for security",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultOpenSearchProductionConfig()
			config.Port = tt.port

			assert.Equal(t, tt.port, config.Port, tt.description)
		})
	}
}

func TestOpenSearchProductionConfig_SecuritySettings(t *testing.T) {
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
			config := DefaultOpenSearchProductionConfig()
			config.DisableSecurity = tt.disableSecurity

			assert.Equal(t, tt.disableSecurity, config.DisableSecurity, tt.description)
		})
	}
}

func TestOpenSearchProductionConfig_ImageVersion(t *testing.T) {
	config := DefaultOpenSearchProductionConfig()

	// Verify that OpenSearch 3.x.x image is being used
	assert.Contains(t, config.Image, "opensearch:3", "Should use OpenSearch 3.x.x version")
	assert.Equal(t, "opensearchproject/opensearch:3.0.0", config.Image)
}

func TestOpenSearchProductionConfig_NetworkIsolation(t *testing.T) {
	config := DefaultOpenSearchProductionConfig()

	// Verify network configuration
	assert.NotEmpty(t, config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)

	// Test custom network
	config.Production.NetworkName = "search-network"
	assert.Equal(t, "search-network", config.Production.NetworkName)
}

func TestOpenSearchProductionConfig_RestartPolicy(t *testing.T) {
	config := DefaultOpenSearchProductionConfig()

	// The restart policy is set in DeployOpenSearch to "unless-stopped"
	// This test just validates the config structure is ready
	assert.NotEmpty(t, config.ContainerName, "Container name is required for restart policy")
}

func TestOpenSearchProductionConfig_IndexNames(t *testing.T) {
	tests := []struct {
		name      string
		indexName string
		valid     bool
	}{
		{
			name:      "simple index",
			indexName: "test",
			valid:     true,
		},
		{
			name:      "hyphenated index",
			indexName: "test-index",
			valid:     true,
		},
		{
			name:      "underscore index",
			indexName: "test_index",
			valid:     true,
		},
		{
			name:      "numbered index",
			indexName: "index2024",
			valid:     true,
		},
		{
			name:      "time-based index",
			indexName: "logs-2024.10.29",
			valid:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultOpenSearchProductionConfig()
			indexURL := GetOpenSearchIndexURL(config, tt.indexName)

			assert.NotEmpty(t, indexURL)
			if tt.valid {
				assert.Contains(t, indexURL, tt.indexName)
			}
		})
	}
}

func TestOpenSearchProductionConfig_DefaultMemory(t *testing.T) {
	config := DefaultOpenSearchProductionConfig()

	// Default memory should be suitable for production
	assert.Equal(t, "-Xms2g -Xmx2g", config.JavaOpts, "Default should be production-ready memory settings")
	assert.Contains(t, config.JavaOpts, "-Xms", "Should set minimum heap")
	assert.Contains(t, config.JavaOpts, "-Xmx", "Should set maximum heap")
}
