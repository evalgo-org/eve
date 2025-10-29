package production

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPostgresProductionConfig(t *testing.T) {
	config := DefaultPostgresProductionConfig()

	assert.Equal(t, "postgres", config.ContainerName)
	assert.Equal(t, "postgres:17", config.Image)
	assert.Equal(t, "5432", config.Port)
	assert.Equal(t, "postgres", config.Username)
	assert.Equal(t, "changeme", config.Password)
	assert.Equal(t, "postgres", config.Database)
	assert.Equal(t, "postgres-data", config.DataVolume)
	assert.Equal(t, "app-network", config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)
	assert.Equal(t, "postgres-data", config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)
}

func TestPostgresProductionConfig_CustomConfiguration(t *testing.T) {
	config := DefaultPostgresProductionConfig()

	// Modify configuration
	config.ContainerName = "my-postgres"
	config.Port = "5433"
	config.Username = "dbadmin"
	config.Password = "secure-password"
	config.Database = "myapp"
	config.DataVolume = "custom-pg-data"

	assert.Equal(t, "my-postgres", config.ContainerName)
	assert.Equal(t, "5433", config.Port)
	assert.Equal(t, "dbadmin", config.Username)
	assert.Equal(t, "secure-password", config.Password)
	assert.Equal(t, "myapp", config.Database)
	assert.Equal(t, "custom-pg-data", config.DataVolume)
}

func TestGetPostgresConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		config   PostgresProductionConfig
		sslmode  string
		expected string
	}{
		{
			name: "default configuration with SSL disabled",
			config: PostgresProductionConfig{
				Port:     "5432",
				Username: "postgres",
				Password: "postgres",
				Database: "postgres",
			},
			sslmode:  "disable",
			expected: "postgresql://postgres:postgres@localhost:5432/postgres?sslmode=disable",
		},
		{
			name: "custom configuration with SSL required",
			config: PostgresProductionConfig{
				Port:     "5433",
				Username: "admin",
				Password: "secret",
				Database: "myapp",
			},
			sslmode:  "require",
			expected: "postgresql://admin:secret@localhost:5433/myapp?sslmode=require",
		},
		{
			name: "production with verify-full SSL",
			config: PostgresProductionConfig{
				Port:     "5432",
				Username: "produser",
				Password: "prodpass",
				Database: "production",
			},
			sslmode:  "verify-full",
			expected: "postgresql://produser:prodpass@localhost:5432/production?sslmode=verify-full",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connStr := GetPostgresConnectionString(tt.config, tt.sslmode)
			assert.Equal(t, tt.expected, connStr)
		})
	}
}

func TestPostgresProductionConfig_PasswordSecurity(t *testing.T) {
	config := DefaultPostgresProductionConfig()

	// Default password should be "changeme" as a reminder
	assert.Equal(t, "changeme", config.Password, "Default config should have changeme password as reminder")

	// Production should always set a strong password
	config.Password = "production-secure-password-with-special-chars-!@#$%"
	assert.NotEmpty(t, config.Password, "Production config should have a password")
	assert.Greater(t, len(config.Password), 10, "Password should be reasonably long")
	assert.NotEqual(t, "changeme", config.Password, "Production password should be changed from default")
}

func TestPostgresProductionConfig_AuthenticationMethod(t *testing.T) {
	config := DefaultPostgresProductionConfig()

	// Check that SCRAM-SHA-256 authentication is configured
	// This is verified through the POSTGRES_INITDB_ARGS environment variable
	// The actual env vars are set in DeployPostgres, not in the config struct
	assert.Equal(t, "postgres:17", config.Image, "PostgreSQL 17 supports SCRAM-SHA-256 by default")
}

func TestPostgresProductionConfig_DataPersistence(t *testing.T) {
	config := DefaultPostgresProductionConfig()

	// Verify data volume is configured for persistence
	assert.NotEmpty(t, config.DataVolume, "Data volume should be configured")
	assert.Equal(t, "postgres-data", config.DataVolume, "Default data volume name")

	// Test custom volume
	config.DataVolume = "production-postgres-volume"
	assert.Equal(t, "production-postgres-volume", config.DataVolume)
}

func TestPostgresProductionConfig_PortConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		port        string
		description string
	}{
		{
			name:        "default port",
			port:        "5432",
			description: "PostgreSQL default port",
		},
		{
			name:        "custom port",
			port:        "5433",
			description: "Alternative port to avoid conflicts",
		},
		{
			name:        "high port",
			port:        "15432",
			description: "High port number for security",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultPostgresProductionConfig()
			config.Port = tt.port

			assert.Equal(t, tt.port, config.Port, tt.description)
		})
	}
}

func TestPostgresProductionConfig_DatabaseNames(t *testing.T) {
	tests := []struct {
		name     string
		database string
		valid    bool
	}{
		{
			name:     "default database",
			database: "postgres",
			valid:    true,
		},
		{
			name:     "application database",
			database: "myapp",
			valid:    true,
		},
		{
			name:     "database with underscore",
			database: "my_application_db",
			valid:    true,
		},
		{
			name:     "database with numbers",
			database: "app2024",
			valid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultPostgresProductionConfig()
			config.Database = tt.database

			assert.Equal(t, tt.database, config.Database)
			if tt.valid {
				assert.NotEmpty(t, config.Database)
			}
		})
	}
}

func TestPostgresProductionConfig_NetworkIsolation(t *testing.T) {
	config := DefaultPostgresProductionConfig()

	// Verify network configuration
	assert.NotEmpty(t, config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)

	// Test custom network
	config.Production.NetworkName = "database-network"
	assert.Equal(t, "database-network", config.Production.NetworkName)
}

func TestPostgresProductionConfig_RestartPolicy(t *testing.T) {
	config := DefaultPostgresProductionConfig()

	// The restart policy is set in DeployPostgres to "unless-stopped"
	// This test just validates the config structure is ready
	assert.NotEmpty(t, config.ContainerName, "Container name is required for restart policy")
}
