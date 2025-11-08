# GDPR & Compliance Features

Comprehensive GDPR compliance for EVE action tracing system.

## Overview

The EVE tracing system includes built-in GDPR compliance features:

1. **Right to Erasure** (Article 17) - Delete all traces for a data subject
2. **Right to Data Portability** (Article 20) - Export data in machine-readable format
3. **Audit Logging** - Track who accessed which traces
4. **Data Residency** - Control where data is stored (US, EU, APAC)
5. **PII Detection** - Automatic detection and flagging of sensitive data
6. **Retention Policies** - Automatic deletion after retention period
7. **Legal Basis Tracking** - Record legal basis for processing

## Quick Start

### Environment Variables

```bash
# Data residency (us, eu, apac)
export DATA_REGION=eu

# Retention period in days (default: 90)
export TRACING_RETENTION_DAYS=90

# Enable PII detection (default: true)
export TRACING_ENABLE_PII=true

# Legal basis for processing (default: "Legitimate Interest")
export TRACING_LEGAL_BASIS="Consent"

# Action traces database connection
export ACTION_TRACES_DSN=postgresql://claude:password@localhost:5433/action_traces?sslmode=disable
```

### Service Integration

```go
import (
    "eve.evalgo.org/tracing"
    "github.com/labstack/echo/v4"
)

func main() {
    // Initialize tracer with GDPR settings
    tracer := tracing.NewFromEnv("myservice", db, s3Client)

    e := echo.New()
    e.Use(tracer.Middleware())

    // Your routes...
}
```

### Specifying Data Subject in Requests

```json
{
  "@context": "https://schema.org",
  "@type": "CreateAction",
  "agent": {
    "@type": "Person",
    "identifier": "user-12345"
  },
  "meta": {
    "dataSubjectId": "user-12345"
  },
  "object": {
    "@type": "Thing",
    "name": "example"
  }
}
```

The tracer will automatically extract `dataSubjectId` from:
- `meta.dataSubjectId`
- `dataSubject.identifier`
- `agent.identifier`
- `participant.identifier`
- `customer.identifier`

## Features

### 1. GDPR Right to Erasure (Article 17)

Delete all traces for a data subject or correlation ID.

**Via MCP Tool:**

```
when_gdpr_erase_traces(
    data_subject_id="user-12345",
    user_id="admin-user",
    purpose="GDPR Right to Erasure request from customer"
)
```

**Via Code:**

```go
result, err := tracer.EraseTraces(ctx,
    "user-12345",  // data_subject_id
    "",            // correlation_id (optional)
    "admin-user",  // user_id
    "GDPR Right to Erasure",
)

if err != nil {
    log.Fatal(err)
}

fmt.Printf("Deleted %d actions, %d PII detections\n",
    result.DeletedActions, result.DeletedPII)
fmt.Printf("S3 objects to delete: %v\n", result.S3URLsToDelete)
```

**What it deletes:**
- All action_executions records
- All pii_detections records
- S3 payloads (request/response JSON files)
- Logs and artifacts

**What it creates:**
- Audit log entry (who deleted what and when)
- Erasure certificate for compliance documentation

### 2. GDPR Data Portability (Article 20)

Export all data for a data subject in JSON format.

**Via MCP Tool:**

```
when_gdpr_export_data(data_subject_id="user-12345")
```

**Via Code:**

```go
data, err := tracer.ExportDataSubjectData(ctx, "user-12345")
if err != nil {
    log.Fatal(err)
}

// data is []map[string]interface{}
jsonData, _ := json.MarshalIndent(data, "", "  ")
fmt.Println(string(jsonData))
```

**Export includes:**
- correlation_id, operation_id
- action_type, action_status
- started_at, duration_ms
- service_id
- metadata
- request_url, response_url (S3 URLs)

### 3. Audit Logging

All trace queries are logged for compliance.

**Automatic logging:**

Every MCP tool call automatically logs:
- Who accessed the data (user_id)
- What was accessed (correlation_id, data_subject_id)
- When it was accessed (timestamp)
- Why it was accessed (purpose)
- How many results were returned

**Via MCP Tool:**

```
when_audit_report(
    data_subject_id="user-12345",
    hours=168  # Last 7 days
)
```

**Via Code:**

```go
// Log access
err := tracer.LogTraceAccess(ctx,
    "user-id",          // who
    "query",            // access_type: query, view, export, delete
    "workflow_trace",   // resource_type
    "wf-123",           // correlation_id
    "",                 // operation_id
    "user-12345",       // data_subject_id
    10,                 // results_count
    "Customer support investigation",
    map[string]interface{}{
        "query": "failed actions",
    },
)

// Get audit trail
trail, err := tracer.GetAuditTrail(ctx, "user-12345", "", 168)
```

### 4. Data Residency

Control where data is stored based on regulations.

**Regional Configuration:**

```bash
# Store EU customer data in EU
export DATA_REGION=eu

# Store US customer data in US
export DATA_REGION=us

# Store APAC customer data in Asia-Pacific
export DATA_REGION=apac
```

**How it works:**

1. Each trace is tagged with `data_region` field
2. PostgreSQL can be partitioned by region (optional)
3. S3 buckets should be regional:
   - `eve-traces-eu` (Frankfurt region)
   - `eve-traces-us` (Virginia region)
   - `eve-traces-apac` (Tokyo region)

**Query by region:**

```sql
SELECT * FROM action_executions
WHERE data_region = 'eu'
  AND started_at > NOW() - INTERVAL '24 hours';
```

### 5. PII Detection

Automatic detection of sensitive data.

**Enabled by default:**

```bash
export TRACING_ENABLE_PII=true
```

**Detected PII types:**

- **email** - Email addresses (john@example.com)
- **phone** - Phone numbers (+1-555-123-4567)
- **ssn** - Social Security Numbers (123-45-6789)
- **credit_card** - Credit card numbers (4532-1234-5678-9010)
- **ip_address** - IP addresses (192.168.1.1)
- **passport** - Passport numbers (US123456789)
- **iban** - Bank account numbers (DE89370400440532013000)

**Via MCP Tool:**

```
when_pii_report(
    correlation_id="wf-123",
    hours=24
)
```

**Via Code:**

```go
// Detect PII in text
detections := tracer.DetectPII("Email: john@example.com, SSN: 123-45-6789", nil)

for _, detection := range detections {
    fmt.Printf("Found %s: %s (confidence: %.2f)\n",
        detection.PIIType,
        detection.PatternMatched,
        detection.Confidence,
    )
}

// Redact PII
redacted := tracing.RedactPII("Email: john@example.com", detections)
// Output: "Email: [REDACTED_EMAIL]"
```

**PII detections are stored:**

```sql
SELECT * FROM pii_detections
WHERE correlation_id = 'wf-123';
```

### 6. Retention Policies

Automatic deletion after retention period.

**Configuration:**

```bash
# Default: 90 days
export TRACING_RETENTION_DAYS=90
```

**How it works:**

1. Each trace gets `retention_until` timestamp
2. Automatic cleanup job deletes expired traces
3. Can be run manually or via cron

**Run cleanup:**

```go
deleted, err := tracer.DeleteExpiredTraces(ctx)
fmt.Printf("Deleted %d expired traces\n", deleted)
```

**SQL function:**

```sql
SELECT delete_expired_traces();
```

**Cron job example:**

```cron
# Run daily at 2 AM
0 2 * * * psql -d action_traces -c "SELECT delete_expired_traces();"
```

### 7. Legal Basis Tracking

Record legal basis for data processing.

**Configuration:**

```bash
# Options: Consent, Contract, Legal Obligation, Vital Interests, Public Task, Legitimate Interest
export TRACING_LEGAL_BASIS="Consent"
```

**Stored in:**

```sql
SELECT legal_basis, COUNT(*)
FROM action_executions
GROUP BY legal_basis;
```

## Database Schema

### GDPR Columns in action_executions

```sql
-- Data subject tracking
data_subject_id TEXT

-- Legal basis for processing
legal_basis TEXT  -- "Consent", "Legitimate Interest", etc.

-- Consent tracking
consent_id TEXT   -- Reference to consent record

-- Data residency
data_region TEXT  -- "us", "eu", "apac"

-- Retention management
retention_until TIMESTAMPTZ

-- PII flags
contains_pii BOOLEAN
pii_redacted BOOLEAN
```

### trace_access_audit Table

```sql
CREATE TABLE trace_access_audit (
    id UUID PRIMARY KEY,
    accessed_at TIMESTAMPTZ NOT NULL,
    user_id TEXT NOT NULL,
    user_email TEXT,
    user_ip INET,
    access_type TEXT NOT NULL,      -- 'query', 'view', 'export', 'delete'
    resource_type TEXT NOT NULL,    -- 'workflow_trace', 'action_detail'
    correlation_id TEXT,
    operation_id TEXT,
    data_subject_id TEXT,
    query_parameters JSONB,
    results_count INTEGER,
    purpose TEXT,
    legal_basis TEXT
);
```

### pii_detections Table

```sql
CREATE TABLE pii_detections (
    id UUID PRIMARY KEY,
    detected_at TIMESTAMPTZ NOT NULL,
    correlation_id TEXT NOT NULL,
    operation_id TEXT NOT NULL,
    location TEXT NOT NULL,         -- 'request', 'response', 'metadata'
    field_path TEXT,                -- JSON path like '$.agent.email'
    pii_type TEXT NOT NULL,         -- 'email', 'phone', 'ssn', etc.
    pattern_matched TEXT,
    confidence FLOAT,               -- 0.0-1.0
    redacted BOOLEAN,
    token TEXT,                     -- If tokenized
    data_subject_id TEXT
);
```

## MCP Tools

### when_gdpr_erase_traces

Delete all traces for a data subject or correlation ID.

**Parameters:**
- `data_subject_id` (optional) - Data subject identifier
- `correlation_id` (optional) - Correlation ID to erase
- `user_id` (optional) - User requesting erasure
- `purpose` (optional) - Justification for erasure

**Returns:** Erasure certificate with deletion counts

### when_gdpr_export_data

Export all data for a data subject in JSON format.

**Parameters:**
- `data_subject_id` (required) - Data subject identifier

**Returns:** JSON export of all traces for the data subject

### when_audit_report

Generate audit trail report.

**Parameters:**
- `data_subject_id` (optional) - Data subject filter
- `correlation_id` (optional) - Correlation filter
- `hours` (optional) - Hours to look back (default: 168 = 7 days)

**Returns:** Markdown report of all access events

### when_pii_report

Generate PII detection report.

**Parameters:**
- `correlation_id` (optional) - Correlation filter
- `data_subject_id` (optional) - Data subject filter
- `hours` (optional) - Hours to look back (default: 24)

**Returns:** Markdown report of all PII detections

## Production Recommendations

### 1. Regional S3 Buckets

```bash
# EU region (Frankfurt)
aws s3 mb s3://eve-traces-eu --region eu-central-1

# US region (Virginia)
aws s3 mb s3://eve-traces-us --region us-east-1

# APAC region (Tokyo)
aws s3 mb s3://eve-traces-apac --region ap-northeast-1
```

### 2. S3 Lifecycle Policies

```json
{
  "Rules": [
    {
      "Id": "MoveToGlacier",
      "Status": "Enabled",
      "Transitions": [
        {
          "Days": 90,
          "StorageClass": "GLACIER"
        }
      ]
    },
    {
      "Id": "DeleteAfterRetention",
      "Status": "Enabled",
      "Expiration": {
        "Days": 365
      }
    }
  ]
}
```

### 3. PostgreSQL Retention Policy

```sql
-- Keep raw traces for 90 days (configurable via TimescaleDB)
SELECT add_retention_policy('action_executions', INTERVAL '90 days');
```

### 4. Automated Cleanup Cron

```bash
#!/bin/bash
# /etc/cron.daily/cleanup-expired-traces

psql -d action_traces -c "SELECT delete_expired_traces();"
```

### 5. Audit Log Retention

```sql
-- Keep audit logs for 7 years (legal requirement in many jurisdictions)
SELECT add_retention_policy('trace_access_audit', INTERVAL '7 years');
```

## Compliance Checklist

- [ ] Configure DATA_REGION for all services
- [ ] Set appropriate TRACING_RETENTION_DAYS
- [ ] Enable PII detection (TRACING_ENABLE_PII=true)
- [ ] Set correct TRACING_LEGAL_BASIS
- [ ] Configure regional S3 buckets
- [ ] Set up S3 lifecycle policies
- [ ] Schedule daily cleanup job
- [ ] Document data processing purposes
- [ ] Train staff on GDPR procedures
- [ ] Test erasure process end-to-end
- [ ] Test data export process
- [ ] Review audit logs regularly
- [ ] Document data retention policies
- [ ] Maintain erasure certificates

## Troubleshooting

### Erasure not deleting S3 objects

Make sure S3 client has delete permissions:

```json
{
  "Effect": "Allow",
  "Action": ["s3:DeleteObject"],
  "Resource": "arn:aws:s3:::eve-traces*/*"
}
```

### PII detection missing patterns

Add custom patterns:

```go
customPatterns := []tracing.PIIPattern{
    {
        Type:       "employee_id",
        Pattern:    `EMP-\d{6}`,
        Confidence: 0.95,
    },
}

detections := tracer.DetectPII(data, customPatterns)
```

### Retention cleanup not running

Check TimescaleDB is enabled:

```sql
SELECT * FROM timescaledb_information.hypertables
WHERE hypertable_name = 'action_executions';
```

## Legal Disclaimer

This implementation provides technical capabilities for GDPR compliance. You are responsible for:
- Ensuring compliance with all applicable laws
- Documenting your data processing activities
- Obtaining necessary consents
- Responding to data subject requests within legal timeframes
- Maintaining proper records

Consult with legal counsel for GDPR compliance guidance.
