// Package tracing - Trace data archival to S3 Glacier for long-term storage
package tracing

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ArchivalConfig configures trace archival behavior
type ArchivalConfig struct {
	// Enabled controls whether automatic archival is active
	Enabled bool

	// ArchiveAfterDays specifies how old traces must be before archival (default: 90 days)
	ArchiveAfterDays int

	// DeleteAfterDays specifies when to delete archived traces (default: 365 days)
	// Set to 0 to keep forever
	DeleteAfterDays int

	// S3Bucket for archived traces
	S3Bucket string

	// S3Prefix for archived traces (default: "archived/")
	S3Prefix string

	// GlacierStorageClass controls which Glacier class to use
	// Options: GLACIER, DEEP_ARCHIVE (cheaper but slower restore)
	GlacierStorageClass string

	// BatchSize for archival operations (default: 1000)
	BatchSize int

	// DryRun mode logs what would be archived without doing it
	DryRun bool
}

// ArchivalManager handles trace archival to S3 Glacier
type ArchivalManager struct {
	db       *sql.DB
	s3Client *s3.Client
	config   ArchivalConfig
}

// ArchivalStats tracks archival operation statistics
type ArchivalStats struct {
	TracesArchived      int64
	TracesDeleted       int64
	BytesArchived       int64
	LifecyclePolicies   int
	LastArchivedAt      time.Time
	LastDeletedAt       time.Time
	OldestArchivedTrace time.Time
}

// ArchivedTrace represents a trace in Glacier
type ArchivedTrace struct {
	CorrelationID string    `json:"correlation_id"`
	OperationID   string    `json:"operation_id"`
	ServiceID     string    `json:"service_id"`
	ArchivedAt    time.Time `json:"archived_at"`
	S3Key         string    `json:"s3_key"`
	StorageClass  string    `json:"storage_class"`
	RestoreStatus string    `json:"restore_status"` // "", "in_progress", "restored"
}

// NewArchivalManager creates a new archival manager
func NewArchivalManager(db *sql.DB, s3Client *s3.Client, config ArchivalConfig) *ArchivalManager {
	// Set defaults
	if config.ArchiveAfterDays == 0 {
		config.ArchiveAfterDays = 90
	}
	if config.DeleteAfterDays == 0 {
		config.DeleteAfterDays = 365
	}
	if config.S3Prefix == "" {
		config.S3Prefix = "archived/"
	}
	if config.GlacierStorageClass == "" {
		config.GlacierStorageClass = "GLACIER"
	}
	if config.BatchSize == 0 {
		config.BatchSize = 1000
	}

	return &ArchivalManager{
		db:       db,
		s3Client: s3Client,
		config:   config,
	}
}

// ArchiveOldTraces archives traces older than ArchiveAfterDays
func (am *ArchivalManager) ArchiveOldTraces(ctx context.Context) (*ArchivalStats, error) {
	stats := &ArchivalStats{}

	// Find traces to archive
	cutoffDate := time.Now().AddDate(0, 0, -am.config.ArchiveAfterDays)

	query := `
		SELECT
			correlation_id,
			operation_id,
			service_id,
			started_at,
			request_s3_url,
			response_s3_url
		FROM action_executions
		WHERE started_at < $1
			AND archived_at IS NULL
		ORDER BY started_at
		LIMIT $2
	`

	rows, err := am.db.QueryContext(ctx, query, cutoffDate, am.config.BatchSize)
	if err != nil {
		return nil, fmt.Errorf("query old traces: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			correlationID string
			operationID   string
			serviceID     string
			startedAt     time.Time
			requestS3URL  sql.NullString
			responseS3URL sql.NullString
		)

		if err := rows.Scan(&correlationID, &operationID, &serviceID, &startedAt, &requestS3URL, &responseS3URL); err != nil {
			return stats, fmt.Errorf("scan row: %w", err)
		}

		// Archive this trace
		if err := am.archiveTrace(ctx, correlationID, operationID, serviceID, startedAt, requestS3URL, responseS3URL); err != nil {
			// Log error but continue
			fmt.Printf("Failed to archive trace %s: %v\n", operationID, err)
			continue
		}

		stats.TracesArchived++
		stats.LastArchivedAt = time.Now()

		if stats.OldestArchivedTrace.IsZero() || startedAt.Before(stats.OldestArchivedTrace) {
			stats.OldestArchivedTrace = startedAt
		}
	}

	return stats, rows.Err()
}

// archiveTrace archives a single trace
func (am *ArchivalManager) archiveTrace(ctx context.Context, correlationID, operationID, serviceID string, startedAt time.Time, requestS3URL, responseS3URL sql.NullString) error {
	if am.config.DryRun {
		fmt.Printf("[DRY RUN] Would archive trace: %s (correlation: %s, service: %s)\n", operationID, correlationID, serviceID)
		return nil
	}

	// Create archived metadata
	archived := ArchivedTrace{
		CorrelationID: correlationID,
		OperationID:   operationID,
		ServiceID:     serviceID,
		ArchivedAt:    time.Now(),
		S3Key:         fmt.Sprintf("%s%s/%s.json", am.config.S3Prefix, startedAt.Format("2006/01/02"), operationID),
		StorageClass:  am.config.GlacierStorageClass,
		RestoreStatus: "",
	}

	// Fetch full trace data from PostgreSQL
	var traceData map[string]interface{}
	err := am.db.QueryRowContext(ctx, `
		SELECT row_to_json(action_executions.*)
		FROM action_executions
		WHERE operation_id = $1
	`, operationID).Scan(&traceData)
	if err != nil {
		return fmt.Errorf("fetch trace data: %w", err)
	}

	// Add S3 payload URLs to metadata
	if requestS3URL.Valid {
		traceData["request_s3_url"] = requestS3URL.String
	}
	if responseS3URL.Valid {
		traceData["response_s3_url"] = responseS3URL.String
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(traceData)
	if err != nil {
		return fmt.Errorf("marshal trace: %w", err)
	}

	// Upload to S3 with Glacier storage class
	_, err = am.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(am.config.S3Bucket),
		Key:          aws.String(archived.S3Key),
		Body:         bytes.NewReader(jsonData),
		StorageClass: types.StorageClass(am.config.GlacierStorageClass),
		Metadata: map[string]string{
			"correlation_id": correlationID,
			"operation_id":   operationID,
			"service_id":     serviceID,
			"archived_at":    archived.ArchivedAt.Format(time.RFC3339),
		},
	})
	if err != nil {
		return fmt.Errorf("upload to S3: %w", err)
	}

	// Mark as archived in PostgreSQL
	_, err = am.db.ExecContext(ctx, `
		UPDATE action_executions
		SET archived_at = $1,
		    archived_s3_key = $2
		WHERE operation_id = $3
	`, archived.ArchivedAt, archived.S3Key, operationID)
	if err != nil {
		return fmt.Errorf("mark as archived: %w", err)
	}

	return nil
}

// DeleteOldArchivedTraces deletes archived traces older than DeleteAfterDays
func (am *ArchivalManager) DeleteOldArchivedTraces(ctx context.Context) (*ArchivalStats, error) {
	stats := &ArchivalStats{}

	if am.config.DeleteAfterDays == 0 {
		// Keep forever
		return stats, nil
	}

	// Find archived traces to delete
	cutoffDate := time.Now().AddDate(0, 0, -am.config.DeleteAfterDays)

	query := `
		SELECT
			operation_id,
			archived_s3_key
		FROM action_executions
		WHERE archived_at < $1
			AND archived_at IS NOT NULL
		ORDER BY archived_at
		LIMIT $2
	`

	rows, err := am.db.QueryContext(ctx, query, cutoffDate, am.config.BatchSize)
	if err != nil {
		return nil, fmt.Errorf("query old archived traces: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			operationID string
			s3Key       string
		)

		if err := rows.Scan(&operationID, &s3Key); err != nil {
			return stats, fmt.Errorf("scan row: %w", err)
		}

		if am.config.DryRun {
			fmt.Printf("[DRY RUN] Would delete archived trace: %s (s3://%s/%s)\n", operationID, am.config.S3Bucket, s3Key)
			continue
		}

		// Delete from S3
		_, err := am.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(am.config.S3Bucket),
			Key:    aws.String(s3Key),
		})
		if err != nil {
			fmt.Printf("Failed to delete S3 object %s: %v\n", s3Key, err)
			// Continue anyway
		}

		// Delete from PostgreSQL
		_, err = am.db.ExecContext(ctx, `
			DELETE FROM action_executions
			WHERE operation_id = $1
		`, operationID)
		if err != nil {
			fmt.Printf("Failed to delete trace record %s: %v\n", operationID, err)
			continue
		}

		stats.TracesDeleted++
		stats.LastDeletedAt = time.Now()
	}

	return stats, rows.Err()
}

// RestoreArchivedTrace initiates Glacier restore for a trace
func (am *ArchivalManager) RestoreArchivedTrace(ctx context.Context, operationID string, days int) error {
	// Get S3 key from database
	var s3Key string
	err := am.db.QueryRowContext(ctx, `
		SELECT archived_s3_key
		FROM action_executions
		WHERE operation_id = $1
			AND archived_at IS NOT NULL
	`, operationID).Scan(&s3Key)
	if err != nil {
		return fmt.Errorf("trace not found or not archived: %w", err)
	}

	// Initiate Glacier restore
	_, err = am.s3Client.RestoreObject(ctx, &s3.RestoreObjectInput{
		Bucket: aws.String(am.config.S3Bucket),
		Key:    aws.String(s3Key),
		RestoreRequest: &types.RestoreRequest{
			Days: aws.Int32(int32(days)),
			GlacierJobParameters: &types.GlacierJobParameters{
				Tier: types.TierStandard, // or Expedited for faster (more expensive) restore
			},
		},
	})
	if err != nil {
		return fmt.Errorf("initiate restore: %w", err)
	}

	return nil
}

// SetupLifecyclePolicy creates S3 lifecycle policy for automatic archival
func (am *ArchivalManager) SetupLifecyclePolicy(ctx context.Context) error {
	// Create lifecycle rule for automatic transition to Glacier
	lifecycleConfig := &s3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(am.config.S3Bucket),
		LifecycleConfiguration: &types.BucketLifecycleConfiguration{
			Rules: []types.LifecycleRule{
				{
					ID:     aws.String("eve-traces-archival"),
					Status: types.ExpirationStatusEnabled,
					Filter: &types.LifecycleRuleFilter{
						Prefix: aws.String("eve-traces/"), // Standard traces
					},
					Transitions: []types.Transition{
						{
							Days:         aws.Int32(int32(am.config.ArchiveAfterDays)),
							StorageClass: types.TransitionStorageClass(am.config.GlacierStorageClass),
						},
					},
					Expiration: func() *types.LifecycleExpiration {
						if am.config.DeleteAfterDays > 0 {
							return &types.LifecycleExpiration{
								Days: aws.Int32(int32(am.config.DeleteAfterDays)),
							}
						}
						return nil
					}(),
				},
				{
					ID:     aws.String("eve-traces-archived-retention"),
					Status: types.ExpirationStatusEnabled,
					Filter: &types.LifecycleRuleFilter{
						Prefix: aws.String(am.config.S3Prefix), // Already archived traces
					},
					Expiration: func() *types.LifecycleExpiration {
						if am.config.DeleteAfterDays > 0 {
							return &types.LifecycleExpiration{
								Days: aws.Int32(int32(am.config.DeleteAfterDays - am.config.ArchiveAfterDays)),
							}
						}
						return nil
					}(),
				},
			},
		},
	}

	_, err := am.s3Client.PutBucketLifecycleConfiguration(ctx, lifecycleConfig)
	if err != nil {
		return fmt.Errorf("setup lifecycle policy: %w", err)
	}

	return nil
}

// GetArchivalStats returns archival statistics
func (am *ArchivalManager) GetArchivalStats(ctx context.Context) (*ArchivalStats, error) {
	stats := &ArchivalStats{}

	// Count archived traces
	err := am.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as total,
			MIN(started_at) as oldest
		FROM action_executions
		WHERE archived_at IS NOT NULL
	`).Scan(&stats.TracesArchived, &stats.OldestArchivedTrace)
	if err != nil {
		return nil, fmt.Errorf("query archived stats: %w", err)
	}

	// Get last archival time
	err = am.db.QueryRowContext(ctx, `
		SELECT MAX(archived_at)
		FROM action_executions
		WHERE archived_at IS NOT NULL
	`).Scan(&stats.LastArchivedAt)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("query last archival: %w", err)
	}

	return stats, nil
}
