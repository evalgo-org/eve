//go:build integration

package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testAccessKey = "minioadmin"
	testSecretKey = "minioadmin"
	testRegion    = "us-east-1"
	testBucket    = "test-bucket"
)

// setupMinIOContainer starts a MinIO container for S3-compatible testing
func setupMinIOContainer(t *testing.T) (string, func()) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     testAccessKey,
			"MINIO_ROOT_PASSWORD": testSecretKey,
		},
		Cmd: []string{"server", "/data"},
		WaitingFor: wait.ForHTTP("/minio/health/live").
			WithPort("9000/tcp").
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "Failed to start MinIO container")

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "9000")
	require.NoError(t, err)

	url := fmt.Sprintf("http://%s:%s", host, port.Port())

	// Create test bucket using lakeFsEnsureBucketExists helper
	err = createMinIOBucket(ctx, url, testBucket)
	require.NoError(t, err, "Failed to create test bucket")

	cleanup := func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}

	return url, cleanup
}

// createMinIOBucket creates a bucket in MinIO
func createMinIOBucket(ctx context.Context, url, bucket string) error {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(testRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(testAccessKey, testSecretKey, "")),
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
		return fmt.Errorf("failed to load config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	// Check if bucket exists
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err == nil {
		return nil // Bucket already exists
	}

	// Create bucket
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	return err
}

// TestMinioGetObject_Integration tests downloading a single object from MinIO
func TestMinioGetObject_Integration(t *testing.T) {
	url, cleanup := setupMinIOContainer(t)
	defer cleanup()

	ctx := context.Background()
	tmpDir := t.TempDir()

	// Upload a test file first
	testContent := []byte("Hello MinIO!")
	uploadPath := filepath.Join(tmpDir, "upload.txt")
	err := os.WriteFile(uploadPath, testContent, 0644)
	require.NoError(t, err)

	// Upload using HetznerUploadFile (works with MinIO)
	err = HetznerUploadFile(ctx, url, testAccessKey, testSecretKey, testBucket, uploadPath, "test/upload.txt")
	require.NoError(t, err)

	// Download the object
	downloadPath := filepath.Join(tmpDir, "download.txt")
	err = MinioGetObject(ctx, url, testAccessKey, testSecretKey, testBucket, "test/upload.txt", downloadPath)
	require.NoError(t, err)

	// Verify downloaded content
	downloadedContent, err := os.ReadFile(downloadPath)
	require.NoError(t, err)
	assert.Equal(t, testContent, downloadedContent)
}

// TestMinioGetObject_Integration_NonExistent tests error handling for missing objects
func TestMinioGetObject_Integration_NonExistent(t *testing.T) {
	url, cleanup := setupMinIOContainer(t)
	defer cleanup()

	ctx := context.Background()
	tmpDir := t.TempDir()

	downloadPath := filepath.Join(tmpDir, "nonexistent.txt")
	err := MinioGetObject(ctx, url, testAccessKey, testSecretKey, testBucket, "nonexistent/file.txt", downloadPath)
	assert.Error(t, err)
	// MinioGetObject returns custom error message for missing objects
	assert.Contains(t, err.Error(), "not found")
}

// TestMinioListObjects_Integration tests listing objects in a bucket
func TestMinioListObjects_Integration(t *testing.T) {
	url, cleanup := setupMinIOContainer(t)
	defer cleanup()

	ctx := context.Background()
	tmpDir := t.TempDir()

	// Upload several test files
	files := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, filename := range files {
		filePath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		require.NoError(t, err)

		err = HetznerUploadFile(ctx, url, testAccessKey, testSecretKey, testBucket, filePath, "test/"+filename)
		require.NoError(t, err)
	}

	// List objects
	objects, err := MinioListObjects(ctx, url, testAccessKey, testSecretKey, testBucket)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(objects), 3, "Should have at least 3 objects")

	// Verify object keys
	objectKeys := make([]string, len(objects))
	for i, obj := range objects {
		objectKeys[i] = *obj.Key
	}

	for _, filename := range files {
		assert.Contains(t, objectKeys, "test/"+filename)
	}
}

// TestMinioGetObjectRecursive_Integration tests recursive download
func TestMinioGetObjectRecursive_Integration(t *testing.T) {
	url, cleanup := setupMinIOContainer(t)
	defer cleanup()

	ctx := context.Background()
	uploadDir := t.TempDir()
	downloadDir := t.TempDir()

	// Create test directory structure
	os.MkdirAll(filepath.Join(uploadDir, "dir1"), 0755)
	os.MkdirAll(filepath.Join(uploadDir, "dir2"), 0755)

	testFiles := map[string]string{
		"file1.txt":      "content 1",
		"dir1/file2.txt": "content 2",
		"dir2/file3.txt": "content 3",
	}

	// Upload test files
	for relPath, content := range testFiles {
		filePath := filepath.Join(uploadDir, relPath)
		os.MkdirAll(filepath.Dir(filePath), 0755)
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)

		err = HetznerUploadFile(ctx, url, testAccessKey, testSecretKey, testBucket, filePath, "prefix/"+relPath)
		require.NoError(t, err)
	}

	// Download recursively
	err := MinioGetObjectRecursive(ctx, url, testAccessKey, testSecretKey, testBucket, "prefix/", downloadDir)
	require.NoError(t, err)

	// Verify downloaded files
	for relPath, expectedContent := range testFiles {
		downloadPath := filepath.Join(downloadDir, "prefix", relPath)
		content, err := os.ReadFile(downloadPath)
		require.NoError(t, err, "Failed to read %s", relPath)
		assert.Equal(t, expectedContent, string(content), "Content mismatch for %s", relPath)
	}
}

// TestHetznerUploadFile_Integration tests single file upload (works with MinIO)
func TestHetznerUploadFile_Integration(t *testing.T) {
	url, cleanup := setupMinIOContainer(t)
	defer cleanup()

	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create test file
	testContent := []byte("Hetzner test content")
	filePath := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(filePath, testContent, 0644)
	require.NoError(t, err)

	// Upload file
	err = HetznerUploadFile(ctx, url, testAccessKey, testSecretKey, testBucket, filePath, "hetzner/test.txt")
	require.NoError(t, err)

	// Verify file was uploaded by downloading it
	downloadPath := filepath.Join(tmpDir, "downloaded.txt")
	err = MinioGetObject(ctx, url, testAccessKey, testSecretKey, testBucket, "hetzner/test.txt", downloadPath)
	require.NoError(t, err)

	downloadedContent, err := os.ReadFile(downloadPath)
	require.NoError(t, err)
	assert.Equal(t, testContent, downloadedContent)
}

// TestHetznerUploadMultipleFiles_Integration_FullUpload tests bulk upload without sync
func TestHetznerUploadMultipleFiles_Integration_FullUpload(t *testing.T) {
	url, cleanup := setupMinIOContainer(t)
	defer cleanup()

	ctx := context.Background()
	rootPath := t.TempDir()

	// Create test directory structure
	os.MkdirAll(filepath.Join(rootPath, "subdir"), 0755)

	testFiles := []string{
		"file1.txt",
		"file2.txt",
		"subdir/file3.txt",
	}

	for _, filename := range testFiles {
		filePath := filepath.Join(rootPath, filename)
		os.MkdirAll(filepath.Dir(filePath), 0755)
		err := os.WriteFile(filePath, []byte("content of "+filename), 0644)
		require.NoError(t, err)
	}

	// Upload all files
	summary, err := HetznerUploadMultipleFiles(ctx, url, testAccessKey, testSecretKey, testRegion, testBucket, rootPath, "uploads", false)
	require.NoError(t, err)

	// Verify summary
	assert.Equal(t, 3, summary.TotalFiles)
	assert.Equal(t, 3, summary.SuccessCount)
	assert.Equal(t, 0, summary.ErrorCount)
	assert.Equal(t, 0, summary.SkippedCount)

	// Verify all files were uploaded
	objects, err := MinioListObjects(ctx, url, testAccessKey, testSecretKey, testBucket)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(objects), 3)
}

// TestHetznerUploadMultipleFiles_Integration_Sync tests intelligent synchronization
func TestHetznerUploadMultipleFiles_Integration_Sync(t *testing.T) {
	url, cleanup := setupMinIOContainer(t)
	defer cleanup()

	ctx := context.Background()
	rootPath := t.TempDir()

	// Create initial test files
	file1 := filepath.Join(rootPath, "file1.txt")
	file2 := filepath.Join(rootPath, "file2.txt")

	err := os.WriteFile(file1, []byte("initial content 1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte("initial content 2"), 0644)
	require.NoError(t, err)

	// First upload - everything should upload
	summary1, err := HetznerUploadMultipleFiles(ctx, url, testAccessKey, testSecretKey, testRegion, testBucket, rootPath, "sync", true)
	require.NoError(t, err)
	assert.Equal(t, 2, summary1.TotalFiles)
	assert.Equal(t, 2, summary1.SuccessCount)
	assert.Equal(t, 0, summary1.SkippedCount)

	// Second upload without changes - everything should skip
	summary2, err := HetznerUploadMultipleFiles(ctx, url, testAccessKey, testSecretKey, testRegion, testBucket, rootPath, "sync", true)
	require.NoError(t, err)
	assert.Equal(t, 2, summary2.TotalFiles)
	assert.Equal(t, 2, summary2.SuccessCount)
	assert.Equal(t, 2, summary2.SkippedCount, "Unchanged files should be skipped")

	// Modify one file
	err = os.WriteFile(file1, []byte("modified content 1"), 0644)
	require.NoError(t, err)

	// Third upload - one should upload, one should skip
	summary3, err := HetznerUploadMultipleFiles(ctx, url, testAccessKey, testSecretKey, testRegion, testBucket, rootPath, "sync", true)
	require.NoError(t, err)
	assert.Equal(t, 2, summary3.TotalFiles)
	assert.Equal(t, 2, summary3.SuccessCount)
	assert.Equal(t, 1, summary3.SkippedCount, "Only unchanged file should be skipped")

	// Verify skipped file
	skippedCount := 0
	uploadedCount := 0
	for _, result := range summary3.Results {
		if result.Skipped {
			skippedCount++
			assert.Contains(t, result.ObjectKey, "file2.txt", "file2.txt should be skipped")
		} else {
			uploadedCount++
		}
	}
	assert.Equal(t, 1, skippedCount)
	assert.Equal(t, 1, uploadedCount)
}

// Note: S3AwsListObjects is not tested here because it uses virtual-hosted-style
// bucket access (required for real AWS S3), which is incompatible with MinIO's
// path-style-only configuration. For S3-compatible testing, use MinioListObjects instead.

// TestLakeFSListObjects_Integration tests LakeFS object listing (using MinIO)
func TestLakeFSListObjects_Integration(t *testing.T) {
	url, cleanup := setupMinIOContainer(t)
	defer cleanup()

	ctx := context.Background()
	tmpDir := t.TempDir()

	branch := "main"

	// Upload test files with branch prefix
	files := []string{"lakefs1.txt", "lakefs2.txt"}
	for _, filename := range files {
		filePath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filePath, []byte("lakefs content"), 0644)
		require.NoError(t, err)

		// LakeFS uses branch prefix
		err = HetznerUploadFile(ctx, url, testAccessKey, testSecretKey, testBucket, filePath, branch+"/"+filename)
		require.NoError(t, err)
	}

	// List objects using LakeFS function
	objects, err := LakeFSListObjects(ctx, url, testAccessKey, testSecretKey, testBucket, branch)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(objects), 2, "Should have at least 2 objects")

	// Verify all objects have the branch prefix
	for _, obj := range objects {
		assert.Contains(t, *obj.Key, branch+"/", "Object should have branch prefix")
	}
}

// TestUploadResult_Integration tests result tracking
func TestUploadResult_Integration(t *testing.T) {
	result := UploadResult{
		FilePath:   "/tmp/test.txt",
		ObjectKey:  "uploads/test.txt",
		Success:    true,
		Error:      nil,
		Skipped:    false,
		SkipReason: "",
	}

	assert.True(t, result.Success)
	assert.NoError(t, result.Error)
	assert.False(t, result.Skipped)
}

// TestConcurrentUploads_Integration tests concurrent upload handling
func TestConcurrentUploads_Integration(t *testing.T) {
	url, cleanup := setupMinIOContainer(t)
	defer cleanup()

	ctx := context.Background()
	rootPath := t.TempDir()

	// Create many test files to trigger concurrent uploads
	numFiles := 20
	for i := 0; i < numFiles; i++ {
		filename := fmt.Sprintf("file%d.txt", i)
		filePath := filepath.Join(rootPath, filename)
		content := fmt.Sprintf("content %d", i)
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Upload all files concurrently
	summary, err := HetznerUploadMultipleFiles(ctx, url, testAccessKey, testSecretKey, testRegion, testBucket, rootPath, "concurrent", false)
	require.NoError(t, err)

	// Verify all files uploaded successfully
	assert.Equal(t, numFiles, summary.TotalFiles)
	assert.Equal(t, numFiles, summary.SuccessCount)
	assert.Equal(t, 0, summary.ErrorCount)

	// Verify all files exist in storage
	objects, err := MinioListObjects(ctx, url, testAccessKey, testSecretKey, testBucket)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(objects), numFiles)
}
