// vault provides AES-256-GCM encryption/decryption for aster modules.
// The key is assembled at runtime from 4 obfuscated fragments (see derive.go).
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
)

var (
	initOnce sync.Once
	masterKey []byte // 32-byte AES-256 key, derived on first use

	ErrInvalidFormat = errors.New("vault: invalid encrypted asset format")
	ErrDecryption    = errors.New("vault: decryption failed")
)

// Key returns the 32-byte master AES key, deriving it on first call.
func Key() []byte {
	initOnce.Do(func() {
		masterKey = deriveKey()
	})
	return masterKey
}

// Encrypt encrypts plaintext with AES-256-GCM using a random nonce.
// Returns: nonce(12) || ciphertext(tag appended).
func Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(Key())
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts ciphertext produced by Encrypt.
// Expects: nonce(12) || ciphertext(tag appended).
func Decrypt(data []byte) ([]byte, error) {
	key := Key()
	if len(key) == 0 {
		return nil, ErrDecryption
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, ErrInvalidFormat
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryption, err)
	}
	return plaintext, nil
}
