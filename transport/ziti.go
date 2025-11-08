package transport

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/openziti/sdk-golang/ziti"
)

// ZitiTransport implements Transport for HTTP over OpenZiti overlay networks.
// This provides zero-trust networking with built-in encryption and identity-based access control.
type ZitiTransport struct {
	config    *Config
	zitiCtx   ziti.Context
	transport *http.Transport
	client    *http.Client
}

// NewZitiTransport creates a new OpenZiti transport
func NewZitiTransport(ctx context.Context, cfg *Config) (*ZitiTransport, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Validate required Ziti configuration
	if cfg.ZitiIdentityFile == "" && cfg.ZitiIdentityJSON == "" {
		return nil, fmt.Errorf("ZitiIdentityFile or ZitiIdentityJSON is required for Ziti transport")
	}

	// Load Ziti identity and create context
	var zitiCtx ziti.Context
	var err error

	if cfg.ZitiIdentityFile != "" {
		// Load from file
		zitiCtx, err = ziti.NewContextFromFile(cfg.ZitiIdentityFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load Ziti identity from file: %w", err)
		}
	} else {
		// Load from JSON string (write to temp file first)
		tmpFile, err := os.CreateTemp("", "ziti-identity-*.json")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp file for Ziti identity: %w", err)
		}
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		if _, err := tmpFile.WriteString(cfg.ZitiIdentityJSON); err != nil {
			return nil, fmt.Errorf("failed to write Ziti identity to temp file: %w", err)
		}
		_ = tmpFile.Close()

		zitiCtx, err = ziti.NewContextFromFile(tmpFile.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to load Ziti identity from JSON: %w", err)
		}
	}

	t := &ZitiTransport{
		config:  cfg,
		zitiCtx: zitiCtx,
	}

	// Create http.Transport with custom dialer that uses Ziti
	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		IdleConnTimeout:     time.Duration(cfg.IdleConnTimeout) * time.Second,
		DialContext:         t.dialContext,
	}

	// Create http.Client
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(cfg.Timeout) * time.Second,
	}

	t.transport = transport
	t.client = client

	return t, nil
}

// dialContext creates a connection through the Ziti overlay network
func (t *ZitiTransport) dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	// Parse service name from address
	// For Ziti, the "host" part of the URL is the Ziti service name
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// If no port, use the address as-is
		host = addr
	}

	// Dial the Ziti service
	conn, err := t.zitiCtx.Dial(host)
	if err != nil {
		return nil, fmt.Errorf("failed to dial Ziti service %s: %w", host, err)
	}

	return conn, nil
}

// RoundTrip executes an HTTP request through the Ziti overlay network
func (t *ZitiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite URL scheme from ziti:// or ziti+http:// to http://
	// The host part is the Ziti service name
	scheme := req.URL.Scheme
	if scheme == "ziti" || scheme == "ziti+http" {
		req.URL.Scheme = "http"
	}

	return t.transport.RoundTrip(req)
}

// Close closes the Ziti context
func (t *ZitiTransport) Close() error {
	if t.zitiCtx != nil {
		t.zitiCtx.Close()
	}
	return nil
}

// Client returns the underlying http.Client for direct use
func (t *ZitiTransport) Client() *http.Client {
	return t.client
}

// GetZitiContext returns the underlying Ziti context for direct service operations
func (t *ZitiTransport) GetZitiContext() ziti.Context {
	return t.zitiCtx
}

// ListServices returns the list of available Ziti services
func (t *ZitiTransport) ListServices() ([]string, error) {
	services, err := t.zitiCtx.GetServices()
	if err != nil {
		return nil, fmt.Errorf("failed to list Ziti services: %w", err)
	}

	names := make([]string, 0, len(services))
	for _, svc := range services {
		if svc.Name != nil {
			names = append(names, *svc.Name)
		}
	}

	return names, nil
}

// Dial creates a direct connection to a Ziti service
// This can be used for non-HTTP protocols over Ziti
func (t *ZitiTransport) Dial(serviceName string) (net.Conn, error) {
	return t.zitiCtx.Dial(serviceName)
}

// Listen creates a Ziti service listener
// This allows the application to host services on the Ziti network
func (t *ZitiTransport) Listen(serviceName string) (net.Listener, error) {
	return t.zitiCtx.Listen(serviceName)
}
