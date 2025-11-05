package auth

// UserStore defines the interface for user persistence
type UserStore interface {
	// User CRUD operations
	CreateUser(user *User) error
	GetUser(id string) (*User, error)
	GetUserByUsername(username string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	UpdateUser(user *User) error
	DeleteUser(id string) error
	ListUsers() ([]*User, error)

	// Authentication helpers
	RecordLoginAttempt(username string, success bool) error

	// Refresh token operations
	SaveRefreshToken(token *RefreshToken) error
	GetRefreshToken(id string) (*RefreshToken, error)
	GetRefreshTokensByUserID(userID string) ([]*RefreshToken, error)
	RevokeRefreshToken(id string) error
	DeleteExpiredRefreshTokens() error

	// Audit logging
	SaveAuditLog(log *AuditLog) error
	GetAuditLogs(criteria AuditSearchCriteria) ([]*AuditLog, error)
}

// AuditLogger defines audit logging interface
type AuditLogger interface {
	Log(entry *AuditLog) error
	Query(criteria AuditSearchCriteria) ([]*AuditLog, error)
}
