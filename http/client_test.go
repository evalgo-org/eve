package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewRequest(t *testing.T) {
	req := NewRequest("GET", "https://example.com")

	if req.Method != "GET" {
		t.Errorf("Expected method GET, got %s", req.Method)
	}
	if req.URL != "https://example.com" {
		t.Errorf("Expected URL https://example.com, got %s", req.URL)
	}
	if req.Timeout != 30 {
		t.Errorf("Expected default timeout 30, got %d", req.Timeout)
	}
	if req.UserAgent != "eve-http/1.0" {
		t.Errorf("Expected default User-Agent eve-http/1.0, got %s", req.UserAgent)
	}
}

func TestExecuteGET(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello, World!"))
	}))
	defer server.Close()

	// Execute request
	req := NewRequest("GET", server.URL)
	resp, err := Execute(req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.BodyString != "Hello, World!" {
		t.Errorf("Expected body 'Hello, World!', got %s", resp.BodyString)
	}
}

func TestExecutePOSTJSON(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		var data map[string]string
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			t.Fatalf("Failed to decode JSON: %v", err)
		}

		if data["key"] != "value" {
			t.Errorf("Expected key=value, got %s", data["key"])
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	// Execute request
	req := NewRequest("POST", server.URL)
	req.JSONBody = `{"key": "value"}`
	resp, err := Execute(req)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.BodyString != `{"status": "success"}` {
		t.Errorf("Expected body '{\"status\": \"success\"}', got %s", resp.BodyString)
	}
}

func TestResponseIsSuccess(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   bool
	}{
		{200, true},
		{201, true},
		{299, true},
		{300, false},
		{400, false},
		{500, false},
	}

	for _, tt := range tests {
		resp := &Response{StatusCode: tt.statusCode}
		if resp.IsSuccess() != tt.expected {
			t.Errorf("StatusCode %d: expected IsSuccess()=%v, got %v",
				tt.statusCode, tt.expected, resp.IsSuccess())
		}
	}
}

func TestResponseIsClientError(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   bool
	}{
		{400, true},
		{404, true},
		{499, true},
		{200, false},
		{500, false},
	}

	for _, tt := range tests {
		resp := &Response{StatusCode: tt.statusCode}
		if resp.IsClientError() != tt.expected {
			t.Errorf("StatusCode %d: expected IsClientError()=%v, got %v",
				tt.statusCode, tt.expected, resp.IsClientError())
		}
	}
}

func TestResponseIsServerError(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   bool
	}{
		{500, true},
		{502, true},
		{599, true},
		{200, false},
		{400, false},
	}

	for _, tt := range tests {
		resp := &Response{StatusCode: tt.statusCode}
		if resp.IsServerError() != tt.expected {
			t.Errorf("StatusCode %d: expected IsServerError()=%v, got %v",
				tt.statusCode, tt.expected, resp.IsServerError())
		}
	}
}

func TestToSemanticRequest(t *testing.T) {
	req := NewRequest("GET", "https://api.example.com/data")
	req.Headers = map[string]string{"Authorization": "Bearer token"}
	req.Timeout = 60

	semantic := req.ToSemanticRequest()

	if semantic.Context != "https://schema.org" {
		t.Errorf("Expected context https://schema.org, got %s", semantic.Context)
	}
	if semantic.Type != "DigitalDocument" {
		t.Errorf("Expected type DigitalDocument, got %s", semantic.Type)
	}
	if semantic.URL != "https://api.example.com/data" {
		t.Errorf("Expected URL https://api.example.com/data, got %s", semantic.URL)
	}
	if semantic.Method != "GET" {
		t.Errorf("Expected method GET, got %s", semantic.Method)
	}
	if semantic.Headers["Authorization"] != "Bearer token" {
		t.Errorf("Expected Authorization header, got %v", semantic.Headers)
	}
}

func TestToJSONLD(t *testing.T) {
	req := NewRequest("POST", "https://api.example.com/users")
	req.JSONBody = `{"name": "Alice"}`

	jsonld, err := req.ToJSONLD()
	if err != nil {
		t.Fatalf("ToJSONLD failed: %v", err)
	}

	var semantic map[string]interface{}
	if err := json.Unmarshal([]byte(jsonld), &semantic); err != nil {
		t.Fatalf("Failed to parse JSON-LD: %v", err)
	}

	if semantic["@context"] != "https://schema.org" {
		t.Errorf("Expected @context https://schema.org, got %v", semantic["@context"])
	}
	if semantic["@type"] != "DigitalDocument" {
		t.Errorf("Expected @type DigitalDocument, got %v", semantic["@type"])
	}
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		attempt  int
		strategy string
		expected string // Duration string for comparison
	}{
		{0, "exponential", "1s"},
		{1, "exponential", "2s"},
		{2, "exponential", "4s"},
		{3, "exponential", "8s"},
		{0, "linear", "1s"},
		{1, "linear", "2s"},
		{2, "linear", "3s"},
		{3, "linear", "4s"},
	}

	for _, tt := range tests {
		backoff := calculateBackoff(tt.attempt, tt.strategy, 1*time.Second)
		if backoff.String() != tt.expected {
			t.Errorf("Attempt %d (%s): expected %s, got %s",
				tt.attempt, tt.strategy, tt.expected, backoff)
		}
	}
}
