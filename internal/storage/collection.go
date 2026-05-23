package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/syndtr/goleveldb/leveldb"
)

// CollectionStore is implemented by storage backends that support first-class
// collection records in addition to request collection labels.
type CollectionStore interface {
	SaveCollection(collection *types.Collection) error
	GetCollection(id string) (*types.Collection, error)
	GetCollectionByName(name string) (*types.Collection, error)
	ListCollections() ([]*types.Collection, error)
	DeleteCollection(id string) error
	UpdateCollection(collection *types.Collection) error
}

func (db *LMDB) SaveCollection(collection *types.Collection) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	return db.saveCollectionLocked(collection)
}

func (db *LMDB) saveCollectionLocked(collection *types.Collection) error {
	if collection == nil {
		return fmt.Errorf("collection cannot be nil")
	}
	if collection.Name == "" {
		return fmt.Errorf("collection name cannot be empty")
	}

	now := time.Now().Unix()
	if collection.ID == "" {
		nameKey := collectionNameIndexKey(collection.Name)
		if existingID, err := db.DB.Get([]byte(nameKey), nil); err == nil && len(existingID) > 0 {
			return fmt.Errorf("collection %q already exists", collection.Name)
		} else {
			collection.ID = uuid.New().String()
		}
	} else if existingID, err := db.DB.Get([]byte(collectionNameIndexKey(collection.Name)), nil); err == nil && string(existingID) != collection.ID {
		return fmt.Errorf("collection %q already exists", collection.Name)
	}

	if collection.CreatedAt == 0 {
		collection.CreatedAt = now
	}
	if collection.UpdatedAt == 0 {
		collection.UpdatedAt = now
	}
	if collection.Variables == nil {
		collection.Variables = make(map[string]string)
	}
	if collection.SecretKeys == nil {
		collection.SecretKeys = make(map[string]bool)
	}

	stored, err := collectionForDBStorage(collection)
	if err != nil {
		return err
	}
	data, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("failed to marshal collection: %w", err)
	}

	batch := new(leveldb.Batch)
	batch.Put([]byte(collectionKey(collection.ID)), data)
	batch.Put([]byte(collectionNameIndexKey(collection.Name)), []byte(collection.ID))
	return db.DB.Write(batch, nil)
}

func (db *LMDB) GetCollection(id string) (*types.Collection, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	return db.getCollectionLocked(id)
}

func (db *LMDB) getCollectionLocked(id string) (*types.Collection, error) {
	collection, err := db.getRawCollectionLocked(id)
	if err != nil {
		return nil, err
	}
	if err := decryptDBCollectionForUse(collection); err != nil {
		return nil, err
	}
	return collection, nil
}

func (db *LMDB) getRawCollectionLocked(id string) (*types.Collection, error) {
	data, err := db.DB.Get([]byte(collectionKey(id)), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("collection not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	var collection types.Collection
	if err := json.Unmarshal(data, &collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal collection: %w", err)
	}
	if collection.Variables == nil {
		collection.Variables = make(map[string]string)
	}
	if collection.SecretKeys == nil {
		collection.SecretKeys = make(map[string]bool)
	}
	return &collection, nil
}

func (db *LMDB) GetCollectionByName(name string) (*types.Collection, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	idData, err := db.DB.Get([]byte(collectionNameIndexKey(name)), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("collection not found: %s", name)
		}
		return nil, fmt.Errorf("failed to look up collection name index: %w", err)
	}
	return db.getCollectionLocked(string(idData))
}

func (db *LMDB) ListCollections() ([]*types.Collection, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	iter := db.DB.NewIterator(nil, nil)
	defer iter.Release()

	var collections []*types.Collection
	prefix := "collection:"
	for iter.Seek([]byte(prefix)); iter.Valid(); iter.Next() {
		key := string(iter.Key())
		if len(key) < len(prefix) || key[:len(prefix)] != prefix {
			break
		}

		var collection types.Collection
		if err := json.Unmarshal(iter.Value(), &collection); err != nil {
			continue
		}
		if collection.Variables == nil {
			collection.Variables = make(map[string]string)
		}
		if collection.SecretKeys == nil {
			collection.SecretKeys = make(map[string]bool)
		}
		if err := decryptDBCollectionForUse(&collection); err != nil {
			return nil, err
		}
		collections = append(collections, &collection)
	}
	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	sort.SliceStable(collections, func(i, j int) bool {
		return collections[i].Name < collections[j].Name
	})
	return collections, nil
}

func (db *LMDB) DeleteCollection(id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	collection, err := db.getRawCollectionLocked(id)
	if err != nil {
		return err
	}

	batch := new(leveldb.Batch)
	batch.Delete([]byte(collectionKey(id)))
	batch.Delete([]byte(collectionNameIndexKey(collection.Name)))
	if err := db.DB.Write(batch, nil); err != nil {
		return err
	}
	if err := removeCollectionLocalKey(id); err != nil {
		return fmt.Errorf("failed to delete collection key: %w", err)
	}
	return nil
}

func (db *LMDB) UpdateCollection(collection *types.Collection) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if collection == nil {
		return fmt.Errorf("collection cannot be nil")
	}
	if collection.ID == "" {
		return fmt.Errorf("cannot update collection without ID")
	}

	existing, err := db.getCollectionLocked(collection.ID)
	if err != nil {
		return err
	}
	if existing.Name != collection.Name {
		if existingID, err := db.DB.Get([]byte(collectionNameIndexKey(collection.Name)), nil); err == nil && string(existingID) != collection.ID {
			return fmt.Errorf("collection %q already exists", collection.Name)
		}
	}
	collection.CreatedAt = existing.CreatedAt
	if collection.UpdatedAt == 0 || collection.UpdatedAt == existing.UpdatedAt {
		collection.UpdatedAt = time.Now().Unix()
	}

	stored, err := collectionForDBStorage(collection)
	if err != nil {
		return err
	}
	data, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("failed to marshal collection: %w", err)
	}

	batch := new(leveldb.Batch)
	batch.Put([]byte(collectionKey(collection.ID)), data)
	if existing.Name != collection.Name {
		batch.Delete([]byte(collectionNameIndexKey(existing.Name)))
	}
	batch.Put([]byte(collectionNameIndexKey(collection.Name)), []byte(collection.ID))
	return db.DB.Write(batch, nil)
}

func (db *LMDB) ensureCollectionBatch(batch *leveldb.Batch, name string) error {
	if name == "" {
		return nil
	}
	if _, err := db.DB.Get([]byte(collectionNameIndexKey(name)), nil); err == nil {
		return nil
	} else if err != leveldb.ErrNotFound {
		return fmt.Errorf("failed to look up collection %q: %w", name, err)
	}

	collection := types.NewCollection(name)
	data, err := json.Marshal(collection)
	if err != nil {
		return fmt.Errorf("failed to marshal collection: %w", err)
	}
	batch.Put([]byte(collectionKey(collection.ID)), data)
	batch.Put([]byte(collectionNameIndexKey(collection.Name)), []byte(collection.ID))
	return nil
}

func collectionForDBStorage(collection *types.Collection) (*types.Collection, error) {
	if !collectionHasSecrets(collection) {
		return collectionForLocalKeyStorage(collection, nil)
	}
	key, err := getOrCreateCollectionLocalKey(collection.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection key: %w", err)
	}
	return collectionForLocalKeyStorage(collection, key)
}

func decryptDBCollectionForUse(collection *types.Collection) error {
	if collection == nil || !collectionHasSecrets(collection) {
		return nil
	}

	hasEncrypted := false
	for key, isSecret := range collection.SecretKeys {
		if isSecret && IsCollectionEncryptedValue(collection.Variables[key]) {
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
			Hint: fmt.Sprintf("run 'gurl collection import ... --passphrase ...' to bind it locally"),
		}
	}

	key, err := readCollectionLocalKey(collection.ID)
	if err != nil {
		if os.IsNotExist(err) {
			return &CollectionLockedError{
				Name: collection.Name,
				Hint: fmt.Sprintf("missing local collection key; restore ~/.local/share/gurl/keys/%s.key or re-import", safePathComponent(collection.ID)),
			}
		}
		return err
	}
	return decryptCollectionSecrets(collection, key)
}

func collectionKey(id string) string {
	return fmt.Sprintf("collection:%s", id)
}

func collectionNameIndexKey(name string) string {
	return fmt.Sprintf("idx:collection:name:%s", name)
}

var _ CollectionStore = (*LMDB)(nil)
