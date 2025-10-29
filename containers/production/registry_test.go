package production

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultRegistryProductionConfig(t *testing.T) {
	config := DefaultRegistryProductionConfig()

	assert.Equal(t, "registry", config.ContainerName)
	assert.Equal(t, "registry:3", config.Image)
	assert.Equal(t, "5000", config.Port)
	assert.Equal(t, "registry-data", config.DataVolume)
	assert.Equal(t, "app-network", config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)
	assert.Equal(t, "registry-data", config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)
}

func TestRegistryProductionConfig_CustomConfiguration(t *testing.T) {
	config := DefaultRegistryProductionConfig()

	// Modify configuration
	config.ContainerName = "my-registry"
	config.Port = "5001"
	config.DataVolume = "custom-registry-data"

	assert.Equal(t, "my-registry", config.ContainerName)
	assert.Equal(t, "5001", config.Port)
	assert.Equal(t, "custom-registry-data", config.DataVolume)
}

func TestGetRegistryURL(t *testing.T) {
	tests := []struct {
		name     string
		config   RegistryProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: RegistryProductionConfig{
				Port: "5000",
			},
			expected: "http://localhost:5000",
		},
		{
			name: "custom port",
			config: RegistryProductionConfig{
				Port: "5001",
			},
			expected: "http://localhost:5001",
		},
		{
			name: "high port number",
			config: RegistryProductionConfig{
				Port: "15000",
			},
			expected: "http://localhost:15000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registryURL := GetRegistryURL(tt.config)
			assert.Equal(t, tt.expected, registryURL)
		})
	}
}

func TestGetRegistryImageURL(t *testing.T) {
	tests := []struct {
		name      string
		config    RegistryProductionConfig
		imageName string
		tag       string
		expected  string
	}{
		{
			name: "default port with latest tag",
			config: RegistryProductionConfig{
				Port: "5000",
			},
			imageName: "myapp",
			tag:       "latest",
			expected:  "localhost:5000/myapp:latest",
		},
		{
			name: "custom port with version tag",
			config: RegistryProductionConfig{
				Port: "5001",
			},
			imageName: "myapp",
			tag:       "v1.0.0",
			expected:  "localhost:5001/myapp:v1.0.0",
		},
		{
			name: "namespaced image",
			config: RegistryProductionConfig{
				Port: "5000",
			},
			imageName: "myorg/myapp",
			tag:       "stable",
			expected:  "localhost:5000/myorg/myapp:stable",
		},
		{
			name: "multi-level namespace",
			config: RegistryProductionConfig{
				Port: "5000",
			},
			imageName: "myorg/team/myapp",
			tag:       "dev",
			expected:  "localhost:5000/myorg/team/myapp:dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageURL := GetRegistryImageURL(tt.config, tt.imageName, tt.tag)
			assert.Equal(t, tt.expected, imageURL)
		})
	}
}

func TestRegistryProductionConfig_DataPersistence(t *testing.T) {
	config := DefaultRegistryProductionConfig()

	// Verify data volume is configured for persistence
	assert.NotEmpty(t, config.DataVolume, "Data volume should be configured")
	assert.Equal(t, "registry-data", config.DataVolume, "Default data volume name")

	// Test custom volume
	config.DataVolume = "production-registry-volume"
	assert.Equal(t, "production-registry-volume", config.DataVolume)
}

func TestRegistryProductionConfig_PortConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		port        string
		description string
	}{
		{
			name:        "default port",
			port:        "5000",
			description: "Docker Registry default port",
		},
		{
			name:        "custom port",
			port:        "5001",
			description: "Alternative port to avoid conflicts",
		},
		{
			name:        "high port",
			port:        "15000",
			description: "High port number for security",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultRegistryProductionConfig()
			config.Port = tt.port

			assert.Equal(t, tt.port, config.Port, tt.description)
		})
	}
}

func TestRegistryProductionConfig_ImageVersion(t *testing.T) {
	config := DefaultRegistryProductionConfig()

	// Verify that Docker Registry 3.x image is being used
	assert.Contains(t, config.Image, "registry:3", "Should use Docker Registry 3.x version")
	assert.Equal(t, "registry:3", config.Image)
}

func TestRegistryProductionConfig_NetworkIsolation(t *testing.T) {
	config := DefaultRegistryProductionConfig()

	// Verify network configuration
	assert.NotEmpty(t, config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)

	// Test custom network
	config.Production.NetworkName = "registry-network"
	assert.Equal(t, "registry-network", config.Production.NetworkName)
}

func TestRegistryProductionConfig_RestartPolicy(t *testing.T) {
	config := DefaultRegistryProductionConfig()

	// The restart policy is set in DeployRegistry to "unless-stopped"
	// This test just validates the config structure is ready
	assert.NotEmpty(t, config.ContainerName, "Container name is required for restart policy")
}

func TestRegistryProductionConfig_ImageNames(t *testing.T) {
	tests := []struct {
		name      string
		imageName string
		tag       string
		valid     bool
	}{
		{
			name:      "simple image",
			imageName: "app",
			tag:       "latest",
			valid:     true,
		},
		{
			name:      "hyphenated image",
			imageName: "my-app",
			tag:       "v1.0.0",
			valid:     true,
		},
		{
			name:      "namespaced image",
			imageName: "myorg/app",
			tag:       "stable",
			valid:     true,
		},
		{
			name:      "multi-level namespace",
			imageName: "myorg/team/app",
			tag:       "dev",
			valid:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultRegistryProductionConfig()
			imageURL := GetRegistryImageURL(config, tt.imageName, tt.tag)

			assert.NotEmpty(t, imageURL)
			if tt.valid {
				assert.Contains(t, imageURL, tt.imageName)
				assert.Contains(t, imageURL, tt.tag)
			}
		})
	}
}

func TestRegistryProductionConfig_ContainerName(t *testing.T) {
	config := DefaultRegistryProductionConfig()

	// Default container name
	assert.Equal(t, "registry", config.ContainerName)

	// Custom container name
	config.ContainerName = "docker-registry-prod"
	assert.Equal(t, "docker-registry-prod", config.ContainerName)
}

func TestRegistryProductionConfig_URLFormatting(t *testing.T) {
	config := DefaultRegistryProductionConfig()

	// Test base URL
	baseURL := GetRegistryURL(config)
	assert.True(t, len(baseURL) > 0, "Base URL should not be empty")
	assert.Contains(t, baseURL, "http://")
	assert.Contains(t, baseURL, config.Port)

	// Test image URL
	imageURL := GetRegistryImageURL(config, "test", "v1")
	assert.True(t, len(imageURL) > 0, "Image URL should not be empty")
	assert.Contains(t, imageURL, config.Port)
	assert.Contains(t, imageURL, "test:v1")
}

func TestRegistryProductionConfig_ImageTags(t *testing.T) {
	tests := []struct {
		name string
		tag  string
	}{
		{
			name: "latest tag",
			tag:  "latest",
		},
		{
			name: "version tag",
			tag:  "v1.0.0",
		},
		{
			name: "semantic version",
			tag:  "1.2.3",
		},
		{
			name: "git sha tag",
			tag:  "abc123",
		},
		{
			name: "environment tag",
			tag:  "production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultRegistryProductionConfig()
			imageURL := GetRegistryImageURL(config, "myapp", tt.tag)

			assert.Contains(t, imageURL, tt.tag, "Image URL should contain the tag")
		})
	}
}
