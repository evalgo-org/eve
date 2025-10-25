package security_test

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"eve.evalgo.org/security"
	"github.com/labstack/echo/v4"
)

// Example demonstrates how to set up an Echo web server with SAP BTP XSUAA authentication.
// This example shows the complete flow of:
//  1. Loading XSUAA credentials from Cloud Foundry environment
//  2. Obtaining an OAuth 2.0 access token using client credentials
//  3. Setting up an Echo server with API key authentication middleware
//  4. Creating protected and public endpoints
func Example() {
	// Get XSUAA credentials from VCAP_SERVICES environment variable
	// In Cloud Foundry, this is automatically provided when bound to an XSUAA service
	creds, err := security.GetXSUAACredentials()
	if err != nil {
		log.Fatalf("Failed to get XSUAA credentials: %v", err)
	}

	// Construct the OAuth token endpoint URL
	tokenURL := creds.URL + "/oauth/token"

	// Obtain access token using client credentials flow
	token, err := security.GetXSUAAToken(tokenURL, creds.ClientID, creds.ClientSecret)
	if err != nil {
		log.Fatalf("Failed to get XSUAA token: %v", err)
	}

	// Cache the token globally for use in handlers
	security.SetGlobalXSUAAToken(token)

	// Create Echo instance
	e := echo.New()

	// Public health check endpoint (no authentication required)
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// Create API group with authentication middleware
	api := e.Group("/api")
	api.Use(security.APIKeyAuth())

	// Protected endpoint - requires X-API-Key header
	api.GET("/response", security.HandleWithClientCredentials)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
	fmt.Printf("Starting server on port %s\n", port)
	if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// ExampleGetXSUAAToken demonstrates how to obtain an XSUAA access token
// using client credentials flow.
func ExampleGetXSUAAToken() {
	tokenURL := "https://tenant.authentication.eu10.hana.ondemand.com/oauth/token"
	clientID := "your-client-id"
	clientSecret := "your-client-secret"

	token, err := security.GetXSUAAToken(tokenURL, clientID, clientSecret)
	if err != nil {
		log.Fatalf("Failed to get token: %v", err)
	}

	fmt.Printf("Token type: %s\n", token.TokenType)
	fmt.Printf("Expires in: %d seconds\n", token.ExpiresIn)
	// Output will show token details
}

// ExampleGetXSUAACredentials demonstrates how to extract XSUAA credentials
// from Cloud Foundry's VCAP_SERVICES environment variable.
func ExampleGetXSUAACredentials() {
	// In Cloud Foundry, VCAP_SERVICES is automatically set
	// Here's an example of what it might contain:
	vcapExample := `{
		"xsuaa": [{
			"credentials": {
				"url": "https://tenant.authentication.eu10.hana.ondemand.com",
				"clientid": "sb-myapp!t123",
				"clientsecret": "secret123",
				"xsappname": "myapp!t123"
			}
		}]
	}`

	// Temporarily set for demonstration
	os.Setenv("VCAP_SERVICES", vcapExample)
	defer os.Unsetenv("VCAP_SERVICES")

	creds, err := security.GetXSUAACredentials()
	if err != nil {
		log.Fatalf("Failed to get credentials: %v", err)
	}

	fmt.Printf("XSUAA URL: %s\n", creds.URL)
	fmt.Printf("Client ID: %s\n", creds.ClientID)
	fmt.Printf("App Name: %s\n", creds.XSAppName)
	// Output:
	// XSUAA URL: https://tenant.authentication.eu10.hana.ondemand.com
	// Client ID: sb-myapp!t123
	// App Name: myapp!t123
}

// ExampleAPIKeyAuth demonstrates how to use API key authentication middleware
// with Echo framework.
func ExampleAPIKeyAuth() {
	// Set API key in environment
	os.Setenv("API_KEY", "my-secret-key")
	defer os.Unsetenv("API_KEY")

	e := echo.New()

	// Public endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// Protected API group
	api := e.Group("/api")
	api.Use(security.APIKeyAuth())

	// This endpoint requires X-API-Key header
	api.GET("/protected", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Access granted",
		})
	})

	// Start server
	// Clients must send: X-API-Key: my-secret-key
	e.Start(":8080")
}

// ExampleSetGlobalXSUAAToken demonstrates token caching for reuse across requests.
func ExampleSetGlobalXSUAAToken() {
	// Obtain token
	token := &security.TokenResponse{
		AccessToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
		TokenType:   "bearer",
		ExpiresIn:   3600,
	}

	// Cache token globally
	security.SetGlobalXSUAAToken(token)

	// Later, retrieve the token
	cachedToken := security.GetGlobalXSUAAToken()
	if cachedToken != nil {
		fmt.Printf("Token expires in: %d seconds\n", cachedToken.ExpiresIn)
	}
	// Output:
	// Token expires in: 3600 seconds
}
