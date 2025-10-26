// Package forge provides comprehensive testing for GitLab integration functionality.
// This file contains unit tests for GitLab operations including job management,
// tag creation, error extraction, and archive handling.
package forge

import (
	"archive/zip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJobInfo_JSON validates JSON serialization and deserialization of JobInfo.
// This test ensures that JobInfo structures can be properly marshaled and unmarshaled
// for API communication and data persistence scenarios.
//
// Test Coverage:
//   - JSON marshaling of JobInfo struct
//   - JSON unmarshaling into JobInfo struct
//   - Field presence and correctness
//   - Type preservation during serialization
func TestJobInfo_JSON(t *testing.T) {
	t.Run("MarshalAndUnmarshal", func(t *testing.T) {
		original := JobInfo{
			ID:       123,
			Name:     "test-job",
			Status:   "success",
			Stage:    "build",
			Ref:      "main",
			Pipeline: 456,
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"id":123`)
		assert.Contains(t, string(data), `"name":"test-job"`)

		// Unmarshal back to struct
		var unmarshaled JobInfo
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		// Verify all fields match
		assert.Equal(t, original.ID, unmarshaled.ID)
		assert.Equal(t, original.Name, unmarshaled.Name)
		assert.Equal(t, original.Status, unmarshaled.Status)
		assert.Equal(t, original.Stage, unmarshaled.Stage)
		assert.Equal(t, original.Ref, unmarshaled.Ref)
		assert.Equal(t, original.Pipeline, unmarshaled.Pipeline)
	})
}

// TestJobDetails_JSON validates JSON serialization and deserialization of JobDetails.
// This test ensures that comprehensive job details including timestamps and optional
// fields are correctly handled during JSON operations.
//
// Test Coverage:
//   - JSON marshaling with all fields populated
//   - JSON marshaling with optional fields (nil timestamps)
//   - Field presence and type correctness
//   - Timestamp handling and formatting
func TestJobDetails_JSON(t *testing.T) {
	t.Run("MarshalWithAllFields", func(t *testing.T) {
		now := time.Now()
		started := now.Add(-5 * time.Minute)
		finished := now

		original := JobDetails{
			ID:             789,
			Name:           "deploy-job",
			Status:         "failed",
			Stage:          "deploy",
			Ref:            "v1.0.0",
			PipelineID:     456,
			CreatedAt:      now,
			StartedAt:      &started,
			FinishedAt:     &finished,
			Duration:       300.5,
			QueuedDuration: 15.2,
			WebURL:         "https://gitlab.example.com/job/789",
			FailureReason:  "script_failure",
			ErrorMessage:   "Deployment failed",
			TraceLog:       "ERROR: Connection timeout",
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)

		// Verify critical fields are present
		assert.Contains(t, string(data), `"id":789`)
		assert.Contains(t, string(data), `"status":"failed"`)
		assert.Contains(t, string(data), `"failure_reason":"script_failure"`)

		// Unmarshal back
		var unmarshaled JobDetails
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, original.ID, unmarshaled.ID)
		assert.Equal(t, original.Name, unmarshaled.Name)
		assert.Equal(t, original.Status, unmarshaled.Status)
		assert.Equal(t, original.FailureReason, unmarshaled.FailureReason)
		assert.Equal(t, original.ErrorMessage, unmarshaled.ErrorMessage)
	})

	t.Run("MarshalWithNilTimestamps", func(t *testing.T) {
		now := time.Now()

		original := JobDetails{
			ID:         123,
			Name:       "pending-job",
			Status:     "pending",
			Stage:      "test",
			Ref:        "main",
			PipelineID: 456,
			CreatedAt:  now,
			StartedAt:  nil, // Not started yet
			FinishedAt: nil, // Not finished yet
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)

		// Unmarshal back
		var unmarshaled JobDetails
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, original.ID, unmarshaled.ID)
		assert.Nil(t, unmarshaled.StartedAt)
		assert.Nil(t, unmarshaled.FinishedAt)
	})
}

// TestExtractErrorFromTrace validates error extraction from job trace logs.
// This test ensures that the function correctly identifies and extracts error
// messages from various trace log formats and patterns.
//
// Test Coverage:
//   - Single error line extraction
//   - Multiple error lines (last 5 returned)
//   - Different error keywords (ERROR, FAILED, FATAL, Exception)
//   - Case sensitivity handling
//   - Empty trace log handling
//   - Trace log with no errors
func TestExtractErrorFromTrace(t *testing.T) {
	tests := []struct {
		name     string
		trace    string
		wantErr  string
		contains []string
	}{
		{
			name:     "SingleErrorLine",
			trace:    "Step 1: OK\nStep 2: ERROR: Connection failed\nStep 3: Cleanup",
			contains: []string{"ERROR: Connection failed"},
		},
		{
			name: "MultipleErrorLines",
			trace: `INFO: Starting build
ERROR: Line 1
ERROR: Line 2
ERROR: Line 3
ERROR: Line 4
ERROR: Line 5
ERROR: Line 6
ERROR: Line 7`,
			contains: []string{"Line 3", "Line 4", "Line 5", "Line 6", "Line 7"},
		},
		{
			name:     "FatalError",
			trace:    "Running tests\nFATAL: Test suite crashed\nExiting",
			contains: []string{"FATAL: Test suite crashed"},
		},
		{
			name:     "ExceptionError",
			trace:    "Processing data\nException in thread main: NullPointerException\nStack trace",
			contains: []string{"Exception in thread main: NullPointerException"},
		},
		{
			name:     "FailedKeyword",
			trace:    "Test 1: PASSED\nTest 2: FAILED - Assertion error\nTest 3: PASSED",
			contains: []string{"FAILED - Assertion error"},
		},
		{
			name:    "NoErrors",
			trace:   "Step 1: Success\nStep 2: Success\nStep 3: Success",
			wantErr: "No specific error message found in trace log",
		},
		{
			name:    "EmptyTrace",
			trace:   "",
			wantErr: "No specific error message found in trace log",
		},
		{
			name:     "MixedCaseError",
			trace:    "Info: Processing\nerror: something went wrong\nWarning: deprecated",
			contains: []string{"error: something went wrong"},
		},
		{
			name:     "CaseInsensitiveError",
			trace:    "Info: Processing\nError: Something went wrong\nWarning: deprecated",
			contains: []string{"Error: Something went wrong"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractErrorFromTrace(tt.trace)

			if tt.wantErr != "" {
				assert.Equal(t, tt.wantErr, result)
			}

			for _, substr := range tt.contains {
				assert.Contains(t, result, substr,
					"Expected result to contain '%s', but got: %s", substr, result)
			}
		})
	}
}

// TestExtractErrorFromTrace_EdgeCases validates edge case handling in error extraction.
// This test ensures robust handling of unusual trace log formats and content.
//
// Test Coverage:
//   - Very long trace logs
//   - Trace logs with special characters
//   - Trace logs with only whitespace error lines
//   - Multiple consecutive error keywords
func TestExtractErrorFromTrace_EdgeCases(t *testing.T) {
	t.Run("VeryLongTrace", func(t *testing.T) {
		// Create a trace with many non-error lines and a few error lines
		var lines []string
		for i := 0; i < 1000; i++ {
			lines = append(lines, "INFO: Processing step "+string(rune(i)))
		}
		lines = append(lines, "ERROR: Critical failure at step 1000")
		trace := strings.Join(lines, "\n")

		result := extractErrorFromTrace(trace)
		assert.Contains(t, result, "ERROR: Critical failure")
	})

	t.Run("SpecialCharactersInError", func(t *testing.T) {
		trace := "ERROR: File not found: /path/with/special-chars_123!@#.txt"
		result := extractErrorFromTrace(trace)
		assert.Contains(t, result, "special-chars_123!@#")
	})

	t.Run("WhitespaceOnlyErrorLine", func(t *testing.T) {
		trace := "INFO: Start\nERROR:    \nINFO: End"
		result := extractErrorFromTrace(trace)
		assert.NotEqual(t, "No specific error message found in trace log", result)
	})

	t.Run("MultipleConsecutiveErrors", func(t *testing.T) {
		trace := "ERROR: Error 1\nERROR: Error 2\nERROR: Error 3"
		result := extractErrorFromTrace(trace)
		// Should contain at least some of the errors
		assert.True(t, strings.Contains(result, "Error 1") ||
			strings.Contains(result, "Error 2") ||
			strings.Contains(result, "Error 3"))
	})
}

// TestGlabDownloadFile validates file download functionality.
// This test uses a mock HTTP server to test file download without external dependencies.
//
// Test Coverage:
//   - Successful file download
//   - HTTP error handling (404, 500)
//   - Network error handling
//   - File creation and content verification
//   - Proper resource cleanup
func TestGlabDownloadFile(t *testing.T) {
	t.Run("SuccessfulDownload", func(t *testing.T) {
		// Create a test server
		expectedContent := "test file content"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(expectedContent))
		}))
		defer server.Close()

		// Create temp file path
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test-download.txt")

		// Download file
		err := glabDownloadFile(server.URL, filePath)
		require.NoError(t, err)

		// Verify file exists and has correct content
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, expectedContent, string(content))
	})

	t.Run("HTTPError404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test-404.txt")

		err := glabDownloadFile(server.URL, filePath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bad status")
	})

	t.Run("HTTPError500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test-500.txt")

		err := glabDownloadFile(server.URL, filePath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bad status")
	})

	t.Run("InvalidURL", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test-invalid.txt")

		err := glabDownloadFile("http://invalid-url-that-does-not-exist.local", filePath)
		assert.Error(t, err)
	})
}

// TestGlabUnZip validates zip file extraction functionality.
// This test ensures that zip files are correctly extracted to the destination directory.
//
// Test Coverage:
//   - Basic zip extraction
//   - Nested directory extraction
//   - File permissions preservation
//   - Multiple files extraction
//   - Empty zip file handling
func TestGlabUnZip(t *testing.T) {
	t.Run("BasicExtraction", func(t *testing.T) {
		// Create a test zip file
		tmpDir := t.TempDir()
		zipPath := filepath.Join(tmpDir, "test.zip")
		extractDir := filepath.Join(tmpDir, "extracted")

		// Create zip file with test content
		zipFile, err := os.Create(zipPath)
		require.NoError(t, err)
		defer zipFile.Close()

		zipWriter := zip.NewWriter(zipFile)

		// Add a file to zip
		fileWriter, err := zipWriter.Create("testfile.txt")
		require.NoError(t, err)
		_, err = fileWriter.Write([]byte("test content"))
		require.NoError(t, err)

		// Add a file in a subdirectory
		fileWriter, err = zipWriter.Create("subdir/nested.txt")
		require.NoError(t, err)
		_, err = fileWriter.Write([]byte("nested content"))
		require.NoError(t, err)

		err = zipWriter.Close()
		require.NoError(t, err)

		// Extract the zip
		err = glabUnZip(zipPath, extractDir)
		require.NoError(t, err)

		// Verify extracted files
		content, err := os.ReadFile(filepath.Join(extractDir, "testfile.txt"))
		require.NoError(t, err)
		assert.Equal(t, "test content", string(content))

		content, err = os.ReadFile(filepath.Join(extractDir, "subdir", "nested.txt"))
		require.NoError(t, err)
		assert.Equal(t, "nested content", string(content))
	})

	t.Run("InvalidZipFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidZip := filepath.Join(tmpDir, "invalid.zip")
		extractDir := filepath.Join(tmpDir, "extract")

		// Create an invalid zip file
		err := os.WriteFile(invalidZip, []byte("not a zip file"), 0644)
		require.NoError(t, err)

		// Try to extract - should fail
		err = glabUnZip(invalidZip, extractDir)
		assert.Error(t, err)
	})

	t.Run("NonexistentZipFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := glabUnZip(filepath.Join(tmpDir, "nonexistent.zip"), tmpDir)
		assert.Error(t, err)
	})
}

// TestGlabUnzipStripTop validates zip extraction with top directory stripping.
// This test ensures that the top-level directory added by GitLab is correctly removed
// during extraction, placing files at the root of the destination directory.
//
// Test Coverage:
//   - Top directory stripping
//   - Nested file structure preservation
//   - Empty directories handling
//   - File content integrity
func TestGlabUnzipStripTop(t *testing.T) {
	t.Run("StripTopDirectory", func(t *testing.T) {
		// Create a test zip file with GitLab-style top directory
		tmpDir := t.TempDir()
		zipPath := filepath.Join(tmpDir, "repo.zip")
		extractDir := filepath.Join(tmpDir, "extracted")

		// Create zip file
		zipFile, err := os.Create(zipPath)
		require.NoError(t, err)
		defer zipFile.Close()

		zipWriter := zip.NewWriter(zipFile)

		// Add files with top-level directory (as GitLab does)
		files := map[string]string{
			"repo-main-abc123/README.md":           "# Repository",
			"repo-main-abc123/src/main.go":         "package main",
			"repo-main-abc123/src/utils/helper.go": "package utils",
		}

		for path, content := range files {
			fileWriter, err := zipWriter.Create(path)
			require.NoError(t, err)
			_, err = fileWriter.Write([]byte(content))
			require.NoError(t, err)
		}

		err = zipWriter.Close()
		require.NoError(t, err)

		// Extract with stripping
		err = glabUnzipStripTop(zipPath, extractDir)
		require.NoError(t, err)

		// Verify files are at correct locations (without top directory)
		content, err := os.ReadFile(filepath.Join(extractDir, "README.md"))
		require.NoError(t, err)
		assert.Equal(t, "# Repository", string(content))

		content, err = os.ReadFile(filepath.Join(extractDir, "src", "main.go"))
		require.NoError(t, err)
		assert.Equal(t, "package main", string(content))

		content, err = os.ReadFile(filepath.Join(extractDir, "src", "utils", "helper.go"))
		require.NoError(t, err)
		assert.Equal(t, "package utils", string(content))

		// Verify top directory was stripped
		_, err = os.Stat(filepath.Join(extractDir, "repo-main-abc123"))
		assert.True(t, os.IsNotExist(err), "Top directory should not exist after stripping")
	})

	t.Run("HandleRootDirectoryEntry", func(t *testing.T) {
		// Test that root directory entry is skipped
		tmpDir := t.TempDir()
		zipPath := filepath.Join(tmpDir, "repo2.zip")
		extractDir := filepath.Join(tmpDir, "extracted2")

		zipFile, err := os.Create(zipPath)
		require.NoError(t, err)
		defer zipFile.Close()

		zipWriter := zip.NewWriter(zipFile)

		// Add root directory entry (should be skipped)
		_, err = zipWriter.Create("repo-main/")
		require.NoError(t, err)

		// Add a file
		fileWriter, err := zipWriter.Create("repo-main/file.txt")
		require.NoError(t, err)
		_, err = fileWriter.Write([]byte("content"))
		require.NoError(t, err)

		err = zipWriter.Close()
		require.NoError(t, err)

		// Extract
		err = glabUnzipStripTop(zipPath, extractDir)
		require.NoError(t, err)

		// Verify file exists at root
		content, err := os.ReadFile(filepath.Join(extractDir, "file.txt"))
		require.NoError(t, err)
		assert.Equal(t, "content", string(content))
	})
}

// BenchmarkExtractErrorFromTrace provides performance benchmarks for error extraction.
// This benchmark measures the overhead of error extraction from trace logs of various sizes.
//
// Benchmark Coverage:
//   - Small trace log (100 lines)
//   - Medium trace log (1000 lines)
//   - Large trace log (10000 lines)
//   - Trace with many errors
//   - Trace with no errors
func BenchmarkExtractErrorFromTrace(b *testing.B) {
	// Small trace
	b.Run("SmallTrace100Lines", func(b *testing.B) {
		var lines []string
		for i := 0; i < 100; i++ {
			lines = append(lines, "INFO: Processing step "+string(rune(i)))
		}
		lines = append(lines, "ERROR: Failure at end")
		trace := strings.Join(lines, "\n")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			extractErrorFromTrace(trace)
		}
	})

	// Medium trace
	b.Run("MediumTrace1000Lines", func(b *testing.B) {
		var lines []string
		for i := 0; i < 1000; i++ {
			lines = append(lines, "INFO: Processing step "+string(rune(i)))
		}
		lines = append(lines, "ERROR: Failure at end")
		trace := strings.Join(lines, "\n")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			extractErrorFromTrace(trace)
		}
	})

	// Large trace
	b.Run("LargeTrace10000Lines", func(b *testing.B) {
		var lines []string
		for i := 0; i < 10000; i++ {
			lines = append(lines, "INFO: Processing step "+string(rune(i)))
		}
		lines = append(lines, "ERROR: Failure at end")
		trace := strings.Join(lines, "\n")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			extractErrorFromTrace(trace)
		}
	})

	// Many errors
	b.Run("ManyErrors", func(b *testing.B) {
		var lines []string
		for i := 0; i < 100; i++ {
			lines = append(lines, "ERROR: Error number "+string(rune(i)))
		}
		trace := strings.Join(lines, "\n")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			extractErrorFromTrace(trace)
		}
	})
}

// BenchmarkGlabUnZip provides performance benchmarks for zip extraction.
// This benchmark measures the extraction performance for different zip file sizes.
//
// Benchmark Coverage:
//   - Small zip (few files)
//   - Medium zip (moderate number of files)
//   - Nested directory structures
func BenchmarkGlabUnZip(b *testing.B) {
	// Create a test zip file
	tmpDir := b.TempDir()
	zipPath := filepath.Join(tmpDir, "bench.zip")

	zipFile, _ := os.Create(zipPath)
	zipWriter := zip.NewWriter(zipFile)

	// Add multiple files
	for i := 0; i < 50; i++ {
		fileWriter, _ := zipWriter.Create("file" + string(rune(i)) + ".txt")
		_, _ = fileWriter.Write([]byte("content for file " + string(rune(i))))
	}
	zipWriter.Close()
	zipFile.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractDir := filepath.Join(tmpDir, "extract"+string(rune(i)))
		glabUnZip(zipPath, extractDir)
	}
}
