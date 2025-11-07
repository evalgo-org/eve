package api

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"eve.evalgo.org/security"
)

func TestBasicAuthMiddleware(t *testing.T) {
	// Generate test password hash
	passwordHash, err := security.HashPassword("secret123")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	tests := []struct {
		name           string
		config         BasicAuthConfig
		authHeader     string
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "valid credentials with plaintext password",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret123",
			},
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret123")),
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name: "valid credentials with bcrypt hash",
			config: BasicAuthConfig{
				Username:     "admin",
				PasswordHash: passwordHash,
			},
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret123")),
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name: "invalid username",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret123",
			},
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("wronguser:secret123")),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid password",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret123",
			},
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:wrongpass")),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing authorization header",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret123",
			},
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid authorization format",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret123",
			},
			authHeader:     "Bearer token123",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid base64 encoding",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret123",
			},
			authHeader:     "Basic not-valid-base64!!!",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing colon in credentials",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret123",
			},
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("adminnosecret")),
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Create handler
			handler := func(c echo.Context) error {
				return c.String(http.StatusOK, "success")
			}

			// Apply middleware
			middleware := BasicAuthMiddleware(tt.config)
			err := middleware(handler)(c)

			if tt.expectedStatus == http.StatusOK {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Code)
				if tt.expectedBody != "" {
					assert.Equal(t, tt.expectedBody, rec.Body.String())
				}
			} else {
				assert.Error(t, err)
				httpErr, ok := err.(*echo.HTTPError)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedStatus, httpErr.Code)
			}
		})
	}
}

func TestBasicAuthMiddleware_Skipper(t *testing.T) {
	config := BasicAuthConfig{
		Username: "admin",
		Password: "secret123",
		Skipper: func(c echo.Context) bool {
			return c.Path() == "/health"
		},
	}

	tests := []struct {
		name           string
		path           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "skipped endpoint without auth",
			path:           "/health",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "protected endpoint without auth",
			path:           "/api",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "protected endpoint with auth",
			path:           "/api",
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret123")),
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath(tt.path)

			// Create handler
			handler := func(c echo.Context) error {
				return c.String(http.StatusOK, "success")
			}

			// Apply middleware
			middleware := BasicAuthMiddleware(config)
			err := middleware(handler)(c)

			if tt.expectedStatus == http.StatusOK {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestBasicAuthMiddleware_CustomValidator(t *testing.T) {
	customValidator := func(username, password string, c echo.Context) bool {
		// Custom validation: username must be "custom" and password must be "pass123"
		return username == "custom" && password == "pass123"
	}

	config := BasicAuthConfig{
		Validator: customValidator,
	}

	tests := []struct {
		name           string
		username       string
		password       string
		expectedStatus int
	}{
		{
			name:           "valid custom credentials",
			username:       "custom",
			password:       "pass123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid custom credentials",
			username:       "admin",
			password:       "secret",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			credentials := base64.StdEncoding.EncodeToString([]byte(tt.username + ":" + tt.password))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Basic "+credentials)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Create handler
			handler := func(c echo.Context) error {
				return c.String(http.StatusOK, "success")
			}

			// Apply middleware
			middleware := BasicAuthMiddleware(config)
			err := middleware(handler)(c)

			if tt.expectedStatus == http.StatusOK {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestBasicAuthMiddleware_Realm(t *testing.T) {
	tests := []struct {
		name          string
		config        BasicAuthConfig
		expectedRealm string
	}{
		{
			name: "default realm",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret",
			},
			expectedRealm: "Restricted",
		},
		{
			name: "custom realm",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret",
				Realm:    "Admin Area",
			},
			expectedRealm: "Admin Area",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Create handler
			handler := func(c echo.Context) error {
				return c.String(http.StatusOK, "success")
			}

			// Apply middleware (should fail and set WWW-Authenticate)
			middleware := BasicAuthMiddleware(tt.config)
			_ = middleware(handler)(c)

			// Check WWW-Authenticate header
			wwwAuth := rec.Header().Get("WWW-Authenticate")
			assert.Contains(t, wwwAuth, tt.expectedRealm)
			assert.Contains(t, wwwAuth, "Basic realm=")
		})
	}
}

func TestBasicAuthMiddleware_UsernameInContext(t *testing.T) {
	config := BasicAuthConfig{
		Username: "testuser",
		Password: "testpass",
	}

	e := echo.New()
	credentials := base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic "+credentials)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Create handler that checks username in context
	handler := func(c echo.Context) error {
		username := GetBasicAuthUsername(c)
		assert.Equal(t, "testuser", username)
		return c.String(http.StatusOK, "success")
	}

	// Apply middleware
	middleware := BasicAuthMiddleware(config)
	err := middleware(handler)(c)

	assert.NoError(t, err)
}

func TestGetBasicAuthUsername(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(echo.Context)
		expected string
	}{
		{
			name: "username in context",
			setup: func(c echo.Context) {
				c.Set("username", "john.doe")
			},
			expected: "john.doe",
		},
		{
			name:     "no username in context",
			setup:    func(c echo.Context) {},
			expected: "",
		},
		{
			name: "wrong type in context",
			setup: func(c echo.Context) {
				c.Set("username", 12345)
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			tt.setup(c)
			username := GetBasicAuthUsername(c)
			assert.Equal(t, tt.expected, username)
		})
	}
}

func TestParseBasicAuth(t *testing.T) {
	tests := []struct {
		name         string
		authHeader   string
		wantUsername string
		wantPassword string
		wantErr      bool
	}{
		{
			name:         "valid credentials",
			authHeader:   "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass")),
			wantUsername: "user",
			wantPassword: "pass",
			wantErr:      false,
		},
		{
			name:         "password with colon",
			authHeader:   "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass:word")),
			wantUsername: "user",
			wantPassword: "pass:word",
			wantErr:      false,
		},
		{
			name:       "invalid prefix",
			authHeader: "Bearer token123",
			wantErr:    true,
		},
		{
			name:       "invalid base64",
			authHeader: "Basic not-valid!!!",
			wantErr:    true,
		},
		{
			name:       "missing colon",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("userpass")),
			wantErr:    true,
		},
		{
			name:       "empty string",
			authHeader: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username, password, err := parseBasicAuth(tt.authHeader)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantUsername, username)
				assert.Equal(t, tt.wantPassword, password)
			}
		})
	}
}

func TestValidateCredentials(t *testing.T) {
	passwordHash, _ := security.HashPassword("correct")

	tests := []struct {
		name     string
		config   BasicAuthConfig
		username string
		password string
		expected bool
	}{
		{
			name: "valid plaintext password",
			config: BasicAuthConfig{
				Username: "user",
				Password: "correct",
			},
			username: "user",
			password: "correct",
			expected: true,
		},
		{
			name: "invalid plaintext password",
			config: BasicAuthConfig{
				Username: "user",
				Password: "correct",
			},
			username: "user",
			password: "wrong",
			expected: false,
		},
		{
			name: "valid bcrypt password",
			config: BasicAuthConfig{
				Username:     "user",
				PasswordHash: passwordHash,
			},
			username: "user",
			password: "correct",
			expected: true,
		},
		{
			name: "invalid bcrypt password",
			config: BasicAuthConfig{
				Username:     "user",
				PasswordHash: passwordHash,
			},
			username: "user",
			password: "wrong",
			expected: false,
		},
		{
			name: "wrong username",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "correct",
			},
			username: "user",
			password: "correct",
			expected: false,
		},
		{
			name: "bcrypt takes precedence over plaintext",
			config: BasicAuthConfig{
				Username:     "user",
				Password:     "plaintext",
				PasswordHash: passwordHash,
			},
			username: "user",
			password: "correct", // matches hash, not plaintext
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateCredentials(tt.username, tt.password, tt.config)
			assert.Equal(t, tt.expected, result)
		})
	}
}
