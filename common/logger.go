// Package common provides enhanced logging utilities for structured logging across EVE services.
// This file extends the base logging functionality with context-aware logging,
// structured field helpers, and service-specific logging patterns.
package common

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"eve.evalgo.org/version"
	"github.com/sirupsen/logrus"
)

// LogLevel represents standard logging levels
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// LoggerConfig contains configuration for creating a logger
type LoggerConfig struct {
	Level      LogLevel // Minimum log level
	Format     string   // "json" or "text"
	Service    string   // Service name for all logs
	Version    string   // Service version
	AddCaller  bool     // Add caller information
	TimeFormat string   // Time format for logs
}

// DefaultLoggerConfig returns a logger config with sensible defaults
func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Level:      LogLevelInfo,
		Format:     "text",
		Service:    "",
		Version:    "",
		AddCaller:  false,
		TimeFormat: time.RFC3339,
	}
}

// NewLogger creates a new configured logger instance
func NewLogger(config LoggerConfig) *logrus.Logger {
	logger := logrus.New()

	// Set log level
	switch config.Level {
	case LogLevelDebug:
		logger.SetLevel(logrus.DebugLevel)
	case LogLevelInfo:
		logger.SetLevel(logrus.InfoLevel)
	case LogLevelWarn:
		logger.SetLevel(logrus.WarnLevel)
	case LogLevelError:
		logger.SetLevel(logrus.ErrorLevel)
	case LogLevelFatal:
		logger.SetLevel(logrus.FatalLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	// Set format
	if config.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: config.TimeFormat,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: config.TimeFormat,
			FullTimestamp:   true,
		})
	}

	// Set caller reporting
	logger.SetReportCaller(config.AddCaller)

	// Set output splitter
	logger.SetOutput(&OutputSplitter{})

	return logger
}

// ContextLogger provides context-aware logging utilities
type ContextLogger struct {
	logger *logrus.Logger
	fields logrus.Fields
}

// NewContextLogger creates a new context-aware logger with base fields
func NewContextLogger(logger *logrus.Logger, fields map[string]interface{}) *ContextLogger {
	if logger == nil {
		logger = Logger
	}

	baseFields := make(logrus.Fields)
	for k, v := range fields {
		baseFields[k] = v
	}

	return &ContextLogger{
		logger: logger,
		fields: baseFields,
	}
}

// WithField adds a single field to the logger context
func (cl *ContextLogger) WithField(key string, value interface{}) *ContextLogger {
	newFields := make(logrus.Fields)
	for k, v := range cl.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &ContextLogger{
		logger: cl.logger,
		fields: newFields,
	}
}

// WithFields adds multiple fields to the logger context
func (cl *ContextLogger) WithFields(fields map[string]interface{}) *ContextLogger {
	newFields := make(logrus.Fields)
	for k, v := range cl.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &ContextLogger{
		logger: cl.logger,
		fields: newFields,
	}
}

// WithError adds an error to the logger context
func (cl *ContextLogger) WithError(err error) *ContextLogger {
	return cl.WithField("error", err.Error())
}

// WithContext extracts request/trace IDs from context
func (cl *ContextLogger) WithContext(ctx context.Context) *ContextLogger {
	newFields := make(logrus.Fields)
	for k, v := range cl.fields {
		newFields[k] = v
	}

	// Extract common context values if present
	if requestID := ctx.Value("request_id"); requestID != nil {
		newFields["request_id"] = requestID
	}
	if traceID := ctx.Value("trace_id"); traceID != nil {
		newFields["trace_id"] = traceID
	}
	if userID := ctx.Value("user_id"); userID != nil {
		newFields["user_id"] = userID
	}

	return &ContextLogger{
		logger: cl.logger,
		fields: newFields,
	}
}

// Debug logs a debug message
func (cl *ContextLogger) Debug(msg string) {
	cl.logger.WithFields(cl.fields).Debug(msg)
}

// Debugf logs a formatted debug message
func (cl *ContextLogger) Debugf(format string, args ...interface{}) {
	cl.logger.WithFields(cl.fields).Debugf(format, args...)
}

// Info logs an info message
func (cl *ContextLogger) Info(msg string) {
	cl.logger.WithFields(cl.fields).Info(msg)
}

// Infof logs a formatted info message
func (cl *ContextLogger) Infof(format string, args ...interface{}) {
	cl.logger.WithFields(cl.fields).Infof(format, args...)
}

// Warn logs a warning message
func (cl *ContextLogger) Warn(msg string) {
	cl.logger.WithFields(cl.fields).Warn(msg)
}

// Warnf logs a formatted warning message
func (cl *ContextLogger) Warnf(format string, args ...interface{}) {
	cl.logger.WithFields(cl.fields).Warnf(format, args...)
}

// Error logs an error message
func (cl *ContextLogger) Error(msg string) {
	cl.logger.WithFields(cl.fields).Error(msg)
}

// Errorf logs a formatted error message
func (cl *ContextLogger) Errorf(format string, args ...interface{}) {
	cl.logger.WithFields(cl.fields).Errorf(format, args...)
}

// Fatal logs a fatal message and exits
func (cl *ContextLogger) Fatal(msg string) {
	cl.logger.WithFields(cl.fields).Fatal(msg)
}

// Fatalf logs a formatted fatal message and exits
func (cl *ContextLogger) Fatalf(format string, args ...interface{}) {
	cl.logger.WithFields(cl.fields).Fatalf(format, args...)
}

// ServiceLogger creates a logger pre-configured with service metadata
// Automatically includes the EVE framework version for debugging purposes
func ServiceLogger(serviceName, serviceVersion string) *ContextLogger {
	eveVersion := version.GetEVEVersion()
	return NewContextLogger(Logger, map[string]interface{}{
		"service":     serviceName,
		"version":     serviceVersion,
		"eve_version": eveVersion,
	})
}

// RequestLogger creates a logger for HTTP request tracking
func RequestLogger(serviceName, method, path, requestID string) *ContextLogger {
	return NewContextLogger(Logger, map[string]interface{}{
		"service":    serviceName,
		"method":     method,
		"path":       path,
		"request_id": requestID,
	})
}

// LogOperation logs the start and end of an operation with timing
func LogOperation(logger *ContextLogger, operation string, fn func() error) error {
	start := time.Now()
	logger.WithField("operation", operation).Info("Operation started")

	err := fn()

	duration := time.Since(start)
	logEntry := logger.WithFields(map[string]interface{}{
		"operation":   operation,
		"duration":    duration.String(),
		"duration_ms": duration.Milliseconds(),
	})

	if err != nil {
		logEntry.WithError(err).Error("Operation failed")
		return err
	}

	logEntry.Info("Operation completed")
	return nil
}

// LogDuration logs the duration of an operation
func LogDuration(logger *ContextLogger, operation string) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start)
		logger.WithFields(map[string]interface{}{
			"operation":   operation,
			"duration":    duration.String(),
			"duration_ms": duration.Milliseconds(),
		}).Info("Operation completed")
	}
}

// LogPanic recovers from panics and logs them
func LogPanic(logger *ContextLogger) {
	if r := recover(); r != nil {
		// Get stack trace
		buf := make([]byte, 4096)
		n := runtime.Stack(buf, false)
		stackTrace := string(buf[:n])

		logger.WithFields(map[string]interface{}{
			"panic":      fmt.Sprintf("%v", r),
			"stacktrace": stackTrace,
		}).Error("Panic recovered")
	}
}

// HTTPFields returns standard fields for HTTP logging
func HTTPFields(method, path string, statusCode int, duration time.Duration) map[string]interface{} {
	return map[string]interface{}{
		"http_method":      method,
		"http_path":        path,
		"http_status_code": statusCode,
		"duration":         duration.String(),
		"duration_ms":      duration.Milliseconds(),
	}
}

// DatabaseFields returns standard fields for database operation logging
func DatabaseFields(operation, table string, rowsAffected int64, duration time.Duration) map[string]interface{} {
	return map[string]interface{}{
		"db_operation":  operation,
		"db_table":      table,
		"rows_affected": rowsAffected,
		"duration":      duration.String(),
		"duration_ms":   duration.Milliseconds(),
	}
}

// ErrorFields returns standard fields for error logging
func ErrorFields(err error, context string) map[string]interface{} {
	fields := map[string]interface{}{
		"error":   err.Error(),
		"context": context,
	}

	// Add error type if available
	fields["error_type"] = fmt.Sprintf("%T", err)

	return fields
}

// StructuredLog provides a builder pattern for structured logging
type StructuredLog struct {
	logger *logrus.Logger
	fields logrus.Fields
	level  logrus.Level
}

// NewStructuredLog creates a new structured log builder
func NewStructuredLog(logger *logrus.Logger) *StructuredLog {
	if logger == nil {
		logger = Logger
	}
	return &StructuredLog{
		logger: logger,
		fields: make(logrus.Fields),
		level:  logrus.InfoLevel,
	}
}

// WithField adds a field to the structured log
func (sl *StructuredLog) WithField(key string, value interface{}) *StructuredLog {
	sl.fields[key] = value
	return sl
}

// WithFields adds multiple fields to the structured log
func (sl *StructuredLog) WithFields(fields map[string]interface{}) *StructuredLog {
	for k, v := range fields {
		sl.fields[k] = v
	}
	return sl
}

// WithError adds an error to the structured log
func (sl *StructuredLog) WithError(err error) *StructuredLog {
	sl.fields["error"] = err.Error()
	sl.fields["error_type"] = fmt.Sprintf("%T", err)
	return sl
}

// Level sets the log level
func (sl *StructuredLog) Level(level LogLevel) *StructuredLog {
	switch level {
	case LogLevelDebug:
		sl.level = logrus.DebugLevel
	case LogLevelInfo:
		sl.level = logrus.InfoLevel
	case LogLevelWarn:
		sl.level = logrus.WarnLevel
	case LogLevelError:
		sl.level = logrus.ErrorLevel
	case LogLevelFatal:
		sl.level = logrus.FatalLevel
	}
	return sl
}

// Log outputs the structured log
func (sl *StructuredLog) Log(msg string) {
	sl.logger.WithFields(sl.fields).Log(sl.level, msg)
}

// Logf outputs a formatted structured log
func (sl *StructuredLog) Logf(format string, args ...interface{}) {
	sl.logger.WithFields(sl.fields).Logf(sl.level, format, args...)
}
