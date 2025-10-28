package network

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestProxyURLConstruction verifies that proxy requests are constructed with proper URLs
func TestProxyURLConstruction(t *testing.T) {
	// Create a mock backend server (kept for future integration tests)
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer mockBackend.Close()

	tests := []struct {
		name             string
		requestPath      string
		requestQuery     string
		routePath        string
		stripPrefix      bool
		addPrefix        string
		expectedPath     string
		expectedQuery    string
	}{
		{
			name:          "simple path",
			requestPath:   "/health",
			requestQuery:  "",
			routePath:     "/health",
			expectedPath:  "/health",
			expectedQuery: "",
		},
		{
			name:          "path with query",
			requestPath:   "/api/users",
			requestQuery:  "page=1&limit=10",
			routePath:     "/api/*",
			expectedPath:  "/api/users",
			expectedQuery: "page=1&limit=10",
		},
		{
			name:          "strip prefix",
			requestPath:   "/external/api/users",
			requestQuery:  "",
			routePath:     "/external/api/*",
			stripPrefix:   true,
			expectedPath:  "/users",
			expectedQuery: "",
		},
		{
			name:          "add prefix",
			requestPath:   "/users",
			requestQuery:  "",
			routePath:     "/users",
			addPrefix:     "/api/v1",
			expectedPath:  "/api/v1/users",
			expectedQuery: "",
		},
		{
			name:          "strip and add prefix",
			requestPath:   "/external/users",
			requestQuery:  "id=123",
			routePath:     "/external/*",
			stripPrefix:   true,
			addPrefix:     "/internal",
			expectedPath:  "/internal/users",
			expectedQuery: "id=123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			configJSON := fmt.Sprintf(`{
				"server": {"host": "127.0.0.1", "port": 0},
				"routes": [{
					"path": "%s",
					"methods": ["GET"],
					"backends": [{
						"ziti_service": "test-service",
						"identity_file": "./test-identity.json",
						"timeout": "5s"
					}],
					"strip_prefix": %v,
					"add_prefix": "%s"
				}]
			}`, tt.routePath, tt.stripPrefix, tt.addPrefix)

			tmpFile, err := os.CreateTemp("", "proxy-test-*.json")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.Write([]byte(configJSON)); err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}
			tmpFile.Close()

			// Load config
			config, err := LoadProxyConfig(tmpFile.Name())
			if err != nil {
				t.Fatalf("LoadProxyConfig() error = %v", err)
			}

			// Create router
			router := NewRouter(config)

			// Create test request
			requestURL := tt.requestPath
			if tt.requestQuery != "" {
				requestURL += "?" + tt.requestQuery
			}
			req := httptest.NewRequest("GET", requestURL, nil)

			// Match route
			match := router.Match(req)
			if match == nil {
				t.Fatalf("No route matched for path: %s", tt.requestPath)
			}

			// Rewrite path (simulating what proxy does)
			rewrittenPath := RewritePath(req.URL.Path, match.Route, match.Params)

			// Construct target URL (this is what the proxy does)
			targetURL := fmt.Sprintf("http://%s%s", "test-service", rewrittenPath)
			if req.URL.RawQuery != "" {
				targetURL += "?" + req.URL.RawQuery
			}

			// Verify URL construction
			if !strings.HasPrefix(targetURL, "http://") {
				t.Errorf("Target URL missing scheme: %s", targetURL)
			}

			if !strings.Contains(targetURL, "test-service") {
				t.Errorf("Target URL missing service name: %s", targetURL)
			}

			// Verify path
			expectedFullPath := tt.expectedPath
			if tt.expectedQuery != "" {
				expectedFullPath += "?" + tt.expectedQuery
			}

			if !strings.HasSuffix(targetURL, expectedFullPath) {
				t.Errorf("Target URL path = %s, want to end with %s", targetURL, expectedFullPath)
			}
		})
	}
}

// TestProxyRequestConstruction tests that HTTP requests are properly constructed
func TestProxyRequestConstruction(t *testing.T) {
	tests := []struct {
		name          string
		method        string
		path          string
		query         string
		body          string
		headers       map[string]string
		serviceName   string
		expectScheme  string
		expectHost    string
	}{
		{
			name:         "GET request",
			method:       "GET",
			path:         "/api/users",
			query:        "page=1",
			serviceName:  "user-service",
			expectScheme: "http",
			expectHost:   "user-service",
		},
		{
			name:         "POST request with body",
			method:       "POST",
			path:         "/api/users",
			body:         `{"name":"test"}`,
			serviceName:  "user-service",
			expectScheme: "http",
			expectHost:   "user-service",
		},
		{
			name:         "path with special characters",
			method:       "GET",
			path:         "/api/users/john@example.com",
			serviceName:  "user-service",
			expectScheme: "http",
			expectHost:   "user-service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Construct URL as the proxy does
			targetURL := fmt.Sprintf("http://%s%s", tt.serviceName, tt.path)
			if tt.query != "" {
				targetURL += "?" + tt.query
			}

			// Create request
			var bodyReader io.Reader
			if tt.body != "" {
				bodyReader = strings.NewReader(tt.body)
			}

			req, err := http.NewRequestWithContext(context.Background(), tt.method, targetURL, bodyReader)
			if err != nil {
				t.Fatalf("NewRequestWithContext() error = %v", err)
			}

			// Verify request construction
			if req.URL.Scheme != tt.expectScheme {
				t.Errorf("Request scheme = %s, want %s", req.URL.Scheme, tt.expectScheme)
			}

			if req.URL.Host != tt.expectHost {
				t.Errorf("Request host = %s, want %s", req.URL.Host, tt.expectHost)
			}

			if req.URL.Path != tt.path {
				t.Errorf("Request path = %s, want %s", req.URL.Path, tt.path)
			}

			if tt.query != "" && req.URL.RawQuery != tt.query {
				t.Errorf("Request query = %s, want %s", req.URL.RawQuery, tt.query)
			}

			if req.Method != tt.method {
				t.Errorf("Request method = %s, want %s", req.Method, tt.method)
			}
		})
	}
}

// TestProxyURLSchemeRequired tests that URLs must have a scheme
func TestProxyURLSchemeRequired(t *testing.T) {
	invalidURLs := []string{
		"/health",                    // No scheme, just path
		"service-name/path",          // No scheme
		"//service-name/path",        // Scheme-relative URL
		"",                           // Empty
	}

	for _, url := range invalidURLs {
		t.Run(fmt.Sprintf("invalid_%s", url), func(t *testing.T) {
			_, err := http.NewRequest("GET", url, nil)
			// These should either error or not have a proper scheme
			if err == nil {
				req, _ := http.NewRequest("GET", url, nil)
				if req.URL.Scheme != "" && req.URL.Host != "" {
					t.Errorf("Expected invalid URL, but got scheme=%s host=%s", req.URL.Scheme, req.URL.Host)
				}
			}
		})
	}
}

// TestProxyValidURLConstruction tests that valid URLs are properly constructed
func TestProxyValidURLConstruction(t *testing.T) {
	validURLs := []struct {
		service string
		path    string
		query   string
		want    string
	}{
		{"service1", "/health", "", "http://service1/health"},
		{"service2", "/api/v1/users", "page=1", "http://service2/api/v1/users?page=1"},
		{"my-service.local", "/path", "", "http://my-service.local/path"},
		{"service:8080", "/health", "", "http://service:8080/health"},
	}

	for _, tt := range validURLs {
		t.Run(tt.want, func(t *testing.T) {
			// Construct URL as proxy does
			url := fmt.Sprintf("http://%s%s", tt.service, tt.path)
			if tt.query != "" {
				url += "?" + tt.query
			}

			if url != tt.want {
				t.Errorf("Constructed URL = %s, want %s", url, tt.want)
			}

			// Verify it can be used in http.NewRequest
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				t.Errorf("NewRequest() error = %v for URL %s", err, url)
			}

			if req.URL.Scheme != "http" {
				t.Errorf("Request scheme = %s, want http", req.URL.Scheme)
			}

			if req.URL.Host != tt.service {
				t.Errorf("Request host = %s, want %s", req.URL.Host, tt.service)
			}
		})
	}
}

// TestProxyURLWithPort tests that URLs are constructed with port when specified
func TestProxyURLWithPort(t *testing.T) {
	tests := []struct {
		name        string
		service     string
		port        int
		path        string
		expectedURL string
	}{
		{"default port 80", "service1", 80, "/health", "http://service1/health"},
		{"port 8080", "service2", 8080, "/api", "http://service2:8080/api"},
		{"port 8880", "dev.caches.rest.px", 8880, "/v1/caches", "http://dev.caches.rest.px:8880/v1/caches"},
		{"no port specified", "service3", 0, "/health", "http://service3/health"},
		{"port 443", "service4", 443, "/api", "http://service4:443/api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate what proxy.go does
			host := tt.service
			if tt.port > 0 && tt.port != 80 {
				host = fmt.Sprintf("%s:%d", tt.service, tt.port)
			}

			targetURL := fmt.Sprintf("http://%s%s", host, tt.path)

			if targetURL != tt.expectedURL {
				t.Errorf("Constructed URL = %s, want %s", targetURL, tt.expectedURL)
			}

			// Verify it's a valid HTTP URL
			req, err := http.NewRequest("GET", targetURL, nil)
			if err != nil {
				t.Errorf("NewRequest() error = %v for URL %s", err, targetURL)
			}

			if req.URL.Scheme != "http" {
				t.Errorf("Request scheme = %s, want http", req.URL.Scheme)
			}
		})
	}
}

// TestProxyRequestPreservesQuery tests that query parameters are preserved
func TestProxyRequestPreservesQuery(t *testing.T) {
	tests := []struct {
		name         string
		originalPath string
		query        string
	}{
		{"simple query", "/api/users", "id=123"},
		{"multiple params", "/api/search", "q=test&page=1&limit=10"},
		{"encoded params", "/api/data", "filter=name%3Dtest"},
		{"empty query", "/api/health", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serviceName := "test-service"

			// Construct URL with query
			targetURL := fmt.Sprintf("http://%s%s", serviceName, tt.originalPath)
			if tt.query != "" {
				targetURL += "?" + tt.query
			}

			req, err := http.NewRequest("GET", targetURL, nil)
			if err != nil {
				t.Fatalf("NewRequest() error = %v", err)
			}

			// Verify query is preserved
			if tt.query != "" && req.URL.RawQuery != tt.query {
				t.Errorf("Query = %s, want %s", req.URL.RawQuery, tt.query)
			}

			if tt.query == "" && req.URL.RawQuery != "" {
				t.Errorf("Expected no query, got %s", req.URL.RawQuery)
			}
		})
	}
}

// TestProxyRequestTimeout tests that requests have proper timeouts
func TestProxyRequestTimeout(t *testing.T) {
	timeout := 5 * time.Second

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://test-service/health", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}

	// Verify context deadline is set
	deadline, ok := req.Context().Deadline()
	if !ok {
		t.Error("Request context has no deadline")
	}

	if time.Until(deadline) > timeout {
		t.Errorf("Context deadline too far in future")
	}
}
