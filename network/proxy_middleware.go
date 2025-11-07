package network

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	eve "eve.evalgo.org/common"
)

// Middleware represents an HTTP middleware function
type Middleware func(http.Handler) http.Handler

// ChainMiddleware chains multiple middleware functions
func ChainMiddleware(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// AuthMiddleware creates authentication middleware
func AuthMiddleware(config *AuthConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for bypass paths
			for _, path := range config.Bypass {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			switch config.Type {
			case "api-key":
				if !validateAPIKey(r, config) {
					http.Error(w, "Unauthorized: Invalid API Key", http.StatusUnauthorized)
					return
				}
			case "jwt":
				if !validateJWT(r, config) {
					http.Error(w, "Unauthorized: Invalid JWT Token", http.StatusUnauthorized)
					return
				}
			case "basic":
				if !validateBasicAuth(r, config) {
					w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			case "none":
				// No authentication required
			default:
				http.Error(w, "Unauthorized: Unknown auth type", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// validateAPIKey validates API key from request header
func validateAPIKey(r *http.Request, config *AuthConfig) bool {
	header := config.Header
	if header == "" {
		header = "X-API-Key"
	}

	providedKey := r.Header.Get(header)
	if providedKey == "" {
		return false
	}

	// Constant-time comparison to prevent timing attacks
	for _, validKey := range config.Keys {
		if subtle.ConstantTimeCompare([]byte(providedKey), []byte(validKey)) == 1 {
			return true
		}
	}

	return false
}

// validateJWT validates JWT token from request header
func validateJWT(r *http.Request, config *AuthConfig) bool {
	if config.JWT == nil {
		return false
	}

	header := config.Header
	if header == "" {
		header = "Authorization"
	}

	authHeader := r.Header.Get(header)
	if authHeader == "" {
		return false
	}

	// Extract token from "Bearer <token>"
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		// No "Bearer " prefix, try using the whole header
		tokenString = authHeader
	}

	// Parse and validate JWT
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing algorithm
		if config.JWT.Algorithm != "" {
			if token.Method.Alg() != config.JWT.Algorithm {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Method.Alg())
			}
		}

		// Return the secret or public key
		if config.JWT.Secret != "" {
			return []byte(config.JWT.Secret), nil
		}
		// TODO: Load public key from file if needed
		return nil, fmt.Errorf("no signing key configured")
	})

	if err != nil || !token.Valid {
		return false
	}

	// Validate claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false
	}

	// Validate issuer
	if config.JWT.Issuer != "" {
		if iss, ok := claims["iss"].(string); !ok || iss != config.JWT.Issuer {
			return false
		}
	}

	// Validate audience
	if len(config.JWT.Audience) > 0 {
		aud, ok := claims["aud"].(string)
		if !ok {
			return false
		}
		validAud := false
		for _, expectedAud := range config.JWT.Audience {
			if aud == expectedAud {
				validAud = true
				break
			}
		}
		if !validAud {
			return false
		}
	}

	// Validate required claims
	for _, requiredClaim := range config.JWT.RequiredClaims {
		if _, ok := claims[requiredClaim]; !ok {
			return false
		}
	}

	return true
}

// validateBasicAuth validates basic authentication
func validateBasicAuth(r *http.Request, config *AuthConfig) bool {
	username, password, ok := r.BasicAuth()
	if !ok {
		return false
	}

	// Check if username:password matches any of the configured keys
	credentials := username + ":" + password
	for _, validCred := range config.Keys {
		if subtle.ConstantTimeCompare([]byte(credentials), []byte(validCred)) == 1 {
			return true
		}
	}

	return false
}

// CORSMiddleware creates CORS middleware
func CORSMiddleware(config *CORSConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if config == nil || !config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowedOrigin := ""
			for _, allowedOrig := range config.AllowedOrigins {
				if allowedOrig == "*" || allowedOrig == origin {
					allowedOrigin = allowedOrig
					break
				}
			}

			if allowedOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)

				if config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				// Set allowed methods
				if len(config.AllowedMethods) > 0 {
					w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
				}

				// Set allowed headers
				if len(config.AllowedHeaders) > 0 {
					w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
				}

				// Set exposed headers
				if len(config.ExposedHeaders) > 0 {
					w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
				}

				// Set max age
				if config.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
				}
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// LoggingMiddleware creates request logging middleware
func LoggingMiddleware(config *LoggingConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if config == nil || !config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Skip logging for excluded paths
			for _, path := range config.ExcludePaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			start := time.Now()

			// Wrap response writer to capture status code
			lrw := &loggingResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Process request
			next.ServeHTTP(lrw, r)

			// Log request details
			duration := time.Since(start)

			logData := map[string]interface{}{
				"method":      r.Method,
				"path":        r.URL.Path,
				"query":       r.URL.RawQuery,
				"status":      lrw.statusCode,
				"duration_ms": duration.Milliseconds(),
				"remote_addr": r.RemoteAddr,
				"user_agent":  r.UserAgent(),
			}

			if config.Format == "json" {
				logJSON, _ := json.Marshal(logData)
				eve.Logger.Info(string(logJSON))
			} else {
				eve.Logger.Info(fmt.Sprintf("%s %s?%s - %d (%dms)",
					r.Method, r.URL.Path, r.URL.RawQuery, lrw.statusCode, duration.Milliseconds()))
			}
		})
	}
}

// loggingResponseWriter wraps http.ResponseWriter to capture status code
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// RecoveryMiddleware creates panic recovery middleware
func RecoveryMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					eve.Logger.Error(fmt.Sprintf("Panic recovered: %v", err))
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// TimeoutMiddleware creates request timeout middleware
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if timeout <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}
