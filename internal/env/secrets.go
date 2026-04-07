package env

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	keyPathDefault = "~/.local/share/gurl/.secret-key"
	nonceSize      = 12
	keySize        = 32
)

var (
	globalKey     []byte
	globalKeyOnce sync.Once
	globalKeyErr  error
)

func GetOrCreateMachineKey() ([]byte, error) {
	globalKeyOnce.Do(func() {
		globalKey, globalKeyErr = getOrCreateMachineKeyAt(getDefaultKeyPath())
	})
	return globalKey, globalKeyErr
}

func getDefaultKeyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return keyPathDefault
	}
	return filepath.Join(home, ".local", "share", "gurl", ".secret-key")
}

func getOrCreateMachineKeyAt(keyPath string) ([]byte, error) {
	if keyPath == "" {
		keyPath = getDefaultKeyPath()
	}

	dir := filepath.Dir(keyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(keyPath)
	if err == nil && len(data) == keySize {
		return data, nil
	}

	if !errors.Is(err, os.ErrNotExist) && err != nil {
		return nil, err
	}

	key := make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}

	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return nil, err
	}

	return key, nil
}

func EncryptSecret(key []byte, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptSecret(key []byte, ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func MaskSecret(value string) string {
	if value == "" {
		return ""
	}
	return "*****"
}

func IsEncryptedValue(value string) bool {
	if value == "" {
		return false
	}

	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return false
	}

	return len(data) > nonceSize
}

func EncryptValueIfNeeded(key []byte, value string, isSecret bool) (string, error) {
	if !isSecret {
		return value, nil
	}
	return EncryptSecret(key, value)
}

func DecryptValueIfNeeded(key []byte, value string, isSecret bool) (string, error) {
	if !isSecret {
		return value, nil
	}
	return DecryptSecret(key, value)
}
