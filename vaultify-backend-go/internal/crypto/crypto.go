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
)

const (
	// AES-256 key length
	keyLength = 32
	// AES block size
	aesBlockSize = 16
	// IV length (same as AES block size for CBC)
	ivLength = 16
	// IV hex length (16 bytes * 2 characters per byte)
	ivHexLength = 32
)

// pkcs7Pad pads data to be a multiple of blockSize using PKCS#7 padding.
func pkcs7Pad(data []byte, blockSize int) ([]byte, error) {
	if blockSize <= 0 {
		return nil, errors.New("block size must be positive")
	}
	if blockSize > 255 {
		return nil, errors.New("block size cannot be greater than 255")
	}
	padding := blockSize - (len(data) % blockSize)
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...), nil
}

// pkcs7Unpad removes PKCS#7 padding from data.
func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if blockSize <= 0 {
		return nil, errors.New("block size must be positive")
	}
	if len(data) == 0 {
		return nil, errors.New("data cannot be empty")
	}
	if len(data)%blockSize != 0 {
		return nil, errors.New("data length is not a multiple of block size")
	}

	padding := int(data[len(data)-1])
	if padding == 0 || padding > blockSize {
		return nil, errors.New("invalid pkcs7 padding: padding size is zero or exceeds block size")
	}

	// Check that all padding bytes are correct
	for i := 0; i < padding; i++ {
		if data[len(data)-padding+i] != byte(padding) {
			return nil, errors.New("invalid pkcs7 padding: padding bytes are inconsistent")
		}
	}

	return data[:len(data)-padding], nil
}

// Encrypt encrypts plaintext using AES-256-CBC, then hex encodes IV and ciphertext,
// concatenates them, and finally Base64 encodes the result.
func Encrypt(plainText string, key []byte) (string, error) {
	if len(key) != keyLength {
		return "", fmt.Errorf("invalid key length: must be %d bytes for AES-256", keyLength)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	iv := make([]byte, ivLength)
	if _, err := rand.Read(iv); err != nil {
		return "", fmt.Errorf("failed to generate IV: %w", err)
	}

	plainTextBytes := []byte(plainText)
	paddedPlaintext, err := pkcs7Pad(plainTextBytes, aesBlockSize)
	if err != nil {
		return "", fmt.Errorf("failed to pad plaintext: %w", err)
	}

	cipherText := make([]byte, len(paddedPlaintext))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherText, paddedPlaintext)

	ivHex := hex.EncodeToString(iv)
	cipherTextHex := hex.EncodeToString(cipherText)

	// Concatenate IV_HEX + CIPHERTEXT_HEX
	combined := ivHex + cipherTextHex

	// Encode the concatenated string to Base64
	encoded := base64.StdEncoding.EncodeToString([]byte(combined))

	return encoded, nil
}

// Decrypt decrypts a Base64 encoded string (containing hex IV + hex ciphertext) using AES-256-CBC.
func Decrypt(cipherTextBase64 string, key []byte) (string, error) {
	if len(key) != keyLength {
		return "", fmt.Errorf("invalid key length: must be %d bytes for AES-256", keyLength)
	}

	decodedBase64, err := base64.StdEncoding.DecodeString(cipherTextBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	combinedHex := string(decodedBase64)
	if len(combinedHex) < ivHexLength {
		return "", errors.New("invalid ciphertext: too short to contain IV")
	}

	ivHex := combinedHex[:ivHexLength]
	cipherTextHex := combinedHex[ivHexLength:]

	iv, err := hex.DecodeString(ivHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode IV from hex: %w", err)
	}
	if len(iv) != ivLength {
		return "", fmt.Errorf("invalid IV length after hex decoding: expected %d, got %d", ivLength, len(iv))
	}

	cipherText, err := hex.DecodeString(cipherTextHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext from hex: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	if len(cipherText)%aesBlockSize != 0 {
		// This can happen if the ciphertext was truncated or not properly padded.
		return "", errors.New("ciphertext is not a multiple of the block size")
	}

	decryptedPaddedText := make([]byte, len(cipherText))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(decryptedPaddedText, cipherText)

	plainTextBytes, err := pkcs7Unpad(decryptedPaddedText, aesBlockSize)
	if err != nil {
		return "", fmt.Errorf("failed to unpad plaintext: %w", err)
	}

	return string(plainTextBytes), nil
}
