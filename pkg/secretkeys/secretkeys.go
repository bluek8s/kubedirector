package secretkeys

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"io"
)

// Encrypt calculates AES encrypted value for given text
// with configured master encryption key
// Due to random nonce the function is not idempotent
func Encrypt(
	text string,
) (string, error) {
	plainText := []byte(text)
	aesgcm, err := aesGCM()
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aesgcm.Seal(nonce, nonce, plainText, nil)
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt decrypts AES encrypted value back to text
func Decrypt(
	encryptedText string,
) (string, error) {
	enc, err := hex.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}
	aesgcm, err := aesGCM()
	if err != nil {
		return "", err
	}
	nonceSize := aesgcm.NonceSize()
	nonce, ciphertext := enc[:nonceSize], enc[nonceSize:]
	result, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func aesGCM() (cipher.AEAD, error) {
	masterEncryptionKey, err := shared.GetMasterEncryptionKey()
	if err != nil {
		return nil, err
	}
	aesBlock, err := aesBlock(masterEncryptionKey)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return nil, err
	}
	return aesGCM, nil
}

func aesBlock(
	masterEncryptionKey string,
) (cipher.Block, error) {
	aesKey, err := hex.DecodeString(masterEncryptionKey)
	if err != nil {
		return nil, err
	}
	aesBlock, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	return aesBlock, nil
}

// GenerateEncryptionKey generates 32-byte encryption key
// For AES encryption 32-byte key selects AES-256 mode
func GenerateEncryptionKey() string {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		panic(fmt.Sprintf("crypto/rand failure %v", err))
	}
	return hex.EncodeToString(key)
}

// ValidateEncryptionKey check if encryption key has size of 16, 24 or 32 bytes
// Otherwise it cannot be used as encryption key for AES
func ValidateEncryptionKey(
	encryptionKey string,
) error {
	_, err := aesBlock(encryptionKey)
	return err
}
