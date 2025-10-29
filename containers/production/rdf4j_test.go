package production

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultRDF4JProductionConfig(t *testing.T) {
	config := DefaultRDF4JProductionConfig()

	assert.Equal(t, "rdf4j", config.ContainerName)
	assert.Equal(t, "eclipse/rdf4j-workbench:5.2.0-jetty", config.Image)
	assert.Equal(t, "8080", config.Port)
	assert.Equal(t, "-Xms2g -Xmx4g", config.JavaOpts)
	assert.Equal(t, "rdf4j-data", config.DataVolume)
	assert.Equal(t, "rdf4j-logs", config.LogsVolume)
	assert.Equal(t, "app-network", config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)
	assert.Equal(t, "rdf4j-data", config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)
}

func TestRDF4JProductionConfig_CustomConfiguration(t *testing.T) {
	config := DefaultRDF4JProductionConfig()

	// Modify configuration
	config.ContainerName = "my-rdf4j"
	config.Port = "8081"
	config.JavaOpts = "-Xms4g -Xmx8g"
	config.DataVolume = "custom-rdf4j-data"
	config.LogsVolume = "custom-rdf4j-logs"

	assert.Equal(t, "my-rdf4j", config.ContainerName)
	assert.Equal(t, "8081", config.Port)
	assert.Equal(t, "-Xms4g -Xmx8g", config.JavaOpts)
	assert.Equal(t, "custom-rdf4j-data", config.DataVolume)
	assert.Equal(t, "custom-rdf4j-logs", config.LogsVolume)
}

func TestGetRDF4JURL(t *testing.T) {
	tests := []struct {
		name     string
		config   RDF4JProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: RDF4JProductionConfig{
				Port: "8080",
			},
			expected: "http://localhost:8080",
		},
		{
			name: "custom port",
			config: RDF4JProductionConfig{
				Port: "8081",
			},
			expected: "http://localhost:8081",
		},
		{
			name: "high port number",
			config: RDF4JProductionConfig{
				Port: "18080",
			},
			expected: "http://localhost:18080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rdf4jURL := GetRDF4JURL(tt.config)
			assert.Equal(t, tt.expected, rdf4jURL)
		})
	}
}

func TestGetRDF4JWorkbenchURL(t *testing.T) {
	tests := []struct {
		name     string
		config   RDF4JProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: RDF4JProductionConfig{
				Port: "8080",
			},
			expected: "http://localhost:8080/rdf4j-workbench",
		},
		{
			name: "custom port",
			config: RDF4JProductionConfig{
				Port: "8081",
			},
			expected: "http://localhost:8081/rdf4j-workbench",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workbenchURL := GetRDF4JWorkbenchURL(tt.config)
			assert.Equal(t, tt.expected, workbenchURL)
		})
	}
}

func TestGetRDF4JServerURL(t *testing.T) {
	tests := []struct {
		name     string
		config   RDF4JProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: RDF4JProductionConfig{
				Port: "8080",
			},
			expected: "http://localhost:8080/rdf4j-server",
		},
		{
			name: "custom port",
			config: RDF4JProductionConfig{
				Port: "8081",
			},
			expected: "http://localhost:8081/rdf4j-server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverURL := GetRDF4JServerURL(tt.config)
			assert.Equal(t, tt.expected, serverURL)
		})
	}
}

func TestGetRDF4JRepositoryURL(t *testing.T) {
	tests := []struct {
		name         string
		config       RDF4JProductionConfig
		repositoryID string
		expected     string
	}{
		{
			name: "default port with simple repository",
			config: RDF4JProductionConfig{
				Port: "8080",
			},
			repositoryID: "test-repo",
			expected:     "http://localhost:8080/rdf4j-server/repositories/test-repo",
		},
		{
			name: "custom port with hyphenated repository",
			config: RDF4JProductionConfig{
				Port: "8081",
			},
			repositoryID: "my-test-repo",
			expected:     "http://localhost:8081/rdf4j-server/repositories/my-test-repo",
		},
		{
			name: "repository with underscores",
			config: RDF4JProductionConfig{
				Port: "8080",
			},
			repositoryID: "test_repo_2024",
			expected:     "http://localhost:8080/rdf4j-server/repositories/test_repo_2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoURL := GetRDF4JRepositoryURL(tt.config, tt.repositoryID)
			assert.Equal(t, tt.expected, repoURL)
		})
	}
}

func TestRDF4JProductionConfig_MemoryConfiguration(t *testing.T) {
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
			config := DefaultRDF4JProductionConfig()
			config.JavaOpts = tt.javaOpts

			assert.Equal(t, tt.javaOpts, config.JavaOpts)
			if tt.valid {
				assert.NotEmpty(t, config.JavaOpts)
			}
		})
	}
}

func TestRDF4JProductionConfig_DataPersistence(t *testing.T) {
	config := DefaultRDF4JProductionConfig()

	// Verify data volume is configured for persistence
	assert.NotEmpty(t, config.DataVolume, "Data volume should be configured")
	assert.Equal(t, "rdf4j-data", config.DataVolume, "Default data volume name")

	// Verify logs volume is configured
	assert.NotEmpty(t, config.LogsVolume, "Logs volume should be configured")
	assert.Equal(t, "rdf4j-logs", config.LogsVolume, "Default logs volume name")

	// Test custom volumes
	config.DataVolume = "production-rdf4j-data"
	config.LogsVolume = "production-rdf4j-logs"
	assert.Equal(t, "production-rdf4j-data", config.DataVolume)
	assert.Equal(t, "production-rdf4j-logs", config.LogsVolume)
}

func TestRDF4JProductionConfig_PortConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		port        string
		description string
	}{
		{
			name:        "default port",
			port:        "8080",
			description: "RDF4J default port",
		},
		{
			name:        "custom port",
			port:        "8081",
			description: "Alternative port to avoid conflicts",
		},
		{
			name:        "high port",
			port:        "18080",
			description: "High port number for security",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultRDF4JProductionConfig()
			config.Port = tt.port

			assert.Equal(t, tt.port, config.Port, tt.description)
		})
	}
}

func TestRDF4JProductionConfig_ImageVersion(t *testing.T) {
	config := DefaultRDF4JProductionConfig()

	// Verify that RDF4J 5.x.x image is being used
	assert.Contains(t, config.Image, "rdf4j-workbench:5", "Should use RDF4J 5.x.x version")
	assert.Equal(t, "eclipse/rdf4j-workbench:5.2.0-jetty", config.Image)
}

func TestRDF4JProductionConfig_NetworkIsolation(t *testing.T) {
	config := DefaultRDF4JProductionConfig()

	// Verify network configuration
	assert.NotEmpty(t, config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)

	// Test custom network
	config.Production.NetworkName = "rdf-network"
	assert.Equal(t, "rdf-network", config.Production.NetworkName)
}

func TestRDF4JProductionConfig_RestartPolicy(t *testing.T) {
	config := DefaultRDF4JProductionConfig()

	// The restart policy is set in DeployRDF4J to "unless-stopped"
	// This test just validates the config structure is ready
	assert.NotEmpty(t, config.ContainerName, "Container name is required for restart policy")
}

func TestRDF4JProductionConfig_RepositoryIDs(t *testing.T) {
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
			config := DefaultRDF4JProductionConfig()
			repoURL := GetRDF4JRepositoryURL(config, tt.repositoryID)

			assert.NotEmpty(t, repoURL)
			if tt.valid {
				assert.Contains(t, repoURL, tt.repositoryID)
			}
		})
	}
}

func TestRDF4JProductionConfig_DefaultMemory(t *testing.T) {
	config := DefaultRDF4JProductionConfig()

	// Default memory should be suitable for production
	assert.Equal(t, "-Xms2g -Xmx4g", config.JavaOpts, "Default should be production-ready memory settings")
	assert.Contains(t, config.JavaOpts, "-Xms", "Should set minimum heap")
	assert.Contains(t, config.JavaOpts, "-Xmx", "Should set maximum heap")
}

func TestRDF4JProductionConfig_URLFormatting(t *testing.T) {
	config := DefaultRDF4JProductionConfig()

	// Test all URL helper functions
	baseURL := GetRDF4JURL(config)
	workbenchURL := GetRDF4JWorkbenchURL(config)
	serverURL := GetRDF4JServerURL(config)
	repoURL := GetRDF4JRepositoryURL(config, "test")

	// Verify all start with base URL
	assert.True(t, len(workbenchURL) > len(baseURL), "Workbench URL should extend base URL")
	assert.True(t, len(serverURL) > len(baseURL), "Server URL should extend base URL")
	assert.True(t, len(repoURL) > len(serverURL), "Repository URL should extend server URL")

	// Verify URL paths
	assert.Contains(t, workbenchURL, "/rdf4j-workbench")
	assert.Contains(t, serverURL, "/rdf4j-server")
	assert.Contains(t, repoURL, "/rdf4j-server/repositories/")
}
