// Package security provides utilities for certificate management and security checks.
// It includes functions for generating Certificate Signing Requests (CSRs) for Ziti,
// checking TLS certificate expiration, and validating signature algorithms.
//
// Features:
//   - ECDSA key pair generation
//   - CSR creation for Ziti edge routers
//   - TLS certificate expiration checking
//   - Signature algorithm validation
//   - Support for various cryptographic operations
//
// The package helps ensure secure communications by validating certificates
// and using strong cryptographic algorithms.
package security

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	eve "eve.evalgo.org/common"
)

// Error message constants for certificate checks
const (
	errExpiringShortly = "%s: ** '%s' (S/N %X) expires in %d hours! **"
	errExpiringSoon    = "%s: '%s' (S/N %X) expires in roughly %d days."
	errSunsetAlg       = "%s: '%s' (S/N %X) expires after the sunset date for its signature algorithm '%s'."

	// Verify that non-root certificates are using a good signature algorithm
	checkSigAlg = true
)

// hostResult contains the results of a host certificate check.
// Used to return information about a host's TLS certificates.
type hostResult struct {
	Host       string // The host that was checked
	Err        error  // Any error that occurred during checking
	CommonName string // The common name from the certificate
}

// sigAlgSunset contains information about signature algorithm sunsets.
// Used to track when cryptographic algorithms become deprecated.
type sigAlgSunset struct {
	name      string    // Human readable name of signature algorithm
	sunsetsAt time.Time // Time the algorithm will be sunset
}

// sunsetSigAlgs maps signature algorithms to their sunset information.
// Contains a list of deprecated or soon-to-be-deprecated signature algorithms.
//
//nolint:gofmt
var sunsetSigAlgs = map[x509.SignatureAlgorithm]sigAlgSunset{
	x509.MD2WithRSA: sigAlgSunset{
		name:      "MD2 with RSA",
		sunsetsAt: time.Now(), // Already deprecated
	},
	x509.MD5WithRSA: sigAlgSunset{
		name:      "MD5 with RSA",
		sunsetsAt: time.Now(), // Already deprecated
	},
	x509.SHA1WithRSA: sigAlgSunset{
		name:      "SHA1 with RSA",
		sunsetsAt: time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC),
	},
	x509.DSAWithSHA1: sigAlgSunset{
		name:      "DSA with SHA1",
		sunsetsAt: time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC),
	},
	x509.ECDSAWithSHA1: sigAlgSunset{
		name:      "ECDSA with SHA1",
		sunsetsAt: time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC),
	},
}

// randReader returns a reader for cryptographic randomness.
// Uses /dev/urandom as a secure source of randomness for cryptographic operations.
//
// Returns:
//   - *os.File: A file handle to /dev/urandom
func randReader() *os.File {
	f, err := os.Open("/dev/urandom")
	if err != nil {
		panic(err)
	}
	return f
}

// ZitiCreateCSR generates a Certificate Signing Request (CSR) and private key for Ziti.
// This function creates an ECDSA key pair and generates a CSR for a Ziti edge router.
//
// Parameters:
//   - privateFilePath: Path to save the generated private key
//   - csrFilePath: Path to save the generated CSR
//
// The function:
//  1. Generates an ECDSA key pair using P-256 curve
//  2. Saves the private key to a PEM file
//  3. Creates a CSR template with appropriate subject information
//  4. Generates the CSR using the private key
//  5. Saves the CSR to a PEM file
//
// The generated CSR will have:
//   - Common Name: "ziti-edge-router"
//   - Organization: "OpenZiti"
//   - Signature Algorithm: ECDSA with SHA-256
func ZitiCreateCSR(privateFilePath, csrFilePath string) error {
	// Step 1: Generate ECDSA key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), randReader())
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	// Save private key to file
	privKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal EC private key: %w", err)
	}

	privKeyFile, err := os.Create(privateFilePath)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer privKeyFile.Close()

	err = pem.Encode(privKeyFile, &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privKeyBytes,
	})
	if err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Step 2: Create CSR template
	subj := pkix.Name{
		CommonName:   "ziti-edge-router",
		Organization: []string{"OpenZiti"},
	}

	csrTemplate := x509.CertificateRequest{
		Subject:            subj,
		SignatureAlgorithm: x509.ECDSAWithSHA256,
	}

	// Step 3: Generate CSR
	csrBytes, err := x509.CreateCertificateRequest(randReader(), &csrTemplate, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create CSR: %w", err)
	}

	// Save CSR to file
	csrFile, err := os.Create(csrFilePath)
	if err != nil {
		return fmt.Errorf("failed to create CSR file: %w", err)
	}
	defer csrFile.Close()

	err = pem.Encode(csrFile, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes})
	if err != nil {
		return fmt.Errorf("failed to write CSR: %w", err)
	}

	eve.Logger.Info("✅ CSR and private key saved to", csrFilePath, "and", privateFilePath)
	return nil
}

// CertsCheckHost checks the TLS certificates of a host for expiration and security issues.
// This function connects to the host, examines its certificate chain, and checks for:
//   - Certificates expiring within the warning period
//   - Use of deprecated signature algorithms
//
// Parameters:
//   - host: The host:port to check (e.g., "example.com:443")
//   - warnYears: Number of years before expiration to start warning
//   - warnMonths: Number of months before expiration to start warning
//   - warnDays: Number of days before expiration to start warning
//
// Returns:
//   - hostResult: A struct containing the host information, any errors found,
//     and the common name from the certificate
//
// The function:
//  1. Establishes a TLS connection to the host
//  2. Examines each certificate in the verified chains
//  3. Checks expiration dates against the warning thresholds
//  4. Validates signature algorithms against known deprecated algorithms
//  5. Returns any issues found
func CertsCheckHost(host string, warnYears, warnMonths, warnDays *int) (result hostResult) {
	result = hostResult{
		Host: host,
	}

	conn, err := tls.Dial("tcp", host, nil)
	if err != nil {
		result.Err = err
		return
	}
	defer conn.Close()

	timeNow := time.Now()
	checkedCerts := make(map[string]struct{})

	for _, chain := range conn.ConnectionState().VerifiedChains {
		for certNum, cert := range chain {
			// Skip certificates we've already checked
			if _, checked := checkedCerts[string(cert.Signature)]; checked {
				continue
			}
			checkedCerts[string(cert.Signature)] = struct{}{}

			// Check the expiration
			warningTime := timeNow.AddDate(*warnYears, *warnMonths, *warnDays)
			if warningTime.After(cert.NotAfter) {
				expiresIn := int64(cert.NotAfter.Sub(timeNow).Hours())
				if expiresIn <= 48 {
					// Certificate expires in less than 48 hours
					result.Err = fmt.Errorf(errExpiringShortly, host, cert.Subject.CommonName, cert.SerialNumber, expiresIn)
				} else {
					// Certificate expires within the warning period
					result.Err = fmt.Errorf(errExpiringSoon, host, cert.Subject.CommonName, cert.SerialNumber, expiresIn/24)
				}
			}

			// Check the signature algorithm (ignoring the root certificate)
			if alg, exists := sunsetSigAlgs[cert.SignatureAlgorithm]; checkSigAlg && exists && certNum != len(chain)-1 {
				if cert.NotAfter.Equal(alg.sunsetsAt) || cert.NotAfter.After(alg.sunsetsAt) {
					result.Err = fmt.Errorf(errSunsetAlg, host, cert.Subject.CommonName, cert.SerialNumber, alg.name)
				}
			}

			// Store the common name from the first certificate
			if result.CommonName == "" {
				result.CommonName = cert.Subject.CommonName
			}
		}
	}

	return
}
