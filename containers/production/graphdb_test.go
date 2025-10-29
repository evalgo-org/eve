package production

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultGraphDBProductionConfig(t *testing.T) {
	config := DefaultGraphDBProductionConfig()

	assert.Equal(t, "graphdb", config.ContainerName)
	assert.Equal(t, "ontotext/graphdb:10.8.1", config.Image)
	assert.Equal(t, "7200", config.Port)
	assert.Equal(t, "-Xms2g -Xmx4g", config.JavaOpts)
	assert.Equal(t, "graphdb-data", config.DataVolume)
	assert.Equal(t, "app-network", config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)
	assert.Equal(t, "graphdb-data", config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)
}

func TestGraphDBProductionConfig_CustomConfiguration(t *testing.T) {
	config := DefaultGraphDBProductionConfig()

	// Modify configuration
	config.ContainerName = "my-graphdb"
	config.Port = "7201"
	config.JavaOpts = "-Xms4g -Xmx8g"
	config.DataVolume = "custom-graphdb-data"

	assert.Equal(t, "my-graphdb", config.ContainerName)
	assert.Equal(t, "7201", config.Port)
	assert.Equal(t, "-Xms4g -Xmx8g", config.JavaOpts)
	assert.Equal(t, "custom-graphdb-data", config.DataVolume)
}

func TestGetGraphDBURL(t *testing.T) {
	tests := []struct {
		name     string
		config   GraphDBProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: GraphDBProductionConfig{
				Port: "7200",
			},
			expected: "http://localhost:7200",
		},
		{
			name: "custom port",
			config: GraphDBProductionConfig{
				Port: "7201",
			},
			expected: "http://localhost:7201",
		},
		{
			name: "high port number",
			config: GraphDBProductionConfig{
				Port: "17200",
			},
			expected: "http://localhost:17200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graphdbURL := GetGraphDBURL(tt.config)
			assert.Equal(t, tt.expected, graphdbURL)
		})
	}
}

func TestGetGraphDBRepositoryURL(t *testing.T) {
	tests := []struct {
		name         string
		config       GraphDBProductionConfig
		repositoryID string
		expected     string
	}{
		{
			name: "default port with simple repository",
			config: GraphDBProductionConfig{
				Port: "7200",
			},
			repositoryID: "test-repo",
			expected:     "http://localhost:7200/repositories/test-repo",
		},
		{
			name: "custom port with hyphenated repository",
			config: GraphDBProductionConfig{
				Port: "7201",
			},
			repositoryID: "my-test-repo",
			expected:     "http://localhost:7201/repositories/my-test-repo",
		},
		{
			name: "repository with underscores",
			config: GraphDBProductionConfig{
				Port: "7200",
			},
			repositoryID: "test_repo_2024",
			expected:     "http://localhost:7200/repositories/test_repo_2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoURL := GetGraphDBRepositoryURL(tt.config, tt.repositoryID)
			assert.Equal(t, tt.expected, repoURL)
		})
	}
}

func TestGraphDBProductionConfig_MemoryConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		javaOpts string
		valid    bool
	}{
		{
			name:     "development settings",
			javaOpts: "-Xms1g -Xmx2g",
			valid:    true,
		},
		{
			name:     "production settings",
			javaOpts: "-Xms2g -Xmx4g",
			valid:    true,
		},
		{
			name:     "large dataset settings",
			javaOpts: "-Xms4g -Xmx8g",
			valid:    true,
		},
		{
			name:     "minimal settings",
			javaOpts: "-Xms512m -Xmx1g",
			valid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultGraphDBProductionConfig()
			config.JavaOpts = tt.javaOpts

			assert.Equal(t, tt.javaOpts, config.JavaOpts)
			if tt.valid {
				assert.NotEmpty(t, config.JavaOpts)
			}
		})
	}
}

func TestGraphDBProductionConfig_DataPersistence(t *testing.T) {
	config := DefaultGraphDBProductionConfig()

	// Verify data volume is configured for persistence
	assert.NotEmpty(t, config.DataVolume, "Data volume should be configured")
	assert.Equal(t, "graphdb-data", config.DataVolume, "Default data volume name")

	// Test custom volume
	config.DataVolume = "production-graphdb-volume"
	assert.Equal(t, "production-graphdb-volume", config.DataVolume)
}

func TestGraphDBProductionConfig_PortConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		port        string
		description string
	}{
		{
			name:        "default port",
			port:        "7200",
			description: "GraphDB default port",
		},
		{
			name:        "custom port",
			port:        "7201",
			description: "Alternative port to avoid conflicts",
		},
		{
			name:        "high port",
			port:        "17200",
			description: "High port number for security",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultGraphDBProductionConfig()
			config.Port = tt.port

			assert.Equal(t, tt.port, config.Port, tt.description)
		})
	}
}

func TestGraphDBProductionConfig_ImageVersion(t *testing.T) {
	config := DefaultGraphDBProductionConfig()

	// Verify that GraphDB 10.x.x image is being used
	assert.Contains(t, config.Image, "graphdb:10", "Should use GraphDB 10.x.x version")
	assert.Equal(t, "ontotext/graphdb:10.8.1", config.Image)
}

func TestGraphDBProductionConfig_NetworkIsolation(t *testing.T) {
	config := DefaultGraphDBProductionConfig()

	// Verify network configuration
	assert.NotEmpty(t, config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)

	// Test custom network
	config.Production.NetworkName = "graph-network"
	assert.Equal(t, "graph-network", config.Production.NetworkName)
}

func TestGraphDBProductionConfig_RestartPolicy(t *testing.T) {
	config := DefaultGraphDBProductionConfig()

	// The restart policy is set in DeployGraphDB to "unless-stopped"
	// This test just validates the config structure is ready
	assert.NotEmpty(t, config.ContainerName, "Container name is required for restart policy")
}

func TestGraphDBProductionConfig_RepositoryIDs(t *testing.T) {
	tests := []struct {
		name         string
		repositoryID string
		valid        bool
	}{
		{
			name:         "simple repository",
			repositoryID: "test",
			valid:        true,
		},
		{
			name:         "hyphenated repository",
			repositoryID: "test-repo",
			valid:        true,
		},
		{
			name:         "underscore repository",
			repositoryID: "test_repo",
			valid:        true,
		},
		{
			name:         "numbered repository",
			repositoryID: "repo2024",
			valid:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultGraphDBProductionConfig()
			repoURL := GetGraphDBRepositoryURL(config, tt.repositoryID)

			assert.NotEmpty(t, repoURL)
			if tt.valid {
				assert.Contains(t, repoURL, tt.repositoryID)
			}
		})
	}
}

func TestGraphDBProductionConfig_DefaultMemory(t *testing.T) {
	config := DefaultGraphDBProductionConfig()

	// Default memory should be suitable for production
	assert.Equal(t, "-Xms2g -Xmx4g", config.JavaOpts, "Default should be production-ready memory settings")
	assert.Contains(t, config.JavaOpts, "-Xms", "Should set minimum heap")
	assert.Contains(t, config.JavaOpts, "-Xmx", "Should set maximum heap")
}
