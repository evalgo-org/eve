package security

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"os"
	"path/filepath"
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
	ZitiCreateCSR(privKeyPath, csrPath)

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
		ZitiCreateCSR(privPath, csrPath)
		os.Remove(privPath)
		os.Remove(csrPath)
	}
}
