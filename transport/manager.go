package transport

import (
	"context"
	"fmt"
	"net/http"
	"sync"
)

// Manager manages multiple transports and routes requests based on URL scheme.
// This allows WHEN to support multiple network transports transparently.
type Manager struct {
	transports map[TransportType]Transport
	configs    map[TransportType]*Config
	mu         sync.RWMutex
	ctx        context.Context
}

// NewManager creates a new transport manager
func NewManager(ctx context.Context) *Manager {
	return &Manager{
		transports: make(map[TransportType]Transport),
		configs:    make(map[TransportType]*Config),
		ctx:        ctx,
	}
}

// RegisterTransport registers a transport for a specific type
func (m *Manager) RegisterTransport(transportType TransportType, transport Transport) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.transports[transportType] = transport
}

// RegisterTransportWithConfig registers and creates a transport based on configuration
func (m *Manager) RegisterTransportWithConfig(transportType TransportType, config *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store config for potential re-initialization
	m.configs[transportType] = config

	// Create transport
	var transport Transport
	var err error

	switch transportType {
	case TransportHTTP:
		transport, err = NewHTTPTransport(m.ctx, config)
	case TransportSSH:
		transport, err = NewSSHTunnelTransport(m.ctx, config)
	case TransportZiti:
		transport, err = NewZitiTransport(m.ctx, config)
	default:
		return fmt.Errorf("unknown transport type: %s", transportType)
	}

	if err != nil {
		return fmt.Errorf("failed to create %s transport: %w", transportType, err)
	}

	m.transports[transportType] = transport
	return nil
}

// GetTransport returns the transport for a given type
func (m *Manager) GetTransport(transportType TransportType) (Transport, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	transport, ok := m.transports[transportType]
	if !ok {
		return nil, fmt.Errorf("transport not registered: %s", transportType)
	}

	return transport, nil
}

// GetTransportForURL returns the appropriate transport based on URL scheme
func (m *Manager) GetTransportForURL(urlScheme string) (Transport, error) {
	// Map URL scheme to transport type
	transportType, ok := URLScheme[urlScheme]
	if !ok {
		return nil, fmt.Errorf("unsupported URL scheme: %s", urlScheme)
	}

	return m.GetTransport(transportType)
}

// RoundTrip implements http.RoundTripper interface.
// This routes the request to the appropriate transport based on URL scheme.
func (m *Manager) RoundTrip(req *http.Request) (*http.Response, error) {
	transport, err := m.GetTransportForURL(req.URL.Scheme)
	if err != nil {
		return nil, err
	}

	return transport.RoundTrip(req)
}

// Client creates an http.Client that uses this transport manager
func (m *Manager) Client(timeout int) *http.Client {
	return &http.Client{
		Transport: m,
		// Note: timeout is handled by individual transports
	}
}

// Close closes all registered transports
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for transportType, transport := range m.transports {
		if err := transport.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close %s transport: %w", transportType, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing transports: %v", errs)
	}

	return nil
}

// DefaultManager creates a manager with HTTP transport pre-configured
func DefaultManager(ctx context.Context) (*Manager, error) {
	manager := NewManager(ctx)

	// Register HTTP transport with default config
	if err := manager.RegisterTransportWithConfig(TransportHTTP, DefaultConfig()); err != nil {
		return nil, err
	}

	return manager, nil
}

// DefaultManagerWithAllTransports creates a manager with all transports configured from environment
func DefaultManagerWithAllTransports(ctx context.Context, httpConfig, sshConfig, zitiConfig *Config) (*Manager, error) {
	manager := NewManager(ctx)

	// Register HTTP transport
	if httpConfig == nil {
		httpConfig = DefaultConfig()
	}
	if err := manager.RegisterTransportWithConfig(TransportHTTP, httpConfig); err != nil {
		return nil, fmt.Errorf("failed to register HTTP transport: %w", err)
	}

	// Register SSH transport if configured
	if sshConfig != nil && sshConfig.SSHHost != "" {
		if err := manager.RegisterTransportWithConfig(TransportSSH, sshConfig); err != nil {
			return nil, fmt.Errorf("failed to register SSH transport: %w", err)
		}
	}

	// Register Ziti transport if configured
	if zitiConfig != nil && (zitiConfig.ZitiIdentityFile != "" || zitiConfig.ZitiIdentityJSON != "") {
		if err := manager.RegisterTransportWithConfig(TransportZiti, zitiConfig); err != nil {
			return nil, fmt.Errorf("failed to register Ziti transport: %w", err)
		}
	}

	return manager, nil
}

// SupportedSchemes returns a list of supported URL schemes
func (m *Manager) SupportedSchemes() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	schemes := make([]string, 0)
	seenTypes := make(map[TransportType]bool)

	for scheme, transportType := range URLScheme {
		if _, ok := m.transports[transportType]; ok {
			if !seenTypes[transportType] {
				schemes = append(schemes, scheme)
				seenTypes[transportType] = true
			}
		}
	}

	return schemes
}

// GetConfig returns the configuration for a transport type
func (m *Manager) GetConfig(transportType TransportType) (*Config, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config, ok := m.configs[transportType]
	return config, ok
}
