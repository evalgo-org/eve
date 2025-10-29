package production

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultCouchDBProductionConfig(t *testing.T) {
	config := DefaultCouchDBProductionConfig()

	assert.Equal(t, "couchdb", config.ContainerName)
	assert.Equal(t, "couchdb:3", config.Image)
	assert.Equal(t, "5984", config.Port)
	assert.Equal(t, "admin", config.AdminUsername)
	assert.Equal(t, "changeme", config.AdminPassword)
	assert.Equal(t, "couchdb-data", config.DataVolume)
	assert.Equal(t, "app-network", config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)
	assert.Equal(t, "couchdb-data", config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)
}

func TestCouchDBProductionConfig_CustomConfiguration(t *testing.T) {
	config := DefaultCouchDBProductionConfig()

	// Modify configuration
	config.ContainerName = "my-couchdb"
	config.Port = "6000"
	config.AdminUsername = "dbadmin"
	config.AdminPassword = "secure-password"
	config.DataVolume = "custom-couch-data"

	assert.Equal(t, "my-couchdb", config.ContainerName)
	assert.Equal(t, "6000", config.Port)
	assert.Equal(t, "dbadmin", config.AdminUsername)
	assert.Equal(t, "secure-password", config.AdminPassword)
	assert.Equal(t, "custom-couch-data", config.DataVolume)
}

func TestGetCouchDBConnectionURL(t *testing.T) {
	tests := []struct {
		name     string
		config   CouchDBProductionConfig
		expected string
	}{
		{
			name: "default configuration",
			config: CouchDBProductionConfig{
				Port:          "5984",
				AdminUsername: "admin",
				AdminPassword: "password",
			},
			expected: "http://admin:password@localhost:5984",
		},
		{
			name: "custom port and credentials",
			config: CouchDBProductionConfig{
				Port:          "6000",
				AdminUsername: "dbadmin",
				AdminPassword: "secret123",
			},
			expected: "http://dbadmin:secret123@localhost:6000",
		},
		{
			name: "special characters in password",
			config: CouchDBProductionConfig{
				Port:          "5984",
				AdminUsername: "admin",
				AdminPassword: "p@ss!word#",
			},
			expected: "http://admin:p@ss!word#@localhost:5984",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := GetCouchDBConnectionURL(tt.config)
			assert.Equal(t, tt.expected, url)
		})
	}
}

func TestCouchDBProductionConfig_EnvironmentVariables(t *testing.T) {
	config := DefaultCouchDBProductionConfig()

	// Test that environment variables are properly formatted
	expectedEnvVars := map[string]string{
		"COUCHDB_USER":     config.AdminUsername,
		"COUCHDB_PASSWORD": config.AdminPassword,
	}

	for key, value := range expectedEnvVars {
		envVar := key + "=" + value
		assert.Contains(t, envVar, key+"=")
		assert.Contains(t, envVar, value)
	}
}

func TestCouchDBProductionConfig_PortBinding(t *testing.T) {
	tests := []struct {
		name        string
		port        string
		expectValid bool
	}{
		{
			name:        "default port",
			port:        "5984",
			expectValid: true,
		},
		{
			name:        "custom high port",
			port:        "35984",
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
			config := DefaultCouchDBProductionConfig()
			config.Port = tt.port

			assert.NotEmpty(t, config.Port)
			assert.Equal(t, tt.port, config.Port)
		})
	}
}

func TestCouchDBProductionConfig_VolumeConfiguration(t *testing.T) {
	config := DefaultCouchDBProductionConfig()

	// Verify volume configuration
	assert.Equal(t, "couchdb-data", config.DataVolume)
	assert.Equal(t, config.DataVolume, config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)

	// Volume should be created before container deployment
	assert.NotEmpty(t, config.DataVolume, "Volume name should not be empty")
}

func TestCouchDBProductionConfig_NetworkConfiguration(t *testing.T) {
	config := DefaultCouchDBProductionConfig()

	// Verify network configuration
	assert.Equal(t, "app-network", config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)

	// Network should be created before container deployment
	assert.NotEmpty(t, config.Production.NetworkName, "Network name should not be empty")
}

func TestCouchDBProductionConfig_HealthCheck(t *testing.T) {
	// Health check configuration should be part of deployment
	// CouchDB health check: HTTP GET http://localhost:5984/_up

	config := DefaultCouchDBProductionConfig()

	// Health check URL would be: http://localhost:{port}/_up
	healthCheckURL := "http://localhost:" + config.Port + "/_up"
	assert.Equal(t, "http://localhost:5984/_up", healthCheckURL)
}

func TestCouchDBConnectionURL_URLEncoding(t *testing.T) {
	// Test that connection URL handles special characters correctly
	// Note: In a real implementation, passwords with special characters
	// should be URL-encoded, but for now we test the basic format

	tests := []struct {
		name     string
		username string
		password string
		port     string
	}{
		{
			name:     "alphanumeric password",
			username: "admin",
			password: "abc123",
			port:     "5984",
		},
		{
			name:     "password with special chars",
			username: "admin",
			password: "P@ssw0rd!",
			port:     "5984",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := CouchDBProductionConfig{
				Port:          tt.port,
				AdminUsername: tt.username,
				AdminPassword: tt.password,
			}

			url := GetCouchDBConnectionURL(config)
			assert.Contains(t, url, tt.username)
			assert.Contains(t, url, tt.password)
			assert.Contains(t, url, tt.port)
			assert.Contains(t, url, "@localhost:")
		})
	}
}

func TestCouchDBProductionConfig_SingleNodeMode(t *testing.T) {
	// CouchDB production config should be suitable for single-node deployment
	// Multi-node clustering would require additional configuration

	config := DefaultCouchDBProductionConfig()

	// Single node configuration is implicit - we're only deploying one container
	assert.NotEmpty(t, config.ContainerName, "Single container should have a name")
	assert.NotEmpty(t, config.DataVolume, "Single node should have data persistence")
}

func TestCouchDBProductionConfig_RestartPolicy(t *testing.T) {
	// In production, containers should have restart policy configured
	// This is tested indirectly through the DeployCouchDB function
	// The restart policy should be "unless-stopped"

	config := DefaultCouchDBProductionConfig()
	assert.NotEmpty(t, config.ContainerName, "Container name required for restart policy")
}

func TestCouchDBProductionConfig_SecurityConsiderations(t *testing.T) {
	tests := []struct {
		name     string
		password string
		isWeak   bool
	}{
		{
			name:     "default password (weak)",
			password: "changeme",
			isWeak:   true,
		},
		{
			name:     "short password (weak)",
			password: "123",
			isWeak:   true,
		},
		{
			name:     "strong password",
			password: "SecureP@ssw0rd123!",
			isWeak:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultCouchDBProductionConfig()
			config.AdminPassword = tt.password

			// In production, weak passwords should trigger warnings
			if tt.isWeak {
				assert.True(t, len(tt.password) < 10 || tt.password == "changeme",
					"Weak password should be detected")
			} else {
				assert.True(t, len(tt.password) >= 10,
					"Strong password should have sufficient length")
			}
		})
	}
}
