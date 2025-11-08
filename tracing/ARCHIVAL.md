# Trace Data Archival to S3 Glacier

Automatic long-term archival of old traces to S3 Glacier for cost-effective compliance and retention.

## Overview

The archival manager automatically:
- **Archives old traces** to S3 Glacier after retention period (default: 90 days)
- **Deletes very old traces** after extended retention (default: 365 days)
- **Reduces storage costs** by 90-95% for archived data
- **Supports restore** for compliance requests (GDPR data export)
- **Maintains audit trail** with archival timestamps

## Cost Comparison

### S3 Standard vs Glacier Storage Costs

**Assumptions**:
- 1 million traces/day
- 10KB average trace size
- 90-day retention in S3 Standard
- 275-day retention in S3 Glacier (365 - 90)

**Monthly Storage Costs** (us-east-1):

| Storage Type | Size | Cost/GB/month | Monthly Cost |
|--------------|------|---------------|--------------|
| S3 Standard (0-90 days) | 900GB | $0.023 | $20.70 |
| S3 Glacier (91-365 days) | 2,750GB | $0.004 | $11.00 |
| **Total** | | | **$31.70** |

**vs S3 Standard Only**:
| S3 Standard (365 days) | 3,650GB | $0.023 | $83.95 |

**Annual Savings**: $627/year per service (65% reduction)

**For 11 EVE services**: $6,897/year savings

## Quick Start

### 1. Setup Database Schema

Add archival columns to `action_executions` table:

```sql
ALTER TABLE action_executions
ADD COLUMN archived_at TIMESTAMP,
ADD COLUMN archived_s3_key TEXT;

CREATE INDEX idx_action_executions_archived
ON action_executions(archived_at)
WHERE archived_at IS NOT NULL;
```

### 2. Initialize Archival Manager

```go
import (
	"context"
	"eve.evalgo.org/tracing"
)

// Create archival manager
archival := tracing.NewArchivalManager(db, s3Client, tracing.ArchivalConfig{
	Enabled:             true,
	ArchiveAfterDays:    90,   // Archive after 90 days
	DeleteAfterDays:     365,  // Delete after 365 days total
	S3Bucket:            "eve-traces",
	S3Prefix:            "archived/",
	GlacierStorageClass: "GLACIER",  // or "DEEP_ARCHIVE" for even cheaper
	BatchSize:           1000,
})
```

### 3. Setup S3 Lifecycle Policies

```go
// One-time setup: configure S3 to auto-transition to Glacier
ctx := context.Background()
if err := archival.SetupLifecyclePolicy(ctx); err != nil {
	log.Fatal(err)
}
```

### 4. Run Archival Jobs

```go
// Run archival job (e.g., daily cron)
stats, err := archival.ArchiveOldTraces(ctx)
if err != nil {
	log.Errorf("Archival failed: %v", err)
} else {
	log.Infof("Archived %d traces", stats.TracesArchived)
}

// Cleanup very old traces
deleteStats, err := archival.DeleteOldArchivedTraces(ctx)
if err != nil {
	log.Errorf("Deletion failed: %v", err)
} else {
	log.Infof("Deleted %d old traces", deleteStats.TracesDeleted)
}
```

## Configuration

### ArchivalConfig

```go
type ArchivalConfig struct {
	// Enabled controls archival (default: false)
	Enabled bool

	// ArchiveAfterDays: move to Glacier after this many days (default: 90)
	ArchiveAfterDays int

	// DeleteAfterDays: delete after this many days total (default: 365)
	// Set to 0 to keep forever
	DeleteAfterDays int

	// S3Bucket for traces (e.g., "eve-traces")
	S3Bucket string

	// S3Prefix for archived traces (default: "archived/")
	S3Prefix string

	// GlacierStorageClass: "GLACIER" or "DEEP_ARCHIVE"
	// GLACIER: $0.004/GB/month, restore in 3-5 hours
	// DEEP_ARCHIVE: $0.00099/GB/month, restore in 12 hours
	GlacierStorageClass string

	// BatchSize for archival operations (default: 1000)
	BatchSize int

	// DryRun mode: log what would happen without actually doing it
	DryRun bool
}
```

### Environment Variables

```bash
# Enable archival
export ARCHIVAL_ENABLED=true

# Archive after 90 days
export ARCHIVAL_ARCHIVE_AFTER_DAYS=90

# Delete after 365 days total
export ARCHIVAL_DELETE_AFTER_DAYS=365

# S3 bucket and prefix
export ARCHIVAL_S3_BUCKET=eve-traces
export ARCHIVAL_S3_PREFIX=archived/

# Glacier storage class (GLACIER or DEEP_ARCHIVE)
export ARCHIVAL_GLACIER_CLASS=GLACIER

# Batch size for operations
export ARCHIVAL_BATCH_SIZE=1000

# Dry run mode for testing
export ARCHIVAL_DRY_RUN=false
```

## S3 Lifecycle Policies

The `SetupLifecyclePolicy()` method creates two S3 lifecycle rules:

### Rule 1: Auto-Transition to Glacier

```json
{
  "Id": "eve-traces-archival",
  "Status": "Enabled",
  "Filter": {
    "Prefix": "eve-traces/"
  },
  "Transitions": [
    {
      "Days": 90,
      "StorageClass": "GLACIER"
    }
  ],
  "Expiration": {
    "Days": 365
  }
}
```

**What it does**:
- After 90 days: Move from S3 Standard → Glacier
- After 365 days: Delete permanently

### Rule 2: Archived Trace Retention

```json
{
  "Id": "eve-traces-archived-retention",
  "Status": "Enabled",
  "Filter": {
    "Prefix": "archived/"
  },
  "Expiration": {
    "Days": 275
  }
}
```

**What it does**:
- Delete traces in `archived/` after 275 days (365 - 90)
- This handles traces we explicitly archived via API

## Archival Process

### Step-by-Step

1. **Identify old traces**: Query `action_executions` for traces > 90 days old
2. **Fetch full trace data**: Get all trace fields from PostgreSQL
3. **Create JSON archive**: Marshal trace data including S3 payload URLs
4. **Upload to S3 Glacier**: Store with metadata (correlation_id, operation_id, etc.)
5. **Mark as archived**: Update PostgreSQL with `archived_at` timestamp and S3 key
6. **S3 lifecycle transitions**: S3 automatically moves to Glacier tier

### Archival Flow

```
PostgreSQL                  S3 Standard               S3 Glacier
   |                            |                         |
   | -- Archive job scans -->   |                         |
   |                            |                         |
   | <-- Fetch trace data --    |                         |
   |                            |                         |
   | -- Upload with Glacier --> |                         |
   |    storage class           |                         |
   |                            | -- Auto-transition -->  |
   |                            |    (after 90 days)      |
   | -- Mark archived_at -->    |                         |
   |                            |                         |
```

## Restore Workflow

### Initiating Restore

```go
// Restore archived trace (e.g., for GDPR data export request)
operationID := "op-xyz789"
days := 7 // Keep restored data accessible for 7 days

err := archival.RestoreArchivedTrace(ctx, operationID, days)
if err != nil {
	log.Errorf("Restore failed: %v", err)
}

// Restore takes 3-5 hours for GLACIER, 12 hours for DEEP_ARCHIVE
```

### Restore Tiers

| Tier | Cost | Time | Use Case |
|------|------|------|----------|
| Expedited | $0.03/GB + $0.01/request | 1-5 minutes | Urgent compliance request |
| Standard | $0.01/GB + $0.05/1000 requests | 3-5 hours | Normal GDPR export |
| Bulk | $0.0025/GB + $0.025/1000 requests | 5-12 hours | Bulk data analysis |

### Check Restore Status

```go
// Check if restore is complete
headOutput, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
	Bucket: aws.String("eve-traces"),
	Key:    aws.String(s3Key),
})

if headOutput.Restore != nil {
	// Parse restore status
	// "ongoing-request=\"true\"" = still restoring
	// "ongoing-request=\"false\", expiry-date=\"...\"" = restored
}
```

### Access Restored Data

```go
// After restore completes, download trace
getOutput, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
	Bucket: aws.String("eve-traces"),
	Key:    aws.String(s3Key),
})

// Read restored trace data
data, _ := io.ReadAll(getOutput.Body)
var trace map[string]interface{}
json.Unmarshal(data, &trace)
```

## Cron Jobs

### Daily Archival Job

```bash
#!/bin/bash
# /etc/cron.daily/eve-archival

# Archive old traces
curl -X POST http://localhost:8080/admin/archival/archive

# Delete very old traces
curl -X POST http://localhost:8080/admin/archival/delete

# Log stats
curl http://localhost:8080/admin/archival/stats >> /var/log/eve/archival.log
```

### Systemd Timer (Alternative)

```ini
# /etc/systemd/system/eve-archival.timer
[Unit]
Description=EVE trace archival timer

[Timer]
OnCalendar=daily
Persistent=true

[Install]
WantedBy=timers.target
```

```ini
# /etc/systemd/system/eve-archival.service
[Unit]
Description=EVE trace archival job

[Service]
Type=oneshot
ExecStart=/usr/local/bin/eve-archival-job
User=eve
Group=eve
```

## REST API Endpoints

```go
// Add to your service
e.POST("/admin/archival/archive", func(c echo.Context) error {
	stats, err := archival.ArchiveOldTraces(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, stats)
})

e.POST("/admin/archival/delete", func(c echo.Context) error {
	stats, err := archival.DeleteOldArchivedTraces(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, stats)
})

e.POST("/admin/archival/restore/:operation_id", func(c echo.Context) error {
	operationID := c.Param("operation_id")
	days := 7 // default

	err := archival.RestoreArchivedTrace(c.Request().Context(), operationID, days)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{
		"message": "Restore initiated",
		"operation_id": operationID,
		"eta": "3-5 hours",
	})
})

e.GET("/admin/archival/stats", func(c echo.Context) error {
	stats, err := archival.GetArchivalStats(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, stats)
})
```

## GDPR Compliance

### Right to Erasure (Article 17)

When a user requests data deletion:

```go
// 1. Find all traces for user
rows, _ := db.Query(`
	SELECT operation_id, archived_at, archived_s3_key
	FROM action_executions
	WHERE metadata->>'user_id' = $1
`, userID)

// 2. Delete active traces
for rows.Next() {
	if !archivedAt.Valid {
		// Delete from PostgreSQL and S3
		deleteTrace(operationID)
	} else {
		// Delete archived trace from Glacier
		s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String("eve-traces"),
			Key:    aws.String(s3Key),
		})
		db.Exec("DELETE FROM action_executions WHERE operation_id = $1", operationID)
	}
}
```

### Right to Access (Article 15)

When a user requests their data:

```go
// 1. Find archived traces
var s3Keys []string
rows, _ := db.Query(`
	SELECT archived_s3_key
	FROM action_executions
	WHERE metadata->>'user_id' = $1
		AND archived_at IS NOT NULL
`, userID)

for rows.Next() {
	var s3Key string
	rows.Scan(&s3Key)
	s3Keys = append(s3Keys, s3Key)
}

// 2. Restore archived traces
for _, s3Key := range s3Keys {
	operationID := extractOperationID(s3Key)
	archival.RestoreArchivedTrace(ctx, operationID, 7)
}

// 3. Wait for restore (3-5 hours)
// 4. Export data to user
```

## Monitoring

### Prometheus Metrics

```go
// In metrics.go
ArchivalOperations: promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "archival_operations_total",
		Help: "Total archival operations",
	},
	[]string{"operation", "status"}, // archive, delete, restore
)

ArchivedTraces: promauto.NewGauge(
	prometheus.GaugeOpts{
		Name: "archived_traces_total",
		Help: "Total number of archived traces",
	},
)

ArchivalDuration: promauto.NewHistogram(
	prometheus.HistogramOpts{
		Name: "archival_duration_seconds",
		Help: "Duration of archival operations",
	},
)
```

### Grafana Dashboard Panel

```yaml
- title: "Archived Traces Over Time"
  targets:
    - expr: "archival_operations_total{operation='archive'}"

- title: "Storage Cost Savings"
  targets:
    - expr: |
        (archived_traces_total * 10 * 1024) * (0.023 - 0.004) / 1024 / 1024 / 1024
      legendFormat: "Monthly Savings (USD)"
```

## Best Practices

### DO

1. **Test with DryRun first**: Use `DryRun: true` to verify what will be archived
2. **Setup lifecycle policies**: Let S3 auto-transition to Glacier
3. **Monitor archival jobs**: Track success rate and duration
4. **Document restore SLA**: 3-5 hours for GDPR requests
5. **Archive metadata**: Keep `archived_at` and `archived_s3_key` in PostgreSQL
6. **Use batch operations**: Archive 1000 traces at a time

### DON'T

1. **Don't archive too aggressively**: 90 days is typical for active debugging
2. **Don't use DEEP_ARCHIVE for short retention**: 12-hour restore is slow
3. **Don't delete without archival**: Always archive before deleting
4. **Don't forget to test restore**: Verify process works before GDPR request
5. **Don't skip lifecycle policies**: Manual archival is expensive
6. **Don't archive without indexing**: Keep searchable metadata in PostgreSQL

## Troubleshooting

### Archival Job Not Running

**Problem**: No traces being archived

**Solution**:
1. Check `Enabled: true` in config
2. Verify traces are > `ArchiveAfterDays` old
3. Check PostgreSQL query results
4. Review logs for errors

### High S3 Costs

**Problem**: S3 bill still high after enabling archival

**Solution**:
1. Verify lifecycle policies are active: `aws s3api get-bucket-lifecycle-configuration --bucket eve-traces`
2. Check storage class distribution: `aws s3api list-objects-v2 --bucket eve-traces | jq '.Contents[].StorageClass'`
3. Wait 90 days for full transition
4. Consider using DEEP_ARCHIVE for cheaper storage

### Restore Failures

**Problem**: Glacier restore not completing

**Solution**:
1. Check restore status with `HeadObject`
2. Verify sufficient time has passed (3-5 hours)
3. Check S3 permissions for restore operations
4. Try `Expedited` tier for faster restore

### Data Loss After Deletion

**Problem**: Need trace that was deleted

**Solution**:
1. **Prevention**: Set `DeleteAfterDays` to longer period (730 days = 2 years)
2. **Prevention**: Backup PostgreSQL metadata before deletion
3. **Recovery**: Not possible - Glacier deletions are permanent
4. **Mitigation**: Use S3 versioning for accidental deletes

## Related Documentation

- S3 Glacier Pricing: https://aws.amazon.com/s3/pricing/
- S3 Lifecycle Policies: https://docs.aws.amazon.com/AmazonS3/latest/userguide/object-lifecycle-mgmt.html
- Glacier Restore: https://docs.aws.amazon.com/AmazonS3/latest/userguide/restoring-objects.html
- Tracing System: `/home/opunix/eve/tracing/`
- Compliance: `/home/opunix/eve/tracing/gdpr.go`
