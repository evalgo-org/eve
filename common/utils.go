// Package common provides common utilities for EVE services
package common

import (
	"fmt"
	"os"
	"strconv"
)

// MaskSecret masks sensitive strings for safe logging
// Shows first 4 and last 4 characters for strings longer than 8 chars
// Returns "***" for short strings and "<not set>" for empty strings
//
// Example:
//
//	MaskSecret("") // "<not set>"
//	MaskSecret("short") // "***"
//	MaskSecret("myverylongsecretkey123") // "myve...y123"
func MaskSecret(secret string) string {
	if secret == "" {
		return "<not set>"
	}
	if len(secret) <= 8 {
		return "***"
	}
	return secret[:4] + "..." + secret[len(secret)-4:]
}

// GetEnv retrieves an environment variable with a fallback default value
func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetEnvInt retrieves an integer environment variable with a fallback default
func GetEnvInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// GetEnvBool retrieves a boolean environment variable with a fallback default
// Accepts: "true", "1", "yes", "on" for true
// Accepts: "false", "0", "no", "off" for false
func GetEnvBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	switch valueStr {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultValue
	}
}

// Must panics if err is not nil, otherwise returns value
// Useful for initialization code that should fail fast
//
// Example:
//
//	config := common.Must(loadConfig())
func Must[T any](value T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("Must: operation failed: %v", err))
	}
	return value
}

// MustNoError panics if err is not nil
// Useful for initialization code that should fail fast
//
// Example:
//
//	common.MustNoError(db.Init())
func MustNoError(err error) {
	if err != nil {
		panic(fmt.Sprintf("MustNoError: operation failed: %v", err))
	}
}

// Ptr returns a pointer to the given value
// Useful for initializing pointer fields in structs
//
// Example:
//
//	config := Config{
//	    Enabled: common.Ptr(true),
//	    Count:   common.Ptr(42),
//	}
func Ptr[T any](v T) *T {
	return &v
}

// PtrValue returns the value of a pointer, or the zero value if nil
//
// Example:
//
//	value := common.PtrValue(config.Timeout) // Returns 0 if Timeout is nil
func PtrValue[T any](ptr *T) T {
	if ptr == nil {
		var zero T
		return zero
	}
	return *ptr
}
