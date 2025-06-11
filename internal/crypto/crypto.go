package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

// GetEncryptionKeyFromEnv retrieves the encryption key from the ENCRYPTION_KEY environment variable
// and decodes it from Base64.
func GetEncryptionKeyFromEnv() ([]byte, error) {
	keyBase64 := os.Getenv("ENCRYPTION_KEY")
	if keyBase64 == "" {
		return nil, errors.New("ENCRYPTION_KEY environment variable not set")
	}
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ENCRYPTION_KEY from base64: %w", err)
	}
	if len(key) != 32 { // AES-256 requires a 32-byte key
		return nil, errors.New("ENCRYPTION_KEY must be a 32-byte key (AES-256), after base64 decoding")
	}
	return key, nil
}

// Encrypt encrypts plaintext using AES-256-CBC with the given key.
// 1. Generate a random 16-byte IV.
// 2. Encrypt the plainText using AES-256-CBC with the key and IV.
// 3. Convert IV to Hex.
// 4. Convert ciphertext to Hex.
// 5. Concatenate IV_HEX + CIPHERTEXT_HEX.
// 6. Base64 encode the concatenated string.
// 7. Return the Base64 encoded string.
func Encrypt(plainText string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", errors.New("encryption key must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Pad plaintext to be a multiple of the block size (PKCS#7)
	plainTextBytes := []byte(plainText)
	padding := aes.BlockSize - len(plainTextBytes)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	plainTextBytes = append(plainTextBytes, padtext...)

	// Generate a random IV (Initialization Vector)
	iv := make([]byte, aes.BlockSize) // AES block size is 16 bytes
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("failed to generate IV: %w", err)
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	cipherTextBytes := make([]byte, len(plainTextBytes))
	mode.CryptBlocks(cipherTextBytes, plainTextBytes)

	ivHex := hex.EncodeToString(iv)
	cipherTextHex := hex.EncodeToString(cipherTextBytes)

	combined := ivHex + cipherTextHex
	encoded := base64.StdEncoding.EncodeToString([]byte(combined))

	return encoded, nil
}

// Decrypt decrypts a Base64 encoded string (IV_HEX + CIPHERTEXT_HEX) using AES-256-CBC.
// 1. Decode the cipherTextBase64 from Base64.
// 2. Extract IV_HEX (first 32 chars) and CIPHERTEXT_HEX.
// 3. Convert IV_HEX to bytes.
// 4. Convert CIPHERTEXT_HEX to bytes.
// 5. Decrypt using AES-256-CBC with the key and IV bytes.
// 6. Remove PKCS#7 padding.
// 7. Return the plain text.
func Decrypt(cipherTextBase64 string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", errors.New("decryption key must be 32 bytes for AES-256")
	}

	combinedBytes, err := base64.StdEncoding.DecodeString(cipherTextBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 input: %w", err)
	}
	combined := string(combinedBytes)

	if len(combined) < 32 { // IV_HEX is 32 characters (16 bytes)
		return "", errors.New("invalid ciphertext: too short to contain IV_HEX")
	}

	ivHex := combined[:32]
	cipherTextHex := combined[32:]

	iv, err := hex.DecodeString(ivHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode IV from hex: %w", err)
	}
	if len(iv) != aes.BlockSize {
		return "", errors.New("decoded IV length is not equal to AES block size")
	}

	cipherTextBytes, err := hex.DecodeString(cipherTextHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext from hex: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	if len(cipherTextBytes)%aes.BlockSize != 0 {
		return "", errors.New("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	plainTextBytes := make([]byte, len(cipherTextBytes))
	mode.CryptBlocks(plainTextBytes, cipherTextBytes)

	// Remove PKCS#7 padding
	if len(plainTextBytes) == 0 {
		return "", errors.New("decrypted plaintext is empty")
	}
	padding := int(plainTextBytes[len(plainTextBytes)-1])
	if padding > aes.BlockSize || padding == 0 {
		// This might indicate a decryption error or invalid padding
		return "", errors.New("invalid PKCS#7 padding")
	}
    // Verify padding bytes
    for i := len(plainTextBytes) - padding; i < len(plainTextBytes)-1; i++ {
        if plainTextBytes[i] != byte(padding) {
            return "", errors.New("invalid PKCS#7 padding bytes")
        }
    }
	if len(plainTextBytes) < padding {
		return "", errors.New("plaintext too short for claimed padding")
	}
	plainTextBytes = plainTextBytes[:len(plainTextBytes)-padding]

	return string(plainTextBytes), nil
}
