// Package tracing - Prometheus metrics instrumentation
package tracing

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for tracing
type Metrics struct {
	// Action execution metrics
	ActionDuration    *prometheus.HistogramVec
	ActionCounter     *prometheus.CounterVec
	ActionErrors      *prometheus.CounterVec
	ActionStatusGauge *prometheus.GaugeVec

	// Business workflow metrics
	WorkflowDuration     *prometheus.HistogramVec
	WorkflowCounter      *prometheus.CounterVec
	WorkflowInFlight     *prometheus.GaugeVec
	WorkflowStepDuration *prometheus.HistogramVec

	// Async exporter metrics
	ExporterQueueSize    prometheus.Gauge
	ExporterQueueDropped *prometheus.CounterVec
	ExporterBatches      *prometheus.CounterVec
	ExporterLatency      prometheus.Histogram

	// GDPR compliance metrics
	GDPRErasures     *prometheus.CounterVec
	GDPRExports      *prometheus.CounterVec
	PIIDetections    *prometheus.CounterVec
	AuditAccess      *prometheus.CounterVec
	RetentionCleanup prometheus.Counter

	// Trace storage metrics
	TracePayloadSize *prometheus.HistogramVec
	S3Uploads        *prometheus.CounterVec
	S3Errors         *prometheus.CounterVec
	PostgreSQLWrites *prometheus.CounterVec
	PostgreSQLErrors *prometheus.CounterVec

	// OpenTelemetry integration metrics
	OTelTraceLinks   *prometheus.CounterVec
	OTelSamplingRate prometheus.Gauge

	// Sampling metrics
	SamplingDecisions *prometheus.CounterVec

	// Dependency metrics
	DependencyCalls   *prometheus.CounterVec
	DependencyLatency *prometheus.HistogramVec
}

// NewMetrics creates and registers Prometheus metrics
func NewMetrics(namespace string) *Metrics {
	if namespace == "" {
		namespace = "eve_tracing"
	}

	m := &Metrics{
		// Action execution metrics
		ActionDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "action_duration_seconds",
				Help:      "Duration of semantic action execution in seconds",
				Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"action_type", "object_type", "service_id", "status"},
		),

		ActionCounter: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "actions_total",
				Help:      "Total number of semantic actions executed",
			},
			[]string{"action_type", "object_type", "service_id", "status"},
		),

		ActionErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "action_errors_total",
				Help:      "Total number of action execution errors",
			},
			[]string{"action_type", "object_type", "service_id", "error_type"},
		),

		ActionStatusGauge: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "action_status",
				Help:      "Current action status (1=completed, 0=failed, 0.5=active)",
			},
			[]string{"correlation_id", "operation_id"},
		),

		// Workflow metrics
		WorkflowDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "workflow_duration_seconds",
				Help:      "Total workflow duration from first to last action",
				Buckets:   []float64{.1, .5, 1, 5, 10, 30, 60, 120, 300, 600},
			},
			[]string{"correlation_id", "status"},
		),

		WorkflowCounter: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "workflows_total",
				Help:      "Total number of workflows completed",
			},
			[]string{"status"},
		),

		WorkflowInFlight: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "workflows_in_flight",
				Help:      "Number of workflows currently executing",
			},
			[]string{"correlation_id"},
		),

		WorkflowStepDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "workflow_step_duration_seconds",
				Help:      "Duration of individual workflow steps",
				Buckets:   []float64{.01, .05, .1, .5, 1, 2, 5},
			},
			[]string{"correlation_id", "step_index", "action_type"},
		),

		// Async exporter metrics
		ExporterQueueSize: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "exporter_queue_size",
				Help:      "Current number of traces in export queue",
			},
		),

		ExporterQueueDropped: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "exporter_queue_dropped_total",
				Help:      "Total number of traces dropped due to full queue",
			},
			[]string{"service_id"},
		),

		ExporterBatches: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "exporter_batches_total",
				Help:      "Total number of trace batches exported",
			},
			[]string{"service_id", "status"},
		),

		ExporterLatency: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "exporter_latency_seconds",
				Help:      "Time to export a batch of traces",
				Buckets:   prometheus.DefBuckets,
			},
		),

		// GDPR compliance metrics
		GDPRErasures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "gdpr_erasures_total",
				Help:      "Total number of GDPR erasure requests processed",
			},
			[]string{"user_id", "reason"},
		),

		GDPRExports: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "gdpr_exports_total",
				Help:      "Total number of GDPR data export requests",
			},
			[]string{"data_subject_id"},
		),

		PIIDetections: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "pii_detections_total",
				Help:      "Total number of PII detections",
			},
			[]string{"pii_type", "location", "redacted"},
		),

		AuditAccess: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "audit_access_total",
				Help:      "Total number of trace access events",
			},
			[]string{"user_id", "access_type", "resource_type"},
		),

		RetentionCleanup: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "retention_cleanup_total",
				Help:      "Total number of traces deleted by retention policy",
			},
		),

		// Storage metrics
		TracePayloadSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "trace_payload_bytes",
				Help:      "Size of trace payloads in bytes",
				Buckets:   []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000},
			},
			[]string{"type"},
		),

		S3Uploads: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "s3_uploads_total",
				Help:      "Total number of S3 uploads",
			},
			[]string{"bucket", "status"},
		),

		S3Errors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "s3_errors_total",
				Help:      "Total number of S3 errors",
			},
			[]string{"operation", "error_type"},
		),

		PostgreSQLWrites: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "postgresql_writes_total",
				Help:      "Total number of PostgreSQL writes",
			},
			[]string{"table", "status"},
		),

		PostgreSQLErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "postgresql_errors_total",
				Help:      "Total number of PostgreSQL errors",
			},
			[]string{"operation", "error_type"},
		),

		// OpenTelemetry integration
		OTelTraceLinks: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "otel_trace_links_total",
				Help:      "Total number of traces linked to OpenTelemetry",
			},
			[]string{"service_id"},
		),

		OTelSamplingRate: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "otel_sampling_rate",
				Help:      "Current OpenTelemetry sampling rate (0.0-1.0)",
			},
		),

		// Sampling metrics
		SamplingDecisions: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "sampling_decisions_total",
				Help:      "Total number of sampling decisions made",
			},
			[]string{"service_id", "decision", "reason"},
		),

		// Dependency metrics
		DependencyCalls: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "dependency_calls_total",
				Help:      "Total calls between services",
			},
			[]string{"from_service", "to_service", "status"},
		),

		DependencyLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "dependency_latency_seconds",
				Help:      "Latency of service-to-service calls",
				Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"from_service", "to_service"},
		),
	}

	return m
}

// RecordAction records metrics for an action execution
func (m *Metrics) RecordAction(actionType, objectType, serviceID, status string, duration time.Duration) {
	m.ActionDuration.WithLabelValues(actionType, objectType, serviceID, status).Observe(duration.Seconds())
	m.ActionCounter.WithLabelValues(actionType, objectType, serviceID, status).Inc()
}

// RecordActionError records an action error
func (m *Metrics) RecordActionError(actionType, objectType, serviceID, errorType string) {
	m.ActionErrors.WithLabelValues(actionType, objectType, serviceID, errorType).Inc()
}

// RecordWorkflow records workflow completion
func (m *Metrics) RecordWorkflow(correlationID, status string, duration time.Duration) {
	m.WorkflowDuration.WithLabelValues(correlationID, status).Observe(duration.Seconds())
	m.WorkflowCounter.WithLabelValues(status).Inc()
}

// UpdateExporterMetrics updates async exporter metrics
func (m *Metrics) UpdateExporterMetrics(stats *ExporterStats, queueSize int) {
	m.ExporterQueueSize.Set(float64(queueSize))
}

// RecordPIIDetection records a PII detection event
func (m *Metrics) RecordPIIDetection(piiType, location string, redacted bool) {
	redactedStr := "false"
	if redacted {
		redactedStr = "true"
	}
	m.PIIDetections.WithLabelValues(piiType, location, redactedStr).Inc()
}

// RecordGDPRErasure records a GDPR erasure request
func (m *Metrics) RecordGDPRErasure(userID, reason string) {
	m.GDPRErasures.WithLabelValues(userID, reason).Inc()
}

// RecordS3Upload records an S3 upload operation
func (m *Metrics) RecordS3Upload(bucket, status string) {
	m.S3Uploads.WithLabelValues(bucket, status).Inc()
}

// RecordPostgreSQLWrite records a PostgreSQL write
func (m *Metrics) RecordPostgreSQLWrite(table, status string) {
	m.PostgreSQLWrites.WithLabelValues(table, status).Inc()
}
