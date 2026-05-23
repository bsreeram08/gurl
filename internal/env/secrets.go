package env

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/sreeram/gurl/internal/secrets"
)

const (
	keyPathDefault = "~/.local/share/gurl/.secret-key"
	nonceSize      = secrets.NonceSize
	keySize        = secrets.KeySize
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
	return secrets.GetOrCreateKeyAt(keyPath)
}

func EncryptSecret(key []byte, plaintext string) (string, error) {
	return secrets.Encrypt(key, plaintext)
}

func DecryptSecret(key []byte, ciphertext string) (string, error) {
	return secrets.Decrypt(key, ciphertext)
}

func MaskSecret(value string) string {
	return secrets.Mask(value)
}

func IsEncryptedValue(value string) bool {
	return secrets.IsEncryptedValue(value)
}

func EncryptValueIfNeeded(key []byte, value string, isSecret bool) (string, error) {
	return secrets.EncryptValueIfNeeded(key, value, isSecret)
}

func DecryptValueIfNeeded(key []byte, value string, isSecret bool) (string, error) {
	return secrets.DecryptValueIfNeeded(key, value, isSecret)
}
