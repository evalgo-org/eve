package storage

import (
	"os"
	"path/filepath"
	"testing"

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
