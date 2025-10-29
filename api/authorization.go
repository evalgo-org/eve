// Package api provides authorization middleware for fine-grained access control.
// This file implements scope-based authorization and user context management.
package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// AuthUser represents an authenticated user with associated claims and permissions.
type AuthUser struct {
	// ID is the unique identifier for the user (typically from "sub" claim)
	ID string `json:"id"`

	// Username is the user's login name
	Username string `json:"username,omitempty"`

	// Email is the user's email address
	Email string `json:"email,omitempty"`

	// Name is the user's display name
	Name string `json:"name,omitempty"`

	// Scopes contains the authorization scopes granted to the user
	Scopes []string `json:"scopes,omitempty"`

	// Claims contains all JWT/OIDC claims for the user
	Claims map[string]interface{} `json:"claims,omitempty"`
}

// Context keys for storing authentication data
const (
	contextKeyUser   = "user"
	contextKeyClaims = "claims"
	contextKeyScopes = "scopes"
)

// SetUser stores the authenticated user in the Echo context.
// This is typically called by authentication middleware after successful authentication.
//
// Parameters:
//   - c: Echo context
//   - user: The authenticated user to store
//
// Example:
//
//	func authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
//	    return func(c echo.Context) error {
//	        // After validating credentials...
//	        user := &AuthUser{
//	            ID:       "user123",
//	            Username: "john.doe",
//	            Email:    "john@example.com",
//	            Scopes:   []string{"read", "write"},
//	        }
//	        SetUser(c, user)
//	        return next(c)
//	    }
//	}
func SetUser(c echo.Context, user *AuthUser) {
	c.Set(contextKeyUser, user)
}

// GetUser retrieves the authenticated user from the Echo context.
// Returns nil if no user is authenticated or if authentication middleware hasn't run.
//
// Parameters:
//   - c: Echo context
//
// Returns:
//   - *AuthUser: The authenticated user, or nil if not available
//   - bool: true if user was found in context, false otherwise
//
// Example:
//
//	func handler(c echo.Context) error {
//	    user, ok := GetUser(c)
//	    if !ok {
//	        return c.String(401, "Not authenticated")
//	    }
//	    return c.JSON(200, map[string]string{
//	        "message": "Hello, " + user.Username,
//	        "user_id": user.ID,
//	    })
//	}
func GetUser(c echo.Context) (*AuthUser, bool) {
	user, ok := c.Get(contextKeyUser).(*AuthUser)
	return user, ok
}

// SetClaims stores JWT/OIDC claims in the Echo context.
// This is typically called by authentication middleware after token validation.
//
// Parameters:
//   - c: Echo context
//   - claims: The claims to store
//
// Example:
//
//	func jwtMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
//	    return func(c echo.Context) error {
//	        // After validating JWT token...
//	        claims := map[string]interface{}{
//	            "sub":   "user123",
//	            "email": "john@example.com",
//	            "role":  "admin",
//	        }
//	        SetClaims(c, claims)
//	        return next(c)
//	    }
//	}
func SetClaims(c echo.Context, claims map[string]interface{}) {
	c.Set(contextKeyClaims, claims)
}

// GetClaims retrieves JWT/OIDC claims from the Echo context.
//
// Parameters:
//   - c: Echo context
//
// Returns:
//   - map[string]interface{}: The claims map, or nil if not available
//   - bool: true if claims were found in context, false otherwise
//
// Example:
//
//	func handler(c echo.Context) error {
//	    claims, ok := GetClaims(c)
//	    if !ok {
//	        return c.String(401, "No claims available")
//	    }
//
//	    role, _ := claims["role"].(string)
//	    return c.JSON(200, map[string]interface{}{
//	        "role":   role,
//	        "claims": claims,
//	    })
//	}
func GetClaims(c echo.Context) (map[string]interface{}, bool) {
	claims, ok := c.Get(contextKeyClaims).(map[string]interface{})
	return claims, ok
}

// SetScopes stores authorization scopes in the Echo context.
// This is typically called by authentication middleware.
//
// Parameters:
//   - c: Echo context
//   - scopes: List of scope strings
func SetScopes(c echo.Context, scopes []string) {
	c.Set(contextKeyScopes, scopes)
}

// GetScopes retrieves authorization scopes from the Echo context.
//
// Parameters:
//   - c: Echo context
//
// Returns:
//   - []string: List of scopes, or nil if not available
//   - bool: true if scopes were found in context, false otherwise
func GetScopes(c echo.Context) ([]string, bool) {
	scopes, ok := c.Get(contextKeyScopes).([]string)
	return scopes, ok
}

// RequireScope returns Echo middleware that enforces scope-based authorization.
// It checks if the authenticated user has at least one of the required scopes.
//
// The middleware expects scopes to be stored in the context (by authentication middleware)
// either in the User object or directly via SetScopes.
//
// If the user doesn't have any of the required scopes, it returns 403 Forbidden.
// If no scopes are stored in context, it returns 401 Unauthorized.
//
// Parameters:
//   - requiredScopes: One or more scopes, where the user must have at least one
//
// Returns:
//   - echo.MiddlewareFunc: Configured middleware function
//
// Example:
//
//	e := echo.New()
//
//	// Require "read" scope for GET endpoints
//	api := e.Group("/api")
//	api.Use(jwtMiddleware) // Must set scopes in context
//	api.GET("/data", handler, RequireScope("read"))
//
//	// Require either "admin" or "write" scope
//	api.POST("/data", handler, RequireScope("admin", "write"))
//
//	// Chain multiple scope requirements
//	api.DELETE("/data/:id", handler,
//	    RequireScope("write"),      // Must have write
//	    RequireScope("admin"),      // AND must have admin
//	)
func RequireScope(requiredScopes ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Try to get scopes from User object first
			var userScopes []string
			if user, ok := GetUser(c); ok && user != nil {
				userScopes = user.Scopes
			}

			// If not in User, try direct scopes in context
			if len(userScopes) == 0 {
				if scopes, ok := GetScopes(c); ok {
					userScopes = scopes
				}
			}

			// If still no scopes, check claims for scope/scopes claim
			if len(userScopes) == 0 {
				if claims, ok := GetClaims(c); ok {
					userScopes = extractScopesFromClaims(claims)
				}
			}

			// No scopes available - user not authenticated properly
			if len(userScopes) == 0 {
				return echo.NewHTTPError(http.StatusUnauthorized,
					"Authentication required: no scopes available")
			}

			// Check if user has at least one of the required scopes
			if !hasAnyScope(userScopes, requiredScopes) {
				return echo.NewHTTPError(http.StatusForbidden,
					"Insufficient permissions: missing required scope")
			}

			// Authorization successful
			return next(c)
		}
	}
}

// RequireAllScopes returns Echo middleware that enforces scope-based authorization.
// Unlike RequireScope, this requires the user to have ALL specified scopes.
//
// Parameters:
//   - requiredScopes: Scopes that the user must have (all of them)
//
// Returns:
//   - echo.MiddlewareFunc: Configured middleware function
//
// Example:
//
//	// User must have both "read" AND "write" scopes
//	api.POST("/data", handler, RequireAllScopes("read", "write"))
func RequireAllScopes(requiredScopes ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get user scopes (same logic as RequireScope)
			var userScopes []string
			if user, ok := GetUser(c); ok && user != nil {
				userScopes = user.Scopes
			}
			if len(userScopes) == 0 {
				if scopes, ok := GetScopes(c); ok {
					userScopes = scopes
				}
			}
			if len(userScopes) == 0 {
				if claims, ok := GetClaims(c); ok {
					userScopes = extractScopesFromClaims(claims)
				}
			}

			if len(userScopes) == 0 {
				return echo.NewHTTPError(http.StatusUnauthorized,
					"Authentication required: no scopes available")
			}

			// Check if user has ALL required scopes
			if !hasAllScopes(userScopes, requiredScopes) {
				return echo.NewHTTPError(http.StatusForbidden,
					"Insufficient permissions: missing required scopes")
			}

			return next(c)
		}
	}
}

// hasAnyScope checks if the user has at least one of the required scopes.
func hasAnyScope(userScopes, requiredScopes []string) bool {
	for _, required := range requiredScopes {
		for _, user := range userScopes {
			if user == required {
				return true
			}
		}
	}
	return false
}

// hasAllScopes checks if the user has all of the required scopes.
func hasAllScopes(userScopes, requiredScopes []string) bool {
	for _, required := range requiredScopes {
		found := false
		for _, user := range userScopes {
			if user == required {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// extractScopesFromClaims extracts scopes from JWT/OIDC claims.
// Handles various claim formats:
//   - "scope": "read write admin" (space-separated string)
//   - "scope": ["read", "write", "admin"] (array)
//   - "scopes": ["read", "write", "admin"] (array)
func extractScopesFromClaims(claims map[string]interface{}) []string {
	// Try "scope" claim (OAuth2/OIDC standard)
	if scope, ok := claims["scope"]; ok {
		// Handle space-separated string
		if scopeStr, ok := scope.(string); ok {
			return parseSpaceSeparatedScopes(scopeStr)
		}
		// Handle array
		if scopeArr, ok := scope.([]interface{}); ok {
			return interfaceArrayToStringArray(scopeArr)
		}
	}

	// Try "scopes" claim (alternative)
	if scopes, ok := claims["scopes"]; ok {
		if scopeArr, ok := scopes.([]interface{}); ok {
			return interfaceArrayToStringArray(scopeArr)
		}
	}

	return nil
}

// parseSpaceSeparatedScopes splits a space-separated scope string.
func parseSpaceSeparatedScopes(scopes string) []string {
	if scopes == "" {
		return nil
	}
	var result []string
	for _, scope := range splitString(scopes, ' ') {
		if scope != "" {
			result = append(result, scope)
		}
	}
	return result
}

// splitString splits a string by a delimiter (simple implementation).
func splitString(s string, delimiter rune) []string {
	var result []string
	var current string
	for _, c := range s {
		if c == delimiter {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// interfaceArrayToStringArray converts []interface{} to []string.
func interfaceArrayToStringArray(arr []interface{}) []string {
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if str, ok := v.(string); ok {
			result = append(result, str)
		}
	}
	return result
}
