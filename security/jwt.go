/*
Package security provides cryptographic and secret-management utilities.

This file implements a lightweight JSON Web Token (JWT) service
for creating and validating tokens using the HMAC SHA-256 algorithm
(HS256) via the `lestrrat-go/jwx` library.

The JWTService type allows secure token generation and validation
for user authentication or session management in Go applications.

Usage Example:

	package main

	import (
		"fmt"
		"time"
		"myapp/security"
	)

	func main() {
		jwtService := security.NewJWTService("supersecretkey")

		// Generate a token valid for 1 hour
		tokenStr, err := jwtService.GenerateToken("user123", time.Hour)
		if err != nil {
			panic(err)
		}
		fmt.Println("Generated token:", tokenStr)

		// Validate the token
		token, err := jwtService.ValidateToken(tokenStr)
		if err != nil {
			panic(err)
		}
		fmt.Println("Token subject:", token.Subject())
	}
*/

package security

import (
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// JWTService provides methods for generating and validating JSON Web Tokens (JWTs)
// using the HMAC SHA-256 (HS256) signing algorithm.
type JWTService struct {
	secret   []byte
	issuer   string
	audience string
}

// NewJWTService initializes and returns a new JWTService instance.
//
// The secret parameter is the signing key used for both token generation
// and validation. It should be a sufficiently random and securely stored string.
//
// Example:
//
//	j := security.NewJWTService("my-super-secret-key")
func NewJWTService(secret string) *JWTService {
	return &JWTService{
		secret: []byte(secret),
	}
}

// NewJWTServiceWithIssuer creates a JWT service with issuer and audience validation.
// This provides enhanced security by validating the token's issuer and audience claims.
//
// Parameters:
//   - secret: The signing key for HMAC SHA-256
//   - issuer: The expected issuer claim (iss) - typically your application's identifier
//   - audience: The expected audience claim (aud) - typically your API's identifier
//
// Example:
//
//	j := security.NewJWTServiceWithIssuer(
//	    "my-super-secret-key",
//	    "https://myapp.example.com",
//	    "https://api.example.com",
//	)
func NewJWTServiceWithIssuer(secret, issuer, audience string) *JWTService {
	return &JWTService{
		secret:   []byte(secret),
		issuer:   issuer,
		audience: audience,
	}
}

// GenerateToken creates a new signed JWT containing the specified user ID as the subject.
//
// Parameters:
//   - userID: The unique identifier of the user (stored as the "sub" claim).
//   - expiration: Token validity duration (e.g. 1 * time.Hour).
//
// The generated token includes the following standard claims:
//   - "sub": The subject (user ID)
//   - "iat": Issued-at timestamp
//   - "exp": Expiration timestamp
//   - "iss": Issuer (if configured)
//   - "aud": Audience (if configured)
//
// Returns:
//   - The signed JWT string.
//   - An error if token building or signing fails.
//
// Example:
//
//	token, err := jwtService.GenerateToken("user123", time.Hour)
func (j *JWTService) GenerateToken(userID string, expiration time.Duration) (string, error) {
	now := time.Now()

	builder := jwt.NewBuilder().
		Subject(userID).
		IssuedAt(now).
		Expiration(now.Add(expiration))

	// Add issuer and audience if configured
	if j.issuer != "" {
		builder = builder.Issuer(j.issuer)
	}
	if j.audience != "" {
		builder = builder.Audience([]string{j.audience})
	}

	token, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build token: %w", err)
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256, j.secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return string(signed), nil
}

// GenerateTokenWithClaims creates a signed JWT with custom claims in addition to standard claims.
//
// Parameters:
//   - userID: The unique identifier of the user (stored as the "sub" claim)
//   - expiration: Token validity duration
//   - customClaims: Additional custom claims to include in the token
//
// Returns:
//   - The signed JWT string
//   - An error if token building or signing fails
//
// Example:
//
//	claims := map[string]interface{}{
//	    "role":  "admin",
//	    "scope": "read write delete",
//	    "email": "admin@example.com",
//	}
//	token, err := jwtService.GenerateTokenWithClaims("user123", time.Hour, claims)
func (j *JWTService) GenerateTokenWithClaims(userID string, expiration time.Duration, customClaims map[string]interface{}) (string, error) {
	now := time.Now()

	builder := jwt.NewBuilder().
		Subject(userID).
		IssuedAt(now).
		Expiration(now.Add(expiration))

	// Add issuer and audience if configured
	if j.issuer != "" {
		builder = builder.Issuer(j.issuer)
	}
	if j.audience != "" {
		builder = builder.Audience([]string{j.audience})
	}

	// Add custom claims
	for key, value := range customClaims {
		builder = builder.Claim(key, value)
	}

	token, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build token: %w", err)
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256, j.secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return string(signed), nil
}

// ValidateToken verifies and parses a JWT string using the configured secret key.
//
// The token's signature and expiration are validated automatically.
// If issuer and audience are configured, they are also validated.
// If validation succeeds, it returns a `jwt.Token` instance that allows
// access to claims such as subject, expiration, and issued-at time.
//
// Parameters:
//   - tokenString: The signed JWT string to validate.
//
// Returns:
//   - jwt.Token: The parsed and validated token.
//   - error: Non-nil if the token is invalid, expired, or improperly signed.
//
// Example:
//
//	token, err := jwtService.ValidateToken(tokenStr)
//	if err != nil {
//		log.Println("Invalid token:", err)
//	} else {
//		fmt.Println("User:", token.Subject())
//	}
func (j *JWTService) ValidateToken(tokenString string) (jwt.Token, error) {
	parseOptions := []jwt.ParseOption{
		jwt.WithKey(jwa.HS256, j.secret),
	}

	// Add issuer validation if configured
	if j.issuer != "" {
		parseOptions = append(parseOptions, jwt.WithIssuer(j.issuer))
	}

	// Add audience validation if configured
	if j.audience != "" {
		parseOptions = append(parseOptions, jwt.WithAudience(j.audience))
	}

	token, err := jwt.Parse([]byte(tokenString), parseOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	return token, nil
}

// ValidateTokenWithOptions validates a JWT with custom validation options.
// This provides fine-grained control over token validation.
//
// Parameters:
//   - tokenString: The signed JWT string to validate
//   - options: Custom validation options (issuer, audience, clock skew, etc.)
//
// Returns:
//   - jwt.Token: The parsed and validated token
//   - error: Non-nil if validation fails
//
// Example:
//
//	token, err := jwtService.ValidateTokenWithOptions(tokenStr,
//	    jwt.WithIssuer("https://myapp.example.com"),
//	    jwt.WithAudience("https://api.example.com"),
//	    jwt.WithAcceptableSkew(30*time.Second),
//	)
func (j *JWTService) ValidateTokenWithOptions(tokenString string, options ...jwt.ParseOption) (jwt.Token, error) {
	// Always include the signing key
	allOptions := []jwt.ParseOption{jwt.WithKey(jwa.HS256, j.secret)}
	allOptions = append(allOptions, options...)

	token, err := jwt.Parse([]byte(tokenString), allOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	return token, nil
}
