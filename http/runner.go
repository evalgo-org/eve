// Package http provides common HTTP server utilities for EVE services.
// This file contains the RunServer helper for standardized service management.
package http

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"

	"eve.evalgo.org/common"
	"eve.evalgo.org/registry"
)

// RunServerConfig contains configuration for running an EVE service
type RunServerConfig struct {
	// Service identification
	ServiceID   string
	ServiceName string
	Version     string
	Description string

	// Server configuration
	Port            int
	Debug           bool
	BodyLimit       string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	AllowedOrigins  []string
	RateLimit       float64

	// Registry configuration (optional)
	EnableRegistry bool
	Directory      string   // Service directory (for registry)
	Binary         string   // Binary name (for registry)
	Capabilities   []string // Service capabilities (for registry)

	// Logger (optional, will create one if nil)
	Logger *common.ContextLogger
}

// DefaultRunServerConfig returns a RunServerConfig with sensible defaults
func DefaultRunServerConfig(serviceID, serviceName, version string) RunServerConfig {
	return RunServerConfig{
		ServiceID:       serviceID,
		ServiceName:     serviceName,
		Version:         version,
		Description:     fmt.Sprintf("%s service", serviceName),
		Port:            8080,
		Debug:           false,
		BodyLimit:       "10M",
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		ShutdownTimeout: 10 * time.Second,
		AllowedOrigins:  []string{"*"},
		RateLimit:       0,
		EnableRegistry:  true,
		Capabilities:    []string{},
	}
}

// SetupFunc is a function that sets up routes and handlers on an Echo instance
type SetupFunc func(*echo.Echo) error

// RunServer creates and runs an Echo server with standard EVE patterns:
//   - Creates Echo instance with standard middleware
//   - Adds health check endpoint
//   - Registers with service registry (if enabled)
//   - Sets up signal handling for graceful shutdown
//   - Unregisters from service registry on shutdown
//
// Example usage:
//
//	cfg := http.DefaultRunServerConfig("myservice", "My Service", "1.0.0")
//	cfg.Port = 8090
//	cfg.Capabilities = []string{"storage", "query"}
//
//	err := http.RunServer(cfg, func(e *echo.Echo) error {
//	    e.POST("/api/action", handleAction)
//	    return nil
//	})
func RunServer(config RunServerConfig, setupFunc SetupFunc) error {
	// Create logger if not provided
	logger := config.Logger
	if logger == nil {
		logger = common.ServiceLogger(config.ServiceID, config.Version)
	}

	// Create server configuration
	serverConfig := ServerConfig{
		Port:            config.Port,
		Debug:           config.Debug,
		BodyLimit:       config.BodyLimit,
		ReadTimeout:     config.ReadTimeout,
		WriteTimeout:    config.WriteTimeout,
		ShutdownTimeout: config.ShutdownTimeout,
		AllowedOrigins:  config.AllowedOrigins,
		RateLimit:       config.RateLimit,
	}

	// Create Echo server with standard middleware
	e := NewEchoServer(serverConfig)

	// Add custom error handler
	e.HTTPErrorHandler = CustomHTTPErrorHandler

	// Add health check endpoint
	e.GET("/health", HealthCheckHandler(config.ServiceName, config.Version))

	// Call setup function to add routes
	if setupFunc != nil {
		if err := setupFunc(e); err != nil {
			return fmt.Errorf("setup function failed: %w", err)
		}
	}

	// Auto-register with service registry (if enabled)
	if config.EnableRegistry {
		if _, err := registry.AutoRegister(registry.AutoRegisterConfig{
			ServiceID:    config.ServiceID,
			ServiceName:  config.ServiceName,
			Description:  config.Description,
			Port:         config.Port,
			Directory:    config.Directory,
			Binary:       config.Binary,
			Capabilities: config.Capabilities,
		}); err != nil {
			logger.WithError(err).Warn("Failed to register with registry (continuing anyway)")
		}
	}

	// Start server in goroutine
	go func() {
		logger.Infof("Starting %s on port %d", config.ServiceName, config.Port)
		if err := e.Start(fmt.Sprintf(":%d", config.Port)); err != nil {
			logger.WithError(err).Error("Server error")
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Unregister from service registry
	if config.EnableRegistry {
		if err := registry.AutoUnregister(config.ServiceID); err != nil {
			logger.WithError(err).Error("Failed to unregister from registry")
		}
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.ShutdownTimeout)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("Error during shutdown")
		return err
	}

	logger.Info("Server stopped")
	return nil
}
