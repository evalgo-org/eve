// Package security provides authentication and authorization utilities for SAP BTP XSUAA integration.
// This package implements OAuth 2.0 client credentials flow for machine-to-machine authentication
// with SAP Business Technology Platform (BTP) services using XSUAA (Extended Services for UAA).
package security

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

var (
	// xsuaaToken stores the cached XSUAA access token for client credentials flow.
	// This is reused across requests until expiration.
	xsuaaToken *TokenResponse = nil
)

// XSUAACredentials represents the XSUAA service binding credentials from Cloud Foundry.
// These credentials are typically provided through the VCAP_SERVICES environment variable
// when your application is bound to an XSUAA service instance.
type XSUAACredentials struct {
	// URL is the XSUAA service base URL (e.g., "https://tenant.authentication.eu10.hana.ondemand.com")
	URL string `json:"url"`

	// ClientID is the OAuth 2.0 client identifier for your application
	ClientID string `json:"clientid"`

	// ClientSecret is the OAuth 2.0 client secret for authentication
	ClientSecret string `json:"clientsecret"`

	// XSAppName is the application name registered in XSUAA
	XSAppName string `json:"xsappname"`
}

// VCAPServices represents the Cloud Foundry VCAP_SERVICES environment variable structure.
// This is automatically provided by Cloud Foundry when services are bound to your application.
type VCAPServices struct {
	// XSUAA contains one or more XSUAA service bindings
	XSUAA []struct {
		Credentials XSUAACredentials `json:"credentials"`
	} `json:"xsuaa"`
}

// TokenResponse represents the OAuth 2.0 token response from XSUAA.
// This contains the access token and metadata required for authenticated API calls.
type TokenResponse struct {
	// AccessToken is the JWT bearer token for API authentication
	AccessToken string `json:"access_token"`

	// TokenType is the type of token (typically "bearer")
	TokenType string `json:"token_type"`

	// ExpiresIn indicates token validity duration in seconds
	ExpiresIn int `json:"expires_in"`
}

// GetXSUAAToken obtains an OAuth 2.0 access token from XSUAA using client credentials flow.
// This is used for machine-to-machine authentication where no user interaction is required.
//
// The client credentials grant is suitable for:
//   - Backend service-to-service communication
//   - Scheduled jobs and batch processes
//   - API clients that act on behalf of the application itself
//
// Parameters:
//   - tokenURL: The XSUAA token endpoint URL (typically "{xsuaa-url}/oauth/token")
//   - clientID: Your application's OAuth client ID
//   - clientSecret: Your application's OAuth client secret
//
// Returns:
//   - *TokenResponse: Contains the access token and expiration information
//   - error: Any error encountered during token acquisition
//
// Example:
//
//	tokenURL := "https://tenant.authentication.eu10.hana.ondemand.com/oauth/token"
//	token, err := GetXSUAAToken(tokenURL, "client-id", "client-secret")
//	if err != nil {
//	    log.Fatalf("Failed to get token: %v", err)
//	}
//	fmt.Printf("Access token: %s\n", token.AccessToken)
func GetXSUAAToken(tokenURL, clientID, clientSecret string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}

// GetXSUAACredentials retrieves XSUAA service credentials from the VCAP_SERVICES environment variable.
// This function is designed for Cloud Foundry environments where service bindings are automatically
// provided through environment variables.
//
// The VCAP_SERVICES variable contains JSON with all bound services. This function extracts
// the first XSUAA service binding credentials.
//
// Returns:
//   - *XSUAACredentials: The XSUAA service credentials including URL, client ID, and secret
//   - error: Error if VCAP_SERVICES is not set, malformed, or contains no XSUAA service
//
// Example:
//
//	creds, err := GetXSUAACredentials()
//	if err != nil {
//	    log.Fatalf("Failed to get XSUAA credentials: %v", err)
//	}
//	fmt.Printf("XSUAA URL: %s\n", creds.URL)
func GetXSUAACredentials() (*XSUAACredentials, error) {
	vcapServices := os.Getenv("VCAP_SERVICES")
	if vcapServices == "" {
		return nil, fmt.Errorf("VCAP_SERVICES environment variable not set")
	}

	var services VCAPServices
	if err := json.Unmarshal([]byte(vcapServices), &services); err != nil {
		return nil, fmt.Errorf("failed to parse VCAP_SERVICES: %w", err)
	}

	if len(services.XSUAA) == 0 {
		return nil, fmt.Errorf("no XSUAA service found in VCAP_SERVICES")
	}

	return &services.XSUAA[0].Credentials, nil
}

// HandleWithClientCredentials is an Echo HTTP handler that forwards requests to a backend service
// using XSUAA client credentials authentication. This demonstrates how to use client credentials
// tokens to authenticate service-to-service API calls.
//
// The handler:
//  1. Constructs a target URL with query parameters
//  2. Adds the XSUAA bearer token to the Authorization header
//  3. Forwards the request to the backend service
//  4. Returns the backend response to the client
//
// Query Parameters:
//   - product_group: Product category identifier
//   - publication_id: Publication identifier for the X-Publication-Id header
//   - filter_columns: Comma-separated list of columns to filter
//
// Returns:
//   - error: Any error encountered during request processing
//
// Example usage in Echo router:
//
//	e := echo.New()
//	e.GET("/api/response", HandleWithClientCredentials)
func HandleWithClientCredentials(c echo.Context) error {
	// Base URL for the backend service
	baseURL := "https://api.example.com/v1/data"

	// Construct query parameters using url.Values for proper encoding
	params := url.Values{}
	params.Set("language", "de-DE")
	params.Set("nodeKey", c.QueryParam("product_group"))
	params.Set("filterColumns", c.QueryParam("filter_columns"))

	// Build full URL with encoded query string
	targetURL := baseURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(c.Request().Context(), "GET", targetURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create backend request: %w", err)
	}

	// Add XSUAA bearer token for authentication
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", xsuaaToken.AccessToken))
	req.Header.Set("X-Publication-Id", c.QueryParam("publication_id"))

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("backend request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read and forward backend response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to read response body")
	}

	// Log response size for monitoring
	bodySizeBytes := len(body)
	bodySizeMB := float64(bodySizeBytes) / (1024 * 1024)
	fmt.Printf("Response body size: %.2f MB (%d bytes)\n", bodySizeMB, bodySizeBytes)

	// Preserve backend content type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}

	return c.Blob(resp.StatusCode, contentType, body)
}

// APIKeyAuth creates an Echo middleware for API key authentication.
// This middleware validates incoming requests against an API key stored in the API_KEY environment variable.
// The /health endpoint is exempted from authentication to allow health checks.
//
// The middleware expects the API key in the X-API-Key request header.
//
// Returns:
//   - echo.MiddlewareFunc: Configured middleware function
//
// Example:
//
//	e := echo.New()
//	api := e.Group("/api")
//	api.Use(APIKeyAuth())
//	api.GET("/protected", handler) // This endpoint requires X-API-Key header
//
//	e.GET("/health", healthHandler) // This endpoint is public
func APIKeyAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip authentication for health check endpoint
			if c.Path() == "/health" {
				return next(c)
			}

			// Validate API_KEY environment variable is configured
			expectedAPIKey := os.Getenv("API_KEY")
			if expectedAPIKey == "" {
				c.Logger().Error("API_KEY environment variable not set")
				return echo.NewHTTPError(http.StatusInternalServerError, "Server configuration error")
			}

			// Extract API key from request header
			apiKey := c.Request().Header.Get("X-API-Key")

			// Validate API key is provided
			if apiKey == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing API key")
			}

			// Validate API key matches expected value
			if apiKey != expectedAPIKey {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid API key")
			}

			// Authentication successful, proceed to handler
			return next(c)
		}
	}
}

// SetGlobalXSUAAToken sets the global XSUAA token for use in handlers.
// This is typically called during application initialization after obtaining
// the token via GetXSUAAToken.
//
// Parameters:
//   - token: The token response to cache globally
//
// Example:
//
//	token, err := GetXSUAAToken(tokenURL, clientID, clientSecret)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	SetGlobalXSUAAToken(token)
func SetGlobalXSUAAToken(token *TokenResponse) {
	xsuaaToken = token
}

// GetGlobalXSUAAToken retrieves the currently cached XSUAA token.
// Returns nil if no token has been set.
//
// Returns:
//   - *TokenResponse: The cached token, or nil if not set
//
// Example:
//
//	token := GetGlobalXSUAAToken()
//	if token == nil {
//	    log.Println("No token cached")
//	}
func GetGlobalXSUAAToken() *TokenResponse {
	return xsuaaToken
}

// VerifyXSUAAToken verifies an XSUAA JWT token using the provider's public key.
// This performs cryptographic validation of the token signature using JWK (JSON Web Key).
//
// The function:
//  1. Fetches the JWK Set from the XSUAA provider's token_keys endpoint
//  2. Verifies the token signature using the appropriate public key
//  3. Validates standard claims (expiration, issuer, etc.)
//
// Parameters:
//   - token: The raw JWT token string to verify
//   - xsuaaURL: The XSUAA service base URL (e.g., from credentials.URL)
//
// Returns:
//   - map[string]interface{}: The validated token claims
//   - error: Any error during verification
//
// Example:
//
//	creds, _ := GetXSUAACredentials()
//	claims, err := VerifyXSUAAToken(tokenString, creds.URL)
//	if err != nil {
//	    log.Printf("Invalid token: %v", err)
//	    return
//	}
//	fmt.Printf("User: %s\n", claims["user_name"])
func VerifyXSUAAToken(token, xsuaaURL string) (map[string]interface{}, error) {
	// The jwx library handles JWK fetching and verification automatically
	// when using jwt.ParseString with jwt.WithKeySet

	// Construct JWK Set URL from XSUAA base URL
	jwkSetURL := xsuaaURL + "/token_keys"

	// Note: For production use, you should cache the JWK Set to avoid
	// fetching it on every token verification. The jwx library provides
	// jwk.AutoRefresh for this purpose.

	// Parse and verify the token
	// This will:
	// 1. Fetch the JWK Set from the URL
	// 2. Find the appropriate key (matching the token's "kid" header)
	// 3. Verify the signature
	// 4. Validate expiration and other standard claims
	_, err := parseTokenWithJWKSet(token, jwkSetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}

	// When parseTokenWithJWKSet is fully implemented, it would return jwt.Token
	// which has Claims() method to extract claims
	return nil, fmt.Errorf("XSUAA JWK verification not yet fully implemented")
}

// parseTokenWithJWKSet parses and verifies a JWT using a JWK Set from a URL.
// This is a helper function that uses the lestrrat-go/jwx library.
func parseTokenWithJWKSet(tokenString, jwkSetURL string) (interface{}, error) {
	// Import jwx packages for JWK handling
	// Note: This requires github.com/lestrrat-go/jwx/v2

	// For now, return an error indicating this needs the jwx library
	// The actual implementation would use:
	// - jwk.Fetch() to get the JWK Set
	// - jwt.ParseString() with jwt.WithKeySet() to verify

	return nil, fmt.Errorf("XSUAA public key verification requires JWK support - use VerifyXSUAATokenWithKey instead")
}

// VerifyXSUAATokenWithKey verifies an XSUAA JWT token using a provided public key.
// Use this when you have the public key directly (e.g., from XSUAA_VERIFICATION_KEY env var).
//
// The public key can be:
//   - PEM-encoded RSA public key
//   - PEM-encoded ECDSA public key
//   - Base64-encoded public key
//
// Parameters:
//   - token: The raw JWT token string to verify
//   - publicKey: The public key in PEM or base64 format
//
// Returns:
//   - map[string]interface{}: The validated token claims
//   - error: Any error during verification
//
// Example:
//
//	publicKey := os.Getenv("XSUAA_VERIFICATION_KEY")
//	claims, err := VerifyXSUAATokenWithKey(tokenString, publicKey)
//	if err != nil {
//	    log.Printf("Invalid token: %v", err)
//	    return
//	}
//	fmt.Printf("Scopes: %v\n", claims["scope"])
func VerifyXSUAATokenWithKey(token, publicKey string) (map[string]interface{}, error) {
	// Parse the public key
	// This would use crypto/x509 and crypto/rsa or crypto/ecdsa
	// depending on the key type

	// For now, return an error indicating this needs implementation
	return nil, fmt.Errorf("XSUAA public key verification requires crypto implementation")
}

// ExtractScopesFromXSUAA extracts scopes from XSUAA token claims.
// XSUAA tokens store scopes in the "scope" claim as an array of strings.
//
// Parameters:
//   - claims: The token claims map
//
// Returns:
//   - []string: List of scopes, or nil if not found
//
// Example:
//
//	claims, _ := VerifyXSUAAToken(token, xsuaaURL)
//	scopes := ExtractScopesFromXSUAA(claims)
//	for _, scope := range scopes {
//	    fmt.Println("Scope:", scope)
//	}
func ExtractScopesFromXSUAA(claims map[string]interface{}) []string {
	// XSUAA stores scopes in "scope" claim as array
	if scopeClaim, ok := claims["scope"]; ok {
		// Handle array of strings
		if scopeArray, ok := scopeClaim.([]interface{}); ok {
			scopes := make([]string, 0, len(scopeArray))
			for _, s := range scopeArray {
				if str, ok := s.(string); ok {
					scopes = append(scopes, str)
				}
			}
			return scopes
		}

		// Handle single string (less common)
		if scopeStr, ok := scopeClaim.(string); ok {
			return []string{scopeStr}
		}
	}

	return nil
}

// HasXSUAAScope checks if the claims contain a specific scope.
// Scopes in XSUAA typically have the format "appname.scopename".
//
// Parameters:
//   - claims: The token claims map
//   - scope: The scope to check for
//
// Returns:
//   - bool: true if the scope is present, false otherwise
//
// Example:
//
//	claims, _ := VerifyXSUAAToken(token, xsuaaURL)
//	if HasXSUAAScope(claims, "myapp.read") {
//	    fmt.Println("User has read permission")
//	}
func HasXSUAAScope(claims map[string]interface{}, scope string) bool {
	scopes := ExtractScopesFromXSUAA(claims)
	for _, s := range scopes {
		if s == scope {
			return true
		}
	}
	return false
}
