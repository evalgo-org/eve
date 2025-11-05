# EVE Auth Package

Unified authentication and user management package for EVE-based projects.

## Overview

The `auth` package provides a comprehensive authentication and authorization solution consolidating duplicate functionality from `graphium` and `pxgraphservice` projects. It offers a flexible, database-agnostic architecture with support for multiple storage backends.

## Features

- **User Management**: Complete CRUD operations for user accounts
- **Password Security**: bcrypt hashing with configurable strength requirements
- **JWT Authentication**: Access and refresh token support
- **Role-Based Access Control**: Flexible role system with multiple roles per user
- **Account Security**: Failed login tracking, account locking, password change enforcement
- **Audit Logging**: Comprehensive audit trail for compliance
- **Database Agnostic**: Interface-based design supports multiple storage backends
- **Session Management**: Cookie-based and header-based authentication

## Installation

```go
import "eve.evalgo.org/auth"
```

## Quick Start

```go
package main

import (
    "eve.evalgo.org/auth"
    "time"
)

func main() {
    // Configure auth service
    config := &auth.Config{
        JWTSecret:             "your-secret-key",
        JWTExpiration:         24 * time.Hour,
        RefreshTokenEnabled:   true,
        PasswordRequireStrong: true,
        MaxFailedAttempts:     5,
        AuditEnabled:          true,
    }

    // Create storage backend (implement auth.UserStore interface)
    store := NewYourStorageImplementation()

    // Create auth service
    service := auth.NewAuthService(config, store)

    // Create a user
    user, err := service.CreateUser(auth.CreateUserRequest{
        Username: "john",
        Email:    "john@example.com",
        Password: "SecurePass123!",
        Roles:    []string{auth.RoleUser},
    })

    // Login
    result, err := service.Login("john", "SecurePass123!")
    if err != nil {
        log.Fatal(err)
    }

    // result.AccessToken contains the JWT
    // result.RefreshToken contains the refresh token (if enabled)
}
```

## Configuration

```go
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

    // Roles
    DefaultRole    string
    AvailableRoles []string

    // Audit logging
    AuditEnabled       bool
    AuditRetentionDays int
}

// Get default configuration
config := auth.DefaultConfig()
```

## User Management

### Create User

```go
user, err := service.CreateUser(auth.CreateUserRequest{
    Username:           "alice",
    Email:              "alice@example.com",
    Password:           "StrongPass456!",
    Name:               "Alice Smith",
    Roles:              []string{auth.RoleUser},
    MustChangePassword: false,
})
```

### Update User

```go
email := "newemail@example.com"
enabled := false

updatedUser, err := service.UpdateUser(userID, auth.UpdateUserRequest{
    Email:   &email,
    Enabled: &enabled,
})
```

### Delete User

```go
err := service.DeleteUser(userID, requestingUserID)
```

### List Users

```go
users, err := service.ListUsers()
for _, user := range users {
    fmt.Printf("%s: %s\n", user.Username, user.Email)
}
```

## Authentication

### Login

```go
result, err := service.Login("alice", "StrongPass456!")
if err != nil {
    switch err {
    case auth.ErrInvalidCredentials:
        // Wrong password or username
    case auth.ErrAccountLocked:
        // Too many failed attempts
    case auth.ErrAccountDisabled:
        // Account disabled
    }
}

// Use result.AccessToken for authenticated requests
```

### Token Validation

```go
claims, err := service.ValidateToken(accessToken)
if err != nil {
    switch err {
    case auth.ErrExpiredToken:
        // Token expired, request refresh
    case auth.ErrInvalidToken:
        // Invalid or malformed token
    }
}

// claims.UserID, claims.Username, claims.Roles available
```

### Change Password

```go
err := service.ChangePassword(userID, "OldPass123!", "NewPass456!")
```

## Authorization

### Check Role

```go
isAdmin, err := service.HasRole(userID, auth.RoleAdmin)
if !isAdmin {
    return errors.New("admin access required")
}
```

### Check Multiple Roles

```go
hasAccess, err := service.HasAnyRole(userID, []string{
    auth.RoleAdmin,
    auth.RoleUser,
})
```

## Standard Roles

```go
const (
    RoleAdmin  = "admin"   // Full system access
    RoleUser   = "user"    // Standard user access
    RoleViewer = "viewer"  // Read-only access
    RoleAgent  = "agent"   // Automated agent access
)
```

## Storage Interface

Implement the `UserStore` interface to support your database:

```go
type UserStore interface {
    // User CRUD
    CreateUser(user *User) error
    GetUser(id string) (*User, error)
    GetUserByUsername(username string) (*User, error)
    GetUserByEmail(email string) (*User, error)
    UpdateUser(user *User) error
    DeleteUser(id string) error
    ListUsers() ([]*User, error)

    // Authentication
    RecordLoginAttempt(username string, success bool) error

    // Refresh tokens
    SaveRefreshToken(token *RefreshToken) error
    GetRefreshToken(id string) (*RefreshToken, error)
    GetRefreshTokensByUserID(userID string) ([]*RefreshToken, error)
    RevokeRefreshToken(id string) error
    DeleteExpiredRefreshTokens() error

    // Audit logging
    SaveAuditLog(log *AuditLog) error
    GetAuditLogs(criteria AuditSearchCriteria) ([]*AuditLog, error)
}
```

## User Model

```go
type User struct {
    // Identity
    ID       string   `json:"id"`
    Username string   `json:"username"`
    Email    string   `json:"email,omitempty"`

    // Authentication
    PasswordHash string   `json:"password_hash"`
    Roles        []string `json:"roles"`

    // Account status
    Enabled            bool `json:"enabled"`
    Locked             bool `json:"locked"`
    MustChangePassword bool `json:"must_change_password"`
    FailedLogins       int  `json:"failed_logins"`

    // Metadata
    Name        string     `json:"name,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
    LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}
```

## Security Features

### Password Validation

```go
// Basic validation (minimum 8 characters)
err := auth.CheckPasswordStrength("password", false)

// Strong validation (uppercase, lowercase, number, special char)
err := auth.CheckPasswordStrength("WeakPass", true)
// Returns: ErrWeakPassword
```

### Account Locking

Configure in `Config`:
```go
config.MaxFailedAttempts = 5  // Lock after 5 failed logins
config.LockoutDuration = 30 * time.Minute
```

Failed logins are automatically tracked. Unlock manually:
```go
locked := false
service.UpdateUser(userID, auth.UpdateUserRequest{
    Locked:       &locked,
    FailedLogins: new(int), // Reset to 0
})
```

### Force Password Change

```go
mustChange := true
service.UpdateUser(userID, auth.UpdateUserRequest{
    MustChangePassword: &mustChange,
})
```

## Audit Logging

All authentication events are logged:
- Login attempts (success/failure)
- Logout
- Password changes
- User CRUD operations
- Token validation failures

Query audit logs:
```go
criteria := auth.AuditSearchCriteria{
    UserID:   "user-id",
    Action:   "login",
    Success:  &trueVal,
    StartTime: &startTime,
    EndTime:   &endTime,
    Limit:    100,
}

logs, err := store.GetAuditLogs(criteria)
```

## Migration from Existing Code

### From graphium

1. Replace `internal/auth` with `eve.evalgo.org/auth`
2. Implement `auth.UserStore` for CouchDB backend
3. Update handlers to use `AuthService` instead of direct storage calls
4. Keep JSON-LD fields (@context, @type, _rev) in User metadata

### From pxgraphservice

1. Replace `auth` package with `eve.evalgo.org/auth`
2. Implement `auth.UserStore` for file-based backend
3. Update handlers to use `AuthService`
4. Migrate single role string to roles array

## Projects Using This Package

This auth package consolidates authentication from:
- **graphium**: CouchDB-backed JSON-LD user management
- **pxgraphservice**: File-based user management with HTMX UI

Benefits:
- **Single source of truth** for authentication logic
- **11 duplicate functions** eliminated
- **Consistent security** across all projects
- **Easier maintenance** and bug fixes

## Advanced Features

### Refresh Token Rotation

```go
// Enable refresh tokens
config.RefreshTokenEnabled = true
config.RefreshTokenExpiration = 7 * 24 * time.Hour

// Login returns both tokens
result, _ := service.Login("user", "pass")
// result.AccessToken - short-lived
// result.RefreshToken - long-lived

// Refresh access token
newPair, _ := service.RefreshToken(result.RefreshToken)
```

### Custom Validation

```go
// Username validation (3-50 chars, alphanumeric + _ -)
err := auth.ValidateUsername("user_name-123")

// Email validation
err := auth.ValidateEmail("user@example.com")

// Password strength
err := auth.CheckPasswordStrength("MyPass123!", true)
```

## Error Handling

```go
var (
    ErrInvalidCredentials = errors.New("invalid username or password")
    ErrAccountLocked      = errors.New("account is locked")
    ErrAccountDisabled    = errors.New("account is disabled")
    ErrExpiredToken       = errors.New("token has expired")
    ErrInvalidToken       = errors.New("invalid token")
    ErrUserNotFound       = errors.New("user not found")
    ErrUserExists         = errors.New("user already exists")
    ErrWeakPassword       = errors.New("password does not meet requirements")
    ErrSelfDelete         = errors.New("cannot delete your own account")
)
```

## Testing

Mock storage for testing:
```go
type MockStore struct {
    users map[string]*auth.User
}

func (m *MockStore) CreateUser(user *auth.User) error {
    m.users[user.ID] = user
    return nil
}
// ... implement other methods
```

## License

Part of the EVE library - see main LICENSE file.

## Version

v0.0.21 - Initial release with consolidated auth functionality
