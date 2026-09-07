package crypto_test

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/session/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenCipher_NewTokenCipher(t *testing.T) {
	t.Run("valid 32-byte key", func(t *testing.T) {
		key := make([]byte, 32)
		_, err := rand.Read(key)
		require.NoError(t, err)

		cipher, err := crypto.NewTokenCipher(key)
		require.NoError(t, err)
		assert.NotNil(t, cipher)
	})

	t.Run("invalid key lengths", func(t *testing.T) {
		tests := []struct {
			name string
			key  []byte
		}{
			{"empty key", []byte{}},
			{"16-byte key", make([]byte, 16)},
			{"24-byte key", make([]byte, 24)},
			{"31-byte key", make([]byte, 31)},
			{"33-byte key", make([]byte, 33)},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				cipher, err := crypto.NewTokenCipher(tc.key)
				assert.Error(t, err)
				assert.Nil(t, cipher)
			})
		}
	})

	t.Run("passphrase key derivation", func(t *testing.T) {
		cipher, err := crypto.NewTokenCipherFromPassphrase("super-secret-passphrase-for-vuhive")
		require.NoError(t, err)
		assert.NotNil(t, cipher)

		_, err = crypto.NewTokenCipherFromPassphrase("")
		assert.Error(t, err)
	})
}

func TestTokenCipher_EncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	cipher, err := crypto.NewTokenCipher(key)
	require.NoError(t, err)

	t.Run("successful roundtrip", func(t *testing.T) {
		plaintext := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0"

		encrypted, err := cipher.Encrypt(plaintext)
		require.NoError(t, err)
		assert.NotEmpty(t, encrypted)
		assert.NotEqual(t, plaintext, encrypted)

		decrypted, err := cipher.Decrypt(encrypted)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("empty plaintext roundtrip", func(t *testing.T) {
		encrypted, err := cipher.Encrypt("")
		require.NoError(t, err)

		decrypted, err := cipher.Decrypt(encrypted)
		require.NoError(t, err)
		assert.Equal(t, "", decrypted)
	})

	t.Run("randomized nonce produces distinct ciphertexts", func(t *testing.T) {
		plaintext := "constant-token-value"

		enc1, err := cipher.Encrypt(plaintext)
		require.NoError(t, err)
		enc2, err := cipher.Encrypt(plaintext)
		require.NoError(t, err)

		assert.NotEqual(t, enc1, enc2)

		dec1, err := cipher.Decrypt(enc1)
		require.NoError(t, err)
		dec2, err := cipher.Decrypt(enc2)
		require.NoError(t, err)

		assert.Equal(t, plaintext, dec1)
		assert.Equal(t, plaintext, dec2)
	})

	t.Run("corrupted base64 returns error", func(t *testing.T) {
		_, err := cipher.Decrypt("not-valid-base64!!!")
		assert.Error(t, err)
	})

	t.Run("tampered ciphertext returns error", func(t *testing.T) {
		encrypted, err := cipher.Encrypt("sensitive-data")
		require.NoError(t, err)

		raw, err := base64.StdEncoding.DecodeString(encrypted)
		require.NoError(t, err)

		// Tamper with the last byte (auth tag)
		raw[len(raw)-1] ^= 0xFF
		tampered := base64.StdEncoding.EncodeToString(raw)

		_, err = cipher.Decrypt(tampered)
		assert.Error(t, err)
	})

	t.Run("truncated ciphertext returns error", func(t *testing.T) {
		// Shorter than 12-byte GCM nonce
		shortPayload := base64.StdEncoding.EncodeToString([]byte("short"))
		_, err := cipher.Decrypt(shortPayload)
		assert.Error(t, err)
	})
}
