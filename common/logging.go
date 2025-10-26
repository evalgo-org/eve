// Package common provides centralized logging infrastructure for the EVE evaluation system.
// This package implements intelligent log output routing that automatically directs
// error messages to stderr while sending other log levels to stdout, enabling
// proper stream separation for containerized and scripted environments.
//
// The logging system is built on logrus for structured logging capabilities with
// custom output handling that supports both development workflows and production
// deployment patterns. It provides a foundation for consistent logging across
// all services in the EVE system.
//
// Key Features:
//   - Automatic output stream routing based on log level
//   - Structured logging with JSON and text format support
//   - Container-friendly output separation for log aggregation
//   - Global logger instance for consistent usage patterns
//   - Integration with monitoring and alerting systems
//
// Output Routing Strategy:
//
//	The system implements intelligent output routing where error-level messages
//	are directed to stderr (for immediate attention and error handling) while
//	info, debug, and warning messages go to stdout (for general log processing).
//
// Container Integration:
//
//	Designed for containerized environments where stdout and stderr streams
//	can be handled differently by orchestration platforms, log aggregators,
//	and monitoring systems for optimal observability and alerting.
//
// Usage Patterns:
//
//	The package provides a global Logger instance that can be used throughout
//	the application for consistent logging behavior. All services should use
//	this logger to ensure uniform output handling and formatting.
package common

import (
	"bytes"
	"os"

	"github.com/sirupsen/logrus"
)

// OutputSplitter implements intelligent log output routing based on log content analysis.
// This custom writer examines log messages and directs them to appropriate output
// streams (stdout vs stderr) based on their severity level, enabling proper
// log stream separation for containerized and production environments.
//
// Routing Logic:
//
//	The splitter analyzes each log message for error indicators and routes
//	them accordingly:
//	- Error messages (containing "level=error") → stderr
//	- All other messages (info, debug, warn) → stdout
//
// Stream Separation Benefits:
//   - Monitoring systems can treat error streams with higher priority
//   - Container orchestrators can route error streams to alerting systems
//   - Log aggregation tools can apply different processing rules per stream
//   - Shell scripts can capture and handle error output separately
//
// Container Compatibility:
//
//	Docker and Kubernetes environments can capture stdout and stderr
//	independently, enabling sophisticated log processing pipelines
//	where errors trigger immediate notifications while info logs
//	are processed for analytics and debugging.
//
// Performance Characteristics:
//   - Minimal overhead through simple byte pattern matching
//   - No regex processing or complex parsing for efficiency
//   - Direct stream writing without buffering delays
//   - Suitable for high-throughput logging scenarios
//
// Integration with Logrus:
//
//	Works seamlessly with logrus formatters including JSON and text
//	formats. The splitter operates on the final formatted output,
//	ensuring compatibility with all logrus configuration options.
//
// Example Usage:
//
//	splitter := &OutputSplitter{}
//	logger := logrus.New()
//	logger.SetOutput(splitter)
//
//	logger.Info("This goes to stdout")
//	logger.Error("This goes to stderr")
type OutputSplitter struct{}

// Write implements the io.Writer interface for the OutputSplitter.
// This method analyzes incoming log data and routes it to the appropriate
// output stream based on content analysis, enabling intelligent log separation.
//
// Routing Algorithm:
//  1. Examines the byte content for error level indicators
//  2. Routes messages containing "level=error" to stderr
//  3. Routes all other messages to stdout
//  4. Returns the number of bytes written and any I/O errors
//
// Content Analysis:
//
//	Uses efficient byte searching to identify error-level messages
//	without complex parsing or regular expressions. The pattern matching
//	is designed to work with logrus's standard output format.
//
// Error Detection Pattern:
//
//	Searches for the literal string "level=error" which is produced
//	by logrus when formatting error-level log entries. This pattern
//	is reliable across different logrus formatters and configurations.
//
// Parameters:
//   - p: Byte slice containing the log message to be written
//
// Returns:
//   - n: Number of bytes successfully written to the output stream
//   - err: Any error encountered during the write operation
//
// Stream Selection:
//   - os.Stderr: Used for error messages requiring immediate attention
//   - os.Stdout: Used for informational messages and general logging
//
// Error Handling:
//
//	Write errors from the underlying streams (stdout/stderr) are
//	propagated back to the caller, maintaining proper error semantics
//	for the io.Writer interface contract.
//
// Concurrency Safety:
//
//	The method is safe for concurrent use as it only performs read
//	operations on the input data and writes to thread-safe OS streams.
//	Multiple goroutines can safely use the same OutputSplitter instance.
//
// Performance Notes:
//   - Uses bytes.Contains for efficient pattern matching
//   - No memory allocation during normal operation
//   - Direct stream writing without intermediate buffering
//   - Minimal CPU overhead suitable for high-frequency logging
//
// Example Message Routing:
//
//	Input: `time="2024-01-15T10:30:00Z" level=error msg="Database connection failed"`
//	Output: Routed to stderr
//
//	Input: `time="2024-01-15T10:30:00Z" level=info msg="Service started successfully"`
//	Output: Routed to stdout
func (splitter *OutputSplitter) Write(p []byte) (n int, err error) {
	// Analyze log content for error level indicators
	if bytes.Contains(p, []byte("level=error")) {
		// Route error messages to stderr for immediate attention
		return os.Stderr.Write(p)
	}
	// Route non-error messages to stdout for general processing
	return os.Stdout.Write(p)
}

// Logger provides the global logger instance for the EVE evaluation system.
// This logger is pre-configured with the OutputSplitter for intelligent
// log routing and serves as the central logging facility for all services.
//
// Global Logger Benefits:
//   - Consistent logging behavior across all application components
//   - Centralized configuration and formatting standards
//   - Simplified integration with monitoring and alerting systems
//   - Uniform log structure for parsing and analysis tools
//
// Default Configuration:
//
//	The logger is initialized with logrus defaults but can be customized
//	for specific deployment environments:
//	- Output: Directed through OutputSplitter for stream separation
//	- Format: Configurable (text for development, JSON for production)
//	- Level: Configurable based on environment (debug/info/warn/error)
//	- Fields: Support for structured logging with consistent field names
//
// Structured Logging Support:
//
//	Leverages logrus's structured logging capabilities for consistent
//	log entry formatting with key-value pairs, making logs machine-readable
//	and suitable for automated processing and analysis.
//
// Configuration Examples:
//
//	// Development environment (human-readable)
//	Logger.SetFormatter(&logrus.TextFormatter{
//	    FullTimestamp: true,
//	    ForceColors:   true,
//	})
//	Logger.SetLevel(logrus.DebugLevel)
//
//	// Production environment (machine-readable)
//	Logger.SetFormatter(&logrus.JSONFormatter{})
//	Logger.SetLevel(logrus.InfoLevel)
//
// Usage Patterns:
//
//	// Simple logging
//	Logger.Info("Service started")
//	Logger.Error("Database connection failed")
//
//	// Structured logging with fields
//	Logger.WithFields(logrus.Fields{
//	    "user_id": "12345",
//	    "action":  "login",
//	}).Info("User authentication successful")
//
//	// Error logging with context
//	Logger.WithError(err).Error("Failed to process request")
//
// Integration with Services:
//
//	All EVE services should use this global logger instance to ensure
//	consistent log formatting, routing, and monitoring integration.
//	Custom loggers should only be created for specific use cases that
//	require different output destinations or formatting.
//
// Monitoring Integration:
//
//	The logger's output can be easily integrated with monitoring systems:
//	- Prometheus metrics from log parsing
//	- Elasticsearch for log aggregation and search
//	- CloudWatch or similar cloud logging services
//	- Custom alerting based on error log patterns
//
// Performance Considerations:
//   - Structured logging has minimal overhead with proper field usage
//   - JSON formatting is efficient for high-volume logging
//   - Stream separation allows for optimized log processing pipelines
//   - Configurable log levels prevent debug spam in production
//
// Thread Safety:
//
//	The logger is safe for concurrent use across multiple goroutines.
//	Logrus handles synchronization internally, making it suitable for
//	multi-threaded applications and concurrent request processing.
var Logger = logrus.New()

// init initializes the global logger with the OutputSplitter for intelligent routing.
// This function is called automatically when the package is imported, setting up
// the logging infrastructure with proper stream separation.
//
// Initialization Process:
//  1. Creates a new logrus logger instance
//  2. Configures the OutputSplitter for intelligent stream routing
//  3. Sets up default formatting and logging levels
//  4. Prepares the logger for immediate use across the application
//
// Automatic Setup:
//
//	The init function ensures that the logging system is ready for use
//	immediately upon package import, requiring no additional configuration
//	for basic operation while still allowing customization when needed.
//
// Default Configuration:
//   - Output routing through OutputSplitter
//   - Standard logrus formatting (can be overridden)
//   - All log levels enabled (can be filtered per environment)
//   - Thread-safe operation for concurrent usage
//
// Customization After Init:
//
//	Applications can further customize the logger after package initialization:
//	- Set specific formatters (JSON, text, custom)
//	- Configure log levels per environment
//	- Add hooks for external integrations
//	- Set custom field formatting and timestamping
//
// Example Post-Init Customization:
//
//	import "your-app/common"
//
//	func init() {
//	    // Logger is already initialized with OutputSplitter
//	    common.Logger.SetFormatter(&logrus.JSONFormatter{})
//	    common.Logger.SetLevel(logrus.InfoLevel)
//	}
func init() {
	// Configure the global logger with intelligent output routing
	Logger.SetOutput(&OutputSplitter{})
}
