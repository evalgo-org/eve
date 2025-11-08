package transport

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// HTTPTransport implements Transport for direct HTTP/HTTPS connections.
// This wraps the standard library's http.Transport with configured timeouts and connection pooling.
type HTTPTransport struct {
	transport *http.Transport
	client    *http.Client
}

// NewHTTPTransport creates a new HTTP transport with the given configuration
func NewHTTPTransport(ctx context.Context, config *Config) (*HTTPTransport, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create http.Transport with connection pooling
	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxIdleConnsPerHost,
		IdleConnTimeout:     time.Duration(config.IdleConnTimeout) * time.Second,
		// Enable HTTP/2 by default
		ForceAttemptHTTP2: true,
	}

	// Create http.Client with timeout
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(config.Timeout) * time.Second,
	}

	return &HTTPTransport{
		transport: transport,
		client:    client,
	}, nil
}

// RoundTrip executes a single HTTP transaction using the standard library.
// This implements the http.RoundTripper interface.
func (t *HTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Validate URL scheme
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return nil, fmt.Errorf("HTTPTransport only supports http:// and https:// schemes, got: %s", req.URL.Scheme)
	}

	return t.transport.RoundTrip(req)
}

// Close closes idle connections
func (t *HTTPTransport) Close() error {
	t.transport.CloseIdleConnections()
	return nil
}

// Client returns the underlying http.Client for direct use
func (t *HTTPTransport) Client() *http.Client {
	return t.client
}
