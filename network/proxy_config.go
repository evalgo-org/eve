// Package network provides utilities for OpenZiti network operations including
// a generic configuration-driven proxy server with enterprise features.
//
// The proxy package includes:
//   - JSON-based configuration for routes, backends, and policies
//   - Multiple load balancing strategies (round-robin, weighted, least-connections)
//   - Automatic health checks with failover
//   - Retry logic with exponential backoff
//   - Authentication (API key, JWT, basic auth)
//   - CORS support
//   - Request logging
//   - Hot configuration reload
//
// Example usage:
//
//	proxy, err := network.NewZitiProxy("config.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	proxy.Start()
package network

import (
	"encoding/json"
	"os"
	"time"
)

// Duration is a custom type for unmarshaling duration strings from JSON
type Duration struct {
	time.Duration
}

// UnmarshalJSON implements json.Unmarshaler for Duration
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		d.Duration = 30 * time.Second
		return nil
	}
}

// LoadBalancingStrategy defines the load balancing algorithm
type LoadBalancingStrategy string

const (
	RoundRobin         LoadBalancingStrategy = "round-robin"
	WeightedRoundRobin LoadBalancingStrategy = "weighted-round-robin"
	LeastConnections   LoadBalancingStrategy = "least-connections"
)

// BackendConfig represents a single backend service in the Ziti network
type BackendConfig struct {
	ZitiService  string   `json:"ziti_service"`  // Ziti service name
	Port         int      `json:"port"`          // Backend service port (optional, default: 80 for http)
	IdentityFile string   `json:"identity_file"` // Path to Ziti identity file
	Weight       int      `json:"weight"`        // Weight for weighted load balancing (default: 1)
	Priority     int      `json:"priority"`      // Priority for failover (higher = preferred)
	Timeout      Duration `json:"timeout"`       // Request timeout
	MaxRetries   int      `json:"max_retries"`   // Maximum retry attempts
}

// HealthCheckConfig defines health check parameters for backends
type HealthCheckConfig struct {
	Enabled        bool     `json:"enabled"`         // Enable health checks
	Interval       Duration `json:"interval"`        // Check interval (e.g., "30s")
	Timeout        Duration `json:"timeout"`         // Health check timeout
	Path           string   `json:"path"`            // Health check endpoint path
	ExpectedStatus int      `json:"expected_status"` // Expected HTTP status code
	FailureCount   int      `json:"failure_count"`   // Failed checks before marking unhealthy
	SuccessCount   int      `json:"success_count"`   // Successful checks before marking healthy
}

// RetryConfig defines retry behavior for failed requests
type RetryConfig struct {
	MaxAttempts     int      `json:"max_attempts"`     // Maximum retry attempts
	InitialInterval Duration `json:"initial_interval"` // Initial backoff interval
	MaxInterval     Duration `json:"max_interval"`     // Maximum backoff interval
	Multiplier      float64  `json:"multiplier"`       // Backoff multiplier
	RetryableStatus []int    `json:"retryable_status"` // HTTP status codes to retry
}

// CircuitBreakerConfig defines circuit breaker parameters
type CircuitBreakerConfig struct {
	Enabled          bool     `json:"enabled"`            // Enable circuit breaker
	FailureThreshold int      `json:"failure_threshold"`  // Failures before opening
	SuccessThreshold int      `json:"success_threshold"`  // Successes to close circuit
	Timeout          Duration `json:"timeout"`            // Timeout before half-open
	HalfOpenRequests int      `json:"half_open_requests"` // Test requests in half-open state
}

// ZitiConfig defines global Ziti identity configuration
type ZitiConfig struct {
	IdentityFile string `json:"identity_file"` // Path to Ziti identity file
}

// AuthConfig defines authentication requirements
type AuthConfig struct {
	Type   string         `json:"type"`   // "api-key", "jwt", "basic", "none"
	Header string         `json:"header"` // Header name for auth token
	Keys   []string       `json:"keys"`   // Valid API keys or secrets
	Bypass []string       `json:"bypass"` // Paths that bypass authentication
	JWT    *JWTAuthConfig `json:"jwt"`    // JWT-specific configuration
	Ziti   *ZitiConfig    `json:"ziti"`   // Global Ziti identity configuration
}

// JWTAuthConfig defines JWT authentication parameters
type JWTAuthConfig struct {
	Secret         string   `json:"secret"`          // JWT signing secret
	PublicKeyFile  string   `json:"public_key_file"` // Path to public key for verification
	Algorithm      string   `json:"algorithm"`       // Signing algorithm (e.g., "HS256", "RS256")
	Issuer         string   `json:"issuer"`          // Expected issuer
	Audience       []string `json:"audience"`        // Expected audience
	RequiredClaims []string `json:"required_claims"` // Claims that must be present
}

// CORSConfig defines CORS settings
type CORSConfig struct {
	Enabled          bool     `json:"enabled"`           // Enable CORS
	AllowedOrigins   []string `json:"allowed_origins"`   // Allowed origins
	AllowedMethods   []string `json:"allowed_methods"`   // Allowed HTTP methods
	AllowedHeaders   []string `json:"allowed_headers"`   // Allowed headers
	ExposedHeaders   []string `json:"exposed_headers"`   // Exposed headers
	AllowCredentials bool     `json:"allow_credentials"` // Allow credentials
	MaxAge           int      `json:"max_age"`           // Preflight cache duration
}

// LoggingConfig defines logging behavior
type LoggingConfig struct {
	Enabled      bool     `json:"enabled"`       // Enable request logging
	Level        string   `json:"level"`         // Log level (debug, info, warn, error)
	Format       string   `json:"format"`        // Log format (json, text)
	IncludeBody  bool     `json:"include_body"`  // Include request/response bodies
	ExcludePaths []string `json:"exclude_paths"` // Paths to exclude from logging
}

// RouteConfig defines a single proxy route
type RouteConfig struct {
	Path           string                `json:"path"`            // Route path pattern
	Methods        []string              `json:"methods"`         // Allowed HTTP methods
	Backends       []BackendConfig       `json:"backends"`        // Backend services
	LoadBalancing  LoadBalancingStrategy `json:"load_balancing"`  // Load balancing strategy
	HealthCheck    *HealthCheckConfig    `json:"health_check"`    // Health check configuration
	Retry          *RetryConfig          `json:"retry"`           // Retry configuration
	CircuitBreaker *CircuitBreakerConfig `json:"circuit_breaker"` // Circuit breaker config
	StripPrefix    bool                  `json:"strip_prefix"`    // Strip path prefix before forwarding
	AddPrefix      string                `json:"add_prefix"`      // Add prefix before forwarding
	RewriteHost    bool                  `json:"rewrite_host"`    // Rewrite Host header
	Timeout        Duration              `json:"timeout"`         // Route-specific timeout
	Auth           *AuthConfig           `json:"auth"`            // Route-specific auth (overrides global)
}

// ProxyConfig is the root configuration structure
type ProxyConfig struct {
	Server struct {
		Host         string   `json:"host"`          // Server bind address
		Port         int      `json:"port"`          // Server bind port
		ReadTimeout  Duration `json:"read_timeout"`  // Read timeout
		WriteTimeout Duration `json:"write_timeout"` // Write timeout
		IdleTimeout  Duration `json:"idle_timeout"`  // Idle timeout
	} `json:"server"`

	Auth    *AuthConfig    `json:"auth"`    // Global authentication config
	CORS    *CORSConfig    `json:"cors"`    // CORS configuration
	Logging *LoggingConfig `json:"logging"` // Logging configuration

	Routes []RouteConfig `json:"routes"` // Route configurations

	Defaults struct {
		Timeout        Duration              `json:"timeout"`         // Default timeout
		MaxRetries     int                   `json:"max_retries"`     // Default max retries
		LoadBalancing  LoadBalancingStrategy `json:"load_balancing"`  // Default load balancing
		HealthCheck    *HealthCheckConfig    `json:"health_check"`    // Default health check
		Retry          *RetryConfig          `json:"retry"`           // Default retry config
		CircuitBreaker *CircuitBreakerConfig `json:"circuit_breaker"` // Default circuit breaker
	} `json:"defaults"`
}

// LoadProxyConfig loads proxy configuration from a JSON file
func LoadProxyConfig(configPath string) (*ProxyConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config ProxyConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Apply defaults to routes that don't specify their own values
	for i := range config.Routes {
		route := &config.Routes[i]

		// Apply default timeout if not specified
		if route.Timeout.Duration == 0 && config.Defaults.Timeout.Duration > 0 {
			route.Timeout = config.Defaults.Timeout
		}

		// Apply default load balancing if not specified
		if route.LoadBalancing == "" {
			route.LoadBalancing = config.Defaults.LoadBalancing
		}
		if route.LoadBalancing == "" {
			route.LoadBalancing = RoundRobin // Fallback to round-robin
		}

		// Apply default health check if not specified
		if route.HealthCheck == nil && config.Defaults.HealthCheck != nil {
			route.HealthCheck = config.Defaults.HealthCheck
		}

		// Apply default retry config if not specified
		if route.Retry == nil && config.Defaults.Retry != nil {
			route.Retry = config.Defaults.Retry
		}

		// Apply default circuit breaker if not specified
		if route.CircuitBreaker == nil && config.Defaults.CircuitBreaker != nil {
			route.CircuitBreaker = config.Defaults.CircuitBreaker
		}

		// Apply defaults to backends
		for j := range route.Backends {
			backend := &route.Backends[j]

			if backend.Timeout.Duration == 0 {
				if route.Timeout.Duration > 0 {
					backend.Timeout = route.Timeout
				} else if config.Defaults.Timeout.Duration > 0 {
					backend.Timeout = config.Defaults.Timeout
				} else {
					backend.Timeout = Duration{30 * time.Second}
				}
			}

			if backend.MaxRetries == 0 {
				if config.Defaults.MaxRetries > 0 {
					backend.MaxRetries = config.Defaults.MaxRetries
				} else {
					backend.MaxRetries = 3
				}
			}

			if backend.Weight == 0 {
				backend.Weight = 1
			}

			// Apply global Ziti identity file if backend doesn't have one
			if backend.IdentityFile == "" && config.Auth != nil && config.Auth.Ziti != nil {
				backend.IdentityFile = config.Auth.Ziti.IdentityFile
			}
		}
	}

	return &config, nil
}
