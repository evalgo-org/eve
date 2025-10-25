// Package api provides HTTP middleware and server utilities for API key authentication.
// It includes middleware for validating API keys and a convenience function for starting
// a server with API key protection.
package api

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

// APIKeyAuth creates an Echo middleware function that validates API keys from request headers.
// The middleware checks for the presence and validity of an API key in the "X-API-Key" header.
// If the key is missing or doesn't match the provided valid key, it returns an HTTP 401 Unauthorized error.
//
// This middleware should be applied to routes that require API key authentication.
// It follows the Echo middleware pattern and can be used with e.Use() or on specific route groups.
//
// Parameters:
//   - validKey: The expected API key that clients must provide for authentication
//
// Returns:
//   - echo.MiddlewareFunc: Middleware function that can be used with Echo router
//
// Usage:
//
//	e := echo.New()
//	e.Use(APIKeyAuth("your-secret-api-key"))
//
// HTTP Headers:
//   - X-API-Key: Required header containing the API key for authentication
//
// Error Responses:
//   - 401 Unauthorized: Returned when API key is missing or invalid
//     Response body: {"message": "invalid or missing API key"}
func APIKeyAuth(validKey string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := c.Request().Header.Get("X-API-Key")
			if key == "" || key != validKey {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or missing API key")
			}
			return next(c)
		}
	}
}

// StartWithApiKey creates and starts an Echo HTTP server with API key authentication middleware.
// This is a convenience function that sets up a basic server with a health check endpoint,
// protected by API key authentication. The server will terminate the program if it fails to start.
//
// The function automatically applies the API key authentication middleware to all routes,
// meaning every request must include a valid "X-API-Key" header.
//
// Default routes:
//   - GET /: Health check endpoint that returns "OK!" when API key is valid
//
// Parameters:
//   - address: The network address and port to bind the server to (e.g., ":8080", "localhost:3000")
//   - apiKey: The API key that clients must provide in the "X-API-Key" header
//
// Behavior:
//   - Creates a new Echo instance
//   - Applies API key authentication middleware globally
//   - Registers a simple health check endpoint at "/"
//   - Starts the server and terminates the program on failure
//
// Usage:
//
//	StartWithApiKey(":8080", "your-secret-api-key")
//
// Example request:
//
//	curl -H "X-API-Key: your-secret-api-key" http://localhost:8080/
//
// Note: This function calls e.Logger.Fatal() which will terminate the program
// if the server fails to start. For production use, consider handling errors
// more gracefully.
func StartWithApiKey(address, apiKey string) {
	e := echo.New()
	e.Use(APIKeyAuth(apiKey))
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK!")
	})
	e.Logger.Fatal(e.Start(address))
}
