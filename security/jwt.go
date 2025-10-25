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
	secret []byte
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

	token, err := jwt.NewBuilder().
		Subject(userID).
		IssuedAt(now).
		Expiration(now.Add(expiration)).
		Build()
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
// The token’s signature and expiration are validated automatically.
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
	token, err := jwt.Parse([]byte(tokenString), jwt.WithKey(jwa.HS256, j.secret))
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	return token, nil
}
