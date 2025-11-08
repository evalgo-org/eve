# EVE Action Tracing Package

Middleware for tracing Schema.org action execution across EVE services using hybrid S3 + PostgreSQL storage.

## Features

- Automatic correlation ID propagation
- Hybrid storage: Metadata in PostgreSQL, payloads in S3
- Polymorphic metadata extraction based on action type
- Asynchronous trace recording
- Support for logs and artifacts upload

## Installation

```go
import "eve.evalgo.org/tracing"
```

## Usage

### Basic Setup

```go
package main

import (
    "database/sql"
    "os"

    "eve.evalgo.org/tracing"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/labstack/echo/v4"
    _ "github.com/lib/pq"
)

func main() {
    // Connect to PostgreSQL (action_traces database)
    db, _ := sql.Open("postgres", os.Getenv("ACTION_TRACES_DSN"))

    // Create S3 client
    cfg, _ := config.LoadDefaultConfig(context.Background())
    s3Client := s3.NewFromConfig(cfg)

    // Option 1: Manual configuration
    tracer := tracing.New(tracing.Config{
        ServiceID:  "containerservice",
        DB:         db,
        S3Client:   s3Client,
        S3Bucket:   "eve-traces",
        Enabled:    true,
        StorePayloads: false,
        ExcludeActionTypes: []string{"WaitAction"},
    })

    // Option 2: Environment variable configuration (recommended)
    tracer := tracing.NewFromEnv("containerservice", db, s3Client)

    // Create Echo server
    e := echo.New()

    // Add tracing middleware (must be early in chain)
    e.Use(tracer.Middleware())

    // Your routes
    e.POST("/v1/api/semantic/action", handleSemanticAction)

    e.Start(":8080")
}
```

### Service-to-Service Calls

When calling another service, propagate correlation headers:

```go
func callDownstreamService(c echo.Context, action Action) {
    req, _ := http.NewRequest("POST", "http://s3service:8092/v1/api/semantic/action", body)

    // Propagate correlation context
    tracing.PropagateFromContext(c, req)

    resp, _ := http.DefaultClient.Do(req)
}
```

### Uploading Logs

Services can upload execution logs to S3:

```go
func handleBuildAction(c echo.Context, tracer *tracing.Tracer) error {
    correlationID := tracing.GetCorrelationID(c)
    operationID := tracing.GetOperationID(c)

    // Run build and capture logs
    logs := runBuild()

    // Upload logs to S3
    tracer.UploadLogs(ctx, correlationID, operationID, logs)
}
```

### Uploading Artifacts

Services can upload build artifacts:

```go
func uploadArtifacts(c echo.Context, tracer *tracing.Tracer) {
    correlationID := tracing.GetCorrelationID(c)
    operationID := tracing.GetOperationID(c)

    // Upload build artifacts
    tracer.UploadArtifact(ctx, correlationID, operationID, "app.zip", zipData)
    tracer.UploadArtifact(ctx, correlationID, operationID, "test-results.xml", xmlData)
}
```

## Selective Tracing

### Method 1: Environment Variables (Recommended)

Use environment variables for flexible configuration:

```bash
# Enable/disable tracing (default: true)
export TRACING_ENABLED=true

# Store payloads in S3 (default: false)
export TRACING_STORE_PAYLOADS=false

# Exclude specific action types (comma-separated)
export TRACING_EXCLUDE_ACTIONS="WaitAction,SearchAction"

# Exclude specific object types (comma-separated)
export TRACING_EXCLUDE_OBJECTS="Database,DataFeed"

# S3 configuration
export S3_BUCKET=eve-traces
export S3_ENDPOINT_URL=https://s3.hetzner.cloud
```

Then use `NewFromEnv()`:

```go
tracer := tracing.NewFromEnv("containerservice", db, s3Client)
e.Use(tracer.Middleware())
```

### Method 2: Config Struct

Exclude actions via Config:

```go
tracer := tracing.New(tracing.Config{
    // ...

    // Exclude WaitAction and SearchAction from tracing
    ExcludeActionTypes: []string{"WaitAction", "SearchAction"},

    // Exclude Database and DataFeed objects from tracing
    ExcludeObjectTypes: []string{"Database", "DataFeed"},
})
```

### Method 3: Action-Level Metadata (Most Granular)

Control tracing per action using the `meta` field in JSON-LD:

```json
{
  "@context": "https://schema.org",
  "@type": "CreateAction",
  "object": {
    "@type": "Secret",
    "name": "API_KEY",
    "value": "secret-value"
  },
  "meta": {
    "trace": false,
    "tracePayload": false
  }
}
```

**Meta flags**:
- `trace: false` - Skip tracing this action entirely
- `tracePayload: false` - Trace metadata only, don't store payload in S3

**Priority order** (highest to lowest):
1. Credential detection (always redacted)
2. Action `meta.trace: false` (skips tracing)
3. Action `meta.tracePayload: false` (skips payload only)
4. Config `ExcludeActionTypes` / `ExcludeObjectTypes`
5. Config `StorePayloads`

### Excluding Actions from Tracing (Legacy)

You can exclude specific action types or object types from being traced:

```go
tracer := tracing.New(tracing.Config{
    // ...

    // Exclude WaitAction and SearchAction from tracing
    ExcludeActionTypes: []string{"WaitAction", "SearchAction"},

    // Exclude Database and DataFeed objects from tracing
    ExcludeObjectTypes: []string{"Database", "DataFeed"},
})
```

When an action is excluded, it will not be recorded in the database at all.

### Credential Security

**IMPORTANT**: The tracer automatically detects and redacts credential-related actions to prevent storing sensitive data.

Actions with the following object types are **automatically redacted**:
- `Credential`
- `PasswordCredential`
- `Secret`
- `DigitalDocument` (may contain credentials)

For redacted actions:
- ✓ Metadata is still recorded in PostgreSQL (timing, status, service)
- ✗ Request/response payloads are **NOT** uploaded to S3
- ✓ S3 URLs are marked as `[REDACTED - Credential payload not stored]`

Example redacted trace:

```
## Action Details: op-abc123

**Service:** infisicalservice
**Action Type:** CreateAction
**Object Type:** Secret
**Status:** CompletedActionStatus

## Storage

**Request:** [REDACTED - Credential payload not stored]
**Response:** [REDACTED - Credential payload not stored]
```

This ensures credentials never leave the service, even in trace data.

### Payload Storage Control

Control whether request/response payloads are stored in S3:

```go
tracer := tracing.New(tracing.Config{
    // ...

    // Enable payload storage (disabled by default for security)
    StorePayloads: true,
})
```

Even when `StorePayloads: true`, credential payloads are **always redacted**.

## Metadata Extraction

The tracer automatically extracts queryable metadata based on action + object type:

| Action Type | Object Type | Extracted Metadata |
|-------------|-------------|--------------------|
| CreateAction | SoftwareApplication | container_id, image, ports, health_status |
| TransferAction | Database | source_database, target_database, progress_percent |
| UploadAction | Dataset | backup_type, checksum, storage_location, expires_at |
| ExecuteAction | SoftwareSourceCode | repository, commit_sha, tests_passed, tests_failed |
| ReplaceAction | DataFeed | input_rows, output_rows, data_quality_passed |

## Configuration

Environment variables:

- `POSTGRES_DSN`: PostgreSQL connection string for **action_traces database** (required)
  - Example: `postgresql://claude:password@localhost:5433/action_traces?sslmode=disable`
  - ⚠️ Must point to `action_traces`, not `claude_metrics` or `when_metrics`!
- `S3_ENDPOINT_URL`: S3 endpoint URL (for Hetzner/MinIO)
- `S3_ACCESS_KEY`: S3 access key
- `S3_SECRET_KEY`: S3 secret key
- `S3_BUCKET`: S3 bucket name (default: eve-traces)
- `TRACING_ENABLED`: Enable/disable tracing (default: true)
- `TRACING_STORE_PAYLOADS`: Store request/response in S3 (default: false for security)

### Config Options

```go
type Config struct {
    ServiceID          string    // Service identifier (required)
    DB                 *sql.DB   // PostgreSQL connection (required)
    S3Client           *s3.Client // S3 client (required if StorePayloads=true)
    S3Bucket           string    // S3 bucket name
    S3Endpoint         string    // S3 endpoint URL
    Enabled            bool      // Enable/disable tracing
    ExcludeActionTypes []string  // Action types to exclude from tracing
    ExcludeObjectTypes []string  // Object types to exclude from tracing
    StorePayloads      bool      // Store request/response in S3 (false by default)
}
```

**Security Notes**:
- `StorePayloads` defaults to `false` to prevent accidental credential leakage
- Credential-related actions are **always redacted** regardless of `StorePayloads` setting
- Use `ExcludeObjectTypes: []string{"Credential", "Secret"}` to completely skip tracing credential operations

## Database Setup

⚠️ **IMPORTANT**: Action tracing uses a SEPARATE database!

Apply the action tracing schema to the `action_traces` database:

```bash
# Create the database
createdb -U claude action_traces

# Apply schema
psql -U claude -d action_traces -f /home/opunix/when/action_tracing_schema.sql
```

**Do NOT use claude_metrics or when_metrics for action tracing!**

See `/home/opunix/when/DATABASE_SEPARATION.md` for details.

## S3 Setup

Create the eve-traces bucket with lifecycle policy (see documentation).

## Querying Traces

See memory-mcp notes for query examples and MCP tools.
