// Package registry provides service discovery and registration client utilities.
// This package enables services to register with a centralized registry service,
// maintain heartbeats, and discover other services in the EVE ecosystem.
package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// Client represents a registry client for service registration and discovery
type Client struct {
	registryURL       string
	serviceIdentifier string
	httpClient        *http.Client
}

// ClientConfig contains configuration for creating a registry client
type ClientConfig struct {
	RegistryURL string // Base URL of registry service (e.g., http://localhost:8096)
	Timeout     time.Duration
}

// ServiceConfig contains configuration for service registration
type ServiceConfig struct {
	ServiceID    string                 // Unique service identifier
	ServiceName  string                 // Human-readable service name
	ServiceURL   string                 // Base URL of this service
	Version      string                 // Service version
	Hostname     string                 // Hostname (auto-detected if empty)
	ServiceType  string                 // Type of service (e.g., "graphdb", "agent")
	Capabilities []string               // List of capabilities
	Properties   map[string]interface{} // Additional properties
}

// Service represents a registered service
type Service struct {
	ID            string            `json:"identifier"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	URL           string            `json:"url"`
	Documentation string            `json:"documentation,omitempty"`
	Properties    ServiceProperties `json:"additionalProperty"`
}

// ServiceProperties contains service metadata
type ServiceProperties struct {
	Port         int      `json:"port"`
	Directory    string   `json:"directory"`
	Binary       string   `json:"binary"`
	LogFile      string   `json:"logFile"`
	HealthCheck  string   `json:"healthCheck,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// SemanticService represents a Schema.org formatted service registration
type SemanticService struct {
	Context    string                 `json:"@context"`
	Type       string                 `json:"@type"`
	Identifier string                 `json:"identifier"`
	Name       string                 `json:"name"`
	URL        string                 `json:"url"`
	Properties map[string]interface{} `json:"additionalProperty,omitempty"`
}

// NewClient creates a new registry client
func NewClient(config ClientConfig) *Client {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &Client{
		registryURL: config.RegistryURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Register registers a service with the registry using Schema.org format
func (c *Client) Register(ctx context.Context, config ServiceConfig) error {
	// Auto-detect hostname if not provided
	hostname := config.Hostname
	if hostname == "" {
		hostname = os.Getenv("HOSTNAME")
		if hostname == "" {
			var err error
			hostname, err = os.Hostname()
			if err != nil {
				hostname = "unknown"
			}
		}
	}

	// Build service identifier
	c.serviceIdentifier = config.ServiceID
	if c.serviceIdentifier == "" {
		c.serviceIdentifier = fmt.Sprintf("%s-%s", config.ServiceType, hostname)
	}

	// Build properties map
	properties := make(map[string]interface{})
	if config.Properties != nil {
		for k, v := range config.Properties {
			properties[k] = v
		}
	}

	// Add standard properties
	properties["version"] = config.Version
	properties["hostname"] = hostname
	properties["serviceType"] = config.ServiceType
	properties["capabilities"] = config.Capabilities
	properties["healthEndpoint"] = fmt.Sprintf("%s/health", config.ServiceURL)

	// Create semantic service registration
	registration := SemanticService{
		Context:    "https://schema.org",
		Type:       "SoftwareApplication",
		Identifier: c.serviceIdentifier,
		Name:       config.ServiceName,
		URL:        config.ServiceURL,
		Properties: properties,
	}

	// Marshal to JSON
	payload, err := json.Marshal(registration)
	if err != nil {
		return fmt.Errorf("failed to marshal registration: %w", err)
	}

	// Send registration
	registrationURL := fmt.Sprintf("%s/v1/api/services", c.registryURL)
	req, err := http.NewRequestWithContext(ctx, "POST", registrationURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create registration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send registration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	log.Printf("Successfully registered %s with registry at %s", c.serviceIdentifier, c.registryURL)
	return nil
}

// StartHeartbeat starts sending periodic heartbeats to the registry
// Returns a context cancel function to stop the heartbeat
func (c *Client) StartHeartbeat(ctx context.Context, interval time.Duration) context.CancelFunc {
	if interval == 0 {
		interval = 30 * time.Second
	}

	heartbeatCtx, cancel := context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := c.sendHeartbeat(heartbeatCtx); err != nil {
					log.Printf("Failed to send registry heartbeat: %v", err)
				}

			case <-heartbeatCtx.Done():
				log.Println("Registry heartbeat stopped")
				return
			}
		}
	}()

	return cancel
}

// sendHeartbeat sends a heartbeat to update service status
func (c *Client) sendHeartbeat(ctx context.Context) error {
	if c.serviceIdentifier == "" {
		return fmt.Errorf("service not registered (no identifier)")
	}

	heartbeatURL := fmt.Sprintf("%s/v1/api/services/%s/heartbeat", c.registryURL, c.serviceIdentifier)

	heartbeat := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"status":    "healthy",
	}

	payload, err := json.Marshal(heartbeat)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", heartbeatURL, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// If service not found (404), we need to re-register
	if resp.StatusCode == http.StatusNotFound {
		log.Printf("Service not found in registry, service may have been removed")
		return fmt.Errorf("service not found in registry")
	}

	return nil
}

// Deregister removes this service from the registry
func (c *Client) Deregister(ctx context.Context) error {
	if c.serviceIdentifier == "" {
		return nil // Nothing to deregister
	}

	deregisterURL := fmt.Sprintf("%s/v1/api/services/%s", c.registryURL, c.serviceIdentifier)

	req, err := http.NewRequestWithContext(ctx, "DELETE", deregisterURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("Successfully deregistered %s from registry", c.serviceIdentifier)
		return nil
	}

	return fmt.Errorf("deregister failed with status %d", resp.StatusCode)
}

// GetServiceURL queries the registry for a service URL by service ID
func (c *Client) GetServiceURL(ctx context.Context, serviceID string) (string, error) {
	serviceURL := fmt.Sprintf("%s/v1/api/services/%s", c.registryURL, serviceID)

	req, err := http.NewRequestWithContext(ctx, "GET", serviceURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("service not found: %s", serviceID)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	var svc Service
	if err := json.NewDecoder(resp.Body).Decode(&svc); err != nil {
		return "", fmt.Errorf("failed to parse service response: %w", err)
	}

	return svc.URL, nil
}

// ListServices returns all services from the registry
func (c *Client) ListServices(ctx context.Context) ([]*Service, error) {
	listURL := fmt.Sprintf("%s/v1/api/services", c.registryURL)

	req, err := http.NewRequestWithContext(ctx, "GET", listURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	var services []*Service
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return nil, fmt.Errorf("failed to parse services response: %w", err)
	}

	return services, nil
}

// GetServiceIdentifier returns the registered service identifier
func (c *Client) GetServiceIdentifier() string {
	return c.serviceIdentifier
}

// AutoRegisterConfig contains configuration for auto-registration
type AutoRegisterConfig struct {
	ServiceID    string
	ServiceName  string
	Description  string
	Port         int
	Directory    string
	Binary       string
	Capabilities []string
	RegistryURL  string // e.g., http://localhost:8096
}

// AutoRegister registers a service with the registry service if REGISTRYSERVICE_API_URL is set
// Returns true if registration was attempted, false if skipped
func AutoRegister(config AutoRegisterConfig) (bool, error) {
	// Check if registry URL is set
	registryURL := os.Getenv("REGISTRYSERVICE_API_URL")
	if registryURL == "" {
		registryURL = config.RegistryURL
	}

	if registryURL == "" {
		log.Println("Auto-registration skipped: REGISTRYSERVICE_API_URL not set")
		return false, nil
	}

	// Build service registration
	service := Service{
		ID:            config.ServiceID,
		Name:          config.ServiceName,
		Description:   config.Description,
		URL:           fmt.Sprintf("http://localhost:%d", config.Port),
		Documentation: fmt.Sprintf("http://localhost:%d/v1/api", config.Port),
		Properties: ServiceProperties{
			Port:         config.Port,
			Directory:    config.Directory,
			Binary:       config.Binary,
			LogFile:      fmt.Sprintf("/tmp/%s.log", config.ServiceID),
			HealthCheck:  fmt.Sprintf("http://localhost:%d/health", config.Port),
			Capabilities: config.Capabilities,
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(service)
	if err != nil {
		return true, fmt.Errorf("failed to marshal service: %w", err)
	}

	// Register with retry
	registerURL := fmt.Sprintf("%s/v1/api/services/register", registryURL)

	var lastErr error
	for i := 0; i < 3; i++ {
		if i > 0 {
			time.Sleep(time.Second * time.Duration(i))
		}

		req, err := http.NewRequest("POST", registerURL, bytes.NewBuffer(data))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to register: %w", err)
			log.Printf("Registration attempt %d failed: %v", i+1, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Printf("Successfully registered %s with registry at %s", config.ServiceID, registryURL)
			return true, nil
		}

		lastErr = fmt.Errorf("registration failed with status %d", resp.StatusCode)
		log.Printf("Registration attempt %d failed with status %d", i+1, resp.StatusCode)
	}

	return true, fmt.Errorf("failed to register after 3 attempts: %w", lastErr)
}

// AutoUnregister removes a service from the registry on shutdown
func AutoUnregister(serviceID string) error {
	registryURL := os.Getenv("REGISTRYSERVICE_API_URL")
	if registryURL == "" {
		return nil // Nothing to do
	}

	unregisterURL := fmt.Sprintf("%s/v1/api/services/%s", registryURL, serviceID)

	req, err := http.NewRequest("DELETE", unregisterURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create unregister request: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to unregister: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("Successfully unregistered %s from registry", serviceID)
		return nil
	}

	return fmt.Errorf("unregister failed with status %d", resp.StatusCode)
}
