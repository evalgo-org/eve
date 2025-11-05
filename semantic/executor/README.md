# EVE Semantic Executor

Universal semantic action execution engine for EVE-based projects.

## Overview

The semantic executor provides a unified, registry-based execution system for Schema.org semantic actions. It automatically routes actions to appropriate handlers based on action type and target URL, with support for service discovery via registry protocol.

## Features

- **Universal Service Routing**: Automatically routes to any service with `/v1/api/semantic/action` endpoint
- **Registry-Based Discovery**: Resolves `registry://servicename/path` URLs via registry service
- **Environment Variable Expansion**: Supports `${ENV:VARIABLE}` placeholders
- **Command Execution**: Executes shell commands via `command` property
- **HTTP Actions**: Delegates to fetcher for HTTP-based actions
- **Priority-Based Dispatch**: Multiple executors with priority-based selection
- **JSON-LD Native**: Full Schema.org semantic action support

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "eve.evalgo.org/semantic"
    "eve.evalgo.org/semantic/executor"
)

func main() {
    // Create registry with default executors
    registry := executor.NewRegistry()

    // Create a semantic action
    action := &semantic.SemanticScheduledAction{
        Type: "RetrieveAction",
        Name: "Get secret from Infisical",
        Target: map[string]interface{}{
            "additionalProperty": map[string]interface{}{
                "url": "registry://infisicalservice/v1/api/semantic/action",
            },
        },
        Object: &semantic.SemanticObject{
            Identifier: "MY_SECRET",
        },
    }

    // Execute
    output, err := registry.Execute(action)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(output)
}
```

## Architecture

### Registry Pattern

The `Registry` manages multiple `Executor` implementations with priority-based dispatch:

1. **URLBasedExecutor** (highest priority): Routes to semantic service endpoints
2. **ScheduledActionExecutor**: Handles ScheduledAction wrappers and HTTP actions
3. **CommandExecutor**: Executes shell commands (fallback)

```go
type Executor interface {
    Execute(action *semantic.SemanticScheduledAction) (string, error)
    CanHandle(action *semantic.SemanticScheduledAction) bool
}
```

### Executors

#### URLBasedExecutor

Routes actions to semantic services:

```go
action := &semantic.SemanticScheduledAction{
    Type: "CreateAction",
    Target: map[string]interface{}{
        "additionalProperty": map[string]interface{}{
            "url": "registry://s3service/v1/api/semantic/action",
        },
    },
    Object: &semantic.SemanticObject{
        Name: "my-file.txt",
        Text: "file contents",
    },
}
```

Supported services (any with `/v1/api/semantic/action`):
- infisicalservice (secrets management)
- s3service (S3 storage)
- basexservice (XML database)
- sparqlservice (SPARQL queries)
- templateservice (template rendering)
- workflowstorageservice (workflow storage)

#### ScheduledActionExecutor

Handles ScheduledAction wrappers with embedded actions:

```go
action := &semantic.SemanticScheduledAction{
    Type: "ScheduledAction",
    Object: &semantic.SemanticObject{
        Text: `{"@type":"RetrieveAction",...}`, // Embedded action
    },
}
```

Delegates to fetcher for HTTP actions.

#### CommandExecutor

Executes shell commands:

```go
action := &semantic.SemanticScheduledAction{
    Type: "ActivateAction",
    Properties: map[string]interface{}{
        "command": "ls -la /tmp",
    },
}
```

### Registry URL Resolution

Resolves `registry://servicename/path` to actual service URLs:

```
registry://infisicalservice/v1/api/semantic/action
    ↓ (queries REGISTRYSERVICE_API_URL)
http://localhost:8093/v1/api/semantic/action
```

**Environment Variable:**
```bash
export REGISTRYSERVICE_API_URL=http://localhost:8096
```

**Service Discovery:**
- Queries `/v1/api/services` on registry service
- Caches results for 5 minutes
- Matches by `identifier` field

### Environment Variable Expansion

Expands `${ENV:VARIABLE}` placeholders before execution:

```go
action := &semantic.SemanticScheduledAction{
    Type: "RetrieveAction",
    Object: &semantic.SemanticObject{
        Identifier: "${ENV:SECRET_NAME}", // Expanded to os.Getenv("SECRET_NAME")
    },
}
```

Pattern: `${ENV:[A-Z_][A-Z0-9_]*}`

## Usage Examples

### Retrieve Secret from Infisical

```go
registry := executor.NewRegistry()

action := &semantic.SemanticScheduledAction{
    Type: "RetrieveAction",
    Name: "Get database password",
    Target: map[string]interface{}{
        "additionalProperty": map[string]interface{}{
            "url": "registry://infisicalservice/v1/api/semantic/action",
        },
    },
    Object: &semantic.SemanticObject{
        Identifier: "DB_PASSWORD",
    },
}

secret, err := registry.Execute(action)
```

### Upload File to S3

```go
action := &semantic.SemanticScheduledAction{
    Type: "CreateAction",
    Name: "Upload report to S3",
    Target: map[string]interface{}{
        "additionalProperty": map[string]interface{}{
            "url": "registry://s3service/v1/api/semantic/action",
        },
    },
    Object: &semantic.SemanticObject{
        Name: "report.pdf",
        ContentUrl: "s3://my-bucket/reports/report.pdf",
        Text: "file contents here",
    },
}

result, err := registry.Execute(action)
```

### Execute Shell Command

```go
action := &semantic.SemanticScheduledAction{
    Type: "ActivateAction",
    Name: "Check disk space",
    Properties: map[string]interface{}{
        "command": "df -h",
    },
}

output, err := registry.Execute(action)
```

### HTTP Action via Fetcher

```go
action := &semantic.SemanticScheduledAction{
    Type: "SearchAction",
    Object: &semantic.SemanticObject{
        CodeRepository: "https://api.example.com/search?q=test",
    },
}

result, err := registry.Execute(action)
```

## Custom Executors

Implement the `Executor` interface and register:

```go
type MyExecutor struct{}

func (e *MyExecutor) CanHandle(action *semantic.SemanticScheduledAction) bool {
    return action.Type == "MyCustomAction"
}

func (e *MyExecutor) Execute(action *semantic.SemanticScheduledAction) (string, error) {
    // Custom execution logic
    return "result", nil
}

// Register (prepended for priority)
registry := executor.NewRegistry()
registry.Register(&MyExecutor{})
```

## Semantic Introspection

Analyze actions before execution:

```go
info := executor.IntrospectAction(action)
// Returns: map with id, name, type, object, target, dependencies, status, etc.

// Query actions by type
retrieveActions := executor.QueryActionsByType(actions, "RetrieveAction")

// Query actions by URL pattern
s3Actions := executor.QueryActionsByURL(actions, "s3://")

// Export as JSON-LD
jsonld, err := executor.ExportAsJSONLD(action)
```

## Service Integration

Services expecting semantic actions should implement `/v1/api/semantic/action` endpoint:

```go
// POST /v1/api/semantic/action
func HandleSemanticAction(w http.ResponseWriter, r *http.Request) {
    var action semantic.SemanticScheduledAction
    if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Process action based on type
    switch action.Type {
    case "RetrieveAction":
        // Handle retrieval
    case "CreateAction":
        // Handle creation
    default:
        http.Error(w, "Unsupported action type", http.StatusBadRequest)
        return
    }

    // Return result
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}
```

## Migration from Project-Specific Executors

### Before (project-specific)

```go
// In your project
output, err := Execute(action)
```

### After (EVE semantic executor)

```go
import "eve.evalgo.org/semantic/executor"

// Create registry once (reusable)
registry := executor.NewRegistry()

// Execute actions
output, err := registry.Execute(action)
```

## Configuration

**Environment Variables:**

- `REGISTRYSERVICE_API_URL`: Registry service URL (default: `http://localhost:8096`)

**Registry Cache:**
- TTL: 5 minutes
- Thread-safe with RWMutex
- Automatic refresh on expiry

## Error Handling

```go
output, err := registry.Execute(action)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "no executor available"):
        // No executor can handle this action type
    case strings.Contains(err.Error(), "service returned"):
        // HTTP error from service
    case strings.Contains(err.Error(), "failed to resolve"):
        // Registry URL resolution failed
    case strings.Contains(err.Error(), "command failed"):
        // Shell command execution failed
    default:
        // Other error
    }
}
```

## Benefits

1. **Unified Execution**: Single API for all semantic actions
2. **Service Discovery**: Automatic routing via registry protocol
3. **Decoupling**: Services don't know about each other
4. **Extensibility**: Easy to add custom executors
5. **JSON-LD Native**: Full Schema.org support
6. **Environment Aware**: Dynamic configuration via environment variables
7. **Type Safe**: Strong typing with Go structs

## Projects Using This Executor

- **when**: Task scheduler (original implementation)
- **graphium**: Container orchestration
- **fetcher**: HTTP fetcher
- All future EVE-based projects

## Version

v0.0.22 - Unified semantic executor from `when` project

## See Also

- [Schema.org Actions](https://schema.org/Action)
- [EVE Semantic Package](../README.md)
- [JSON-LD Specification](https://json-ld.org/)
