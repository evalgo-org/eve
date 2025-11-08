package tracing

import (
	"database/sql"
	"os"
	"strings"

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

	// ExcludeActionTypes lists action types to exclude from tracing (e.g., ["WaitAction"])
	ExcludeActionTypes []string

	// ExcludeObjectTypes lists object types to exclude from tracing (e.g., ["Credential"])
	ExcludeObjectTypes []string

	// StorePayloads controls whether to store request/response bodies in S3
	// Automatically disabled for credential-related actions
	StorePayloads bool
}

// Tracer handles action execution tracing
type Tracer struct {
	config Config
}

// New creates a new tracer instance
func New(config Config) *Tracer {
	return &Tracer{config: config}
}

// NewFromEnv creates a tracer instance with configuration from environment variables
// Required: serviceID, db, s3Client
// Environment variables:
//   - TRACING_ENABLED: Enable/disable tracing (default: true)
//   - TRACING_STORE_PAYLOADS: Store request/response in S3 (default: false)
//   - TRACING_EXCLUDE_ACTIONS: Comma-separated action types to exclude (e.g., "WaitAction,SearchAction")
//   - TRACING_EXCLUDE_OBJECTS: Comma-separated object types to exclude (e.g., "Database,DataFeed")
//   - S3_BUCKET: S3 bucket name (default: eve-traces)
//   - S3_ENDPOINT_URL: S3 endpoint URL (optional, for Hetzner/MinIO)
func NewFromEnv(serviceID string, db *sql.DB, s3Client *s3.Client) *Tracer {
	config := Config{
		ServiceID: serviceID,
		DB:        db,
		S3Client:  s3Client,
	}

	// Parse enabled flag (default: true)
	config.Enabled = os.Getenv("TRACING_ENABLED") != "false"

	// Parse payload storage (default: false for security)
	config.StorePayloads = os.Getenv("TRACING_STORE_PAYLOADS") == "true"

	// Parse S3 bucket
	config.S3Bucket = os.Getenv("S3_BUCKET")
	if config.S3Bucket == "" {
		config.S3Bucket = "eve-traces"
	}

	// Parse S3 endpoint
	config.S3Endpoint = os.Getenv("S3_ENDPOINT_URL")

	// Parse exclusion lists
	if excludeActions := os.Getenv("TRACING_EXCLUDE_ACTIONS"); excludeActions != "" {
		config.ExcludeActionTypes = strings.Split(excludeActions, ",")
		// Trim whitespace from each entry
		for i := range config.ExcludeActionTypes {
			config.ExcludeActionTypes[i] = strings.TrimSpace(config.ExcludeActionTypes[i])
		}
	}

	if excludeObjects := os.Getenv("TRACING_EXCLUDE_OBJECTS"); excludeObjects != "" {
		config.ExcludeObjectTypes = strings.Split(excludeObjects, ",")
		// Trim whitespace from each entry
		for i := range config.ExcludeObjectTypes {
			config.ExcludeObjectTypes[i] = strings.TrimSpace(config.ExcludeObjectTypes[i])
		}
	}

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

// shouldTrace checks if an action should be traced based on exclusion rules
func (t *Tracer) shouldTrace(actionType, objectType string) bool {
	// Check if action type is excluded
	for _, excluded := range t.config.ExcludeActionTypes {
		if actionType == excluded {
			return false
		}
	}

	// Check if object type is excluded
	for _, excluded := range t.config.ExcludeObjectTypes {
		if objectType == excluded {
			return false
		}
	}

	return true
}

// isCredentialRelated checks if an action involves credentials or secrets
func isCredentialRelated(actionType, objectType string) bool {
	// Check object type
	credentialTypes := []string{
		"Credential",
		"PasswordCredential",
		"Secret",
		"DigitalDocument", // Can contain credentials
	}

	for _, t := range credentialTypes {
		if objectType == t {
			return true
		}
	}

	return false
}

// shouldStorePayload checks if request/response payloads should be stored in S3
func (t *Tracer) shouldStorePayload(actionType, objectType string) bool {
	// Never store credential payloads
	if isCredentialRelated(actionType, objectType) {
		return false
	}

	// Respect global StorePayloads config
	return t.config.StorePayloads
}
