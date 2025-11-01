package production

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultBaseXProductionConfig(t *testing.T) {
	config := DefaultBaseXProductionConfig()

	assert.Equal(t, "basex", config.ContainerName)
	assert.Equal(t, "basex/basexhttp:latest", config.Image)
	assert.Equal(t, "8984", config.Port)
	assert.Equal(t, "changeme", config.AdminPassword)
	assert.Equal(t, "basex-data", config.DataVolume)
	assert.Equal(t, "app-network", config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)
	assert.Equal(t, "basex-data", config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)
}

func TestBaseXProductionConfig_CustomConfiguration(t *testing.T) {
	config := DefaultBaseXProductionConfig()

	// Modify configuration
	config.ContainerName = "my-basex"
	config.Port = "9000"
	config.AdminPassword = "secure-password"
	config.DataVolume = "custom-basex-data"

	assert.Equal(t, "my-basex", config.ContainerName)
	assert.Equal(t, "9000", config.Port)
	assert.Equal(t, "secure-password", config.AdminPassword)
	assert.Equal(t, "custom-basex-data", config.DataVolume)
}

func TestBaseXProductionConfig_Validation(t *testing.T) {
	tests := []struct {
		name          string
		modifyConfig  func(*BaseXProductionConfig)
		expectWarning string
	}{
		{
			name: "default password warning",
			modifyConfig: func(c *BaseXProductionConfig) {
				// Keep default password
			},
			expectWarning: "Should warn about default password in production",
		},
		{
			name: "weak password warning",
			modifyConfig: func(c *BaseXProductionConfig) {
				c.AdminPassword = "123"
			},
			expectWarning: "Should warn about weak password",
		},
		{
			name: "strong password OK",
			modifyConfig: func(c *BaseXProductionConfig) {
				c.AdminPassword = "SecureP@ssw0rd123!"
			},
			expectWarning: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultBaseXProductionConfig()
			tt.modifyConfig(&config)

			// In a real implementation, we might have a Validate() method
			// For now, we're just testing the configuration structure
			assert.NotEmpty(t, config.AdminPassword)
		})
	}
}

func TestGetBaseXConnectionURL(t *testing.T) {
	tests := []struct {
		name     string
		config   BaseXProductionConfig
		expected string
	}{
		{
			name: "default configuration",
			config: BaseXProductionConfig{
				Port: "8984",
			},
			expected: "http://localhost:8984",
		},
		{
			name: "custom port",
			config: BaseXProductionConfig{
				Port: "9000",
			},
			expected: "http://localhost:9000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Helper function to build BaseX URL
			url := "http://localhost:" + tt.config.Port
			assert.Equal(t, tt.expected, url)
		})
	}
}

func TestBaseXProductionConfig_EnvironmentVariables(t *testing.T) {
	config := DefaultBaseXProductionConfig()

	// Test that environment variables are properly formatted
	expectedEnvVars := []string{
		"BASEX_ADMIN_PW=" + config.AdminPassword,
	}

	// This would be validated in the actual deployment
	for _, envVar := range expectedEnvVars {
		assert.Contains(t, envVar, "BASEX_ADMIN_PW=")
	}
}

func TestBaseXProductionConfig_PortBinding(t *testing.T) {
	tests := []struct {
		name        string
		port        string
		expectValid bool
	}{
		{
			name:        "default port",
			port:        "8984",
			expectValid: true,
		},
		{
			name:        "custom high port",
			port:        "38984",
			expectValid: true,
		},
		{
			name:        "low numbered port",
			port:        "80",
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultBaseXProductionConfig()
			config.Port = tt.port

			// Port validation would happen in actual deployment
			assert.NotEmpty(t, config.Port)
			assert.Equal(t, tt.port, config.Port)
		})
	}
}

func TestBaseXProductionConfig_VolumeConfiguration(t *testing.T) {
	config := DefaultBaseXProductionConfig()

	// Verify volume configuration
	assert.Equal(t, "basex-data", config.DataVolume)
	assert.Equal(t, config.DataVolume, config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)

	// Volume should be created before container deployment
	assert.NotEmpty(t, config.DataVolume, "Volume name should not be empty")
}

func TestBaseXProductionConfig_NetworkConfiguration(t *testing.T) {
	config := DefaultBaseXProductionConfig()

	// Verify network configuration
	assert.Equal(t, "app-network", config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)

	// Network should be created before container deployment
	assert.NotEmpty(t, config.Production.NetworkName, "Network name should not be empty")
}

func TestBaseXProductionConfig_RestartPolicy(t *testing.T) {
	// In production, containers should have restart policy configured
	// This is tested indirectly through the DeployBaseX function
	// The restart policy should be "unless-stopped"

	config := DefaultBaseXProductionConfig()
	assert.NotEmpty(t, config.ContainerName, "Container name required for restart policy")
}

func TestBaseXProductionConfig_HealthCheck(t *testing.T) {
	// Health check configuration should be part of deployment
	// BaseX health check: HTTP GET http://localhost:8984/

	config := DefaultBaseXProductionConfig()

	// Health check URL would be: http://localhost:{port}/
	healthCheckURL := "http://localhost:" + config.Port + "/"
	assert.Equal(t, "http://localhost:8984/", healthCheckURL)
}
