package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"eve.evalgo.org/semantic"
)

// registryCache caches service URLs from registry to avoid repeated lookups
var (
	registryCache    = make(map[string]string)
	registryCacheMu  sync.RWMutex
	registryCacheTTL = 5 * time.Minute
	lastCacheUpdate  time.Time
)

// Service represents a registered service from the registry API
type Service struct {
	Identifier string                 `json:"identifier"`
	Name       string                 `json:"name"`
	URL        string                 `json:"url"`
	Properties map[string]interface{} `json:"additionalProperty"`
}

// ResolveRegistryURL resolves registry://servicename/path to http://host:port/path
// Example: registry://infisicalservice/v1/api/semantic/action -> http://localhost:8093/v1/api/semantic/action
func ResolveRegistryURL(registryURL string) (string, error) {
	if !strings.HasPrefix(registryURL, "registry://") {
		return registryURL, nil // Not a registry URL, return as-is
	}

	// Check cache first
	registryCacheMu.RLock()
	if time.Since(lastCacheUpdate) < registryCacheTTL {
		if cachedURL, ok := registryCache[registryURL]; ok {
			registryCacheMu.RUnlock()
			return cachedURL, nil
		}
	}
	registryCacheMu.RUnlock()

	// Parse registry://servicename/path
	urlPart := strings.TrimPrefix(registryURL, "registry://")
	parts := strings.SplitN(urlPart, "/", 2)
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid registry URL: %s", registryURL)
	}

	serviceName := parts[0]
	path := ""
	if len(parts) > 1 {
		path = "/" + parts[1]
	}

	// Query registry service
	registryAPIURL := os.Getenv("REGISTRYSERVICE_API_URL")
	if registryAPIURL == "" {
		registryAPIURL = "http://localhost:8096" // Default
	}

	resp, err := http.Get(registryAPIURL + "/v1/api/services")
	if err != nil {
		return "", fmt.Errorf("failed to query registry: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	var services []Service
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return "", fmt.Errorf("failed to decode registry response: %w", err)
	}

	// Find matching service
	for _, svc := range services {
		if svc.Identifier == serviceName {
			resolvedURL := svc.URL + path

			// Update cache
			registryCacheMu.Lock()
			registryCache[registryURL] = resolvedURL
			lastCacheUpdate = time.Now()
			registryCacheMu.Unlock()

			return resolvedURL, nil
		}
	}

	return "", fmt.Errorf("service %s not found in registry", serviceName)
}

// extractTargetURL extracts the service endpoint URL from an action's _meta
// Supports registry:// URLs which are resolved via the registry service
func extractTargetURL(action *semantic.SemanticScheduledAction) string {
	if action.Meta == nil || action.Meta.URL == "" {
		return ""
	}

	rawURL := action.Meta.URL

	// Resolve registry:// URLs
	if strings.HasPrefix(rawURL, "registry://") {
		resolved, err := ResolveRegistryURL(rawURL)
		if err != nil {
			// Log error but return original URL as fallback
			fmt.Fprintf(os.Stderr, "Warning: failed to resolve %s: %v\n", rawURL, err)
			return rawURL
		}
		return resolved
	}

	return rawURL
}

// URLBasedExecutor routes actions to any service with /v1/api/semantic/action endpoint
// This is the universal executor for all semantic service endpoints.
type URLBasedExecutor struct{}

func (e *URLBasedExecutor) CanHandle(action *semantic.SemanticScheduledAction) bool {
	targetURL := extractTargetURL(action)
	return strings.Contains(targetURL, "/v1/api/semantic/action")
}

func (e *URLBasedExecutor) Execute(action *semantic.SemanticScheduledAction) (string, error) {
	targetURL := extractTargetURL(action)
	if targetURL == "" {
		return "", fmt.Errorf("target URL is required")
	}

	// Clone the action to avoid modifying the original
	actionCopy := *action

	// Remove execution metadata (Result, Error, ActionStatus, StartTime, EndTime) before sending
	// These are populated by the executor after the service responds
	actionCopy.Result = nil
	actionCopy.Error = nil
	actionCopy.ActionStatus = ""
	actionCopy.StartTime = nil
	actionCopy.EndTime = nil

	// Remove control metadata (Meta) before sending
	// Meta is for the when-daemon executor only (routing, retry, singleton)
	// Services should only receive semantic properties in additionalProperty
	actionCopy.Meta = nil

	// Remove the service endpoint URL from target before sending, but keep other URLs (like S3 bucket URLs)
	// The service endpoint URL is for routing only and should not be part of the semantic action payload
	// But URLs like S3 bucket endpoints are part of the semantic action and should be preserved
	if actionCopy.Target != nil {
		switch target := actionCopy.Target.(type) {
		case *semantic.EntryPoint:
			targetCopy := *target
			// Only remove URL if it matches the service endpoint (routing URL)
			if targetCopy.URL == targetURL {
				targetCopy.URL = ""
			}
			actionCopy.Target = &targetCopy
		case map[string]interface{}:
			targetMap := make(map[string]interface{})
			for k, v := range target {
				// Only remove url if it matches the service endpoint (routing URL)
				if k == "url" {
					if urlStr, ok := v.(string); ok && urlStr == targetURL {
						continue // Skip this field
					}
				}
				// Handle additionalProperty to remove service endpoint URL from there too
				if k == "additionalProperty" {
					if props, ok := v.(map[string]interface{}); ok {
						propsCopy := make(map[string]interface{})
						for pk, pv := range props {
							// Remove url from additionalProperty if it's the service endpoint
							if pk == "url" {
								if urlStr, ok := pv.(string); ok && urlStr == targetURL {
									continue // Skip this field
								}
							}
							propsCopy[pk] = pv
						}
						targetMap[k] = propsCopy
						continue
					}
				}
				targetMap[k] = v
			}
			actionCopy.Target = targetMap
		}
	}

	// Serialize action to JSON-LD
	jsonldData, err := json.Marshal(&actionCopy)
	if err != nil {
		return "", fmt.Errorf("failed to serialize action: %w", err)
	}

	// DEBUG: Log what's being sent
	fmt.Fprintf(os.Stderr, "DEBUG URL_EXECUTOR sending to %s: %s\n", targetURL, string(jsonldData))

	// POST to service (services expect application/json, not application/ld+json)
	resp, err := http.Post(targetURL, "application/json", bytes.NewReader(jsonldData))
	if err != nil {
		return "", fmt.Errorf("failed to call service: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read service response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return string(body), fmt.Errorf("service returned %d: %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

// FindServicesByCapability queries the registry for services with a specific capability
func FindServicesByCapability(capability string) ([]*Service, error) {
	registryAPIURL := os.Getenv("REGISTRYSERVICE_API_URL")
	if registryAPIURL == "" {
		registryAPIURL = "http://localhost:8096" // Default
	}

	resp, err := http.Get(registryAPIURL + "/v1/api/services")
	if err != nil {
		return nil, fmt.Errorf("failed to query registry: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	var services []*Service
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return nil, fmt.Errorf("failed to decode registry response: %w", err)
	}

	// Filter services by capability
	var matches []*Service
	for _, svc := range services {
		if svc.Properties == nil {
			continue
		}
		// Properties is map[string]interface{}, capabilities is an array
		if caps, ok := svc.Properties["capabilities"]; ok {
			if capArray, ok := caps.([]interface{}); ok {
				for _, cap := range capArray {
					if capStr, ok := cap.(string); ok && capStr == capability {
						matches = append(matches, svc)
						break
					}
				}
			}
		}
	}

	return matches, nil
}
