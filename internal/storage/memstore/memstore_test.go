package memstore

import (
	"errors"
	"testing"

	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

func TestCollectionLockLifecycle(t *testing.T) {
	var db DB // zero value is usable
	c := types.NewCollection("payments")
	c.SetVariable("BASE_URL", "https://api.example.com")
	if err := db.SaveCollection(c); err != nil {
		t.Fatalf("SaveCollection: %v", err)
	}
	db.Lock("payments", "pass")

	if _, err := db.GetCollectionByName("payments"); !storage.IsCollectionLocked(err) {
		t.Fatalf("GetCollectionByName on locked = %v, want locked error", err)
	}
	if _, err := db.GetCollection(c.ID); !storage.IsCollectionLocked(err) {
		t.Fatalf("GetCollection on locked = %v, want locked error", err)
	}
	if err := db.SaveCollection(c); !storage.IsCollectionLocked(err) {
		t.Fatalf("SaveCollection over locked = %v, want locked error", err)
	}
	raw, err := db.GetRawCollectionByName("payments")
	if err != nil || raw.ID != c.ID {
		t.Fatalf("GetRawCollectionByName = %+v, %v", raw, err)
	}
	if list, err := db.ListCollections(); err != nil || len(list) != 1 {
		t.Fatalf("ListCollections = %+v, %v; want the locked collection listed", list, err)
	}

	if err := db.UnlockCollection("payments", "wrong"); err == nil {
		t.Fatal("UnlockCollection with wrong passphrase succeeded")
	}
	if err := db.UnlockCollection("payments", "pass"); err != nil {
		t.Fatalf("UnlockCollection: %v", err)
	}
	if _, err := db.GetCollectionByName("payments"); err != nil {
		t.Fatalf("GetCollectionByName after unlock: %v", err)
	}

	db.Lock("payments", "pass")
	if err := db.SaveCollectionAllowLocked(raw); err != nil {
		t.Fatalf("SaveCollectionAllowLocked: %v", err)
	}
	if _, err := db.GetCollectionByName("payments"); err != nil {
		t.Fatalf("collection still locked after SaveCollectionAllowLocked: %v", err)
	}
}

func TestCollectionNotFoundAndUnsupported(t *testing.T) {
	db := New()
	if _, err := db.GetCollectionByName("missing"); !storage.IsCollectionNotFound(err) {
		t.Fatalf("GetCollectionByName = %v, want not found", err)
	}
	if err := db.DeleteCollection("missing"); !storage.IsCollectionNotFound(err) {
		t.Fatalf("DeleteCollection = %v, want not found", err)
	}
	if _, _, err := db.MigrateCollectionToFiles("x"); !errors.Is(err, storage.ErrNotSupported) {
		t.Fatalf("MigrateCollectionToFiles = %v, want ErrNotSupported", err)
	}
}

func TestSaveRequestCreatesCollectionAndCopies(t *testing.T) {
	db := New()
	req := &types.SavedRequest{Name: "list", URL: "https://example.com", Collection: "payments", Folder: "a/b"}
	if err := db.SaveRequest(req); err != nil {
		t.Fatalf("SaveRequest: %v", err)
	}
	if req.ID == "" {
		t.Fatal("SaveRequest did not assign an ID")
	}
	if _, err := db.GetCollectionByName("payments"); err != nil {
		t.Fatalf("collection not created on save: %v", err)
	}

	req.URL = "mutated"
	got, err := db.GetRequestByName("list")
	if err != nil {
		t.Fatalf("GetRequestByName: %v", err)
	}
	if got.URL != "https://example.com" {
		t.Fatalf("stored request aliased caller's value: %q", got.URL)
	}

	if list, _ := db.ListFolderRecursive("a"); len(list) != 1 {
		t.Fatalf("ListFolderRecursive(a) = %d requests, want 1", len(list))
	}
	if err := db.DeleteRequest(got.ID); err != nil {
		t.Fatalf("DeleteRequest: %v", err)
	}
	if _, err := db.GetRequest(got.ID); err == nil {
		t.Fatal("GetRequest after delete succeeded")
	}
}
