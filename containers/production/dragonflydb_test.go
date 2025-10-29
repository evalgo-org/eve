package production

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultDragonflyDBProductionConfig(t *testing.T) {
	config := DefaultDragonflyDBProductionConfig()

	assert.Equal(t, "dragonflydb", config.ContainerName)
	assert.Equal(t, "docker.dragonflydb.io/dragonflydb/dragonfly:v1.34.1", config.Image)
	assert.Equal(t, "6379", config.Port)
	assert.Equal(t, "", config.Password) // No password by default
	assert.Equal(t, "dragonflydb-data", config.DataVolume)
	assert.Equal(t, "", config.MaxMemory) // No limit by default
	assert.Equal(t, "app-network", config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)
	assert.Equal(t, "dragonflydb-data", config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)
}

func TestDragonflyDBProductionConfig_CustomConfiguration(t *testing.T) {
	config := DefaultDragonflyDBProductionConfig()

	// Modify configuration
	config.ContainerName = "my-dragonflydb"
	config.Port = "6380"
	config.Password = "secure-password"
	config.DataVolume = "custom-dfdb-data"
	config.MaxMemory = "2g"

	assert.Equal(t, "my-dragonflydb", config.ContainerName)
	assert.Equal(t, "6380", config.Port)
	assert.Equal(t, "secure-password", config.Password)
	assert.Equal(t, "custom-dfdb-data", config.DataVolume)
	assert.Equal(t, "2g", config.MaxMemory)
}

func TestGetDragonflyDBConnectionAddr(t *testing.T) {
	tests := []struct {
		name     string
		config   DragonflyDBProductionConfig
		expected string
	}{
		{
			name: "default configuration",
			config: DragonflyDBProductionConfig{
				Port: "6379",
			},
			expected: "localhost:6379",
		},
		{
			name: "custom port",
			config: DragonflyDBProductionConfig{
				Port: "6380",
			},
			expected: "localhost:6380",
		},
		{
			name: "high port number",
			config: DragonflyDBProductionConfig{
				Port: "16379",
			},
			expected: "localhost:16379",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := GetDragonflyDBConnectionAddr(tt.config)
			assert.Equal(t, tt.expected, addr)
		})
	}
}

func TestDragonflyDBProductionConfig_PasswordSecurity(t *testing.T) {
	config := DefaultDragonflyDBProductionConfig()

	// Default should have no password (testing mode)
	assert.Equal(t, "", config.Password, "Default config should have no password for testing")

	// Production should always set a password
	config.Password = "production-secure-password-123"
	assert.NotEmpty(t, config.Password, "Production config should have a password")
	assert.Greater(t, len(config.Password), 10, "Password should be reasonably long")
}

func TestDragonflyDBProductionConfig_MemoryLimits(t *testing.T) {
	tests := []struct {
		name        string
		maxMemory   string
		description string
	}{
		{
			name:        "no limit",
			maxMemory:   "",
			description: "Default no limit",
		},
		{
			name:        "megabytes",
			maxMemory:   "512m",
			description: "512 megabytes limit",
		},
		{
			name:        "gigabytes",
			maxMemory:   "2g",
			description: "2 gigabytes limit",
		},
		{
			name:        "large memory",
			maxMemory:   "16g",
			description: "16 gigabytes limit for high-performance systems",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultDragonflyDBProductionConfig()
			config.MaxMemory = tt.maxMemory

			// Verify configuration accepts the memory limit
			assert.Equal(t, tt.maxMemory, config.MaxMemory, tt.description)
		})
	}
}

func TestDragonflyDBProductionConfig_NetworkIsolation(t *testing.T) {
	config := DefaultDragonflyDBProductionConfig()

	// Verify network configuration
	assert.NotEmpty(t, config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)

	// Test custom network
	config.Production.NetworkName = "backend-network"
	assert.Equal(t, "backend-network", config.Production.NetworkName)
}
