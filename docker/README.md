# EVE Distributed Tracing - Docker Compose Setup

Complete observability stack for EVE distributed tracing system.

## Quick Start

```bash
# Start the entire stack
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the stack
docker-compose down

# Stop and remove all data
docker-compose down -v
```

## Services

The Docker Compose stack includes:

### Data Storage
- **PostgreSQL** (port 5432) - TimescaleDB for trace metadata
  - Database: `eve_traces`
  - User: `eve_user`
  - Password: `eve_password`

- **MinIO** (ports 9000, 9001) - S3-compatible storage for trace payloads
  - Console: http://localhost:9001
  - User: `minioadmin`
  - Password: `minioadmin`
  - Bucket: `eve-traces`

### Monitoring & Observability
- **Prometheus** (port 9090) - Metrics collection
  - UI: http://localhost:9090
  - Scrapes metrics from example-service every 15s

- **AlertManager** (port 9093) - Alert routing
  - UI: http://localhost:9093
  - 43 pre-configured alert rules

- **Grafana** (port 3000) - Dashboards and visualization
  - UI: http://localhost:3000
  - User: `admin`
  - Password: `admin`
  - 3 pre-loaded dashboards:
    - Operational (SRE/DevOps)
    - Business (Product metrics)
    - Compliance (GDPR/Privacy)

- **Loki** (port 3100) - Log aggregation
  - 30-day retention
  - Automatic compaction

- **Promtail** - Log shipping to Loki

### Example Service
- **example-service** (ports 8080, 9091)
  - REST API: http://localhost:8080
  - Metrics: http://localhost:9091/metrics
  - Demonstrates EVE tracing features

## Access URLs

| Service | URL | Credentials |
|---------|-----|-------------|
| Grafana | http://localhost:3000 | admin / admin |
| Prometheus | http://localhost:9090 | - |
| AlertManager | http://localhost:9093 | - |
| MinIO Console | http://localhost:9001 | minioadmin / minioadmin |
| Example Service | http://localhost:8080 | - |
| Metrics | http://localhost:9091/metrics | - |

## Example Service API

The example service provides several endpoints to demonstrate tracing:

```bash
# Home / API documentation
curl http://localhost:8080/

# Create a normal workflow (fast, sampled at 10% base rate)
curl -X POST http://localhost:8080/v1/api/workflow/create

# Create a slow workflow (>5s, always sampled)
curl -X POST http://localhost:8080/v1/api/workflow/slow

# Create a failing workflow (always sampled)
curl -X POST http://localhost:8080/v1/api/workflow/error

# Health check
curl http://localhost:8080/health

# View Prometheus metrics
curl http://localhost:9091/metrics
```

## Viewing Traces

### In Grafana

1. Open http://localhost:3000
2. Login with admin/admin
3. Navigate to Dashboards → EVE folder
4. Open one of the 3 dashboards:
   - **Operational** - Request rate, errors, latency, queue metrics
   - **Business** - Workflow execution patterns, success rates
   - **Compliance** - GDPR metrics, PII detection, audit access

### In PostgreSQL

```bash
# Connect to PostgreSQL
docker exec -it eve-postgres psql -U eve_user -d eve_traces

# View all traces
SELECT correlation_id, operation_id, service_id, action_type, action_status, duration_ms
FROM action_executions
ORDER BY started_at DESC
LIMIT 10;

# Get workflow trace
SELECT * FROM get_workflow_trace('wf-demo-001');

# Get workflow statistics
SELECT * FROM get_workflow_stats('wf-demo-001');
```

### In Prometheus

1. Open http://localhost:9090
2. Go to Graph tab
3. Example queries:
   ```promql
   # Request rate
   rate(eve_tracing_actions_total[5m])

   # Error rate
   rate(eve_tracing_action_errors_total[5m])

   # Average latency (p95)
   histogram_quantile(0.95, rate(eve_tracing_action_duration_seconds_bucket[5m]))

   # Queue size
   eve_tracing_exporter_queue_size

   # Sampling decisions
   rate(eve_tracing_sampling_decisions_total[5m])
   ```

### In Loki (Logs)

1. Open http://localhost:3000
2. Go to Explore
3. Select Loki datasource
4. Example queries:
   ```logql
   # All logs from example-service
   {service="example-service"}

   # Error logs only
   {service="example-service", level="error"}

   # Logs for specific correlation_id
   {service="example-service"} |= "wf-demo-001"

   # Action execution logs
   {service="example-service", event_type="action_execution"}
   ```

## Generate Test Traffic

Create a script to generate continuous traffic:

```bash
#!/bin/bash
# generate-traffic.sh

while true; do
  # Normal requests (90% of traffic)
  for i in {1..9}; do
    curl -X POST http://localhost:8080/v1/api/workflow/create &
  done

  # Slow request (sampled)
  curl -X POST http://localhost:8080/v1/api/workflow/slow &

  # Random error (sampled)
  if [ $((RANDOM % 10)) -eq 0 ]; then
    curl -X POST http://localhost:8080/v1/api/workflow/error &
  fi

  sleep 1
done
```

Run it:
```bash
chmod +x generate-traffic.sh
./generate-traffic.sh
```

## Monitoring Alerts

AlertManager is configured with 43 alert rules across 3 categories:

### Operational Alerts (12 rules)
- Queue overflow (CRITICAL)
- Traces dropped (CRITICAL)
- High error rate (CRITICAL)
- Database failures (CRITICAL)
- S3 failures (CRITICAL)
- Queue high (WARNING)
- High latency (WARNING)

### Business Alerts (15 rules)
- Low success rate <90% (CRITICAL)
- SLO violation p95 >120s (CRITICAL)
- Degraded success <95% (WARNING)
- Slow workflows (WARNING)
- High throughput (INFO)

### Compliance Alerts (16 rules)
- Data residency violations (CRITICAL)
- Unredacted PII (CRITICAL)
- Erasure failures (CRITICAL)
- Low redaction rate (WARNING)
- Slow erasure (WARNING)
- Unauthorized access (WARNING)

View active alerts:
```bash
# AlertManager UI
open http://localhost:9093

# Or via API
curl http://localhost:9093/api/v2/alerts
```

## Architecture

```
┌─────────────────┐
│ Example Service │ ──► Traces ──► PostgreSQL (metadata)
│   (Port 8080)   │                      │
└────────┬────────┘                      │
         │                               │
         ├──► Metrics ──► Prometheus ────┤
         │                   │           │
         │                   ▼           │
         │             AlertManager      │
         │                               │
         └──► Logs ──► Promtail ──► Loki│
                                         │
                                         ▼
                                    Grafana
                                   (Dashboards)
```

## Configuration

### Tracing Features Enabled

- ✅ **Async Export**: 10,000 queue size, 100 batch size, 4 workers
- ✅ **Sampling**: 10% base rate, always sample errors/slow
- ✅ **Metrics**: 23 Prometheus metrics
- ✅ **Logging**: JSON structured logs with correlation_id
- ✅ **Dashboards**: 3 Grafana dashboards with 44 panels
- ✅ **Alerts**: 43 AlertManager rules

### Sampling Configuration

```yaml
Base Rate: 10%              # Normal traffic sampled at 10%
Always Sample Errors: true  # Keep 100% of failed traces
Always Sample Slow: true    # Keep traces >5000ms
Slow Threshold: 5000ms      # Definition of "slow"
Deterministic: true         # Consistent sampling per correlation_id
```

### Storage Configuration

```yaml
PostgreSQL:
  Retention: Unlimited (configure via archival)
  Partitioning: Daily chunks (TimescaleDB)
  Indexes: 7 indexes for fast queries

MinIO (S3):
  Bucket: eve-traces
  Structure: eve-traces/{correlation_id}/{operation_id}/
  Lifecycle: Not configured (add via archival manager)

Loki:
  Retention: 30 days
  Compaction: Every 10 minutes
```

## Troubleshooting

### Services not starting

```bash
# Check service status
docker-compose ps

# View logs for specific service
docker-compose logs postgres
docker-compose logs example-service

# Restart a service
docker-compose restart example-service
```

### Database connection errors

```bash
# Check PostgreSQL is ready
docker exec eve-postgres pg_isready -U eve_user

# Connect manually
docker exec -it eve-postgres psql -U eve_user -d eve_traces

# View initialization logs
docker-compose logs postgres | grep "initialized successfully"
```

### MinIO not accessible

```bash
# Check MinIO health
curl http://localhost:9000/minio/health/live

# View MinIO logs
docker-compose logs minio

# Recreate bucket
docker-compose restart minio-init
```

### No metrics in Prometheus

```bash
# Check Prometheus targets
open http://localhost:9090/targets

# Check example service metrics endpoint
curl http://localhost:9091/metrics | grep eve_tracing

# Reload Prometheus config
curl -X POST http://localhost:9090/-/reload
```

### No data in Grafana

```bash
# Check datasource configuration
open http://localhost:3000/datasources

# Test Prometheus connection
curl http://prometheus:9090/api/v1/query?query=up

# Check if dashboards loaded
ls -la grafana/dashboards/
```

## Performance Tuning

### For Production

Update `docker-compose.yml` with:

```yaml
example-service:
  environment:
    # Increase async export capacity
    ASYNC_EXPORT_QUEUE_SIZE: "50000"
    ASYNC_EXPORT_BATCH_SIZE: "500"
    ASYNC_EXPORT_WORKERS: "8"

    # Adjust sampling for lower overhead
    SAMPLING_BASE_RATE: "0.01"  # 1% instead of 10%
```

### Database Optimization

```sql
-- Add to init.sql for production

-- Enable parallel query execution
ALTER SYSTEM SET max_parallel_workers_per_gather = 4;

-- Increase shared buffers (25% of RAM)
ALTER SYSTEM SET shared_buffers = '256MB';

-- Reload config
SELECT pg_reload_conf();
```

## Cleanup

```bash
# Stop all services
docker-compose down

# Remove all data volumes
docker-compose down -v

# Remove all images
docker-compose down --rmi all

# Complete cleanup
docker-compose down -v --rmi all --remove-orphans
```

## Next Steps

1. **Add More Services**: Add containerservice, workflowstorageservice, etc.
2. **Enable Archival**: Configure S3 Glacier lifecycle policies
3. **Add Dependency Mapping**: Build service dependency graph
4. **Configure Alerts**: Update AlertManager with email/Slack webhooks
5. **Production Deployment**: Use Kubernetes or Docker Swarm

## Related Documentation

- Main Tracing Documentation: `/home/opunix/eve/tracing/README.md`
- Grafana Dashboards: `/home/opunix/eve/grafana/README.md`
- AlertManager Setup: `/home/opunix/eve/grafana/alertmanager/README.md`
- Archival Guide: `/home/opunix/eve/tracing/ARCHIVAL.md`
- Dependency Mapping: `/home/opunix/eve/tracing/DEPENDENCIES.md`
- Sampling Guide: `/home/opunix/eve/tracing/SAMPLING.md`
