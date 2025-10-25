// Package assets provides utilities for interacting with external asset management APIs.
// It includes functions for retrieving component inventories and other asset-related
// operations with proper URL construction, authentication, and error handling.
//
// The package focuses on HTTP API interactions with external services, providing
// clean interfaces for asset management operations while handling authentication,
// query parameter construction, and response processing.
package assets

import (
	"io"
	"net/http"
	"net/url"
	"strconv"

	eve "eve.evalgo.org/common"
)

// ComponentsQueryParams represents the query parameters for component inventory requests.
// This struct provides a structured way to build and validate query parameters
// for the components API endpoint, ensuring type safety and proper URL encoding.
type ComponentsQueryParams struct {
	Limit       int    // Maximum number of components to return (default: 50)
	Offset      int    // Number of components to skip for pagination (default: 0)
	OrderNumber *int   // Optional order number filter (null for no filter)
	Sort        string // Field to sort by (default: "created_at")
	Order       string // Sort direction: "asc" or "desc" (default: "desc")
	Expand      bool   // Whether to expand related data (default: false)
}

// DefaultComponentsParams returns the default query parameters for component requests.
// These defaults match the original hardcoded values and provide sensible starting
// values for most component inventory queries.
//
// Default values:
//   - Limit: 50 components per request
//   - Offset: 0 (start from beginning)
//   - OrderNumber: null (no filtering by order number)
//   - Sort: "created_at" (sort by creation date)
//   - Order: "desc" (newest first)
//   - Expand: false (compact response format)
//
// Returns:
//
//	ComponentsQueryParams with default values configured
//
// Example usage:
//
//	params := DefaultComponentsParams()
//	params.Limit = 100  // Override default limit
//	response := InvComponentsWithParams(baseURL, token, params)
func DefaultComponentsParams() ComponentsQueryParams {
	return ComponentsQueryParams{
		Limit:       50,
		Offset:      0,
		OrderNumber: nil,
		Sort:        "created_at",
		Order:       "desc",
		Expand:      false,
	}
}

// ToURLValues converts ComponentsQueryParams to url.Values for HTTP requests.
// This method handles proper type conversion and null value representation
// for all query parameters, ensuring they are correctly encoded in the URL.
//
// Parameter handling:
//   - Numeric values are converted to strings
//   - Boolean values use "true"/"false" string representation
//   - Nil OrderNumber is represented as "null" string
//   - All values are properly URL-encoded
//
// Returns:
//
//	url.Values ready to be encoded as query string
//
// Example usage:
//
//	params := DefaultComponentsParams()
//	values := params.ToURLValues()
//	queryString := values.Encode()  // "limit=50&offset=0&order_number=null..."
func (p ComponentsQueryParams) ToURLValues() url.Values {
	values := url.Values{}

	values.Set("limit", strconv.Itoa(p.Limit))
	values.Set("offset", strconv.Itoa(p.Offset))

	if p.OrderNumber != nil {
		values.Set("order_number", strconv.Itoa(*p.OrderNumber))
	} else {
		values.Set("order_number", "null")
	}

	values.Set("sort", p.Sort)
	values.Set("order", p.Order)
	values.Set("expand", strconv.FormatBool(p.Expand))

	return values
}

// InvComponents retrieves component inventory data from an external API using default parameters.
// This function provides a backward-compatible interface that maintains the same behavior
// as the original implementation while using the new structured parameter system internally.
//
// The function performs an authenticated GET request to the components API endpoint
// with predefined query parameters optimized for general inventory retrieval.
//
// API Endpoint: GET {baseURL}/api/v1/components
// Query Parameters:
//   - limit=50: Return up to 50 components
//   - offset=0: Start from the first component
//   - order_number=null: No order number filtering
//   - sort=created_at: Sort by creation timestamp
//   - order=desc: Newest components first
//   - expand=false: Return compact component data
//
// Authentication:
//
//	Uses Bearer token authentication with the provided token
//
// Parameters:
//   - baseURL: Base URL of the API server (e.g., "https://api.example.com")
//   - token: JWT or API token for authentication
//
// Returns:
//
//	string: JSON response body containing component inventory data
//	       Empty string if request fails (error logged via eve.Logger)
//
// Error Handling:
//   - HTTP request errors are logged using eve.Logger.Error()
//   - Network failures return empty string
//   - Invalid responses return empty string
//   - No panics or exceptions are thrown
//
// Example usage:
//
//	components := InvComponents("https://api.example.com", "your-auth-token")
//	// Parse the JSON response as needed
//
// Note: For more control over query parameters, use InvComponentsWithParams()
func InvComponents(baseURL string, token string) string {
	return InvComponentsWithParams(baseURL, token, DefaultComponentsParams())
}

// InvComponentsWithParams retrieves component inventory data with custom query parameters.
// This function provides full control over the API request parameters while maintaining
// proper URL construction, authentication, and error handling.
//
// The function builds a properly formatted API request with custom query parameters,
// performs authentication using Bearer token, and returns the raw JSON response.
//
// API Endpoint: GET {baseURL}/api/v1/components
// Query Parameters: Customizable via ComponentsQueryParams struct
//
// Authentication:
//   - Uses Bearer token authentication
//   - Sets proper Accept headers for JSON response
//   - Includes all required headers for API compatibility
//
// Parameters:
//   - baseURL: Base URL of the API server (must not include trailing slash)
//   - token: Authentication token (JWT or API key)
//   - params: ComponentsQueryParams struct with query configuration
//
// Returns:
//
//	string: JSON response body from the API
//	       Empty string if request fails (errors are logged)
//
// Error Handling:
//   - URL construction errors are logged and return empty string
//   - HTTP request failures are logged via eve.Logger.Error()
//   - Response reading errors are logged and return empty string
//   - Network timeouts and connection issues are handled gracefully
//
// Query Parameter Examples:
//
//	// Get first 25 components, sorted by name
//	params := ComponentsQueryParams{
//	    Limit: 25,
//	    Offset: 0,
//	    Sort: "name",
//	    Order: "asc",
//	    Expand: true,
//	}
//
//	// Get components for specific order
//	orderNum := 12345
//	params := ComponentsQueryParams{
//	    Limit: 100,
//	    OrderNumber: &orderNum,
//	    Sort: "created_at",
//	    Order: "desc",
//	}
//
// Example usage:
//
//	params := DefaultComponentsParams()
//	params.Limit = 100
//	params.Expand = true
//
//	response := InvComponentsWithParams("https://api.example.com", "token", params)
//
//	// Parse JSON response
//	var components []Component
//	json.Unmarshal([]byte(response), &components)
//
// Security Considerations:
//   - Token is transmitted via Authorization header (not URL)
//   - HTTPS should be used for all API communications
//   - Tokens should be securely stored and rotated regularly
//
// Performance Notes:
//   - Uses http.DefaultClient (consider custom client for production)
//   - Response body is fully loaded into memory
//   - No built-in retry logic for failed requests
func InvComponentsWithParams(baseURL string, token string, params ComponentsQueryParams) string {
	// Parse base URL to ensure it's valid
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		eve.Logger.Error("Invalid base URL:", err)
		return ""
	}

	// Build the complete API endpoint URL
	parsedURL.Path = "/api/v1/components"
	parsedURL.RawQuery = params.ToURLValues().Encode()

	// Create HTTP GET request
	req, err := http.NewRequest("GET", parsedURL.String(), nil)
	if err != nil {
		eve.Logger.Error("Failed to create HTTP request:", err)
		return ""
	}

	// Set required headers for API authentication and content negotiation
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Execute the HTTP request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		eve.Logger.Error("HTTP request failed:", err)
		return ""
	}
	defer res.Body.Close()

	// Read the complete response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		eve.Logger.Error("Failed to read response body:", err)
		return ""
	}

	// Check for HTTP error status codes
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		eve.Logger.Error("API returned error status:", res.StatusCode, string(body))
		return ""
	}

	return string(body)
}
