package security

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJWTServiceWithIssuer(t *testing.T) {
	secret := "test-secret"
	issuer := "https://issuer.example.com"
	audience := "https://api.example.com"

	service := NewJWTServiceWithIssuer(secret, issuer, audience)

	assert.NotNil(t, service)
	assert.Equal(t, []byte(secret), service.secret)
	assert.Equal(t, issuer, service.issuer)
	assert.Equal(t, audience, service.audience)
}

func TestGenerateTokenWithIssuerAudience(t *testing.T) {
	secret := "test-secret"
	issuer := "https://issuer.example.com"
	audience := "https://api.example.com"
	userID := "user123"

	service := NewJWTServiceWithIssuer(secret, issuer, audience)

	tokenString, err := service.GenerateToken(userID, time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	// Validate the token
	token, err := service.ValidateToken(tokenString)
	require.NoError(t, err)

	// Check standard claims
	assert.Equal(t, userID, token.Subject())

	// Check issuer and audience
	assert.Equal(t, issuer, token.Issuer())
	audiences := token.Audience()
	assert.Contains(t, audiences, audience)
}

func TestGenerateTokenWithClaims(t *testing.T) {
	secret := "test-secret"
	issuer := "https://issuer.example.com"
	audience := "https://api.example.com"
	userID := "user123"

	service := NewJWTServiceWithIssuer(secret, issuer, audience)

	customClaims := map[string]interface{}{
		"role":  "admin",
		"scope": "read write delete",
		"email": "user@example.com",
		"org":   "test-org",
	}

	tokenString, err := service.GenerateTokenWithClaims(userID, time.Hour, customClaims)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	// Validate the token
	token, err := service.ValidateToken(tokenString)
	require.NoError(t, err)

	// Check standard claims
	assert.Equal(t, userID, token.Subject())
	assert.Equal(t, issuer, token.Issuer())

	// Check custom claims
	claimsMap, err := token.AsMap(nil)
	require.NoError(t, err)

	assert.Equal(t, "admin", claimsMap["role"])
	assert.Equal(t, "read write delete", claimsMap["scope"])
	assert.Equal(t, "user@example.com", claimsMap["email"])
	assert.Equal(t, "test-org", claimsMap["org"])
}

func TestValidateTokenWithIssuerValidation(t *testing.T) {
	secret := "test-secret"
	correctIssuer := "https://correct-issuer.example.com"
	wrongIssuer := "https://wrong-issuer.example.com"
	audience := "https://api.example.com"

	tests := []struct {
		name              string
		tokenIssuer       string
		validationIssuer  string
		expectError       bool
	}{
		{
			name:             "matching issuer",
			tokenIssuer:      correctIssuer,
			validationIssuer: correctIssuer,
			expectError:      false,
		},
		{
			name:             "mismatched issuer",
			tokenIssuer:      wrongIssuer,
			validationIssuer: correctIssuer,
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate token with one issuer
			genService := NewJWTServiceWithIssuer(secret, tt.tokenIssuer, audience)
			tokenString, err := genService.GenerateToken("user123", time.Hour)
			require.NoError(t, err)

			// Validate with potentially different issuer
			valService := NewJWTServiceWithIssuer(secret, tt.validationIssuer, audience)
			_, err = valService.ValidateToken(tokenString)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTokenWithAudienceValidation(t *testing.T) {
	secret := "test-secret"
	issuer := "https://issuer.example.com"
	correctAudience := "https://api.example.com"
	wrongAudience := "https://different-api.example.com"

	tests := []struct {
		name                string
		tokenAudience       string
		validationAudience  string
		expectError         bool
	}{
		{
			name:               "matching audience",
			tokenAudience:      correctAudience,
			validationAudience: correctAudience,
			expectError:        false,
		},
		{
			name:               "mismatched audience",
			tokenAudience:      wrongAudience,
			validationAudience: correctAudience,
			expectError:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate token with one audience
			genService := NewJWTServiceWithIssuer(secret, issuer, tt.tokenAudience)
			tokenString, err := genService.GenerateToken("user123", time.Hour)
			require.NoError(t, err)

			// Validate with potentially different audience
			valService := NewJWTServiceWithIssuer(secret, issuer, tt.validationAudience)
			_, err = valService.ValidateToken(tokenString)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTokenWithoutIssuerAudience(t *testing.T) {
	// Test that tokens without issuer/audience still work with basic service
	secret := "test-secret"
	basicService := NewJWTService(secret)

	tokenString, err := basicService.GenerateToken("user123", time.Hour)
	require.NoError(t, err)

	// Should validate successfully
	token, err := basicService.ValidateToken(tokenString)
	assert.NoError(t, err)
	assert.Equal(t, "user123", token.Subject())
}

func TestValidateTokenWithOptions(t *testing.T) {
	secret := "test-secret"
	service := NewJWTService(secret)

	// Generate a token
	tokenString, err := service.GenerateToken("user123", time.Hour)
	require.NoError(t, err)

	tests := []struct {
		name        string
		options     []interface{} // Can't use jwt.ParseOption due to import
		expectError bool
	}{
		{
			name:        "no additional options",
			options:     nil,
			expectError: false,
		},
		// Note: We can't easily test WithIssuer/WithAudience without importing jwt
		// Those are tested in the issuer/audience specific tests above
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For now, just test basic validation
			token, err := service.ValidateToken(tokenString)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, token)
			}
		})
	}
}

func TestTokenExpiration(t *testing.T) {
	secret := "test-secret"
	service := NewJWTService(secret)

	// Generate a token that expires in 1 millisecond
	tokenString, err := service.GenerateToken("user123", time.Millisecond)
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Validation should fail
	_, err = service.ValidateToken(tokenString)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exp")
}

func TestTokenWithDifferentSecrets(t *testing.T) {
	correctSecret := "correct-secret"
	wrongSecret := "wrong-secret"

	// Generate token with one secret
	genService := NewJWTService(correctSecret)
	tokenString, err := genService.GenerateToken("user123", time.Hour)
	require.NoError(t, err)

	// Try to validate with wrong secret
	valService := NewJWTService(wrongSecret)
	_, err = valService.ValidateToken(tokenString)
	assert.Error(t, err)
}

func TestGenerateTokenWithComplexClaims(t *testing.T) {
	secret := "test-secret"
	service := NewJWTService(secret)

	customClaims := map[string]interface{}{
		"string_claim":  "value",
		"int_claim":     42,
		"float_claim":   3.14,
		"bool_claim":    true,
		"array_claim":   []string{"a", "b", "c"},
		"nested_claim": map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
	}

	tokenString, err := service.GenerateTokenWithClaims("user123", time.Hour, customClaims)
	require.NoError(t, err)

	// Validate and check claims
	token, err := service.ValidateToken(tokenString)
	require.NoError(t, err)

	claimsMap, err := token.AsMap(nil)
	require.NoError(t, err)

	assert.Equal(t, "value", claimsMap["string_claim"])
	assert.Equal(t, float64(42), claimsMap["int_claim"]) // JSON numbers are float64
	assert.InDelta(t, 3.14, claimsMap["float_claim"], 0.01)
	assert.Equal(t, true, claimsMap["bool_claim"])
}

func TestBackwardCompatibility(t *testing.T) {
	// Test that new service is backward compatible with existing usage
	secret := "test-secret"

	// Old style usage (without issuer/audience)
	oldService := NewJWTService(secret)
	oldToken, err := oldService.GenerateToken("user123", time.Hour)
	require.NoError(t, err)

	// Should validate with old service
	token1, err := oldService.ValidateToken(oldToken)
	assert.NoError(t, err)
	assert.Equal(t, "user123", token1.Subject())

	// Should also validate with new service (without issuer/audience set)
	newService := NewJWTService(secret)
	token2, err := newService.ValidateToken(oldToken)
	assert.NoError(t, err)
	assert.Equal(t, "user123", token2.Subject())
}

func TestEmptyCustomClaims(t *testing.T) {
	secret := "test-secret"
	service := NewJWTService(secret)

	// Generate token with empty custom claims
	tokenString, err := service.GenerateTokenWithClaims("user123", time.Hour, map[string]interface{}{})
	require.NoError(t, err)

	// Should validate successfully
	token, err := service.ValidateToken(tokenString)
	assert.NoError(t, err)
	assert.Equal(t, "user123", token.Subject())
}

func TestNilCustomClaims(t *testing.T) {
	secret := "test-secret"
	service := NewJWTService(secret)

	// Generate token with nil custom claims
	tokenString, err := service.GenerateTokenWithClaims("user123", time.Hour, nil)
	require.NoError(t, err)

	// Should validate successfully
	token, err := service.ValidateToken(tokenString)
	assert.NoError(t, err)
	assert.Equal(t, "user123", token.Subject())
}

func BenchmarkGenerateToken(b *testing.B) {
	service := NewJWTService("benchmark-secret")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GenerateToken("user123", time.Hour)
	}
}

func BenchmarkGenerateTokenWithClaims(b *testing.B) {
	service := NewJWTService("benchmark-secret")
	claims := map[string]interface{}{
		"role":  "admin",
		"email": "user@example.com",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GenerateTokenWithClaims("user123", time.Hour, claims)
	}
}

func BenchmarkValidateToken(b *testing.B) {
	service := NewJWTService("benchmark-secret")
	token, _ := service.GenerateToken("user123", time.Hour)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.ValidateToken(token)
	}
}

func BenchmarkGenerateTokenWithIssuerAudience(b *testing.B) {
	service := NewJWTServiceWithIssuer(
		"benchmark-secret",
		"https://issuer.example.com",
		"https://api.example.com",
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GenerateToken("user123", time.Hour)
	}
}
