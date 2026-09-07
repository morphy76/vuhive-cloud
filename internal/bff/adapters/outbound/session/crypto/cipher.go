package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
)

// KeySize represents the required key length for AES-256 (32 bytes).
const KeySize = 32

// TokenCipher provides authenticated AES-256-GCM symmetric encryption for OAuth tokens.
type TokenCipher struct {
	aead cipher.AEAD
}

// NewTokenCipher constructs a TokenCipher using a 32-byte secret key.
func NewTokenCipher(key []byte) (*TokenCipher, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("%w: cipher key must be exactly %d bytes, got %d", model.ErrInvalidParameter, KeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: failed creating aes cipher: %v", model.ErrInternal, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: failed creating gcm: %v", model.ErrInternal, err)
	}

	return &TokenCipher{aead: gcm}, nil
}

// NewTokenCipherFromPassphrase derives a 32-byte AES-256 key from an arbitrary string using SHA-256.
func NewTokenCipherFromPassphrase(passphrase string) (*TokenCipher, error) {
	if passphrase == "" {
		return nil, fmt.Errorf("%w: passphrase cannot be empty", model.ErrInvalidParameter)
	}
	hash := sha256.Sum256([]byte(passphrase))
	return NewTokenCipher(hash[:])
}

// Encrypt encrypts a plaintext string with AES-256-GCM and returns a base64-encoded ciphertext.
func (c *TokenCipher) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("%w: failed generating random nonce: %v", model.ErrInternal, err)
	}

	ciphertext := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decodes and decrypts an AES-256-GCM base64 ciphertext, verifying the authentication tag.
func (c *TokenCipher) Decrypt(encodedCiphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encodedCiphertext)
	if err != nil {
		return "", fmt.Errorf("%w: invalid base64 encoding: %v", model.ErrInvalidParameter, err)
	}

	nonceSize := c.aead.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("%w: ciphertext is too short", model.ErrInvalidParameter)
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("%w: decryption authentication failed: %v", model.ErrUnauthorized, err)
	}

	return string(plaintext), nil
}
