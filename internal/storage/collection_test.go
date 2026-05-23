package storage

import (
	"path/filepath"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

func TestCollectionCRUD(t *testing.T) {
	db := NewLMDBWithPath(filepath.Join(t.TempDir(), "collections.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	collection := types.NewCollection("payments")
	collection.SetVariable("BASE_URL", "https://api.example.com")
	collection.SetSecretVariable("API_KEY", "secret")
	if err := db.SaveCollection(collection); err != nil {
		t.Fatalf("SaveCollection failed: %v", err)
	}

	loaded, err := db.GetCollectionByName("payments")
	if err != nil {
		t.Fatalf("GetCollectionByName failed: %v", err)
	}
	if loaded.ID == "" || loaded.Variables["BASE_URL"] != "https://api.example.com" || !loaded.IsSecret("API_KEY") {
		t.Fatalf("loaded collection mismatch: %+v", loaded)
	}

	loaded.Name = "billing"
	loaded.SetVariable("BASE_URL", "https://billing.example.com")
	if err := db.UpdateCollection(loaded); err != nil {
		t.Fatalf("UpdateCollection failed: %v", err)
	}
	if _, err := db.GetCollectionByName("payments"); err == nil {
		t.Fatal("expected old collection name index to be removed")
	}
	renamed, err := db.GetCollectionByName("billing")
	if err != nil {
		t.Fatalf("renamed collection not found: %v", err)
	}
	if renamed.Variables["BASE_URL"] != "https://billing.example.com" {
		t.Fatalf("expected updated variable, got %+v", renamed.Variables)
	}

	collections, err := db.ListCollections()
	if err != nil {
		t.Fatalf("ListCollections failed: %v", err)
	}
	if len(collections) != 1 || collections[0].Name != "billing" {
		t.Fatalf("expected one renamed collection, got %+v", collections)
	}

	if err := db.DeleteCollection(renamed.ID); err != nil {
		t.Fatalf("DeleteCollection failed: %v", err)
	}
	if collections, err := db.ListCollections(); err != nil || len(collections) != 0 {
		t.Fatalf("expected no collections after delete, got %+v err=%v", collections, err)
	}
}

func TestSaveRequestAutoCreatesCollectionRecord(t *testing.T) {
	db := NewLMDBWithPath(filepath.Join(t.TempDir(), "request-collection.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	if err := db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "list charges",
		URL:        "https://example.com/charges",
		Method:     "GET",
		Collection: "payments",
	}); err != nil {
		t.Fatalf("SaveRequest failed: %v", err)
	}

	collection, err := db.GetCollectionByName("payments")
	if err != nil {
		t.Fatalf("expected collection record to be created: %v", err)
	}
	if collection.Name != "payments" {
		t.Fatalf("unexpected collection: %+v", collection)
	}
}

func TestSaveRequestUpsertMovesCollectionIndex(t *testing.T) {
	db := NewLMDBWithPath(filepath.Join(t.TempDir(), "move-collection.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	if err := db.SaveRequest(&types.SavedRequest{
		Name:       "shared name",
		URL:        "https://example.com/old",
		Method:     "GET",
		Collection: "old",
	}); err != nil {
		t.Fatalf("initial SaveRequest failed: %v", err)
	}
	if err := db.SaveRequest(&types.SavedRequest{
		Name:       "shared name",
		URL:        "https://example.com/new",
		Method:     "GET",
		Collection: "new",
	}); err != nil {
		t.Fatalf("upsert SaveRequest failed: %v", err)
	}

	oldRequests, err := db.ListRequests(&ListOptions{Collection: "old"})
	if err != nil {
		t.Fatalf("ListRequests old failed: %v", err)
	}
	if len(oldRequests) != 0 {
		t.Fatalf("expected old collection index to be cleaned, got %+v", oldRequests)
	}

	newRequests, err := db.ListRequests(&ListOptions{Collection: "new"})
	if err != nil {
		t.Fatalf("ListRequests new failed: %v", err)
	}
	if len(newRequests) != 1 || newRequests[0].URL != "https://example.com/new" {
		t.Fatalf("expected request in new collection, got %+v", newRequests)
	}
}
