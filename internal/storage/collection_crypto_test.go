package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/project"
	"github.com/sreeram/gurl/pkg/types"
)

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

func TestFileStoreUnlocksPassphraseCollectionToLocalKey(t *testing.T) {
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

	locked, err := store.GetCollectionByName("shared")
	if err != nil {
		t.Fatalf("GetCollectionByName failed: %v", err)
	}
	if !IsCollectionEncryptedValue(locked.Variables["TOKEN"]) {
		t.Fatalf("expected collection to start locked, got %q", locked.Variables["TOKEN"])
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
	if _, err := os.Stat(filepath.Join(collectionPath, collectionKeyFileName)); err != nil {
		t.Fatalf("expected local collection key after unlock: %v", err)
	}

	rawData, err := os.ReadFile(filepath.Join(collectionPath, collectionFileName))
	if err != nil {
		t.Fatalf("failed to read unlocked collection file: %v", err)
	}
	if strings.Contains(string(rawData), "shared-secret") {
		t.Fatal("unlocked collection should be re-encrypted with local key")
	}
}
