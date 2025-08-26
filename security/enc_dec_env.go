package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"os"
)

// EncryptFile encrypts a plaintext file to a ciphertext file using AES-256-GCM
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
	// GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	// Random nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	// Encrypt
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	// Write ciphertext to file
	return os.WriteFile(outputPath, ciphertext, 0600)
}

// DecryptFile decrypts a ciphertext file to a plaintext file using AES-256-GCM
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
