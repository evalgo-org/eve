package auth

import "time"

// User represents a user account in the system
// Fully semantic with JSON-LD support (@context, @type)
// CouchDB-compatible with _id and _rev fields
type User struct {
	// JSON-LD semantic fields
	Context string `json:"@context,omitempty"` // JSON-LD context (https://schema.org)
	Type    string `json:"@type,omitempty"`    // JSON-LD type (Person)

	// Identity fields
	ID       string `json:"_id,omitempty"`   // UUID (CouchDB _id)
	Rev      string `json:"_rev,omitempty"`  // CouchDB revision
	Username string `json:"username"`        // Unique, 3-50 chars
	Email    string `json:"email,omitempty"` // Optional, unique if provided
	Name     string `json:"name,omitempty"`  // Display name

	// Authentication fields
	PasswordHash string   `json:"password_hash,omitempty"` // bcrypt hash (never sent to client)
	Roles        []string `json:"roles"`                   // Array of role names
	APIKeys      []string `json:"api_keys,omitempty"`      // Hashed API keys (optional)

	// Account status
	Enabled            bool `json:"enabled"`              // Account active/inactive
	Locked             bool `json:"locked"`               // Account locked due to failed attempts
	MustChangePassword bool `json:"must_change_password"` // Force password change
	FailedLogins       int  `json:"failed_logins"`        // Failed login counter

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`

	// Extensible metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// UserResponse represents a user with sensitive fields removed
type UserResponse struct {
	ID          string                 `json:"id"`
	Username    string                 `json:"username"`
	Email       string                 `json:"email,omitempty"`
	Roles       []string               `json:"roles"`
	Enabled     bool                   `json:"enabled"`
	Locked      bool                   `json:"locked"`
	Name        string                 `json:"name,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	LastLoginAt *time.Time             `json:"last_login_at,omitempty"`
	Context     string                 `json:"@context,omitempty"`
	Type        string                 `json:"@type,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ToResponse converts User to UserResponse, removing sensitive fields
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		Roles:       u.Roles,
		Enabled:     u.Enabled,
		Locked:      u.Locked,
		Name:        u.Name,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
		LastLoginAt: u.LastLoginAt,
		Context:     u.Context,
		Type:        u.Type,
		Metadata:    u.Metadata,
	}
}

// CreateUserRequest represents a request to create a new user
type CreateUserRequest struct {
	Username           string   `json:"username"`
	Email              string   `json:"email,omitempty"`
	Password           string   `json:"password"`
	Name               string   `json:"name,omitempty"`
	Roles              []string `json:"roles,omitempty"`
	MustChangePassword bool     `json:"must_change_password,omitempty"`
}

// UpdateUserRequest represents a request to update an existing user
type UpdateUserRequest struct {
	Email              *string   `json:"email,omitempty"`
	Password           *string   `json:"password,omitempty"`
	Name               *string   `json:"name,omitempty"`
	Roles              *[]string `json:"roles,omitempty"`
	Enabled            *bool     `json:"enabled,omitempty"`
	Locked             *bool     `json:"locked,omitempty"`
	MustChangePassword *bool     `json:"must_change_password,omitempty"`
	FailedLogins       *int      `json:"failed_logins,omitempty"`
}

// RefreshToken represents a refresh token for token rotation
// Fully semantic with JSON-LD support
type RefreshToken struct {
	// JSON-LD semantic fields
	Context string `json:"@context,omitempty"` // JSON-LD context
	Type    string `json:"@type,omitempty"`    // JSON-LD type (RefreshToken)

	// Identity fields
	ID     string `json:"_id,omitempty"`  // UUID (CouchDB _id)
	Rev    string `json:"_rev,omitempty"` // CouchDB revision
	UserID string `json:"user_id"`        // Foreign key to User

	// Token fields
	Token      string     `json:"token"` // Hashed refresh token
	ExpiresAt  time.Time  `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	Revoked    bool       `json:"revoked"`
}

// AuditLog represents an audit log entry
// Fully semantic with JSON-LD support
type AuditLog struct {
	// JSON-LD semantic fields
	Context string `json:"@context,omitempty"` // JSON-LD context
	Type    string `json:"@type,omitempty"`    // JSON-LD type (AuditLog)

	// Identity fields
	ID  string `json:"_id,omitempty"`  // UUID or timestamp-based (CouchDB _id)
	Rev string `json:"_rev,omitempty"` // CouchDB revision

	// Audit fields
	Timestamp    time.Time `json:"timestamp"`
	UserID       string    `json:"user_id,omitempty"`
	Username     string    `json:"username,omitempty"`
	Action       string    `json:"action"`             // login, logout, create_user, etc.
	Resource     string    `json:"resource,omitempty"` // user:username, container:id, etc.
	ResourceID   string    `json:"resource_id,omitempty"`
	Method       string    `json:"method,omitempty"` // HTTP method
	Path         string    `json:"path,omitempty"`   // API path
	IPAddress    string    `json:"ip_address,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message,omitempty"`

	// Extensible metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AuditSearchCriteria represents search criteria for audit logs
type AuditSearchCriteria struct {
	UserID    string
	Username  string
	Action    string
	Resource  string
	Success   *bool
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}

// AuthResult represents the result of a successful authentication
type AuthResult struct {
	User         *User     `json:"user"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// TokenPair represents an access token and refresh token pair
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Standard roles
const (
	RoleAdmin  = "admin"
	RoleUser   = "user"
	RoleViewer = "viewer"
	RoleAgent  = "agent"
)

// User helper methods for role checking and authorization

// HasRole checks if the user has a specific role
func (u *User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the user has any of the specified roles
func (u *User) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if u.HasRole(role) {
			return true
		}
	}
	return false
}

// IsAdmin checks if the user has admin role
func (u *User) IsAdmin() bool {
	return u.HasRole(RoleAdmin)
}

// CanWrite checks if the user can write (admin or user role)
func (u *User) CanWrite() bool {
	return u.HasAnyRole(RoleAdmin, RoleUser)
}

// CanRead checks if the user can read (any role except disabled)
func (u *User) CanRead() bool {
	return u.Enabled && len(u.Roles) > 0
}

// IsAgent checks if the user has agent role
func (u *User) IsAgent() bool {
	return u.HasRole(RoleAgent)
}
