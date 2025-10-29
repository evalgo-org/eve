package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractScopesFromXSUAA(t *testing.T) {
	tests := []struct {
		name     string
		claims   map[string]interface{}
		expected []string
	}{
		{
			name: "scope as array of strings",
			claims: map[string]interface{}{
				"scope": []interface{}{"myapp.read", "myapp.write", "myapp.admin"},
			},
			expected: []string{"myapp.read", "myapp.write", "myapp.admin"},
		},
		{
			name: "scope as single string",
			claims: map[string]interface{}{
				"scope": "myapp.read",
			},
			expected: []string{"myapp.read"},
		},
		{
			name: "no scope claim",
			claims: map[string]interface{}{
				"sub": "user123",
				"aud": "myapp",
			},
			expected: nil,
		},
		{
			name: "empty scope array",
			claims: map[string]interface{}{
				"scope": []interface{}{},
			},
			expected: []string{},
		},
		{
			name: "scope array with non-string values",
			claims: map[string]interface{}{
				"scope": []interface{}{"myapp.read", 123, "myapp.write"},
			},
			expected: []string{"myapp.read", "myapp.write"},
		},
		{
			name: "typical XSUAA scopes",
			claims: map[string]interface{}{
				"scope": []interface{}{
					"openid",
					"uaa.user",
					"myapp.Display",
					"myapp.Create",
					"myapp.Update",
					"myapp.Delete",
				},
			},
			expected: []string{"openid", "uaa.user", "myapp.Display", "myapp.Create", "myapp.Update", "myapp.Delete"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractScopesFromXSUAA(tt.claims)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasXSUAAScope(t *testing.T) {
	tests := []struct {
		name     string
		claims   map[string]interface{}
		scope    string
		expected bool
	}{
		{
			name: "has scope",
			claims: map[string]interface{}{
				"scope": []interface{}{"myapp.read", "myapp.write"},
			},
			scope:    "myapp.read",
			expected: true,
		},
		{
			name: "does not have scope",
			claims: map[string]interface{}{
				"scope": []interface{}{"myapp.read", "myapp.write"},
			},
			scope:    "myapp.admin",
			expected: false,
		},
		{
			name: "empty scopes",
			claims: map[string]interface{}{
				"scope": []interface{}{},
			},
			scope:    "myapp.read",
			expected: false,
		},
		{
			name: "no scope claim",
			claims: map[string]interface{}{
				"sub": "user123",
			},
			scope:    "myapp.read",
			expected: false,
		},
		{
			name: "exact scope match required",
			claims: map[string]interface{}{
				"scope": []interface{}{"myapp.read", "myapp.write"},
			},
			scope:    "myapp",
			expected: false,
		},
		{
			name: "case sensitive scope check",
			claims: map[string]interface{}{
				"scope": []interface{}{"myapp.Read"},
			},
			scope:    "myapp.read",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasXSUAAScope(tt.claims, tt.scope)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasXSUAAScope_MultipleScopes(t *testing.T) {
	claims := map[string]interface{}{
		"scope": []interface{}{
			"openid",
			"uaa.user",
			"myapp.Display",
			"myapp.Create",
			"myapp.Update",
			"myapp.Delete",
		},
	}

	// Test each scope individually
	assert.True(t, HasXSUAAScope(claims, "openid"))
	assert.True(t, HasXSUAAScope(claims, "uaa.user"))
	assert.True(t, HasXSUAAScope(claims, "myapp.Display"))
	assert.True(t, HasXSUAAScope(claims, "myapp.Create"))
	assert.True(t, HasXSUAAScope(claims, "myapp.Update"))
	assert.True(t, HasXSUAAScope(claims, "myapp.Delete"))

	// Test scopes that don't exist
	assert.False(t, HasXSUAAScope(claims, "myapp.Admin"))
	assert.False(t, HasXSUAAScope(claims, "other.app.scope"))
}

func TestXSUAAScopeWorkflow(t *testing.T) {
	// Simulate a typical XSUAA token claims structure
	t.Run("complete XSUAA token workflow", func(t *testing.T) {
		// Typical XSUAA token claims
		claims := map[string]interface{}{
			"jti":        "abc123",
			"sub":        "user-guid-1234",
			"scope":      []interface{}{"openid", "uaa.user", "myapp.read", "myapp.write"},
			"client_id":  "myapp!t1234",
			"cid":        "myapp!t1234",
			"azp":        "myapp!t1234",
			"grant_type": "authorization_code",
			"user_id":    "P123456",
			"origin":     "sap.default",
			"user_name":  "john.doe@example.com",
			"email":      "john.doe@example.com",
			"auth_time":  1234567890,
			"rev_sig":    "signature",
			"iat":        1234567890,
			"exp":        1234571490,
			"iss":        "https://tenant.authentication.eu10.hana.ondemand.com/oauth/token",
			"zid":        "tenant-zone-id",
			"aud": []interface{}{
				"openid",
				"uaa",
				"myapp!t1234",
			},
		}

		// Extract scopes
		scopes := ExtractScopesFromXSUAA(claims)
		assert.NotNil(t, scopes)
		assert.Len(t, scopes, 4)
		assert.Contains(t, scopes, "openid")
		assert.Contains(t, scopes, "uaa.user")
		assert.Contains(t, scopes, "myapp.read")
		assert.Contains(t, scopes, "myapp.write")

		// Check specific scopes
		assert.True(t, HasXSUAAScope(claims, "myapp.read"))
		assert.True(t, HasXSUAAScope(claims, "myapp.write"))
		assert.False(t, HasXSUAAScope(claims, "myapp.admin"))
	})

	t.Run("application-specific scopes", func(t *testing.T) {
		claims := map[string]interface{}{
			"scope": []interface{}{
				"openid",
				"myapp!t1234.Display",
				"myapp!t1234.Create",
				"myapp!t1234.Update",
			},
		}

		// Check application-specific scopes with app name prefix
		assert.True(t, HasXSUAAScope(claims, "myapp!t1234.Display"))
		assert.True(t, HasXSUAAScope(claims, "myapp!t1234.Create"))
		assert.True(t, HasXSUAAScope(claims, "myapp!t1234.Update"))
		assert.False(t, HasXSUAAScope(claims, "myapp!t1234.Delete"))
	})
}

func TestExtractScopesFromXSUAA_EdgeCases(t *testing.T) {
	t.Run("nil claims", func(t *testing.T) {
		var claims map[string]interface{}
		result := ExtractScopesFromXSUAA(claims)
		assert.Nil(t, result)
	})

	t.Run("empty claims", func(t *testing.T) {
		claims := make(map[string]interface{})
		result := ExtractScopesFromXSUAA(claims)
		assert.Nil(t, result)
	})

	t.Run("scope with wrong type", func(t *testing.T) {
		claims := map[string]interface{}{
			"scope": 12345, // not a string or array
		}
		result := ExtractScopesFromXSUAA(claims)
		assert.Nil(t, result)
	})

	t.Run("mixed array types", func(t *testing.T) {
		claims := map[string]interface{}{
			"scope": []interface{}{
				"valid.scope",
				nil,
				123,
				true,
				"another.valid.scope",
			},
		}
		result := ExtractScopesFromXSUAA(claims)
		assert.Equal(t, []string{"valid.scope", "another.valid.scope"}, result)
	})
}

func BenchmarkExtractScopesFromXSUAA(b *testing.B) {
	claims := map[string]interface{}{
		"scope": []interface{}{"scope1", "scope2", "scope3", "scope4", "scope5"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractScopesFromXSUAA(claims)
	}
}

func BenchmarkHasXSUAAScope(b *testing.B) {
	claims := map[string]interface{}{
		"scope": []interface{}{"scope1", "scope2", "scope3", "scope4", "scope5"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HasXSUAAScope(claims, "scope3")
	}
}
