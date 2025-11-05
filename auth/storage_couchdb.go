package auth

import (
	"fmt"
	"time"

	"eve.evalgo.org/db"
)

// CouchDBUserStore implements UserStore for CouchDB with JSON-LD support
type CouchDBUserStore struct {
	service *db.CouchDBService
}

// NewCouchDBUserStore creates a new CouchDB-backed user store
func NewCouchDBUserStore(service *db.CouchDBService) UserStore {
	return &CouchDBUserStore{
		service: service,
	}
}

// CreateUser creates a new user in CouchDB
func (s *CouchDBUserStore) CreateUser(user *User) error {
	// Set JSON-LD semantic fields
	if user.Context == "" {
		user.Context = "https://schema.org"
	}
	if user.Type == "" {
		user.Type = "Person"
	}

	// Set timestamps
	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now

	// Save to CouchDB
	resp, err := s.service.SaveGenericDocument(user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Update user with CouchDB _id and _rev
	user.ID = resp.ID
	user.Rev = resp.Rev

	return nil
}

// GetUser retrieves a user by ID
func (s *CouchDBUserStore) GetUser(id string) (*User, error) {
	var user User
	if err := s.service.GetGenericDocument(id, &user); err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetUserByUsername retrieves a user by username using semantic query
func (s *CouchDBUserStore) GetUserByUsername(username string) (*User, error) {
	query := db.NewQueryBuilder().
		Where("@type", "$eq", "Person").
		And().
		Where("username", "$eq", username).
		Limit(1).
		Build()

	users, err := db.FindTyped[User](s.service, query)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	if len(users) == 0 {
		return nil, ErrUserNotFound
	}

	return &users[0], nil
}

// GetUserByEmail retrieves a user by email using semantic query
func (s *CouchDBUserStore) GetUserByEmail(email string) (*User, error) {
	query := db.NewQueryBuilder().
		Where("@type", "$eq", "Person").
		And().
		Where("email", "$eq", email).
		Limit(1).
		Build()

	users, err := db.FindTyped[User](s.service, query)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	if len(users) == 0 {
		return nil, ErrUserNotFound
	}

	return &users[0], nil
}

// UpdateUser updates an existing user in CouchDB
func (s *CouchDBUserStore) UpdateUser(user *User) error {
	// Ensure semantic fields are set
	if user.Context == "" {
		user.Context = "https://schema.org"
	}
	if user.Type == "" {
		user.Type = "Person"
	}

	// Update timestamp
	user.UpdatedAt = time.Now()

	// Save to CouchDB (uses _rev for optimistic locking)
	resp, err := s.service.SaveGenericDocument(user)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Update _rev
	user.Rev = resp.Rev

	return nil
}

// DeleteUser deletes a user from CouchDB
func (s *CouchDBUserStore) DeleteUser(id string) error {
	// Get current user to get _rev
	user, err := s.GetUser(id)
	if err != nil {
		return err
	}

	return s.service.DeleteDocument(id, user.Rev)
}

// ListUsers retrieves all users using semantic query
func (s *CouchDBUserStore) ListUsers() ([]*User, error) {
	query := db.NewQueryBuilder().
		Where("@type", "$eq", "Person").
		Build()

	users, err := db.FindTyped[User](s.service, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Convert to pointer slice
	result := make([]*User, len(users))
	for i := range users {
		result[i] = &users[i]
	}

	return result, nil
}

// RecordLoginAttempt records a login attempt (updates user's failed login count)
func (s *CouchDBUserStore) RecordLoginAttempt(username string, success bool) error {
	user, err := s.GetUserByUsername(username)
	if err != nil {
		return err
	}

	if success {
		// Reset failed logins on success
		user.FailedLogins = 0
		user.LastLoginAt = timePtr(time.Now())
	} else {
		// Increment failed logins
		user.FailedLogins++
	}

	return s.UpdateUser(user)
}

// SaveRefreshToken saves a refresh token to CouchDB
func (s *CouchDBUserStore) SaveRefreshToken(token *RefreshToken) error {
	// Set JSON-LD semantic fields
	if token.Context == "" {
		token.Context = "https://schema.org"
	}
	if token.Type == "" {
		token.Type = "RefreshToken"
	}

	// Save to CouchDB
	resp, err := s.service.SaveGenericDocument(token)
	if err != nil {
		return fmt.Errorf("failed to save refresh token: %w", err)
	}

	// Update with CouchDB fields
	token.ID = resp.ID
	token.Rev = resp.Rev

	return nil
}

// GetRefreshToken retrieves a refresh token by ID
func (s *CouchDBUserStore) GetRefreshToken(id string) (*RefreshToken, error) {
	var token RefreshToken
	if err := s.service.GetGenericDocument(id, &token); err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}
	return &token, nil
}

// GetRefreshTokensByUserID retrieves all refresh tokens for a user using semantic query
func (s *CouchDBUserStore) GetRefreshTokensByUserID(userID string) ([]*RefreshToken, error) {
	query := db.NewQueryBuilder().
		Where("@type", "$eq", "RefreshToken").
		And().
		Where("user_id", "$eq", userID).
		Build()

	tokens, err := db.FindTyped[RefreshToken](s.service, query)
	if err != nil {
		return nil, fmt.Errorf("failed to find refresh tokens: %w", err)
	}

	// Convert to pointer slice
	result := make([]*RefreshToken, len(tokens))
	for i := range tokens {
		result[i] = &tokens[i]
	}

	return result, nil
}

// RevokeRefreshToken revokes a refresh token
func (s *CouchDBUserStore) RevokeRefreshToken(id string) error {
	token, err := s.GetRefreshToken(id)
	if err != nil {
		return err
	}

	token.Revoked = true

	// Save updated token
	resp, err := s.service.SaveGenericDocument(token)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	token.Rev = resp.Rev
	return nil
}

// DeleteExpiredRefreshTokens deletes all expired refresh tokens
func (s *CouchDBUserStore) DeleteExpiredRefreshTokens() error {
	now := time.Now()

	// Query for expired tokens
	query := db.NewQueryBuilder().
		Where("@type", "$eq", "RefreshToken").
		And().
		Where("expires_at", "$lt", now.Format(time.RFC3339)).
		Build()

	tokens, err := db.FindTyped[RefreshToken](s.service, query)
	if err != nil {
		return fmt.Errorf("failed to find expired tokens: %w", err)
	}

	// Delete each expired token
	for _, token := range tokens {
		if err := s.service.DeleteDocument(token.ID, token.Rev); err != nil {
			// Log error but continue with other tokens
			fmt.Printf("Warning: failed to delete expired token %s: %v\n", token.ID, err)
		}
	}

	return nil
}

// SaveAuditLog saves an audit log entry to CouchDB
func (s *CouchDBUserStore) SaveAuditLog(log *AuditLog) error {
	// Set JSON-LD semantic fields
	if log.Context == "" {
		log.Context = "https://schema.org"
	}
	if log.Type == "" {
		log.Type = "AuditLog"
	}

	// Generate ID if not set (timestamp-based for ordering)
	if log.ID == "" {
		log.ID = fmt.Sprintf("audit-%d", time.Now().UnixNano())
	}

	// Save to CouchDB
	_, err := s.service.SaveGenericDocument(log)
	if err != nil {
		return fmt.Errorf("failed to save audit log: %w", err)
	}

	return nil
}

// GetAuditLogs retrieves audit logs based on search criteria using semantic queries
func (s *CouchDBUserStore) GetAuditLogs(criteria AuditSearchCriteria) ([]*AuditLog, error) {
	// Build query based on criteria
	qb := db.NewQueryBuilder().
		Where("@type", "$eq", "AuditLog")

	if criteria.UserID != "" {
		qb = qb.And().Where("user_id", "$eq", criteria.UserID)
	}

	if criteria.Username != "" {
		qb = qb.And().Where("username", "$eq", criteria.Username)
	}

	if criteria.Action != "" {
		qb = qb.And().Where("action", "$eq", criteria.Action)
	}

	if criteria.Resource != "" {
		qb = qb.And().Where("resource", "$eq", criteria.Resource)
	}

	if criteria.Success != nil {
		qb = qb.And().Where("success", "$eq", *criteria.Success)
	}

	if criteria.StartTime != nil {
		qb = qb.And().Where("timestamp", "$gte", criteria.StartTime.Format(time.RFC3339))
	}

	if criteria.EndTime != nil {
		qb = qb.And().Where("timestamp", "$lte", criteria.EndTime.Format(time.RFC3339))
	}

	// Apply limit and offset
	if criteria.Limit > 0 {
		qb = qb.Limit(criteria.Limit)
	} else {
		qb = qb.Limit(100) // Default limit
	}

	if criteria.Offset > 0 {
		qb = qb.Skip(criteria.Offset)
	}

	// Execute query
	logs, err := db.FindTyped[AuditLog](s.service, qb.Build())
	if err != nil {
		return nil, fmt.Errorf("failed to find audit logs: %w", err)
	}

	// Convert to pointer slice
	result := make([]*AuditLog, len(logs))
	for i := range logs {
		result[i] = &logs[i]
	}

	return result, nil
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}
