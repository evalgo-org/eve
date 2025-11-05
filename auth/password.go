package auth

import (
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptCost is the cost factor for bcrypt hashing
	BcryptCost = 10

	// MinPasswordLength is the minimum password length
	MinPasswordLength = 8
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// ValidatePassword checks if a password matches the hash
func ValidatePassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// CheckPasswordStrength validates password strength
func CheckPasswordStrength(password string, requireStrong bool) error {
	if password == "" {
		return ErrEmptyPassword
	}

	if len(password) < MinPasswordLength {
		return ErrPasswordTooShort
	}

	if !requireStrong {
		return nil
	}

	// Strong password requirements:
	// - At least one uppercase letter
	// - At least one lowercase letter
	// - At least one digit
	// - At least one special character

	var (
		hasUpper   = regexp.MustCompile(`[A-Z]`).MatchString(password)
		hasLower   = regexp.MustCompile(`[a-z]`).MatchString(password)
		hasNumber  = regexp.MustCompile(`[0-9]`).MatchString(password)
		hasSpecial = regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)
	)

	if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
		return ErrWeakPassword
	}

	return nil
}

// ValidateUsername validates username format
func ValidateUsername(username string) error {
	if username == "" {
		return ErrInvalidUsername
	}

	// Username must be 3-50 characters
	if len(username) < 3 || len(username) > 50 {
		return ErrInvalidUsername
	}

	// Only alphanumeric, underscore, and hyphen allowed
	validUsername := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validUsername.MatchString(username) {
		return ErrInvalidUsername
	}

	return nil
}

// ValidateEmail validates email format (basic validation)
func ValidateEmail(email string) error {
	if email == "" {
		return nil // Email is optional
	}

	// Basic email validation
	email = strings.TrimSpace(email)
	validEmail := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !validEmail.MatchString(email) {
		return ErrInvalidEmail
	}

	return nil
}
