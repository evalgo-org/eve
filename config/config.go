// Package config provides comprehensive configuration management for EVE services.
//
// This package handles loading configuration from multiple sources with proper precedence:
//   - YAML configuration files
//   - Environment variables (configurable prefix)
//   - .env files
//   - Default values
//
// # Configuration Sources Priority
//
// Configuration is loaded in the following order (later sources override earlier ones):
//  1. Default values (set via SetDefaults)
//  2. Configuration files (./config.yaml, ./configs/config.yaml, ~/.eve/config.yaml, /etc/eve/config.yaml)
//  3. .env files
//  4. Environment variables (configurable prefix, default: EVE_)
//
// # Usage Example
//
//	cfg, err := config.LoadConfig("myservice", "config.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
//
// # Environment Variables
//
// Environment variables override all other configuration sources.
// Use prefix and underscores for nested keys:
//   - MYSERVICE_SERVER_PORT=8095
//   - MYSERVICE_DATABASE_URL=http://localhost:5984
//   - MYSERVICE_DEBUG=true
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// ServerConfig contains HTTP server configuration.
type ServerConfig struct {
	// Host is the server bind address (default: 0.0.0.0)
	Host string `mapstructure:"host"`

	// Port is the server listen port (default: 8080)
	Port int `mapstructure:"port"`

	// ReadTimeout is the maximum duration for reading requests
	ReadTimeout time.Duration `mapstructure:"read_timeout"`

	// WriteTimeout is the maximum duration for writing responses
	WriteTimeout time.Duration `mapstructure:"write_timeout"`

	// ShutdownTimeout is the maximum duration for graceful shutdown
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`

	// Debug enables debug logging and additional endpoints
	Debug bool `mapstructure:"debug"`

	// TLSEnabled enables HTTPS
	TLSEnabled bool `mapstructure:"tls_enabled"`

	// TLSCert is the path to the TLS certificate file
	TLSCert string `mapstructure:"tls_cert"`

	// TLSKey is the path to the TLS private key file
	TLSKey string `mapstructure:"tls_key"`
}

// DatabaseConfig contains database connection settings.
type DatabaseConfig struct {
	// URL is the database server URL (e.g., http://localhost:5984)
	URL string `mapstructure:"url"`

	// Database is the database name to use
	Database string `mapstructure:"database"`

	// Username for database authentication
	Username string `mapstructure:"username"`

	// Password for database authentication
	Password string `mapstructure:"password"`

	// MaxConnections is the maximum number of concurrent connections
	MaxConnections int `mapstructure:"max_connections"`

	// Timeout in seconds for database operations
	Timeout int `mapstructure:"timeout"`

	// CreateIfMissing automatically creates database if it doesn't exist
	CreateIfMissing bool `mapstructure:"create_if_missing"`
}

// RegistryConfig contains service registry configuration.
type RegistryConfig struct {
	// URL is the registry service URL
	URL string `mapstructure:"url"`

	// HeartbeatInterval is the duration between heartbeats
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`

	// Timeout for registry operations
	Timeout time.Duration `mapstructure:"timeout"`
}

// LoggingConfig contains logging configuration.
type LoggingConfig struct {
	// Level is the log level (debug, info, warn, error)
	Level string `mapstructure:"level"`

	// Format is the log format (json, text)
	Format string `mapstructure:"format"`

	// Output is the log output destination (stdout, stderr, file)
	Output string `mapstructure:"output"`

	// MaxSize is the maximum log file size in megabytes
	MaxSize int `mapstructure:"max_size"`

	// MaxBackups is the maximum number of old log files to keep
	MaxBackups int `mapstructure:"max_backups"`

	// MaxAge is the maximum number of days to keep old log files
	MaxAge int `mapstructure:"max_age"`
}

// SecurityConfig contains security and authentication settings.
type SecurityConfig struct {
	// RateLimit is the maximum requests per second per client
	RateLimit int `mapstructure:"rate_limit"`

	// AllowedOrigins are the CORS allowed origins
	AllowedOrigins []string `mapstructure:"allowed_origins"`

	// APIKey for simple API key authentication
	APIKey string `mapstructure:"api_key"`

	// JWTSecret is the secret key for signing JWT tokens
	JWTSecret string `mapstructure:"jwt_secret"`

	// JWTExpiration is the JWT token expiration duration (default: 24h)
	JWTExpiration time.Duration `mapstructure:"jwt_expiration"`

	// RefreshTokenExpiration is the refresh token expiration duration (default: 7 days)
	RefreshTokenExpiration time.Duration `mapstructure:"refresh_token_expiration"`
}

// ServiceConfig contains service-specific metadata.
type ServiceConfig struct {
	// Name is the service name
	Name string `mapstructure:"name"`

	// Version is the service version
	Version string `mapstructure:"version"`

	// Environment is the deployment environment (development, staging, production)
	Environment string `mapstructure:"environment"`
}

// Config is a flexible configuration structure for EVE services.
// Services can embed this or use only the sections they need.
type Config struct {
	// Service contains service metadata
	Service ServiceConfig `mapstructure:"service"`

	// Server contains HTTP server configuration
	Server ServerConfig `mapstructure:"server"`

	// Database contains database connection settings
	Database DatabaseConfig `mapstructure:"database"`

	// Registry contains service registry settings
	Registry RegistryConfig `mapstructure:"registry"`

	// Logging contains logging settings
	Logging LoggingConfig `mapstructure:"logging"`

	// Security contains security settings
	Security SecurityConfig `mapstructure:"security"`
}

// Loader provides configuration loading functionality.
type Loader struct {
	v      *viper.Viper
	prefix string
}

// NewLoader creates a new configuration loader with the given environment prefix.
// The prefix is used for environment variables (e.g., "MYSERVICE" -> "MYSERVICE_SERVER_PORT").
func NewLoader(envPrefix string) *Loader {
	return &Loader{
		v:      viper.New(),
		prefix: envPrefix,
	}
}

// SetDefaults sets default configuration values.
// This should be called before Load().
func (l *Loader) SetDefaults(defaults map[string]interface{}) {
	for key, value := range defaults {
		l.v.SetDefault(key, value)
	}
}

// SetConfigDefaults sets standard EVE service defaults.
func (l *Loader) SetConfigDefaults() {
	l.v.SetDefault("server.host", "0.0.0.0")
	l.v.SetDefault("server.port", 8080)
	l.v.SetDefault("server.read_timeout", "30s")
	l.v.SetDefault("server.write_timeout", "30s")
	l.v.SetDefault("server.shutdown_timeout", "10s")
	l.v.SetDefault("server.debug", false)
	l.v.SetDefault("server.tls_enabled", false)

	l.v.SetDefault("database.url", "http://localhost:5984")
	l.v.SetDefault("database.database", "")
	l.v.SetDefault("database.username", "")
	l.v.SetDefault("database.password", "")
	l.v.SetDefault("database.max_connections", 10)
	l.v.SetDefault("database.timeout", 30)
	l.v.SetDefault("database.create_if_missing", true)

	l.v.SetDefault("registry.url", "http://localhost:8096")
	l.v.SetDefault("registry.heartbeat_interval", "30s")
	l.v.SetDefault("registry.timeout", "10s")

	l.v.SetDefault("logging.level", "info")
	l.v.SetDefault("logging.format", "json")
	l.v.SetDefault("logging.output", "stdout")
	l.v.SetDefault("logging.max_size", 100)
	l.v.SetDefault("logging.max_backups", 3)
	l.v.SetDefault("logging.max_age", 7)

	l.v.SetDefault("security.rate_limit", 100)
	l.v.SetDefault("security.allowed_origins", []string{"*"})
	l.v.SetDefault("security.jwt_expiration", "24h")
	l.v.SetDefault("security.refresh_token_expiration", "168h") // 7 days
}

// Load reads configuration from file, .env, and environment variables.
// If cfgFile is empty, searches for config.yaml in standard locations.
//
// Configuration precedence (highest to lowest):
//  1. Environment variables (with prefix)
//  2. .env file
//  3. Configuration file
//  4. Default values
func (l *Loader) Load(cfgFile string, target interface{}) error {
	// Set config file
	if cfgFile != "" {
		l.v.SetConfigFile(cfgFile)
	} else {
		l.v.SetConfigName("config")
		l.v.SetConfigType("yaml")
		l.v.AddConfigPath(".")
		l.v.AddConfigPath("./configs")
		l.v.AddConfigPath("$HOME/.eve")
		l.v.AddConfigPath("/etc/eve")
	}

	// Read config file
	if err := l.v.ReadInConfig(); err != nil {
		// Only fail on non-NotFound errors for explicit file paths
		if cfgFile != "" && !isFileNotFoundError(err) {
			return fmt.Errorf("error reading config file: %w", err)
		}
		// For auto-discovery, only fail on non-NotFound errors
		if cfgFile == "" {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return fmt.Errorf("error reading config file: %w", err)
			}
		}
	}

	// Merge .env file if present
	l.v.SetConfigFile(".env")
	l.v.SetConfigType("env")
	_ = l.v.MergeInConfig() // Ignore if .env doesn't exist

	// Setup environment variable binding
	if l.prefix != "" {
		l.v.SetEnvPrefix(l.prefix)
	}
	l.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	l.v.AutomaticEnv()

	// Unmarshal into target
	if err := l.v.Unmarshal(target); err != nil {
		return fmt.Errorf("unable to decode config: %w", err)
	}

	return nil
}

// LoadConfig is a convenience function that loads configuration with standard defaults.
// The envPrefix is used for environment variables (e.g., "MYSERVICE" -> "MYSERVICE_SERVER_PORT").
func LoadConfig(envPrefix, cfgFile string) (*Config, error) {
	loader := NewLoader(envPrefix)
	loader.SetConfigDefaults()

	cfg := &Config{}
	if err := loader.Load(cfgFile, cfg); err != nil {
		return nil, err
	}

	if err := ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// ValidateConfig validates the loaded configuration.
func ValidateConfig(cfg *Config) error {
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Server.Port)
	}

	// Only validate database if it's configured
	if cfg.Database.Database != "" {
		if cfg.Database.URL == "" {
			return fmt.Errorf("database url is required when database is specified")
		}
	}

	return nil
}

// BuildDatabaseURL constructs the full database URL with authentication.
func (c *DatabaseConfig) BuildURL() string {
	if c.Username != "" && c.Password != "" {
		url := strings.Replace(c.URL, "://", "://"+c.Username+":"+c.Password+"@", 1)
		return url
	}
	return c.URL
}

// isFileNotFoundError checks if an error is a file not found error.
func isFileNotFoundError(err error) bool {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return errors.Is(pathErr, os.ErrNotExist)
	}
	return false
}
