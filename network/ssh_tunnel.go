// Package network provides utilities for secure network operations
package network

import (
	"fmt"
	"io"
	"net"

	"golang.org/x/crypto/ssh"
)

// SSHTunnel represents an SSH tunnel for port forwarding.
// It maintains the SSH connection and provides methods to create forwarded connections.
type SSHTunnel struct {
	client   *ssh.Client
	localNet string
}

// NewSSHTunnel creates a new SSH tunnel to a remote host.
// This establishes an SSH connection that can be used for port forwarding.
//
// Parameters:
//   - address: The remote SSH server address in "host:port" format
//   - username: The username for SSH authentication
//   - keyfile: Path to the private key file
//   - certfile: Path to the certificate file (optional, can be empty)
//
// Returns:
//   - *SSHTunnel: The established tunnel
//   - error: If connection fails
//
// Example:
//
//	tunnel, err := NewSSHTunnel("192.168.1.100:22", "user", "/home/user/.ssh/id_rsa", "")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer tunnel.Close()
func NewSSHTunnel(address string, username string, keyfile string, certfile string) (*SSHTunnel, error) {
	// Create signer from key files
	signer, err := ssh_keyfile(keyfile, certfile)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	// Configure SSH client
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to the remote host
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SSH: %w", err)
	}

	return &SSHTunnel{
		client:   client,
		localNet: "unix",
	}, nil
}

// Dial creates a connection through the SSH tunnel to a remote address.
// This is typically used to connect to services on the remote host.
//
// Parameters:
//   - network: The network type ("tcp", "unix", etc.)
//   - address: The address to connect to on the remote host
//
// Returns:
//   - net.Conn: The tunneled connection
//   - error: If the connection fails
//
// Example:
//
//	conn, err := tunnel.Dial("unix", "/var/run/docker.sock")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer conn.Close()
func (t *SSHTunnel) Dial(network, address string) (net.Conn, error) {
	conn, err := t.client.Dial(network, address)
	if err != nil {
		return nil, fmt.Errorf("failed to dial through tunnel: %w", err)
	}
	return conn, nil
}

// Close closes the SSH tunnel and all associated connections.
func (t *SSHTunnel) Close() error {
	if t.client != nil {
		return t.client.Close()
	}
	return nil
}

// ForwardConnection forwards data between a local connection and a remote address through the SSH tunnel.
// This is useful for setting up bidirectional port forwarding.
//
// Parameters:
//   - local: The local connection
//   - remoteNetwork: The network type for the remote connection ("tcp", "unix", etc.)
//   - remoteAddress: The address on the remote host
//
// Returns:
//   - error: If forwarding fails
func (t *SSHTunnel) ForwardConnection(local net.Conn, remoteNetwork, remoteAddress string) error {
	// Connect to remote through SSH tunnel
	remote, err := t.client.Dial(remoteNetwork, remoteAddress)
	if err != nil {
		return fmt.Errorf("failed to dial remote: %w", err)
	}

	// Forward data bidirectionally
	go func() {
		_, _ = io.Copy(remote, local)
		remote.Close()
	}()
	go func() {
		_, _ = io.Copy(local, remote)
		local.Close()
	}()

	return nil
}
