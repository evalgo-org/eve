package semantic

import (
	"encoding/json"
	"fmt"
)

// SPARQL Semantic Types for PoolParty and Triple Store Operations
// These types map SPARQL operations to Schema.org vocabulary for semantic orchestration

// ============================================================================
// SearchAction for SPARQL Queries
// ============================================================================

// SearchAction represents a SPARQL query operation
// Maps to Schema.org SearchAction for querying knowledge graphs
type SearchAction struct {
	Context      string                 `json:"@context,omitempty"`
	Type         string                 `json:"@type"` // "SearchAction"
	Identifier   string                 `json:"identifier"`
	Name         string                 `json:"name,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Query        *SearchQuery           `json:"query"`                  // SPARQL query details
	Target       *SPARQLEndpoint        `json:"target"`                 // SPARQL endpoint/project
	Result       interface{}            `json:"result,omitempty"`       // Query result (Dataset or text)
	ActionStatus string                 `json:"actionStatus,omitempty"` // Status of the query
	Error        *PropertyValue         `json:"error,omitempty"`        // Error details if failed
	Properties   map[string]interface{} `json:"additionalProperty,omitempty"`
}

// SearchQuery represents a SPARQL query
// Can be inline query text or reference to a query file
type SearchQuery struct {
	Type       string `json:"@type"`                // "SearchAction"
	QueryInput string `json:"queryInput,omitempty"` // Inline SPARQL query
	ContentURL string `json:"contentUrl,omitempty"` // Path to SPARQL template file
	// Template parameters for dynamic queries
	Parameters map[string]interface{} `json:"additionalProperty,omitempty"`
}

// SPARQLEndpoint represents a PoolParty project or SPARQL endpoint
// Maps to Schema.org DataCatalog for semantic representation
type SPARQLEndpoint struct {
	Type           string                 `json:"@type"`                        // "DataCatalog"
	Identifier     string                 `json:"identifier"`                   // Project ID or endpoint name
	URL            string                 `json:"url"`                          // Base URL of PoolParty/endpoint
	EncodingFormat string                 `json:"encodingFormat,omitempty"`     // Desired result format
	Properties     map[string]interface{} `json:"additionalProperty,omitempty"` // Credentials, etc.
}

// ============================================================================
// Helper Functions
// ============================================================================

// ParseSPARQLAction parses a JSON-LD SPARQL action and returns SearchAction
func ParseSPARQLAction(data []byte) (*SearchAction, error) {
	var typeCheck struct {
		Type string `json:"@type"`
	}

	if err := json.Unmarshal(data, &typeCheck); err != nil {
		return nil, fmt.Errorf("failed to parse @type: %w", err)
	}

	if typeCheck.Type != "SearchAction" {
		return nil, fmt.Errorf("expected SearchAction, got: %s", typeCheck.Type)
	}

	var action SearchAction
	if err := json.Unmarshal(data, &action); err != nil {
		return nil, fmt.Errorf("failed to parse SearchAction: %w", err)
	}

	return &action, nil
}

// ExtractSPARQLCredentials extracts connection details from a SPARQLEndpoint
// Returns: baseURL, username, password, projectID, error
func ExtractSPARQLCredentials(endpoint *SPARQLEndpoint) (string, string, string, string, error) {
	if endpoint == nil {
		return "", "", "", "", fmt.Errorf("endpoint is nil")
	}

	props := endpoint.Properties
	username, _ := props["username"].(string)
	password, _ := props["password"].(string)

	// Parse URL to extract base URL and project ID
	// Expected format: https://host/PoolParty/sparql/PROJECT_ID
	// Need to extract: baseURL=https://host, projectID=PROJECT_ID
	url := endpoint.URL

	// Check if URL contains /PoolParty/sparql/
	// If so, extract base URL and project ID from it
	// Otherwise, use the URL as base URL and Identifier as project ID
	if idx := findIndex(url, "/PoolParty/sparql/"); idx != -1 {
		baseURL := url[:idx]
		projectID := url[idx+len("/PoolParty/sparql/"):]
		// Remove any trailing slashes or query parameters
		if slashIdx := findIndex(projectID, "/"); slashIdx != -1 {
			projectID = projectID[:slashIdx]
		}
		if qIdx := findIndex(projectID, "?"); qIdx != -1 {
			projectID = projectID[:qIdx]
		}
		return baseURL, username, password, projectID, nil
	}

	// Fallback: use URL as base and Identifier as project
	return endpoint.URL, username, password, endpoint.Identifier, nil
}

// findIndex returns the index of substr in s, or -1 if not found
func findIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// ExtractQueryTemplate extracts the query template path or inline query
// Returns: templatePath, inlineQuery, parameters, error
func ExtractQueryTemplate(query *SearchQuery) (string, string, map[string]interface{}, error) {
	if query == nil {
		return "", "", nil, fmt.Errorf("query is nil")
	}

	// Template file takes precedence
	if query.ContentURL != "" {
		return query.ContentURL, "", query.Parameters, nil
	}

	// Otherwise use inline query
	if query.QueryInput != "" {
		return "", query.QueryInput, query.Parameters, nil
	}

	return "", "", nil, fmt.Errorf("no query specified (neither contentUrl nor queryInput)")
}

// NewSearchAction creates a new semantic SearchAction for SPARQL queries
func NewSearchAction(id, name, queryTemplate string, endpoint *SPARQLEndpoint) *SearchAction {
	return &SearchAction{
		Context:    "https://schema.org",
		Type:       "SearchAction",
		Identifier: id,
		Name:       name,
		Query: &SearchQuery{
			Type:       "SearchAction",
			ContentURL: queryTemplate,
		},
		Target:       endpoint,
		ActionStatus: "PotentialActionStatus",
	}
}

// NewSPARQLEndpoint creates a new semantic SPARQL endpoint representation
func NewSPARQLEndpoint(url, projectID, username, password, contentType string) *SPARQLEndpoint {
	endpoint := &SPARQLEndpoint{
		Type:           "DataCatalog",
		Identifier:     projectID,
		URL:            url,
		EncodingFormat: contentType,
	}

	if username != "" || password != "" {
		endpoint.Properties = map[string]interface{}{
			"username": username,
			"password": password,
		}
	}

	return endpoint
}
