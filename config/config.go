// Package config provides common configuration loading and management utilities for EVE services.
// This package includes standard environment variable loading, validation, and
// configuration patterns used across the EVE ecosystem.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// EnvConfig provides utilities for loading configuration from environment variables
type EnvConfig struct {
	prefix string // Optional prefix for all environment variables
}

// NewEnvConfig creates a new environment configuration loader
func NewEnvConfig(prefix string) *EnvConfig {
	return &EnvConfig{
		prefix: prefix,
	}
}

// GetString retrieves a string value from environment with optional default
func (ec *EnvConfig) GetString(key, defaultValue string) string {
	fullKey := ec.buildKey(key)
	if value := os.Getenv(fullKey); value != "" {
		return value
	}
	return defaultValue
}

// MustGetString retrieves a required string value from environment or panics
func (ec *EnvConfig) MustGetString(key string) string {
	fullKey := ec.buildKey(key)
	value := os.Getenv(fullKey)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s not set", fullKey))
	}
	return value
}

// GetInt retrieves an integer value from environment with optional default
func (ec *EnvConfig) GetInt(key string, defaultValue int) int {
	fullKey := ec.buildKey(key)
	if value := os.Getenv(fullKey); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// MustGetInt retrieves a required integer value from environment or panics
func (ec *EnvConfig) MustGetInt(key string) int {
	fullKey := ec.buildKey(key)
	value := os.Getenv(fullKey)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s not set", fullKey))
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf("environment variable %s is not a valid integer: %v", fullKey, err))
	}
	return intValue
}

// GetBool retrieves a boolean value from environment with optional default
func (ec *EnvConfig) GetBool(key string, defaultValue bool) bool {
	fullKey := ec.buildKey(key)
	if value := os.Getenv(fullKey); value != "" {
		boolValue, err := strconv.ParseBool(value)
		if err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// GetDuration retrieves a duration value from environment with optional default
func (ec *EnvConfig) GetDuration(key string, defaultValue time.Duration) time.Duration {
	fullKey := ec.buildKey(key)
	if value := os.Getenv(fullKey); value != "" {
		duration, err := time.ParseDuration(value)
		if err == nil {
			return duration
		}
	}
	return defaultValue
}

// GetStringSlice retrieves a comma-separated string slice from environment
func (ec *EnvConfig) GetStringSlice(key string, defaultValue []string) []string {
	fullKey := ec.buildKey(key)
	if value := os.Getenv(fullKey); value != "" {
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	return defaultValue
}

// buildKey builds the full environment variable key with optional prefix
func (ec *EnvConfig) buildKey(key string) string {
	if ec.prefix != "" {
		return ec.prefix + "_" + key
	}
	return key
}

// ServerConfig contains common server configuration
type ServerConfig struct {
	Port            int
	Host            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	Debug           bool
}

// LoadServerConfig loads server configuration from environment
func LoadServerConfig(prefix string) ServerConfig {
	env := NewEnvConfig(prefix)
	return ServerConfig{
		Port:            env.GetInt("PORT", 8080),
		Host:            env.GetString("HOST", "0.0.0.0"),
		ReadTimeout:     env.GetDuration("READ_TIMEOUT", 30*time.Second),
		WriteTimeout:    env.GetDuration("WRITE_TIMEOUT", 30*time.Second),
		ShutdownTimeout: env.GetDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		Debug:           env.GetBool("DEBUG", false),
	}
}

// DatabaseConfig contains common database configuration
type DatabaseConfig struct {
	URL             string
	Database        string
	Username        string
	Password        string
	MaxConnections  int
	Timeout         time.Duration
	CreateIfMissing bool
}

// LoadDatabaseConfig loads database configuration from environment
func LoadDatabaseConfig(prefix string) DatabaseConfig {
	env := NewEnvConfig(prefix)
	return DatabaseConfig{
		URL:             env.GetString("URL", "http://localhost:5984"),
		Database:        env.GetString("DATABASE", ""),
		Username:        env.GetString("USERNAME", ""),
		Password:        env.GetString("PASSWORD", ""),
		MaxConnections:  env.GetInt("MAX_CONNECTIONS", 10),
		Timeout:         env.GetDuration("TIMEOUT", 30*time.Second),
		CreateIfMissing: env.GetBool("CREATE_IF_MISSING", true),
	}
}

// RegistryConfig contains registry service configuration
type RegistryConfig struct {
	URL               string
	HeartbeatInterval time.Duration
	Timeout           time.Duration
}

// LoadRegistryConfig loads registry configuration from environment
func LoadRegistryConfig(prefix string) RegistryConfig {
	env := NewEnvConfig(prefix)
	return RegistryConfig{
		URL:               env.GetString("URL", "http://localhost:8096"),
		HeartbeatInterval: env.GetDuration("HEARTBEAT_INTERVAL", 30*time.Second),
		Timeout:           env.GetDuration("TIMEOUT", 10*time.Second),
	}
}

// ServiceConfig contains common service configuration
type ServiceConfig struct {
	Name        string
	Version     string
	Environment string
	LogLevel    string
	LogFormat   string
}

// LoadServiceConfig loads service configuration from environment
func LoadServiceConfig(prefix string) ServiceConfig {
	env := NewEnvConfig(prefix)
	return ServiceConfig{
		Name:        env.GetString("NAME", ""),
		Version:     env.GetString("VERSION", "0.0.1"),
		Environment: env.GetString("ENVIRONMENT", "development"),
		LogLevel:    env.GetString("LOG_LEVEL", "info"),
		LogFormat:   env.GetString("LOG_FORMAT", "text"),
	}
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	APIKey        string
	JWTSecret     string
	JWTExpiry     time.Duration
	SessionExpiry time.Duration
}

// LoadAuthConfig loads authentication configuration from environment
func LoadAuthConfig(prefix string) AuthConfig {
	env := NewEnvConfig(prefix)
	return AuthConfig{
		APIKey:        env.GetString("API_KEY", ""),
		JWTSecret:     env.GetString("JWT_SECRET", ""),
		JWTExpiry:     env.GetDuration("JWT_EXPIRY", 24*time.Hour),
		SessionExpiry: env.GetDuration("SESSION_EXPIRY", 7*24*time.Hour),
	}
}

// CORSConfig contains CORS configuration
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxAge         time.Duration
}

// LoadCORSConfig loads CORS configuration from environment
func LoadCORSConfig(prefix string) CORSConfig {
	env := NewEnvConfig(prefix)
	return CORSConfig{
		AllowedOrigins: env.GetStringSlice("ALLOWED_ORIGINS", []string{"*"}),
		AllowedMethods: env.GetStringSlice("ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		AllowedHeaders: env.GetStringSlice("ALLOWED_HEADERS", []string{"Content-Type", "Authorization", "X-API-Key"}),
		MaxAge:         env.GetDuration("MAX_AGE", 12*time.Hour),
	}
}

// Validator provides configuration validation utilities
type Validator struct {
	errors []string
}

// NewValidator creates a new configuration validator
func NewValidator() *Validator {
	return &Validator{
		errors: make([]string, 0),
	}
}

// RequireString validates that a string field is not empty
func (v *Validator) RequireString(field, value string) {
	if value == "" {
		v.errors = append(v.errors, fmt.Sprintf("%s is required", field))
	}
}

// RequireInt validates that an integer field is within range
func (v *Validator) RequireInt(field string, value, min, max int) {
	if value < min || value > max {
		v.errors = append(v.errors, fmt.Sprintf("%s must be between %d and %d", field, min, max))
	}
}

// RequirePositiveInt validates that an integer field is positive
func (v *Validator) RequirePositiveInt(field string, value int) {
	if value <= 0 {
		v.errors = append(v.errors, fmt.Sprintf("%s must be positive", field))
	}
}

// RequireURL validates that a string is a valid URL
func (v *Validator) RequireURL(field, value string) {
	if value == "" {
		v.errors = append(v.errors, fmt.Sprintf("%s is required", field))
		return
	}
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		v.errors = append(v.errors, fmt.Sprintf("%s must be a valid URL (http:// or https://)", field))
	}
}

// RequireOneOf validates that a value is one of the allowed options
func (v *Validator) RequireOneOf(field, value string, allowed []string) {
	if value == "" {
		v.errors = append(v.errors, fmt.Sprintf("%s is required", field))
		return
	}
	for _, option := range allowed {
		if value == option {
			return
		}
	}
	v.errors = append(v.errors, fmt.Sprintf("%s must be one of: %s", field, strings.Join(allowed, ", ")))
}

// IsValid returns true if there are no validation errors
func (v *Validator) IsValid() bool {
	return len(v.errors) == 0
}

// Errors returns all validation errors
func (v *Validator) Errors() []string {
	return v.errors
}

// ErrorString returns all validation errors as a single string
func (v *Validator) ErrorString() string {
	if len(v.errors) == 0 {
		return ""
	}
	return strings.Join(v.errors, "; ")
}

// Validate runs validation and returns error if invalid
func (v *Validator) Validate() error {
	if !v.IsValid() {
		return fmt.Errorf("configuration validation failed: %s", v.ErrorString())
	}
	return nil
}

// ConfigLoader provides a fluent interface for loading configuration
type ConfigLoader struct {
	prefix string
	env    *EnvConfig
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader(prefix string) *ConfigLoader {
	return &ConfigLoader{
		prefix: prefix,
		env:    NewEnvConfig(prefix),
	}
}

// LoadAll loads all common configurations
func (cl *ConfigLoader) LoadAll() (*AllConfig, error) {
	config := &AllConfig{
		Server:   LoadServerConfig(cl.prefix),
		Database: LoadDatabaseConfig(cl.prefix + "_DB"),
		Registry: LoadRegistryConfig(cl.prefix + "_REGISTRY"),
		Service:  LoadServiceConfig(cl.prefix),
		Auth:     LoadAuthConfig(cl.prefix + "_AUTH"),
		CORS:     LoadCORSConfig(cl.prefix + "_CORS"),
	}

	// Validate configuration
	if err := cl.validate(config); err != nil {
		return nil, err
	}

	return config, nil
}

// validate validates the loaded configuration
func (cl *ConfigLoader) validate(config *AllConfig) error {
	validator := NewValidator()

	// Validate service config
	validator.RequireString("Service.Name", config.Service.Name)
	validator.RequireOneOf("Service.Environment", config.Service.Environment,
		[]string{"development", "staging", "production"})
	validator.RequireOneOf("Service.LogLevel", config.Service.LogLevel,
		[]string{"debug", "info", "warn", "error"})

	// Validate server config
	validator.RequirePositiveInt("Server.Port", config.Server.Port)

	return validator.Validate()
}

// AllConfig contains all common configurations
type AllConfig struct {
	Server   ServerConfig
	Database DatabaseConfig
	Registry RegistryConfig
	Service  ServiceConfig
	Auth     AuthConfig
	CORS     CORSConfig
}
