# Log Correlation with Trace Context

Automatic correlation of application logs with distributed traces for unified observability.

## Overview

The `tracing.Logger` provides structured logging with automatic trace context injection. Every log entry includes:
- `correlation_id` - Links logs across all services in a workflow
- `operation_id` - Unique ID for this specific operation
- `trace_id` / `span_id` - OpenTelemetry trace identifiers
- `request_id` - HTTP request identifier
- `service` - Service name

This enables:
- **Find all logs for a workflow**: Search by `correlation_id`
- **Debug specific actions**: Search by `operation_id`
- **Correlate with traces**: Use `trace_id` to link with OpenTelemetry
- **Track requests**: Use `request_id` for HTTP request tracing

## Quick Start

### 1. Initialize Logger

```go
package main

import (
	"eve.evalgo.org/tracing"
	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	// Create base logger
	logger := tracing.NewLogger(nil, "containerservice")

	// Or use console logger for development
	// logger := tracing.NewConsoleLogger("containerservice")

	// Add logging middleware (must come after tracing middleware)
	e.Use(tracing.LoggingMiddleware(logger))

	// Your routes...
	e.POST("/v1/api/semantic-action", handleSemanticAction)

	e.Start(":8080")
}
```

### 2. Use Logger in Handlers

```go
func handleSemanticAction(c echo.Context) error {
	// Get trace-aware logger from context
	logger := tracing.GetLogger(c)

	// All logs automatically include correlation_id, operation_id, etc.
	logger.Info("Processing semantic action")

	// Log with additional fields
	logger.WithField("action_type", "CreateAction").
		Info("Executing action")

	// Log errors with context
	if err := doSomething(); err != nil {
		logger.ErrorWithErr(err, "Failed to execute action")
		return err
	}

	logger.Info("Action completed successfully")
	return c.JSON(200, result)
}
```

### 3. Use in Background Goroutines

```go
func handleSemanticAction(c echo.Context) error {
	logger := tracing.GetLogger(c)

	// Extract correlation/operation IDs before spawning goroutine
	correlationID := tracing.GetCorrelationID(c)
	operationID := tracing.GetOperationID(c)

	// Spawn background worker
	go func() {
		// Create context with trace IDs
		ctx := tracing.ContextWithTraceIDs(context.Background(), correlationID, operationID)

		// Create logger from context
		bgLogger := logger.WithCtx(ctx)

		bgLogger.Info("Background job started")
		// Do work...
		bgLogger.Info("Background job completed")
	}()

	return c.JSON(202, "Accepted")
}
```

## Log Output Examples

### JSON Output (Production)

```json
{
  "level": "info",
  "service": "containerservice",
  "correlation_id": "wf-20250108-abc123",
  "operation_id": "op-xyz789",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "request_id": "req-001",
  "time": "2025-01-08T10:30:45Z",
  "message": "Processing semantic action"
}

{
  "level": "info",
  "service": "containerservice",
  "correlation_id": "wf-20250108-abc123",
  "operation_id": "op-xyz789",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "request_id": "req-001",
  "event_type": "action_execution",
  "action_type": "CreateAction",
  "object_type": "SoftwareApplication",
  "status": "completed",
  "duration_ms": 1234.56,
  "time": "2025-01-08T10:30:46Z",
  "message": "Action executed"
}
```

### Console Output (Development)

```
2025-01-08T10:30:45-05:00 INF Processing semantic action
  correlation_id=wf-20250108-abc123
  operation_id=op-xyz789
  service=containerservice

2025-01-08T10:30:46-05:00 INF Action executed
  action_type=CreateAction
  correlation_id=wf-20250108-abc123
  duration_ms=1234.56
  event_type=action_execution
  object_type=SoftwareApplication
  operation_id=op-xyz789
  service=containerservice
  status=completed
```

## Logging Methods

### Basic Logging

```go
logger.Debug("Debug message")
logger.Debugf("Debug with args: %s", arg)

logger.Info("Info message")
logger.Infof("Info with args: %s", arg)

logger.Warn("Warning message")
logger.Warnf("Warning with args: %s", arg)

logger.Error("Error message")
logger.Errorf("Error with args: %s", arg)

logger.ErrorWithErr(err, "Error with error object")

logger.Fatal("Fatal message")  // Exits process
logger.Fatalf("Fatal with args: %s", arg)
```

### Structured Logging

```go
// Add single field
logger.WithField("user_id", "user123").
	Info("User logged in")

// Add multiple fields
logger.WithFields(map[string]interface{}{
	"user_id": "user123",
	"ip_address": "192.168.1.1",
	"session_id": "sess-xyz",
}).Info("User logged in")
```

### Domain-Specific Logging

```go
// Log action execution
logger.Action("CreateAction", "SoftwareApplication", "completed", 1234.56)
// Output: {"event_type":"action_execution","action_type":"CreateAction",...}

// Log workflow execution
logger.Workflow("wf-001", "completed", 5, 5678.90)
// Output: {"event_type":"workflow_execution","workflow_id":"wf-001",...}

// Log trace events
logger.Trace(correlationID, operationID, "trace_recorded")
// Output: {"event_type":"trace_event","event":"trace_recorded",...}

// Log GDPR events
logger.GDPR("erasure_request", "user-123", "data_deleted")
// Output: {"event_type":"gdpr_event","gdpr_event_type":"erasure_request",...}
```

## Querying Logs

### Find All Logs for a Workflow

```bash
# With jq
cat logs.json | jq -c 'select(.correlation_id == "wf-20250108-abc123")'

# With grep
grep '"correlation_id":"wf-20250108-abc123"' logs.json

# In Loki/Grafana
{correlation_id="wf-20250108-abc123"}

# In Elasticsearch
GET /logs/_search
{
  "query": {
    "term": { "correlation_id": "wf-20250108-abc123" }
  }
}
```

### Find Logs for Specific Operation

```bash
# In Loki
{operation_id="op-xyz789"}

# With jq
cat logs.json | jq -c 'select(.operation_id == "op-xyz789")'
```

### Find All Errors in a Workflow

```bash
# In Loki
{correlation_id="wf-20250108-abc123"} |~ "level.*error"

# With jq
cat logs.json | jq -c 'select(.correlation_id == "wf-20250108-abc123" and .level == "error")'
```

### Cross-Reference with Traces

```bash
# Get correlation_id from trace query
correlation_id=$(psql -U eve -d tracing -tAc \
  "SELECT correlation_id FROM action_executions WHERE operation_id = 'op-xyz789'")

# Query logs with that correlation_id
cat logs.json | jq -c "select(.correlation_id == \"$correlation_id\")"
```

## Integration with Log Aggregation

### Grafana Loki

**Promtail Configuration** (`promtail.yml`):

```yaml
server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: eve-services
    static_configs:
      - targets:
          - localhost
        labels:
          job: eve-services
          __path__: /var/log/eve/*.log

    # Parse JSON logs
    pipeline_stages:
      - json:
          expressions:
            level: level
            service: service
            correlation_id: correlation_id
            operation_id: operation_id
            trace_id: trace_id
            message: message
            event_type: event_type

      # Extract labels
      - labels:
          level:
          service:
          correlation_id:
          operation_id:
          event_type:

      # Add timestamp
      - timestamp:
          source: time
          format: RFC3339
```

**Querying in Grafana**:

```logql
# All logs for a service
{service="containerservice"}

# Error logs
{service="containerservice"} |~ "level.*error"

# Logs for specific workflow
{correlation_id="wf-20250108-abc123"}

# Action execution logs
{event_type="action_execution"}

# Logs with duration > 1 second
{service="containerservice"} | json | duration_ms > 1000
```

### Elasticsearch / OpenSearch

**Filebeat Configuration** (`filebeat.yml`):

```yaml
filebeat.inputs:
  - type: log
    enabled: true
    paths:
      - /var/log/eve/*.log
    json.keys_under_root: true
    json.add_error_key: true

output.elasticsearch:
  hosts: ["localhost:9200"]
  index: "eve-logs-%{+yyyy.MM.dd}"

setup.ilm.enabled: false
setup.template.name: "eve-logs"
setup.template.pattern: "eve-logs-*"
```

**Querying in Kibana**:

```
correlation_id:"wf-20250108-abc123"
service:"containerservice" AND level:"error"
event_type:"action_execution" AND duration_ms:>1000
```

### CloudWatch Logs

```go
import (
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"io"
)

// Send logs to CloudWatch
func NewCloudWatchLogger(cwClient *cloudwatchlogs.Client, logGroup, logStream string) *Logger {
	// Use custom writer that sends to CloudWatch
	writer := &CloudWatchWriter{
		client:    cwClient,
		logGroup:  logGroup,
		logStream: logStream,
	}

	return tracing.NewLogger(writer, "containerservice")
}
```

## Advanced Usage

### Custom Log Levels

```go
import "github.com/rs/zerolog"

// Set global log level
zerolog.SetGlobalLevel(zerolog.InfoLevel)

// Or set per-logger
logger := tracing.NewLogger(nil, "containerservice")
zlog := logger.GetZerolog()
zlog.Level(zerolog.DebugLevel)
```

### Multiple Output Targets

```go
import (
	"os"
	"io"
)

// Write to both stdout and file
file, _ := os.OpenFile("/var/log/eve/service.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
multiWriter := io.MultiWriter(os.Stdout, file)

logger := tracing.NewLogger(multiWriter, "containerservice")
```

### Sampling (Reduce Log Volume)

```go
import "github.com/rs/zerolog"

// Sample: only log 1 in every 10 debug messages
zlog := logger.GetZerolog()
sampledLogger := zlog.Sample(&zerolog.BurstSampler{
	Burst:       5,
	Period:      time.Second,
	NextSampler: &zerolog.BasicSampler{N: 10},
})

logger := &tracing.Logger{}  // Wrap sampled logger
```

### Async Logging (High-Throughput)

```go
import "github.com/rs/zerolog/diode"

// Non-blocking async writer
writer := diode.NewWriter(os.Stdout, 1000, 10*time.Millisecond, func(missed int) {
	fmt.Printf("Logger dropped %d messages\n", missed)
})

logger := tracing.NewLogger(writer, "containerservice")
```

## Best Practices

### DO

1. **Always use trace-aware logger**:
   ```go
   logger := tracing.GetLogger(c)  // ✓ Good
   ```

2. **Use structured fields**:
   ```go
   logger.WithField("user_id", userID).Info("Action completed")  // ✓ Good
   ```

3. **Log action start and end**:
   ```go
   logger.Info("Starting container creation")
   // ... do work
   logger.Action("CreateAction", "SoftwareApplication", "completed", durationMs)
   ```

4. **Include error details**:
   ```go
   logger.ErrorWithErr(err, "Failed to create container")  // ✓ Good
   ```

5. **Use appropriate log levels**:
   - Debug: Detailed diagnostic info
   - Info: Normal operation events
   - Warn: Unexpected but handled situations
   - Error: Errors requiring attention
   - Fatal: Unrecoverable errors

### DON'T

1. **Don't use raw fmt.Println**:
   ```go
   fmt.Println("Action completed")  // ✗ Bad - no trace context
   ```

2. **Don't log sensitive data**:
   ```go
   logger.WithField("password", pwd).Info("Login")  // ✗ Bad
   logger.WithField("ssn", ssn).Info("User created")  // ✗ Bad
   ```

3. **Don't string-format structured data**:
   ```go
   logger.Infof("User %s did action %s", user, action)  // ✗ Bad
   logger.WithFields(map[string]interface{}{           // ✓ Good
       "user_id": user,
       "action": action,
   }).Info("User action")
   ```

4. **Don't create new loggers in hot paths**:
   ```go
   func processItem() {
       logger := tracing.NewLogger(...)  // ✗ Bad - creates logger per item
   }
   ```

5. **Don't over-log**:
   ```go
   // ✗ Bad - too verbose
   logger.Debug("Step 1")
   logger.Debug("Step 2")
   logger.Debug("Step 3")

   // ✓ Good - meaningful checkpoints
   logger.Info("Processing started")
   logger.Info("Processing completed")
   ```

## Troubleshooting

### Logs Missing Trace IDs

**Problem**: Logs show `correlation_id=""` or field is missing

**Solution**:
1. Ensure tracing middleware is installed:
   ```go
   e.Use(tracer.Middleware())
   ```
2. Ensure logging middleware comes AFTER tracing middleware
3. Check that requests include `X-Correlation-ID` header

### Logs Not Appearing

**Problem**: No log output at all

**Solution**:
1. Check log level: `zerolog.SetGlobalLevel(zerolog.DebugLevel)`
2. Check writer is not nil: `tracing.NewLogger(os.Stdout, "service")`
3. Check file permissions if writing to file

### Logs Not in JSON Format

**Problem**: Logs are human-readable instead of JSON

**Solution**:
- Use `tracing.NewLogger()` not `tracing.NewConsoleLogger()`
- NewConsoleLogger is for development only

### Performance Impact

**Problem**: Logging slows down application

**Solution**:
1. Use async logging (diode writer)
2. Reduce log level (Info instead of Debug)
3. Use sampling for high-frequency logs
4. Profile with pprof to identify hot spots

## Related Documentation

- Tracing Middleware: `/home/opunix/eve/tracing/middleware.go`
- Metrics Integration: `/home/opunix/eve/tracing/metrics.go`
- MCP Trace Queries: `/home/opunix/when` (when-mcp)
- Grafana Dashboards: `/home/opunix/eve/grafana/dashboards/`
