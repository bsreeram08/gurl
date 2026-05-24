package storage

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/sreeram/gurl/internal/secrets"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/zalando/go-keyring"
)

const collectionPassphraseKeyringService = "gurl"

var errCachedCollectionKeyNotFound = errors.New("cached collection key not found")

type collectionKeyring interface {
	Get(service, user string) (string, error)
	Set(service, user, password string) error
	Delete(service, user string) error
}

type osCollectionKeyring struct{}

func (osCollectionKeyring) Get(service, user string) (string, error) {
	value, err := keyring.Get(service, user)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", errCachedCollectionKeyNotFound
	}
	return value, err
}

func (osCollectionKeyring) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

func (osCollectionKeyring) Delete(service, user string) error {
	err := keyring.Delete(service, user)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

var collectionPassphraseKeyring collectionKeyring = osCollectionKeyring{}

func cacheCollectionPassphraseKey(collectionID string, encryption *types.CollectionEncryption, key []byte) error {
	user, err := collectionPassphraseKeyringUser(collectionID, encryption)
	if err != nil {
		return err
	}
	return collectionPassphraseKeyring.Set(collectionPassphraseKeyringService, user, base64.StdEncoding.EncodeToString(key))
}

func cachedCollectionPassphraseKey(collectionID string, encryption *types.CollectionEncryption) ([]byte, error) {
	user, err := collectionPassphraseKeyringUser(collectionID, encryption)
	if err != nil {
		return nil, err
	}
	value, err := collectionPassphraseKeyring.Get(collectionPassphraseKeyringService, user)
	if err != nil {
		return nil, err
	}
	key, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("cached collection key is invalid: %w", err)
	}
	if len(key) != secrets.KeySize {
		return nil, fmt.Errorf("cached collection key has invalid size")
	}
	return key, nil
}

func deleteCachedCollectionPassphraseKey(collectionID string, encryption *types.CollectionEncryption) error {
	user, err := collectionPassphraseKeyringUser(collectionID, encryption)
	if err != nil {
		return err
	}
	return collectionPassphraseKeyring.Delete(collectionPassphraseKeyringService, user)
}

func collectionPassphraseKeyringUser(collectionID string, encryption *types.CollectionEncryption) (string, error) {
	if collectionID == "" {
		return "", fmt.Errorf("collection ID is required for keychain caching")
	}
	if encryption == nil {
		return "", fmt.Errorf("encryption metadata is missing")
	}
	fingerprintInput := fmt.Sprintf("%s:%s:%d:%d", encryption.Mode, encryption.Salt, encryption.Iterations, encryption.Version)
	fingerprint := sha256.Sum256([]byte(fingerprintInput))
	return "collection-passphrase:" + collectionID + ":" + hex.EncodeToString(fingerprint[:8]), nil
}
