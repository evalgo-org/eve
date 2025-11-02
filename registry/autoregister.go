package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

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
