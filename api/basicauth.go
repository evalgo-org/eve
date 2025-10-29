// Package api provides HTTP authentication middleware for the EVE service.
// This file implements HTTP Basic Authentication with bcrypt password verification.
package api

import (
	"encoding/base64"
	"net/http"
	"strings"

	"eve.evalgo.org/security"
	"github.com/labstack/echo/v4"
)

// BasicAuthConfig contains configuration for Basic Authentication middleware.
type BasicAuthConfig struct {
	// Username is the expected username for authentication
	Username string

	// Password is the plaintext password (will be compared with PasswordHash via bcrypt)
	// Either Password or PasswordHash must be set (PasswordHash is preferred for security)
	Password string

	// PasswordHash is the bcrypt hash of the password
	// If set, this takes precedence over Password field
	PasswordHash string

	// Realm is the authentication realm shown in the browser's login prompt
	// Default: "Restricted"
	Realm string

	// Skipper defines a function to skip middleware for specific requests
	// Default: nil (all requests require authentication)
	Skipper func(c echo.Context) bool

	// Validator is a custom validation function for username/password
	// If set, this overrides the default bcrypt validation
	// Return true if credentials are valid, false otherwise
	Validator func(username, password string, c echo.Context) bool
}

// BasicAuthMiddleware returns an Echo middleware that enforces HTTP Basic Authentication.
// It validates credentials using bcrypt password hashing for security.
//
// The middleware:
//  1. Checks if the request should be skipped (via Skipper function)
//  2. Extracts credentials from the Authorization header
//  3. Validates username and password against configured values
//  4. Returns 401 Unauthorized with WWW-Authenticate header if authentication fails
//
// Configuration options:
//   - Username: Required - the expected username
//   - Password: Plaintext password (less secure, use PasswordHash instead)
//   - PasswordHash: Bcrypt hash of password (recommended for production)
//   - Realm: Authentication realm (default: "Restricted")
//   - Skipper: Function to skip auth for specific requests (optional)
//   - Validator: Custom validation function (optional)
//
// Parameters:
//   - config: Basic authentication configuration
//
// Returns:
//   - echo.MiddlewareFunc: Configured middleware function
//
// Example:
//
//	// Using plaintext password (development only)
//	e := echo.New()
//	e.Use(BasicAuthMiddleware(BasicAuthConfig{
//	    Username: "admin",
//	    Password: "secret123",
//	    Realm:    "Admin Area",
//	}))
//
//	// Using bcrypt hash (recommended for production)
//	passwordHash, _ := security.HashPassword("secret123")
//	e.Use(BasicAuthMiddleware(BasicAuthConfig{
//	    Username:     "admin",
//	    PasswordHash: passwordHash,
//	    Realm:        "Admin Area",
//	}))
//
//	// Skip authentication for health checks
//	e.Use(BasicAuthMiddleware(BasicAuthConfig{
//	    Username:     "admin",
//	    PasswordHash: passwordHash,
//	    Skipper: func(c echo.Context) bool {
//	        return c.Path() == "/health"
//	    },
//	}))
//
//	// Custom validator
//	e.Use(BasicAuthMiddleware(BasicAuthConfig{
//	    Validator: func(username, password string, c echo.Context) bool {
//	        // Check against database or external service
//	        return validateAgainstDB(username, password)
//	    },
//	}))
func BasicAuthMiddleware(config BasicAuthConfig) echo.MiddlewareFunc {
	// Set default realm if not specified
	if config.Realm == "" {
		config.Realm = "Restricted"
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip authentication if Skipper returns true
			if config.Skipper != nil && config.Skipper(c) {
				return next(c)
			}

			// Extract Authorization header
			auth := c.Request().Header.Get("Authorization")
			if auth == "" {
				return unauthorized(c, config.Realm)
			}

			// Parse Basic Auth header
			username, password, err := parseBasicAuth(auth)
			if err != nil {
				return unauthorized(c, config.Realm)
			}

			// Validate credentials
			var valid bool
			if config.Validator != nil {
				// Use custom validator
				valid = config.Validator(username, password, c)
			} else {
				// Use built-in validation
				valid = validateCredentials(username, password, config)
			}

			if !valid {
				return unauthorized(c, config.Realm)
			}

			// Store username in context for use in handlers
			c.Set("username", username)

			// Authentication successful
			return next(c)
		}
	}
}

// parseBasicAuth extracts username and password from a Basic Auth header.
// Expects format: "Basic <base64-encoded-credentials>"
func parseBasicAuth(auth string) (username, password string, err error) {
	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return "", "", echo.NewHTTPError(http.StatusUnauthorized, "Invalid authorization header format")
	}

	// Decode base64 credentials
	encoded := auth[len(prefix):]
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", echo.NewHTTPError(http.StatusUnauthorized, "Invalid base64 encoding")
	}

	// Split into username:password
	credentials := string(decoded)
	parts := strings.SplitN(credentials, ":", 2)
	if len(parts) != 2 {
		return "", "", echo.NewHTTPError(http.StatusUnauthorized, "Invalid credentials format")
	}

	return parts[0], parts[1], nil
}

// validateCredentials checks username and password against configured values.
// Supports both plaintext password and bcrypt hash verification.
func validateCredentials(username, password string, config BasicAuthConfig) bool {
	// Check username
	if username != config.Username {
		return false
	}

	// Validate password
	if config.PasswordHash != "" {
		// Use bcrypt verification (recommended)
		err := security.VerifyPassword(config.PasswordHash, password)
		return err == nil
	} else if config.Password != "" {
		// Use plaintext comparison (not recommended for production)
		return password == config.Password
	}

	// No password configured
	return false
}

// unauthorized returns a 401 Unauthorized response with WWW-Authenticate header.
// This prompts browsers to show a login dialog.
func unauthorized(c echo.Context, realm string) error {
	c.Response().Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
	return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
}

// GetBasicAuthUsername retrieves the authenticated username from the Echo context.
// Returns empty string if not authenticated via Basic Auth.
//
// Parameters:
//   - c: Echo context
//
// Returns:
//   - string: The authenticated username, or empty string if not available
//
// Example:
//
//	func handler(c echo.Context) error {
//	    username := GetBasicAuthUsername(c)
//	    if username != "" {
//	        return c.String(200, "Hello, "+username)
//	    }
//	    return c.String(200, "Hello, guest")
//	}
func GetBasicAuthUsername(c echo.Context) string {
	username, ok := c.Get("username").(string)
	if !ok {
		return ""
	}
	return username
}
