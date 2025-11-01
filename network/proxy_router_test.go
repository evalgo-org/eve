package network

import (
	"net/http/httptest"
	"testing"
)

func TestNewRouter(t *testing.T) {
	config := &ProxyConfig{
		Routes: []RouteConfig{
			{Path: "/api/v1/users", Methods: []string{"GET"}},
			{Path: "/api/v1/posts/*", Methods: []string{"GET", "POST"}},
			{Path: "/users/:id", Methods: []string{"GET"}},
		},
	}

	router := NewRouter(config)

	if len(router.staticRoutes) != 1 {
		t.Errorf("len(staticRoutes) = %v, want 1", len(router.staticRoutes))
	}
	if len(router.patternRoutes) != 2 {
		t.Errorf("len(patternRoutes) = %v, want 2", len(router.patternRoutes))
	}
}

func TestRouterMatchStaticRoute(t *testing.T) {
	config := &ProxyConfig{
		Routes: []RouteConfig{
			{Path: "/api/v1/users", Methods: []string{"GET"}},
			{Path: "/api/v1/posts", Methods: []string{"POST"}},
		},
	}

	router := NewRouter(config)

	tests := []struct {
		name        string
		method      string
		path        string
		shouldMatch bool
	}{
		{"exact match GET", "GET", "/api/v1/users", true},
		{"exact match POST", "POST", "/api/v1/posts", true},
		{"no match - wrong path", "GET", "/api/v1/comments", false},
		{"no match - wrong method", "POST", "/api/v1/users", false},
		{"no match - extra path", "GET", "/api/v1/users/123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			match := router.Match(req)

			if (match != nil) != tt.shouldMatch {
				t.Errorf("Match() = %v, want match = %v", match != nil, tt.shouldMatch)
			}
		})
	}
}

func TestRouterMatchWildcardRoute(t *testing.T) {
	config := &ProxyConfig{
		Routes: []RouteConfig{
			{Path: "/api/v1/*", Methods: []string{"GET"}},
			{Path: "/files/*", Methods: []string{}}, // No methods = allow all
		},
	}

	router := NewRouter(config)

	tests := []struct {
		name        string
		method      string
		path        string
		shouldMatch bool
	}{
		{"wildcard match", "GET", "/api/v1/users", true},
		{"wildcard deep match", "GET", "/api/v1/users/123/posts", true},
		{"wildcard no match", "GET", "/api/v2/users", false},
		{"no method restriction", "POST", "/files/upload", true},
		{"no method restriction GET", "GET", "/files/download", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			match := router.Match(req)

			if (match != nil) != tt.shouldMatch {
				t.Errorf("Match() = %v, want match = %v", match != nil, tt.shouldMatch)
			}
		})
	}
}

func TestRouterMatchParameterRoute(t *testing.T) {
	config := &ProxyConfig{
		Routes: []RouteConfig{
			{Path: "/users/:id", Methods: []string{"GET"}},
			{Path: "/posts/:postId/comments/:commentId", Methods: []string{"GET"}},
		},
	}

	router := NewRouter(config)

	tests := []struct {
		name           string
		method         string
		path           string
		shouldMatch    bool
		expectedParams map[string]string
	}{
		{
			name:           "single parameter",
			method:         "GET",
			path:           "/users/123",
			shouldMatch:    true,
			expectedParams: map[string]string{"id": "123"},
		},
		{
			name:           "multiple parameters",
			method:         "GET",
			path:           "/posts/456/comments/789",
			shouldMatch:    true,
			expectedParams: map[string]string{"postId": "456", "commentId": "789"},
		},
		{
			name:        "no match - missing segment",
			method:      "GET",
			path:        "/users",
			shouldMatch: false,
		},
		{
			name:        "no match - extra segments",
			method:      "GET",
			path:        "/users/123/extra",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			match := router.Match(req)

			if (match != nil) != tt.shouldMatch {
				t.Errorf("Match() = %v, want match = %v", match != nil, tt.shouldMatch)
				return
			}

			if match != nil && tt.expectedParams != nil {
				for key, expected := range tt.expectedParams {
					if actual, ok := match.Params[key]; !ok || actual != expected {
						t.Errorf("Param[%s] = %v, want %v", key, actual, expected)
					}
				}
			}
		})
	}
}

func TestRouterMethodAllowed(t *testing.T) {
	config := &ProxyConfig{
		Routes: []RouteConfig{
			{Path: "/readonly", Methods: []string{"GET"}},
			{Path: "/writeonly", Methods: []string{"POST", "PUT"}},
			{Path: "/anymethod", Methods: []string{}},
		},
	}

	router := NewRouter(config)

	tests := []struct {
		name        string
		method      string
		path        string
		shouldMatch bool
	}{
		{"allowed GET", "GET", "/readonly", true},
		{"not allowed POST", "POST", "/readonly", false},
		{"allowed POST", "POST", "/writeonly", true},
		{"allowed PUT", "PUT", "/writeonly", true},
		{"not allowed GET", "GET", "/writeonly", false},
		{"any method GET", "GET", "/anymethod", true},
		{"any method POST", "POST", "/anymethod", true},
		{"any method DELETE", "DELETE", "/anymethod", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			match := router.Match(req)

			if (match != nil) != tt.shouldMatch {
				t.Errorf("Match() = %v, want match = %v", match != nil, tt.shouldMatch)
			}
		})
	}
}

func TestRewritePath(t *testing.T) {
	tests := []struct {
		name         string
		originalPath string
		route        RouteConfig
		params       map[string]string
		expected     string
	}{
		{
			name:         "no rewrite",
			originalPath: "/api/v1/users",
			route:        RouteConfig{Path: "/api/v1/users"},
			params:       map[string]string{},
			expected:     "/api/v1/users",
		},
		{
			name:         "strip prefix",
			originalPath: "/external/api/users",
			route:        RouteConfig{Path: "/external/api/*", StripPrefix: true},
			params:       map[string]string{},
			expected:     "/users",
		},
		{
			name:         "add prefix",
			originalPath: "/users",
			route:        RouteConfig{Path: "/users", AddPrefix: "/api/v1"},
			params:       map[string]string{},
			expected:     "/api/v1/users",
		},
		{
			name:         "strip and add prefix",
			originalPath: "/external/users",
			route:        RouteConfig{Path: "/external/*", StripPrefix: true, AddPrefix: "/internal"},
			params:       map[string]string{},
			expected:     "/internal/users",
		},
		{
			name:         "replace parameter",
			originalPath: "/users/123",
			route:        RouteConfig{Path: "/users/:id"},
			params:       map[string]string{"id": "123"},
			expected:     "/users/123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RewritePath(tt.originalPath, &tt.route, tt.params)
			if result != tt.expected {
				t.Errorf("RewritePath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRouterGetAllowedMethods(t *testing.T) {
	config := &ProxyConfig{
		Routes: []RouteConfig{
			{Path: "/users", Methods: []string{"GET", "POST"}},
			{Path: "/posts/*", Methods: []string{"GET"}},
			{Path: "/files/*", Methods: []string{}}, // Allow all
		},
	}

	router := NewRouter(config)

	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{
			name:     "specific methods",
			path:     "/users",
			expected: []string{"GET", "POST", "OPTIONS"},
		},
		{
			name:     "wildcard route",
			path:     "/posts/123",
			expected: []string{"GET", "OPTIONS"},
		},
		{
			name:     "allow all methods",
			path:     "/files/upload",
			expected: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"},
		},
		{
			name:     "no matching route",
			path:     "/nonexistent",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			methods := router.GetAllowedMethods(tt.path)

			if len(methods) != len(tt.expected) {
				t.Errorf("len(GetAllowedMethods()) = %v, want %v", len(methods), len(tt.expected))
				return
			}

			// Check all expected methods are present
			methodMap := make(map[string]bool)
			for _, m := range methods {
				methodMap[m] = true
			}

			for _, expected := range tt.expected {
				if !methodMap[expected] {
					t.Errorf("GetAllowedMethods() missing method %v", expected)
				}
			}
		})
	}
}

func TestRouterPriority(t *testing.T) {
	// Static routes should match before pattern routes
	config := &ProxyConfig{
		Routes: []RouteConfig{
			{Path: "/api/*", Methods: []string{"GET"}},     // Pattern route
			{Path: "/api/users", Methods: []string{"GET"}}, // Static route
		},
	}

	router := NewRouter(config)

	req := httptest.NewRequest("GET", "/api/users", nil)
	match := router.Match(req)

	if match == nil {
		t.Fatal("Match() returned nil")
	}

	// Should match the static route, not the wildcard
	if match.Route.Path != "/api/users" {
		t.Errorf("Matched route path = %v, want /api/users (static route should have priority)", match.Route.Path)
	}
}

func TestRouterCaseSensitivity(t *testing.T) {
	config := &ProxyConfig{
		Routes: []RouteConfig{
			{Path: "/api/Users", Methods: []string{"GET"}},
		},
	}

	router := NewRouter(config)

	tests := []struct {
		name        string
		path        string
		shouldMatch bool
	}{
		{"exact case", "/api/Users", true},
		{"lowercase", "/api/users", false}, // Paths are case-sensitive
		{"uppercase", "/API/USERS", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			match := router.Match(req)

			if (match != nil) != tt.shouldMatch {
				t.Errorf("Match() = %v, want match = %v", match != nil, tt.shouldMatch)
			}
		})
	}
}
