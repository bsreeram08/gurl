package storage

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sreeram/gurl/internal/secrets"
	"github.com/sreeram/gurl/pkg/types"
)

const (
	CollectionEncryptionModeLocal      = "local"
	CollectionEncryptionModePassphrase = "passphrase"
	CollectionEncryptionKDF            = "pbkdf2-sha256"
	CollectionEncryptionIterations     = 210000
	collectionKeyFileName              = "collection.key"
	passphraseSaltSize                 = 16
)

func IsCollectionEncryptedValue(value string) bool {
	return secrets.IsEncryptedValue(value)
}

type CollectionLockedError struct {
	Name string
	Hint string
}

func (e *CollectionLockedError) Error() string {
	if e.Hint == "" {
		return fmt.Sprintf("collection %q has locked secrets", e.Name)
	}
	return fmt.Sprintf("collection %q has locked secrets: %s", e.Name, e.Hint)
}

func IsCollectionLocked(err error) bool {
	var locked *CollectionLockedError
	return errors.As(err, &locked)
}

func (s *FileStore) collectionForStorage(collection *types.Collection, dir string) (*types.Collection, error) {
	stored := cloneCollectionForStorage(collection)
	if !collectionHasSecrets(stored) {
		stored.Encryption = nil
		return stored, nil
	}
	if stored.Encryption != nil && stored.Encryption.Mode == CollectionEncryptionModePassphrase {
		for name, isSecret := range stored.SecretKeys {
			if isSecret && secrets.IsEncryptedValue(stored.Variables[name]) {
				return nil, fmt.Errorf("collection %q is passphrase-locked; run collection unlock first", stored.Name)
			}
		}
	}

	key, err := s.getOrCreateCollectionKey(dir)
	if err != nil {
		return nil, err
	}
	if err := encryptCollectionSecrets(stored, key); err != nil {
		return nil, err
	}
	stored.Encryption = &types.CollectionEncryption{
		Version: 1,
		Mode:    CollectionEncryptionModeLocal,
	}
	return stored, nil
}

func (s *FileStore) decryptCollectionForUse(collection *types.Collection, dir string) error {
	if collection == nil || !collectionHasSecrets(collection) {
		return nil
	}

	hasEncrypted := false
	for key, isSecret := range collection.SecretKeys {
		if isSecret && secrets.IsEncryptedValue(collection.Variables[key]) {
			hasEncrypted = true
			break
		}
	}
	if !hasEncrypted {
		return nil
	}

	if collection.Encryption != nil && collection.Encryption.Mode == CollectionEncryptionModePassphrase {
		return &CollectionLockedError{
			Name: collection.Name,
			Hint: fmt.Sprintf("run 'gurl collection unlock %s --passphrase ...' before using it", collection.Name),
		}
	}

	key, err := s.readCollectionKey(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return &CollectionLockedError{
				Name: collection.Name,
				Hint: "missing collection.key; restore the local key or re-import a passphrase-protected export",
			}
		}
		return err
	}
	return decryptCollectionSecrets(collection, key)
}

func (s *FileStore) UnlockCollection(name string, passphrase string) error {
	if passphrase == "" {
		return fmt.Errorf("passphrase is required")
	}
	collection, dir, err := s.findRawCollectionByName(name)
	if err != nil {
		return fmt.Errorf("collection not found: %s", name)
	}
	if collection.Encryption == nil || collection.Encryption.Mode != CollectionEncryptionModePassphrase {
		return fmt.Errorf("collection %q is not passphrase-locked", name)
	}

	key, err := passphraseKey(collection.Encryption, passphrase)
	if err != nil {
		return err
	}
	if err := decryptCollectionSecrets(collection, key); err != nil {
		return fmt.Errorf("failed to unlock collection %q: %w", name, err)
	}
	collection.Encryption = nil

	localKeyPath := collectionKeyPath(dir)
	if _, err := secrets.GetOrCreateKeyAt(localKeyPath); err != nil {
		return err
	}
	return s.saveCollection(collection, true)
}

func (s *FileStore) getOrCreateCollectionKey(dir string) ([]byte, error) {
	return secrets.GetOrCreateKeyAt(collectionKeyPath(dir))
}

func (s *FileStore) readCollectionKey(dir string) ([]byte, error) {
	key, err := os.ReadFile(collectionKeyPath(dir))
	if err != nil {
		return nil, err
	}
	if len(key) != secrets.KeySize {
		return nil, fmt.Errorf("invalid collection key size")
	}
	return key, nil
}

func collectionKeyPath(dir string) string {
	return filepath.Join(dir, collectionKeyFileName)
}

func encryptCollectionSecrets(collection *types.Collection, key []byte) error {
	for name, isSecret := range collection.SecretKeys {
		if !isSecret {
			continue
		}
		value := collection.Variables[name]
		if value == "" || secrets.IsEncryptedValue(value) {
			continue
		}
		encrypted, err := secrets.Encrypt(key, value)
		if err != nil {
			return err
		}
		collection.Variables[name] = encrypted
	}
	return nil
}

func decryptCollectionSecrets(collection *types.Collection, key []byte) error {
	for name, isSecret := range collection.SecretKeys {
		if !isSecret {
			continue
		}
		value := collection.Variables[name]
		if value == "" || !secrets.IsEncryptedValue(value) {
			continue
		}
		decrypted, err := secrets.Decrypt(key, value)
		if err != nil {
			return err
		}
		collection.Variables[name] = decrypted
	}
	return nil
}

func collectionHasSecrets(collection *types.Collection) bool {
	if collection == nil || len(collection.SecretKeys) == 0 {
		return false
	}
	for _, isSecret := range collection.SecretKeys {
		if isSecret {
			return true
		}
	}
	return false
}

func cloneCollectionForStorage(source *types.Collection) *types.Collection {
	if source == nil {
		return nil
	}
	clone := *source
	clone.Variables = cloneStringMap(source.Variables)
	clone.SecretKeys = cloneBoolMap(source.SecretKeys)
	if source.Encryption != nil {
		encryption := *source.Encryption
		clone.Encryption = &encryption
	}
	return &clone
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func cloneBoolMap(source map[string]bool) map[string]bool {
	if source == nil {
		return nil
	}
	clone := make(map[string]bool, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func encryptCollectionForPassphrase(collection *types.Collection, passphrase string) (*types.Collection, error) {
	exported := cloneCollectionForStorage(collection)
	if exported == nil || !collectionHasSecrets(exported) {
		return exported, nil
	}
	if passphrase == "" {
		return nil, fmt.Errorf("passphrase is required to export collection secrets")
	}

	for name, isSecret := range exported.SecretKeys {
		if isSecret && secrets.IsEncryptedValue(exported.Variables[name]) {
			return nil, fmt.Errorf("collection %q has locked secrets; run collection unlock first", exported.Name)
		}
	}

	salt, err := secrets.RandomBytes(passphraseSaltSize)
	if err != nil {
		return nil, err
	}
	encryption := &types.CollectionEncryption{
		Version:    1,
		Mode:       CollectionEncryptionModePassphrase,
		KDF:        CollectionEncryptionKDF,
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Iterations: CollectionEncryptionIterations,
	}
	key, err := passphraseKey(encryption, passphrase)
	if err != nil {
		return nil, err
	}
	if err := encryptCollectionSecrets(exported, key); err != nil {
		return nil, err
	}
	exported.Encryption = encryption
	return exported, nil
}

func decryptCollectionFromPassphrase(collection *types.Collection, passphrase string) (*types.Collection, error) {
	imported := cloneCollectionForStorage(collection)
	if imported == nil || imported.Encryption == nil || imported.Encryption.Mode != CollectionEncryptionModePassphrase {
		return imported, nil
	}
	if passphrase == "" {
		return nil, fmt.Errorf("passphrase is required to import collection secrets")
	}
	key, err := passphraseKey(imported.Encryption, passphrase)
	if err != nil {
		return nil, err
	}
	if err := decryptCollectionSecrets(imported, key); err != nil {
		return nil, fmt.Errorf("failed to decrypt collection secrets: %w", err)
	}
	imported.Encryption = nil
	return imported, nil
}

func passphraseKey(encryption *types.CollectionEncryption, passphrase string) ([]byte, error) {
	if encryption == nil {
		return nil, fmt.Errorf("encryption metadata is missing")
	}
	if encryption.KDF != "" && encryption.KDF != CollectionEncryptionKDF {
		return nil, fmt.Errorf("unsupported collection key derivation: %s", encryption.KDF)
	}
	salt, err := base64.StdEncoding.DecodeString(encryption.Salt)
	if err != nil {
		return nil, fmt.Errorf("invalid collection encryption salt: %w", err)
	}
	iterations := encryption.Iterations
	if iterations <= 0 {
		iterations = CollectionEncryptionIterations
	}
	return secrets.PBKDF2SHA256(passphrase, salt, iterations, secrets.KeySize), nil
}
