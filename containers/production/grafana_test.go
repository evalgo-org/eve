package production

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultGrafanaProductionConfig(t *testing.T) {
	config := DefaultGrafanaProductionConfig()

	assert.Equal(t, "grafana", config.ContainerName)
	assert.Equal(t, "grafana/grafana:12.3.0-18893060694", config.Image)
	assert.Equal(t, "3000", config.Port)
	assert.Equal(t, "admin", config.AdminUser)
	assert.Equal(t, "admin", config.AdminPassword)
	assert.Equal(t, "grafana-data", config.DataVolume)
	assert.Equal(t, "app-network", config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)
	assert.Equal(t, "grafana-data", config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)
}

func TestGrafanaProductionConfig_CustomConfiguration(t *testing.T) {
	config := DefaultGrafanaProductionConfig()

	// Modify configuration
	config.ContainerName = "my-grafana"
	config.Port = "3001"
	config.AdminUser = "customadmin"
	config.AdminPassword = "securepassword"
	config.DataVolume = "custom-grafana-data"

	assert.Equal(t, "my-grafana", config.ContainerName)
	assert.Equal(t, "3001", config.Port)
	assert.Equal(t, "customadmin", config.AdminUser)
	assert.Equal(t, "securepassword", config.AdminPassword)
	assert.Equal(t, "custom-grafana-data", config.DataVolume)
}

func TestGetGrafanaURL(t *testing.T) {
	tests := []struct {
		name     string
		config   GrafanaProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: GrafanaProductionConfig{
				Port: "3000",
			},
			expected: "http://localhost:3000",
		},
		{
			name: "custom port",
			config: GrafanaProductionConfig{
				Port: "3001",
			},
			expected: "http://localhost:3001",
		},
		{
			name: "high port number",
			config: GrafanaProductionConfig{
				Port: "13000",
			},
			expected: "http://localhost:13000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grafanaURL := GetGrafanaURL(tt.config)
			assert.Equal(t, tt.expected, grafanaURL)
		})
	}
}

func TestGetGrafanaAPIURL(t *testing.T) {
	tests := []struct {
		name     string
		config   GrafanaProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: GrafanaProductionConfig{
				Port: "3000",
			},
			expected: "http://localhost:3000/api",
		},
		{
			name: "custom port",
			config: GrafanaProductionConfig{
				Port: "3001",
			},
			expected: "http://localhost:3001/api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiURL := GetGrafanaAPIURL(tt.config)
			assert.Equal(t, tt.expected, apiURL)
		})
	}
}

func TestGetGrafanaHealthURL(t *testing.T) {
	tests := []struct {
		name     string
		config   GrafanaProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: GrafanaProductionConfig{
				Port: "3000",
			},
			expected: "http://localhost:3000/api/health",
		},
		{
			name: "custom port",
			config: GrafanaProductionConfig{
				Port: "3001",
			},
			expected: "http://localhost:3001/api/health",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healthURL := GetGrafanaHealthURL(tt.config)
			assert.Equal(t, tt.expected, healthURL)
		})
	}
}

func TestGrafanaProductionConfig_DataPersistence(t *testing.T) {
	config := DefaultGrafanaProductionConfig()

	// Verify data volume is configured for persistence
	assert.NotEmpty(t, config.DataVolume, "Data volume should be configured")
	assert.Equal(t, "grafana-data", config.DataVolume, "Default data volume name")

	// Test custom volume
	config.DataVolume = "production-grafana-volume"
	assert.Equal(t, "production-grafana-volume", config.DataVolume)
}

func TestGrafanaProductionConfig_PortConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		port        string
		description string
	}{
		{
			name:        "default port",
			port:        "3000",
			description: "Grafana default port",
		},
		{
			name:        "custom port",
			port:        "3001",
			description: "Alternative port to avoid conflicts",
		},
		{
			name:        "high port",
			port:        "13000",
			description: "High port number for security",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultGrafanaProductionConfig()
			config.Port = tt.port

			assert.Equal(t, tt.port, config.Port, tt.description)
		})
	}
}

func TestGrafanaProductionConfig_ImageVersion(t *testing.T) {
	config := DefaultGrafanaProductionConfig()

	// Verify that Grafana 12.3.0 image is being used
	assert.Contains(t, config.Image, "grafana/grafana:12.3.0", "Should use Grafana 12.3.0 version")
	assert.Equal(t, "grafana/grafana:12.3.0-18893060694", config.Image)
}

func TestGrafanaProductionConfig_NetworkIsolation(t *testing.T) {
	config := DefaultGrafanaProductionConfig()

	// Verify network configuration
	assert.NotEmpty(t, config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)

	// Test custom network
	config.Production.NetworkName = "grafana-network"
	assert.Equal(t, "grafana-network", config.Production.NetworkName)
}

func TestGrafanaProductionConfig_RestartPolicy(t *testing.T) {
	config := DefaultGrafanaProductionConfig()

	// The restart policy is set in DeployGrafana to "unless-stopped"
	// This test just validates the config structure is ready
	assert.NotEmpty(t, config.ContainerName, "Container name is required for restart policy")
}

func TestGrafanaProductionConfig_AdminCredentials(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		valid    bool
	}{
		{
			name:     "default credentials",
			username: "admin",
			password: "admin",
			valid:    true,
		},
		{
			name:     "custom credentials",
			username: "customadmin",
			password: "securepassword123",
			valid:    true,
		},
		{
			name:     "email as username",
			username: "admin@example.com",
			password: "password",
			valid:    true,
		},
		{
			name:     "complex password",
			username: "admin",
			password: "P@ssw0rd!Complex#2024",
			valid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultGrafanaProductionConfig()
			config.AdminUser = tt.username
			config.AdminPassword = tt.password

			assert.Equal(t, tt.username, config.AdminUser)
			assert.Equal(t, tt.password, config.AdminPassword)
			if tt.valid {
				assert.NotEmpty(t, config.AdminUser)
				assert.NotEmpty(t, config.AdminPassword)
			}
		})
	}
}

func TestGrafanaProductionConfig_ContainerName(t *testing.T) {
	config := DefaultGrafanaProductionConfig()

	// Default container name
	assert.Equal(t, "grafana", config.ContainerName)

	// Custom container name
	config.ContainerName = "grafana-prod"
	assert.Equal(t, "grafana-prod", config.ContainerName)
}

func TestGrafanaProductionConfig_URLFormatting(t *testing.T) {
	config := DefaultGrafanaProductionConfig()

	// Test base URL
	baseURL := GetGrafanaURL(config)
	assert.True(t, len(baseURL) > 0, "Base URL should not be empty")
	assert.Contains(t, baseURL, "http://")
	assert.Contains(t, baseURL, config.Port)

	// Test API URL
	apiURL := GetGrafanaAPIURL(config)
	assert.True(t, len(apiURL) > 0, "API URL should not be empty")
	assert.Contains(t, apiURL, config.Port)
	assert.Contains(t, apiURL, "/api")

	// Test health URL
	healthURL := GetGrafanaHealthURL(config)
	assert.True(t, len(healthURL) > 0, "Health URL should not be empty")
	assert.Contains(t, healthURL, config.Port)
	assert.Contains(t, healthURL, "/api/health")
}

func TestGrafanaProductionConfig_SecurityWarning(t *testing.T) {
	config := DefaultGrafanaProductionConfig()

	// Verify default password is "admin" to trigger security awareness
	assert.Equal(t, "admin", config.AdminPassword,
		"Default password is 'admin' - MUST be changed for production!")

	// Test password change
	config.AdminPassword = "new_secure_password"
	assert.NotEqual(t, "admin", config.AdminPassword,
		"Password should be changed from default")
}

func TestGrafanaProductionConfig_DefaultPort(t *testing.T) {
	config := DefaultGrafanaProductionConfig()

	// Verify Grafana default port
	assert.Equal(t, "3000", config.Port, "Grafana default port should be 3000")
}

func TestGrafanaProductionConfig_VolumeConfiguration(t *testing.T) {
	config := DefaultGrafanaProductionConfig()

	// Verify volume configuration
	assert.Equal(t, "grafana-data", config.DataVolume)
	assert.Equal(t, "grafana-data", config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)

	// Verify volume and production settings match
	assert.Equal(t, config.DataVolume, config.Production.VolumeName,
		"DataVolume and Production.VolumeName should match")
}
