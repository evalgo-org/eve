// Package http provides common HTTP server utilities for EVE services.
// This package includes standard middleware, health checks, and server setup patterns
// used across the EVE ecosystem.
package http

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

// ServerConfig contains configuration for creating an Echo server
type ServerConfig struct {
	Port            int
	Debug           bool
	BodyLimit       string // e.g., "100M"
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	AllowedOrigins  []string // For CORS
	RateLimit       float64  // Requests per second (0 = no limit)
}

// DefaultServerConfig returns a server config with sensible defaults
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Port:            8080,
		Debug:           false,
		BodyLimit:       "10M",
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		ShutdownTimeout: 10 * time.Second,
		AllowedOrigins:  []string{"*"},
		RateLimit:       0, // No limit by default
	}
}

// NewEchoServer creates a new Echo server with standard middleware
func NewEchoServer(config ServerConfig) *echo.Echo {
	e := echo.New()

	// Configure Echo
	e.HideBanner = true
	e.HidePort = true
	e.Debug = config.Debug

	// Logger middleware with standard format
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "[${time_rfc3339}] ${status} ${method} ${uri} (${latency_human})\n",
	}))

	// Recover middleware for panic recovery
	e.Use(middleware.Recover())

	// Body limit middleware
	if config.BodyLimit != "" {
		e.Use(middleware.BodyLimit(config.BodyLimit))
	}

	// CORS middleware
	if len(config.AllowedOrigins) > 0 {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: config.AllowedOrigins,
			AllowMethods: []string{
				http.MethodGet,
				http.MethodPost,
				http.MethodPut,
				http.MethodDelete,
				http.MethodPatch,
				http.MethodOptions,
			},
			AllowHeaders: []string{
				echo.HeaderOrigin,
				echo.HeaderContentType,
				echo.HeaderAccept,
				echo.HeaderAuthorization,
				"X-API-Key",
			},
		}))
	}

	// Request ID middleware
	e.Use(middleware.RequestID())

	// Rate limiting (if enabled)
	if config.RateLimit > 0 {
		e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(
			rate.Limit(config.RateLimit),
		)))
	}

	return e
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status  string                 `json:"status"`
	Service string                 `json:"service,omitempty"`
	Version string                 `json:"version,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// HealthCheckHandler returns a standard health check handler
func HealthCheckHandler(serviceName, version string) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, HealthResponse{
			Status:  "healthy",
			Service: serviceName,
			Version: version,
		})
	}
}

// HealthCheckHandlerWithDetails returns a health check handler with custom details
func HealthCheckHandlerWithDetails(serviceName, version string, detailsFunc func() map[string]interface{}) echo.HandlerFunc {
	return func(c echo.Context) error {
		details := make(map[string]interface{})
		if detailsFunc != nil {
			details = detailsFunc()
		}

		return c.JSON(http.StatusOK, HealthResponse{
			Status:  "healthy",
			Service: serviceName,
			Version: version,
			Details: details,
		})
	}
}

// StartServer starts an Echo server with graceful shutdown support
func StartServer(e *echo.Echo, config ServerConfig) error {
	// Create HTTP server with timeouts
	s := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	}

	// Start server
	log.Printf("Starting server on port %d", config.Port)
	return e.StartServer(s)
}

// GracefulShutdown performs a graceful shutdown of the Echo server
func GracefulShutdown(e *echo.Echo, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Println("Shutting down server gracefully...")
	if err := e.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("Server stopped")
	return nil
}

// APIKeyMiddleware creates a middleware that validates API keys
func APIKeyMiddleware(apiKey string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip if no API key is configured
			if apiKey == "" {
				return next(c)
			}

			// Check x-api-key header
			key := c.Request().Header.Get("X-API-Key")
			if key == "" {
				key = c.Request().Header.Get("x-api-key")
			}

			if key == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing API key")
			}

			if key != apiKey {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid API key")
			}

			return next(c)
		}
	}
}

// SecurityHeadersMiddleware adds security headers to responses
func SecurityHeadersMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Add security headers
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			c.Response().Header().Set("X-Frame-Options", "DENY")
			c.Response().Header().Set("X-XSS-Protection", "1; mode=block")

			return next(c)
		}
	}
}

// JSONContentTypeMiddleware ensures JSON content type for API responses
func JSONContentTypeMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Set default content type for responses
			if c.Request().Method != http.MethodOptions {
				c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			}
			return next(c)
		}
	}
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// CustomHTTPErrorHandler provides a standard error handler for Echo
func CustomHTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	message := err.Error()

	// Check if it's an Echo HTTP error
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		if msg, ok := he.Message.(string); ok {
			message = msg
		}
	}

	// Don't send response if it's already committed
	if !c.Response().Committed {
		if c.Request().Method == http.MethodHead {
			err = c.NoContent(code)
		} else {
			err = c.JSON(code, ErrorResponse{
				Error:   http.StatusText(code),
				Message: message,
			})
		}
		if err != nil {
			log.Printf("Error sending error response: %v", err)
		}
	}
}

// GetPortInt parses a port from environment variable with a default fallback
func GetPortInt(envVar string, defaultPort int) int {
	portStr := ""
	if envVar != "" {
		portStr = envVar
	}

	if portStr == "" {
		return defaultPort
	}

	var port int
	_, err := fmt.Sscanf(portStr, "%d", &port)
	if err != nil || port <= 0 || port > 65535 {
		return defaultPort
	}

	return port
}
