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

const (
	errExpiringShortly = "%s: ** '%s' (S/N %X) expires in %d hours! **"
	errExpiringSoon    = "%s: '%s' (S/N %X) expires in roughly %d days."
	errSunsetAlg       = "%s: '%s' (S/N %X) expires after the sunset date for its signature algorithm '%s'."
	// Verify that non-root certificates are using a good signature algorithm
	checkSigAlg = true
)

type hostResult struct {
	Host       string
	Err        error
	CommonName string
}

type sigAlgSunset struct {
	name      string    // Human readable name of signature algorithm
	sunsetsAt time.Time // Time the algorithm will be sunset
}

var sunsetSigAlgs = map[x509.SignatureAlgorithm]sigAlgSunset{
	x509.MD2WithRSA: sigAlgSunset{
		name:      "MD2 with RSA",
		sunsetsAt: time.Now(),
	},
	x509.MD5WithRSA: sigAlgSunset{
		name:      "MD5 with RSA",
		sunsetsAt: time.Now(),
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

func randReader() *os.File {
	f, err := os.Open("/dev/urandom")
	if err != nil {
		panic(err)
	}
	return f
}

func ZitiCreateCSR(privateFilePath, csrFilePath string) {
	// Step 1: Generate ECDSA key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), randReader())
	if err != nil {
		eve.Logger.Fatal("Failed to generate key:", err)
	}
	// Save private key to file
	privKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		eve.Logger.Fatal("Failed to marshal EC private key:", err)
	}

	privKeyFile, err := os.Create(privateFilePath)
	if err != nil {
		eve.Logger.Fatal("Failed to create private key file:", err)
	}
	defer privKeyFile.Close()

	err = pem.Encode(privKeyFile, &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privKeyBytes,
	})
	if err != nil {
		eve.Logger.Fatal("Failed to write private key:", err)
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
		eve.Logger.Fatal("Failed to create CSR:", err)
	}
	// Save CSR to file
	csrFile, _ := os.Create(csrFilePath)
	defer csrFile.Close()
	pem.Encode(csrFile, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes})
	eve.Logger.Info("✅ CSR and private key saved to ziti.csr and ziti-key.pem")
}

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
			if _, checked := checkedCerts[string(cert.Signature)]; checked {
				continue
			}
			checkedCerts[string(cert.Signature)] = struct{}{}

			// Check the expiration.
			if timeNow.AddDate(*warnYears, *warnMonths, *warnDays).After(cert.NotAfter) {
				expiresIn := int64(cert.NotAfter.Sub(timeNow).Hours())
				if expiresIn <= 48 {
					result.Err = fmt.Errorf(errExpiringShortly, host, cert.Subject.CommonName, cert.SerialNumber, expiresIn)
				} else {
					result.Err = fmt.Errorf(errExpiringSoon, host, cert.Subject.CommonName, cert.SerialNumber, expiresIn/24)
				}
			}

			// Check the signature algorithm, ignoring the root certificate.
			if alg, exists := sunsetSigAlgs[cert.SignatureAlgorithm]; checkSigAlg && exists && certNum != len(chain)-1 {
				if cert.NotAfter.Equal(alg.sunsetsAt) || cert.NotAfter.After(alg.sunsetsAt) {
					result.Err = fmt.Errorf(errSunsetAlg, host, cert.Subject.CommonName, cert.SerialNumber, alg.name)
				}
			}

			result.CommonName = cert.Subject.CommonName
		}
	}

	return
}
