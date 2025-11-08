# EVE Tracing Grafana Dashboards

Production-ready Grafana dashboards for monitoring EVE's distributed action tracing system.

## Dashboards

### 1. Operational Dashboard (`dashboards/operational.json`)
**Audience**: SRE, DevOps, Platform Engineers

**Purpose**: Monitor technical health and performance of the tracing system itself

**Panels**:
- Action request rate per second (by service)
- Action error rate with alerting (> 5 errors/sec)
- Action latency percentiles (p50, p95, p99)
- Async exporter queue size with alerting (> 8000 traces)
- Dropped traces with alerting (> 1/sec)
- Exporter batch latency and success rate
- PostgreSQL write rate and errors
- S3 upload rate and errors
- Trace payload sizes
- OpenTelemetry trace links

**Alerts**:
- High Error Rate: > 5 errors/sec
- Queue Near Capacity: > 8000 traces in queue
- Traces Being Dropped: any dropped traces

### 2. Business Dashboard (`dashboards/business.json`)
**Audience**: Product Managers, Business Analysts, Engineering Managers

**Purpose**: Understand workflow execution patterns and business-level performance

**Panels**:
- Workflow success rate (with SLO thresholds)
- Workflows completed (5-minute rolling window)
- Active workflows (in-flight gauge)
- Average workflow duration
- Workflow completion/failure rate over time
- Workflow duration percentiles
- Actions by type and object type
- Workflow step duration by action type
- Top 10 slowest workflows
- Service throughput (actions/sec by service)
- Error rate by service with alerting
- Action success rate by type
- Business SLO gauge: 95% of workflows < 60s

**Alerts**:
- High Business Error Rate: > 10 errors/sec per service

### 3. Compliance Dashboard (`dashboards/compliance.json`)
**Audience**: Legal, Privacy Officers, Compliance Teams, Auditors

**Purpose**: Monitor GDPR compliance, PII handling, and data governance

**Panels**:
- GDPR erasure requests (24h)
- GDPR data export requests (24h)
- PII detections (24h)
- Trace retention cleanup (24h)
- Erasure requests by reason
- Data export requests over time
- PII detections by type and location
- PII redaction rate (SLO: > 99%)
- Audit access by user, type, and resource
- Retention policy compliance
- Data residency violations (SLO: 0)
- Unredacted PII in traces (SLO: 0)
- Erasure response time (SLO: < 5 minutes)

**Alerts**:
- Data Residency Violation Detected: any violations
- Unredacted PII Detected: > 5 instances in 5 minutes

## Installation

### Prerequisites
- Grafana 9.0+ installed
- Prometheus data source configured in Grafana
- EVE tracing system running with metrics enabled

### Import Dashboards

1. **Via Grafana UI**:
   ```bash
   # Open Grafana -> Dashboards -> Import
   # Upload JSON files from dashboards/ directory
   ```

2. **Via Grafana API**:
   ```bash
   # Set your Grafana URL and API key
   export GRAFANA_URL="http://localhost:3000"
   export GRAFANA_API_KEY="your-api-key"

   # Import operational dashboard
   curl -X POST "$GRAFANA_URL/api/dashboards/db" \
     -H "Authorization: Bearer $GRAFANA_API_KEY" \
     -H "Content-Type: application/json" \
     -d @dashboards/operational.json

   # Import business dashboard
   curl -X POST "$GRAFANA_URL/api/dashboards/db" \
     -H "Authorization: Bearer $GRAFANA_API_KEY" \
     -H "Content-Type: application/json" \
     -d @dashboards/business.json

   # Import compliance dashboard
   curl -X POST "$GRAFANA_URL/api/dashboards/db" \
     -H "Authorization: Bearer $GRAFANA_API_KEY" \
     -H "Content-Type: application/json" \
     -d @dashboards/compliance.json
   ```

3. **Via Grafana Provisioning** (recommended for production):
   ```yaml
   # /etc/grafana/provisioning/dashboards/eve-tracing.yaml
   apiVersion: 1
   providers:
     - name: 'EVE Tracing'
       orgId: 1
       folder: 'EVE'
       type: file
       disableDeletion: false
       updateIntervalSeconds: 30
       options:
         path: /path/to/eve/grafana/dashboards
   ```

## Configuration

### Prometheus Data Source

Ensure your Prometheus instance is scraping the `/metrics` endpoint from all EVE services:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'eve-services'
    scrape_interval: 15s
    static_configs:
      - targets:
          - 'containerservice:8080'
          - 'basexservice:8080'
          - 'sparqlservice:8080'
          - 's3service:8080'
          - 'infisicalservice:8080'
          - 'workflowstorageservice:8080'
          - 'templateservice:8080'
          - 'fetcher:8080'
          - 'antwrapperservice:8080'
          - 'rabbitmqservice:8080'
          - 'graphdbservice:8080'
```

### Dashboard Variables

All dashboards support the following variables (configure in Grafana UI):

- `$datasource`: Prometheus data source (default: Prometheus)
- `$service`: Filter by service_id (multi-select)
- `$interval`: Auto-refresh interval (default: 30s)

## Alerts

### Enable Alerting

1. **Configure Alert Notification Channels**:
   - Grafana UI -> Alerting -> Notification channels
   - Add Slack, PagerDuty, email, etc.

2. **Assign Channels to Dashboards**:
   - Edit each dashboard
   - For panels with alerts, configure notification channels
   - Save dashboard

3. **Alert Rules Included**:
   - **Operational**: High error rate, queue capacity, dropped traces
   - **Business**: High business error rate per service
   - **Compliance**: Data residency violations, unredacted PII

### AlertManager Integration (Optional)

For advanced alerting, use Prometheus AlertManager with the alert rules in `../alertmanager/rules/`.

## Usage Examples

### Debugging a Slow Workflow
1. Open **Business Dashboard**
2. Check "Top 10 Slowest Workflows" table
3. Note correlation_id
4. Open **Operational Dashboard**
5. Filter by correlation_id (add variable)
6. Check "Action Latency" to identify bottleneck

### Investigating Error Spike
1. Open **Operational Dashboard**
2. Check "Action Error Rate" panel - identify which service
3. Check "PostgreSQL Errors" or "S3 Errors" - identify error type
4. Use when-mcp tools to query failed actions:
   ```
   when_get_failed_actions(time_range="1h", service_id="containerservice")
   ```

### GDPR Audit
1. Open **Compliance Dashboard**
2. Review "GDPR Erasure Requests" - verify all processed
3. Check "PII Redaction Rate" - should be > 99%
4. Review "Audit Access by User" - identify who accessed what
5. Check "Data Residency Violations" - should be 0

### Capacity Planning
1. Open **Business Dashboard**
2. Check "Service Throughput" trends over 7 days
3. Open **Operational Dashboard**
4. Check "Async Exporter Queue Size" - high water marks
5. Adjust TRACING_ASYNC_QUEUE_SIZE if needed

## Customization

### Adding Custom Panels

1. Clone an existing panel in the dashboard JSON
2. Modify the PromQL query (`targets[].expr`)
3. Update title, labels, thresholds as needed
4. Re-import dashboard

### Adjusting Alert Thresholds

Edit dashboard JSON, find alert configurations:

```json
"alert": {
  "conditions": [
    {
      "evaluator": {"params": [5], "type": "gt"},  // Change threshold here
      "query": {"params": ["A", "5m", "now"]},      // Change time window here
      "type": "query"
    }
  ]
}
```

## Metrics Reference

All metrics are documented in `/home/opunix/eve/tracing/metrics.go`.

### Key Metrics

- `eve_tracing_actions_total`: Total actions executed (counter)
- `eve_tracing_action_duration_seconds`: Action latency (histogram)
- `eve_tracing_action_errors_total`: Action errors (counter)
- `eve_tracing_workflows_total`: Workflows completed (counter)
- `eve_tracing_workflow_duration_seconds`: Workflow duration (histogram)
- `eve_tracing_exporter_queue_size`: Current queue size (gauge)
- `eve_tracing_gdpr_erasures_total`: GDPR erasures (counter)
- `eve_tracing_pii_detections_total`: PII detections (counter)

See metrics.go for complete list of 23+ metrics.

## Troubleshooting

### No Data in Dashboards

1. Check Prometheus is scraping EVE services:
   ```bash
   curl http://localhost:9090/api/v1/targets
   ```

2. Check EVE service has `/metrics` endpoint:
   ```bash
   curl http://containerservice:8080/metrics
   ```

3. Check metrics are enabled in service config:
   ```bash
   echo $TRACING_METRICS_ENABLED  # should be true or unset
   ```

### High Query Latency

1. Reduce dashboard refresh rate (30s → 1m)
2. Reduce time range (24h → 6h)
3. Add recording rules in Prometheus for expensive queries
4. Enable Prometheus query result caching

### Missing Metrics

If certain metrics don't appear:
1. Check metric is implemented in tracing/metrics.go
2. Check metric is recorded in tracing/middleware.go or tracing/async.go
3. Verify action type/object type triggers metric recording
4. Check Prometheus retention period (default 15d)

## Performance

### Dashboard Query Load

Each dashboard generates ~15-20 PromQL queries every refresh interval.

**Recommended settings**:
- Operational Dashboard: 30s refresh (real-time monitoring)
- Business Dashboard: 1m refresh (trend analysis)
- Compliance Dashboard: 5m refresh (audit review)

### Prometheus Resources

For 11 EVE services with 1000 req/sec aggregate:
- Memory: ~4GB
- Storage: ~10GB/day (15d retention = 150GB)
- CPU: 2-4 cores

## Related Documentation

- Prometheus Metrics: `/home/opunix/eve/tracing/metrics.go`
- Async Exporter: `/home/opunix/eve/tracing/async.go`
- Tracing Configuration: `/home/opunix/eve/tracing/config.go`
- MCP Tools: `/home/opunix/when` (when-mcp for trace queries)
- AlertManager Rules: `/home/opunix/eve/grafana/alertmanager/` (pending)

## Support

For issues with dashboards:
1. Check Grafana logs: `journalctl -u grafana-server -f`
2. Check Prometheus logs: `journalctl -u prometheus -f`
3. Verify PromQL queries in Prometheus UI: http://localhost:9090/graph
4. Test metrics endpoint: `curl http://service:8080/metrics | grep eve_tracing`
