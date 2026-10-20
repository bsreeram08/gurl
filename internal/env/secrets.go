package env

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/sreeram/gurl/internal/secrets"
)

const keyPathDefault = "~/.local/share/gurl/.secret-key"

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
