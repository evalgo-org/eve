package http

import "time"

// Request represents an HTTP operation with all configuration options
type Request struct {
	// HTTP basics
	Method string // GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS
	URL    string // Target URL

	// Headers and authentication
	Headers map[string]string // HTTP headers

	// Request body options
	JSONBody string            // JSON body for application/json requests
	FormData map[string]string // Form data (key-value pairs)
	Files    map[string]string // Files to upload (form field -> file path)
	RawBody  []byte            // Raw body bytes (for custom content types)

	// Response handling
	SaveTo string // Save response to file (optional)

	// Network configuration
	Timeout        int  // Timeout in seconds (0 = default 30s)
	FollowRedirect bool // Follow HTTP redirects (default: true)
	MaxRedirects   int  // Maximum number of redirects (default: 10)

	// Retry configuration
	RetryCount    int           // Number of retries on failure (default: 0)
	RetryBackoff  string        // "exponential" or "linear" (default: "exponential")
	RetryInterval time.Duration // Initial retry interval (default: 1s)

	// Caching
	UseCache       bool   // Enable HTTP caching (ETag, Last-Modified)
	CacheValidator string // Custom cache validation logic

	// TLS/SSL
	InsecureSkipVerify bool // Skip TLS certificate verification (dangerous!)

	// Advanced
	UserAgent string // Custom User-Agent header
	Proxy     string // HTTP proxy URL
}

// NewRequest creates a new Request with sensible defaults
func NewRequest(method, url string) *Request {
	return &Request{
		Method:         method,
		URL:            url,
		Headers:        make(map[string]string),
		FormData:       make(map[string]string),
		Files:          make(map[string]string),
		Timeout:        30,
		FollowRedirect: true,
		MaxRedirects:   10,
		RetryCount:     0,
		RetryBackoff:   "exponential",
		RetryInterval:  1 * time.Second,
		UseCache:       false,
		UserAgent:      "eve-http/1.0",
	}
}

// Response represents an HTTP response with metadata
type Response struct {
	StatusCode int               // HTTP status code
	Status     string            // HTTP status message
	Headers    map[string]string // Response headers
	Body       []byte            // Response body
	BodyString string            // Response body as string
	FromCache  bool              // Whether response came from cache
	Duration   time.Duration     // Request duration
}

// IsSuccess returns true if status code is 2xx
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsRedirect returns true if status code is 3xx
func (r *Response) IsRedirect() bool {
	return r.StatusCode >= 300 && r.StatusCode < 400
}

// IsClientError returns true if status code is 4xx
func (r *Response) IsClientError() bool {
	return r.StatusCode >= 400 && r.StatusCode < 500
}

// IsServerError returns true if status code is 5xx
func (r *Response) IsServerError() bool {
	return r.StatusCode >= 500 && r.StatusCode < 600
}
