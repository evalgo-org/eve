package network

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWriteCounter tests WriteCounter functionality
func TestWriteCounter(t *testing.T) {
	wc := &WriteCounter{}

	tests := []struct {
		name          string
		data          []byte
		expectedTotal uint64
	}{
		{
			name:          "SmallWrite",
			data:          []byte("Hello"),
			expectedTotal: 5,
		},
		{
			name:          "EmptyWrite",
			data:          []byte(""),
			expectedTotal: 5, // Cumulative from previous test
		},
		{
			name:          "LargerWrite",
			data:          []byte("This is a longer message"),
			expectedTotal: 29, // 5 + 0 + 24
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := wc.Write(tt.data)
			assert.NoError(t, err)
			assert.Equal(t, len(tt.data), n)
			assert.Equal(t, tt.expectedTotal, wc.Total)
		})
	}
}

// TestWriteCounter_Fresh tests a fresh counter
func TestWriteCounter_Fresh(t *testing.T) {
	wc := &WriteCounter{}

	data := []byte("Test data")
	n, err := wc.Write(data)

	assert.NoError(t, err)
	assert.Equal(t, 9, n)
	assert.Equal(t, uint64(9), wc.Total)
}

// TestWriteCounter_Large tests larger data
func TestWriteCounter_Large(t *testing.T) {
	wc := &WriteCounter{}

	// Write 1MB of data
	data := make([]byte, 1024*1024)
	n, err := wc.Write(data)

	assert.NoError(t, err)
	assert.Equal(t, 1024*1024, n)
	assert.Equal(t, uint64(1024*1024), wc.Total)
}

// TestDownloadFile tests file download with mock server
func TestDownloadFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		content        string
		token          string
		statusCode     int
		expectError    bool
		validateHeader bool
	}{
		{
			name:        "SuccessfulDownload",
			content:     "Hello, World!",
			token:       "",
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:           "AuthenticatedDownload",
			content:        "Secret data",
			token:          "test-token-123",
			statusCode:     http.StatusOK,
			expectError:    false,
			validateHeader: true,
		},
		{
			name:        "NotFound",
			content:     "",
			token:       "",
			statusCode:  http.StatusNotFound,
			expectError: true,
		},
		{
			name:        "Unauthorized",
			content:     "",
			token:       "",
			statusCode:  http.StatusUnauthorized,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Validate auth header if needed
				if tt.validateHeader {
					authHeader := r.Header.Get("Authorization")
					assert.Equal(t, "token "+tt.token, authHeader)
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					w.Write([]byte(tt.content))
				}
			}))
			defer server.Close()

			// Download file
			filePath := filepath.Join(tmpDir, tt.name+".txt")
			err := DownloadFile(tt.token, server.URL, filePath)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify file exists
			assert.FileExists(t, filePath)

			// Verify content
			content, err := os.ReadFile(filePath)
			require.NoError(t, err)
			assert.Equal(t, tt.content, string(content))

			// Verify tmp file is cleaned up
			assert.NoFileExists(t, filePath+".tmp")
		})
	}
}

// TestDownloadFile_LargeFile tests downloading larger content
func TestDownloadFile_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 1MB of test data
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(largeData)
	}))
	defer server.Close()

	filePath := filepath.Join(tmpDir, "large.bin")
	err := DownloadFile("", server.URL, filePath)
	require.NoError(t, err)

	// Verify file size
	info, err := os.Stat(filePath)
	require.NoError(t, err)
	assert.Equal(t, int64(len(largeData)), info.Size())
}

// TestDownloadFile_InvalidURL tests error handling
func TestDownloadFile_InvalidURL(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	err := DownloadFile("", "http://invalid-url-that-does-not-exist.invalid", filePath)
	assert.Error(t, err)
}

// TestHttpClientDownloadFile_MockServer tests HttpClientDownloadFile with mock server
func TestHttpClientDownloadFile_MockServer(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		content     string
		statusCode  int
		expectError bool
	}{
		{
			name:        "SuccessfulDownload",
			content:     "Test content",
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "BadStatus",
			content:     "",
			statusCode:  http.StatusNotFound,
			expectError: true,
		},
		{
			name:        "ServerError",
			content:     "",
			statusCode:  http.StatusInternalServerError,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify User-Agent header
				userAgent := r.Header.Get("User-Agent")
				assert.Contains(t, userAgent, "Mozilla")

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					w.Write([]byte(tt.content))
				}
			}))
			defer server.Close()

			filePath := filepath.Join(tmpDir, tt.name+".txt")

			err := HttpClientDownloadFile(server.URL, filePath)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify file exists and content
			assert.FileExists(t, filePath)
			content, err := os.ReadFile(filePath)
			require.NoError(t, err)
			assert.Equal(t, tt.content, string(content))
		})
	}
}

// TestWriteCounter_MultipleWrites tests sequential writes
func TestWriteCounter_MultipleWrites(t *testing.T) {
	wc := &WriteCounter{}

	writes := [][]byte{
		[]byte("First "),
		[]byte("second "),
		[]byte("third"),
	}

	expectedTotal := uint64(0)
	for _, data := range writes {
		n, err := wc.Write(data)
		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
		expectedTotal += uint64(len(data))
		assert.Equal(t, expectedTotal, wc.Total)
	}

	assert.Equal(t, uint64(18), wc.Total) // "First second third" = 18 chars
}

// BenchmarkWriteCounter benchmarks the WriteCounter
func BenchmarkWriteCounter(b *testing.B) {
	wc := &WriteCounter{}
	data := []byte("Benchmark data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wc.Write(data)
	}
}

// BenchmarkWriteCounter_Large benchmarks large writes
func BenchmarkWriteCounter_Large(b *testing.B) {
	wc := &WriteCounter{}
	data := make([]byte, 1024*1024) // 1MB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wc.Write(data)
	}
}

// TestDownloadFile_EmptyToken tests download without authentication
func TestDownloadFile_EmptyToken(t *testing.T) {
	tmpDir := t.TempDir()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify no auth header is set
		authHeader := r.Header.Get("Authorization")
		assert.Empty(t, authHeader)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Public data"))
	}))
	defer server.Close()

	filePath := filepath.Join(tmpDir, "public.txt")
	err := DownloadFile("", server.URL, filePath)
	require.NoError(t, err)

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, "Public data", string(content))
}
