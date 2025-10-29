package production

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultRabbitMQProductionConfig(t *testing.T) {
	config := DefaultRabbitMQProductionConfig()

	assert.Equal(t, "rabbitmq", config.ContainerName)
	assert.Equal(t, "rabbitmq:4.1.0-management", config.Image)
	assert.Equal(t, "5672", config.AMQPPort)
	assert.Equal(t, "15672", config.ManagementPort)
	assert.Equal(t, "guest", config.Username)
	assert.Equal(t, "changeme", config.Password)
	assert.Equal(t, "rabbitmq-data", config.DataVolume)
	assert.Equal(t, "app-network", config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)
	assert.Equal(t, "rabbitmq-data", config.Production.VolumeName)
	assert.True(t, config.Production.CreateVolume)
}

func TestRabbitMQProductionConfig_CustomConfiguration(t *testing.T) {
	config := DefaultRabbitMQProductionConfig()

	// Modify configuration
	config.ContainerName = "my-rabbitmq"
	config.AMQPPort = "5673"
	config.ManagementPort = "15673"
	config.Username = "admin"
	config.Password = "secure-password"
	config.DataVolume = "custom-rabbitmq-data"

	assert.Equal(t, "my-rabbitmq", config.ContainerName)
	assert.Equal(t, "5673", config.AMQPPort)
	assert.Equal(t, "15673", config.ManagementPort)
	assert.Equal(t, "admin", config.Username)
	assert.Equal(t, "secure-password", config.Password)
	assert.Equal(t, "custom-rabbitmq-data", config.DataVolume)
}

func TestGetRabbitMQAMQPURL(t *testing.T) {
	tests := []struct {
		name     string
		config   RabbitMQProductionConfig
		vhost    string
		expected string
	}{
		{
			name: "default vhost",
			config: RabbitMQProductionConfig{
				AMQPPort: "5672",
				Username: "guest",
				Password: "guest",
			},
			vhost:    "/",
			expected: "amqp://guest:guest@localhost:5672/",
		},
		{
			name: "empty vhost defaults to /",
			config: RabbitMQProductionConfig{
				AMQPPort: "5672",
				Username: "guest",
				Password: "guest",
			},
			vhost:    "",
			expected: "amqp://guest:guest@localhost:5672/",
		},
		{
			name: "custom vhost",
			config: RabbitMQProductionConfig{
				AMQPPort: "5672",
				Username: "admin",
				Password: "secret",
			},
			vhost:    "production",
			expected: "amqp://admin:secret@localhost:5672/production",
		},
		{
			name: "custom port and credentials",
			config: RabbitMQProductionConfig{
				AMQPPort: "5673",
				Username: "appuser",
				Password: "apppass",
			},
			vhost:    "app-vhost",
			expected: "amqp://appuser:apppass@localhost:5673/app-vhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amqpURL := GetRabbitMQAMQPURL(tt.config, tt.vhost)
			assert.Equal(t, tt.expected, amqpURL)
		})
	}
}

func TestGetRabbitMQManagementURL(t *testing.T) {
	tests := []struct {
		name     string
		config   RabbitMQProductionConfig
		expected string
	}{
		{
			name: "default port",
			config: RabbitMQProductionConfig{
				ManagementPort: "15672",
			},
			expected: "http://localhost:15672",
		},
		{
			name: "custom port",
			config: RabbitMQProductionConfig{
				ManagementPort: "15673",
			},
			expected: "http://localhost:15673",
		},
		{
			name: "high port number",
			config: RabbitMQProductionConfig{
				ManagementPort: "25672",
			},
			expected: "http://localhost:25672",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			managementURL := GetRabbitMQManagementURL(tt.config)
			assert.Equal(t, tt.expected, managementURL)
		})
	}
}

func TestRabbitMQProductionConfig_PasswordSecurity(t *testing.T) {
	config := DefaultRabbitMQProductionConfig()

	// Default password should be "changeme" as a reminder
	assert.Equal(t, "changeme", config.Password, "Default config should have changeme password as reminder")

	// Production should always set a strong password
	config.Password = "production-secure-password-with-special-chars-!@#$%"
	assert.NotEmpty(t, config.Password, "Production config should have a password")
	assert.Greater(t, len(config.Password), 10, "Password should be reasonably long")
	assert.NotEqual(t, "changeme", config.Password, "Production password should be changed from default")
}

func TestRabbitMQProductionConfig_DataPersistence(t *testing.T) {
	config := DefaultRabbitMQProductionConfig()

	// Verify data volume is configured for persistence
	assert.NotEmpty(t, config.DataVolume, "Data volume should be configured")
	assert.Equal(t, "rabbitmq-data", config.DataVolume, "Default data volume name")

	// Test custom volume
	config.DataVolume = "production-rabbitmq-volume"
	assert.Equal(t, "production-rabbitmq-volume", config.DataVolume)
}

func TestRabbitMQProductionConfig_PortConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		amqpPort       string
		managementPort string
		description    string
	}{
		{
			name:           "default ports",
			amqpPort:       "5672",
			managementPort: "15672",
			description:    "RabbitMQ default ports",
		},
		{
			name:           "custom ports",
			amqpPort:       "5673",
			managementPort: "15673",
			description:    "Alternative ports to avoid conflicts",
		},
		{
			name:           "high ports",
			amqpPort:       "25672",
			managementPort: "35672",
			description:    "High port numbers for security",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultRabbitMQProductionConfig()
			config.AMQPPort = tt.amqpPort
			config.ManagementPort = tt.managementPort

			assert.Equal(t, tt.amqpPort, config.AMQPPort, tt.description)
			assert.Equal(t, tt.managementPort, config.ManagementPort, tt.description)
		})
	}
}

func TestRabbitMQProductionConfig_VirtualHosts(t *testing.T) {
	tests := []struct {
		name  string
		vhost string
		valid bool
	}{
		{
			name:  "default vhost",
			vhost: "/",
			valid: true,
		},
		{
			name:  "simple vhost",
			vhost: "production",
			valid: true,
		},
		{
			name:  "vhost with hyphen",
			vhost: "app-production",
			valid: true,
		},
		{
			name:  "vhost with underscore",
			vhost: "app_vhost",
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultRabbitMQProductionConfig()
			amqpURL := GetRabbitMQAMQPURL(config, tt.vhost)

			assert.NotEmpty(t, amqpURL)
			if tt.valid {
				assert.Contains(t, amqpURL, "amqp://")
			}
		})
	}
}

func TestRabbitMQProductionConfig_NetworkIsolation(t *testing.T) {
	config := DefaultRabbitMQProductionConfig()

	// Verify network configuration
	assert.NotEmpty(t, config.Production.NetworkName)
	assert.True(t, config.Production.CreateNetwork)

	// Test custom network
	config.Production.NetworkName = "messaging-network"
	assert.Equal(t, "messaging-network", config.Production.NetworkName)
}

func TestRabbitMQProductionConfig_RestartPolicy(t *testing.T) {
	config := DefaultRabbitMQProductionConfig()

	// The restart policy is set in DeployRabbitMQ to "unless-stopped"
	// This test just validates the config structure is ready
	assert.NotEmpty(t, config.ContainerName, "Container name is required for restart policy")
}

func TestRabbitMQProductionConfig_ManagementPlugin(t *testing.T) {
	config := DefaultRabbitMQProductionConfig()

	// Verify that management image is being used
	assert.Contains(t, config.Image, "management", "Should use rabbitmq image with management plugin")
	assert.Equal(t, "rabbitmq:4.1.0-management", config.Image)
}

func TestRabbitMQProductionConfig_Credentials(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		valid    bool
	}{
		{
			name:     "default credentials",
			username: "guest",
			password: "guest",
			valid:    true,
		},
		{
			name:     "admin credentials",
			username: "admin",
			password: "admin123",
			valid:    true,
		},
		{
			name:     "custom user",
			username: "appuser",
			password: "apppass",
			valid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultRabbitMQProductionConfig()
			config.Username = tt.username
			config.Password = tt.password

			assert.Equal(t, tt.username, config.Username)
			assert.Equal(t, tt.password, config.Password)
			if tt.valid {
				assert.NotEmpty(t, config.Username)
				assert.NotEmpty(t, config.Password)
			}
		})
	}
}
