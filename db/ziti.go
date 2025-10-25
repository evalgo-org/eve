// Package db provides zero-trust networking capabilities through OpenZiti integration.
// This package enables secure, encrypted communication over the Ziti network fabric
// without requiring traditional VPNs or complex firewall configurations.
//
// OpenZiti is a zero-trust networking platform that provides:
//   - End-to-end encryption for all communications
//   - Identity-based access controls
//   - Network invisibility and micro-segmentation
//   - Application-embedded connectivity
//   - Dynamic policy enforcement
//
// The package simplifies Ziti integration by providing HTTP transport creation
// that automatically routes traffic through the Ziti network overlay, enabling
// secure database connections and service-to-service communication.
//
// Zero-Trust Principles:
//   - Never trust, always verify identity
//   - Least privilege access enforcement
//   - Assume network compromise and encrypt everything
//   - Continuous verification and monitoring
//   - Application-level access controls
//
// Use Cases:
//   - Secure database connections without exposing ports
//   - Service mesh communication with strong identity
//   - Remote access without traditional VPN complexity
//   - Multi-cloud and hybrid cloud connectivity
//   - Compliance and regulatory requirements for data protection
//
// Security Benefits:
//   - Network invisibility (dark network, no open ports)
//   - Strong cryptographic identity for all connections
//   - Automatic encryption for all data in transit
//   - Granular access policies per service/user
//   - Real-time policy enforcement and revocation
package db

import (
	"context"
	"github.com/openziti/sdk-golang/ziti"
	"net"
	"net/http"
)

// Package-level variables for Ziti configuration and context management.
// These variables maintain the global state for Ziti connectivity and are
// initialized through the ZitiSetup function for reuse across the application.
//
// Variable Descriptions:
//   - identityFile: Path to the Ziti identity file for authentication
//   - cfg: Parsed Ziti configuration from the identity file
//   - zitiContext: Active Ziti context for network operations
//   - err: Global error state for initialization tracking
//
// Initialization Pattern:
//
//	These variables follow a lazy initialization pattern where they remain
//	nil/empty until ZitiSetup is called with appropriate parameters.
//
// Thread Safety Considerations:
//
//	The current implementation uses package-level variables which may not
//	be safe for concurrent initialization. Consider using sync.Once or
//	similar synchronization primitives for production use.
var (
	identityFile string       = ""  // Path to Ziti identity file
	cfg          *ziti.Config = nil // Parsed Ziti configuration
	zitiContext  ziti.Context = nil // Active Ziti network context
	err          error        = nil // Initialization error state
)

// ZitiSetup initializes a Ziti network connection and returns an HTTP transport
// configured to route traffic through the Ziti overlay network. This function
// establishes the zero-trust networking foundation for secure service communication.
//
// Initialization Process:
//  1. Loads and parses the Ziti identity file containing cryptographic credentials
//  2. Creates a Ziti context for network operations and policy enforcement
//  3. Constructs an HTTP transport with custom dialer for Ziti routing
//  4. Returns the transport ready for use with HTTP clients
//
// Identity File Requirements:
//
//	The identity file must be a valid Ziti identity containing:
//	- Cryptographic certificates for authentication
//	- Network configuration and controller information
//	- Service access policies and permissions
//	- Enrollment and authentication tokens
//
// Service Name Resolution:
//
//	The serviceName parameter specifies the Ziti service to connect to.
//	This service must be:
//	- Defined in the Ziti network configuration
//	- Accessible according to current identity policies
//	- Running and available on the Ziti overlay network
//
// Parameters:
//   - identityFile: Filesystem path to the Ziti identity file (JSON format)
//   - serviceName: Name of the Ziti service to connect to
//
// Returns:
//   - *http.Transport: HTTP transport configured for Ziti network routing
//   - error: Configuration parsing, context creation, or validation errors
//
// Error Conditions:
//   - Identity file not found, corrupted, or invalid format
//   - Network connectivity issues to Ziti controllers
//   - Authentication failures with provided identity
//   - Service not found or access denied by policies
//   - Invalid service name or configuration parameters
//
// HTTP Transport Configuration:
//
//	The returned transport replaces the standard TCP dialer with a Ziti-aware
//	dialer that:
//	- Establishes connections through the Ziti overlay network
//	- Enforces identity-based access policies automatically
//	- Provides end-to-end encryption for all communications
//	- Handles service discovery and routing transparently
//
// Usage with HTTP Clients:
//
//	The transport can be used with any standard HTTP client:
//
//	transport, err := ZitiSetup("/path/to/identity.json", "database-service")
//	if err != nil {
//	    log.Fatal("Ziti setup failed:", err)
//	}
//
//	client := &http.Client{Transport: transport}
//	resp, err := client.Get("http://database-service/api/v1/data")
//
// Network Invisibility:
//
//	Services accessed through Ziti are not visible on traditional networks.
//	They exist only within the Ziti overlay, providing "dark" networking
//	where services cannot be discovered or accessed without proper identity.
//
// Policy Enforcement:
//
//	All connections are subject to real-time policy evaluation:
//	- Identity verification for every connection attempt
//	- Service access authorization based on current policies
//	- Dynamic policy updates without service interruption
//	- Automatic connection termination on policy revocation
//
// Performance Considerations:
//   - Initial connection establishment may have higher latency
//   - Subsequent connections benefit from connection pooling
//   - Encryption overhead is minimal with modern hardware
//   - Network routing may add latency depending on overlay topology
//
// Security Features:
//   - Mutual TLS authentication for all connections
//   - Certificate-based identity verification
//   - Automatic key rotation and certificate management
//   - Protection against man-in-the-middle attacks
//   - Network traffic analysis resistance
//
// Production Deployment:
//   - Store identity files securely (encrypted storage, secret management)
//   - Implement proper certificate lifecycle management
//   - Monitor connection health and policy compliance
//   - Plan for identity rotation and revocation scenarios
//   - Configure appropriate timeouts and retry logic
//
// Example Integration:
//
//	// Database connection through Ziti
//	transport, err := ZitiSetup("/etc/ziti/db-client.json", "postgres-db")
//	if err != nil {
//	    return fmt.Errorf("failed to setup Ziti: %w", err)
//	}
//
//	// Use with database HTTP API
//	client := &http.Client{
//	    Transport: transport,
//	    Timeout:   30 * time.Second,
//	}
//
//	// All requests now go through Ziti zero-trust network
//	resp, err := client.Post("http://postgres-db/query", "application/json", queryBody)
//
// Troubleshooting:
//
//	Common issues and solutions:
//	- Identity file errors: Verify file format and certificate validity
//	- Service not found: Check service configuration and network policies
//	- Connection failures: Verify Ziti controller connectivity
//	- Permission denied: Review identity access policies and service permissions
func ZitiSetup(identityFile, serviceName string) (*http.Transport, error) {
	// Load and parse Ziti identity configuration
	cfg, err = ziti.NewConfigFromFile(identityFile)
	if err != nil {
		return nil, err
	}

	// Create Ziti network context for operations
	zitiContext, err = ziti.NewContext(cfg)
	if err != nil {
		return nil, err
	}

	// Configure HTTP transport with Ziti network dialer
	zitiTransport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Route all connections through Ziti service
			return zitiContext.Dial(serviceName)
		},
	}

	return zitiTransport, nil
}
