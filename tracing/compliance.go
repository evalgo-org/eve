// Package tracing - GDPR and compliance features
package tracing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/lib/pq"
)

// PIIPattern defines patterns for detecting PII
type PIIPattern struct {
	Type       string  // email, phone, ssn, credit_card, etc.
	Pattern    string  // regex pattern
	Confidence float64 // 0.0-1.0
}

// Common PII patterns
var DefaultPIIPatterns = []PIIPattern{
	{Type: "email", Pattern: `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, Confidence: 0.95},
	{Type: "phone", Pattern: `\b(\+\d{1,3}[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b`, Confidence: 0.85},
	{Type: "ssn", Pattern: `\b\d{3}-\d{2}-\d{4}\b`, Confidence: 0.95},
	{Type: "credit_card", Pattern: `\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`, Confidence: 0.90},
	{Type: "ip_address", Pattern: `\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`, Confidence: 0.80},
	{Type: "passport", Pattern: `\b[A-Z]{1,2}\d{6,9}\b`, Confidence: 0.70},
	{Type: "iban", Pattern: `\b[A-Z]{2}\d{2}[A-Z0-9]{1,30}\b`, Confidence: 0.75},
}

// PIIDetection represents a detected PII occurrence
type PIIDetection struct {
	CorrelationID  string
	OperationID    string
	Location       string // request, response, metadata
	FieldPath      string
	PIIType        string
	PatternMatched string
	Confidence     float64
	Redacted       bool
	Token          string
	DataSubjectID  string
}

// ErasureResult contains results of GDPR erasure operation
type ErasureResult struct {
	DeletedActions int
	DeletedPII     int
	S3URLsToDelete []string
	ErasureCertID  string
}

// ComplianceConfig holds configuration for compliance features
type ComplianceConfig struct {
	EnablePIIDetection bool
	EnableAuditLogging bool
	DataRegion         string // us, eu, apac
	RetentionDays      int
	PIIPatterns        []PIIPattern
}

// EraseTraces implements GDPR Right to Erasure (Article 17)
// Deletes all traces for a data subject or correlation ID from PostgreSQL and S3
func (t *Tracer) EraseTraces(ctx context.Context, dataSubjectID, correlationID, userID, purpose string) (*ErasureResult, error) {
	if dataSubjectID == "" && correlationID == "" {
		return nil, fmt.Errorf("must provide either data_subject_id or correlation_id")
	}

	// Call PostgreSQL function to perform erasure
	query := `SELECT * FROM gdpr_erase_traces($1, $2, $3, $4)`

	var result ErasureResult
	var s3URLsArray pq.StringArray

	row := t.config.DB.QueryRowContext(ctx, query,
		nullString(dataSubjectID),
		nullString(correlationID),
		userID,
		purpose,
	)

	err := row.Scan(
		&result.DeletedActions,
		&result.DeletedPII,
		&s3URLsArray,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute erasure: %w", err)
	}

	// Convert SQL array to string slice
	for _, url := range s3URLsArray {
		if url != "" {
			result.S3URLsToDelete = append(result.S3URLsToDelete, url)
		}
	}

	// Delete S3 objects
	if t.config.S3Client != nil {
		for _, s3URL := range result.S3URLsToDelete {
			if err := t.deleteS3Object(ctx, s3URL); err != nil {
				t.logError(fmt.Sprintf("Failed to delete S3 object: %s", s3URL), err)
				// Continue with other deletions
			}
		}
	}

	// Generate erasure certificate ID
	result.ErasureCertID = fmt.Sprintf("ERASURE-%s", generateID())

	return &result, nil
}

// PseudonymizeTraces implements GDPR pseudonymization (Article 17 alternative)
// Replaces identifiable data with hashes instead of full deletion
func (t *Tracer) PseudonymizeTraces(ctx context.Context, dataSubjectID, userID string) (int, error) {
	if dataSubjectID == "" {
		return 0, fmt.Errorf("data_subject_id is required")
	}

	query := `SELECT gdpr_pseudonymize_traces($1, $2)`

	var updated int
	err := t.config.DB.QueryRowContext(ctx, query, dataSubjectID, userID).Scan(&updated)
	if err != nil {
		return 0, fmt.Errorf("failed to pseudonymize traces: %w", err)
	}

	return updated, nil
}

// ExportDataSubjectData implements GDPR Right to Data Portability (Article 20)
// Returns all data for a data subject in structured format
func (t *Tracer) ExportDataSubjectData(ctx context.Context, dataSubjectID string) ([]map[string]interface{}, error) {
	if dataSubjectID == "" {
		return nil, fmt.Errorf("data_subject_id is required")
	}

	query := `SELECT * FROM gdpr_export_data($1)`

	rows, err := t.config.DB.QueryContext(ctx, query, dataSubjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to export data: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}

	for rows.Next() {
		var (
			correlationID sql.NullString
			operationID   sql.NullString
			actionType    sql.NullString
			startedAt     sql.NullTime
			durationMS    sql.NullInt64
			actionStatus  sql.NullString
			serviceID     sql.NullString
			metadata      []byte
			requestURL    sql.NullString
			responseURL   sql.NullString
		)

		err := rows.Scan(
			&correlationID, &operationID, &actionType,
			&startedAt, &durationMS, &actionStatus,
			&serviceID, &metadata, &requestURL, &responseURL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		result := map[string]interface{}{
			"correlation_id": correlationID.String,
			"operation_id":   operationID.String,
			"action_type":    actionType.String,
			"started_at":     startedAt.Time,
			"duration_ms":    durationMS.Int64,
			"action_status":  actionStatus.String,
			"service_id":     serviceID.String,
			"request_url":    requestURL.String,
			"response_url":   responseURL.String,
		}

		// Parse metadata JSON
		if len(metadata) > 0 {
			result["metadata"] = string(metadata)
		}

		results = append(results, result)
	}

	return results, nil
}

// LogTraceAccess logs access to traces for audit trail
func (t *Tracer) LogTraceAccess(ctx context.Context, userID, accessType, resourceType string,
	correlationID, operationID, dataSubjectID string, resultsCount int, purpose string,
	queryParams map[string]interface{}) error {

	query := `SELECT log_trace_access($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	// Convert query params to JSON
	var paramsJSON []byte
	if queryParams != nil {
		var err error
		paramsJSON, err = json.Marshal(queryParams)
		if err != nil {
			return fmt.Errorf("failed to marshal query params: %w", err)
		}
	}

	var auditID string
	err := t.config.DB.QueryRowContext(ctx, query,
		userID,
		accessType,
		resourceType,
		nullString(correlationID),
		nullString(operationID),
		nullString(dataSubjectID),
		nullInt(resultsCount),
		nullString(purpose),
		paramsJSON,
	).Scan(&auditID)

	if err != nil {
		return fmt.Errorf("failed to log trace access: %w", err)
	}

	return nil
}

// GetAuditTrail retrieves audit trail for a data subject or correlation ID
func (t *Tracer) GetAuditTrail(ctx context.Context, dataSubjectID, correlationID string, hours int) ([]map[string]interface{}, error) {
	query := `SELECT * FROM get_audit_trail($1, $2, $3)`

	rows, err := t.config.DB.QueryContext(ctx, query,
		nullString(dataSubjectID),
		nullString(correlationID),
		hours,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit trail: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}

	for rows.Next() {
		var (
			accessedAt   sql.NullTime
			userID       sql.NullString
			accessType   sql.NullString
			resourceType sql.NullString
			purpose      sql.NullString
			resultsCount sql.NullInt64
		)

		err := rows.Scan(&accessedAt, &userID, &accessType, &resourceType, &purpose, &resultsCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit row: %w", err)
		}

		results = append(results, map[string]interface{}{
			"accessed_at":   accessedAt.Time,
			"user_id":       userID.String,
			"access_type":   accessType.String,
			"resource_type": resourceType.String,
			"purpose":       purpose.String,
			"results_count": resultsCount.Int64,
		})
	}

	return results, nil
}

// DetectPII scans data for PII patterns
func (t *Tracer) DetectPII(data string, patterns []PIIPattern) []PIIDetection {
	if patterns == nil {
		patterns = DefaultPIIPatterns
	}

	var detections []PIIDetection

	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern.Pattern)
		if err != nil {
			continue
		}

		matches := re.FindAllString(data, -1)
		for _, match := range matches {
			detections = append(detections, PIIDetection{
				PIIType:        pattern.Type,
				PatternMatched: match,
				Confidence:     pattern.Confidence,
			})
		}
	}

	return detections
}

// RecordPIIDetection stores PII detection in database
func (t *Tracer) RecordPIIDetection(ctx context.Context, detection PIIDetection) error {
	query := `
		INSERT INTO pii_detections (
			correlation_id, operation_id, location, field_path,
			pii_type, pattern_matched, confidence,
			redacted, token, data_subject_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := t.config.DB.ExecContext(ctx, query,
		detection.CorrelationID,
		detection.OperationID,
		detection.Location,
		nullString(detection.FieldPath),
		detection.PIIType,
		detection.PatternMatched,
		detection.Confidence,
		detection.Redacted,
		nullString(detection.Token),
		nullString(detection.DataSubjectID),
	)

	if err != nil {
		return fmt.Errorf("failed to record PII detection: %w", err)
	}

	return nil
}

// RedactPII replaces PII with redaction markers
func RedactPII(data string, detections []PIIDetection) string {
	redacted := data

	for _, detection := range detections {
		if detection.Confidence >= 0.85 {
			// High confidence - redact
			replacement := fmt.Sprintf("[REDACTED_%s]", strings.ToUpper(detection.PIIType))
			redacted = strings.ReplaceAll(redacted, detection.PatternMatched, replacement)
		}
	}

	return redacted
}

// deleteS3Object deletes an S3 object by S3 URL
func (t *Tracer) deleteS3Object(ctx context.Context, s3URL string) error {
	// Parse S3 URL: s3://bucket/key
	if !strings.HasPrefix(s3URL, "s3://") {
		return fmt.Errorf("invalid S3 URL: %s", s3URL)
	}

	parts := strings.SplitN(strings.TrimPrefix(s3URL, "s3://"), "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid S3 URL format: %s", s3URL)
	}

	bucket := parts[0]
	key := parts[1]

	_, err := t.config.S3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	return err
}

// DeleteExpiredTraces runs retention policy enforcement
func (t *Tracer) DeleteExpiredTraces(ctx context.Context) (int, error) {
	query := `SELECT delete_expired_traces()`

	var deleted int
	err := t.config.DB.QueryRowContext(ctx, query).Scan(&deleted)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired traces: %w", err)
	}

	return deleted, nil
}

// Helper functions

func nullInt(i int) interface{} {
	if i == 0 {
		return nil
	}
	return i
}

// generateID generates a unique ID for erasure certificates
func generateID() string {
	return fmt.Sprintf("%d", time.Now().Unix())
}
