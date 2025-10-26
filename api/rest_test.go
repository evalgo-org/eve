// Package api provides comprehensive testing utilities and examples for HTTP API handlers.
// This file demonstrates testing patterns for Echo framework handlers using mock data,
// HTTP test utilities, and assertion libraries. It includes examples of both JSON and
// form-based request testing.
package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// mockDB represents an in-memory database for testing purposes.
// It simulates a user storage system using email addresses as keys.
// This mock data is used across multiple test cases to ensure consistent
// test behavior and avoid external database dependencies during testing.
var (
	mockDB = map[string]*User{
		"jon@labstack.com": &User{"Jon Snow", "jon@labstack.com"},
	}
	// userJSON contains the expected JSON representation of the mock user.
	// This constant is used for response validation in tests to ensure
	// the API returns properly formatted JSON data.
	userJSON = `{"name":"Jon Snow","email":"jon@labstack.com"}`
)

// User represents a user entity in the system with basic profile information.
// It supports both JSON and form-based serialization through struct tags,
// making it suitable for testing different content types and request formats.
//
// The struct uses Echo's binding tags to support:
//   - JSON requests/responses via the `json` tag
//   - HTML form submissions via the `form` tag
type User struct {
	Name  string `json:"name" form:"name"`   // User's display name
	Email string `json:"email" form:"email"` // User's email address (used as unique identifier)
}

// handler provides HTTP request handlers for user management operations.
// It contains a mock database for testing purposes, allowing tests to run
// without external dependencies. In production, this would typically contain
// references to actual database connections or service layers.
//
// The handler implements common CRUD operations and serves as an example
// of how to structure API handlers for testing.
type handler struct {
	db map[string]*User // Mock database mapping email addresses to User objects
}

// createUser handles HTTP POST requests to create new users in the system.
// It demonstrates proper request binding, error handling, and JSON response
// generation patterns commonly used in REST API handlers.
//
// HTTP Method: POST
// Content-Type: Supports both application/json and application/x-www-form-urlencoded
//
// Request Body (JSON):
//
//	{
//	  "name": "string",    // User's display name
//	  "email": "string"    // User's email address
//	}
//
// Request Body (Form):
//
//	name=value&email=value
//
// Response:
//
//	Success (201): Returns the created user as JSON
//	Bad Request (400): Returned when request binding fails
//
// Example usage in tests:
//
//	h := &handler{mockDB}
//	c := createTestContext(userPayload)
//	err := h.createUser(c)
func (h *handler) createUser(c echo.Context) error {
	u := new(User)
	if err := c.Bind(u); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, u)
}

// getUser handles HTTP GET requests to retrieve user information by email.
// It demonstrates parameter extraction, database lookup, error handling,
// and conditional response patterns used in REST API handlers.
//
// HTTP Method: GET
// Path: /users/:email (email parameter extracted from URL path)
//
// Path Parameters:
//   - email: User's email address used as unique identifier
//
// Response:
//
//	Success (200): Returns the user object as JSON
//	Not Found (404): {"message": "user not found"} when user doesn't exist
//
// Example usage in tests:
//
//	h := &handler{mockDB}
//	c := createTestContextWithParam("email", "jon@labstack.com")
//	err := h.getUser(c)
func (h *handler) getUser(c echo.Context) error {
	email := c.Param("email")
	user := h.db[email]
	if user == nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}
	return c.JSON(http.StatusOK, user)
}

// TestApi demonstrates comprehensive testing patterns for Echo HTTP handlers.
// This test showcases several important testing concepts:
//
// 1. **Mock Data Usage**: Uses predefined mock database and expected responses
// 2. **Multiple Content Types**: Shows both JSON and form-based request testing
// 3. **HTTP Test Utilities**: Demonstrates httptest.NewRequest and httptest.NewRecorder
// 4. **Context Creation**: Shows how to create Echo contexts for isolated testing
// 5. **Assertion Patterns**: Uses testify/assert for readable test assertions
//
// Test Structure:
//   - Sets up Echo instance and test data
//   - Creates HTTP request with form-encoded payload
//   - Generates Echo context from request/response recorder
//   - Executes handler function
//   - Validates response status and body content
//
// The test includes commented code showing alternative JSON request testing,
// demonstrating how to test the same handler with different content types.
//
// Key Testing Patterns Demonstrated:
//   - Handler isolation using dependency injection
//   - Request payload preparation (both form and JSON examples)
//   - Response validation using assertions
//   - Error-free execution verification
//
// Usage:
//
//	go test -v ./api
//	go test -run TestApi
func TestApi(t *testing.T) {
	// Initialize Echo instance for testing
	e := echo.New()

	// Prepare form-encoded request payload
	f := make(url.Values)
	f.Set("name", "Jon Snow")
	f.Set("email", "jon@labstack.com")

	// Alternative JSON request example (commented for reference):
	// req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(userJSON))
	// req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	// Create HTTP test request with form-encoded payload
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	// Create response recorder to capture handler output
	rec := httptest.NewRecorder()

	// Generate Echo context from request and recorder
	c := e.NewContext(req, rec)

	// Initialize handler with mock database
	h := &handler{mockDB}

	// Execute handler and validate results
	if assert.NoError(t, h.createUser(c)) {
		// Verify HTTP status code
		assert.Equal(t, http.StatusCreated, rec.Code)
		// Verify response body matches expected JSON (with trailing newline)
		assert.Equal(t, userJSON+"\n", rec.Body.String())
	}
}

// TestAPIKeyAuth_ValidKey tests middleware with valid API key
func TestAPIKeyAuth_ValidKey(t *testing.T) {
	e := echo.New()
	validKey := "test-api-key-123"

	// Create request with valid API key
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", validKey)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Create middleware
	middleware := APIKeyAuth(validKey)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "authorized")
	})

	// Execute handler
	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "authorized", rec.Body.String())
}

// TestAPIKeyAuth_InvalidKey tests middleware with invalid API key
func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	e := echo.New()
	validKey := "test-api-key-123"

	// Create request with invalid API key
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Create middleware
	middleware := APIKeyAuth(validKey)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "should not reach here")
	})

	// Execute handler
	err := handler(c)
	assert.Error(t, err)

	// Verify it's an HTTP error with status 401
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
	assert.Equal(t, "invalid or missing API key", httpErr.Message)
}

// TestAPIKeyAuth_MissingKey tests middleware with missing API key
func TestAPIKeyAuth_MissingKey(t *testing.T) {
	e := echo.New()
	validKey := "test-api-key-123"

	// Create request without API key
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Create middleware
	middleware := APIKeyAuth(validKey)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "should not reach here")
	})

	// Execute handler
	err := handler(c)
	assert.Error(t, err)

	// Verify it's an HTTP error with status 401
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
	assert.Equal(t, "invalid or missing API key", httpErr.Message)
}

// TestAPIKeyAuth_EmptyKey tests middleware with empty API key
func TestAPIKeyAuth_EmptyKey(t *testing.T) {
	e := echo.New()
	validKey := "test-api-key-123"

	// Create request with empty API key
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Create middleware
	middleware := APIKeyAuth(validKey)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "should not reach here")
	})

	// Execute handler
	err := handler(c)
	assert.Error(t, err)

	// Verify it's an HTTP error with status 401
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
}

// TestAPIKeyAuth_CaseSensitive tests that API keys are case-sensitive
func TestAPIKeyAuth_CaseSensitive(t *testing.T) {
	e := echo.New()
	validKey := "TestKey123"

	tests := []struct {
		name        string
		providedKey string
		expectAuth  bool
	}{
		{
			name:        "ExactMatch",
			providedKey: "TestKey123",
			expectAuth:  true,
		},
		{
			name:        "Lowercase",
			providedKey: "testkey123",
			expectAuth:  false,
		},
		{
			name:        "Uppercase",
			providedKey: "TESTKEY123",
			expectAuth:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("X-API-Key", tt.providedKey)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			middleware := APIKeyAuth(validKey)
			handler := middleware(func(c echo.Context) error {
				return c.String(http.StatusOK, "authorized")
			})

			err := handler(c)

			if tt.expectAuth {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Code)
			} else {
				assert.Error(t, err)
				httpErr, ok := err.(*echo.HTTPError)
				assert.True(t, ok)
				assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
			}
		})
	}
}

// TestAPIKeyAuth_MultipleRequests tests middleware with sequential requests
func TestAPIKeyAuth_MultipleRequests(t *testing.T) {
	e := echo.New()
	validKey := "my-secret-key"
	middleware := APIKeyAuth(validKey)

	requests := []struct {
		name     string
		apiKey   string
		expectOK bool
	}{
		{"FirstValid", validKey, true},
		{"Invalid", "wrong-key", false},
		{"SecondValid", validKey, true},
		{"Missing", "", false},
		{"ThirdValid", validKey, true},
	}

	for _, tt := range requests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := middleware(func(c echo.Context) error {
				return c.String(http.StatusOK, "OK")
			})

			err := handler(c)

			if tt.expectOK {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
