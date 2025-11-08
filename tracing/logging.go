// Package tracing - Log correlation with trace context
package tracing

import (
	"context"
	"io"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

// Logger wraps zerolog with automatic trace context injection
type Logger struct {
	log zerolog.Logger
}

// NewLogger creates a logger with trace correlation support
func NewLogger(writer io.Writer, serviceName string) *Logger {
	if writer == nil {
		writer = os.Stdout
	}

	// Configure zerolog for JSON structured logging
	log := zerolog.New(writer).With().
		Timestamp().
		Str("service", serviceName).
		Logger()

	return &Logger{log: log}
}

// NewConsoleLogger creates a human-readable console logger for development
func NewConsoleLogger(serviceName string) *Logger {
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}
	log := zerolog.New(consoleWriter).With().
		Timestamp().
		Str("service", serviceName).
		Logger()

	return &Logger{log: log}
}

// WithContext creates a logger with trace IDs from Echo context
func (l *Logger) WithContext(c echo.Context) *Logger {
	log := l.log

	// Add correlation ID if present
	if correlationID := c.Get("correlation_id"); correlationID != nil {
		if id, ok := correlationID.(string); ok && id != "" {
			log = log.With().Str("correlation_id", id).Logger()
		}
	}

	// Add operation ID if present
	if operationID := c.Get("operation_id"); operationID != nil {
		if id, ok := operationID.(string); ok && id != "" {
			log = log.With().Str("operation_id", id).Logger()
		}
	}

	// Add request ID
	if requestID := c.Response().Header().Get(echo.HeaderXRequestID); requestID != "" {
		log = log.With().Str("request_id", requestID).Logger()
	}

	// Add trace ID if present (from OpenTelemetry)
	if traceID := c.Get("trace_id"); traceID != nil {
		if id, ok := traceID.(string); ok && id != "" {
			log = log.With().Str("trace_id", id).Logger()
		}
	}

	// Add span ID if present (from OpenTelemetry)
	if spanID := c.Get("span_id"); spanID != nil {
		if id, ok := spanID.(string); ok && id != "" {
			log = log.With().Str("span_id", id).Logger()
		}
	}

	return &Logger{log: log}
}

// WithCtx creates a logger with trace IDs from standard context.Context
func (l *Logger) WithCtx(ctx context.Context) *Logger {
	log := l.log

	// Extract correlation ID from context
	if correlationID := ctx.Value("correlation_id"); correlationID != nil {
		if id, ok := correlationID.(string); ok && id != "" {
			log = log.With().Str("correlation_id", id).Logger()
		}
	}

	// Extract operation ID from context
	if operationID := ctx.Value("operation_id"); operationID != nil {
		if id, ok := operationID.(string); ok && id != "" {
			log = log.With().Str("operation_id", id).Logger()
		}
	}

	return &Logger{log: log}
}

// WithFields creates a logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	log := l.log
	for k, v := range fields {
		log = log.With().Interface(k, v).Logger()
	}
	return &Logger{log: log}
}

// WithField creates a logger with a single additional field
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{log: l.log.With().Interface(key, value).Logger()}
}

// Logging methods with structured fields

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	l.log.Debug().Msg(msg)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log.Debug().Msgf(format, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	l.log.Info().Msg(msg)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log.Info().Msgf(format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	l.log.Warn().Msg(msg)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log.Warn().Msgf(format, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	l.log.Error().Msg(msg)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log.Error().Msgf(format, args...)
}

// ErrorWithErr logs an error message with error object
func (l *Logger) ErrorWithErr(err error, msg string) {
	l.log.Error().Err(err).Msg(msg)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string) {
	l.log.Fatal().Msg(msg)
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log.Fatal().Msgf(format, args...)
}

// Action logs an action execution event
func (l *Logger) Action(actionType, objectType, status string, durationMs float64) {
	l.log.Info().
		Str("event_type", "action_execution").
		Str("action_type", actionType).
		Str("object_type", objectType).
		Str("status", status).
		Float64("duration_ms", durationMs).
		Msg("Action executed")
}

// Workflow logs a workflow event
func (l *Logger) Workflow(workflowID, status string, stepCount int, durationMs float64) {
	l.log.Info().
		Str("event_type", "workflow_execution").
		Str("workflow_id", workflowID).
		Str("status", status).
		Int("step_count", stepCount).
		Float64("duration_ms", durationMs).
		Msg("Workflow executed")
}

// Trace logs a trace event
func (l *Logger) Trace(correlationID, operationID string, event string) {
	l.log.Info().
		Str("event_type", "trace_event").
		Str("correlation_id", correlationID).
		Str("operation_id", operationID).
		Str("event", event).
		Msg("Trace event")
}

// GDPR logs a GDPR-related event
func (l *Logger) GDPR(eventType, dataSubjectID, action string) {
	l.log.Info().
		Str("event_type", "gdpr_event").
		Str("gdpr_event_type", eventType).
		Str("data_subject_id", dataSubjectID).
		Str("action", action).
		Msg("GDPR event")
}

// GetZerolog returns the underlying zerolog.Logger for advanced usage
func (l *Logger) GetZerolog() *zerolog.Logger {
	return &l.log
}

// LoggingMiddleware returns Echo middleware that adds trace-aware logger to context
func LoggingMiddleware(baseLogger *Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Create logger with trace context
			logger := baseLogger.WithContext(c)

			// Store logger in context for handlers to use
			c.Set("logger", logger)

			// Log request start
			logger.Info(
				"Request started: " + c.Request().Method + " " + c.Request().RequestURI,
			)

			// Call next handler
			err := next(c)

			// Log request completion
			status := c.Response().Status
			if err != nil {
				logger.ErrorWithErr(err, "Request failed")
			} else if status >= 500 {
				logger.Error("Request completed with server error")
			} else if status >= 400 {
				logger.Warn("Request completed with client error")
			} else {
				logger.Info("Request completed successfully")
			}

			return err
		}
	}
}

// GetLogger extracts the trace-aware logger from Echo context
func GetLogger(c echo.Context) *Logger {
	if logger := c.Get("logger"); logger != nil {
		if l, ok := logger.(*Logger); ok {
			return l
		}
	}
	// Fallback to basic logger if not found
	return NewLogger(os.Stdout, "unknown")
}

// ContextWithTraceIDs adds trace IDs to standard context.Context
func ContextWithTraceIDs(ctx context.Context, correlationID, operationID string) context.Context {
	ctx = context.WithValue(ctx, "correlation_id", correlationID)
	ctx = context.WithValue(ctx, "operation_id", operationID)
	return ctx
}

// GetCorrelationIDFromContext extracts correlation ID from context
func GetCorrelationIDFromContext(ctx context.Context) string {
	if id := ctx.Value("correlation_id"); id != nil {
		if correlationID, ok := id.(string); ok {
			return correlationID
		}
	}
	return ""
}

// GetOperationIDFromContext extracts operation ID from context
func GetOperationIDFromContext(ctx context.Context) string {
	if id := ctx.Value("operation_id"); id != nil {
		if operationID, ok := id.(string); ok {
			return operationID
		}
	}
	return ""
}
