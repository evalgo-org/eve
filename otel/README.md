# EVE OpenTelemetry Integration

Hybrid observability: OpenTelemetry for technical metrics + Semantic Action Tracing for business workflows.

## Quick Start

### 1. Start Jaeger

```bash
cd /home/opunix/eve
docker-compose -f docker-compose.jaeger.yml up -d

# Verify Jaeger is running
curl http://localhost:14269/
```

Access Jaeger UI: http://localhost:16686

### 2. Integrate into Service

```go
package main

import (
    "context"

    "eve.evalgo.org/otel"
    "eve.evalgo.org/tracing"
    "github.com/labstack/echo/v4"
    "go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

func main() {
    // Initialize OpenTelemetry (technical tracing)
    otelProvider := otel.Init("containerservice", "1.0.0")
    if otelProvider != nil {
        defer otelProvider.Shutdown(context.Background())
    }

    // Initialize Semantic Action Tracing (business workflows)
    semanticTracer := tracing.Init(tracing.InitConfig{
        ServiceID:        "containerservice",
        DisableIfMissing: true,
    })

    // Create Echo server
    e := echo.New()

    // Add OpenTelemetry middleware (FIRST - measures everything)
    if otelProvider != nil {
        e.Use(otelecho.Middleware("containerservice"))
    }

    // Add Semantic tracing middleware (SECOND - captures semantic actions)
    if semanticTracer != nil {
        e.Use(semanticTracer.Middleware())
    }

    // Your routes
    e.POST("/v1/api/semantic/action", handleSemanticAction)

    e.Start(":8080")
}
```

## Environment Variables

**OpenTelemetry:**
```bash
# Enable/disable OTel (default: true)
export OTEL_ENABLED=true

# OTLP endpoint (default: http://localhost:4318)
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318

# Service name override
export OTEL_SERVICE_NAME=containerservice

# Sampling ratio 0.0-1.0 (default: 1.0 = trace everything)
export OTEL_SAMPLING_RATIO=0.1

# Environment (production, staging, development)
export OTEL_ENVIRONMENT=production
```

**Semantic Tracing:**
```bash
# PostgreSQL connection for action_traces database
export ACTION_TRACES_DSN=postgresql://claude:password@localhost:5433/action_traces?sslmode=disable

# Enable/disable semantic tracing
export TRACING_ENABLED=true

# Store payloads in S3 (default: false for security)
export TRACING_STORE_PAYLOADS=false

# S3 configuration
export S3_BUCKET=eve-traces
export S3_ENDPOINT_URL=https://s3.hetzner.cloud
export S3_ACCESS_KEY=your-key
export S3_SECRET_KEY=your-secret
```

## How the Hybrid System Works

### Layer 1: OpenTelemetry (Technical)

**What it tracks:**
- HTTP request/response timing
- Database queries
- External API calls
- Errors and stack traces
- Service dependencies

**Where to view:**
- Jaeger UI: http://localhost:16686

**Example query:**
```
# Find slow requests
service="containerservice" AND duration > 1s
```

### Layer 2: Semantic Action Tracing (Business)

**What it tracks:**
- Schema.org action execution (CreateAction, TransferAction, etc.)
- Business metadata (container_id, image, database names)
- Workflow correlation across services
- Credential detection and redaction

**Where to query:**
- MCP tools: `when_get_workflow_trace(correlation_id)`
- PostgreSQL: `SELECT * FROM action_executions WHERE correlation_id = 'wf-123'`

### Bidirectional Linking

**From Semantic → OTel:**
```sql
-- Get OTel trace ID for a workflow
SELECT otel_trace_id FROM action_executions
WHERE correlation_id = 'wf-abc-123'
LIMIT 1;

-- Open in Jaeger:
-- http://localhost:16686/trace/{otel_trace_id}
```

**From OTel → Semantic:**
```sql
-- Find semantic actions for an OTel trace
SELECT correlation_id, operation_id, action_type, metadata
FROM action_executions
WHERE otel_trace_id = '4bf92f3577b34da6a3ce929d0e0e4736';
```

**In MCP tools:**
```
when_get_action_details(operation_id="op-001")

Output:
## OpenTelemetry Link

**Trace ID:** `4bf92f3577b34da6a3ce929d0e0e4736`
**Jaeger:** http://localhost:16686/trace/4bf92f3577b34da6a3ce929d0e0e4736
```

## Sampling Strategies

**OpenTelemetry: Sampled (default 100%)**
```bash
# Sample 10% of requests (reduce storage)
export OTEL_SAMPLING_RATIO=0.1
```

**Semantic Tracing: All workflow actions (no sampling)**
- Business workflows need complete audit trail
- Use exclusions instead: `TRACING_EXCLUDE_ACTIONS=WaitAction`

## Correlation Flow

```
1. Request arrives at Service A
   ├─ OTel creates trace_id=4bf92f...
   ├─ Semantic creates correlation_id=wf-abc-123
   └─ Semantic stores: {otel_trace_id: "4bf92f..."}

2. Service A calls Service B
   ├─ OTel propagates: traceparent: 00-4bf92f...-...
   ├─ Semantic propagates: X-Correlation-ID: wf-abc-123
   └─ Service B links both contexts

3. Query workflow
   ├─ MCP: when_get_workflow_trace("wf-abc-123")
   ├─ Returns all actions with OTel trace links
   └─ Click link → Opens Jaeger UI with technical details
```

## Use Cases

### Technical Debugging

**"Why is this endpoint slow?"**

1. Jaeger: View trace spans
2. Find bottleneck (database query taking 5s)
3. Get correlation_id from span baggage
4. MCP: See which workflow/customer affected

### Business Workflow Tracking

**"What happened to customer workflow wf-123?"**

1. MCP: `when_get_workflow_trace("wf-123")`
2. See all actions, correlation, business metadata
3. Click OTel link for technical details
4. Jaeger: View performance breakdown

### Hybrid Query

**"Find slow container creations and their business context"**

1. Prometheus: `action_duration_seconds{action_type="CreateAction"} > 5`
2. Get otel_trace_id from alert
3. Jaeger: View technical trace
4. PostgreSQL: Get business metadata (which container, which image, who requested)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         HTTP Request                         │
└────────────────────────────┬────────────────────────────────┘
                             │
        ┌────────────────────┴────────────────────┐
        │                                          │
   ┌────▼────┐                              ┌─────▼─────┐
   │  OTel   │◄──────Linked via IDs────────►│ Semantic  │
   │Middleware│                              │ Middleware│
   └────┬────┘                              └─────┬─────┘
        │                                          │
   ┌────▼─────┐                              ┌────▼────┐
   │  Jaeger  │                              │PostgreSQL│
   │  (UI)    │                              │   + S3   │
   └──────────┘                              └──────────┘
     Technical                                  Business
   Observability                               Workflow
```

## Troubleshooting

**OTel traces not appearing in Jaeger:**
```bash
# Check Jaeger is running
curl http://localhost:14269/

# Check OTLP endpoint
curl http://localhost:4318/v1/traces

# Enable debug logging
export OTEL_LOG_LEVEL=debug
```

**Semantic traces missing otel_trace_id:**
- Make sure OTel middleware is added BEFORE semantic middleware
- Check OTEL_ENABLED=true
- Verify otelProvider != nil before adding middleware

**No correlation between traces:**
- Ensure both middlewares are active
- Check PostgreSQL schema has otel_trace_id column
- Apply migration: `psql -d action_traces -f action_tracing_schema.sql`

## Production Recommendations

**OpenTelemetry:**
- Use tail-based sampling (keep errors, sample successes)
- Send to Grafana Tempo for long-term storage
- Set OTEL_SAMPLING_RATIO=0.1 (10% sampling)
- Use HTTPS: OTLP_EXPORTER_OTLP_ENDPOINT=https://tempo:4318

**Semantic Tracing:**
- Keep full workflow traces (no sampling)
- Use exclusions: TRACING_EXCLUDE_ACTIONS for high-volume actions
- Enable S3 lifecycle: 90d → Glacier, 365d → Delete
- Rotate PostgreSQL with TimescaleDB compression

**Linking:**
- Always propagate both contexts (traceparent + X-Correlation-ID)
- Store otel_trace_id even if not storing payloads
- Index otel_trace_id for fast lookups
