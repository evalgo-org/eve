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
// Note: Legacy SearchAction struct has been removed.
// Use SemanticAction with NewSemanticSearchAction instead.

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

// ParseSPARQLAction parses a JSON-LD SPARQL action as SemanticAction
func ParseSPARQLAction(data []byte) (*SemanticAction, error) {
	var typeCheck struct {
		Type string `json:"@type"`
	}

	if err := json.Unmarshal(data, &typeCheck); err != nil {
		return nil, fmt.Errorf("failed to parse @type: %w", err)
	}

	if typeCheck.Type != "SearchAction" {
		return nil, fmt.Errorf("expected SearchAction, got: %s", typeCheck.Type)
	}

	var action SemanticAction
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

	// If URL is empty, check for sparql_endpoint in additionalProperty
	if url == "" {
		if sparqlEndpoint, ok := props["sparql_endpoint"].(string); ok {
			url = sparqlEndpoint
		}
	}

	// Also check for content_type in additionalProperty to set EncodingFormat
	if contentType, ok := props["content_type"].(string); ok && endpoint.EncodingFormat == "" {
		endpoint.EncodingFormat = contentType
	}

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

// NewSemanticSearchAction creates a SearchAction using SemanticAction
func NewSemanticSearchAction(id, name string, query *SearchQuery, target *SPARQLEndpoint) *SemanticAction {
	action := &SemanticAction{
		Context:      "https://schema.org",
		Type:         "SearchAction",
		Identifier:   id,
		Name:         name,
		ActionStatus: "PotentialActionStatus",
		Properties:   make(map[string]interface{}),
	}

	if query != nil {
		action.Properties["query"] = query
	}
	if target != nil {
		action.Properties["target"] = target
	}

	return action
}

// GetSearchQueryFromAction extracts SearchQuery from SemanticAction properties
func GetSearchQueryFromAction(action *SemanticAction) (*SearchQuery, error) {
	if action == nil || action.Properties == nil {
		return nil, fmt.Errorf("action or properties is nil")
	}

	query, ok := action.Properties["query"]
	if !ok {
		return nil, fmt.Errorf("no query found in action properties")
	}

	switch v := query.(type) {
	case *SearchQuery:
		return v, nil
	case SearchQuery:
		return &v, nil
	case string:
		// Allow inline query string - convert to SearchQuery
		return &SearchQuery{
			Type:       "SearchAction",
			QueryInput: v,
		}, nil
	case map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal SearchQuery: %w", err)
		}
		var sq SearchQuery
		if err := json.Unmarshal(data, &sq); err != nil {
			return nil, fmt.Errorf("failed to unmarshal SearchQuery: %w", err)
		}
		return &sq, nil
	default:
		return nil, fmt.Errorf("unexpected query type: %T", query)
	}
}

// GetSPARQLEndpointFromAction extracts SPARQLEndpoint from SemanticAction properties
func GetSPARQLEndpointFromAction(action *SemanticAction) (*SPARQLEndpoint, error) {
	if action == nil || action.Properties == nil {
		return nil, fmt.Errorf("action or properties is nil")
	}

	target, ok := action.Properties["target"]
	if !ok {
		return nil, fmt.Errorf("no target found in action properties")
	}

	switch v := target.(type) {
	case *SPARQLEndpoint:
		return v, nil
	case SPARQLEndpoint:
		return &v, nil
	case map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal SPARQLEndpoint: %w", err)
		}
		var endpoint SPARQLEndpoint
		if err := json.Unmarshal(data, &endpoint); err != nil {
			return nil, fmt.Errorf("failed to unmarshal SPARQLEndpoint: %w", err)
		}
		return &endpoint, nil
	default:
		return nil, fmt.Errorf("unexpected target type: %T", target)
	}
}
