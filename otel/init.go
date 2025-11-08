// Package otel provides OpenTelemetry initialization and instrumentation for EVE services
package otel

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// Config holds OpenTelemetry configuration
type Config struct {
	ServiceName string
	ServiceID   string
	Version     string

	// OTLP endpoint (Jaeger, Tempo, etc.)
	// Default: http://localhost:4318 (Jaeger OTLP HTTP)
	OTLPEndpoint string

	// Enable/disable OpenTelemetry
	Enabled bool

	// Sampling ratio (0.0 to 1.0)
	// 1.0 = trace everything, 0.1 = trace 10%
	SamplingRatio float64

	// Environment (production, staging, development)
	Environment string
}

// Provider wraps the OpenTelemetry TracerProvider
type Provider struct {
	tp *sdktrace.TracerProvider
}

// Init initializes OpenTelemetry from environment variables
// Environment variables:
//   - OTEL_ENABLED: Enable/disable OTel (default: true)
//   - OTEL_EXPORTER_OTLP_ENDPOINT: OTLP endpoint (default: http://localhost:4318)
//   - OTEL_SERVICE_NAME: Service name (override serviceID)
//   - OTEL_SAMPLING_RATIO: Sampling ratio 0.0-1.0 (default: 1.0)
//   - OTEL_ENVIRONMENT: Environment name (default: development)
func Init(serviceID, version string) *Provider {
	config := Config{
		ServiceID:   serviceID,
		ServiceName: serviceID,
		Version:     version,
	}

	// Parse environment variables
	config.Enabled = os.Getenv("OTEL_ENABLED") != "false"
	if !config.Enabled {
		log.Println("⚠️  OpenTelemetry explicitly disabled via OTEL_ENABLED=false")
		return nil
	}

	// OTLP endpoint
	config.OTLPEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if config.OTLPEndpoint == "" {
		config.OTLPEndpoint = "http://localhost:4318" // Jaeger OTLP HTTP default
	}

	// Service name override
	if name := os.Getenv("OTEL_SERVICE_NAME"); name != "" {
		config.ServiceName = name
	}

	// Sampling ratio
	config.SamplingRatio = 1.0 // Default: trace everything
	if ratio := os.Getenv("OTEL_SAMPLING_RATIO"); ratio != "" {
		if _, err := fmt.Sscanf(ratio, "%f", &config.SamplingRatio); err != nil {
			log.Printf("⚠️  Invalid OTEL_SAMPLING_RATIO: %s, using 1.0", ratio)
		}
	}

	// Environment
	config.Environment = os.Getenv("OTEL_ENVIRONMENT")
	if config.Environment == "" {
		config.Environment = "development"
	}

	// Initialize provider
	provider, err := NewProvider(config)
	if err != nil {
		log.Printf("⚠️  OpenTelemetry initialization failed: %v", err)
		return nil
	}

	log.Printf("✓ OpenTelemetry initialized for %s (endpoint: %s, sampling: %.2f)",
		config.ServiceName, config.OTLPEndpoint, config.SamplingRatio)

	return provider
}

// NewProvider creates a new OpenTelemetry provider with the given configuration
func NewProvider(config Config) (*Provider, error) {
	ctx := context.Background()

	// Create OTLP HTTP exporter
	exporter, err := otlptrace.New(
		ctx,
		otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(stripProtocol(config.OTLPEndpoint)),
			otlptracehttp.WithInsecure(), // Use HTTPS in production
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.Version),
			semconv.DeploymentEnvironmentKey.String(config.Environment),
		),
		resource.WithProcess(),
		resource.WithHost(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create sampler
	var sampler sdktrace.Sampler
	if config.SamplingRatio >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if config.SamplingRatio <= 0.0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(config.SamplingRatio)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global provider
	otel.SetTracerProvider(tp)

	// Set global propagators (W3C Trace Context + Baggage)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Provider{tp: tp}, nil
}

// Shutdown gracefully shuts down the provider
func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil || p.tp == nil {
		return nil
	}

	// Give traces 5 seconds to flush
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return p.tp.Shutdown(shutdownCtx)
}

// stripProtocol removes http:// or https:// from endpoint
func stripProtocol(endpoint string) string {
	if len(endpoint) > 7 && endpoint[:7] == "http://" {
		return endpoint[7:]
	}
	if len(endpoint) > 8 && endpoint[:8] == "https://" {
		return endpoint[8:]
	}
	return endpoint
}
