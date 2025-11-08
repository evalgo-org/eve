package transport

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// SSHTunnelTransport implements Transport for HTTP over SSH tunnels.
// This creates SSH connections to remote hosts and tunnels HTTP traffic through them.
type SSHTunnelTransport struct {
	config     *Config
	sshClient  *ssh.Client
	transport  *http.Transport
	client     *http.Client
	mu         sync.Mutex
	tunnelPool map[string]net.Listener
}

// NewSSHTunnelTransport creates a new SSH tunnel transport
func NewSSHTunnelTransport(ctx context.Context, config *Config) (*SSHTunnelTransport, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Validate required SSH configuration
	if config.SSHHost == "" {
		return nil, fmt.Errorf("SSHHost is required for SSH transport")
	}
	if config.SSHUser == "" {
		return nil, fmt.Errorf("SSHUser is required for SSH transport")
	}

	// Build SSH client config
	sshConfig, err := buildSSHConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build SSH config: %w", err)
	}

	// Connect to SSH server
	sshAddr := fmt.Sprintf("%s:%d", config.SSHHost, config.SSHPort)
	sshClient, err := ssh.Dial("tcp", sshAddr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH server %s: %w", sshAddr, err)
	}

	// Create custom dialer that uses SSH tunnel
	t := &SSHTunnelTransport{
		config:     config,
		sshClient:  sshClient,
		tunnelPool: make(map[string]net.Listener),
	}

	// Create http.Transport with custom dialer
	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxIdleConnsPerHost,
		IdleConnTimeout:     time.Duration(config.IdleConnTimeout) * time.Second,
		DialContext:         t.dialContext,
	}

	// Create http.Client
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(config.Timeout) * time.Second,
	}

	t.transport = transport
	t.client = client

	return t, nil
}

// buildSSHConfig creates an SSH client configuration
func buildSSHConfig(config *Config) (*ssh.ClientConfig, error) {
	var authMethods []ssh.AuthMethod

	// Add key-based authentication
	if config.SSHKeyFile != "" {
		key, err := os.ReadFile(config.SSHKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read SSH key file: %w", err)
		}

		var signer ssh.Signer
		if config.SSHPassword != "" {
			// Encrypted key
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(config.SSHPassword))
		} else {
			// Unencrypted key
			signer, err = ssh.ParsePrivateKey(key)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse SSH key: %w", err)
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// Add password authentication
	if config.SSHPassword != "" && config.SSHKeyFile == "" {
		authMethods = append(authMethods, ssh.Password(config.SSHPassword))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no SSH authentication method configured (need SSHKeyFile or SSHPassword)")
	}

	// Configure host key verification
	var hostKeyCallback ssh.HostKeyCallback
	if config.SSHKnownHosts != "" {
		var err error
		hostKeyCallback, err = knownhosts.New(config.SSHKnownHosts)
		if err != nil {
			return nil, fmt.Errorf("failed to load known_hosts file: %w", err)
		}
	} else {
		// INSECURE: Accept any host key
		// In production, always use known_hosts file
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	return &ssh.ClientConfig{
		User:            config.SSHUser,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         time.Duration(config.Timeout) * time.Second,
	}, nil
}

// dialContext creates a connection through the SSH tunnel
func (t *SSHTunnelTransport) dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	// Dial through SSH tunnel
	conn, err := t.sshClient.Dial(network, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial through SSH tunnel: %w", err)
	}

	return conn, nil
}

// RoundTrip executes an HTTP request through the SSH tunnel
func (t *SSHTunnelTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// The request URL should be rewritten to use the tunnel
	// For ssh:// or ssh+http:// schemes, strip the ssh+ prefix and use http://
	// For ssh+https://, use https://

	switch req.URL.Scheme {
	case "ssh", "ssh+http":
		req.URL.Scheme = "http"
	case "ssh+https":
		req.URL.Scheme = "https"
	}

	return t.transport.RoundTrip(req)
}

// Close closes the SSH connection and all tunnels
func (t *SSHTunnelTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Close all tunnel listeners
	for _, listener := range t.tunnelPool {
		_ = listener.Close()
	}
	t.tunnelPool = make(map[string]net.Listener)

	// Close SSH client
	if t.sshClient != nil {
		return t.sshClient.Close()
	}

	return nil
}

// Client returns the underlying http.Client for direct use
func (t *SSHTunnelTransport) Client() *http.Client {
	return t.client
}

// CreateLocalForward creates a local port forward through the SSH tunnel
// This can be used for long-lived tunnels to specific services
func (t *SSHTunnelTransport) CreateLocalForward(localAddr, remoteAddr string) (net.Listener, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Check if tunnel already exists
	if existing, ok := t.tunnelPool[localAddr]; ok {
		return existing, nil
	}

	// Create local listener
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create local listener: %w", err)
	}

	// Accept connections and forward through SSH
	go func() {
		for {
			localConn, err := listener.Accept()
			if err != nil {
				return // Listener closed
			}

			go t.handleLocalForward(localConn, remoteAddr)
		}
	}()

	t.tunnelPool[localAddr] = listener
	return listener, nil
}

// handleLocalForward forwards a local connection through SSH tunnel
func (t *SSHTunnelTransport) handleLocalForward(localConn net.Conn, remoteAddr string) {
	defer func() { _ = localConn.Close() }()

	// Dial remote address through SSH
	remoteConn, err := t.sshClient.Dial("tcp", remoteAddr)
	if err != nil {
		return
	}
	defer func() { _ = remoteConn.Close() }()

	// Bidirectional copy
	done := make(chan struct{}, 2)

	go func() {
		_, _ = io.Copy(remoteConn, localConn)
		done <- struct{}{}
	}()

	go func() {
		_, _ = io.Copy(localConn, remoteConn)
		done <- struct{}{}
	}()

	// Wait for one direction to finish
	<-done
}
