# EVE v0.0.29 - New Utilities Guide

**Release Date:** November 5, 2025
**Status:** ✅ Released

---

## Executive Summary

EVE v0.0.29 introduces high-impact utilities that will save **800-900 LOC** across the service ecosystem by standardizing common patterns for server management, error handling, and configuration.

### Key Features

1. **http.RunServer()** - Complete server lifecycle management (~700 LOC savings)
2. **Semantic error helpers** - Standardized action error handling (~150 LOC savings)
3. **Common utilities** - MaskSecret(), environment helpers, pointer utilities (~70 LOC savings)

---

## 1. http.RunServer() - Server Lifecycle Management

### Overview

`RunServer()` eliminates 60-70 lines of boilerplate per service by providing standardized:
- Echo server creation with middleware
- Health check endpoints
- Service registry integration
- Signal handling and graceful shutdown

### Before (82 lines per service)

```go
package main

import (
    "os"
    "os/signal"
    "syscall"
    "strconv"

    "eve.evalgo.org/common"
    "eve.evalgo.org/registry"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

func main() {
    logger := common.ServiceLogger("myservice", "1.0.0")

    // Parse port
    port := os.Getenv("PORT")
    if port == "" {
        port = "8090"
    }
    portInt, _ := strconv.Atoi(port)

    // Create Echo server
    e := echo.New()
    e.Use(middleware.Logger())
    e.Use(middleware.Recover())
    e.Use(middleware.CORS())

    // Health check
    e.GET("/health", func(c echo.Context) error {
        return c.JSON(200, map[string]string{
            "status": "healthy",
            "service": "myservice",
            "version": "1.0.0",
        })
    })

    // Register with registry
    if _, err := registry.AutoRegister(registry.AutoRegisterConfig{
        ServiceID:    "myservice",
        ServiceName:  "My Service",
        Description:  "Service description",
        Port:         portInt,
        Directory:    "/home/user/myservice",
        Binary:       "myservice",
        Capabilities: []string{"storage"},
    }); err != nil {
        logger.WithError(err).Warn("Failed to register")
    }

    // Add routes
    e.POST("/api/action", handleAction)

    // Start server
    go func() {
        logger.Infof("Starting service on port %s", port)
        if err := e.Start(":" + port); err != nil {
            logger.WithError(err).Error("Server error")
        }
    }()

    // Signal handling
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info("Shutting down server...")

    // Unregister
    if err := registry.AutoUnregister("myservice"); err != nil {
        logger.WithError(err).Error("Failed to unregister")
    }

    // Graceful shutdown
    if err := e.Close(); err != nil {
        logger.WithError(err).Error("Error during shutdown")
    }

    logger.Info("Server stopped")
}
```

### After (15 lines per service)

```go
package main

import (
    "os"

    evehttp "eve.evalgo.org/http"
    "github.com/labstack/echo/v4"
)

func main() {
    port := evehttp.GetPortInt(os.Getenv("PORT"), 8090)

    cfg := evehttp.DefaultRunServerConfig("myservice", "My Service", "1.0.0")
    cfg.Port = port
    cfg.Capabilities = []string{"storage"}
    cfg.Directory = "/home/user/myservice"
    cfg.Binary = "myservice"

    evehttp.RunServer(cfg, func(e *echo.Echo) error {
        e.POST("/api/action", handleAction)
        return nil
    })
}
```

**Savings: 67 lines per service × 11 services = 737 LOC**

### API Reference

#### RunServerConfig

```go
type RunServerConfig struct {
    // Service identification
    ServiceID   string
    ServiceName string
    Version     string
    Description string

    // Server configuration
    Port            int
    Debug           bool
    BodyLimit       string        // e.g., "10M"
    ReadTimeout     time.Duration
    WriteTimeout    time.Duration
    ShutdownTimeout time.Duration
    AllowedOrigins  []string
    RateLimit       float64 // Requests per second (0 = no limit)

    // Registry configuration (optional)
    EnableRegistry bool
    Directory      string   // Service directory
    Binary         string   // Binary name
    Capabilities   []string // Service capabilities

    // Logger (optional, will create one if nil)
    Logger *common.ContextLogger
}
```

#### DefaultRunServerConfig

```go
func DefaultRunServerConfig(serviceID, serviceName, version string) RunServerConfig
```

Returns a config with sensible defaults:
- Port: 8080
- Body limit: 10M
- Timeouts: 30s read/write, 10s shutdown
- CORS: Allow all origins
- Registry: Enabled

#### RunServer

```go
func RunServer(config RunServerConfig, setupFunc SetupFunc) error
```

**SetupFunc** signature:
```go
type SetupFunc func(*echo.Echo) error
```

**Features:**
- Creates Echo with standard middleware (logger, recover, CORS, request ID)
- Adds health check at `/health`
- Auto-registers with service registry
- Handles SIGINT/SIGTERM for graceful shutdown
- Auto-unregisters on shutdown

---

## 2. Semantic Error Helpers

### Overview

Standardizes error handling across semantic action services, eliminating duplicate error-setting code.

### Before (28 lines per handler)

```go
func handleAction(c echo.Context) error {
    body, err := io.ReadAll(c.Request().Body)
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "Failed to read body")
    }

    action, err := semantic.ParseBaseXAction(body)
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to parse: %v", err))
    }

    result, err := executeAction(action)
    if err != nil {
        // Manual error setting (repeated for each action type)
        switch v := action.(type) {
        case *semantic.TransformAction:
            v.ActionStatus = "FailedActionStatus"
            v.Error = &semantic.PropertyValue{
                Type: "PropertyValue",
                Name: "error",
                Value: fmt.Sprintf("Execution failed: %v", err),
            }
            v.EndTime = time.Now().Format(time.RFC3339)
            return c.JSON(http.StatusInternalServerError, v)
        case *semantic.QueryAction:
            v.ActionStatus = "FailedActionStatus"
            v.Error = &semantic.PropertyValue{
                Type: "PropertyValue",
                Name: "error",
                Value: fmt.Sprintf("Execution failed: %v", err),
            }
            v.EndTime = time.Now().Format(time.RFC3339)
            return c.JSON(http.StatusInternalServerError, v)
        // ... repeated for each action type
        }
    }

    return c.JSON(http.StatusOK, result)
}
```

### After (8 lines per handler)

```go
func handleAction(c echo.Context) error {
    body, err := io.ReadAll(c.Request().Body)
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "Failed to read body")
    }

    action, err := semantic.ParseBaseXAction(body)
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to parse: %v", err))
    }

    result, err := executeAction(action)
    if err != nil {
        return semantic.ReturnActionError(c, action, "Execution failed", err)
    }

    return c.JSON(http.StatusOK, result)
}
```

**Savings: 20 lines per handler × 5-7 handlers per service = 100-140 LOC per semantic service**

### API Reference

#### SetErrorOnAction

```go
func SetErrorOnAction(action interface{}, message string)
```

Sets error status on actions with `string` EndTime fields:
- BaseX actions: Transform, Query, Upload, CreateDatabase, DeleteDatabase
- S3 actions: Upload, Download, Delete, List
- SPARQL: SearchAction
- Infisical: RetrieveAction
- Container: Activate, Deactivate, Download, Build
- Canonical: CanonicalSemanticAction

#### SetErrorOnTimeAction

```go
func SetErrorOnTimeAction(action interface{}, message string)
```

Sets error on actions with `*time.Time` EndTime fields:
- GraphDB actions: Transfer, Create, Delete, Update, Upload

#### ReturnActionError

```go
func ReturnActionError(c echo.Context, action interface{}, message string, err error) error
```

Convenience helper that:
1. Combines message and error
2. Sets error on action
3. Returns HTTP 500 JSON response

#### ReturnActionErrorWithStatus

```go
func ReturnActionErrorWithStatus(c echo.Context, action interface{}, statusCode int, message string, err error) error
```

Same as `ReturnActionError` but with custom HTTP status code.

#### SetSuccessOnAction / SetSuccessOnTimeAction

```go
func SetSuccessOnAction(action interface{})
func SetSuccessOnTimeAction(action interface{})
```

Sets `ActionStatus = "CompletedActionStatus"` and `EndTime` on successful actions.

---

## 3. Common Utilities

### Overview

General-purpose helpers for environment variables, secrets, and pointer operations.

### MaskSecret

```go
func MaskSecret(secret string) string
```

Masks sensitive strings for safe logging:
- Empty string → `"<not set>"`
- Short strings (≤8 chars) → `"***"`
- Long strings → First 4 + "..." + last 4 chars

**Example:**
```go
logger.Infof("API key: %s", common.MaskSecret(apiKey))
// Output: "API key: sk_t...x7Qz"
```

**Savings: 10 lines per service**

### Environment Helpers

```go
func GetEnv(key, defaultValue string) string
func GetEnvInt(key string, defaultValue int) int
func GetEnvBool(key string, defaultValue bool) bool
```

Simple environment variable retrieval with defaults.

**Example:**
```go
port := common.GetEnvInt("PORT", 8080)
debug := common.GetEnvBool("DEBUG", false)
apiURL := common.GetEnv("API_URL", "http://localhost:8080")
```

### Must Helpers

```go
func Must[T any](value T, err error) T
func MustNoError(err error)
```

Panic-on-error helpers for initialization code.

**Example:**
```go
config := common.Must(loadConfig())
common.MustNoError(db.Init())
```

### Pointer Utilities

```go
func Ptr[T any](v T) *T
func PtrValue[T any](ptr *T) T
```

Convenience helpers for pointer operations.

**Example:**
```go
config := Config{
    Enabled: common.Ptr(true),
    Timeout: common.Ptr(30 * time.Second),
}

timeout := common.PtrValue(config.Timeout) // Returns 30s or zero value if nil
```

---

## 4. HTTP Utilities

### GetPortInt

```go
func GetPortInt(envVar string, defaultPort int) int
```

Parses port from environment variable with validation (1-65535) and fallback.

**Example:**
```go
port := http.GetPortInt(os.Getenv("PORT"), 8090)
```

---

## Implementation Roadmap

### Phase 1: Service Updates (Priority: High)

**Target Services for RunServer() adoption (11 services):**
1. basexservice
2. sparqlservice
3. s3service
4. workflowstorageservice
5. infisicalservice
6. pxgraphservice
7. graphium
8. registryservice
9. fetcher
10. templateservice
11. when (daemon mode)

**Estimated Impact:** 737 LOC reduction

**Effort:** 1-2 hours per service

### Phase 2: Semantic Services (Priority: High)

**Target Services for semantic error helpers:**
1. basexservice
2. sparqlservice
3. s3service
4. workflowstorageservice
5. infisicalservice

**Estimated Impact:** 100-150 LOC reduction

**Effort:** 30-45 minutes per service

### Phase 3: Config Migration (Priority: Medium)

See separate EVE Config Adoption Analysis for detailed roadmap.

**Simple services to migrate to EVE config (5 services):**
- basexservice, sparqlservice, s3service, workflowstorageservice, infisicalservice

**Complex service:**
- graphium (custom config → EVE config)

**Estimated Impact:** 400 LOC reduction

---

## Migration Examples

### Example 1: Simple Service with RunServer

**File:** `/home/opunix/basexservice/cmd/main.go`

**Before:**
```go
// 98 lines of server setup, registry, signal handling
```

**After:**
```go
package main

import (
    "os"

    evehttp "eve.evalgo.org/http"
    "github.com/labstack/echo/v4"
)

func main() {
    port := evehttp.GetPortInt(os.Getenv("PORT"), 8090)
    apiKey := os.Getenv("BASEX_API_KEY")

    cfg := evehttp.DefaultRunServerConfig("basexservice", "BaseX Service", "1.0.0")
    cfg.Port = port
    cfg.Capabilities = []string{"xml-storage", "xquery"}
    cfg.Directory = "/home/opunix/basexservice"
    cfg.Binary = "basexservice"

    evehttp.RunServer(cfg, func(e *echo.Echo) error {
        apiKeyMiddleware := evehttp.APIKeyMiddleware(apiKey)
        e.POST("/v1/api/semantic/action", handleSemanticAction, apiKeyMiddleware)
        return nil
    })
}

func handleSemanticAction(c echo.Context) error {
    // Handler implementation with semantic error helpers
    // ...
}
```

### Example 2: Semantic Error Handling

**File:** `/home/opunix/basexservice/internal/api/semantic_api.go`

**Before:**
```go
func handleSemanticAction(c echo.Context) error {
    body, _ := io.ReadAll(c.Request().Body)
    action, err := semantic.ParseBaseXAction(body)
    if err != nil {
        return echo.NewHTTPError(400, fmt.Sprintf("Parse error: %v", err))
    }

    result, err := executeAction(action)
    if err != nil {
        // 20+ lines of manual error setting for each action type
        switch v := action.(type) {
        case *semantic.TransformAction:
            v.ActionStatus = "FailedActionStatus"
            v.Error = &semantic.PropertyValue{...}
            v.EndTime = time.Now().Format(time.RFC3339)
            return c.JSON(500, v)
        // ... repeated for all action types
        }
    }

    return c.JSON(200, result)
}
```

**After:**
```go
func handleSemanticAction(c echo.Context) error {
    body, _ := io.ReadAll(c.Request().Body)
    action, err := semantic.ParseBaseXAction(body)
    if err != nil {
        return echo.NewHTTPError(400, fmt.Sprintf("Parse error: %v", err))
    }

    result, err := executeAction(action)
    if err != nil {
        return semantic.ReturnActionError(c, action, "Execution failed", err)
    }

    return c.JSON(200, result)
}
```

---

## Testing Checklist

### For RunServer() Adoption

- [ ] Service starts successfully on configured port
- [ ] Health endpoint returns correct JSON: `GET /health`
- [ ] Service registers with registry (check registry logs)
- [ ] API endpoints work correctly
- [ ] API key middleware functions (if applicable)
- [ ] Graceful shutdown works (SIGTERM)
- [ ] Service unregisters on shutdown
- [ ] Logs are properly formatted (JSON)

### For Semantic Error Helpers

- [ ] Parse errors return proper error response
- [ ] Execution errors set ActionStatus = "FailedActionStatus"
- [ ] Error field is populated with message
- [ ] EndTime is set correctly
- [ ] Success responses set ActionStatus = "CompletedActionStatus"
- [ ] HTTP status codes are correct (500 for errors, 200 for success)

---

## Benefits Summary

### Developer Experience
- **Less boilerplate:** 60-70 lines saved per service
- **Consistent patterns:** All services use same server setup
- **Faster onboarding:** New services start from template
- **Better errors:** Standardized error handling

### Code Quality
- **Reduced duplication:** 800-900 LOC eliminated
- **Centralized logic:** Updates benefit all services
- **Tested utilities:** Better coverage
- **Type safety:** Compile-time checks

### Operations
- **Consistent behavior:** All services shutdown gracefully
- **Better logging:** Masked secrets, structured logs
- **Service discovery:** Automatic registry integration
- **Health checks:** Standardized format

### Maintenance
- **Single source of truth:** EVE package
- **Easier updates:** Change once, update everywhere
- **Clear patterns:** Easy to understand and follow
- **Documentation:** Built into utilities

---

## Next Steps

1. **Update services to EVE v0.0.29:**
   ```bash
   cd /home/opunix/basexservice
   go get eve.evalgo.org@v0.0.29
   go mod tidy
   ```

2. **Adopt RunServer() pattern:**
   - Start with simple services (basexservice, sparqlservice, s3service)
   - Test thoroughly before moving to complex services
   - Update one service at a time

3. **Adopt semantic error helpers:**
   - Update semantic action handlers
   - Remove duplicate error-setting code
   - Test all action types

4. **Migrate to EVE config** (optional, Phase 3):
   - See EVE Config Adoption Analysis
   - Start with simple ENV-based services
   - Migrate graphium last (most complex)

---

## Support & References

- **EVE Repository:** `/home/opunix/eve`
- **Documentation:** `/home/opunix/eve/docs/`
- **Utility Patterns Guide:** `/home/opunix/eve/docs/UTILITY_PATTERNS.md`
- **Consolidation Summary:** `/home/opunix/eve/docs/EVE_CONSOLIDATION_SUMMARY.md`

---

**Release:** v0.0.29
**Date:** November 5, 2025
**Author:** EVE Platform Team
**Status:** ✅ Ready for adoption
