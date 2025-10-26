package network

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHttpClientDownloadFile_Success(t *testing.T) {
	// Create a test HTTP server
	testContent := "test file content for download"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify User-Agent header is set
		userAgent := r.Header.Get("User-Agent")
		if userAgent == "" {
			t.Error("Expected User-Agent header to be set")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	// Create temporary file for download
	tmpfile, err := os.CreateTemp("", "download-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	// Test download
	err = HttpClientDownloadFile(server.URL, tmpfile.Name())
	if err != nil {
		t.Fatalf("HttpClientDownloadFile failed: %v", err)
	}

	// Verify file contents
	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, string(content))
	}
}

func TestHttpClientDownloadFile_WithRedirect(t *testing.T) {
	testContent := "redirected content"

	// Create redirect target server
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testContent))
	}))
	defer targetServer.Close()

	// Create redirect server
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, targetServer.URL, http.StatusMovedPermanently)
	}))
	defer redirectServer.Close()

	// Create temporary file for download
	tmpfile, err := os.CreateTemp("", "redirect-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	// Test download with redirect
	err = HttpClientDownloadFile(redirectServer.URL, tmpfile.Name())
	if err != nil {
		t.Fatalf("HttpClientDownloadFile with redirect failed: %v", err)
	}

	// Verify file contents
	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, string(content))
	}
}

func TestHttpClientDownloadFile_BadStatus(t *testing.T) {
	// Create a test HTTP server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	// Create temporary file for download
	tmpfile, err := os.CreateTemp("", "badstatus-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	// Test download with bad status
	err = HttpClientDownloadFile(server.URL, tmpfile.Name())
	if err == nil {
		t.Error("Expected error for bad HTTP status")
	}

	expectedError := "bad status:"
	if err != nil && len(err.Error()) < len(expectedError) {
		t.Errorf("Expected 'bad status' error, got: %v", err)
	}
}

func TestHttpClientDownloadFile_InvalidURL(t *testing.T) {
	// Create temporary file for download
	tmpfile, err := os.CreateTemp("", "invalidurl-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	// Test download with invalid URL
	err = HttpClientDownloadFile("://invalid-url", tmpfile.Name())
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestHttpClientDownloadFile_InvalidFilePath(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	}))
	defer server.Close()

	// Test download with invalid file path (directory that doesn't exist)
	err := HttpClientDownloadFile(server.URL, "/nonexistent/directory/file.txt")
	if err == nil {
		t.Error("Expected error for invalid file path")
	}

	expectedError := "failed to create file:"
	if err != nil && len(err.Error()) < len(expectedError) {
		t.Errorf("Expected 'failed to create file' error, got: %v", err)
	}
}

func TestHttpClientDownloadFile_ServerError(t *testing.T) {
	// Create a test HTTP server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	// Create temporary file for download
	tmpfile, err := os.CreateTemp("", "servererror-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	// Test download with server error
	err = HttpClientDownloadFile(server.URL, tmpfile.Name())
	if err == nil {
		t.Error("Expected error for server error status")
	}

	if err != nil {
		errMsg := err.Error()
		if len(errMsg) < len("bad status:") || errMsg[:len("bad status:")] != "bad status:" {
			t.Errorf("Expected 'bad status' error, got: %v", err)
		}
	}
}

func TestHttpClientDownloadFile_LargeFile(t *testing.T) {
	// Create a test HTTP server with larger content
	testContent := make([]byte, 10*1024) // 10KB
	for i := range testContent {
		testContent[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
	}))
	defer server.Close()

	// Create temporary file for download
	tmpfile, err := os.CreateTemp("", "largefile-test-*.bin")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	// Test download
	err = HttpClientDownloadFile(server.URL, tmpfile.Name())
	if err != nil {
		t.Fatalf("HttpClientDownloadFile failed: %v", err)
	}

	// Verify file size
	info, err := os.Stat(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to stat downloaded file: %v", err)
	}

	if info.Size() != int64(len(testContent)) {
		t.Errorf("Expected file size %d, got %d", len(testContent), info.Size())
	}
}

func TestHttpClientDownloadFile_EmptyResponse(t *testing.T) {
	// Create a test HTTP server with empty response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// No content written
	}))
	defer server.Close()

	// Create temporary file for download
	tmpfile, err := os.CreateTemp("", "empty-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	// Test download
	err = HttpClientDownloadFile(server.URL, tmpfile.Name())
	if err != nil {
		t.Fatalf("HttpClientDownloadFile failed: %v", err)
	}

	// Verify file is empty
	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if len(content) != 0 {
		t.Errorf("Expected empty file, got %d bytes", len(content))
	}
}

func TestHttpClientDownloadFile_MultipleRedirects(t *testing.T) {
	testContent := "final content after multiple redirects"

	// Create final target server
	finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testContent))
	}))
	defer finalServer.Close()

	// Create intermediate redirect server
	redirect2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, finalServer.URL, http.StatusFound)
	}))
	defer redirect2.Close()

	// Create first redirect server
	redirect1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirect2.URL, http.StatusMovedPermanently)
	}))
	defer redirect1.Close()

	// Create temporary file for download
	tmpfile, err := os.CreateTemp("", "multiredirect-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	// Test download with multiple redirects
	err = HttpClientDownloadFile(redirect1.URL, tmpfile.Name())
	if err != nil {
		t.Fatalf("HttpClientDownloadFile with multiple redirects failed: %v", err)
	}

	// Verify file contents
	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, string(content))
	}
}

func TestHttpClientDownloadFile_NetworkError(t *testing.T) {
	// Use a non-routable IP address to simulate network error
	// This should fail quickly
	tmpfile, err := os.CreateTemp("", "network-error-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	// Test download with network error
	err = HttpClientDownloadFile("http://192.0.2.1:8080/file", tmpfile.Name())
	if err == nil {
		t.Error("Expected error for network failure")
	}
}

func TestHttpClientDownloadFile_CustomHeaders(t *testing.T) {
	// Verify that custom headers are set correctly
	headerReceived := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent := r.Header.Get("User-Agent")
		if userAgent == "Mozilla/5.0 (X11; Linux x86_64; rv:140.0) Gecko/20100101 Firefox/140.0" {
			headerReceived = true
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	}))
	defer server.Close()

	tmpfile, err := os.CreateTemp("", "headers-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	err = HttpClientDownloadFile(server.URL, tmpfile.Name())
	if err != nil {
		t.Fatalf("HttpClientDownloadFile failed: %v", err)
	}

	if !headerReceived {
		t.Error("Expected User-Agent header to be set to Firefox string")
	}
}

func TestHttpClientDownloadFile_ContentTypes(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		content     []byte
	}{
		{"textplain", "text/plain", []byte("plain text content")},
		{"json", "application/json", []byte(`{"key": "value"}`)},
		{"octetstream", "application/octet-stream", []byte{0x00, 0x01, 0x02, 0x03}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(http.StatusOK)
				w.Write(tt.content)
			}))
			defer server.Close()

			tmpfile, err := os.CreateTemp("", fmt.Sprintf("contenttype-%s-*.dat", tt.name))
			if err != nil {
				t.Fatal(err)
			}
			tmpfile.Close()
			defer os.Remove(tmpfile.Name())

			err = HttpClientDownloadFile(server.URL, tmpfile.Name())
			if err != nil {
				t.Fatalf("HttpClientDownloadFile failed for %s: %v", tt.name, err)
			}

			// Verify content
			downloaded, err := os.ReadFile(tmpfile.Name())
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			if len(downloaded) != len(tt.content) {
				t.Errorf("Content length mismatch for %s: expected %d, got %d", tt.name, len(tt.content), len(downloaded))
			}
		})
	}
}
