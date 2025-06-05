package core

import (
	"fmt"
	"vaultify-backend-go/internal/crypto" // To call the actual encryption functions
)

// encryptionService implements the EncryptionService interface.
// It acts as a wrapper around the package-level functions in internal/crypto.
type encryptionService struct {
	// No fields are needed if we are directly calling package-level functions
	// from internal/crypto and the encryption key is passed in each call.
	// If the service were to manage or preload the key from config,
	// it would be a field here. However, the current design implies
	// the calling service (e.g., SecretService) fetches the key from config
	// and passes it to these methods.
}

// NewEncryptionService creates a new EncryptionService instance.
func NewEncryptionService() EncryptionService {
	return &encryptionService{}
}

// Encrypt delegates the encryption task to the crypto package.
func (s *encryptionService) Encrypt(plainText string, key []byte) (string, error) {
	encryptedData, err := crypto.Encrypt(plainText, key)
	if err != nil {
		// It's good practice to wrap errors from lower layers if you want to add context,
		// or if the service layer might handle specific crypto errors differently.
		// For a simple wrapper, direct return is also fine, but wrapping is more robust.
		return "", fmt.Errorf("encryption_service: failed to encrypt: %w", err)
	}
	return encryptedData, nil
}

// Decrypt delegates the decryption task to the crypto package.
func (s *encryptionService) Decrypt(cipherTextBase64 string, key []byte) (string, error) {
	decryptedData, err := crypto.Decrypt(cipherTextBase64, key)
	if err != nil {
		return "", fmt.Errorf("encryption_service: failed to decrypt: %w", err)
	}
	return decryptedData, nil
}
