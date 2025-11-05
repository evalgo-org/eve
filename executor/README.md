# EVE Executor Package

Unified execution interface for semantic actions across all EVE-based projects.

## Overview

The `executor` package provides a standardized way to execute semantic actions (Schema.org based) with support for multiple execution types, retry logic, and extensible architecture.

## Features

- **Unified Interface**: Single `Executor` interface for all execution types
- **Registry Pattern**: Pluggable executor implementations with automatic selection
- **Context Support**: Full context.Context integration for cancellation and timeouts
- **Structured Results**: Rich `Result` type with metadata, timing, and error details
- **Built-in Executors**:
  - HTTPExecutor: Execute HTTP-based semantic actions
  - CommandExecutor: Execute shell commands
- **Extensible**: Easy to add custom executors

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "time"

    "eve.evalgo.org/executor"
    "eve.evalgo.org/semantic"
)

func main() {
    // Create a registry
    registry := executor.NewRegistry()

    // Register executors
    registry.Register(executor.NewHTTPExecutor())
    registry.Register(executor.NewCommandExecutor())

    // Create an action
    action := &semantic.SemanticScheduledAction{
        Type: "RetrieveAction",
        Object: &semantic.SemanticObject{
            ContentUrl: "https://api.example.com/data",
        },
    }

    // Execute with context
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    result, err := registry.Execute(ctx, action)
    if err != nil {
        fmt.Printf("Execution failed: %v\n", err)
        return
    }

    fmt.Printf("Status: %s\n", result.Status)
    fmt.Printf("Output: %s\n", result.Output)
    fmt.Printf("Duration: %v\n", result.Duration)
}
```

## Action Types

### HTTP Actions

The HTTP executor supports these action types:
- `SearchAction` → GET
- `RetrieveAction` → GET
- `SendAction` → POST
- `CreateAction` → POST
- `UpdateAction` → PUT
- `ReplaceAction` → PUT
- `DeleteAction` → DELETE

Example:
```go
action := &semantic.SemanticScheduledAction{
    Type: "CreateAction",
    Object: &semantic.SemanticObject{
        ContentUrl: "https://api.example.com/users",
        Text: `{"name": "John Doe"}`,
        EncodingFormat: "application/json",
    },
}
```

### Command Actions

The command executor handles shell command execution:
- URLs starting with `exec://`, `command://`, or `shell://`

Example:
```go
action := &semantic.SemanticScheduledAction{
    Type: "Action",
    Object: &semantic.SemanticObject{
        ContentUrl: "exec://ls -la /tmp",
    },
}
```

## Custom Executors

Create custom executors by implementing the `Executor` interface:

```go
type MyExecutor struct {
    // Your fields
}

func (e *MyExecutor) Name() string {
    return "my-executor"
}

func (e *MyExecutor) CanHandle(action *semantic.SemanticScheduledAction) bool {
    // Return true if this executor can handle the action
    return action.Object != nil && strings.HasPrefix(action.Object.ContentUrl, "myprotocol://")
}

func (e *MyExecutor) Execute(ctx context.Context, action *semantic.SemanticScheduledAction) (*Result, error) {
    result := &Result{
        StartTime: time.Now(),
        Status: StatusRunning,
        Metadata: make(map[string]interface{}),
    }

    // Your execution logic here

    result.Status = StatusCompleted
    result.Output = "execution result"
    result.EndTime = time.Now()
    result.Duration = result.EndTime.Sub(result.StartTime)

    return result, nil
}
```

Then register it:
```go
registry.Register(&MyExecutor{})
```

## Result Structure

```go
type Result struct {
    Output    string                 // Primary execution result
    Status    ExecutionStatus        // pending, running, completed, failed, cancelled
    Metadata  map[string]interface{} // Additional execution information
    Error     *ExecutionError        // Detailed error if failed
    StartTime time.Time              // When execution began
    EndTime   time.Time              // When execution completed
    Duration  time.Duration          // Execution duration
}
```

## Migration from Existing Code

### Before (project-specific Execute)
```go
func Execute(req *Request) (string, error) {
    resp, err := http.Get(req.URL)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    return string(body), nil
}
```

### After (using eve/executor)
```go
func Execute(req *Request) (string, error) {
    registry := executor.NewRegistry()
    registry.Register(executor.NewHTTPExecutor())

    action := &semantic.SemanticScheduledAction{
        Type: "RetrieveAction",
        Object: &semantic.SemanticObject{
            ContentUrl: req.URL,
        },
    }

    result, err := registry.Execute(context.Background(), action)
    if err != nil {
        return "", err
    }

    return result.Output, nil
}
```

## Projects Using This Package

This executor package consolidates Execute implementations from:
- claude-tools
- fetcher
- graphium
- http2amqp
- pg-queue
- px-containers
- pxgraphservice
- pxtool
- when

## Advanced Features (Coming Soon)

- Retry policies with exponential/linear backoff
- Execution hooks (before/after/error)
- Storage abstraction for persistence
- Dependency resolution
- Workflow execution

## License

Part of the EVE library - see main LICENSE file.
