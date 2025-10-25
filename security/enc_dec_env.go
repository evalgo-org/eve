/*
Package security provides simple AES-256-GCM file encryption and decryption utilities.

This package allows encrypting and decrypting files using a password-derived key.
The password is hashed with SHA-256 to derive a 32-byte key suitable for AES-256.
It uses AES in Galois/Counter Mode (GCM) to provide both confidentiality and integrity.

Usage Example:

    err := security.EncryptFile("mysecret", "plain.txt", "cipher.enc")
    if err != nil {
        log.Fatal(err)
    }

    err = security.DecryptFile("mysecret", "cipher.enc", "plain.txt")
    if err != nil {
        log.Fatal(err)
    }

The resulting ciphertext file contains both the nonce and the encrypted data.
The nonce is randomly generated for each encryption operation.
*/

package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"os"
)

// EncryptFile encrypts a plaintext file to a ciphertext file using AES-256-GCM.
//
// The encryption key is derived from the provided password using SHA-256,
// producing a 32-byte key for AES-256. A random nonce is generated and
// prepended to the ciphertext. The resulting data is written to outputPath.
//
// Parameters:
//   - pass:       Password used to derive the encryption key.
//   - inputPath:  Path to the plaintext file to encrypt.
//   - outputPath: Path to write the resulting ciphertext file.
//
// Returns an error if reading, encryption, or writing fails.
func EncryptFile(pass, inputPath, outputPath string) error {
	key := sha256.Sum256([]byte(pass)) // 32 bytes = AES-256 key
	plaintext, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return os.WriteFile(outputPath, ciphertext, 0600)
}

// DecryptFile decrypts a ciphertext file to a plaintext file using AES-256-GCM.
//
// The decryption key is derived from the provided password using SHA-256.
// The ciphertext file must contain a prepended nonce (generated during encryption).
// The function verifies authenticity and integrity during decryption.
//
// Parameters:
//   - pass:       Password used to derive the decryption key.
//   - inputPath:  Path to the ciphertext file to decrypt.
//   - outputPath: Path to write the decrypted plaintext file.
//
// Returns an error if reading, decryption, authentication, or writing fails.
func DecryptFile(pass, inputPath, outputPath string) error {
	key := sha256.Sum256([]byte(pass)) // 32 bytes = AES-256 key
	ciphertext, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return errors.New("ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ct, nil)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, plaintext, 0600)
}
