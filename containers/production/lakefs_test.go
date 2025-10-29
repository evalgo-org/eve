package production

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultLakeFSProductionConfig(t *testing.T) {
	config := DefaultLakeFSProductionConfig()

	assert.Equal(t, "lakefs", config.ContainerName)
	assert.Equal(t, "treeverse/lakefs:1.70", config.Image)
	assert.Equal(t, "8000", config.Port)
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", config.AccessKeyID)
	assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", config.SecretAccessKey)
	assert.Equal(t, "lakefs-data", config.DataVolume)
	assert.Equal(t, "app-network", config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)
	assert.Equal(t, "lakefs-data", config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)
}

func TestLakeFSProductionConfig_CustomConfiguration(t *testing.T) {
	config := DefaultLakeFSProductionConfig()

	// Modify configuration
	config.ContainerName = "my-lakefs"
	config.Port = "8001"
	config.AccessKeyID = "custom-access-key"
	config.SecretAccessKey = "custom-secret-key"
	config.DataVolume = "custom-lakefs-data"

	assert.Equal(t, "my-lakefs", config.ContainerName)
	assert.Equal(t, "8001", config.Port)
	assert.Equal(t, "custom-access-key", config.AccessKeyID)
	assert.Equal(t, "custom-secret-key", config.SecretAccessKey)
	assert.Equal(t, "custom-lakefs-data", config.DataVolume)
}

func TestGetLakeFSURL(t *testing.T) {
	tests := []struct {
		name     string
		config   LakeFSProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: LakeFSProductionConfig{
				Port: "8000",
			},
			expected: "http://localhost:8000",
		},
		{
			name: "custom port",
			config: LakeFSProductionConfig{
				Port: "8001",
			},
			expected: "http://localhost:8001",
		},
		{
			name: "high port number",
			config: LakeFSProductionConfig{
				Port: "18000",
			},
			expected: "http://localhost:18000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lakeFSURL := GetLakeFSURL(tt.config)
			assert.Equal(t, tt.expected, lakeFSURL)
		})
	}
}

func TestGetLakeFSAPIURL(t *testing.T) {
	tests := []struct {
		name     string
		config   LakeFSProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: LakeFSProductionConfig{
				Port: "8000",
			},
			expected: "http://localhost:8000/api/v1",
		},
		{
			name: "custom port",
			config: LakeFSProductionConfig{
				Port: "8001",
			},
			expected: "http://localhost:8001/api/v1",
		},
		{
			name: "high port number",
			config: LakeFSProductionConfig{
				Port: "18000",
			},
			expected: "http://localhost:18000/api/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiURL := GetLakeFSAPIURL(tt.config)
			assert.Equal(t, tt.expected, apiURL)
		})
	}
}

func TestGetLakeFSHealthURL(t *testing.T) {
	tests := []struct {
		name     string
		config   LakeFSProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: LakeFSProductionConfig{
				Port: "8000",
			},
			expected: "http://localhost:8000/api/v1/healthcheck",
		},
		{
			name: "custom port",
			config: LakeFSProductionConfig{
				Port: "8001",
			},
			expected: "http://localhost:8001/api/v1/healthcheck",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healthURL := GetLakeFSHealthURL(tt.config)
			assert.Equal(t, tt.expected, healthURL)
		})
	}
}

func TestLakeFSProductionConfig_DataPersistence(t *testing.T) {
	config := DefaultLakeFSProductionConfig()

	// Verify data volume is configured for persistence
	assert.NotEmpty(t, config.DataVolume, "Data volume should be configured")
	assert.Equal(t, "lakefs-data", config.DataVolume, "Default data volume name")

	// Test custom volume
	config.DataVolume = "production-lakefs-volume"
	assert.Equal(t, "production-lakefs-volume", config.DataVolume)
}

func TestLakeFSProductionConfig_PortConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		port        string
		description string
	}{
		{
			name:        "default port",
			port:        "8000",
			description: "LakeFS default port",
		},
		{
			name:        "custom port",
			port:        "8001",
			description: "Alternative port to avoid conflicts",
		},
		{
			name:        "high port",
			port:        "18000",
			description: "High port number for security",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultLakeFSProductionConfig()
			config.Port = tt.port

			assert.Equal(t, tt.port, config.Port, tt.description)
		})
	}
}

func TestLakeFSProductionConfig_ImageVersion(t *testing.T) {
	config := DefaultLakeFSProductionConfig()

	// Verify that LakeFS 1.70 image is being used
	assert.Contains(t, config.Image, "treeverse/lakefs:1.70", "Should use LakeFS 1.70 version")
	assert.Equal(t, "treeverse/lakefs:1.70", config.Image)
}

func TestLakeFSProductionConfig_NetworkIsolation(t *testing.T) {
	config := DefaultLakeFSProductionConfig()

	// Verify network configuration
	assert.NotEmpty(t, config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)

	// Test custom network
	config.Production.NetworkName = "lakefs-network"
	assert.Equal(t, "lakefs-network", config.Production.NetworkName)
}

func TestLakeFSProductionConfig_RestartPolicy(t *testing.T) {
	config := DefaultLakeFSProductionConfig()

	// The restart policy is set in DeployLakeFS to "unless-stopped"
	// This test just validates the config structure is ready
	assert.NotEmpty(t, config.ContainerName, "Container name is required for restart policy")
}

func TestLakeFSProductionConfig_Credentials(t *testing.T) {
	tests := []struct {
		name            string
		accessKeyID     string
		secretAccessKey string
		valid           bool
	}{
		{
			name:            "default credentials",
			accessKeyID:     "AKIAIOSFODNN7EXAMPLE",
			secretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			valid:           true,
		},
		{
			name:            "custom credentials",
			accessKeyID:     "my-access-key",
			secretAccessKey: "my-secret-key-12345678",
			valid:           true,
		},
		{
			name:            "short access key",
			accessKeyID:     "KEY123",
			secretAccessKey: "secret",
			valid:           true,
		},
		{
			name:            "long credentials",
			accessKeyID:     "AKIAIOSFODNN7EXAMPLEVERYLONGKEY",
			secretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEYVERYLONG",
			valid:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultLakeFSProductionConfig()
			config.AccessKeyID = tt.accessKeyID
			config.SecretAccessKey = tt.secretAccessKey

			assert.Equal(t, tt.accessKeyID, config.AccessKeyID)
			assert.Equal(t, tt.secretAccessKey, config.SecretAccessKey)
			if tt.valid {
				assert.NotEmpty(t, config.AccessKeyID)
				assert.NotEmpty(t, config.SecretAccessKey)
			}
		})
	}
}

func TestLakeFSProductionConfig_ContainerName(t *testing.T) {
	config := DefaultLakeFSProductionConfig()

	// Default container name
	assert.Equal(t, "lakefs", config.ContainerName)

	// Custom container name
	config.ContainerName = "lakefs-prod"
	assert.Equal(t, "lakefs-prod", config.ContainerName)
}

func TestLakeFSProductionConfig_URLFormatting(t *testing.T) {
	config := DefaultLakeFSProductionConfig()

	// Test base URL
	baseURL := GetLakeFSURL(config)
	assert.True(t, len(baseURL) > 0, "Base URL should not be empty")
	assert.Contains(t, baseURL, "http://")
	assert.Contains(t, baseURL, config.Port)

	// Test API URL
	apiURL := GetLakeFSAPIURL(config)
	assert.True(t, len(apiURL) > 0, "API URL should not be empty")
	assert.Contains(t, apiURL, config.Port)
	assert.Contains(t, apiURL, "/api/v1")

	// Test health URL
	healthURL := GetLakeFSHealthURL(config)
	assert.True(t, len(healthURL) > 0, "Health URL should not be empty")
	assert.Contains(t, healthURL, config.Port)
	assert.Contains(t, healthURL, "/api/v1/healthcheck")
}

func TestLakeFSProductionConfig_SecurityWarning(t *testing.T) {
	config := DefaultLakeFSProductionConfig()

	// Verify default credentials are example values to trigger security awareness
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", config.AccessKeyID,
		"Default access key is example - MUST be changed for production!")
	assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", config.SecretAccessKey,
		"Default secret key is example - MUST be changed for production!")

	// Test credential change
	config.AccessKeyID = "new_access_key"
	config.SecretAccessKey = "new_secret_key"
	assert.NotEqual(t, "AKIAIOSFODNN7EXAMPLE", config.AccessKeyID,
		"Access key should be changed from default")
	assert.NotEqual(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", config.SecretAccessKey,
		"Secret key should be changed from default")
}

func TestLakeFSProductionConfig_DefaultPort(t *testing.T) {
	config := DefaultLakeFSProductionConfig()

	// Verify LakeFS default port
	assert.Equal(t, "8000", config.Port, "LakeFS default port should be 8000")
}

func TestLakeFSProductionConfig_VolumeConfiguration(t *testing.T) {
	config := DefaultLakeFSProductionConfig()

	// Verify volume configuration
	assert.Equal(t, "lakefs-data", config.DataVolume)
	assert.Equal(t, "lakefs-data", config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)

	// Verify volume and production settings match
	assert.Equal(t, config.DataVolume, config.Production.VolumeName,
		"DataVolume and Production.VolumeName should match")
}

func TestLakeFSProductionConfig_MultipleInstances(t *testing.T) {
	// Test creating multiple configurations for different instances
	config1 := DefaultLakeFSProductionConfig()
	config1.ContainerName = "lakefs-prod"
	config1.Port = "8000"

	config2 := DefaultLakeFSProductionConfig()
	config2.ContainerName = "lakefs-staging"
	config2.Port = "8001"

	// Ensure they are different
	assert.NotEqual(t, config1.ContainerName, config2.ContainerName)
	assert.NotEqual(t, config1.Port, config2.Port)
}

func TestLakeFSProductionConfig_ImageRepository(t *testing.T) {
	config := DefaultLakeFSProductionConfig()

	// Verify image is from official treeverse repository
	assert.Contains(t, config.Image, "treeverse/lakefs", "Should use official treeverse image")
}

func TestLakeFSProductionConfig_CredentialValidation(t *testing.T) {
	config := DefaultLakeFSProductionConfig()

	// Test that credentials are not empty
	assert.NotEmpty(t, config.AccessKeyID, "Access key ID should not be empty")
	assert.NotEmpty(t, config.SecretAccessKey, "Secret access key should not be empty")

	// Test credential length
	assert.Greater(t, len(config.AccessKeyID), 0, "Access key ID should have positive length")
	assert.Greater(t, len(config.SecretAccessKey), 0, "Secret access key should have positive length")
}

func TestLakeFSProductionConfig_URLEndpoints(t *testing.T) {
	config := DefaultLakeFSProductionConfig()
	config.Port = "8000"

	// Test all URL helper functions
	baseURL := GetLakeFSURL(config)
	apiURL := GetLakeFSAPIURL(config)
	healthURL := GetLakeFSHealthURL(config)

	// Verify base URL structure
	assert.Equal(t, "http://localhost:8000", baseURL)

	// Verify API URL structure
	assert.Equal(t, "http://localhost:8000/api/v1", apiURL)

	// Verify health URL structure
	assert.Equal(t, "http://localhost:8000/api/v1/healthcheck", healthURL)

	// Verify API URL is extension of base URL
	assert.Contains(t, apiURL, baseURL)

	// Verify health URL is extension of API URL
	assert.Contains(t, healthURL, apiURL)
}
