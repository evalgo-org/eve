# ActionRegistry Migration Guide

This guide explains how to migrate EVE services from switch-based action handling to the extensible `ActionRegistry` pattern.

## Problem Statement

The old pattern uses switch statements to dispatch semantic actions:

```go
func handleSemanticAction(c echo.Context) error {
    action, err := semantic.ParseSemanticAction(bodyBytes)
    if err != nil {
        return semantic.ReturnActionError(c, nil, "Failed to parse", err)
    }

    switch action.Type {
    case "ReplaceAction":
        return handleSemanticReplace(c, action)
    case "CreateAction":
        return handleSemanticCreate(c, action)
    default:
        return semantic.ReturnActionError(c, action, fmt.Sprintf("Unsupported: %s", action.Type), nil)
    }
}
```

**Problems:**
- Violates Open/Closed Principle (must modify code to extend)
- Requires refactoring when adding new action types
- Switch statements are brittle and error-prone
- No self-documentation of service capabilities

## Solution: ActionRegistry Pattern

The new pattern uses a registry where handlers self-register at startup:

```go
// In main() - register handlers at startup
semantic.MustRegister("ReplaceAction", handleSemanticReplace)
semantic.MustRegister("CreateAction", handleSemanticCreate)

// In handler - dispatch to registered handler
func handleSemanticAction(c echo.Context) error {
    action, err := semantic.ParseSemanticAction(bodyBytes)
    if err != nil {
        return semantic.ReturnActionError(c, nil, "Failed to parse", err)
    }

    return semantic.Handle(c, action)  // Dispatches to registered handler
}
```

**Benefits:**
- Add new action types without modifying core code
- Self-documenting service capabilities
- Thread-safe handler registration
- Enables plugin-style architecture
- Maintains backward compatibility

## Migration Steps

### Step 1: Import semantic package

If your `main.go` doesn't already import `eve.evalgo.org/semantic`, add it:

```go
import (
    "eve.evalgo.org/semantic"
    // ... other imports
)
```

### Step 2: Register handlers at startup

In your `main()` function, after logger initialization but before starting the server, register all action handlers:

```go
func main() {
    // Initialize logger
    logger = common.ServiceLogger("yourservice", "1.0.0")

    // Register action handlers with the semantic action registry
    semantic.MustRegister("ReplaceAction", handleSemanticReplace)
    semantic.MustRegister("CreateAction", handleSemanticCreate)
    semantic.MustRegister("SearchAction", handleSemanticSearch)
    // ... register all your action handlers

    e := echo.New()
    // ... rest of main()
}
```

**Notes:**
- Use `semantic.MustRegister()` - it panics on duplicate registration (prevents bugs)
- Register handlers before starting the server (during initialization)
- Registration order doesn't matter

### Step 3: Simplify handleSemanticAction

Replace the switch statement with `semantic.Handle()`:

**Before:**
```go
func handleSemanticAction(c echo.Context) error {
    action, err := semantic.ParseSemanticAction(bodyBytes)
    if err != nil {
        return semantic.ReturnActionError(c, nil, "Failed to parse", err)
    }

    switch action.Type {
    case "ReplaceAction":
        return handleSemanticReplace(c, action)
    case "CreateAction":
        return handleSemanticCreate(c, action)
    default:
        return semantic.ReturnActionError(c, action, fmt.Sprintf("Unsupported: %s", action.Type), nil)
    }
}
```

**After:**
```go
func handleSemanticAction(c echo.Context) error {
    action, err := semantic.ParseSemanticAction(bodyBytes)
    if err != nil {
        return semantic.ReturnActionError(c, nil, "Failed to parse", err)
    }

    // Dispatch to registered handler using the ActionRegistry
    // No switch statement needed - handlers are registered at startup
    return semantic.Handle(c, action)
}
```

### Step 4: Remove unused imports

If you removed the switch statement and no longer use `fmt.Sprintf()` for error messages, remove the `fmt` import if it's no longer needed.

### Step 5: Update go.mod

Upgrade to eve.evalgo.org v0.0.32 or later:

```bash
cd /path/to/your/service
GOPROXY=direct go get eve.evalgo.org@v0.0.32
go mod tidy
```

### Step 6: Build and test

```bash
go build ./cmd/yourservice
```

Verify:
- Service compiles successfully
- All registered action types still work
- Error handling for unsupported action types works

## Complete Example: templateservice

See [templateservice](../../templateservice/) for the reference implementation.

### main.go
```go
package main

import (
    "eve.evalgo.org/common"
    "eve.evalgo.org/semantic"
    "eve.evalgo.org/statemanager"
    "github.com/labstack/echo/v4"
)

func main() {
    logger = common.ServiceLogger("templateservice", "1.0.0")

    // Register action handlers with the semantic action registry
    // This allows the service to handle semantic actions without modifying switch statements
    semantic.MustRegister("ReplaceAction", handleSemanticReplace)

    e := echo.New()
    // ... rest of setup
}
```

### semantic_api.go
```go
package main

import (
    "bytes"
    "net/http"
    "os"
    "text/template"

    "eve.evalgo.org/semantic"
    "github.com/labstack/echo/v4"
)

func handleSemanticAction(c echo.Context) error {
    // Parse semantic action
    buf := new(bytes.Buffer)
    if _, err := buf.ReadFrom(c.Request().Body); err != nil {
        return semantic.ReturnActionError(c, nil, "Failed to read request body", err)
    }
    bodyBytes := buf.Bytes()

    action, err := semantic.ParseSemanticAction(bodyBytes)
    if err != nil {
        return semantic.ReturnActionError(c, nil, "Failed to parse semantic action", err)
    }

    // Dispatch to registered handler using the ActionRegistry
    // No switch statement needed - handlers are registered at startup
    return semantic.Handle(c, action)
}

func handleSemanticReplace(c echo.Context, action *semantic.SemanticAction) error {
    // ... handler implementation
}
```

## Adding New Action Types

After migration, adding new action types requires NO changes to `handleSemanticAction()`:

### Old way (requires refactoring):
```go
// Must modify switch statement
switch action.Type {
case "ReplaceAction":
    return handleSemanticReplace(c, action)
case "CreateAction":  // NEW - must add case
    return handleSemanticCreate(c, action)
default:
    return semantic.ReturnActionError(c, action, "Unsupported", nil)
}
```

### New way (no refactoring needed):
```go
// In main() - just add one line:
semantic.MustRegister("CreateAction", handleSemanticCreate)

// handleSemanticAction() stays unchanged - it just dispatches
return semantic.Handle(c, action)
```

## ActionRegistry API Reference

### Registration Functions

```go
// Register a handler for a specific action type
// Returns error if action type already registered
semantic.Register(actionType string, handler ActionHandler) error

// Register a handler and panic if it fails
// Use this in initialization code
semantic.MustRegister(actionType string, handler ActionHandler)

// Remove a handler for a specific action type
semantic.Unregister(actionType string)
```

### Dispatch Functions

```go
// Dispatch an action to the appropriate handler
// Returns error if no handler registered for action.Type
semantic.Handle(c echo.Context, action *SemanticAction) error
```

### Introspection Functions

```go
// Get list of all registered action types
semantic.GetRegisteredActions() []string

// Check if a handler is registered for a specific action type
semantic.HasHandler(actionType string) bool
```

### ActionHandler Type

```go
// ActionHandler is the function signature for all action handlers
type ActionHandler func(c echo.Context, action *SemanticAction) error
```

Your handler functions must match this signature:
```go
func handleSemanticCreate(c echo.Context, action *semantic.SemanticAction) error {
    // ... implementation
    return c.JSON(http.StatusOK, action)
}
```

## Advanced Usage

### Custom Registry (per-service isolation)

If you need service-specific registries (e.g., for testing):

```go
// Create a custom registry
registry := semantic.NewActionRegistry()

// Register handlers
registry.MustRegister("ReplaceAction", handleSemanticReplace)

// Use the registry
return registry.Handle(c, action)
```

Most services should use the default global registry via `semantic.Register()` / `semantic.Handle()`.

### Dynamic Handler Registration

Handlers can be registered at any time (not just at startup):

```go
// Register a handler dynamically
if err := semantic.Register("CustomAction", handleCustom); err != nil {
    logger.Errorf("Failed to register custom handler: %v", err)
}

// Unregister when done
defer semantic.Unregister("CustomAction")
```

### Plugin Architecture

The registry enables plugin-style architectures:

```go
// In plugin.go
type Plugin interface {
    Register(registry *semantic.ActionRegistry)
}

// In main.go
for _, plugin := range plugins {
    plugin.Register(semantic.DefaultRegistry)
}
```

## Services to Migrate

The following services need migration to ActionRegistry:

1. ✅ **templateservice** - Reference implementation (commit ce25135)
2. ⬜ **workflowstorageservice** - Uses CreateAction, SearchAction, UpdateAction, DeleteAction
3. ⬜ **infisicalservice** - Uses CreateAction, SearchAction, UpdateAction, DeleteAction
4. ⬜ **s3service** - Uses UploadAction, DownloadAction, DeleteAction
5. ⬜ **basexservice** - Uses CreateAction, SearchAction, UpdateAction, DeleteAction
6. ⬜ **sparqlservice** - Uses SearchAction, CreateAction, UpdateAction
7. ⬜ **containerservice** - Uses CreateAction, ActivateAction, DeactivateAction, DeleteAction
8. ⬜ **antwrapperservice** - Uses ExecuteAction
9. ⬜ **rabbitmqservice** - Uses SendAction, ReceiveAction
10. ⬜ **graphdbservice** (pxgraphservice) - Uses SearchAction, CreateAction, UpdateAction, DeleteAction
11. ⬜ **when** - Workflow orchestration with multiple action types

## Backward Compatibility

The ActionRegistry pattern is 100% backward compatible:

- **Semantic action format unchanged** - JSON-LD request/response format is identical
- **Handler signatures unchanged** - Existing `handleSemantic*` functions work as-is
- **REST endpoints unchanged** - REST adapters continue to work
- **State tracking unchanged** - StateManager integration continues to work
- **Tracing unchanged** - Tracing middleware continues to work

The only changes are:
1. Handler registration (one line per action type in `main()`)
2. Dispatcher simplification (replace switch with `semantic.Handle()`)

## Testing

After migration, verify:

1. **Build succeeds**
   ```bash
   go build ./cmd/yourservice
   ```

2. **All registered actions work**
   ```bash
   curl -X POST http://localhost:8095/v1/api/semantic/action \
     -H "Content-Type: application/json" \
     -d '{"@type": "ReplaceAction", ...}'
   ```

3. **Unsupported actions return proper errors**
   ```bash
   curl -X POST http://localhost:8095/v1/api/semantic/action \
     -H "Content-Type: application/json" \
     -d '{"@type": "UnsupportedAction", ...}'
   ```

4. **Introspection works**
   ```go
   actions := semantic.GetRegisteredActions()
   // Should return: ["ReplaceAction", "CreateAction", ...]
   ```

## Troubleshooting

### "undefined: semantic.MustRegister"

**Cause:** Using eve.evalgo.org < v0.0.32

**Fix:**
```bash
GOPROXY=direct go get eve.evalgo.org@v0.0.32
go mod tidy
```

### "handler for action type X already registered"

**Cause:** Calling `semantic.MustRegister()` multiple times for the same action type

**Fix:** Ensure each action type is registered only once, typically in `main()`:
```go
// In main() - register once
semantic.MustRegister("ReplaceAction", handleSemanticReplace)

// Don't register again elsewhere
```

### Action not dispatching to handler

**Cause:** Handler not registered or action type mismatch

**Debug:**
```go
// Check if handler is registered
if !semantic.HasHandler("ReplaceAction") {
    logger.Error("ReplaceAction handler not registered")
}

// List all registered handlers
logger.Infof("Registered actions: %v", semantic.GetRegisteredActions())
```

## Performance

The ActionRegistry uses a `sync.RWMutex` for thread-safe access:

- **Read operations** (dispatch): Multiple goroutines can dispatch concurrently
- **Write operations** (register): Exclusive lock required

Typical usage:
- Register handlers once at startup (write lock, negligible cost)
- Dispatch thousands of times per second (read lock, no contention)

**Performance impact:** Negligible (< 1 microsecond per dispatch)

## Summary

The ActionRegistry pattern provides:

✅ **Extensibility** - Add new action types without modifying core code
✅ **Maintainability** - Self-documenting handler registration
✅ **Safety** - Thread-safe, prevents duplicate registration
✅ **Simplicity** - Eliminates brittle switch statements
✅ **Compatibility** - 100% backward compatible

Migration is straightforward:
1. Register handlers in `main()` - one line per action type
2. Replace switch with `semantic.Handle()` - one line change
3. Upgrade to eve.evalgo.org v0.0.32+

See [templateservice](../../templateservice/) for the reference implementation.

## Questions?

For questions or issues with migration, see:
- [ActionRegistry source code](../semantic/actionregistry.go)
- [templateservice reference implementation](../../templateservice/)
- [EVE documentation](../docs/)
