# EVE Distributed Tracing - Quick Start Guide

Get the entire EVE observability stack running in 5 minutes!

## Prerequisites

- Docker and Docker Compose installed
- 8GB RAM available
- Ports 3000, 5432, 8080, 9000-9001, 9090-9093, 3100 available

## 1. Start the Stack

```bash
# Option 1: Using Make (recommended)
make docker-up

# Option 2: Using Docker Compose directly
docker-compose up -d
```

Wait 30 seconds for all services to initialize.

## 2. Verify Services

```bash
# Run smoke tests
make docker-test

# Or check manually
curl http://localhost:8080/health
```

## 3. Access the Dashboards

### Grafana - Primary Dashboard
```bash
# Open in browser
make grafana

# Or manually: http://localhost:3000
# Login: admin / admin
```

**Available Dashboards:**
1. **EVE / Operational** - SRE/DevOps monitoring
   - Request rate, errors, latency
   - Queue metrics, traces dropped
   - Database and S3 health

2. **EVE / Business** - Product metrics
   - Workflow success rates
   - Duration percentiles (p50, p95, p99)
   - Throughput by service
   - SLO tracking

3. **EVE / Compliance** - GDPR/Privacy
   - Erasure requests
   - PII detection rates
   - Audit access tracking
   - Data residency compliance

### Prometheus - Metrics
```bash
make prometheus
# http://localhost:9090
```

Example queries:
```promql
# Request rate
rate(eve_tracing_actions_total[5m])

# Error rate percentage
rate(eve_tracing_action_errors_total[5m]) / rate(eve_tracing_actions_total[5m]) * 100

# P95 latency
histogram_quantile(0.95, rate(eve_tracing_action_duration_seconds_bucket[5m]))
```

### AlertManager - Alerts
```bash
make alertmanager
# http://localhost:9093
```

43 pre-configured alerts for operational, business, and compliance monitoring.

### MinIO - S3 Storage
```bash
make minio
# http://localhost:9001
# Login: minioadmin / minioadmin
```

Browse trace payloads in the `eve-traces` bucket.

## 4. Generate Traffic

Create realistic traffic patterns:

```bash
# Generate continuous traffic
make docker-traffic
```

This creates:
- 90% normal requests (sampled at 10%)
- 10% slow requests (>5s, always sampled)
- ~10% error requests (always sampled)

## 5. View Traces

### In PostgreSQL
```bash
# Connect to database
make postgres

# Query recent traces
SELECT correlation_id, operation_id, service_id, action_type,
       action_status, duration_ms, started_at
FROM action_executions
ORDER BY started_at DESC
LIMIT 10;

# Get workflow trace
SELECT * FROM get_workflow_trace('wf-demo-001');

# Get workflow stats
SELECT * FROM get_workflow_stats('wf-demo-001');
```

### Using Make Command
```bash
# Quick view of recent traces
make traces
```

## 6. Explore Features

### Async Export
Traces are exported asynchronously with:
- 10,000 queue size
- 100 batch size
- 4 worker goroutines
- 10-second flush interval

Monitor queue size in Grafana:
```promql
eve_tracing_exporter_queue_size
```

### Sampling
- **Base Rate**: 10% of normal traffic
- **Always Sample Errors**: 100% of failed requests
- **Always Sample Slow**: 100% of requests >5000ms
- **Deterministic**: Same correlation_id always makes same decision

View sampling decisions:
```promql
rate(eve_tracing_sampling_decisions_total[5m])
```

### Log Correlation
Logs automatically include:
- `correlation_id` - Workflow identifier
- `operation_id` - Action identifier
- `trace_id` - OpenTelemetry trace ID
- `span_id` - OpenTelemetry span ID

View in Loki (Grafana → Explore → Loki):
```logql
{service="example-service"} |= "correlation_id"
```

## 7. Test Scenarios

### Normal Workflow
```bash
curl -X POST http://localhost:8080/v1/api/workflow/create
```

Response:
```json
{
  "status": "completed",
  "correlation_id": "wf-xxx",
  "operation_id": "op-yyy",
  "message": "Workflow created successfully"
}
```

### Slow Workflow (Triggers Sampling)
```bash
curl -X POST http://localhost:8080/v1/api/workflow/slow
```

Always sampled due to >5s duration.

### Error Workflow (Triggers Sampling)
```bash
curl -X POST http://localhost:8080/v1/api/workflow/error
```

Always sampled due to failed status.

## 8. View Metrics

```bash
# In terminal
make metrics

# Or with curl
curl http://localhost:9091/metrics | grep eve_tracing
```

Key metrics:
- `eve_tracing_actions_total` - Total actions executed
- `eve_tracing_action_duration_seconds` - Latency histogram
- `eve_tracing_exporter_queue_size` - Current queue size
- `eve_tracing_sampling_decisions_total` - Sampling decisions

## 9. Monitor Logs

View structured JSON logs:
```bash
# Service logs
docker-compose logs -f example-service

# All logs
make docker-logs
```

## 10. Shutdown

```bash
# Stop services (keep data)
make docker-down

# Stop and remove all data
make docker-clean
```

## Architecture Overview

```
┌─────────────────┐
│ Example Service │
│   Port 8080     │ ──► HTTP API
│   Port 9091     │ ──► Metrics
└────────┬────────┘
         │
         ├──► Traces ──► PostgreSQL (metadata)
         │              + MinIO (payloads)
         │
         ├──► Metrics ──► Prometheus
         │                   │
         │                   ▼
         │              AlertManager
         │
         └──► Logs ──► Promtail ──► Loki
                                      │
                                      ▼
                                  Grafana
```

## Features Demonstrated

✅ **1. Async Trace Export**
- Non-blocking trace storage
- Batched writes to PostgreSQL
- Configurable queue and worker count

✅ **2. Prometheus Metrics**
- 23 metrics across 8 categories
- Request rate, latency, errors
- Queue metrics, sampling decisions

✅ **3. Grafana Dashboards**
- Operational: SRE monitoring
- Business: Product metrics
- Compliance: GDPR/privacy

✅ **4. AlertManager**
- 43 pre-configured alert rules
- Operational, business, compliance
- Email template included

✅ **5. Log Correlation**
- Structured JSON logs
- Automatic trace context injection
- Correlation ID in all logs

✅ **6. Tail-Based Sampling**
- Always sample errors and slow requests
- Configurable base rate
- Deterministic workflow sampling

✅ **7. Service Dependency Mapping**
- Auto-discovery from traces
- Circular dependency detection
- Graphviz visualization

✅ **8. Trace Archival**
- S3 Glacier for long-term storage
- 65% cost reduction
- GDPR-compliant restore workflow

## Common Operations

### View Service Status
```bash
make status
```

### Restart Services
```bash
make docker-restart
```

### View Specific Service Logs
```bash
docker-compose logs -f postgres
docker-compose logs -f prometheus
docker-compose logs -f example-service
```

### Connect to Services
```bash
# PostgreSQL
make postgres

# MinIO Console
make minio

# Grafana
make grafana
```

## Troubleshooting

### Services won't start
```bash
# Check logs
docker-compose logs

# Check available ports
netstat -tulpn | grep -E '(3000|5432|8080|9090|9093)'

# Restart
make docker-restart
```

### No metrics in Grafana
```bash
# Check Prometheus targets
curl http://localhost:9090/targets

# Check metrics endpoint
curl http://localhost:9091/metrics

# Verify datasource in Grafana
open http://localhost:3000/datasources
```

### Database connection failed
```bash
# Check PostgreSQL health
docker exec eve-postgres pg_isready -U eve_user

# View initialization logs
docker-compose logs postgres | grep "initialized"

# Manually connect
docker exec -it eve-postgres psql -U eve_user -d eve_traces
```

## Next Steps

1. **Add More Services**: Integrate containerservice, workflowstorageservice
2. **Configure Archival**: Setup S3 Glacier lifecycle policies
3. **Add Dependency Mapping**: Visualize service dependencies
4. **Configure Alerts**: Add email/Slack webhooks to AlertManager
5. **Production Deploy**: Move to Kubernetes with Helm charts

## Documentation

- **Main README**: `/home/opunix/eve/README.md`
- **Docker Setup**: `/home/opunix/eve/docker/README.md`
- **Tracing Package**: `/home/opunix/eve/tracing/`
- **Grafana Dashboards**: `/home/opunix/eve/grafana/`
- **AlertManager**: `/home/opunix/eve/grafana/alertmanager/`

## Support

For issues or questions:
1. Check logs: `make docker-logs`
2. Run tests: `make docker-test`
3. View status: `make status`

---

**You're now running a state-of-the-art distributed tracing system! 🎉**

Access Grafana at http://localhost:3000 (admin/admin) to see your traces in action.
