# EVE Tracing AlertManager Configuration

Production-ready Prometheus AlertManager configuration for EVE's distributed action tracing system.

## Overview

This AlertManager setup provides:
- **60+ alert rules** across 3 categories (operational, business, compliance)
- **Critical/warning/info severity levels**
- **Multi-channel notifications** (Slack, email, PagerDuty)
- **Smart routing** based on alert category and severity
- **Inhibition rules** to reduce alert noise
- **HTML email templates** for professional notifications
- **GDPR/compliance-specific alerting** with legal team routing

## Directory Structure

```
alertmanager/
├── alertmanager.yml          # Main AlertManager configuration
├── rules/                    # Prometheus alert rules
│   ├── tracing_operational.yml   # Infrastructure/SRE alerts (12 rules)
│   ├── tracing_business.yml      # Workflow/product alerts (15 rules)
│   └── tracing_compliance.yml    # GDPR/privacy alerts (16 rules)
├── templates/                # Notification templates
│   └── email.tmpl            # HTML email templates
└── README.md                 # This file
```

## Alert Categories

### 1. Operational Alerts (`tracing_operational.yml`)

**Audience**: SRE, DevOps, Platform Engineers

**Critical Alerts**:
- `TraceExporterQueueCritical`: Queue >90% full (9000/10000 traces)
- `TracesBeingDropped`: Data loss occurring due to queue overflow
- `HighActionErrorRate`: >10 errors/sec across services
- `PostgreSQLWriteFailures`: Database write failures
- `S3UploadFailures`: S3 upload failures
- `ExporterBatchFailureRate`: >10% of export batches failing

**Warning Alerts**:
- `TraceExporterQueueHigh`: Queue >70% full
- `HighActionLatency`: p95 latency >5 seconds
- `ExporterBatchLatencyHigh`: Slow batch exports (>10s)
- `TracePayloadSizeLarge`: Payloads >500KB
- `LowPostgreSQLWriteRate`: Possible export stall

### 2. Business Alerts (`tracing_business.yml`)

**Audience**: Product Managers, Engineering Managers, Business Analysts

**Critical Alerts**:
- `WorkflowSuccessRateLow`: <90% success rate (SLO: >95%)
- `HighBusinessErrorRate`: >5 errors/sec for business actions
- `WorkflowSLOViolation`: p95 duration >120s (SLO: <60s)
- `WorkflowStormDetected`: >100 workflows in flight simultaneously
- `CreateActionFailureSpike`: High creation failure rate

**Warning Alerts**:
- `WorkflowSuccessRateDegraded`: <95% success rate
- `WorkflowDurationIncreasing`: p95 approaching 60s SLO
- `ActionTypeFailureSpike`: >5% error rate for specific action types
- `LowThroughputWarning`: Unexpectedly low traffic
- `SpecificActionTypeSlow`: Individual action type p95 >10s
- `DeleteActionAnomalyDetected`: Unusual spike in deletions (5x normal)

### 3. Compliance Alerts (`tracing_compliance.yml`)

**Audience**: Legal, DPO (Data Protection Officer), Compliance Officers

**Critical Alerts**:
- `DataResidencyViolation`: Data written to wrong region (GDPR violation!)
- `UnredactedPIIInTraces`: Unredacted PII detected in traces
- `GDPRErasureRequestBacklog`: Erasure requests backing up (legal SLA risk)
- `GDPRErasureFailures`: Cannot fulfill erasure requests
- `AuditLogWriteFailures`: Audit trail broken
- `EUDataInNonEURegion`: EU citizen data in non-EU region

**Warning Alerts**:
- `PIIRedactionRateLow`: <95% redaction rate (target: >99%)
- `HighPIIDetectionRate`: Unusually high PII detection
- `GDPRErasureResponseTimeSlow`: p95 >30 minutes (target: <5 minutes)
- `RetentionPolicyNotRunning`: Old data not being deleted
- `UnauthorizedTraceAccess`: Unauthorized access attempts
- `SuspiciousAuditAccessPattern`: Potential data exfiltration

## Installation

### 1. Install AlertManager

```bash
# Download AlertManager
wget https://github.com/prometheus/alertmanager/releases/download/v0.26.0/alertmanager-0.26.0.linux-amd64.tar.gz
tar xvf alertmanager-0.26.0.linux-amd64.tar.gz
cd alertmanager-0.26.0.linux-amd64

# Copy configuration
sudo cp /home/opunix/eve/grafana/alertmanager/alertmanager.yml /etc/alertmanager/
sudo cp -r /home/opunix/eve/grafana/alertmanager/templates /etc/alertmanager/

# Set environment variables
export SMTP_PASSWORD="your-smtp-password"
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
export PAGERDUTY_SERVICE_KEY="your-pagerduty-integration-key"

# Start AlertManager
./alertmanager --config.file=/etc/alertmanager/alertmanager.yml
```

### 2. Configure Prometheus to Use AlertManager

Edit `/etc/prometheus/prometheus.yml`:

```yaml
# Alertmanager configuration
alerting:
  alertmanagers:
    - static_configs:
        - targets:
            - localhost:9093

# Load alert rules
rule_files:
  - /home/opunix/eve/grafana/alertmanager/rules/*.yml
```

Restart Prometheus:

```bash
sudo systemctl restart prometheus
```

### 3. Verify Configuration

```bash
# Check AlertManager is running
curl http://localhost:9093/-/healthy

# Validate alert rules
promtool check rules /home/opunix/eve/grafana/alertmanager/rules/*.yml

# Check loaded rules in Prometheus
curl http://localhost:9090/api/v1/rules | jq
```

## Configuration

### Update Notification Channels

Edit `alertmanager.yml` and update:

1. **SMTP Settings**:
   ```yaml
   smtp_smarthost: 'smtp.gmail.com:587'
   smtp_from: 'eve-alerts@example.com'
   smtp_auth_username: 'eve-alerts@example.com'
   smtp_auth_password: '${SMTP_PASSWORD}'
   ```

2. **Slack Webhooks**:
   ```yaml
   slack_api_url: '${SLACK_WEBHOOK_URL}'

   slack_configs:
     - channel: '#eve-platform-alerts'    # Update channel names
     - channel: '#eve-compliance-alerts'
     - channel: '#eve-product-alerts'
   ```

3. **Email Recipients**:
   ```yaml
   email_configs:
     - to: 'platform-team@example.com'    # Update email addresses
     - to: 'legal@example.com,compliance@example.com'
     - to: 'product-team@example.com'
   ```

4. **PagerDuty**:
   ```yaml
   pagerduty_configs:
     - service_key: '${PAGERDUTY_SERVICE_KEY}'
   ```

### Adjust Alert Thresholds

Edit rule files in `rules/` directory:

**Example: Change queue threshold**:
```yaml
# In tracing_operational.yml
- alert: TraceExporterQueueCritical
  expr: eve_tracing_exporter_queue_size > 9000  # Change threshold here
  for: 5m  # Change duration here
```

**Example: Change SLO target**:
```yaml
# In tracing_business.yml
- alert: WorkflowSLOViolation
  expr: |
    histogram_quantile(0.95,
      sum(rate(eve_tracing_workflow_duration_seconds_bucket[10m])) by (le)
    ) > 120  # Change from 120s to your SLO target
```

### Customize Alert Routing

Edit `alertmanager.yml` routes:

```yaml
routes:
  # Add custom routing rule
  - match:
      service_id: containerservice
      severity: critical
    receiver: 'team-container-experts'
    group_wait: 10s
    repeat_interval: 30m
```

## Usage Examples

### Testing Alerts

```bash
# Trigger test alert
curl -X POST http://localhost:9093/api/v1/alerts -d '[
  {
    "labels": {
      "alertname": "TestAlert",
      "severity": "warning",
      "category": "operational"
    },
    "annotations": {
      "summary": "Test alert from EVE",
      "description": "This is a test alert"
    }
  }
]'
```

### Silencing Alerts

```bash
# Silence specific alert for 2 hours
amtool silence add \
  alertname=TraceExporterQueueHigh \
  --duration=2h \
  --comment="Planned maintenance"

# Silence all alerts for service during deployment
amtool silence add \
  service_id=containerservice \
  --duration=30m \
  --comment="Deploying v1.2.3"

# List active silences
amtool silence query

# Expire silence
amtool silence expire <silence-id>
```

### Viewing Active Alerts

```bash
# List all firing alerts
amtool alert query

# Filter by severity
amtool alert query severity=critical

# Filter by category
amtool alert query category=compliance

# View in Prometheus UI
# http://localhost:9090/alerts
```

## Alert Runbooks

Each alert includes a `runbook_url` annotation. Create runbooks at the specified URLs:

### Example Runbook Structure

**URL**: `https://wiki.example.com/eve/runbooks/traces-dropped`

**Content**:
```markdown
# Runbook: TracesBeingDropped

## Symptoms
- Alert: TracesBeingDropped firing
- Metric: eve_tracing_exporter_queue_dropped_total increasing
- Impact: Data loss - traces not being persisted

## Diagnosis
1. Check queue size:
   ```
   curl http://service:8080/metrics | grep exporter_queue_size
   ```

2. Check worker health:
   ```
   curl http://service:8080/health/tracing
   ```

3. Check database connectivity:
   ```
   psql -h postgres -U eve -d tracing -c "SELECT 1"
   ```

## Resolution
### Quick Fix (stop bleeding):
```bash
# Increase queue size temporarily
kubectl set env deployment/containerservice TRACING_ASYNC_QUEUE_SIZE=20000
```

### Root Cause Fixes:
1. **Database slow**: Scale up PostgreSQL, add connection pooling
2. **S3 slow**: Check network, verify credentials, check bucket region
3. **High traffic**: Add more worker instances
4. **Worker stalled**: Restart service

## Prevention
- Set up capacity alerts at 70% queue
- Monitor database query performance
- Review retention policies to reduce load
```

## Monitoring AlertManager

### Health Check

```bash
# AlertManager health
curl http://localhost:9093/-/healthy

# Check configuration
curl http://localhost:9093/api/v1/status | jq
```

### Metrics

AlertManager exposes metrics on `:9093/metrics`:

```promql
# Alert processing rate
rate(alertmanager_alerts_received_total[5m])

# Notification success rate
rate(alertmanager_notifications_total{integration="slack"}[5m])
rate(alertmanager_notifications_failed_total{integration="slack"}[5m])

# Silences
alertmanager_silences_active
```

### Logs

```bash
# View AlertManager logs
journalctl -u alertmanager -f

# Search for specific alert
journalctl -u alertmanager | grep "TracesBeingDropped"
```

## Integration with Grafana

### Create AlertManager Data Source

1. Grafana → Configuration → Data Sources → Add data source
2. Select "Alertmanager"
3. URL: `http://localhost:9093`
4. Save & Test

### View Alerts in Grafana

1. Create dashboard panel with type "Alert list"
2. Configure to show alerts from AlertManager data source
3. Filter by severity, category, or labels

## Email Template Customization

Edit `templates/email.tmpl`:

```html
<!-- Add company logo -->
<div class="header">
    <img src="https://example.com/logo.png" alt="Company Logo">
    <h1>EVE Tracing Alert</h1>
</div>

<!-- Customize colors -->
<style>
    .header {
        background: #your-brand-color;
    }
</style>
```

Test templates:

```bash
# Send test notification
amtool alert add \
  alertname=TestAlert \
  severity=warning \
  category=operational
```

## Troubleshooting

### Alerts Not Firing

1. **Check Prometheus rule evaluation**:
   ```bash
   curl http://localhost:9090/api/v1/rules | jq '.data.groups[].rules[] | select(.name=="TracesBeingDropped")'
   ```

2. **Verify metrics exist**:
   ```bash
   curl http://service:8080/metrics | grep eve_tracing_exporter_queue_dropped_total
   ```

3. **Check AlertManager receiving alerts**:
   ```bash
   curl http://localhost:9093/api/v1/alerts | jq
   ```

### Notifications Not Sending

1. **Check AlertManager logs**:
   ```bash
   journalctl -u alertmanager | grep -i error
   ```

2. **Test SMTP connection**:
   ```bash
   telnet smtp.gmail.com 587
   ```

3. **Verify webhook URLs**:
   ```bash
   curl -X POST $SLACK_WEBHOOK_URL -d '{"text":"Test"}'
   ```

### Too Many Alerts

1. **Review inhibition rules** in `alertmanager.yml`
2. **Adjust `group_interval`** to batch more alerts together
3. **Increase thresholds** for noisy alerts
4. **Add silences** during known maintenance windows

## Best Practices

1. **Test alerts regularly**: Use `amtool` to send test alerts monthly
2. **Keep runbooks updated**: Review and update runbooks quarterly
3. **Monitor alert fatigue**: Track alert acknowledge/resolve times
4. **Tune thresholds**: Adjust based on actual system behavior
5. **Document on-call procedures**: Ensure team knows how to respond
6. **Review compliance alerts immediately**: Never ignore GDPR violations
7. **Use silences for maintenance**: Don't disable alerting rules
8. **Archive resolved alerts**: Keep history for post-mortems

## Alert Severity Guidelines

### Critical (Immediate Action Required)
- Data loss occurring
- GDPR violations
- System completely broken
- SLO violations affecting users
- **Response Time**: <15 minutes
- **Escalation**: PagerDuty + immediate phone call

### Warning (Action Required Soon)
- Performance degradation
- Approaching thresholds
- Potential issues developing
- **Response Time**: <1 hour during business hours
- **Escalation**: Slack + email

### Info (Awareness Only)
- System recovered
- Metrics looking good
- Background tasks completed
- **Response Time**: Review during business hours
- **Escalation**: Email only

## Related Documentation

- Prometheus Metrics: `/home/opunix/eve/tracing/metrics.go`
- Grafana Dashboards: `/home/opunix/eve/grafana/dashboards/`
- MCP Tools: `/home/opunix/when` (when-mcp for trace queries)
- Tracing Configuration: `/home/opunix/eve/tracing/config.go`

## Support

For issues with AlertManager:
1. Check AlertManager docs: https://prometheus.io/docs/alerting/latest/alertmanager/
2. Validate config: `amtool check-config alertmanager.yml`
3. Check logs: `journalctl -u alertmanager -f`
4. Test rules: `promtool check rules rules/*.yml`
