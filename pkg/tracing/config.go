package tracing

import (
	"database/sql"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Config holds tracer configuration
type Config struct {
	// ServiceID identifies the service (e.g., "containerservice")
	ServiceID string

	// PostgreSQL database connection
	DB *sql.DB

	// S3 client for payload storage
	S3Client *s3.Client

	// S3 bucket name for traces (e.g., "eve-traces")
	S3Bucket string

	// S3 endpoint URL (for Hetzner or MinIO)
	S3Endpoint string

	// Enable/disable tracing
	Enabled bool
}

// Tracer handles action execution tracing
type Tracer struct {
	config Config
}

// New creates a new tracer instance
func New(config Config) *Tracer {
	return &Tracer{config: config}
}

// GetCorrelationID extracts correlation ID from Echo context
func GetCorrelationID(c interface{}) string {
	// Type assertion for Echo context
	if ec, ok := c.(interface{ Get(string) interface{} }); ok {
		if id, ok := ec.Get("correlation_id").(string); ok {
			return id
		}
	}
	return ""
}

// GetOperationID extracts operation ID from Echo context
func GetOperationID(c interface{}) string {
	if ec, ok := c.(interface{ Get(string) interface{} }); ok {
		if id, ok := ec.Get("operation_id").(string); ok {
			return id
		}
	}
	return ""
}
