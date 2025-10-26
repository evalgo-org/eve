// Package network provides utilities for secure network operations, particularly
// SSH connections with certificate-based authentication.
//
// Features:
//   - SSH key and certificate parsing
//   - Support for both regular private keys and certificate-based authentication
//   - SSH command execution on remote hosts
//   - Support for various key types (RSA, EC, DSA)
//   - Handling of encrypted PEM blocks
package network

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	eve "eve.evalgo.org/common"
	"golang.org/x/crypto/ssh"
)

// ssh_keyfile creates an SSH signer from a private key file and optional certificate.
// This function reads both the private key and certificate files, then creates
// a certificate signer that combines both for SSH authentication.
//
// Parameters:
//   - privateKeyPath: Path to the private key file
//   - certKeyPath: Path to the certificate key file (can be empty if no certificate)
//
// Returns:
//   - ssh.Signer: A signer that can be used for SSH authentication
//   - error: If any step in the process fails (file reading, parsing, etc.)
//
// The function:
//  1. Reads the private key file
//  2. Parses the private key
//  3. Reads the certificate file (if provided)
//  4. Parses the certificate
//  5. Creates and returns a certificate signer combining both
func ssh_keyfile(privateKeyPath string, certKeyPath string) (ssh.Signer, error) {
	// Parse the user's private key
	pvtKeyBts, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(pvtKeyBts)
	if err != nil {
		return nil, err
	}

	// If no certificate path provided, return just the private key signer
	if certKeyPath == "" {
		return signer, nil
	}

	// Parse the user's certificate
	certBts, err := os.ReadFile(certKeyPath)
	if err != nil {
		return nil, err
	}

	cert, _, _, _, err := ssh.ParseAuthorizedKey(certBts)
	if err != nil {
		return nil, err
	}

	// Create a signer using both the certificate and the private key
	return ssh.NewCertSigner(cert.(*ssh.Certificate), signer)
}

// signerFromPem creates an SSH signer from a PEM-encoded private key.
// This function handles both encrypted and unencrypted PEM blocks.
//
// Parameters:
//   - pemBytes: The PEM-encoded private key bytes
//   - password: Password for decrypting encrypted keys (can be empty)
//
// Returns:
//   - ssh.Signer: A signer that can be used for SSH authentication
//   - error: If any step in the process fails
//
// The function:
//  1. Decodes the PEM block
//  2. Handles encrypted keys by decrypting them
//  3. Parses the key based on its type (RSA, EC, DSA)
//  4. Creates and returns an SSH signer from the key
//
// nolint:unused
func signerFromPem(pemBytes []byte, password []byte) (ssh.Signer, error) {
	// Read PEM block
	pemBlock, _ := pem.Decode(pemBytes)
	if pemBlock == nil {
		return nil, errors.New("PEM decode failed, no key found")
	}

	// Handle encrypted key
	if x509.IsEncryptedPEMBlock(pemBlock) { //nolint:staticcheck // legacy PEM encryption support needed
		// Decrypt PEM
		var err error
		pemBlock.Bytes, err = x509.DecryptPEMBlock(pemBlock, password) //nolint:staticcheck // legacy PEM encryption support needed
		if err != nil {
			return nil, fmt.Errorf("decrypting PEM block failed: %v", err)
		}

		// Get RSA, EC or DSA key
		key, err := parsePemBlock(pemBlock)
		if err != nil {
			return nil, err
		}

		// Generate signer instance from key
		signer, err := ssh.NewSignerFromKey(key)
		if err != nil {
			return nil, fmt.Errorf("creating signer from encrypted key failed: %v", err)
		}
		return signer, nil
	}

	// Generate signer instance from plain key
	signer, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing plain private key failed: %v", err)
	}
	return signer, nil
}

// parsePemBlock parses a PEM block into a crypto key based on its type.
// This is a helper function for signerFromPem that handles different key types.
//
// Parameters:
//   - block: The PEM block to parse
//
// Returns:
//   - interface{}: The parsed key (crypto.PrivateKey)
//   - error: If parsing fails
//
// Supported key types:
//   - RSA PRIVATE KEY
//   - EC PRIVATE KEY
//   - DSA PRIVATE KEY
//
// nolint:unused
func parsePemBlock(block *pem.Block) (interface{}, error) {
	switch block.Type {
	case "RSA PRIVATE KEY":
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parsing PKCS private key failed: %v", err)
		}
		return key, nil

	case "EC PRIVATE KEY":
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parsing EC private key failed: %v", err)
		}
		return key, nil

	case "DSA PRIVATE KEY":
		key, err := ssh.ParseDSAPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parsing DSA private key failed: %v", err)
		}
		return key, nil

	default:
		return nil, fmt.Errorf("parsing private key failed, unsupported key type %q", block.Type)
	}
}

// SshExec executes a command on a remote host via SSH.
// This function establishes an SSH connection to the specified host and
// executes the given command, returning its output.
//
// Parameters:
//   - address: The remote host address in "host:port" format
//   - username: The username for SSH authentication
//   - keyfile: Path to the private key file
//   - certfile: Path to the certificate file (optional, can be empty)
//   - cmd: The command to execute on the remote host
//
// The function:
//  1. Creates an SSH signer from the provided key and certificate files
//  2. Configures the SSH client with the signer
//  3. Establishes a connection to the remote host
//  4. Creates a new session
//  5. Executes the command and captures its output
//  6. Prints the command output to stdout
//
// Note: This function uses InsecureIgnoreHostKey() for host key verification,
// which is not secure for production use. In production, you should implement
// proper host key verification.
func SshExec(address string, username string, keyfile string, certfile string, cmd string) {
	var signer ssh.Signer
	var err error

	// Create signer from key files
	signer, err = ssh_keyfile(keyfile, certfile)
	if err != nil {
		eve.Logger.Fatal("Failed to create signer: ", err)
	}

	// Configure SSH client
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to the remote host
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		eve.Logger.Fatal("Failed to dial: ", err)
	}
	defer client.Close()

	// Create a new session
	session, err := client.NewSession()
	if err != nil {
		eve.Logger.Fatal("Failed to create session: ", err)
	}
	defer session.Close()

	// Execute the command and capture output
	out, err := session.CombinedOutput(cmd)
	if err != nil {
		eve.Logger.Fatal("Command execution failed: ", err)
	}

	// Print the command output
	fmt.Println(string(out))
}
