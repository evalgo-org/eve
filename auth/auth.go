package auth

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"eve.evalgo.org/semantic"
)

// AuthService provides authentication and authorization
type AuthService interface {
	// Authentication
	Login(username, password string) (*AuthResult, error)
	Logout(userID string) error

	// Token management
	GenerateToken(user *User) (string, error)
	ValidateToken(token string) (*Claims, error)
	GenerateTokenPair(user *User) (*TokenPair, error)
	RefreshToken(refreshToken string) (*TokenPair, error)

	// Password management
	ChangePassword(userID, currentPassword, newPassword string) error
	HashPassword(password string) (string, error)
	ValidatePasswordHash(password, hash string) error

	// User management
	CreateUser(req CreateUserRequest) (*User, error)
	UpdateUser(userID string, req UpdateUserRequest) (*User, error)
	DeleteUser(userID string, requestingUserID string) error
	GetUser(userID string) (*User, error)
	GetUserByUsername(username string) (*User, error)
	ListUsers() ([]*User, error)

	// Authorization
	HasRole(userID string, role string) (bool, error)
	HasAnyRole(userID string, roles []string) (bool, error)
}

// authService implements AuthService
type authService struct {
	config       *Config
	store        UserStore
	tokenService *TokenService
}

// NewAuthService creates a new auth service
func NewAuthService(config *Config, store UserStore) AuthService {
	if config == nil {
		config = DefaultConfig()
	}

	tokenService := NewTokenService(
		config.JWTSecret,
		config.JWTExpiration,
		config.RefreshTokenExpiration,
	)

	return &authService{
		config:       config,
		store:        store,
		tokenService: tokenService,
	}
}

// Login authenticates a user and returns tokens
func (s *authService) Login(username, password string) (*AuthResult, error) {
	// Get user
	user, err := s.store.GetUserByUsername(username)
	if err != nil {
		s.audit("login_failed", username, "", false, "user not found")
		return nil, ErrInvalidCredentials
	}

	// Check if locked
	if user.Locked {
		s.audit("login_failed", username, user.ID, false, "account locked")
		return nil, ErrAccountLocked
	}

	// Check if enabled
	if !user.Enabled {
		s.audit("login_failed", username, user.ID, false, "account disabled")
		return nil, ErrAccountDisabled
	}

	// Verify password
	if err := ValidatePassword(password, user.PasswordHash); err != nil {
		s.store.RecordLoginAttempt(username, false)
		s.audit("login_failed", username, user.ID, false, "invalid password")
		return nil, ErrInvalidCredentials
	}

	// Record successful login
	s.store.RecordLoginAttempt(username, true)

	// Generate tokens
	var result *AuthResult
	if s.config.RefreshTokenEnabled {
		tokenPair, err := s.GenerateTokenPair(user)
		if err != nil {
			return nil, fmt.Errorf("failed to generate tokens: %w", err)
		}
		result = &AuthResult{
			User:         user,
			AccessToken:  tokenPair.AccessToken,
			RefreshToken: tokenPair.RefreshToken,
			ExpiresAt:    tokenPair.ExpiresAt,
		}
	} else {
		token, err := s.GenerateToken(user)
		if err != nil {
			return nil, fmt.Errorf("failed to generate token: %w", err)
		}
		result = &AuthResult{
			User:        user,
			AccessToken: token,
			ExpiresAt:   time.Now().Add(s.config.JWTExpiration),
		}
	}

	// Update last login
	now := time.Now()
	user.LastLoginAt = &now
	user.UpdatedAt = now
	s.store.UpdateUser(user)

	// Audit successful login
	s.audit("login", username, user.ID, true, "")

	return result, nil
}

// Logout logs out a user
func (s *authService) Logout(userID string) error {
	// Revoke all refresh tokens for the user
	if s.config.RefreshTokenEnabled {
		tokens, err := s.store.GetRefreshTokensByUserID(userID)
		if err == nil {
			for _, token := range tokens {
				s.store.RevokeRefreshToken(token.ID)
			}
		}
	}

	s.audit("logout", "", userID, true, "")
	return nil
}

// GenerateToken generates a JWT access token for a user
func (s *authService) GenerateToken(user *User) (string, error) {
	return s.tokenService.GenerateToken(user)
}

// ValidateToken validates a JWT token and returns the claims
func (s *authService) ValidateToken(token string) (*Claims, error) {
	return s.tokenService.ValidateToken(token)
}

// GenerateTokenPair generates both access and refresh tokens
func (s *authService) GenerateTokenPair(user *User) (*TokenPair, error) {
	tokenPair, err := s.tokenService.GenerateTokenPair(user)
	if err != nil {
		return nil, err
	}

	// Hash and store refresh token
	hashedToken, err := HashRefreshToken(tokenPair.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to hash refresh token: %w", err)
	}

	refreshToken := &RefreshToken{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		Token:     hashedToken,
		ExpiresAt: time.Now().Add(s.config.RefreshTokenExpiration),
		CreatedAt: time.Now(),
		Revoked:   false,
	}

	if err := s.store.SaveRefreshToken(refreshToken); err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	return tokenPair, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *authService) RefreshToken(refreshToken string) (*TokenPair, error) {
	// This is a simplified implementation
	// In a full implementation, you'd look up the stored refresh token,
	// validate it, get the user, and generate a new token pair
	return nil, fmt.Errorf("not implemented")
}

// ChangePassword changes a user's password
func (s *authService) ChangePassword(userID, currentPassword, newPassword string) error {
	user, err := s.store.GetUser(userID)
	if err != nil {
		return err
	}

	// Verify current password
	if err := ValidatePassword(currentPassword, user.PasswordHash); err != nil {
		s.audit("change_password_failed", user.Username, userID, false, "invalid current password")
		return ErrInvalidCredentials
	}

	// Validate new password
	if err := CheckPasswordStrength(newPassword, s.config.PasswordRequireStrong); err != nil {
		return err
	}

	// Hash new password
	hashedPassword, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update user
	user.PasswordHash = hashedPassword
	user.MustChangePassword = false
	user.UpdatedAt = time.Now()

	if err := s.store.UpdateUser(user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	s.audit("change_password", user.Username, userID, true, "")
	return nil
}

// HashPassword hashes a password
func (s *authService) HashPassword(password string) (string, error) {
	return HashPassword(password)
}

// ValidatePasswordHash validates a password against its hash
func (s *authService) ValidatePasswordHash(password, hash string) error {
	return ValidatePassword(password, hash)
}

// CreateUser creates a new user
func (s *authService) CreateUser(req CreateUserRequest) (*User, error) {
	// Validate username
	if err := ValidateUsername(req.Username); err != nil {
		return nil, err
	}

	// Validate email
	if err := ValidateEmail(req.Email); err != nil {
		return nil, err
	}

	// Validate password
	if err := CheckPasswordStrength(req.Password, s.config.PasswordRequireStrong); err != nil {
		return nil, err
	}

	// Check if username exists
	if _, err := s.store.GetUserByUsername(req.Username); err == nil {
		return nil, ErrUserExists
	}

	// Check if email exists (if provided)
	if req.Email != "" {
		if _, err := s.store.GetUserByEmail(req.Email); err == nil {
			return nil, ErrUserExists
		}
	}

	// Hash password
	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Set default role if none provided
	roles := req.Roles
	if len(roles) == 0 {
		roles = []string{s.config.DefaultRole}
	}

	// Create user
	now := time.Now()
	user := &User{
		ID:                 uuid.New().String(),
		Username:           req.Username,
		Email:              req.Email,
		PasswordHash:       hashedPassword,
		Roles:              roles,
		Enabled:            true,
		Locked:             false,
		MustChangePassword: req.MustChangePassword,
		FailedLogins:       0,
		Name:               req.Name,
		CreatedAt:          now,
		UpdatedAt:          now,
		Context:            "https://schema.org",
		Type:               "Person",
	}

	if err := s.store.CreateUser(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.audit("create_user", "", "", true, fmt.Sprintf("created user %s", req.Username))
	return user, nil
}

// UpdateUser updates an existing user
func (s *authService) UpdateUser(userID string, req UpdateUserRequest) (*User, error) {
	user, err := s.store.GetUser(userID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Email != nil {
		if err := ValidateEmail(*req.Email); err != nil {
			return nil, err
		}
		user.Email = *req.Email
	}

	if req.Password != nil {
		if err := CheckPasswordStrength(*req.Password, s.config.PasswordRequireStrong); err != nil {
			return nil, err
		}
		hashedPassword, err := HashPassword(*req.Password)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = hashedPassword
	}

	if req.Name != nil {
		user.Name = *req.Name
	}

	if req.Roles != nil {
		user.Roles = *req.Roles
	}

	if req.Enabled != nil {
		user.Enabled = *req.Enabled
	}

	if req.Locked != nil {
		user.Locked = *req.Locked
	}

	if req.MustChangePassword != nil {
		user.MustChangePassword = *req.MustChangePassword
	}

	if req.FailedLogins != nil {
		user.FailedLogins = *req.FailedLogins
	}

	user.UpdatedAt = time.Now()

	if err := s.store.UpdateUser(user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	s.audit("update_user", user.Username, userID, true, "")
	return user, nil
}

// DeleteUser deletes a user
func (s *authService) DeleteUser(userID string, requestingUserID string) error {
	// Prevent self-deletion
	if userID == requestingUserID {
		return ErrSelfDelete
	}

	user, err := s.store.GetUser(userID)
	if err != nil {
		return err
	}

	if err := s.store.DeleteUser(userID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	s.audit("delete_user", user.Username, userID, true, "")
	return nil
}

// GetUser gets a user by ID
func (s *authService) GetUser(userID string) (*User, error) {
	return s.store.GetUser(userID)
}

// GetUserByUsername gets a user by username
func (s *authService) GetUserByUsername(username string) (*User, error) {
	return s.store.GetUserByUsername(username)
}

// ListUsers lists all users
func (s *authService) ListUsers() ([]*User, error) {
	return s.store.ListUsers()
}

// HasRole checks if a user has a specific role
func (s *authService) HasRole(userID string, role string) (bool, error) {
	user, err := s.store.GetUser(userID)
	if err != nil {
		return false, err
	}

	for _, r := range user.Roles {
		if r == role {
			return true, nil
		}
	}

	return false, nil
}

// HasAnyRole checks if a user has any of the specified roles
func (s *authService) HasAnyRole(userID string, roles []string) (bool, error) {
	user, err := s.store.GetUser(userID)
	if err != nil {
		return false, err
	}

	for _, requiredRole := range roles {
		for _, userRole := range user.Roles {
			if userRole == requiredRole {
				return true, nil
			}
		}
	}

	return false, nil
}

// audit logs an audit entry
func (s *authService) audit(action, username, userID string, success bool, message string) {
	if !s.config.AuditEnabled {
		return
	}

	now := time.Now()

	// Determine action status
	actionStatus := "CompletedActionStatus"
	if !success {
		actionStatus = "FailedActionStatus"
	}

	log := &AuditLog{
		// Semantic fields
		Context:      "https://schema.org",
		Type:         "AssessAction",
		Name:         action,
		ActionStatus: actionStatus,
		StartTime:    now,

		// Identity
		ID: uuid.New().String(),

		// Legacy fields (for backward compatibility)
		Timestamp:    now,
		UserID:       userID,
		Username:     username,
		Action:       action,
		Success:      success,
		ErrorMessage: message,
	}

	// Create semantic agent if user info available (CANONICAL TYPE)
	if userID != "" || username != "" {
		log.Agent = &semantic.SemanticAgent{
			Type: "Person",
			Name: username,
		}
		// Store userID in properties since SemanticAgent doesn't have identifier field
		if log.Properties == nil {
			log.Properties = make(map[string]interface{})
		}
		log.Properties["userId"] = userID
	}

	// Create semantic error if failed (CANONICAL TYPE)
	if !success && message != "" {
		log.Error = &semantic.SemanticError{
			Type:    "Error",
			Message: message,
		}
	}

	s.store.SaveAuditLog(log)
}
