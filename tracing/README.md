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
    // Connect to PostgreSQL
    db, _ := sql.Open("postgres", os.Getenv("POSTGRES_DSN"))

    // Create S3 client
    cfg, _ := config.LoadDefaultConfig(context.Background())
    s3Client := s3.NewFromConfig(cfg)

    // Create tracer
    tracer := tracing.New(tracing.Config{
        ServiceID:  "containerservice",
        DB:         db,
        S3Client:   s3Client,
        S3Bucket:   "eve-traces",
        Enabled:    os.Getenv("TRACING_ENABLED") != "false",
    })

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
