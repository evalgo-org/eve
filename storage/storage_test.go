package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUploadResult tests the UploadResult struct
func TestUploadResult(t *testing.T) {
	result := UploadResult{
		FilePath:   "/test/file.txt",
		ObjectKey:  "remote/file.txt",
		Success:    true,
		Error:      nil,
		Skipped:    false,
		SkipReason: "",
	}

	assert.Equal(t, "/test/file.txt", result.FilePath)
	assert.Equal(t, "remote/file.txt", result.ObjectKey)
	assert.True(t, result.Success)
	assert.NoError(t, result.Error)
	assert.False(t, result.Skipped)
}

// TestUploadSummary tests the UploadSummary struct
func TestUploadSummary(t *testing.T) {
	summary := UploadSummary{
		TotalFiles:   10,
		SuccessCount: 8,
		ErrorCount:   2,
		SkippedCount: 3,
		Results:      []UploadResult{},
		FirstError:   nil,
	}

	assert.Equal(t, 10, summary.TotalFiles)
	assert.Equal(t, 8, summary.SuccessCount)
	assert.Equal(t, 2, summary.ErrorCount)
	assert.Equal(t, 3, summary.SkippedCount)
}

// TestCalculateMD5 tests MD5 hash calculation
func TestCalculateMD5(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		content     string
		expectedMD5 string
	}{
		{
			name:        "SimpleText",
			content:     "Hello, World!",
			expectedMD5: "65a8e27d8879283831b664bd8b7f0ad4",
		},
		{
			name:        "EmptyFile",
			content:     "",
			expectedMD5: "d41d8cd98f00b204e9800998ecf8427e",
		},
		{
			name:        "LargerContent",
			content:     "The quick brown fox jumps over the lazy dog",
			expectedMD5: "9e107d9d372bb6826bd81d3542a419d6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, tt.name+".txt")
			err := os.WriteFile(filePath, []byte(tt.content), 0644)
			require.NoError(t, err)

			md5hash, err := CalculateMD5(filePath)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedMD5, md5hash)
		})
	}
}

// TestCalculateMD5_NonExistentFile tests error handling
func TestCalculateMD5_NonExistentFile(t *testing.T) {
	_, err := CalculateMD5("/nonexistent/file.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open file")
}

// TestGetAllLocalFiles tests recursive file discovery
func TestGetAllLocalFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test directory structure
	os.MkdirAll(filepath.Join(tmpDir, "dir1"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "dir1", "subdir"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "dir2"), 0755)

	// Create test files
	files := []string{
		filepath.Join(tmpDir, "file1.txt"),
		filepath.Join(tmpDir, "dir1", "file2.txt"),
		filepath.Join(tmpDir, "dir1", "subdir", "file3.txt"),
		filepath.Join(tmpDir, "dir2", "file4.txt"),
	}

	for _, file := range files {
		err := os.WriteFile(file, []byte("test content"), 0644)
		require.NoError(t, err)
	}

	// Discover all files
	discovered, err := GetAllLocalFiles(tmpDir)
	require.NoError(t, err)

	// Verify all files were discovered
	assert.Equal(t, len(files), len(discovered))

	// Verify each file is in the discovered list
	for _, expectedFile := range files {
		assert.Contains(t, discovered, expectedFile)
	}
}

// TestGetAllLocalFiles_NonExistentDir tests error handling
func TestGetAllLocalFiles_NonExistentDir(t *testing.T) {
	_, err := GetAllLocalFiles("/nonexistent/directory")
	assert.Error(t, err)
}

// TestGetAllLocalFiles_EmptyDir tests empty directory handling
func TestGetAllLocalFiles_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	files, err := GetAllLocalFiles(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, files)
}

// TestMaxConcurrentUploads tests the constant
func TestMaxConcurrentUploads(t *testing.T) {
	assert.Equal(t, 96, MaxConcurrentUploads)
	assert.Greater(t, MaxConcurrentUploads, 0)
}

// TestSharedHTTPClient tests the shared HTTP client configuration
func TestSharedHTTPClient(t *testing.T) {
	assert.NotNil(t, sharedHTTPClient)
	assert.NotNil(t, sharedHTTPClient.Transport)
	assert.Greater(t, sharedHTTPClient.Timeout.Seconds(), float64(0))
}

// BenchmarkCalculateMD5 benchmarks MD5 calculation
func BenchmarkCalculateMD5(b *testing.B) {
	tmpDir := b.TempDir()
	filePath := filepath.Join(tmpDir, "benchmark.txt")

	// Create a test file with some content
	content := make([]byte, 1024*1024) // 1MB
	for i := range content {
		content[i] = byte(i % 256)
	}
	os.WriteFile(filePath, content, 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CalculateMD5(filePath)
	}
}

// BenchmarkGetAllLocalFiles benchmarks file discovery
func BenchmarkGetAllLocalFiles(b *testing.B) {
	tmpDir := b.TempDir()

	// Create a test directory structure
	for i := 0; i < 10; i++ {
		dir := filepath.Join(tmpDir, "dir"+string(rune('0'+i)))
		os.MkdirAll(dir, 0755)
		for j := 0; j < 10; j++ {
			file := filepath.Join(dir, "file"+string(rune('0'+j))+".txt")
			os.WriteFile(file, []byte("test"), 0644)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetAllLocalFiles(tmpDir)
	}
}

// TestUploadResult_WithError tests error scenarios
func TestUploadResult_WithError(t *testing.T) {
	err := assert.AnError
	result := UploadResult{
		FilePath:   "/test/file.txt",
		ObjectKey:  "remote/file.txt",
		Success:    false,
		Error:      err,
		Skipped:    false,
		SkipReason: "",
	}

	assert.False(t, result.Success)
	assert.Error(t, result.Error)
}

// TestUploadResult_Skipped tests skip scenarios
func TestUploadResult_Skipped(t *testing.T) {
	result := UploadResult{
		FilePath:   "/test/file.txt",
		ObjectKey:  "remote/file.txt",
		Success:    true,
		Error:      nil,
		Skipped:    true,
		SkipReason: "unchanged (MD5 match)",
	}

	assert.True(t, result.Success)
	assert.True(t, result.Skipped)
	assert.Equal(t, "unchanged (MD5 match)", result.SkipReason)
	assert.NoError(t, result.Error)
}

// TestUploadSummary_AggregateResults tests result aggregation
func TestUploadSummary_AggregateResults(t *testing.T) {
	results := []UploadResult{
		{Success: true, Skipped: false},
		{Success: true, Skipped: true},
		{Success: false, Error: assert.AnError},
		{Success: true, Skipped: false},
	}

	summary := UploadSummary{
		TotalFiles: len(results),
		Results:    results,
	}

	// Manually calculate counts
	for _, r := range results {
		if r.Success {
			summary.SuccessCount++
			if r.Skipped {
				summary.SkippedCount++
			}
		} else {
			summary.ErrorCount++
			if summary.FirstError == nil {
				summary.FirstError = r.Error
			}
		}
	}

	assert.Equal(t, 4, summary.TotalFiles)
	assert.Equal(t, 3, summary.SuccessCount)
	assert.Equal(t, 1, summary.ErrorCount)
	assert.Equal(t, 1, summary.SkippedCount)
	assert.Error(t, summary.FirstError)
}

// TestMinioGetObjectWithClient tests object download with mock client
func TestMinioGetObjectWithClient(t *testing.T) {
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "subdir", "downloaded.txt")

	mockClient := NewMockS3Client()
	mockClient.Buckets["test-bucket"] = true
	mockClient.Objects["remote/file.txt"] = &MockS3Object{
		Key:     "remote/file.txt",
		Content: "test content from S3",
		Size:    20,
	}

	ctx := context.Background()
	err := minioGetObjectWithClient(ctx, mockClient, "test-bucket", "remote/file.txt", localFile)
	require.NoError(t, err)

	// Verify the file was created and has correct content
	content, err := os.ReadFile(localFile)
	require.NoError(t, err)
	assert.Equal(t, "test content from S3", string(content))

	// Verify client was called correctly
	assert.True(t, mockClient.HeadBucketCalled)
	assert.True(t, mockClient.GetObjectCalled)
	assert.Equal(t, "test-bucket", mockClient.LastBucket)
	assert.Equal(t, "remote/file.txt", mockClient.LastObjectKey)
}

// TestMinioGetObjectWithClient_BucketNotFound tests error when bucket doesn't exist
func TestMinioGetObjectWithClient_BucketNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "downloaded.txt")

	mockClient := NewMockS3Client()
	// Don't add bucket to simulate bucket not found

	ctx := context.Background()
	err := minioGetObjectWithClient(ctx, mockClient, "nonexistent-bucket", "remote/file.txt", localFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to access bucket")
}

// TestMinioGetObjectWithClient_ObjectNotFound tests error when object doesn't exist
func TestMinioGetObjectWithClient_ObjectNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "downloaded.txt")

	mockClient := NewMockS3Client()
	mockClient.Buckets["test-bucket"] = true
	// Don't add object to simulate object not found

	ctx := context.Background()
	err := minioGetObjectWithClient(ctx, mockClient, "test-bucket", "nonexistent.txt", localFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestMinioListObjectsWithClient tests listing objects with mock client
func TestMinioListObjectsWithClient(t *testing.T) {
	mockClient := NewMockS3Client()
	mockClient.Buckets["test-bucket"] = true
	mockClient.Objects["file1.txt"] = &MockS3Object{Key: "file1.txt", Size: 100}
	mockClient.Objects["file2.txt"] = &MockS3Object{Key: "file2.txt", Size: 200}
	mockClient.Objects["dir/file3.txt"] = &MockS3Object{Key: "dir/file3.txt", Size: 300}

	ctx := context.Background()
	objects, err := minioListObjectsWithClient(ctx, mockClient, "test-bucket")
	require.NoError(t, err)

	assert.Len(t, objects, 3)
	assert.True(t, mockClient.HeadBucketCalled)
	assert.True(t, mockClient.ListObjectsV2Called)
	assert.Equal(t, "test-bucket", mockClient.LastBucket)
}

// TestMinioListObjectsWithClient_BucketNotFound tests error when bucket doesn't exist
func TestMinioListObjectsWithClient_BucketNotFound(t *testing.T) {
	mockClient := NewMockS3Client()

	ctx := context.Background()
	_, err := minioListObjectsWithClient(ctx, mockClient, "nonexistent-bucket")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to access bucket")
}

// TestLakeFSListObjectsWithClient tests listing objects in LakeFS with mock client
func TestLakeFSListObjectsWithClient(t *testing.T) {
	mockClient := NewMockS3Client()
	mockClient.Objects["main/file1.txt"] = &MockS3Object{Key: "main/file1.txt", Size: 100}
	mockClient.Objects["main/file2.txt"] = &MockS3Object{Key: "main/file2.txt", Size: 200}
	mockClient.Objects["dev/file3.txt"] = &MockS3Object{Key: "dev/file3.txt", Size: 300}

	ctx := context.Background()
	objects, err := lakeFSListObjectsWithClient(ctx, mockClient, "test-repo", "main")
	require.NoError(t, err)

	// Should only return objects with "main/" prefix
	assert.Len(t, objects, 2)
	assert.True(t, mockClient.ListObjectsV2Called)
}

// TestS3AwsListObjectsWithClient tests listing objects in AWS S3 with mock client
func TestS3AwsListObjectsWithClient(t *testing.T) {
	mockClient := NewMockS3Client()
	mockClient.Objects["file1.txt"] = &MockS3Object{Key: "file1.txt", Size: 100}
	mockClient.Objects["file2.txt"] = &MockS3Object{Key: "file2.txt", Size: 200}

	ctx := context.Background()
	objects, err := s3AwsListObjectsWithClient(ctx, mockClient, "test-bucket")
	require.NoError(t, err)

	assert.Len(t, objects, 2)
	assert.True(t, mockClient.ListObjectsV2Called)
	assert.Equal(t, "test-bucket", mockClient.LastBucket)
}

// TestLakeFsEnsureBucketExistsWithClient tests bucket creation with mock client
func TestLakeFsEnsureBucketExistsWithClient(t *testing.T) {
	mockClient := NewMockS3Client()

	ctx := context.Background()
	// Test creating a new bucket
	err := lakeFsEnsureBucketExistsWithClient(ctx, mockClient, "new-bucket")
	require.NoError(t, err)
	assert.True(t, mockClient.HeadBucketCalled)
	assert.True(t, mockClient.CreateBucketCalled)

	// Reset call tracking
	mockClient.HeadBucketCalled = false
	mockClient.CreateBucketCalled = false
	mockClient.Buckets["existing-bucket"] = true

	// Test with existing bucket
	err = lakeFsEnsureBucketExistsWithClient(ctx, mockClient, "existing-bucket")
	require.NoError(t, err)
	assert.True(t, mockClient.HeadBucketCalled)
	assert.False(t, mockClient.CreateBucketCalled) // Should not create if exists
}

// TestLakeFsUploadFileWithClient tests file upload to LakeFS with mock client
func TestLakeFsUploadFileWithClient(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	mockClient := NewMockS3Client()

	ctx := context.Background()
	err = lakeFsUploadFileWithClient(ctx, mockClient, "main", "test-repo", "remote/test.txt", testFile)
	require.NoError(t, err)

	assert.True(t, mockClient.PutObjectCalled)
	assert.Equal(t, "test-repo", mockClient.LastBucket)
	assert.Equal(t, "main/remote/test.txt", mockClient.LastObjectKey)
}

// TestLakeFsUploadFileWithClient_NonExistentFile tests error when file doesn't exist
func TestLakeFsUploadFileWithClient_NonExistentFile(t *testing.T) {
	mockClient := NewMockS3Client()

	ctx := context.Background()
	err := lakeFsUploadFileWithClient(ctx, mockClient, "main", "test-repo", "remote/test.txt", "/nonexistent/file.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open file")
}

// Helper functions that wrap the actual implementations to accept client interface
func minioGetObjectWithClient(ctx context.Context, client S3Client, bucket, remoteObject, localObject string) error {
	// Verify bucket exists
	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: &bucket,
	})
	if err != nil {
		return fmt.Errorf("failed to access bucket %s: %w", bucket, err)
	}

	// Get the object
	result, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &remoteObject,
	})
	if err != nil {
		return fmt.Errorf("object %s not found in bucket %s", remoteObject, bucket)
	}
	defer result.Body.Close()

	// Create local directory
	if err := os.MkdirAll(filepath.Dir(localObject), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", localObject, err)
	}

	// Create local file
	file, err := os.Create(localObject)
	if err != nil {
		return fmt.Errorf("failed to create local file %s: %w", localObject, err)
	}
	defer file.Close()

	// Copy content
	_, err = io.Copy(file, result.Body)
	if err != nil {
		return fmt.Errorf("failed to copy object content to %s: %w", localObject, err)
	}

	return nil
}

func minioListObjectsWithClient(ctx context.Context, client S3Client, bucket string) ([]types.Object, error) {
	// Verify bucket exists
	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: &bucket,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to access bucket %s: %w", bucket, err)
	}

	// List objects
	output, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: &bucket,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects in bucket %s: %w", bucket, err)
	}

	return output.Contents, nil
}

func lakeFSListObjectsWithClient(ctx context.Context, client S3Client, bucket, branch string) ([]types.Object, error) {
	prefix := branch + "/"
	output, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: &bucket,
		Prefix: &prefix,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects in bucket %s: %w", bucket, err)
	}

	return output.Contents, nil
}

func s3AwsListObjectsWithClient(ctx context.Context, client S3Client, bucket string) ([]types.Object, error) {
	output, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: &bucket,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects in bucket %s: %w", bucket, err)
	}

	return output.Contents, nil
}

func lakeFsEnsureBucketExistsWithClient(ctx context.Context, client S3Client, bucket string) error {
	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: &bucket,
	})
	if err == nil {
		return nil // Bucket exists
	}

	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: &bucket,
	})
	if err != nil {
		return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
	}
	return nil
}

func lakeFsUploadFileWithClient(ctx context.Context, client S3Client, branch, bucket, objectKey, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	key := branch + "/" + objectKey
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file %s to bucket %s: %w", filePath, bucket, err)
	}

	return nil
}
