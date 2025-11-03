package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// Registry manages service discovery and registration
type Registry struct {
	filePath string
	services map[string]*Service
	mu       sync.RWMutex
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

// registryFile represents the JSON-LD structure
type registryFile struct {
	Context         string         `json:"@context"`
	Type            string         `json:"@type"`
	Identifier      string         `json:"identifier"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	DateModified    string         `json:"dateModified"`
	ItemListElement []registryItem `json:"itemListElement"`
}

type registryItem struct {
	Type     string   `json:"@type"`
	Position int      `json:"position"`
	Item     *Service `json:"item"`
}

// NewRegistry creates a new registry instance
func NewRegistry(filePath string) (*Registry, error) {
	r := &Registry{
		filePath: filePath,
		services: make(map[string]*Service),
	}

	// Load existing registry if it exists
	if err := r.Load(); err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	return r, nil
}

// Load reads the registry from the JSON-LD file
func (r *Registry) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, that's OK
		}
		return fmt.Errorf("failed to read registry file: %w", err)
	}

	var rf registryFile
	if err := json.Unmarshal(data, &rf); err != nil {
		return fmt.Errorf("failed to parse registry: %w", err)
	}

	// Build services map
	r.services = make(map[string]*Service)
	for _, item := range rf.ItemListElement {
		if item.Item != nil {
			r.services[item.Item.ID] = item.Item
		}
	}

	return nil
}

// Save writes the registry to the JSON-LD file
func (r *Registry) Save() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Build registry structure
	rf := registryFile{
		Context:      "https://schema.org",
		Type:         "ItemList",
		Identifier:   "service-registry",
		Name:         "Microservices Registry",
		Description:  "Central registry of all running microservices with their endpoints and capabilities",
		DateModified: time.Now().Format(time.RFC3339),
	}

	position := 1
	for _, svc := range r.services {
		rf.ItemListElement = append(rf.ItemListElement, registryItem{
			Type:     "ListItem",
			Position: position,
			Item:     svc,
		})
		position++
	}

	data, err := json.MarshalIndent(rf, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(r.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry file: %w", err)
	}

	return nil
}

// Register adds or updates a service in the registry
func (r *Registry) Register(svc *Service) error {
	r.mu.Lock()
	r.services[svc.ID] = svc
	r.mu.Unlock()

	return r.Save()
}

// Unregister removes a service from the registry
func (r *Registry) Unregister(serviceID string) error {
	r.mu.Lock()
	delete(r.services, serviceID)
	r.mu.Unlock()

	return r.Save()
}

// Get retrieves a service by ID
func (r *Registry) Get(serviceID string) (*Service, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	svc, exists := r.services[serviceID]
	if !exists {
		return nil, fmt.Errorf("service not found: %s", serviceID)
	}

	return svc, nil
}

// List returns all registered services
func (r *Registry) List() []*Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]*Service, 0, len(r.services))
	for _, svc := range r.services {
		services = append(services, svc)
	}

	return services
}

// FindByCapability returns services that have a specific capability
func (r *Registry) FindByCapability(capability string) []*Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []*Service
	for _, svc := range r.services {
		for _, cap := range svc.Properties.Capabilities {
			if cap == capability {
				matches = append(matches, svc)
				break
			}
		}
	}

	return matches
}

// HealthCheck checks if a service is responding
func (r *Registry) HealthCheck(serviceID string) (bool, error) {
	svc, err := r.Get(serviceID)
	if err != nil {
		return false, err
	}

	// If no health check URL, just try the main URL
	checkURL := svc.Properties.HealthCheck
	if checkURL == "" {
		checkURL = svc.URL
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(checkURL)
	if err != nil {
		return false, nil
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300, nil
}

// HealthCheckAll checks health of all registered services
func (r *Registry) HealthCheckAll() map[string]bool {
	services := r.List()
	results := make(map[string]bool)

	for _, svc := range services {
		healthy, _ := r.HealthCheck(svc.ID)
		results[svc.ID] = healthy
	}

	return results
}

// GetServiceURL returns the URL for a service by ID
func (r *Registry) GetServiceURL(serviceID string) (string, error) {
	svc, err := r.Get(serviceID)
	if err != nil {
		return "", err
	}
	return svc.URL, nil
}

// Default global registry instance
var defaultRegistry *Registry
var registryOnce sync.Once

// DefaultRegistry returns the default global registry
func DefaultRegistry() (*Registry, error) {
	var err error
	registryOnce.Do(func() {
		// Default to /home/opunix/registry.json
		registryPath := os.Getenv("SERVICE_REGISTRY_PATH")
		if registryPath == "" {
			registryPath = "/home/opunix/registry.json"
		}
		defaultRegistry, err = NewRegistry(registryPath)
	})
	return defaultRegistry, err
}

// Helper functions for common operations using default registry

// Get retrieves a service from the default registry
func Get(serviceID string) (*Service, error) {
	reg, err := DefaultRegistry()
	if err != nil {
		return nil, err
	}
	return reg.Get(serviceID)
}

// GetURL returns the URL for a service from the default registry
func GetURL(serviceID string) (string, error) {
	// If REGISTRYSERVICE_API_URL is set, query the HTTP API instead of using file-based registry
	registryURL := os.Getenv("REGISTRYSERVICE_API_URL")
	if registryURL != "" {
		// Fetch services from HTTP API
		servicesURL := fmt.Sprintf("%s/v1/api/services", registryURL)
		resp, err := http.Get(servicesURL)
		if err != nil {
			return "", fmt.Errorf("failed to query registry API: %w", err)
		}
		defer resp.Body.Close()

		var services []*Service
		if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
			return "", fmt.Errorf("failed to parse registry response: %w", err)
		}

		// Find the service by ID
		for _, svc := range services {
			if svc.ID == serviceID {
				return svc.URL, nil
			}
		}
		return "", fmt.Errorf("service not found: %s", serviceID)
	}

	// Fall back to file-based registry
	reg, err := DefaultRegistry()
	if err != nil {
		return "", err
	}
	return reg.GetServiceURL(serviceID)
}

// List returns all services from the default registry
func List() ([]*Service, error) {
	reg, err := DefaultRegistry()
	if err != nil {
		return nil, err
	}
	return reg.List(), nil
}

// FindByCapability finds services by capability in the default registry
func FindByCapability(capability string) ([]*Service, error) {
	reg, err := DefaultRegistry()
	if err != nil {
		return nil, err
	}
	return reg.FindByCapability(capability), nil
}

// HealthCheck checks a service's health in the default registry
func HealthCheck(serviceID string) (bool, error) {
	reg, err := DefaultRegistry()
	if err != nil {
		return false, err
	}
	return reg.HealthCheck(serviceID)
}

// ReadRegistryFromURL fetches and parses a registry from a URL
func ReadRegistryFromURL(url string) ([]*Service, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch registry: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rf registryFile
	if err := json.Unmarshal(data, &rf); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	services := make([]*Service, 0, len(rf.ItemListElement))
	for _, item := range rf.ItemListElement {
		if item.Item != nil {
			services = append(services, item.Item)
		}
	}

	return services, nil
}
