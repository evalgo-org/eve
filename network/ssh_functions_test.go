package network

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"

	"golang.org/x/crypto/ssh"
)

// generateTestRSAKey generates a test RSA private key
func generateTestRSAKey() ([]byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return privateKeyPEM, nil
}

// generateTestSSHCertificate generates a test SSH certificate
func generateTestSSHCertificate(publicKey ssh.PublicKey) ([]byte, error) {
	cert := &ssh.Certificate{
		Key:             publicKey,
		CertType:        ssh.UserCert,
		ValidPrincipals: []string{"testuser"},
		ValidAfter:      0,
		ValidBefore:     ssh.CertTimeInfinity,
	}

	// Sign the certificate (in real use, this would be done by a CA)
	// For testing, we'll create a dummy signer
	testKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.NewSignerFromKey(testKey)
	if err != nil {
		return nil, err
	}

	err = cert.SignCert(rand.Reader, signer)
	if err != nil {
		return nil, err
	}

	return ssh.MarshalAuthorizedKey(cert), nil
}

func TestSshKeyfile_WithPrivateKeyOnly(t *testing.T) {
	// Generate a test RSA key
	privateKeyPEM, err := generateTestRSAKey()
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	// Write key to temporary file
	tmpfile, err := os.CreateTemp("", "testkey-*.pem")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(privateKeyPEM); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// Test ssh_keyfile with only private key
	signer, err := ssh_keyfile(tmpfile.Name(), "")
	if err != nil {
		t.Fatalf("ssh_keyfile failed: %v", err)
	}

	if signer == nil {
		t.Error("Expected non-nil signer")
	}

	// Verify the signer can produce a public key
	pubKey := signer.PublicKey()
	if pubKey == nil {
		t.Error("Expected non-nil public key")
	}
}

func TestSshKeyfile_WithCertificate(t *testing.T) {
	// Generate a test RSA key
	privateKeyPEM, err := generateTestRSAKey()
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	// Parse the private key to get the public key
	signer, err := ssh.ParsePrivateKey(privateKeyPEM)
	if err != nil {
		t.Fatalf("Failed to parse private key: %v", err)
	}

	// Generate a test certificate
	certBytes, err := generateTestSSHCertificate(signer.PublicKey())
	if err != nil {
		t.Fatalf("Failed to generate test certificate: %v", err)
	}

	// Write key and certificate to temporary files
	keyFile, err := os.CreateTemp("", "testkey-*.pem")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(keyFile.Name())

	certFile, err := os.CreateTemp("", "testcert-*.pub")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(certFile.Name())

	if _, err := keyFile.Write(privateKeyPEM); err != nil {
		t.Fatal(err)
	}
	keyFile.Close()

	if _, err := certFile.Write(certBytes); err != nil {
		t.Fatal(err)
	}
	certFile.Close()

	// Test ssh_keyfile with private key and certificate
	certSigner, err := ssh_keyfile(keyFile.Name(), certFile.Name())
	if err != nil {
		t.Fatalf("ssh_keyfile with certificate failed: %v", err)
	}

	if certSigner == nil {
		t.Error("Expected non-nil signer")
	}
}

func TestSshKeyfile_InvalidPrivateKey(t *testing.T) {
	// Create a file with invalid key data
	tmpfile, err := os.CreateTemp("", "invalidkey-*.pem")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	tmpfile.WriteString("invalid key data")
	tmpfile.Close()

	// Test ssh_keyfile with invalid key
	_, err = ssh_keyfile(tmpfile.Name(), "")
	if err == nil {
		t.Error("Expected error for invalid private key")
	}
}

func TestSshKeyfile_NonExistentFile(t *testing.T) {
	// Test ssh_keyfile with non-existent file
	_, err := ssh_keyfile("/nonexistent/key/file", "")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestSignerFromPem_PlainKey(t *testing.T) {
	// Generate a test RSA key
	privateKeyPEM, err := generateTestRSAKey()
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	// Test signerFromPem with plain key
	signer, err := signerFromPem(privateKeyPEM, nil)
	if err != nil {
		t.Fatalf("signerFromPem failed: %v", err)
	}

	if signer == nil {
		t.Error("Expected non-nil signer")
	}

	// Verify the signer can produce a public key
	pubKey := signer.PublicKey()
	if pubKey == nil {
		t.Error("Expected non-nil public key")
	}
}

func TestSignerFromPem_InvalidPEM(t *testing.T) {
	// Test with invalid PEM data
	invalidPEM := []byte("not a valid PEM block")

	_, err := signerFromPem(invalidPEM, nil)
	if err == nil {
		t.Error("Expected error for invalid PEM")
	}

	if err != nil && err.Error() != "PEM decode failed, no key found" {
		t.Errorf("Expected 'PEM decode failed' error, got: %v", err)
	}
}

func TestParsePemBlock_RSA(t *testing.T) {
	// Generate RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	key, err := parsePemBlock(block)
	if err != nil {
		t.Fatalf("parsePemBlock failed for RSA: %v", err)
	}

	if key == nil {
		t.Error("Expected non-nil key")
	}

	_, ok := key.(*rsa.PrivateKey)
	if !ok {
		t.Error("Expected RSA private key")
	}
}

func TestParsePemBlock_UnsupportedType(t *testing.T) {
	block := &pem.Block{
		Type:  "UNSUPPORTED KEY TYPE",
		Bytes: []byte("dummy data"),
	}

	_, err := parsePemBlock(block)
	if err == nil {
		t.Error("Expected error for unsupported key type")
	}

	expectedError := "parsing private key failed, unsupported key type"
	if err != nil && err.Error()[:len(expectedError)] != expectedError {
		t.Errorf("Expected unsupported key type error, got: %v", err)
	}
}

func TestParsePemBlock_InvalidRSAKey(t *testing.T) {
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: []byte("invalid rsa key data"),
	}

	_, err := parsePemBlock(block)
	if err == nil {
		t.Error("Expected error for invalid RSA key data")
	}
}
