package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestSetGetAuthUser(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test when no user is set
	user, ok := GetUser(c)
	assert.False(t, ok)
	assert.Nil(t, user)

	// Test setting and getting user
	expectedUser := &AuthUser{
		ID:       "user123",
		Username: "john.doe",
		Email:    "john@example.com",
		Name:     "John Doe",
		Scopes:   []string{"read", "write"},
		Claims:   map[string]interface{}{"role": "admin"},
	}

	SetUser(c, expectedUser)
	user, ok = GetUser(c)
	assert.True(t, ok)
	assert.Equal(t, expectedUser, user)
}

func TestSetGetClaims(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test when no claims are set
	claims, ok := GetClaims(c)
	assert.False(t, ok)
	assert.Nil(t, claims)

	// Test setting and getting claims
	expectedClaims := map[string]interface{}{
		"sub":   "user123",
		"email": "john@example.com",
		"role":  "admin",
	}

	SetClaims(c, expectedClaims)
	claims, ok = GetClaims(c)
	assert.True(t, ok)
	assert.Equal(t, expectedClaims, claims)
}

func TestSetGetScopes(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test when no scopes are set
	scopes, ok := GetScopes(c)
	assert.False(t, ok)
	assert.Nil(t, scopes)

	// Test setting and getting scopes
	expectedScopes := []string{"read", "write", "admin"}

	SetScopes(c, expectedScopes)
	scopes, ok = GetScopes(c)
	assert.True(t, ok)
	assert.Equal(t, expectedScopes, scopes)
}

func TestRequireScope(t *testing.T) {
	tests := []struct {
		name           string
		requiredScopes []string
		userScopes     []string
		setupContext   func(*echo.Context)
		expectedStatus int
	}{
		{
			name:           "user has required scope",
			requiredScopes: []string{"read"},
			setupContext: func(c *echo.Context) {
				SetUser(*c, &AuthUser{
					ID:     "user1",
					Scopes: []string{"read", "write"},
				})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user has one of multiple required scopes",
			requiredScopes: []string{"admin", "write"},
			setupContext: func(c *echo.Context) {
				SetUser(*c, &AuthUser{
					ID:     "user1",
					Scopes: []string{"read", "write"},
				})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user missing required scope",
			requiredScopes: []string{"admin"},
			setupContext: func(c *echo.Context) {
				SetUser(*c, &AuthUser{
					ID:     "user1",
					Scopes: []string{"read", "write"},
				})
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "no user in context",
			requiredScopes: []string{"read"},
			setupContext:   func(c *echo.Context) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "user with no scopes",
			requiredScopes: []string{"read"},
			setupContext: func(c *echo.Context) {
				SetUser(*c, &AuthUser{
					ID:     "user1",
					Scopes: []string{},
				})
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "scopes from direct context",
			requiredScopes: []string{"read"},
			setupContext: func(c *echo.Context) {
				SetScopes(*c, []string{"read", "write"})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "scopes from claims (space-separated)",
			requiredScopes: []string{"read"},
			setupContext: func(c *echo.Context) {
				SetClaims(*c, map[string]interface{}{
					"scope": "read write admin",
				})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "scopes from claims (array)",
			requiredScopes: []string{"admin"},
			setupContext: func(c *echo.Context) {
				SetClaims(*c, map[string]interface{}{
					"scope": []interface{}{"read", "write", "admin"},
				})
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Setup context
			tt.setupContext(&c)

			// Create handler
			handler := func(c echo.Context) error {
				return c.String(http.StatusOK, "success")
			}

			// Apply middleware
			middleware := RequireScope(tt.requiredScopes...)
			err := middleware(handler)(c)

			if tt.expectedStatus == http.StatusOK {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Code)
			} else {
				assert.Error(t, err)
				httpErr, ok := err.(*echo.HTTPError)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedStatus, httpErr.Code)
			}
		})
	}
}

func TestRequireAllScopes(t *testing.T) {
	tests := []struct {
		name           string
		requiredScopes []string
		setupContext   func(*echo.Context)
		expectedStatus int
	}{
		{
			name:           "user has all required scopes",
			requiredScopes: []string{"read", "write"},
			setupContext: func(c *echo.Context) {
				SetUser(*c, &AuthUser{
					ID:     "user1",
					Scopes: []string{"read", "write", "admin"},
				})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user missing one scope",
			requiredScopes: []string{"read", "write", "admin"},
			setupContext: func(c *echo.Context) {
				SetUser(*c, &AuthUser{
					ID:     "user1",
					Scopes: []string{"read", "write"},
				})
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "user has no scopes",
			requiredScopes: []string{"read", "write"},
			setupContext: func(c *echo.Context) {
				SetUser(*c, &AuthUser{
					ID:     "user1",
					Scopes: []string{},
				})
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "single required scope present",
			requiredScopes: []string{"admin"},
			setupContext: func(c *echo.Context) {
				SetUser(*c, &AuthUser{
					ID:     "user1",
					Scopes: []string{"admin"},
				})
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Setup context
			tt.setupContext(&c)

			// Create handler
			handler := func(c echo.Context) error {
				return c.String(http.StatusOK, "success")
			}

			// Apply middleware
			middleware := RequireAllScopes(tt.requiredScopes...)
			err := middleware(handler)(c)

			if tt.expectedStatus == http.StatusOK {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Code)
			} else {
				assert.Error(t, err)
				httpErr, ok := err.(*echo.HTTPError)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedStatus, httpErr.Code)
			}
		})
	}
}

func TestExtractScopesFromClaims(t *testing.T) {
	tests := []struct {
		name     string
		claims   map[string]interface{}
		expected []string
	}{
		{
			name: "space-separated scope string",
			claims: map[string]interface{}{
				"scope": "read write admin",
			},
			expected: []string{"read", "write", "admin"},
		},
		{
			name: "scope array",
			claims: map[string]interface{}{
				"scope": []interface{}{"read", "write", "admin"},
			},
			expected: []string{"read", "write", "admin"},
		},
		{
			name: "scopes array (alternative claim name)",
			claims: map[string]interface{}{
				"scopes": []interface{}{"read", "write"},
			},
			expected: []string{"read", "write"},
		},
		{
			name: "no scope claim",
			claims: map[string]interface{}{
				"sub": "user123",
			},
			expected: nil,
		},
		{
			name: "empty scope string",
			claims: map[string]interface{}{
				"scope": "",
			},
			expected: nil,
		},
		{
			name: "single scope",
			claims: map[string]interface{}{
				"scope": "admin",
			},
			expected: []string{"admin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractScopesFromClaims(tt.claims)
			assert.Equal(t, tt.expected, result)
		})
	}
}
