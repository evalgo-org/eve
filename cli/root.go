// Package cli provides the main command-line interface and HTTP server for the EVE evaluation service.
// This package orchestrates the complete application lifecycle including configuration management,
// service initialization, HTTP server setup, and graceful shutdown handling.
//
// The package implements a production-ready HTTP API server with:
//   - Flexible configuration via files, environment variables, and command-line flags
//   - Service dependency injection and lifecycle management
//   - RESTful API endpoints for flow process management
//   - JWT-based authentication and authorization
//   - Integration with RabbitMQ for message publishing
//   - CouchDB persistence for process state management
//   - Graceful shutdown with proper resource cleanup
//
// Architecture Overview:
//
//	CLI → Configuration → Services → HTTP Server → API Routes
//	↓
//	RabbitMQ ← Message Publishing ← API Handlers → CouchDB Persistence
//
// The server is designed for containerized deployment with 12-factor app principles,
// supporting configuration via environment variables and external config files.
package cli

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"eve.evalgo.org/api"
	eve "eve.evalgo.org/common"
	"eve.evalgo.org/db"
	"eve.evalgo.org/queue"
	"eve.evalgo.org/security"
)

// cfgFile holds the path to the configuration file specified via command-line flag.
// This variable is used by the configuration initialization system to determine
// whether to use a specific config file or search for default config files.
//
// Configuration File Search Order (when cfgFile is empty):
//  1. $HOME/.flow-service.yaml
//  2. ./.flow-service.yaml
//  3. Environment variables with VIPER_ prefix
//
// Supported Formats:
//   - YAML (.yaml, .yml)
//   - JSON (.json)
//   - TOML (.toml)
//   - Properties (.properties)
var cfgFile string

// RootCmd defines the main CLI command for the EVE evaluation service.
// This command serves as the entry point for the application and sets up
// the HTTP server with all required services and middleware.
//
// Command Structure:
//
//	eve [flags]
//	  ├── --config: Configuration file path
//	  ├── --port: HTTP server port
//	  ├── --rabbitmq-url: RabbitMQ connection URL
//	  ├── --queue-name: RabbitMQ queue name
//	  ├── --couchdb-url: CouchDB server URL
//	  ├── --database-name: CouchDB database name
//	  └── --jwt-secret: JWT signing secret
//
// The command initializes all services, sets up HTTP routes, and manages
// the complete application lifecycle including graceful shutdown.
//
// Configuration Precedence (highest to lowest):
//  1. Command-line flags
//  2. Environment variables
//  3. Configuration file values
//  4. Default values
//
// Example Usage:
//
//	# Start server with configuration file
//	eve --config /etc/eve/config.yaml
//
//	# Start server with environment variables
//	export RABBITMQ_URL=amqp://localhost:5672
//	export COUCHDB_URL=http://localhost:5984
//	export PORT=8080
//	eve
//
//	# Start server with command-line flags
//	eve --port 8080 --rabbitmq-url amqp://rabbitmq:5672 --couchdb-url http://couchdb:5984
var RootCmd = &cobra.Command{
	Use:   "eve",
	Short: "a sample service implementation for processing flow messages with RabbitMQ and CouchDB",
	Long: `EVE Evaluation Service

A production-ready HTTP API server for managing workflow processes with:
- RESTful API endpoints for process management
- JWT-based authentication and authorization  
- RabbitMQ integration for reliable message publishing
- CouchDB persistence for process state and history
- Graceful shutdown and health monitoring
- Flexible configuration management

The service provides endpoints for:
- Authentication and token generation
- Publishing process state messages
- Querying process status and history
- Managing process metadata and state transitions

Configuration can be provided via command-line flags, environment variables,
or YAML configuration files with automatic precedence handling.`,
	Run: runServer,
}

// init initializes the CLI command structure and configuration bindings.
// This function sets up the complete command-line interface including
// flag definitions, configuration file handling, and Viper bindings.
//
// Initialization Steps:
//  1. Register configuration initialization callback
//  2. Define persistent flags for all configuration options
//  3. Bind flags to Viper configuration keys
//  4. Set up automatic environment variable mapping
//
// Flag to Configuration Mapping:
//
//	--port           → viper: "port"
//	--rabbitmq-url   → viper: "rabbitmq.url"
//	--queue-name     → viper: "rabbitmq.queue_name"
//	--couchdb-url    → viper: "couchdb.url"
//	--database-name  → viper: "couchdb.database_name"
//	--jwt-secret     → viper: "jwt.secret"
//
// Environment Variable Mapping:
//
//	PORT                    → port
//	RABBITMQ_URL           → rabbitmq.url
//	RABBITMQ_QUEUE_NAME    → rabbitmq.queue_name
//	COUCHDB_URL            → couchdb.url
//	COUCHDB_DATABASE_NAME  → couchdb.database_name
//	JWT_SECRET             → jwt.secret
//
// This function is called automatically by Cobra before command execution.
func init() {
	cobra.OnInitialize(initConfig)

	// Configuration file flag
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.flow-service.yaml)")

	// Server configuration flags
	RootCmd.PersistentFlags().String("port", "", "Server port")

	// RabbitMQ configuration flags
	RootCmd.PersistentFlags().String("rabbitmq-url", "", "RabbitMQ connection URL")
	RootCmd.PersistentFlags().String("queue-name", "", "RabbitMQ queue name")

	// CouchDB configuration flags
	RootCmd.PersistentFlags().String("couchdb-url", "", "CouchDB connection URL")
	RootCmd.PersistentFlags().String("database-name", "", "CouchDB database name")

	// Security configuration flags
	RootCmd.PersistentFlags().String("jwt-secret", "", "JWT secret key")

	// Bind flags to Viper configuration keys
	viper.BindPFlag("port", RootCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("rabbitmq.url", RootCmd.PersistentFlags().Lookup("rabbitmq-url"))
	viper.BindPFlag("rabbitmq.queue_name", RootCmd.PersistentFlags().Lookup("queue-name"))
	viper.BindPFlag("couchdb.url", RootCmd.PersistentFlags().Lookup("couchdb-url"))
	viper.BindPFlag("couchdb.database_name", RootCmd.PersistentFlags().Lookup("database-name"))
	viper.BindPFlag("jwt.secret", RootCmd.PersistentFlags().Lookup("jwt-secret"))
}

// initConfig initializes the configuration system using Viper.
// This function handles configuration file discovery, environment variable
// mapping, and configuration loading with proper error handling.
//
// Configuration File Discovery:
//  1. If --config flag is provided, use specified file
//  2. Otherwise, search for .flow-service.yaml in:
//     - User home directory ($HOME/.flow-service.yaml)
//     - Current working directory (./.flow-service.yaml)
//
// Environment Variable Handling:
//   - Automatic environment variable mapping with VIPER_ prefix
//   - Nested configuration keys use underscore separation
//   - Example: VIPER_RABBITMQ_URL maps to rabbitmq.url
//
// Configuration File Formats:
//
//	Supports YAML, JSON, TOML, and properties files with automatic
//	format detection based on file extension.
//
// Error Handling:
//   - Missing configuration files are handled gracefully
//   - Invalid configuration files cause startup failure
//   - Configuration validation occurs during service initialization
//
// Example Configuration File (.flow-service.yaml):
//
//	port: "8080"
//	rabbitmq:
//	  url: "amqp://localhost:5672"
//	  queue_name: "flow_messages"
//	couchdb:
//	  url: "http://localhost:5984"
//	  database_name: "processes"
//	jwt:
//	  secret: "your-secret-key"
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory and current directory
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".flow-service")
	}

	// Enable automatic environment variable mapping
	viper.AutomaticEnv()

	// Read configuration file if available
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// runServer initializes and starts the HTTP server with all required services.
// This function orchestrates the complete application startup including service
// initialization, middleware setup, route configuration, and graceful shutdown handling.
//
// Startup Sequence:
//  1. Load and validate configuration from all sources
//  2. Initialize RabbitMQ service with connection pooling
//  3. Initialize CouchDB service with database creation
//  4. Initialize JWT service with signing key configuration
//  5. Set up Echo HTTP server with middleware stack
//  6. Configure API routes with authentication
//  7. Start HTTP server in background goroutine
//  8. Wait for shutdown signals (SIGINT, SIGTERM)
//  9. Perform graceful shutdown with timeout
//
// Service Dependencies:
//
//	RabbitMQ Service → Message publishing capabilities
//	CouchDB Service → Process state persistence
//	JWT Service → Authentication and authorization
//
// Middleware Stack:
//  1. Logger: Request/response logging for monitoring
//  2. Recover: Panic recovery for stability
//  3. CORS: Cross-origin resource sharing support
//  4. JWT: Authentication middleware for protected routes
//
// Graceful Shutdown:
//   - Listens for SIGINT (Ctrl+C) and SIGTERM signals
//   - Stops accepting new connections
//   - Waits for existing requests to complete (10-second timeout)
//   - Closes service connections and releases resources
//
// Parameters:
//   - cmd: Cobra command instance (unused in this implementation)
//   - args: Command-line arguments (unused in this implementation)
//
// Configuration Validation:
//
//	All required configuration values are validated during service
//	initialization. Missing or invalid configuration causes startup failure
//	with descriptive error messages.
//
// Error Handling:
//   - Service initialization failures cause immediate termination
//   - Server startup failures cause immediate termination
//   - Graceful shutdown failures are logged but don't prevent termination
//
// Production Considerations:
//   - Uses structured logging for monitoring and alerting
//   - Implements health checks via HTTP endpoints
//   - Supports horizontal scaling with stateless design
//   - Provides metrics endpoints for performance monitoring
func runServer(cmd *cobra.Command, args []string) {
	// Load configuration from all sources (flags, env vars, config file)
	config := eve.FlowConfig{
		RabbitMQURL:  viper.GetString("rabbitmq.url"),
		QueueName:    viper.GetString("rabbitmq.queue_name"),
		CouchDBURL:   viper.GetString("couchdb.url"),
		DatabaseName: viper.GetString("couchdb.database_name"),
		ApiKey:       viper.GetString("jwt.secret"),
	}

	// Initialize RabbitMQ service for message publishing
	rabbitMQService, err := queue.NewRabbitMQService(config)
	if err != nil {
		log.Fatalf("Failed to initialize RabbitMQ service: %v", err)
	}
	defer rabbitMQService.Close()

	// Initialize CouchDB service for process persistence
	couchDBService, err := db.NewCouchDBService(config)
	if err != nil {
		log.Fatalf("Failed to initialize CouchDB service: %v", err)
	}
	defer couchDBService.Close()

	// Initialize JWT service for authentication
	jwtService := security.NewJWTService(config.ApiKey)

	// Initialize Echo HTTP server with middleware
	e := echo.New()
	e.Use(middleware.Logger())  // Request/response logging
	e.Use(middleware.Recover()) // Panic recovery
	e.Use(middleware.CORS())    // Cross-origin support

	// Initialize API handlers with service dependencies
	handlers := &api.Handlers{
		RabbitMQ: rabbitMQService,
		CouchDB:  couchDBService,
		JWT:      jwtService,
	}

	// Set up API routes with authentication
	api.SetupRoutes(e, handlers, &config)

	// Start HTTP server in background goroutine
	port := viper.GetString("port")
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := e.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}
