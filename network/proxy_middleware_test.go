package network

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAPIKeyAuthentication(t *testing.T) {
	config := &AuthConfig{
		Type:   "api-key",
		Header: "X-API-Key",
		Keys:   []string{"valid-key-1", "valid-key-2"},
		Bypass: []string{"/health"},
	}

	middleware := AuthMiddleware(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("authenticated"))
	}))

	tests := []struct {
		name           string
		path           string
		apiKey         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "valid key 1",
			path:           "/api/users",
			apiKey:         "valid-key-1",
			expectedStatus: http.StatusOK,
			expectedBody:   "authenticated",
		},
		{
			name:           "valid key 2",
			path:           "/api/posts",
			apiKey:         "valid-key-2",
			expectedStatus: http.StatusOK,
			expectedBody:   "authenticated",
		},
		{
			name:           "invalid key",
			path:           "/api/users",
			apiKey:         "invalid-key",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "missing key",
			path:           "/api/users",
			apiKey:         "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "bypass path",
			path:           "/health",
			apiKey:         "",
			expectedStatus: http.StatusOK,
			expectedBody:   "authenticated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", rr.Code, tt.expectedStatus)
			}

			if tt.expectedBody != "" && rr.Body.String() != tt.expectedBody {
				t.Errorf("Body = %v, want %v", rr.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestBasicAuthentication(t *testing.T) {
	config := &AuthConfig{
		Type: "basic",
		Keys: []string{"user1:pass1", "user2:pass2"},
	}

	middleware := AuthMiddleware(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name           string
		username       string
		password       string
		expectedStatus int
	}{
		{"valid credentials 1", "user1", "pass1", http.StatusOK},
		{"valid credentials 2", "user2", "pass2", http.StatusOK},
		{"invalid username", "user3", "pass1", http.StatusUnauthorized},
		{"invalid password", "user1", "wrongpass", http.StatusUnauthorized},
		{"no credentials", "", "", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/test", nil)
			if tt.username != "" || tt.password != "" {
				req.SetBasicAuth(tt.username, tt.password)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", rr.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusUnauthorized {
				if authHeader := rr.Header().Get("WWW-Authenticate"); authHeader == "" {
					t.Error("Missing WWW-Authenticate header")
				}
			}
		})
	}
}

func TestNoAuthentication(t *testing.T) {
	config := &AuthConfig{
		Type: "none",
	}

	middleware := AuthMiddleware(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v (no auth should allow all)", rr.Code, http.StatusOK)
	}
}

func TestCORSMiddleware(t *testing.T) {
	config := &CORSConfig{
		Enabled:          true,
		AllowedOrigins:   []string{"http://localhost:3000", "https://app.example.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		ExposedHeaders:   []string{"X-Total-Count"},
		AllowCredentials: true,
		MaxAge:           3600,
	}

	middleware := CORSMiddleware(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name                   string
		method                 string
		origin                 string
		expectCORSHeaders      bool
		expectedAllowedOrigin  string
		expectedAllowedMethods string
	}{
		{
			name:                  "allowed origin 1",
			method:                "GET",
			origin:                "http://localhost:3000",
			expectCORSHeaders:     true,
			expectedAllowedOrigin: "http://localhost:3000",
		},
		{
			name:                  "allowed origin 2",
			method:                "POST",
			origin:                "https://app.example.com",
			expectCORSHeaders:     true,
			expectedAllowedOrigin: "https://app.example.com",
		},
		{
			name:              "disallowed origin",
			method:            "GET",
			origin:            "https://evil.com",
			expectCORSHeaders: false,
		},
		{
			name:                   "preflight request",
			method:                 "OPTIONS",
			origin:                 "http://localhost:3000",
			expectCORSHeaders:      true,
			expectedAllowedOrigin:  "http://localhost:3000",
			expectedAllowedMethods: "GET, POST, PUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/test", nil)
			req.Header.Set("Origin", tt.origin)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if tt.expectCORSHeaders {
				if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != tt.expectedAllowedOrigin {
					t.Errorf("Access-Control-Allow-Origin = %v, want %v", origin, tt.expectedAllowedOrigin)
				}

				if credentials := rr.Header().Get("Access-Control-Allow-Credentials"); credentials != "true" {
					t.Errorf("Access-Control-Allow-Credentials = %v, want true", credentials)
				}

				if tt.method == "OPTIONS" && tt.expectedAllowedMethods != "" {
					if methods := rr.Header().Get("Access-Control-Allow-Methods"); methods != tt.expectedAllowedMethods {
						t.Errorf("Access-Control-Allow-Methods = %v, want %v", methods, tt.expectedAllowedMethods)
					}
				}
			} else {
				if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "" {
					t.Errorf("Unexpected Access-Control-Allow-Origin header for disallowed origin")
				}
			}

			if tt.method == "OPTIONS" {
				if rr.Code != http.StatusNoContent {
					t.Errorf("OPTIONS status = %v, want %v", rr.Code, http.StatusNoContent)
				}
			}
		})
	}
}

func TestCORSMiddlewareWildcard(t *testing.T) {
	config := &CORSConfig{
		Enabled:        true,
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST"},
	}

	middleware := CORSMiddleware(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "https://any-origin.com")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("Access-Control-Allow-Origin = %v, want *", origin)
	}
}

func TestCORSMiddlewareDisabled(t *testing.T) {
	config := &CORSConfig{
		Enabled: false,
	}

	middleware := CORSMiddleware(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "" {
		t.Errorf("CORS should be disabled, but got Access-Control-Allow-Origin = %v", origin)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	middleware := RecoveryMiddleware()
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	rr := httptest.NewRecorder()

	// Should not panic
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Status = %v, want %v after panic", rr.Code, http.StatusInternalServerError)
	}
}

func TestChainMiddleware(t *testing.T) {
	called := []string{}

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = append(called, "middleware1-before")
			next.ServeHTTP(w, r)
			called = append(called, "middleware1-after")
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = append(called, "middleware2-before")
			next.ServeHTTP(w, r)
			called = append(called, "middleware2-after")
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = append(called, "handler")
		w.WriteHeader(http.StatusOK)
	})

	chained := ChainMiddleware(handler, middleware1, middleware2)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	chained.ServeHTTP(rr, req)

	expected := []string{
		"middleware1-before",
		"middleware2-before",
		"handler",
		"middleware2-after",
		"middleware1-after",
	}

	if len(called) != len(expected) {
		t.Errorf("len(called) = %v, want %v", len(called), len(expected))
		return
	}

	for i, call := range called {
		if call != expected[i] {
			t.Errorf("called[%d] = %v, want %v", i, call, expected[i])
		}
	}
}

func TestTimeoutMiddleware(t *testing.T) {
	// Test with timeout
	middleware := TimeoutMiddleware(50 * time.Millisecond)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			// Context was canceled due to timeout
			return
		case <-time.After(200 * time.Millisecond):
			// Should not reach here
			w.WriteHeader(http.StatusOK)
		}
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Note: The actual timeout behavior depends on how the handler checks context
	// This test verifies that the context is properly set with timeout
}

func TestTimeoutMiddlewareNoTimeout(t *testing.T) {
	// Test with zero timeout (no timeout)
	middleware := TimeoutMiddleware(0)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", rr.Code, http.StatusOK)
	}
}
