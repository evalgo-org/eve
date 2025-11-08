package tracing

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	_ "github.com/lib/pq"
)

// InitConfig holds initialization parameters for tracing
type InitConfig struct {
	ServiceID string
	// Optional: Override defaults
	DisableIfMissing bool // If true, silently disable tracing if database/S3 unavailable
}

// Init initializes tracing from environment variables with sensible defaults
// Returns nil tracer if tracing is disabled or initialization fails (when DisableIfMissing=true)
func Init(cfg InitConfig) *Tracer {
	// Check if tracing is globally disabled
	if os.Getenv("TRACING_ENABLED") == "false" {
		if !cfg.DisableIfMissing {
			log.Println("⚠️  Tracing explicitly disabled via TRACING_ENABLED=false")
		}
		return nil
	}

	// Connect to action_traces database
	tracingDSN := os.Getenv("ACTION_TRACES_DSN")
	if tracingDSN == "" {
		tracingDSN = "postgresql://claude:claude_dev_password@localhost:5433/action_traces?sslmode=disable"
	}

	db, err := sql.Open("postgres", tracingDSN)
	if err != nil {
		if cfg.DisableIfMissing {
			log.Printf("⚠️  Tracing disabled: failed to connect to action_traces database: %v", err)
			return nil
		}
		log.Fatalf("Failed to connect to action_traces database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		if cfg.DisableIfMissing {
			log.Printf("⚠️  Tracing disabled: action_traces database unreachable: %v", err)
			return nil
		}
		log.Fatalf("action_traces database unreachable: %v", err)
	}

	// Initialize S3 client
	s3Client, err := initS3Client()
	if err != nil {
		if cfg.DisableIfMissing {
			log.Printf("⚠️  Tracing disabled: S3 initialization failed: %v", err)
			return nil
		}
		log.Fatalf("S3 initialization failed: %v", err)
	}

	// Create tracer from environment
	tracer := NewFromEnv(cfg.ServiceID, db, s3Client)

	log.Printf("✓ Tracing initialized for %s (payloads: %v)", cfg.ServiceID, tracer.config.StorePayloads)

	return tracer
}

// initS3Client creates S3 client from environment variables
func initS3Client() (*s3.Client, error) {
	ctx := context.Background()

	// Check for custom S3 endpoint (Hetzner, MinIO, etc.)
	s3Endpoint := os.Getenv("S3_ENDPOINT_URL")
	s3AccessKey := os.Getenv("S3_ACCESS_KEY")
	s3SecretKey := os.Getenv("S3_SECRET_KEY")

	// Load AWS config
	var err error

	if s3Endpoint != "" && s3AccessKey != "" && s3SecretKey != "" {
		// Custom S3-compatible endpoint
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				s3AccessKey,
				s3SecretKey,
				"",
			)),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load S3 config: %w", err)
		}

		// Create client with custom endpoint
		client := s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = &s3Endpoint
			o.UsePathStyle = true // Required for MinIO/Hetzner
		})

		return client, nil
	}

	// Default AWS S3
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return s3.NewFromConfig(cfg), nil
}
