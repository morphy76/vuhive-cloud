package outbound

// Encryptor defines the driven port for symmetric encryption and decryption of secret values.
type Encryptor interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}
