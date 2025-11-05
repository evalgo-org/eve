package auth

import "time"

// Config represents authentication service configuration
type Config struct {
	// JWT settings
	JWTSecret              string
	JWTExpiration          time.Duration
	RefreshTokenEnabled    bool
	RefreshTokenExpiration time.Duration

	// Password policy
	PasswordMinLength     int
	PasswordRequireStrong bool // uppercase, lowercase, number, special char

	// Account locking
	MaxFailedAttempts int
	LockoutDuration   time.Duration

	// Session management
	SessionTimeout time.Duration
	CookieSecure   bool
	CookieHTTPOnly bool
	CookieSameSite string

	// Roles
	DefaultRole    string
	AvailableRoles []string

	// Audit logging
	AuditEnabled       bool
	AuditRetentionDays int
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		JWTExpiration:          24 * time.Hour,
		RefreshTokenEnabled:    true,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		PasswordMinLength:      8,
		PasswordRequireStrong:  false,
		MaxFailedAttempts:      5,
		LockoutDuration:        30 * time.Minute,
		SessionTimeout:         24 * time.Hour,
		CookieSecure:           true,
		CookieHTTPOnly:         true,
		CookieSameSite:         "Lax",
		DefaultRole:            RoleUser,
		AvailableRoles:         []string{RoleAdmin, RoleUser, RoleViewer, RoleAgent},
		AuditEnabled:           true,
		AuditRetentionDays:     90,
	}
}
