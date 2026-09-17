package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/rs/zerolog/log"
)

// KeySize represents the required key length in bytes for AES-256.
const KeySize = 32

var (
	// ErrInvalidKeyLength is returned when the encryption key is not exactly 32 bytes.
	ErrInvalidKeyLength = errors.New("key must be exactly 32 bytes")

	// ErrCiphertextTooShort is returned when ciphertext is shorter than the minimum expected length (nonce + tag).
	ErrCiphertextTooShort = errors.New("ciphertext too short")

	// ErrDecryptionFailed is returned when authentication tag verification or decryption fails.
	ErrDecryptionFailed = errors.New("decryption failed")
)

// AESEncryptor implements outbound.Encryptor using AES-256-GCM authenticated encryption.
type AESEncryptor struct {
	key []byte
}

// Compile-time interface assertion
var _ outbound.Encryptor = (*AESEncryptor)(nil)

// NewAESEncryptor creates a new AESEncryptor with the provided 32-byte key.
func NewAESEncryptor(key []byte) (*AESEncryptor, error) {
	start := time.Now()
	log.Debug().Int("key_len", len(key)).Msg("NewAESEncryptor entered")

	if len(key) != KeySize {
		err := fmt.Errorf("%w: expected %d bytes, got %d", ErrInvalidKeyLength, KeySize, len(key))
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("invalid key length")
		return nil, err
	}

	keyCopy := make([]byte, KeySize)
	copy(keyCopy, key)

	log.Info().Dur("duration_ms", time.Since(start)).Msg("NewAESEncryptor completed")
	return &AESEncryptor{key: keyCopy}, nil
}

// Encrypt encrypts the plaintext using AES-256-GCM and prepends a randomized nonce.
func (e *AESEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	start := time.Now()
	log.Debug().Int("plaintext_len", len(plaintext)).Msg("AESEncryptor.Encrypt entered")

	block, err := aes.NewCipher(e.key)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to create AES cipher")
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to create GCM AEAD")
		return nil, fmt.Errorf("failed to create GCM AEAD: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to generate random nonce")
		return nil, fmt.Errorf("failed to generate random nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	log.Info().
		Int("ciphertext_len", len(ciphertext)).
		Dur("duration_ms", time.Since(start)).
		Msg("AESEncryptor.Encrypt completed")
	return ciphertext, nil
}

// Decrypt splits the nonce from the ciphertext and decrypts the payload using AES-256-GCM.
func (e *AESEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	start := time.Now()
	log.Debug().Int("ciphertext_len", len(ciphertext)).Msg("AESEncryptor.Decrypt entered")

	block, err := aes.NewCipher(e.key)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to create AES cipher")
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to create GCM AEAD")
		return nil, fmt.Errorf("failed to create GCM AEAD: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize+gcm.Overhead() {
		err := ErrCiphertextTooShort
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("ciphertext too short")
		return nil, err
	}

	nonce := ciphertext[:nonceSize]
	enc := ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, enc, nil)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to decrypt ciphertext")
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}
	if plaintext == nil {
		plaintext = []byte{}
	}

	log.Info().
		Int("plaintext_len", len(plaintext)).
		Dur("duration_ms", time.Since(start)).
		Msg("AESEncryptor.Decrypt completed")
	return plaintext, nil
}
