package transport

import (
	"context"
	"net/http"
)

// Transport represents a network transport mechanism for HTTP requests.
// This abstraction allows WHEN to support multiple transport layers:
// - Direct HTTP/HTTPS
// - HTTP over SSH tunnels
// - HTTP over OpenZiti overlay networks
type Transport interface {
	// RoundTrip executes a single HTTP transaction, returning the response.
	// This is compatible with http.RoundTripper interface.
	RoundTrip(*http.Request) (*http.Response, error)

	// Close closes any underlying connections and cleans up resources.
	Close() error
}

// Config holds configuration for transport creation
type Config struct {
	// SSH configuration (for SSH transport)
	SSHUser       string
	SSHHost       string
	SSHPort       int
	SSHKeyFile    string
	SSHPassword   string
	SSHKnownHosts string

	// Ziti configuration (for Ziti transport)
	ZitiIdentityFile string
	ZitiIdentityJSON string

	// HTTP configuration (for all transports)
	Timeout             int // seconds
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	IdleConnTimeout     int // seconds
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		SSHPort:             22,
		Timeout:             30,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90,
	}
}

// TransportType identifies the type of transport
type TransportType string

const (
	TransportHTTP TransportType = "http"
	TransportSSH  TransportType = "ssh"
	TransportZiti TransportType = "ziti"
)

// URLScheme maps URL schemes to transport types
var URLScheme = map[string]TransportType{
	"http":      TransportHTTP,
	"https":     TransportHTTP,
	"ssh":       TransportSSH,
	"ssh+http":  TransportSSH,
	"ssh+https": TransportSSH,
	"ziti":      TransportZiti,
	"ziti+http": TransportZiti,
}

// Factory creates a Transport based on the configuration and type
type Factory interface {
	CreateTransport(ctx context.Context, transportType TransportType, config *Config) (Transport, error)
}
