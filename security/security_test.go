package security

import (
	"bytes"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestZitiCreateCSR tests CSR generation
func TestZitiCreateCSR(t *testing.T) {
	tmpDir := t.TempDir()

	privKeyPath := filepath.Join(tmpDir, "private.pem")
	csrPath := filepath.Join(tmpDir, "csr.pem")

	// Generate CSR
	err := ZitiCreateCSR(privKeyPath, csrPath)
	require.NoError(t, err)

	// Verify private key file exists
	assert.FileExists(t, privKeyPath)

	// Verify CSR file exists
	assert.FileExists(t, csrPath)

	// Read and parse private key
	privKeyData, err := os.ReadFile(privKeyPath)
	require.NoError(t, err)

	block, _ := pem.Decode(privKeyData)
	require.NotNil(t, block)
	assert.Equal(t, "EC PRIVATE KEY", block.Type)

	// Read and parse CSR
	csrData, err := os.ReadFile(csrPath)
	require.NoError(t, err)

	csrBlock, _ := pem.Decode(csrData)
	require.NotNil(t, csrBlock)
	assert.Equal(t, "CERTIFICATE REQUEST", csrBlock.Type)

	csr, err := x509.ParseCertificateRequest(csrBlock.Bytes)
	require.NoError(t, err)

	// Verify CSR details
	assert.Equal(t, "ziti-edge-router", csr.Subject.CommonName)
	assert.Contains(t, csr.Subject.Organization, "OpenZiti")
	assert.Equal(t, x509.ECDSAWithSHA256, csr.SignatureAlgorithm)
}

// TestSunsetSigAlgs tests the sunset signature algorithms map
func TestSunsetSigAlgs(t *testing.T) {
	// Verify deprecated algorithms are present
	_, hasMD2 := sunsetSigAlgs[x509.MD2WithRSA]
	assert.True(t, hasMD2, "MD2WithRSA should be in sunset list")

	_, hasMD5 := sunsetSigAlgs[x509.MD5WithRSA]
	assert.True(t, hasMD5, "MD5WithRSA should be in sunset list")

	_, hasSHA1 := sunsetSigAlgs[x509.SHA1WithRSA]
	assert.True(t, hasSHA1, "SHA1WithRSA should be in sunset list")

	// Verify sunset dates are in the past
	md2Sunset := sunsetSigAlgs[x509.MD2WithRSA].sunsetsAt
	assert.True(t, md2Sunset.Before(time.Now()), "MD2 should already be sunset")

	md5Sunset := sunsetSigAlgs[x509.MD5WithRSA].sunsetsAt
	assert.True(t, md5Sunset.Before(time.Now()), "MD5 should already be sunset")
}

// TestHostResultStruct tests the hostResult struct
func TestHostResultStruct(t *testing.T) {
	result := hostResult{
		Host:       "example.com:443",
		Err:        nil,
		CommonName: "example.com",
	}

	assert.Equal(t, "example.com:443", result.Host)
	assert.NoError(t, result.Err)
	assert.Equal(t, "example.com", result.CommonName)
}

// TestSigAlgSunsetStruct tests the sigAlgSunset struct
func TestSigAlgSunsetStruct(t *testing.T) {
	sunset := sigAlgSunset{
		name:      "MD5 with RSA",
		sunsetsAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	assert.Equal(t, "MD5 with RSA", sunset.name)
	assert.Equal(t, 2020, sunset.sunsetsAt.Year())
}

// TestCertsCheckHost_InvalidHost tests checking invalid hosts
func TestCertsCheckHost_InvalidHost(t *testing.T) {
	warnYears := 0
	warnMonths := 1
	warnDays := 0

	tests := []struct {
		name string
		host string
	}{
		{
			name: "NonExistentHost",
			host: "nonexistent-host-12345.invalid:443",
		},
		{
			name: "InvalidPort",
			host: "example.com:99999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CertsCheckHost(tt.host, &warnYears, &warnMonths, &warnDays)
			assert.Equal(t, tt.host, result.Host)
			assert.Error(t, result.Err)
		})
	}
}

// TestErrorMessageFormats tests error message formatting
func TestErrorMessageFormats(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{
			name:   "ExpiringShortly",
			format: errExpiringShortly,
		},
		{
			name:   "ExpiringSoon",
			format: errExpiringSoon,
		},
		{
			name:   "SunsetAlg",
			format: errSunsetAlg,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.format)
			assert.Contains(t, tt.format, "%s")
		})
	}
}

// TestPKIXName tests certificate subject structure
func TestPKIXName(t *testing.T) {
	subj := pkix.Name{
		CommonName:   "test-router",
		Organization: []string{"TestOrg"},
	}

	assert.Equal(t, "test-router", subj.CommonName)
	assert.Contains(t, subj.Organization, "TestOrg")
}

// TestX509SignatureAlgorithms tests signature algorithm constants
func TestX509SignatureAlgorithms(t *testing.T) {
	// Verify deprecated algorithms exist
	assert.NotEqual(t, x509.UnknownSignatureAlgorithm, x509.MD2WithRSA)
	assert.NotEqual(t, x509.UnknownSignatureAlgorithm, x509.MD5WithRSA)
	assert.NotEqual(t, x509.UnknownSignatureAlgorithm, x509.SHA1WithRSA)
	assert.NotEqual(t, x509.UnknownSignatureAlgorithm, x509.ECDSAWithSHA256)
}

// BenchmarkZitiCreateCSR benchmarks CSR generation
func BenchmarkZitiCreateCSR(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		privPath := filepath.Join(tmpDir, "priv.pem")
		csrPath := filepath.Join(tmpDir, "csr.pem")
		_ = ZitiCreateCSR(privPath, csrPath)
		os.Remove(privPath)
		os.Remove(csrPath)
	}
}

// TestNewJWTService tests JWT service initialization
func TestNewJWTService(t *testing.T) {
	tests := []struct {
		name   string
		secret string
	}{
		{
			name:   "SimpleSecret",
			secret: "test-secret",
		},
		{
			name:   "LongSecret",
			secret: "this-is-a-very-long-secret-key-for-testing-purposes",
		},
		{
			name:   "EmptySecret",
			secret: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewJWTService(tt.secret)
			assert.NotNil(t, service)
			assert.NotNil(t, service.secret)
			assert.Equal(t, []byte(tt.secret), service.secret)
		})
	}
}

// TestJWTService_GenerateToken tests token generation
func TestJWTService_GenerateToken(t *testing.T) {
	service := NewJWTService("test-secret-key")

	tests := []struct {
		name       string
		userID     string
		expiration time.Duration
	}{
		{
			name:       "OneHourExpiration",
			userID:     "user123",
			expiration: time.Hour,
		},
		{
			name:       "OneMinuteExpiration",
			userID:     "user456",
			expiration: time.Minute,
		},
		{
			name:       "EmptyUserID",
			userID:     "",
			expiration: time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := service.GenerateToken(tt.userID, tt.expiration)
			assert.NoError(t, err)
			assert.NotEmpty(t, token)

			// Token should have 3 parts separated by dots (header.payload.signature)
			parts := strings.Split(token, ".")
			assert.Equal(t, 3, len(parts))
		})
	}
}

// TestJWTService_ValidateToken tests token validation
func TestJWTService_ValidateToken(t *testing.T) {
	service := NewJWTService("test-secret-key")

	t.Run("ValidToken", func(t *testing.T) {
		// Generate a valid token
		tokenStr, err := service.GenerateToken("user789", time.Hour)
		require.NoError(t, err)

		// Validate it
		token, err := service.ValidateToken(tokenStr)
		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.Equal(t, "user789", token.Subject())
	})

	t.Run("InvalidToken", func(t *testing.T) {
		_, err := service.ValidateToken("invalid.token.here")
		assert.Error(t, err)
	})

	t.Run("EmptyToken", func(t *testing.T) {
		_, err := service.ValidateToken("")
		assert.Error(t, err)
	})

	t.Run("WrongSecret", func(t *testing.T) {
		// Generate token with one service
		service1 := NewJWTService("secret1")
		tokenStr, err := service1.GenerateToken("user999", time.Hour)
		require.NoError(t, err)

		// Try to validate with different secret
		service2 := NewJWTService("secret2")
		_, err = service2.ValidateToken(tokenStr)
		assert.Error(t, err)
	})
}

// TestJWTService_TokenRoundtrip tests full token lifecycle
func TestJWTService_TokenRoundtrip(t *testing.T) {
	service := NewJWTService("roundtrip-secret")

	userID := "test-user-123"
	expiration := time.Hour

	// Generate token
	tokenStr, err := service.GenerateToken(userID, expiration)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenStr)

	// Validate token
	token, err := service.ValidateToken(tokenStr)
	require.NoError(t, err)
	assert.NotNil(t, token)

	// Verify claims
	assert.Equal(t, userID, token.Subject())
	assert.False(t, token.IssuedAt().IsZero())
	assert.False(t, token.Expiration().IsZero())
	assert.True(t, token.Expiration().After(time.Now()))
}

// BenchmarkJWTService_GenerateToken benchmarks token generation
func BenchmarkJWTService_GenerateToken(b *testing.B) {
	service := NewJWTService("benchmark-secret")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GenerateToken("user123", time.Hour)
	}
}

// BenchmarkJWTService_ValidateToken benchmarks token validation
func BenchmarkJWTService_ValidateToken(b *testing.B) {
	service := NewJWTService("benchmark-secret")
	token, _ := service.GenerateToken("user123", time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.ValidateToken(token)
	}
}

// TestEncryptFile_Success tests successful file encryption
func TestEncryptFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "plain.txt")
	outputFile := filepath.Join(tmpDir, "cipher.enc")

	// Create test file
	testData := []byte("Hello, World! This is secret data.")
	err := os.WriteFile(inputFile, testData, 0644)
	require.NoError(t, err)

	// Encrypt the file
	password := "test-password-123"
	err = EncryptFile(password, inputFile, outputFile)
	assert.NoError(t, err)

	// Verify encrypted file exists
	assert.FileExists(t, outputFile)

	// Verify encrypted file is different from original
	encryptedData, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.NotEqual(t, testData, encryptedData)
	assert.Greater(t, len(encryptedData), len(testData), "Encrypted file should be larger (contains nonce)")
}

// TestDecryptFile_Success tests successful file decryption
func TestDecryptFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	plainFile := filepath.Join(tmpDir, "plain.txt")
	cipherFile := filepath.Join(tmpDir, "cipher.enc")
	decryptedFile := filepath.Join(tmpDir, "decrypted.txt")

	// Create and encrypt test file
	testData := []byte("Secret message for decryption test")
	err := os.WriteFile(plainFile, testData, 0644)
	require.NoError(t, err)

	password := "secure-password"
	err = EncryptFile(password, plainFile, cipherFile)
	require.NoError(t, err)

	// Decrypt the file
	err = DecryptFile(password, cipherFile, decryptedFile)
	assert.NoError(t, err)

	// Verify decrypted content matches original
	decryptedData, err := os.ReadFile(decryptedFile)
	require.NoError(t, err)
	assert.Equal(t, testData, decryptedData)
}

// TestEncryptDecrypt_Roundtrip tests full encryption/decryption cycle
func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		password string
		data     []byte
	}{
		{
			name:     "SimpleText",
			password: "password123",
			data:     []byte("Hello, World!"),
		},
		{
			name:     "EmptyFile",
			password: "empty-pass",
			data:     []byte(""),
		},
		{
			name:     "LargeData",
			password: "large-pass",
			data:     bytes.Repeat([]byte("A"), 10000),
		},
		{
			name:     "BinaryData",
			password: "binary-pass",
			data:     []byte{0x00, 0xFF, 0x01, 0xFE, 0x02, 0xFD},
		},
		{
			name:     "SpecialChars",
			password: "special!@#$%^&*()",
			data:     []byte("Test with special chars: !@#$%^&*()"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plainFile := filepath.Join(tmpDir, tt.name+"-plain.txt")
			cipherFile := filepath.Join(tmpDir, tt.name+"-cipher.enc")
			decryptedFile := filepath.Join(tmpDir, tt.name+"-decrypted.txt")

			// Write original data
			err := os.WriteFile(plainFile, tt.data, 0644)
			require.NoError(t, err)

			// Encrypt
			err = EncryptFile(tt.password, plainFile, cipherFile)
			require.NoError(t, err)

			// Decrypt
			err = DecryptFile(tt.password, cipherFile, decryptedFile)
			require.NoError(t, err)

			// Verify roundtrip
			decryptedData, err := os.ReadFile(decryptedFile)
			require.NoError(t, err)
			assert.Equal(t, tt.data, decryptedData)
		})
	}
}

// TestDecryptFile_WrongPassword tests decryption with incorrect password
func TestDecryptFile_WrongPassword(t *testing.T) {
	tmpDir := t.TempDir()
	plainFile := filepath.Join(tmpDir, "plain.txt")
	cipherFile := filepath.Join(tmpDir, "cipher.enc")
	decryptedFile := filepath.Join(tmpDir, "decrypted.txt")

	// Create and encrypt test file
	testData := []byte("Secret data")
	err := os.WriteFile(plainFile, testData, 0644)
	require.NoError(t, err)

	correctPassword := "correct-password"
	err = EncryptFile(correctPassword, plainFile, cipherFile)
	require.NoError(t, err)

	// Try to decrypt with wrong password
	wrongPassword := "wrong-password"
	err = DecryptFile(wrongPassword, cipherFile, decryptedFile)
	assert.Error(t, err, "Decryption should fail with wrong password")
}

// TestEncryptFile_NonExistentInput tests encryption of non-existent file
func TestEncryptFile_NonExistentInput(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentFile := filepath.Join(tmpDir, "does-not-exist.txt")
	outputFile := filepath.Join(tmpDir, "output.enc")

	err := EncryptFile("password", nonExistentFile, outputFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file")
}

// TestDecryptFile_NonExistentInput tests decryption of non-existent file
func TestDecryptFile_NonExistentInput(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentFile := filepath.Join(tmpDir, "does-not-exist.enc")
	outputFile := filepath.Join(tmpDir, "output.txt")

	err := DecryptFile("password", nonExistentFile, outputFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file")
}

// TestDecryptFile_CorruptedData tests decryption of corrupted ciphertext
func TestDecryptFile_CorruptedData(t *testing.T) {
	tmpDir := t.TempDir()
	cipherFile := filepath.Join(tmpDir, "corrupted.enc")
	outputFile := filepath.Join(tmpDir, "output.txt")

	// Create a file with random data (not properly encrypted)
	corruptedData := []byte("This is not properly encrypted data")
	err := os.WriteFile(cipherFile, corruptedData, 0644)
	require.NoError(t, err)

	// Try to decrypt corrupted data
	err = DecryptFile("password", cipherFile, outputFile)
	assert.Error(t, err, "Decryption should fail with corrupted data")
}

// TestDecryptFile_TooShortCiphertext tests decryption with ciphertext shorter than nonce
func TestDecryptFile_TooShortCiphertext(t *testing.T) {
	tmpDir := t.TempDir()
	cipherFile := filepath.Join(tmpDir, "short.enc")
	outputFile := filepath.Join(tmpDir, "output.txt")

	// Create a file with data shorter than nonce size (12 bytes for GCM)
	shortData := []byte("short")
	err := os.WriteFile(cipherFile, shortData, 0644)
	require.NoError(t, err)

	// Try to decrypt too-short data
	err = DecryptFile("password", cipherFile, outputFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ciphertext too short")
}

// TestEncryptFile_DifferentPasswords tests that different passwords produce different ciphertext
func TestEncryptFile_DifferentPasswords(t *testing.T) {
	tmpDir := t.TempDir()
	plainFile := filepath.Join(tmpDir, "plain.txt")

	testData := []byte("Same data, different passwords")
	err := os.WriteFile(plainFile, testData, 0644)
	require.NoError(t, err)

	// Encrypt with first password
	cipher1 := filepath.Join(tmpDir, "cipher1.enc")
	err = EncryptFile("password1", plainFile, cipher1)
	require.NoError(t, err)

	// Encrypt with second password
	cipher2 := filepath.Join(tmpDir, "cipher2.enc")
	err = EncryptFile("password2", plainFile, cipher2)
	require.NoError(t, err)

	// Read both encrypted files
	data1, err := os.ReadFile(cipher1)
	require.NoError(t, err)
	data2, err := os.ReadFile(cipher2)
	require.NoError(t, err)

	// Verify they're different
	assert.NotEqual(t, data1, data2, "Different passwords should produce different ciphertext")
}

// TestEncryptFile_SamePasswordDifferentNonce tests that same password produces different ciphertext due to random nonce
func TestEncryptFile_SamePasswordDifferentNonce(t *testing.T) {
	tmpDir := t.TempDir()
	plainFile := filepath.Join(tmpDir, "plain.txt")

	testData := []byte("Data encrypted twice")
	err := os.WriteFile(plainFile, testData, 0644)
	require.NoError(t, err)

	password := "same-password"

	// Encrypt twice with same password
	cipher1 := filepath.Join(tmpDir, "cipher1.enc")
	err = EncryptFile(password, plainFile, cipher1)
	require.NoError(t, err)

	cipher2 := filepath.Join(tmpDir, "cipher2.enc")
	err = EncryptFile(password, plainFile, cipher2)
	require.NoError(t, err)

	// Read both encrypted files
	data1, err := os.ReadFile(cipher1)
	require.NoError(t, err)
	data2, err := os.ReadFile(cipher2)
	require.NoError(t, err)

	// Verify they're different (due to random nonce)
	assert.NotEqual(t, data1, data2, "Same password should produce different ciphertext due to random nonce")

	// But both should decrypt to the same original data
	decrypt1 := filepath.Join(tmpDir, "decrypt1.txt")
	err = DecryptFile(password, cipher1, decrypt1)
	require.NoError(t, err)

	decrypt2 := filepath.Join(tmpDir, "decrypt2.txt")
	err = DecryptFile(password, cipher2, decrypt2)
	require.NoError(t, err)

	decData1, err := os.ReadFile(decrypt1)
	require.NoError(t, err)
	decData2, err := os.ReadFile(decrypt2)
	require.NoError(t, err)

	assert.Equal(t, testData, decData1)
	assert.Equal(t, testData, decData2)
}

// TestEncryptFile_EmptyPassword tests encryption with empty password
func TestEncryptFile_EmptyPassword(t *testing.T) {
	tmpDir := t.TempDir()
	plainFile := filepath.Join(tmpDir, "plain.txt")
	cipherFile := filepath.Join(tmpDir, "cipher.enc")
	decryptFile := filepath.Join(tmpDir, "decrypt.txt")

	testData := []byte("Test data with empty password")
	err := os.WriteFile(plainFile, testData, 0644)
	require.NoError(t, err)

	// Encrypt with empty password
	err = EncryptFile("", plainFile, cipherFile)
	assert.NoError(t, err, "Empty password should still work (SHA-256 of empty string)")

	// Decrypt with same empty password
	err = DecryptFile("", cipherFile, decryptFile)
	assert.NoError(t, err)

	decData, err := os.ReadFile(decryptFile)
	require.NoError(t, err)
	assert.Equal(t, testData, decData)
}

// TestEncryptFile_LongPassword tests encryption with very long password
func TestEncryptFile_LongPassword(t *testing.T) {
	tmpDir := t.TempDir()
	plainFile := filepath.Join(tmpDir, "plain.txt")
	cipherFile := filepath.Join(tmpDir, "cipher.enc")
	decryptFile := filepath.Join(tmpDir, "decrypt.txt")

	testData := []byte("Test data")
	err := os.WriteFile(plainFile, testData, 0644)
	require.NoError(t, err)

	// Use very long password (should be hashed to 32 bytes)
	longPassword := strings.Repeat("a", 10000)

	err = EncryptFile(longPassword, plainFile, cipherFile)
	assert.NoError(t, err)

	err = DecryptFile(longPassword, cipherFile, decryptFile)
	assert.NoError(t, err)

	decData, err := os.ReadFile(decryptFile)
	require.NoError(t, err)
	assert.Equal(t, testData, decData)
}

// BenchmarkEncryptFile benchmarks file encryption
func BenchmarkEncryptFile(b *testing.B) {
	tmpDir := b.TempDir()
	plainFile := filepath.Join(tmpDir, "plain.txt")
	testData := bytes.Repeat([]byte("A"), 1024) // 1KB
	os.WriteFile(plainFile, testData, 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cipherFile := filepath.Join(tmpDir, "cipher"+string(rune(i))+".enc")
		_ = EncryptFile("password", plainFile, cipherFile)
	}
}

// BenchmarkDecryptFile benchmarks file decryption
func BenchmarkDecryptFile(b *testing.B) {
	tmpDir := b.TempDir()
	plainFile := filepath.Join(tmpDir, "plain.txt")
	cipherFile := filepath.Join(tmpDir, "cipher.enc")
	testData := bytes.Repeat([]byte("A"), 1024) // 1KB
	os.WriteFile(plainFile, testData, 0644)
	EncryptFile("password", plainFile, cipherFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		decryptFile := filepath.Join(tmpDir, "decrypt"+string(rune(i))+".txt")
		_ = DecryptFile("password", cipherFile, decryptFile)
	}
}

// TestZitiCreateCSR_ErrorPaths tests error handling in CSR creation
func TestZitiCreateCSR_ErrorPaths(t *testing.T) {
	t.Run("invalid private key path", func(t *testing.T) {
		tmpDir := t.TempDir()
		privKeyPath := filepath.Join(tmpDir, "nonexistent", "subdir", "private.pem")
		csrPath := filepath.Join(tmpDir, "csr.pem")

		err := ZitiCreateCSR(privKeyPath, csrPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create private key file")
	})

	t.Run("invalid csr path", func(t *testing.T) {
		tmpDir := t.TempDir()
		privKeyPath := filepath.Join(tmpDir, "private.pem")
		csrPath := filepath.Join(tmpDir, "nonexistent", "subdir", "csr.pem")

		err := ZitiCreateCSR(privKeyPath, csrPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create CSR file")
	})

	t.Run("read-only directory for private key", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "readonly")
		err := os.MkdirAll(subDir, 0755)
		require.NoError(t, err)

		// Make directory read-only
		err = os.Chmod(subDir, 0444)
		require.NoError(t, err)
		defer os.Chmod(subDir, 0755) // Restore for cleanup

		privKeyPath := filepath.Join(subDir, "private.pem")
		csrPath := filepath.Join(tmpDir, "csr.pem")

		err = ZitiCreateCSR(privKeyPath, csrPath)
		assert.Error(t, err)
	})

	t.Run("read-only directory for csr", func(t *testing.T) {
		tmpDir := t.TempDir()
		privKeyPath := filepath.Join(tmpDir, "private.pem")

		subDir := filepath.Join(tmpDir, "readonly")
		err := os.MkdirAll(subDir, 0755)
		require.NoError(t, err)

		// Make directory read-only
		err = os.Chmod(subDir, 0444)
		require.NoError(t, err)
		defer os.Chmod(subDir, 0755) // Restore for cleanup

		csrPath := filepath.Join(subDir, "csr.pem")

		err = ZitiCreateCSR(privKeyPath, csrPath)
		assert.Error(t, err)
	})
}

// TestZitiCreateCSR_FileContents tests the actual contents of generated files
func TestZitiCreateCSR_FileContents(t *testing.T) {
	tmpDir := t.TempDir()
	privKeyPath := filepath.Join(tmpDir, "key.pem")
	csrPath := filepath.Join(tmpDir, "request.pem")

	err := ZitiCreateCSR(privKeyPath, csrPath)
	require.NoError(t, err)

	// Verify private key PEM format
	privKeyData, err := os.ReadFile(privKeyPath)
	require.NoError(t, err)
	assert.Contains(t, string(privKeyData), "BEGIN EC PRIVATE KEY")
	assert.Contains(t, string(privKeyData), "END EC PRIVATE KEY")

	// Verify CSR PEM format
	csrData, err := os.ReadFile(csrPath)
	require.NoError(t, err)
	assert.Contains(t, string(csrData), "BEGIN CERTIFICATE REQUEST")
	assert.Contains(t, string(csrData), "END CERTIFICATE REQUEST")

	// Verify CSR can be parsed
	block, _ := pem.Decode(csrData)
	require.NotNil(t, block)
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	require.NoError(t, err)
	assert.Equal(t, "ziti-edge-router", csr.Subject.CommonName)
	assert.Contains(t, csr.Subject.Organization, "OpenZiti")
}

// TestZitiCreateCSR_MultipleInvocations tests creating multiple CSRs
func TestZitiCreateCSR_MultipleInvocations(t *testing.T) {
	tmpDir := t.TempDir()

	for i := 0; i < 3; i++ {
		privKeyPath := filepath.Join(tmpDir, fmt.Sprintf("key%d.pem", i))
		csrPath := filepath.Join(tmpDir, fmt.Sprintf("csr%d.pem", i))

		err := ZitiCreateCSR(privKeyPath, csrPath)
		require.NoError(t, err)

		assert.FileExists(t, privKeyPath)
		assert.FileExists(t, csrPath)
	}

	// Verify all files were created
	files, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	assert.Len(t, files, 6) // 3 keys + 3 CSRs
}

// TestZitiCreateCSR_FileOverwrite tests overwriting existing files
func TestZitiCreateCSR_FileOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	privKeyPath := filepath.Join(tmpDir, "key.pem")
	csrPath := filepath.Join(tmpDir, "csr.pem")

	// Create first time
	err := ZitiCreateCSR(privKeyPath, csrPath)
	require.NoError(t, err)

	// Read original content
	originalKey, err := os.ReadFile(privKeyPath)
	require.NoError(t, err)

	// Create second time (should overwrite)
	err = ZitiCreateCSR(privKeyPath, csrPath)
	require.NoError(t, err)

	// Verify files still exist and have new content
	newKey, err := os.ReadFile(privKeyPath)
	require.NoError(t, err)

	// Keys should be different (different random generation)
	assert.NotEqual(t, originalKey, newKey)
}
