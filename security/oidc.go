// Package security provides authentication utilities including OpenID Connect (OIDC) integration.
// This file implements OIDC provider discovery and ID token verification for authentication
// with external identity providers like Auth0, Keycloak, Azure AD, Google, and others.
package security

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDCProvider wraps an OpenID Connect provider with token verification capabilities.
// It handles provider discovery, token verification, and claims extraction.
type OIDCProvider struct {
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	config   OIDCConfig
}

// OIDCConfig contains the configuration for an OIDC provider.
type OIDCConfig struct {
	// ProviderURL is the OIDC provider's discovery URL (e.g., "https://accounts.google.com")
	ProviderURL string

	// ClientID is your application's client ID registered with the OIDC provider
	ClientID string

	// ClientSecret is your application's client secret (required for some flows)
	ClientSecret string

	// RedirectURL is the callback URL for OAuth2 authorization code flow
	RedirectURL string

	// Scopes are the OAuth2 scopes to request (default: ["openid", "profile", "email"])
	Scopes []string

	// SkipIssuerCheck disables issuer validation (not recommended for production)
	SkipIssuerCheck bool

	// SkipExpiryCheck disables expiration time validation (not recommended for production)
	SkipExpiryCheck bool
}

// Claims represents the standard OIDC claims extracted from an ID token.
type Claims struct {
	// Subject is the unique identifier for the user (required)
	Subject string `json:"sub"`

	// Email is the user's email address
	Email string `json:"email,omitempty"`

	// EmailVerified indicates if the email has been verified by the provider
	EmailVerified bool `json:"email_verified,omitempty"`

	// Name is the user's full name
	Name string `json:"name,omitempty"`

	// GivenName is the user's first name
	GivenName string `json:"given_name,omitempty"`

	// FamilyName is the user's last name
	FamilyName string `json:"family_name,omitempty"`

	// Picture is the URL of the user's profile picture
	Picture string `json:"picture,omitempty"`

	// Locale is the user's preferred locale
	Locale string `json:"locale,omitempty"`

	// Issuer is the token issuer URL
	Issuer string `json:"iss,omitempty"`

	// Audience is the intended audience for the token
	Audience string `json:"aud,omitempty"`

	// ExpiresAt is the token expiration time (Unix timestamp)
	ExpiresAt int64 `json:"exp,omitempty"`

	// IssuedAt is the token issuance time (Unix timestamp)
	IssuedAt int64 `json:"iat,omitempty"`

	// Custom claims as a map for provider-specific claims
	Extra map[string]interface{} `json:"-"`
}

// NewOIDCProvider creates a new OIDC provider instance with automatic discovery.
// It contacts the OIDC provider's discovery endpoint to retrieve configuration
// and sets up token verification.
//
// The provider URL should be the issuer URL without the /.well-known/openid-configuration path.
// For example:
//   - Google: "https://accounts.google.com"
//   - Auth0: "https://YOUR_DOMAIN.auth0.com"
//   - Keycloak: "https://keycloak.example.com/realms/YOUR_REALM"
//   - Azure AD: "https://login.microsoftonline.com/YOUR_TENANT_ID/v2.0"
//
// Parameters:
//   - ctx: Context for the HTTP request to the discovery endpoint
//   - config: OIDC provider configuration
//
// Returns:
//   - *OIDCProvider: Initialized OIDC provider
//   - error: Any error during provider discovery or initialization
//
// Example:
//
//	config := OIDCConfig{
//	    ProviderURL:  "https://accounts.google.com",
//	    ClientID:     "your-client-id.apps.googleusercontent.com",
//	    ClientSecret: "your-client-secret",
//	    Scopes:       []string{"openid", "profile", "email"},
//	}
//
//	provider, err := NewOIDCProvider(context.Background(), config)
//	if err != nil {
//	    log.Fatalf("Failed to initialize OIDC provider: %v", err)
//	}
func NewOIDCProvider(ctx context.Context, config OIDCConfig) (*OIDCProvider, error) {
	if config.ProviderURL == "" {
		return nil, fmt.Errorf("provider URL is required")
	}
	if config.ClientID == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	// Set default scopes if none provided
	if len(config.Scopes) == 0 {
		config.Scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}

	// Discover OIDC provider configuration
	provider, err := oidc.NewProvider(ctx, config.ProviderURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover OIDC provider: %w", err)
	}

	// Create ID token verifier
	verifierConfig := &oidc.Config{
		ClientID:        config.ClientID,
		SkipIssuerCheck: config.SkipIssuerCheck,
		SkipExpiryCheck: config.SkipExpiryCheck,
	}
	verifier := provider.Verifier(verifierConfig)

	return &OIDCProvider{
		provider: provider,
		verifier: verifier,
		config:   config,
	}, nil
}

// VerifyIDToken verifies and parses an OIDC ID token.
// It validates the token signature, expiration, issuer, and audience claims.
//
// The token string should be the raw JWT ID token (not an access token).
//
// Parameters:
//   - ctx: Context for the verification operation
//   - token: The raw JWT ID token string
//
// Returns:
//   - *Claims: Parsed and verified claims from the ID token
//   - error: Any error during verification or parsing
//
// Example:
//
//	claims, err := provider.VerifyIDToken(context.Background(), rawIDToken)
//	if err != nil {
//	    log.Printf("Invalid token: %v", err)
//	    return
//	}
//	fmt.Printf("Authenticated user: %s (%s)\n", claims.Name, claims.Email)
func (p *OIDCProvider) VerifyIDToken(ctx context.Context, token string) (*Claims, error) {
	idToken, err := p.verifier.Verify(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	var claims Claims
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse token claims: %w", err)
	}

	// Extract custom claims
	var allClaims map[string]interface{}
	if err := idToken.Claims(&allClaims); err == nil {
		claims.Extra = allClaims
	}

	return &claims, nil
}

// OAuth2Config returns an OAuth2 configuration for the authorization code flow.
// Use this to redirect users to the provider's login page and exchange authorization
// codes for tokens.
//
// Returns:
//   - *oauth2.Config: OAuth2 configuration ready for authorization code flow
//
// Example:
//
//	oauth2Config := provider.OAuth2Config()
//
//	// Redirect user to login
//	authURL := oauth2Config.AuthCodeURL("state-string")
//	http.Redirect(w, r, authURL, http.StatusFound)
//
//	// In callback handler
//	code := r.URL.Query().Get("code")
//	token, err := oauth2Config.Exchange(context.Background(), code)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Verify ID token
//	rawIDToken, ok := token.Extra("id_token").(string)
//	if !ok {
//	    log.Fatal("No id_token in response")
//	}
//	claims, err := provider.VerifyIDToken(context.Background(), rawIDToken)
func (p *OIDCProvider) OAuth2Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.config.ClientID,
		ClientSecret: p.config.ClientSecret,
		RedirectURL:  p.config.RedirectURL,
		Endpoint:     p.provider.Endpoint(),
		Scopes:       p.config.Scopes,
	}
}

// GetUserInfo fetches additional user information from the provider's UserInfo endpoint.
// This is optional and provides claims not included in the ID token.
//
// Parameters:
//   - ctx: Context for the HTTP request
//   - tokenSource: OAuth2 token source (from oauth2.Config.TokenSource)
//
// Returns:
//   - *oidc.UserInfo: User information from the provider
//   - error: Any error during the UserInfo request
//
// Example:
//
//	token, _ := oauth2Config.Exchange(ctx, code)
//	userInfo, err := provider.GetUserInfo(ctx, oauth2Config.TokenSource(ctx, token))
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	var claims Claims
//	userInfo.Claims(&claims)
//	fmt.Printf("User: %s\n", claims.Email)
func (p *OIDCProvider) GetUserInfo(ctx context.Context, tokenSource oauth2.TokenSource) (*oidc.UserInfo, error) {
	userInfo, err := p.provider.UserInfo(ctx, tokenSource)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	return userInfo, nil
}

// Endpoint returns the OAuth2 endpoint configuration for this provider.
//
// Returns:
//   - oauth2.Endpoint: The provider's authorization and token endpoints
func (p *OIDCProvider) Endpoint() oauth2.Endpoint {
	return p.provider.Endpoint()
}
