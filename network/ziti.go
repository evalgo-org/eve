// Package network provides utilities for interacting with OpenZiti networks.
// It includes functions for authenticating with Ziti controllers, managing services,
// creating policies, and handling configurations.
//
// Features:
//   - Ziti authentication and session management
//   - Service creation and configuration
//   - Service policy management
//   - Edge router policy management
//   - Configuration type querying
//   - Identity management
//   - HTTP client creation with Ziti context
//
// All functions use the Ziti REST API and require appropriate authentication tokens.
// The package handles JSON serialization/deserialization for Ziti API interactions.
package network

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	eve "eve.evalgo.org/common"
	sdk_golang "github.com/openziti/sdk-golang"
	"github.com/openziti/sdk-golang/ziti"
)

// Global cache for Ziti contexts to prevent multiple contexts from same identity
var (
	zitiContextCache = make(map[string]ziti.Context)
	zitiCacheMutex   sync.RWMutex
)

// ZitiServiceConfig represents a Ziti service configuration.
// Used to store information about Ziti services.
type ZitiServiceConfig struct {
	ID   string `json:"id"`   // The service configuration ID
	Name string `json:"name"` // The service configuration name
}

// ZitiServiceConfigsResult represents a collection of Ziti service configurations.
// Used to store API responses containing multiple service configurations.
type ZitiServiceConfigsResult struct {
	Data []ZitiServiceConfig `json:"data"` // Array of service configurations
}

// ZitiToken represents a Ziti authentication token.
// Used to store authentication tokens returned by the Ziti controller.
type ZitiToken struct {
	Token string `json:"token"` // The authentication token
	ID    string `json:"id"`    // The token ID
}

// ZitiResult represents a generic Ziti API response.
// Used to store the data portion of Ziti API responses.
type ZitiResult struct {
	Data ZitiToken `json:"data"` // The response data
}

// ZitiClient creates a new HTTP client that routes traffic through the Ziti network.
// This client can be used to make HTTP requests that will be tunneled through the Ziti overlay network.
//
// Parameters:
//   - id: The Ziti identity configuration file path
//
// Returns:
//   - *http.Client: An HTTP client configured with the Ziti context
//
// The function:
//  1. Creates a Ziti configuration from the specified file
//  2. Creates a Ziti context from the configuration
//  3. Returns an HTTP client that uses the Ziti context
func ZitiClient(id string) *http.Client {
	cfg, _ := ziti.NewConfigFromFile(id)
	ctx, _ := ziti.NewContext(cfg)
	return sdk_golang.NewHttpClient(ctx, nil)
}

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
	// Check if we already have a context for this identity file
	zitiCacheMutex.RLock()
	zitiContext, exists := zitiContextCache[identityFile]
	zitiCacheMutex.RUnlock()

	if !exists {
		// Need to create a new context - acquire write lock
		zitiCacheMutex.Lock()

		// Double-check in case another goroutine created it while we waited for the lock
		zitiContext, exists = zitiContextCache[identityFile]
		if !exists {
			eve.Logger.Info(fmt.Sprintf("Creating new Ziti context for identity: %s", identityFile))

			// Load and parse Ziti identity configuration
			cfg, err := ziti.NewConfigFromFile(identityFile)
			if err != nil {
				zitiCacheMutex.Unlock()
				return nil, err
			}

			// Create Ziti network context for operations
			zitiContext, err = ziti.NewContext(cfg)
			if err != nil {
				zitiCacheMutex.Unlock()
				return nil, err
			}

			// Wait for Ziti context to authenticate and sync services
			eve.Logger.Info("Waiting 5 seconds for Ziti authentication...")
			time.Sleep(5 * time.Second)

			// Verify service availability (this triggers service sync like in working test)
			eve.Logger.Info("Verifying service availability...")
			services, err := zitiContext.GetServices()
			if err != nil {
				zitiCacheMutex.Unlock()
				return nil, fmt.Errorf("failed to get services: %w", err)
			}

			eve.Logger.Info(fmt.Sprintf("✓ Context created and authenticated (found %d total services)", len(services)))

			// Cache the context for reuse
			zitiContextCache[identityFile] = zitiContext
		}

		zitiCacheMutex.Unlock()
	} else {
		eve.Logger.Info(fmt.Sprintf("Reusing existing Ziti context for identity: %s", identityFile))
	}

	// Check if the requested service exists
	services, err := zitiContext.GetServices()
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}

	found := false
	for _, svc := range services {
		if *svc.Name == serviceName {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("service '%s' not found in available services (found %d total services)", serviceName, len(services))
	}
	eve.Logger.Info(fmt.Sprintf("✓ Service '%s' verified in context", serviceName))

	// Configure HTTP transport with Ziti network dialer
	zitiTransport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Route all connections through Ziti service
			eve.Logger.Info(fmt.Sprintf("Dialing Ziti service: %s (requested addr: %s)", serviceName, addr))
			conn, err := zitiContext.Dial(serviceName)
			if err != nil {
				eve.Logger.Error(fmt.Sprintf("Failed to dial Ziti service '%s': %v", serviceName, err))
			}
			return conn, err
		},
	}

	return zitiTransport, nil
}

// postWithAuthMap makes an authenticated POST request to a Ziti API endpoint.
// This is a helper function for making authenticated POST requests with a map payload.
//
// Parameters:
//   - url: The complete URL to post to
//   - token: The Ziti session token for authentication
//   - payload: The request payload as a map
//
// Returns:
//   - string: The ID from the response data
//   - error: If the request fails or returns an error status
//
// The function:
//  1. Marshals the payload to JSON
//  2. Creates a new POST request with the JSON payload
//  3. Sets appropriate headers including the Ziti session token
//  4. Makes the request with a custom HTTP client that skips TLS verification
//  5. Checks the response status code
//  6. Decodes the response and returns the data ID
func postWithAuthMap(url, token string, payload map[string]interface{}) (string, error) {
	eve.Logger.Info(url, " <> ", token)
	data, _ := json.Marshal(payload)
	eve.Logger.Info(string(data))

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Zt-Session", token)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	eve.Logger.Info(resp.StatusCode)
	if resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read error response body: %w", err)
		}
		eve.Logger.Info(string(body))
		return "", errors.New(resp.Status)
	}

	result := ZitiResult{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return result.Data.ID, nil
}

// ZitiAuthenticate authenticates with a Ziti controller and obtains a session token.
// This function is the first step in interacting with a Ziti network.
//
// Parameters:
//   - url: The base URL of the Ziti controller
//   - user: The username for authentication
//   - pass: The password for authentication
//
// Returns:
//   - string: The Ziti session token
//   - error: If authentication fails
//
// The function:
//  1. Creates a JSON payload with the credentials
//  2. Makes a POST request to the authenticate endpoint
//  3. Decodes the response to extract the session token
func ZitiAuthenticate(url, user, pass string) (string, error) {
	payload := map[string]string{
		"username": user,
		"password": pass,
	}
	data, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url+"/authenticate?method=password", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result ZitiResult
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return result.Data.Token, nil
}

// ZitiCreateService creates a new Ziti service with the specified configuration.
// Services define what network resources are available through the Ziti network.
//
// Parameters:
//   - url: The base URL of the Ziti controller
//   - token: The Ziti session token
//   - name: The name of the service to create
//   - hostV1: The host.v1 configuration ID
//   - interceptV1: The intercept.v1 configuration ID
//
// Returns:
//   - string: The ID of the created service
//   - error: If service creation fails
//
// The function:
//  1. Creates a payload with the service configuration
//  2. Makes a POST request to the services endpoint
//  3. Returns the ID of the created service
func ZitiCreateService(url, token, name, hostV1, interceptV1 string) (string, error) {
	body := map[string]interface{}{
		"configs":            []string{hostV1, interceptV1},
		"encryptionRequired": true,
		"name":               name,
	}
	return postWithAuthMap(url+"/edge/management/v1/services", token, body)
}

// ZitiCreateServicePolicy creates a new Ziti service policy.
// Service policies define which identities can access which services.
//
// Parameters:
//   - url: The base URL of the Ziti controller
//   - token: The Ziti session token
//   - name: The name of the policy to create
//   - policyType: The type of policy (Dial or Bind)
//   - serviceID: The service role ID
//   - identity: The identity role ID
//
// Returns:
//   - string: The ID of the created policy
//   - error: If policy creation fails
//
// The function:
//  1. Creates a payload with the policy configuration
//  2. Makes a POST request to the service-policies endpoint
//  3. Returns the ID of the created policy
func ZitiCreateServicePolicy(url, token, name, policyType, serviceID, identity string) (string, error) {
	body := map[string]interface{}{
		"name":          name,
		"type":          policyType,
		"identityRoles": []string{identity},
		"serviceRoles":  []string{serviceID},
		"semantic":      "AnyOf",
	}
	pID, err := postWithAuthMap(url+"/edge/management/v1/service-policies", token, body)
	return pID, err
}

// ZitiCreateServiceConfig creates a new Ziti service configuration.
// Service configurations define how services behave in the Ziti network.
//
// Parameters:
//   - url: The base URL of the Ziti controller
//   - token: The Ziti session token
//   - name: The name of the configuration to create
//   - configType: The type of configuration (e.g., "host.v1", "intercept.v1")
//   - config: The configuration data as a map
//
// Returns:
//   - string: The ID of the created configuration
//   - error: If configuration creation fails
//
// The function:
//  1. Creates a payload with the configuration data
//  2. Makes a POST request to the configs endpoint
//  3. Returns the ID of the created configuration
func ZitiCreateServiceConfig(url, token, name, configType string, config map[string]interface{}) (string, error) {
	payload := map[string]interface{}{
		"name":         name,
		"configTypeId": configType,
		"data":         config,
	}
	eve.Logger.Info(payload)
	confID, err := postWithAuthMap(url+"/edge/management/v1/configs", token, payload)
	return confID, err
}

// ZitiCreateEdgeRouterPolicy creates a new Ziti edge router policy.
// Edge router policies define which edge routers can access which services.
//
// Parameters:
//   - url: The base URL of the Ziti controller
//   - token: The Ziti session token
//   - name: The name of the policy to create
//   - routers: Array of edge router role IDs
//   - services: Array of service role IDs
//   - roles: Array of permission roles
//
// Returns:
//   - error: If policy creation fails
//
// The function:
//  1. Creates a payload with the policy configuration
//  2. Makes a POST request to the service-edge-router-policies endpoint
func ZitiCreateEdgeRouterPolicy(url, token, name string, routers, services, roles []string) error {
	body := map[string]interface{}{
		"name":            name,
		"edgeRouterRoles": routers,
		"serviceRoles":    services,
		"semantic":        "AllOf",
		"permissions":     roles,
	}
	_, err := postWithAuthMap(url+"/edge/v1/service-edge-router-policies", token, body)
	return err
}

// ZitiGetConfigTypes retrieves the ID of a specific Ziti configuration type.
// This function queries all available configuration types and finds the one with the specified name.
//
// Parameters:
//   - url: The base URL of the Ziti controller
//   - token: The Ziti session token
//   - name: The name of the configuration type to find
//
// Returns:
//   - string: The ID of the found configuration type
//   - error: If the configuration type is not found or if the request fails
func ZitiGetConfigTypes(url, token, name string) (string, error) {
	req, _ := http.NewRequest("GET", url+"/edge/management/v1/config-types", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Zt-Session", token)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result ZitiServiceConfigsResult
	_ = json.NewDecoder(resp.Body).Decode(&result)

	for _, conf := range result.Data {
		if conf.Name == name {
			return conf.ID, nil
		}
	}

	return "", errors.New("could not find config: " + name)
}

// ZitiServicePolicies retrieves all Ziti service policies.
// This function queries the service policies endpoint and logs the raw response.
//
// Parameters:
//   - url: The base URL of the Ziti controller
//   - token: The Ziti session token
//
// The function:
//  1. Makes a GET request to the service-policies endpoint
//  2. Logs the raw response body
func ZitiServicePolicies(url, token string) error {
	req, err := http.NewRequest("GET", url+"/edge/management/v1/service-policies", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Zt-Session", token)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	eve.Logger.Info(string(body))
	return nil
}

// ZitiIdentities retrieves all Ziti identities and logs them.
// This function queries the identities endpoint and logs all found identities.
//
// Parameters:
//   - urlSrc: The base URL of the Ziti controller
//   - token: The Ziti session token
//
// The function:
//  1. Makes a GET request to the identities endpoint with a large limit
//  2. Decodes the response and logs each identity's ID and name
func ZitiIdentities(urlSrc, token string) error {
	q := url.Values{}
	q.Add("limit", "10000")
	tgtURL := urlSrc + "/edge/management/v1/identities"

	parsedURL, err := url.Parse(tgtURL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}
	parsedURL.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", parsedURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Zt-Session", token)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	var result ZitiServiceConfigsResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	for _, conf := range result.Data {
		eve.Logger.Info(conf.ID, " <> ", conf.Name)
	}
	return nil
}

// ZitiGetIdentity finds a Ziti identity by name and returns its ID.
// This function queries all identities and searches for one with the specified name.
//
// Parameters:
//   - urlSrc: The base URL of the Ziti controller
//   - token: The Ziti session token
//   - name: The name of the identity to find
//
// Returns:
//   - string: The ID of the found identity
//   - error: If the identity is not found or if the request fails
func ZitiGetIdentity(urlSrc, token, name string) (string, error) {
	q := url.Values{}
	q.Add("limit", "10000")
	tgtURL := urlSrc + "/edge/management/v1/identities"

	parsedURL, err := url.Parse(tgtURL)
	if err != nil {
		return "", err
	}
	parsedURL.RawQuery = q.Encode()

	req, _ := http.NewRequest("GET", parsedURL.String(), nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Zt-Session", token)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result ZitiServiceConfigsResult
	_ = json.NewDecoder(resp.Body).Decode(&result)

	for _, ident := range result.Data {
		if ident.Name == name {
			return ident.ID, nil
		}
	}

	return "", errors.New("could not find identity with the name: " + name)
}
