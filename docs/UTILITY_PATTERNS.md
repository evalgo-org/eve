# EVE Utility Patterns Guide

This guide provides standardized patterns for using EVE utilities across all services in the EVE ecosystem.

## Table of Contents
- [Logging](#logging)
- [HTTP Utilities](#http-utilities)
- [Configuration](#configuration)
- [Service Registry](#service-registry)
- [Migration Checklist](#migration-checklist)

---

## Logging

### Structured Logging with ContextLogger

Replace standard library `log` package with EVE's structured logging for JSON output, context fields, and consistent formatting.

#### Before (Standard Library)
```go
import "log"

func main() {
    log.Printf("Starting service on port %s", port)

    if err := doSomething(); err != nil {
        log.Fatalf("Failed to do something: %v", err)
    }
}
```

#### After (EVE Structured Logging)
```go
import "eve.evalgo.org/common"

func main() {
    logger := common.ServiceLogger("myservice", "1.0.0")

    logger.Infof("Starting service on port %s", port)

    if err := doSomething(); err != nil {
        logger.WithError(err).Error("Failed to do something")
        os.Exit(1)
    }
}
```

### Logging Levels

```go
logger.Debug("Debugging information")
logger.Info("Informational message")
logger.Infof("Formatted info: %s", value)
logger.Warn("Warning message")
logger.Warnf("Formatted warning: %s", value)
logger.Error("Error message")
logger.Errorf("Formatted error: %s", value)
```

### Error Logging Pattern

Always use `.WithError()` chain for error logging:

```go
if err := operation(); err != nil {
    logger.WithError(err).Error("Operation failed")
    return err
}
```

### Context Fields

Add structured fields to log entries:

```go
logger.WithFields(map[string]interface{}{
    "user_id": userID,
    "action": "login",
    "ip": clientIP,
}).Info("User logged in")
```

---

## HTTP Utilities

### Health Check Handler

Replace custom health check implementations with `evehttp.HealthCheckHandler`.

#### Before (Custom Implementation)
```go
e.GET("/health", func(c echo.Context) error {
    return c.JSON(200, map[string]string{
        "status":  "healthy",
        "service": "myservice",
    })
})
```

#### After (EVE Health Check)
```go
import evehttp "eve.evalgo.org/http"

e.GET("/health", evehttp.HealthCheckHandler("myservice", "1.0.0"))
```

### Health Check with Custom Details

```go
e.GET("/health", evehttp.HealthCheckHandlerWithDetails(
    "myservice",
    "1.0.0",
    func() map[string]interface{} {
        return map[string]interface{}{
            "database": checkDatabase(),
            "cache": checkCache(),
        }
    },
))
```

### API Key Middleware

Replace custom API key implementations with `evehttp.APIKeyMiddleware`.

#### Before (Custom Middleware)
```go
apiKey := os.Getenv("MY_API_KEY")
if apiKey != "" {
    apiKeyMiddleware := middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
        return key == apiKey, nil
    })
    e.POST("/api/endpoint", handler, apiKeyMiddleware)
    log.Printf("API Key authentication enabled")
} else {
    e.POST("/api/endpoint", handler)
    log.Printf("Running in development mode (no API key required)")
}
```

#### After (EVE Middleware)
```go
import evehttp "eve.evalgo.org/http"

apiKey := os.Getenv("MY_API_KEY")
apiKeyMiddleware := evehttp.APIKeyMiddleware(apiKey)
e.POST("/api/endpoint", handler, apiKeyMiddleware)
```

**Benefits:**
- Automatically skips validation if `apiKey` is empty (development mode)
- Checks both `X-API-Key` and `x-api-key` headers
- Returns proper HTTP 401 errors with clear messages
- Reduces boilerplate by ~15 lines

### Complete HTTP Service Setup

```go
package main

import (
    "os"
    "eve.evalgo.org/common"
    evehttp "eve.evalgo.org/http"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

func main() {
    logger := common.ServiceLogger("myservice", "1.0.0")

    e := echo.New()
    e.Use(middleware.Logger())
    e.Use(middleware.Recover())
    e.Use(middleware.CORS())

    // Health check
    e.GET("/health", evehttp.HealthCheckHandler("myservice", "1.0.0"))

    // Protected endpoint
    apiKey := os.Getenv("MY_API_KEY")
    apiKeyMiddleware := evehttp.APIKeyMiddleware(apiKey)
    e.POST("/api/endpoint", handleEndpoint, apiKeyMiddleware)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    logger.Infof("Starting service on port %s", port)
    if err := e.Start(":" + port); err != nil {
        logger.WithError(err).Error("Server error")
    }
}
```

---

## Configuration

### Viper-based Configuration

EVE provides a flexible configuration system with multiple sources.

#### Before (Environment Variables Only)
```go
port := os.Getenv("PORT")
if port == "" {
    port = "8080"
}

dbURL := os.Getenv("DATABASE_URL")
if dbURL == "" {
    dbURL = "http://localhost:5984"
}
```

#### After (EVE Config)
```go
import "eve.evalgo.org/config"

cfg, err := config.LoadConfig("MYSERVICE", "")
if err != nil {
    logger.WithError(err).Error("Failed to load config")
    os.Exit(1)
}

port := cfg.Server.Port  // Defaults to 8080
dbURL := cfg.Database.URL  // Defaults to http://localhost:5984
```

### Configuration Sources (Priority Order)

1. Environment variables (highest priority)
2. `.env` file
3. YAML configuration file (`config.yaml`)
4. Default values (lowest priority)

### Example config.yaml

```yaml
service:
  name: myservice
  version: 1.0.0
  environment: production

server:
  host: 0.0.0.0
  port: 8080
  debug: false

database:
  url: http://localhost:5984
  database: myservice_db
  username: admin
  password: secret

logging:
  level: info
  format: json
  output: stdout

security:
  api_key: ${API_KEY}  # Can reference env vars
  rate_limit: 100
```

### Environment Variable Mapping

With prefix `MYSERVICE`:
- `MYSERVICE_SERVER_PORT=9000` → `cfg.Server.Port`
- `MYSERVICE_DATABASE_URL=...` → `cfg.Database.URL`
- `MYSERVICE_LOGGING_LEVEL=debug` → `cfg.Logging.Level`

---

## Service Registry

### Auto-registration Pattern

Register services with the EVE registry for discovery and health monitoring.

```go
import "eve.evalgo.org/registry"

// Auto-register with registry
portInt, _ := strconv.Atoi(port)
if _, err := registry.AutoRegister(registry.AutoRegisterConfig{
    ServiceID:    "myservice",
    ServiceName:  "My Service",
    Description:  "Service description",
    Port:         portInt,
    Directory:    "/home/user/myservice",
    Binary:       "myservice",
    Capabilities: []string{"capability1", "capability2"},
}); err != nil {
    logger.WithError(err).Error("Failed to register with registry")
}

// Graceful shutdown with unregistration
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

logger.Info("Shutting down server...")

if err := registry.AutoUnregister("myservice"); err != nil {
    logger.WithError(err).Error("Failed to unregister from registry")
}
```

**Note:** Registry registration is optional and skipped if `REGISTRYSERVICE_API_URL` is not set.

---

## Migration Checklist

### For HTTP Services

- [ ] Replace `log` imports with `eve.evalgo.org/common`
- [ ] Initialize logger: `logger := common.ServiceLogger(name, version)`
- [ ] Replace `log.Printf` with `logger.Infof`
- [ ] Replace `log.Fatalf` with `logger.WithError(err).Error()` + `os.Exit(1)`
- [ ] Replace custom health check with `evehttp.HealthCheckHandler`
- [ ] Replace custom API key middleware with `evehttp.APIKeyMiddleware`
- [ ] Add proper error handling for registry operations
- [ ] Add `evehttp` import alias: `evehttp "eve.evalgo.org/http"`
- [ ] Remove unused imports (`log`, `fmt` if only used for Printf)
- [ ] Run `go mod tidy`
- [ ] Test build: `go build ./cmd`
- [ ] Test health endpoint: `curl http://localhost:PORT/health`
- [ ] Test API key middleware with and without key

### For CLI Tools

- [ ] Replace `log` imports with `eve.evalgo.org/common`
- [ ] Initialize logger: `logger := common.ServiceLogger(name, version)`
- [ ] Replace `log.Printf` with `logger.Infof`
- [ ] Replace `log.Fatalf` with `logger.WithError(err).Error()` + `os.Exit(1)`
- [ ] Keep `flag` package for command-line arguments (appropriate for CLI)
- [ ] Add informative startup logging
- [ ] Run `go mod tidy`
- [ ] Test build and execution

### Code Reduction Metrics

Expected LOC reduction per service:
- **Logging replacement**: ~5 lines
- **Health check**: ~8-10 lines
- **API key middleware**: ~15-18 lines
- **Total**: ~28-33 lines per HTTP service

---

## Common Patterns

### Graceful Shutdown

```go
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

logger.Info("Shutting down server...")

// Cleanup operations
if err := registry.AutoUnregister(serviceID); err != nil {
    logger.WithError(err).Error("Failed to unregister from registry")
}

if err := e.Close(); err != nil {
    logger.WithError(err).Error("Error during shutdown")
}

logger.Info("Server stopped")
```

### Error Handling

Always use the `.WithError()` chain for consistent error logging:

```go
// ✅ Good
if err := operation(); err != nil {
    logger.WithError(err).Error("Operation failed")
    return err
}

// ❌ Bad
if err := operation(); err != nil {
    logger.Error("Operation failed:", err)  // Wrong signature
    return err
}

// ❌ Bad
if err := operation(); err != nil {
    logger.Errorf("Operation failed: %v", err)  // Loses structured error
    return err
}
```

### Checking Unchecked Errors

Linters will catch unchecked errors. Always handle registry operations:

```go
// ✅ Good
if _, err := registry.AutoRegister(config); err != nil {
    logger.WithError(err).Error("Failed to register")
}

// ❌ Bad - linter error
registry.AutoRegister(config)  // Error not checked

// For defer calls that must succeed
defer func() {
    _ = resp.Body.Close()  // Explicitly ignore with _
}()
```

---

## Testing

### Verify Structured Logging

```bash
# Start service
./myservice &

# Check logs are JSON formatted
# Should see: {"level":"info","msg":"Starting service on port 8080",...}
```

### Test Health Endpoint

```bash
curl http://localhost:8080/health
# Expected: {"service":"myservice","status":"healthy","version":"1.0.0"}
```

### Test API Key Middleware

```bash
# Without API key (development mode - should work)
curl -X POST http://localhost:8080/api/endpoint

# With API key required
export MY_API_KEY=secret123
./myservice &

# Without key (should fail with 401)
curl -X POST http://localhost:8080/api/endpoint
# Expected: {"message":"Missing API key"}

# With correct key (should work)
curl -X POST http://localhost:8080/api/endpoint -H "X-API-Key: secret123"
```

---

## Examples

See these services for reference implementations:
- **s3service**: `/home/opunix/s3service/cmd/main.go`
- **sparqlservice**: `/home/opunix/sparqlservice/cmd/main.go`
- **infisicalservice**: `/home/opunix/infisicalservice/cmd/main.go`
- **when-daemon**: `/home/opunix/when/cmd/when-daemon/main.go` (CLI example)

All examples demonstrate:
- Structured logging
- EVE health checks
- API key middleware
- Registry integration
- Graceful shutdown

---

## Benefits Summary

### Code Quality
- Consistent logging format across all services
- Standardized error handling
- Reduced code duplication

### Developer Experience
- Less boilerplate code (~30 lines per service)
- Easier to onboard new services
- Clear patterns to follow

### Operations
- JSON structured logs for easy parsing
- Consistent health check responses
- Service discovery via registry
- Better error visibility

---

## Support

For questions or issues:
- Check EVE source: `/home/opunix/eve`
- Review examples in semantic services
- See EVE package documentation in source files
