package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"eve.evalgo.org/tracing"
)

func main() {
	// Load configuration from environment
	cfg := loadConfig()

	// Initialize database
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize S3 client
	s3Client := createS3Client(cfg)

	// Initialize tracer
	var tracer *tracing.Tracer
	if cfg.TracingEnabled {
		tracingConfig := tracing.Config{
			ServiceID: cfg.ServiceID,
			DB:        db,
			S3Client:  s3Client,
			S3Bucket:  cfg.S3Bucket,
			Enabled:   true,

			// Async export
			AsyncExport: cfg.AsyncExportEnabled,
			AsyncConfig: tracing.AsyncExporterConfig{
				QueueSize:   cfg.AsyncQueueSize,
				BatchSize:   cfg.AsyncBatchSize,
				Workers:     cfg.AsyncWorkerCount,
				FlushPeriod: 10 * time.Second,
			},

			// Sampling
			SamplingEnabled: cfg.SamplingEnabled,
			SamplingConfig: tracing.SamplingConfig{
				Enabled:               cfg.SamplingEnabled,
				BaseRate:              cfg.SamplingBaseRate,
				AlwaysSampleErrors:    cfg.SamplingAlwaysSampleErrors,
				AlwaysSampleSlow:      cfg.SamplingAlwaysSampleSlow,
				SlowThresholdMs:       float64(cfg.SamplingSlowThresholdMs),
				DeterministicSampling: true,
			},

			// Enable metrics
			EnableMetrics: true,
		}

		tracer = tracing.New(tracingConfig)
		log.Println("Tracing initialized successfully")
	}

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Add tracing middleware
	if tracer != nil {
		e.Use(tracer.Middleware())
		log.Println("Tracing middleware enabled")
	}

	// Routes
	e.GET("/", handleHome)
	e.POST("/v1/api/workflow/create", handleCreateWorkflow(tracer))
	e.POST("/v1/api/workflow/slow", handleSlowWorkflow(tracer))
	e.POST("/v1/api/workflow/error", handleErrorWorkflow(tracer))
	e.GET("/health", handleHealth)

	// Metrics endpoint
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// Start server
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Port)
		log.Printf("Starting example service on %s", addr)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Metrics server
	go func() {
		metricsAddr := fmt.Sprintf(":%d", cfg.MetricsPort)
		log.Printf("Starting metrics server on %s", metricsAddr)
		http.ListenAndServe(metricsAddr, promhttp.Handler())
	}()

	// Wait forever
	select {}
}

// Config holds application configuration
type Config struct {
	ServiceID   string
	Port        int
	MetricsPort int
	DatabaseURL string

	// S3 configuration
	S3Endpoint     string
	S3AccessKey    string
	S3SecretKey    string
	S3Bucket       string
	S3Region       string
	S3UsePathStyle bool

	// Tracing configuration
	TracingEnabled bool

	// Async export
	AsyncExportEnabled bool
	AsyncQueueSize     int
	AsyncBatchSize     int
	AsyncWorkerCount   int

	// Sampling
	SamplingEnabled            bool
	SamplingBaseRate           float64
	SamplingAlwaysSampleErrors bool
	SamplingAlwaysSampleSlow   bool
	SamplingSlowThresholdMs    int64

	// OpenTelemetry
	OTelEnabled bool
}

func loadConfig() Config {
	return Config{
		ServiceID:   getEnv("SERVICE_ID", "example-service"),
		Port:        getEnvInt("PORT", 8080),
		MetricsPort: getEnvInt("METRICS_PORT", 9091),
		DatabaseURL: getEnv("DATABASE_URL", ""),

		S3Endpoint:     getEnv("S3_ENDPOINT", ""),
		S3AccessKey:    getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:    getEnv("S3_SECRET_KEY", ""),
		S3Bucket:       getEnv("S3_BUCKET", "eve-traces"),
		S3Region:       getEnv("S3_REGION", "us-east-1"),
		S3UsePathStyle: getEnvBool("S3_USE_PATH_STYLE", true),

		TracingEnabled: getEnvBool("TRACING_ENABLED", true),

		AsyncExportEnabled: getEnvBool("ASYNC_EXPORT_ENABLED", true),
		AsyncQueueSize:     getEnvInt("ASYNC_EXPORT_QUEUE_SIZE", 10000),
		AsyncBatchSize:     getEnvInt("ASYNC_EXPORT_BATCH_SIZE", 100),
		AsyncWorkerCount:   getEnvInt("ASYNC_EXPORT_WORKERS", 4),

		SamplingEnabled:            getEnvBool("SAMPLING_ENABLED", true),
		SamplingBaseRate:           getEnvFloat("SAMPLING_BASE_RATE", 0.1),
		SamplingAlwaysSampleErrors: getEnvBool("SAMPLING_ALWAYS_SAMPLE_ERRORS", true),
		SamplingAlwaysSampleSlow:   getEnvBool("SAMPLING_ALWAYS_SAMPLE_SLOW", true),
		SamplingSlowThresholdMs:    int64(getEnvInt("SAMPLING_SLOW_THRESHOLD_MS", 5000)),

		OTelEnabled: getEnvBool("OTEL_ENABLED", false),
	}
}

func createS3Client(cfg Config) *s3.Client {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if cfg.S3Endpoint != "" {
			return aws.Endpoint{
				PartitionID:   "aws",
				URL:           cfg.S3Endpoint,
				SigningRegion: cfg.S3Region,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.S3Region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.S3AccessKey,
			cfg.S3SecretKey,
			"",
		)),
	)
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.S3UsePathStyle
	})
}

func handleHome(c echo.Context) error {
	return c.JSON(200, map[string]interface{}{
		"service": "example-service",
		"version": "1.0.0",
		"endpoints": []string{
			"POST /v1/api/workflow/create - Create a sample workflow",
			"POST /v1/api/workflow/slow - Create a slow workflow (>5s)",
			"POST /v1/api/workflow/error - Create a failing workflow",
			"GET /health - Health check",
			"GET /metrics - Prometheus metrics",
		},
	})
}

func handleCreateWorkflow(tracer *tracing.Tracer) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Simulate workflow execution
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

		return c.JSON(200, map[string]interface{}{
			"status":         "completed",
			"correlation_id": c.Response().Header().Get("X-Correlation-ID"),
			"operation_id":   c.Response().Header().Get("X-Operation-ID"),
			"message":        "Workflow created successfully",
		})
	}
}

func handleSlowWorkflow(tracer *tracing.Tracer) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Simulate slow operation (triggers sampling)
		time.Sleep(6 * time.Second)

		return c.JSON(200, map[string]interface{}{
			"status":         "completed",
			"correlation_id": c.Response().Header().Get("X-Correlation-ID"),
			"operation_id":   c.Response().Header().Get("X-Operation-ID"),
			"message":        "Slow workflow completed",
			"duration_ms":    6000,
		})
	}
}

func handleErrorWorkflow(tracer *tracing.Tracer) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Simulate error (triggers sampling)
		return c.JSON(500, map[string]interface{}{
			"status":         "failed",
			"correlation_id": c.Response().Header().Get("X-Correlation-ID"),
			"operation_id":   c.Response().Header().Get("X-Operation-ID"),
			"error":          "Simulated error for demonstration",
		})
	}
}

func handleHealth(c echo.Context) error {
	return c.JSON(200, map[string]string{
		"status": "healthy",
	})
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return defaultValue
}
