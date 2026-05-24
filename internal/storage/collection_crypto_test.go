package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/project"
	"github.com/sreeram/gurl/internal/secrets"
	"github.com/sreeram/gurl/pkg/types"
)

type memoryCollectionKeyring struct {
	values map[string]string
}

func (k *memoryCollectionKeyring) Get(service, user string) (string, error) {
	value, ok := k.values[service+"\x00"+user]
	if !ok {
		return "", errCachedCollectionKeyNotFound
	}
	return value, nil
}

func (k *memoryCollectionKeyring) Set(service, user, password string) error {
	k.values[service+"\x00"+user] = password
	return nil
}

func (k *memoryCollectionKeyring) Delete(service, user string) error {
	delete(k.values, service+"\x00"+user)
	return nil
}

func withMemoryCollectionKeyring(t *testing.T) *memoryCollectionKeyring {
	t.Helper()
	previous := collectionPassphraseKeyring
	memory := &memoryCollectionKeyring{values: make(map[string]string)}
	collectionPassphraseKeyring = memory
	t.Cleanup(func() {
		collectionPassphraseKeyring = previous
	})
	return memory
}

type failingSetCollectionKeyring struct {
	err error
}

func (k failingSetCollectionKeyring) Get(service, user string) (string, error) {
	return "", errCachedCollectionKeyNotFound
}

func (k failingSetCollectionKeyring) Set(service, user, password string) error {
	return k.err
}

func (k failingSetCollectionKeyring) Delete(service, user string) error {
	return nil
}

func withFailingSetCollectionKeyring(t *testing.T, err error) {
	t.Helper()
	previous := collectionPassphraseKeyring
	collectionPassphraseKeyring = failingSetCollectionKeyring{err: err}
	t.Cleanup(func() {
		collectionPassphraseKeyring = previous
	})
}

func TestFileStoreEncryptsCollectionSecretsAtRest(t *testing.T) {
	proj, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	store := NewFileStore(proj)

	collection := types.NewCollection("payments")
	collection.SetVariable("BASE_URL", "https://api.example.com")
	collection.SetSecretVariable("API_KEY", "secret-token")
	if err := store.SaveCollection(collection); err != nil {
		t.Fatalf("SaveCollection failed: %v", err)
	}

	collectionPath, err := store.CollectionPath("payments")
	if err != nil {
		t.Fatalf("CollectionPath failed: %v", err)
	}
	rawData, err := os.ReadFile(filepath.Join(collectionPath, collectionFileName))
	if err != nil {
		t.Fatalf("failed to read collection file: %v", err)
	}
	var raw types.Collection
	if err := json.Unmarshal(rawData, &raw); err != nil {
		t.Fatalf("failed to unmarshal raw collection: %v", err)
	}
	if raw.Variables["API_KEY"] == "secret-token" {
		t.Fatal("expected secret variable to be encrypted at rest")
	}
	if !IsCollectionEncryptedValue(raw.Variables["API_KEY"]) {
		t.Fatalf("expected encrypted value marker, got %q", raw.Variables["API_KEY"])
	}
	if raw.Encryption == nil || raw.Encryption.Mode != CollectionEncryptionModeLocal {
		t.Fatalf("expected local encryption metadata, got %+v", raw.Encryption)
	}
	if _, err := os.Stat(filepath.Join(collectionPath, collectionKeyFileName)); err != nil {
		t.Fatalf("expected local collection key file: %v", err)
	}

	loaded, err := store.GetCollectionByName("payments")
	if err != nil {
		t.Fatalf("GetCollectionByName failed: %v", err)
	}
	if loaded.Variables["API_KEY"] != "secret-token" {
		t.Fatalf("expected decrypted secret, got %q", loaded.Variables["API_KEY"])
	}
	if loaded.Variables["BASE_URL"] != "https://api.example.com" {
		t.Fatalf("expected non-secret variable to stay readable")
	}
}

func TestFileStoreEncryptsPrefixedPlaintextSecrets(t *testing.T) {
	proj, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	store := NewFileStore(proj)

	foreignKey, err := secrets.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	prefixedPlaintext, err := secrets.Encrypt(foreignKey, "foreign-secret")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	collection := types.NewCollection("prefixes")
	collection.SetSecretVariable("TOKEN", prefixedPlaintext)
	if err := store.SaveCollection(collection); err != nil {
		t.Fatalf("SaveCollection failed: %v", err)
	}

	collectionPath, err := store.CollectionPath("prefixes")
	if err != nil {
		t.Fatalf("CollectionPath failed: %v", err)
	}
	rawData, err := os.ReadFile(filepath.Join(collectionPath, collectionFileName))
	if err != nil {
		t.Fatalf("failed to read collection file: %v", err)
	}
	if strings.Contains(string(rawData), prefixedPlaintext) {
		t.Fatal("prefixed plaintext secret should be encrypted at rest")
	}
	loaded, err := store.GetCollectionByName("prefixes")
	if err != nil {
		t.Fatalf("GetCollectionByName failed: %v", err)
	}
	if loaded.Variables["TOKEN"] != prefixedPlaintext {
		t.Fatalf("expected prefixed plaintext to round trip, got %q", loaded.Variables["TOKEN"])
	}
}

func TestCollectionExportEncryptsSecretsWithPassphrase(t *testing.T) {
	collection := types.NewCollection("payments")
	collection.SetSecretVariable("API_KEY", "secret-token")
	request := &types.SavedRequest{
		ID:         "req-1",
		Name:       "list charges",
		URL:        "https://example.com/charges",
		Collection: "payments",
	}

	exportData, err := BuildCollectionExport(collection, []*types.SavedRequest{request}, "correct horse")
	if err != nil {
		t.Fatalf("BuildCollectionExport failed: %v", err)
	}
	if exportData.Collection.Variables["API_KEY"] == "secret-token" {
		t.Fatal("expected export secret to be encrypted")
	}
	if exportData.Collection.Encryption == nil || exportData.Collection.Encryption.Mode != CollectionEncryptionModePassphrase {
		t.Fatalf("expected passphrase encryption metadata, got %+v", exportData.Collection.Encryption)
	}
	if exportData.Collection.Encryption.KDF != CollectionEncryptionKDF {
		t.Fatalf("expected PBKDF2 metadata, got %+v", exportData.Collection.Encryption)
	}

	data, err := MarshalCollectionExport(exportData)
	if err != nil {
		t.Fatalf("MarshalCollectionExport failed: %v", err)
	}
	if strings.Contains(string(data), "secret-token") {
		t.Fatal("export should not contain plaintext secret")
	}
	if _, _, err := ParseCollectionExport(data, "wrong passphrase"); err == nil {
		t.Fatal("expected wrong passphrase to fail")
	}

	imported, requests, err := ParseCollectionExport(data, "correct horse")
	if err != nil {
		t.Fatalf("ParseCollectionExport failed: %v", err)
	}
	if imported.Variables["API_KEY"] != "secret-token" {
		t.Fatalf("expected decrypted secret, got %q", imported.Variables["API_KEY"])
	}
	if imported.Encryption != nil {
		t.Fatalf("expected imported collection to be ready for local re-encryption, got %+v", imported.Encryption)
	}
	if len(requests) != 1 || requests[0].Collection != "payments" {
		t.Fatalf("expected request collection metadata, got %+v", requests)
	}
}

func TestFileStoreUnlockCachesPassphraseKey(t *testing.T) {
	withMemoryCollectionKeyring(t)

	proj, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	store := NewFileStore(proj)

	collection := types.NewCollection("shared")
	collection.SetSecretVariable("TOKEN", "shared-secret")
	exportData, err := BuildCollectionExport(collection, nil, "team-pass")
	if err != nil {
		t.Fatalf("BuildCollectionExport failed: %v", err)
	}

	collectionPath, err := store.CollectionPath("shared")
	if err != nil {
		t.Fatalf("CollectionPath failed: %v", err)
	}
	if err := os.MkdirAll(collectionPath, 0755); err != nil {
		t.Fatalf("failed to create collection dir: %v", err)
	}
	raw, err := json.MarshalIndent(exportData.Collection, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal passphrase collection: %v", err)
	}
	if err := os.WriteFile(filepath.Join(collectionPath, collectionFileName), raw, 0644); err != nil {
		t.Fatalf("failed to write collection file: %v", err)
	}

	if _, err := store.GetCollectionByName("shared"); !IsCollectionLocked(err) {
		t.Fatalf("expected collection to start locked, got %v", err)
	}

	if err := store.UnlockCollection("shared", "team-pass"); err != nil {
		t.Fatalf("UnlockCollection failed: %v", err)
	}
	unlocked, err := store.GetCollectionByName("shared")
	if err != nil {
		t.Fatalf("GetCollectionByName after unlock failed: %v", err)
	}
	if unlocked.Variables["TOKEN"] != "shared-secret" {
		t.Fatalf("expected decrypted secret after unlock, got %q", unlocked.Variables["TOKEN"])
	}
	if _, err := os.Stat(filepath.Join(collectionPath, collectionKeyFileName)); !os.IsNotExist(err) {
		t.Fatalf("unlock should cache passphrase key without writing local key, got %v", err)
	}

	rawData, err := os.ReadFile(filepath.Join(collectionPath, collectionFileName))
	if err != nil {
		t.Fatalf("failed to read unlocked collection file: %v", err)
	}
	if strings.Contains(string(rawData), "shared-secret") {
		t.Fatal("unlocked collection should remain encrypted at rest")
	}
	var rawCollection types.Collection
	if err := json.Unmarshal(rawData, &rawCollection); err != nil {
		t.Fatalf("failed to unmarshal collection file: %v", err)
	}
	if rawCollection.Encryption == nil || rawCollection.Encryption.Mode != CollectionEncryptionModePassphrase {
		t.Fatalf("expected passphrase encryption metadata to remain, got %+v", rawCollection.Encryption)
	}
}

func TestFileStoreSavesUnlockedPassphraseCollectionWithCachedKey(t *testing.T) {
	withMemoryCollectionKeyring(t)

	proj, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	store := NewFileStore(proj)

	collection := types.NewCollection("shared")
	collection.SetSecretVariable("TOKEN", "shared-secret")
	exportData, err := BuildCollectionExport(collection, nil, "team-pass")
	if err != nil {
		t.Fatalf("BuildCollectionExport failed: %v", err)
	}

	collectionPath, err := store.CollectionPath("shared")
	if err != nil {
		t.Fatalf("CollectionPath failed: %v", err)
	}
	if err := os.MkdirAll(collectionPath, 0755); err != nil {
		t.Fatalf("failed to create collection dir: %v", err)
	}
	raw, err := json.MarshalIndent(exportData.Collection, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal passphrase collection: %v", err)
	}
	if err := os.WriteFile(filepath.Join(collectionPath, collectionFileName), raw, 0644); err != nil {
		t.Fatalf("failed to write collection file: %v", err)
	}

	if err := store.UnlockCollection("shared", "team-pass"); err != nil {
		t.Fatalf("UnlockCollection failed: %v", err)
	}
	unlocked, err := store.GetCollectionByName("shared")
	if err != nil {
		t.Fatalf("GetCollectionByName after unlock failed: %v", err)
	}
	unlocked.SetSecretVariable("TOKEN", "updated-secret")
	if err := store.SaveCollection(unlocked); err != nil {
		t.Fatalf("SaveCollection failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(collectionPath, collectionKeyFileName)); !os.IsNotExist(err) {
		t.Fatalf("passphrase collection save should not write local key, got %v", err)
	}
	reloaded, err := store.GetCollectionByName("shared")
	if err != nil {
		t.Fatalf("GetCollectionByName after save failed: %v", err)
	}
	if reloaded.Variables["TOKEN"] != "updated-secret" {
		t.Fatalf("expected updated secret, got %q", reloaded.Variables["TOKEN"])
	}
	rawData, err := os.ReadFile(filepath.Join(collectionPath, collectionFileName))
	if err != nil {
		t.Fatalf("failed to read saved collection file: %v", err)
	}
	if strings.Contains(string(rawData), "updated-secret") {
		t.Fatal("saved passphrase collection should remain encrypted at rest")
	}
	var rawCollection types.Collection
	if err := json.Unmarshal(rawData, &rawCollection); err != nil {
		t.Fatalf("failed to unmarshal saved collection: %v", err)
	}
	if rawCollection.Encryption == nil || rawCollection.Encryption.Mode != CollectionEncryptionModePassphrase {
		t.Fatalf("expected passphrase encryption metadata after save, got %+v", rawCollection.Encryption)
	}
}

func TestFileStoreUnlockFallsBackToLocalKeyWhenKeychainUnavailable(t *testing.T) {
	withFailingSetCollectionKeyring(t, errors.New("secret service unavailable"))

	proj, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	store := NewFileStore(proj)

	collection := types.NewCollection("shared")
	collection.SetSecretVariable("TOKEN", "shared-secret")
	exportData, err := BuildCollectionExport(collection, nil, "team-pass")
	if err != nil {
		t.Fatalf("BuildCollectionExport failed: %v", err)
	}

	collectionPath, err := store.CollectionPath("shared")
	if err != nil {
		t.Fatalf("CollectionPath failed: %v", err)
	}
	if err := os.MkdirAll(collectionPath, 0755); err != nil {
		t.Fatalf("failed to create collection dir: %v", err)
	}
	raw, err := json.MarshalIndent(exportData.Collection, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal passphrase collection: %v", err)
	}
	if err := os.WriteFile(filepath.Join(collectionPath, collectionFileName), raw, 0644); err != nil {
		t.Fatalf("failed to write collection file: %v", err)
	}

	if err := store.UnlockCollection("shared", "team-pass"); err != nil {
		t.Fatalf("UnlockCollection should fall back to local key: %v", err)
	}
	if _, err := os.Stat(filepath.Join(collectionPath, collectionKeyFileName)); err != nil {
		t.Fatalf("expected local key fallback after keychain failure: %v", err)
	}
	unlocked, err := store.GetCollectionByName("shared")
	if err != nil {
		t.Fatalf("GetCollectionByName after fallback unlock failed: %v", err)
	}
	if unlocked.Variables["TOKEN"] != "shared-secret" {
		t.Fatalf("expected decrypted fallback secret, got %q", unlocked.Variables["TOKEN"])
	}
	rawData, err := os.ReadFile(filepath.Join(collectionPath, collectionFileName))
	if err != nil {
		t.Fatalf("failed to read fallback collection file: %v", err)
	}
	if strings.Contains(string(rawData), "shared-secret") {
		t.Fatal("fallback collection should remain encrypted at rest")
	}
	var rawCollection types.Collection
	if err := json.Unmarshal(rawData, &rawCollection); err != nil {
		t.Fatalf("failed to unmarshal fallback collection: %v", err)
	}
	if rawCollection.Encryption == nil || rawCollection.Encryption.Mode != CollectionEncryptionModeLocal {
		t.Fatalf("expected local encryption metadata after fallback, got %+v", rawCollection.Encryption)
	}
}

func TestFileStoreMissingLocalCollectionKeyFailsLocked(t *testing.T) {
	proj, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	store := NewFileStore(proj)

	collection := types.NewCollection("cloned")
	collection.SetSecretVariable("TOKEN", "local-secret")
	if err := store.SaveCollection(collection); err != nil {
		t.Fatalf("SaveCollection failed: %v", err)
	}
	if err := store.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "uses secret",
		URL:        "https://example.com/{{TOKEN}}",
		Collection: "cloned",
	}); err != nil {
		t.Fatalf("SaveRequest failed: %v", err)
	}

	collectionPath, err := store.CollectionPath("cloned")
	if err != nil {
		t.Fatalf("CollectionPath failed: %v", err)
	}
	if err := os.Remove(filepath.Join(collectionPath, collectionKeyFileName)); err != nil {
		t.Fatalf("failed to remove collection key: %v", err)
	}

	if _, err := store.GetCollectionByName("cloned"); !IsCollectionLocked(err) {
		t.Fatalf("expected locked collection error, got %v", err)
	}
	if _, err := store.GetRequestByName("uses secret"); !IsCollectionLocked(err) {
		t.Fatalf("expected request lookup to fail locked, got %v", err)
	}
	if _, err := store.ListRequests(&ListOptions{Collection: "cloned"}); !IsCollectionLocked(err) {
		t.Fatalf("expected request list to fail locked, got %v", err)
	}
}

func TestProjectDBDoesNotFallbackToDBForLockedCollection(t *testing.T) {
	base := NewLMDBWithPath(filepath.Join(t.TempDir(), "gurl.db"))
	if err := base.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer base.Close()
	if err := base.SaveRequest(&types.SavedRequest{
		ID:         "db-req",
		Name:       "uses secret",
		URL:        "https://db.example.com",
		Collection: "cloned",
	}); err != nil {
		t.Fatalf("base SaveRequest failed: %v", err)
	}
	if err := base.SaveRequest(&types.SavedRequest{
		ID:         "file-req",
		Name:       "file shadow",
		URL:        "https://db-shadow.example.com",
		Collection: "cloned",
	}); err != nil {
		t.Fatalf("base shadow SaveRequest failed: %v", err)
	}

	proj, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	fileStore := NewFileStore(proj)
	collection := types.NewCollection("cloned")
	collection.SetSecretVariable("TOKEN", "local-secret")
	if err := fileStore.SaveCollection(collection); err != nil {
		t.Fatalf("SaveCollection failed: %v", err)
	}
	if err := fileStore.SaveRequest(&types.SavedRequest{
		ID:         "file-req",
		Name:       "uses secret",
		URL:        "https://file.example.com/{{TOKEN}}",
		Collection: "cloned",
	}); err != nil {
		t.Fatalf("file SaveRequest failed: %v", err)
	}
	collectionPath, err := fileStore.CollectionPath("cloned")
	if err != nil {
		t.Fatalf("CollectionPath failed: %v", err)
	}
	if err := os.Remove(filepath.Join(collectionPath, collectionKeyFileName)); err != nil {
		t.Fatalf("failed to remove collection key: %v", err)
	}

	db := NewProjectDB(base, fileStore)
	if _, err := db.GetRequestByName("uses secret"); !IsCollectionLocked(err) {
		t.Fatalf("expected locked collection error instead of DB fallback, got %v", err)
	}
	if err := db.SaveRequest(&types.SavedRequest{
		ID:         "new-req",
		Name:       "new request",
		URL:        "https://new.example.com/{{TOKEN}}",
		Collection: "cloned",
	}); !IsCollectionLocked(err) {
		t.Fatalf("expected SaveRequest to fail locked, got %v", err)
	}
	if err := db.UpdateRequest(&types.SavedRequest{
		ID:         "file-req",
		Name:       "uses secret",
		URL:        "https://updated.example.com/{{TOKEN}}",
		Collection: "cloned",
	}); !IsCollectionLocked(err) {
		t.Fatalf("expected UpdateRequest to fail locked, got %v", err)
	}
	if err := db.DeleteRequest("file-req"); !IsCollectionLocked(err) {
		t.Fatalf("expected DeleteRequest to fail locked, got %v", err)
	}
	if _, err := base.GetRequest("file-req"); err != nil {
		t.Fatalf("expected DB shadow row to survive locked delete attempt: %v", err)
	}
	if err := db.SaveCollection(collection); !IsCollectionLocked(err) {
		t.Fatalf("expected SaveCollection to fail locked, got %v", err)
	}
	if err := db.UpdateCollection(collection); !IsCollectionLocked(err) {
		t.Fatalf("expected UpdateCollection to fail locked, got %v", err)
	}
	if err := db.DeleteCollection(collection.ID); !IsCollectionLocked(err) {
		t.Fatalf("expected DeleteCollection to fail locked, got %v", err)
	}
}
