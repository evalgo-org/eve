// Package security provides cryptographic and authentication utilities.
// This file implements password hashing and verification using bcrypt algorithm.
package security

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultBcryptCost is the default cost factor for bcrypt password hashing.
	// Cost factor of 10 provides a good balance between security and performance.
	// Higher values increase security but also increase hashing time exponentially.
	DefaultBcryptCost = 10
)

// HashPassword creates a bcrypt hash of the provided password.
// Uses the default cost factor (10) which provides good security for most use cases.
//
// Bcrypt is a password hashing function designed to be slow and computationally expensive,
// making it resistant to brute-force attacks. Each hash includes a random salt automatically.
//
// Parameters:
//   - password: The plaintext password to hash
//
// Returns:
//   - string: The bcrypt hash string (includes algorithm, cost, salt, and hash)
//   - error: Any error encountered during hashing
//
// Example:
//
//	hash, err := HashPassword("mySecurePassword123")
//	if err != nil {
//	    log.Fatalf("Failed to hash password: %v", err)
//	}
//	// Store hash in database: $2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), DefaultBcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// HashPasswordWithCost creates a bcrypt hash of the provided password with a custom cost factor.
// Use this when you need fine-grained control over the hashing cost.
//
// Cost factor recommendations:
//   - 10: Default, suitable for most applications
//   - 12: Higher security, slower hashing (~250ms)
//   - 14: Very high security, much slower (~1 second)
//   - 4-9: Faster but less secure (not recommended for production)
//
// Parameters:
//   - password: The plaintext password to hash
//   - cost: The bcrypt cost factor (must be between bcrypt.MinCost and bcrypt.MaxCost)
//
// Returns:
//   - string: The bcrypt hash string
//   - error: Any error encountered during hashing
//
// Example:
//
//	// Use higher cost for admin passwords
//	hash, err := HashPasswordWithCost("adminPassword", 12)
//	if err != nil {
//	    log.Fatalf("Failed to hash password: %v", err)
//	}
func HashPasswordWithCost(password string, cost int) (string, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return "", fmt.Errorf("invalid cost factor %d: must be between %d and %d", cost, bcrypt.MinCost, bcrypt.MaxCost)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword compares a plaintext password with a bcrypt hash.
// Returns nil if the password matches the hash, or an error if they don't match
// or if verification fails.
//
// This function is constant-time to prevent timing attacks.
//
// Parameters:
//   - hash: The bcrypt hash to verify against
//   - password: The plaintext password to check
//
// Returns:
//   - error: nil if password matches, bcrypt.ErrMismatchedHashAndPassword if mismatch,
//     or other error if verification fails
//
// Example:
//
//	// During login
//	err := VerifyPassword(storedHash, userProvidedPassword)
//	if err != nil {
//	    if err == bcrypt.ErrMismatchedHashAndPassword {
//	        log.Println("Invalid password")
//	    } else {
//	        log.Printf("Verification error: %v", err)
//	    }
//	    return
//	}
//	log.Println("Password correct, user authenticated")
func VerifyPassword(hash, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return err // Return bcrypt.ErrMismatchedHashAndPassword or other error
	}
	return nil
}

// NeedsRehash checks if a password hash needs to be regenerated with a different cost factor.
// This is useful when you increase the cost factor in your application and want to
// upgrade old hashes during user login.
//
// Parameters:
//   - hash: The bcrypt hash to check
//   - cost: The desired cost factor
//
// Returns:
//   - bool: true if the hash needs to be regenerated with the new cost
//   - error: Any error encountered while parsing the hash
//
// Example:
//
//	const CurrentCost = 12
//
//	// During login after successful password verification
//	needsRehash, err := NeedsRehash(storedHash, CurrentCost)
//	if err == nil && needsRehash {
//	    newHash, _ := HashPasswordWithCost(password, CurrentCost)
//	    // Update hash in database
//	}
func NeedsRehash(hash string, cost int) (bool, error) {
	actualCost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return false, fmt.Errorf("failed to get hash cost: %w", err)
	}
	return actualCost != cost, nil
}
