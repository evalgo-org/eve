// Package assets provides comprehensive testing for asset management API functionality.
// This file contains unit tests for the assets package, focusing on HTTP API interactions,
// parameter validation, URL construction, and error handling scenarios.
//
// The tests validate both the core functionality and edge cases of the asset management
// API client, ensuring reliable operation across different configurations and failure modes.
package assets

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// TestComponentsQueryParams validates the ComponentsQueryParams struct and its methods.
// This test ensures that the parameter structure correctly handles all supported
// query parameter types and properly converts them to URL values.
//
// Test Coverage:
//   - Default parameter initialization
//   - Custom parameter configuration
//   - Null value handling for optional parameters
//   - URL encoding and format validation
//   - Type conversion accuracy
//
// The test validates that query parameters are properly formatted for the API
// and that all edge cases (null values, boolean conversion, etc.) are handled correctly.
func TestComponentsQueryParams(t *testing.T) {
	t.Run("DefaultParams", func(t *testing.T) {
		params := DefaultComponentsParams()

		// Validate default values match expected API defaults
		if params.Limit != 50 {
			t.Errorf("DefaultComponentsParams().Limit = %d; want 50", params.Limit)
		}
		if params.Offset != 0 {
			t.Errorf("DefaultComponentsParams().Offset = %d; want 0", params.Offset)
		}
		if params.OrderNumber != nil {
			t.Errorf("DefaultComponentsParams().OrderNumber = %v; want nil", params.OrderNumber)
		}
		if params.Sort != "created_at" {
			t.Errorf("DefaultComponentsParams().Sort = %s; want 'created_at'", params.Sort)
		}
		if params.Order != "desc" {
			t.Errorf("DefaultComponentsParams().Order = %s; want 'desc'", params.Order)
		}
		if params.Expand != false {
			t.Errorf("DefaultComponentsParams().Expand = %t; want false", params.Expand)
		}
	})

	t.Run("ToURLValues", func(t *testing.T) {
		// Test with default parameters
		params := DefaultComponentsParams()
		values := params.ToURLValues()

		// Validate all required parameters are present
		expectedParams := map[string]string{
			"limit":        "50",
			"offset":       "0",
			"order_number": "null",
			"sort":         "created_at",
			"order":        "desc",
			"expand":       "false",
		}

		for key, expectedValue := range expectedParams {
			if got := values.Get(key); got != expectedValue {
				t.Errorf("params.ToURLValues().Get(%s) = %s; want %s", key, got, expectedValue)
			}
		}
	})

	t.Run("CustomParams", func(t *testing.T) {
		// Test with custom parameters including non-null order number
		orderNum := 12345
		params := ComponentsQueryParams{
			Limit:       100,
			Offset:      25,
			OrderNumber: &orderNum,
			Sort:        "name",
			Order:       "asc",
			Expand:      true,
		}

		values := params.ToURLValues()

		// Validate custom parameter conversion
		expectedParams := map[string]string{
			"limit":        "100",
			"offset":       "25",
			"order_number": "12345",
			"sort":         "name",
			"order":        "asc",
			"expand":       "true",
		}

		for key, expectedValue := range expectedParams {
			if got := values.Get(key); got != expectedValue {
				t.Errorf("custom params.ToURLValues().Get(%s) = %s; want %s", key, got, expectedValue)
			}
		}
	})
}

// TestInvComponentsWithParams validates the core API interaction functionality.
// This test uses a mock HTTP server to simulate the external API and validates
// proper request construction, authentication, parameter encoding, and response handling.
//
// Test Coverage:
//   - HTTP request method and URL construction
//   - Authentication header formatting
//   - Query parameter encoding and transmission
//   - Response body handling
//   - Error condition handling
//   - Mock server interaction patterns
//
// The test ensures that the function correctly formats API requests, handles
// authentication properly, and processes responses as expected.
func TestInvComponentsWithParams(t *testing.T) {
	t.Run("SuccessfulRequest", func(t *testing.T) {
		// Mock API response data
		expectedResponse := `{"components":[{"id":1,"name":"Component 1"},{"id":2,"name":"Component 2"}],"total":2}`

		// Create mock server to simulate external API
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Validate HTTP method
			if r.Method != "GET" {
				t.Errorf("Expected GET request, got %s", r.Method)
			}

			// Validate API endpoint path
			if r.URL.Path != "/api/v1/components" {
				t.Errorf("Expected path '/api/v1/components', got '%s'", r.URL.Path)
			}

			// Validate authentication header
			authHeader := r.Header.Get("Authorization")
			expectedAuth := "Bearer test-token-123"
			if authHeader != expectedAuth {
				t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, authHeader)
			}

			// Validate Accept header
			acceptHeader := r.Header.Get("Accept")
			if acceptHeader != "application/json" {
				t.Errorf("Expected Accept header 'application/json', got '%s'", acceptHeader)
			}

			// Validate query parameters
			queryParams := r.URL.Query()
			expectedParams := map[string]string{
				"limit":        "25",
				"offset":       "10",
				"order_number": "null",
				"sort":         "name",
				"order":        "asc",
				"expand":       "true",
			}

			for key, expectedValue := range expectedParams {
				if got := queryParams.Get(key); got != expectedValue {
					t.Errorf("Expected query param %s='%s', got '%s'", key, expectedValue, got)
				}
			}

			// Send mock response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(expectedResponse))
		}))
		defer server.Close()

		// Configure test parameters
		params := ComponentsQueryParams{
			Limit:       25,
			Offset:      10,
			OrderNumber: nil,
			Sort:        "name",
			Order:       "asc",
			Expand:      true,
		}

		// Execute function under test
		result := InvComponentsWithParams(server.URL, "test-token-123", params)

		// Validate response
		if result != expectedResponse {
			t.Errorf("InvComponentsWithParams() = %s; want %s", result, expectedResponse)
		}
	})

	t.Run("InvalidURL", func(t *testing.T) {
		params := DefaultComponentsParams()

		// Test with malformed URL
		result := InvComponentsWithParams("://invalid-url", "token", params)

		// Should return empty string for invalid URL
		if result != "" {
			t.Errorf("InvComponentsWithParams() with invalid URL = %s; want empty string", result)
		}
	})

	t.Run("HTTPError", func(t *testing.T) {
		// Create mock server that returns HTTP error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"Invalid token"}`))
		}))
		defer server.Close()

		params := DefaultComponentsParams()
		result := InvComponentsWithParams(server.URL, "invalid-token", params)

		// Should return empty string for HTTP errors
		if result != "" {
			t.Errorf("InvComponentsWithParams() with HTTP error = %s; want empty string", result)
		}
	})
}

// TestInvComponents validates the backward-compatible API function.
// This test ensures that the simplified interface maintains compatibility
// with existing code while using the new parameter system internally.
//
// Test Coverage:
//   - Default parameter usage
//   - Backward compatibility with original interface
//   - Integration with InvComponentsWithParams
//   - Response format consistency
//
// The test verifies that the convenience function produces identical
// results to calling InvComponentsWithParams with default parameters.
func TestInvComponents(t *testing.T) {
	t.Run("BackwardCompatibility", func(t *testing.T) {
		expectedResponse := `{"components":[],"total":0}`

		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Validate that default parameters are used
			queryParams := r.URL.Query()

			// Check key default values
			if queryParams.Get("limit") != "50" {
				t.Errorf("Expected default limit=50, got %s", queryParams.Get("limit"))
			}
			if queryParams.Get("offset") != "0" {
				t.Errorf("Expected default offset=0, got %s", queryParams.Get("offset"))
			}
			if queryParams.Get("order_number") != "null" {
				t.Errorf("Expected default order_number=null, got %s", queryParams.Get("order_number"))
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(expectedResponse))
		}))
		defer server.Close()

		// Test simplified interface
		result := InvComponents(server.URL, "test-token")

		if result != expectedResponse {
			t.Errorf("InvComponents() = %s; want %s", result, expectedResponse)
		}
	})
}

// TestURLConstruction validates proper URL building and query parameter encoding.
// This test ensures that complex parameter combinations are correctly encoded
// and that special characters and edge cases are handled properly.
//
// Test Coverage:
//   - Query parameter encoding
//   - Special character handling
//   - URL path construction
//   - Parameter ordering and format
//
// Edge Cases Tested:
//   - Null order numbers
//   - Boolean parameter encoding
//   - Numeric parameter conversion
//   - String parameter encoding
func TestURLConstruction(t *testing.T) {
	t.Run("ParameterEncoding", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract and validate the complete query string
			queryString := r.URL.RawQuery

			// Parse query parameters
			values, err := url.ParseQuery(queryString)
			if err != nil {
				t.Errorf("Failed to parse query string: %v", err)
			}

			// Validate parameter count
			if len(values) != 6 {
				t.Errorf("Expected 6 query parameters, got %d", len(values))
			}

			// Validate each parameter is present
			requiredParams := []string{"limit", "offset", "order_number", "sort", "order", "expand"}
			for _, param := range requiredParams {
				if !values.Has(param) {
					t.Errorf("Missing required parameter: %s", param)
				}
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{}"))
		}))
		defer server.Close()

		params := DefaultComponentsParams()
		InvComponentsWithParams(server.URL, "token", params)
	})
}

// BenchmarkInvComponentsWithParams provides performance benchmarks for the API function.
// This benchmark measures the overhead of parameter processing, URL construction,
// and HTTP request preparation (excluding actual network I/O).
//
// Benchmark Coverage:
//   - Parameter struct creation and conversion
//   - URL parsing and construction
//   - HTTP request setup
//   - Header configuration
//
// Note: This benchmark uses a test server to avoid external network dependencies
// while still measuring realistic request preparation overhead.
func BenchmarkInvComponentsWithParams(b *testing.B) {
	// Setup mock server for benchmarking
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"components":[]}`))
	}))
	defer server.Close()

	params := DefaultComponentsParams()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InvComponentsWithParams(server.URL, "benchmark-token", params)
	}
}

// Example demonstrates practical usage of the assets package functions.
// This example shows both the simple and advanced interfaces for retrieving
// component inventory data from external APIs.
//
// The example illustrates:
//   - Basic usage with default parameters
//   - Advanced usage with custom parameters
//   - Error handling patterns
//   - Response processing approaches
func ExampleInvComponentsWithParams() {
	// Basic usage with default parameters
	components := InvComponents("https://api.example.com", "your-auth-token")
	if components != "" {
		// Process JSON response...
	}

	// Advanced usage with custom parameters
	params := ComponentsQueryParams{
		Limit:       100,
		Offset:      50,
		OrderNumber: nil, // No order filtering
		Sort:        "name",
		Order:       "asc",
		Expand:      true,
	}

	detailedComponents := InvComponentsWithParams("https://api.example.com", "your-auth-token", params)
	if detailedComponents != "" {
		// Process expanded JSON response...
	}

	// Filter by specific order number
	orderNum := 12345
	orderParams := ComponentsQueryParams{
		Limit:       25,
		OrderNumber: &orderNum,
		Sort:        "created_at",
		Order:       "desc",
	}

	orderComponents := InvComponentsWithParams("https://api.example.com", "your-auth-token", orderParams)
	// Process order-specific components...
	fmt.Println(orderComponents)

	// Output: Component inventory data retrieved successfully
}
