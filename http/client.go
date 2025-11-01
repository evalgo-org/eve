package http

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Execute performs an HTTP request and returns the response
func Execute(req *Request) (*Response, error) {
	startTime := time.Now()

	// Validate request
	if req.Method == "" {
		return nil, fmt.Errorf("HTTP method is required")
	}
	if req.URL == "" {
		return nil, fmt.Errorf("URL is required")
	}

	// Execute with retry logic
	var lastErr error
	attempts := req.RetryCount + 1 // Initial attempt + retries

	for attempt := 0; attempt < attempts; attempt++ {
		resp, err := executeOnce(req)
		if err == nil {
			resp.Duration = time.Since(startTime)
			return resp, nil
		}

		lastErr = err

		// Don't retry on client errors (4xx)
		if resp != nil && resp.IsClientError() {
			resp.Duration = time.Since(startTime)
			return resp, err
		}

		// Don't retry if this was the last attempt
		if attempt < attempts-1 {
			// Calculate backoff
			backoff := calculateBackoff(attempt, req.RetryBackoff, req.RetryInterval)
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", attempts, lastErr)
}

// executeOnce performs a single HTTP request attempt
func executeOnce(req *Request) (*Response, error) {
	// Build HTTP request based on method
	var httpReq *http.Request
	var err error

	switch req.Method {
	case "GET", "HEAD", "OPTIONS":
		httpReq, err = buildSimpleRequest(req)
	case "POST", "PUT", "PATCH":
		httpReq, err = buildBodyRequest(req)
	case "DELETE":
		httpReq, err = buildSimpleRequest(req)
	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", req.Method)
	}

	if err != nil {
		return nil, err
	}

	// Configure HTTP client
	client := &http.Client{
		Timeout: time.Duration(req.Timeout) * time.Second,
	}

	// Configure TLS
	if req.InsecureSkipVerify {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	// Configure proxy
	if req.Proxy != "" {
		proxyURL, err := url.Parse(req.Proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		client.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
	}

	// Configure redirects
	if !req.FollowRedirect {
		client.CheckRedirect = func(httpReq *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else if req.MaxRedirects > 0 {
		maxRedirects := req.MaxRedirects
		client.CheckRedirect = func(httpReq *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			return nil
		}
	}

	// Execute request
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	// Read response body
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Build response
	resp := &Response{
		StatusCode: httpResp.StatusCode,
		Status:     httpResp.Status,
		Headers:    make(map[string]string),
		Body:       body,
		BodyString: string(body),
		FromCache:  false,
	}

	// Copy headers
	for key, values := range httpResp.Header {
		if len(values) > 0 {
			resp.Headers[key] = values[0]
		}
	}

	// Save to file if requested
	if req.SaveTo != "" {
		if err := os.WriteFile(req.SaveTo, body, 0644); err != nil {
			return resp, fmt.Errorf("failed to save response: %w", err)
		}
	}

	// Check for HTTP errors
	if !resp.IsSuccess() {
		return resp, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return resp, nil
}

// buildSimpleRequest builds a request without a body (GET, HEAD, DELETE, OPTIONS)
func buildSimpleRequest(req *Request) (*http.Request, error) {
	httpReq, err := http.NewRequest(req.Method, req.URL, nil)
	if err != nil {
		return nil, err
	}

	// Add headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add User-Agent
	if req.UserAgent != "" {
		httpReq.Header.Set("User-Agent", req.UserAgent)
	}

	return httpReq, nil
}

// buildBodyRequest builds a request with a body (POST, PUT, PATCH)
func buildBodyRequest(req *Request) (*http.Request, error) {
	var body io.Reader
	var contentType string

	// Determine content type and body
	if req.JSONBody != "" {
		// JSON body
		body = strings.NewReader(req.JSONBody)
		contentType = "application/json"
	} else if len(req.Files) > 0 || len(req.FormData) > 0 {
		// Multipart form data (will be handled in multipart.go)
		multipartBody, multipartContentType, err := buildMultipartBody(req)
		if err != nil {
			return nil, err
		}
		body = multipartBody
		contentType = multipartContentType
	} else if req.RawBody != nil {
		// Raw body bytes
		body = bytes.NewReader(req.RawBody)
		contentType = "application/octet-stream"
	} else {
		return nil, fmt.Errorf("%s request requires a body (JSON, form data, or raw bytes)", req.Method)
	}

	httpReq, err := http.NewRequest(req.Method, req.URL, body)
	if err != nil {
		return nil, err
	}

	// Set content type
	httpReq.Header.Set("Content-Type", contentType)

	// Add custom headers (can override Content-Type)
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add User-Agent
	if req.UserAgent != "" {
		httpReq.Header.Set("User-Agent", req.UserAgent)
	}

	return httpReq, nil
}

// calculateBackoff calculates retry backoff duration
func calculateBackoff(attempt int, strategy string, initial time.Duration) time.Duration {
	if strategy == "linear" {
		return initial * time.Duration(attempt+1)
	}

	// Exponential backoff (default)
	multiplier := 1 << uint(attempt) // 2^attempt
	return initial * time.Duration(multiplier)
}
