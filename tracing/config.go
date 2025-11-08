package tracing

import (
	"context"
	"database/sql"
	"os"
	"strconv"
	"strings"
	"time"

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

	// GDPR & Compliance settings
	DataRegion    string // us, eu, apac - controls data residency
	RetentionDays int    // Default retention period in days
	EnablePII     bool   // Enable PII detection
	LegalBasis    string // Default legal basis for processing

	// Async export settings
	AsyncExport bool                // Enable async trace export (default: true)
	AsyncConfig AsyncExporterConfig // Async exporter configuration

	// Metrics settings
	EnableMetrics    bool   // Enable Prometheus metrics (default: true)
	MetricsNamespace string // Prometheus namespace (default: "eve_tracing")

	// Sampling settings
	SamplingEnabled bool           // Enable tail-based sampling (default: false)
	SamplingConfig  SamplingConfig // Sampling configuration
}

// Tracer handles action execution tracing
type Tracer struct {
	config        Config
	asyncExporter *AsyncExporter // Optional async exporter
	metrics       *Metrics       // Optional Prometheus metrics
	sampler       *Sampler       // Optional tail-based sampler
}

// New creates a new tracer instance
func New(config Config) *Tracer {
	tracer := &Tracer{config: config}

	// Initialize Prometheus metrics if enabled
	if config.EnableMetrics {
		namespace := config.MetricsNamespace
		if namespace == "" {
			namespace = "eve_tracing"
		}
		tracer.metrics = NewMetrics(namespace)
	}

	// Initialize async exporter if enabled
	if config.AsyncExport {
		tracer.asyncExporter = NewAsyncExporter(tracer, config.AsyncConfig)
	}

	// Initialize sampler if enabled
	if config.SamplingEnabled {
		tracer.sampler = NewSampler(config.SamplingConfig)
	}

	return tracer
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

	// Parse GDPR compliance settings
	config.DataRegion = os.Getenv("DATA_REGION")
	if config.DataRegion == "" {
		config.DataRegion = "us" // Default
	}

	// Parse retention days (default: 90 days)
	config.RetentionDays = 90
	if retDays := os.Getenv("TRACING_RETENTION_DAYS"); retDays != "" {
		if days, err := strconv.Atoi(retDays); err == nil && days > 0 {
			config.RetentionDays = days
		}
	}

	// Parse PII detection setting (default: true)
	config.EnablePII = os.Getenv("TRACING_ENABLE_PII") != "false"

	// Parse legal basis
	config.LegalBasis = os.Getenv("TRACING_LEGAL_BASIS")
	if config.LegalBasis == "" {
		config.LegalBasis = "Legitimate Interest" // Default
	}

	// Parse async export settings (enabled by default)
	config.AsyncExport = os.Getenv("TRACING_ASYNC_EXPORT") != "false"

	// Parse async config
	if workers := os.Getenv("TRACING_ASYNC_WORKERS"); workers != "" {
		if w, err := strconv.Atoi(workers); err == nil && w > 0 {
			config.AsyncConfig.Workers = w
		}
	}
	if queueSize := os.Getenv("TRACING_ASYNC_QUEUE_SIZE"); queueSize != "" {
		if q, err := strconv.Atoi(queueSize); err == nil && q > 0 {
			config.AsyncConfig.QueueSize = q
		}
	}
	if batchSize := os.Getenv("TRACING_ASYNC_BATCH_SIZE"); batchSize != "" {
		if b, err := strconv.Atoi(batchSize); err == nil && b > 0 {
			config.AsyncConfig.BatchSize = b
		}
	}
	if flushPeriod := os.Getenv("TRACING_ASYNC_FLUSH_PERIOD"); flushPeriod != "" {
		if d, err := time.ParseDuration(flushPeriod); err == nil {
			config.AsyncConfig.FlushPeriod = d
		}
	}

	// Parse metrics settings (enabled by default)
	config.EnableMetrics = os.Getenv("TRACING_METRICS_ENABLED") != "false"
	config.MetricsNamespace = os.Getenv("TRACING_METRICS_NAMESPACE")
	if config.MetricsNamespace == "" {
		config.MetricsNamespace = "eve_tracing"
	}

	return New(config)
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

// Shutdown gracefully shuts down the tracer and async exporter
func (t *Tracer) Shutdown(ctx context.Context) error {
	if t.asyncExporter != nil {
		return t.asyncExporter.Shutdown(ctx)
	}
	return nil
}

// Stats returns async exporter statistics (if enabled)
func (t *Tracer) Stats() *ExporterStats {
	if t.asyncExporter != nil {
		stats := t.asyncExporter.Stats()
		return &stats
	}
	return nil
}

// IsHealthy returns true if tracer is functioning properly
func (t *Tracer) IsHealthy() bool {
	if t.asyncExporter != nil {
		return t.asyncExporter.IsHealthy()
	}
	return true
}
