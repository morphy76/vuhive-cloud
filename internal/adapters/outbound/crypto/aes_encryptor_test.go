package crypto_test

import (
	"crypto/rand"
	"testing"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/crypto"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ outbound.Encryptor = (*crypto.AESEncryptor)(nil)

func randomKey(t *testing.T, size int) []byte {
	t.Helper()
	k := make([]byte, size)
	_, err := rand.Read(k)
	require.NoError(t, err)
	return k
}

func TestNewAESEncryptor(t *testing.T) {
	t.Run("valid 32-byte key succeeds", func(t *testing.T) {
		key := randomKey(t, 32)
		enc, err := crypto.NewAESEncryptor(key)
		require.NoError(t, err)
		assert.NotNil(t, enc)
	})

	t.Run("invalid key lengths fail", func(t *testing.T) {
		tests := []struct {
			name string
			key  []byte
		}{
			{"nil key", nil},
			{"empty key", []byte{}},
			{"16-byte key", randomKey(t, 16)},
			{"24-byte key", randomKey(t, 24)},
			{"31-byte key", randomKey(t, 31)},
			{"33-byte key", randomKey(t, 33)},
			{"64-byte key", randomKey(t, 64)},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				enc, err := crypto.NewAESEncryptor(tc.key)
				assert.Error(t, err)
				assert.ErrorIs(t, err, crypto.ErrInvalidKeyLength)
				assert.Nil(t, enc)
			})
		}
	})
}

func TestAESEncryptor_EncryptDecrypt(t *testing.T) {
	key := randomKey(t, 32)
	enc, err := crypto.NewAESEncryptor(key)
	require.NoError(t, err)

	t.Run("roundtrip with arbitrary plaintext", func(t *testing.T) {
		plaintext := []byte("vuhive-cloud-secret-token-value-12345!@#$%^&*()")

		ciphertext, err := enc.Encrypt(plaintext)
		require.NoError(t, err)
		assert.NotEmpty(t, ciphertext)

		decrypted, err := enc.Decrypt(ciphertext)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("roundtrip with empty plaintext", func(t *testing.T) {
		plaintext := []byte("")

		ciphertext, err := enc.Encrypt(plaintext)
		require.NoError(t, err)
		assert.NotEmpty(t, ciphertext)

		decrypted, err := enc.Decrypt(ciphertext)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("encrypted output is different from plaintext", func(t *testing.T) {
		plaintext := []byte("database-password-must-be-hidden")

		ciphertext, err := enc.Encrypt(plaintext)
		require.NoError(t, err)
		assert.NotEqual(t, plaintext, ciphertext)
	})

	t.Run("encrypted output has minimum length (nonce + ciphertext + tag)", func(t *testing.T) {
		// Nonce (12 bytes) + tag (16 bytes) = 28 bytes overhead minimum
		const minOverhead = 12 + 16

		plaintext := []byte("sample-data")
		ciphertext, err := enc.Encrypt(plaintext)
		require.NoError(t, err)

		expectedLength := minOverhead + len(plaintext)
		assert.Equal(t, expectedLength, len(ciphertext))
		assert.GreaterOrEqual(t, len(ciphertext), minOverhead)
	})

	t.Run("randomized nonce produces distinct ciphertexts", func(t *testing.T) {
		plaintext := []byte("constant-plaintext-payload")

		c1, err := enc.Encrypt(plaintext)
		require.NoError(t, err)

		c2, err := enc.Encrypt(plaintext)
		require.NoError(t, err)

		assert.NotEqual(t, c1, c2)

		d1, err := enc.Decrypt(c1)
		require.NoError(t, err)
		d2, err := enc.Decrypt(c2)
		require.NoError(t, err)

		assert.Equal(t, plaintext, d1)
		assert.Equal(t, plaintext, d2)
	})
}

func TestAESEncryptor_Decrypt_WrongKey(t *testing.T) {
	key1 := randomKey(t, 32)
	key2 := randomKey(t, 32)

	enc1, err := crypto.NewAESEncryptor(key1)
	require.NoError(t, err)

	enc2, err := crypto.NewAESEncryptor(key2)
	require.NoError(t, err)

	plaintext := []byte("confidential-payload")
	ciphertext, err := enc1.Encrypt(plaintext)
	require.NoError(t, err)

	decrypted, err := enc2.Decrypt(ciphertext)
	assert.Error(t, err)
	assert.Nil(t, decrypted)
}

func TestAESEncryptor_Decrypt_CorruptedData(t *testing.T) {
	key := randomKey(t, 32)
	enc, err := crypto.NewAESEncryptor(key)
	require.NoError(t, err)

	plaintext := []byte("sensitive-api-token")
	ciphertext, err := enc.Encrypt(plaintext)
	require.NoError(t, err)

	t.Run("tampered ciphertext body fails", func(t *testing.T) {
		corrupted := make([]byte, len(ciphertext))
		copy(corrupted, ciphertext)
		// Flip a bit in the ciphertext / tag region
		corrupted[len(corrupted)-1] ^= 0xFF

		decrypted, err := enc.Decrypt(corrupted)
		assert.Error(t, err)
		assert.Nil(t, decrypted)
	})

	t.Run("tampered nonce fails", func(t *testing.T) {
		corrupted := make([]byte, len(ciphertext))
		copy(corrupted, ciphertext)
		// Flip a bit in the nonce region (first 12 bytes)
		corrupted[0] ^= 0xFF

		decrypted, err := enc.Decrypt(corrupted)
		assert.Error(t, err)
		assert.Nil(t, decrypted)
	})

	t.Run("truncated ciphertext shorter than nonce fails", func(t *testing.T) {
		shortData := []byte("short")
		decrypted, err := enc.Decrypt(shortData)
		assert.Error(t, err)
		assert.Nil(t, decrypted)
	})

	t.Run("empty ciphertext fails", func(t *testing.T) {
		decrypted, err := enc.Decrypt([]byte{})
		assert.Error(t, err)
		assert.Nil(t, decrypted)
	})
}
