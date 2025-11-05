package auth

import "errors"

// Authentication errors
var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrAccountLocked      = errors.New("account is locked")
	ErrAccountDisabled    = errors.New("account is disabled")
	ErrExpiredToken       = errors.New("token has expired")
	ErrInvalidToken       = errors.New("invalid token")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrWeakPassword       = errors.New("password does not meet requirements")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidUsername    = errors.New("invalid username format")
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrEmptyPassword      = errors.New("password cannot be empty")
	ErrPasswordTooShort   = errors.New("password is too short")
	ErrSelfDelete         = errors.New("cannot delete your own account")
)
