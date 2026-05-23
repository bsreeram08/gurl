package storage

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/project"
	"github.com/sreeram/gurl/pkg/types"
)

func TestFileStoreSavesCollectionAndRequests(t *testing.T) {
	proj, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	store := NewFileStore(proj)

	collection := types.NewCollection("payments")
	collection.SetVariable("BASE_URL", "https://api.example.com")
	if err := store.SaveCollection(collection); err != nil {
		t.Fatalf("SaveCollection failed: %v", err)
	}

	for _, req := range []*types.SavedRequest{
		{ID: "req-one", Name: "GET /users", URL: "https://api.example.com/users", Method: "GET", Collection: "payments"},
		{ID: "req-two", Name: "GET /users", URL: "https://api.example.com/users/2", Method: "GET", Collection: "payments"},
	} {
		if err := store.SaveRequest(req); err != nil {
			t.Fatalf("SaveRequest failed: %v", err)
		}
	}

	requests, err := store.ListRequests(&ListOptions{Collection: "payments"})
	if err != nil {
		t.Fatalf("ListRequests failed: %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("expected duplicate request names to be stored separately, got %d", len(requests))
	}

	collectionPath, err := store.CollectionPath("payments")
	if err != nil {
		t.Fatalf("CollectionPath failed: %v", err)
	}
	files, err := filepath.Glob(filepath.Join(collectionPath, "*.json"))
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("expected collection.json plus two request files, got %v", files)
	}
	for _, file := range files {
		if strings.Contains(filepath.Base(file), "/") {
			t.Fatalf("request filename should not contain raw slash: %s", file)
		}
	}
}

func TestFileStoreUsesSafeCollectionDirectory(t *testing.T) {
	proj, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	store := NewFileStore(proj)

	collection := types.NewCollection("team/api")
	if err := store.SaveCollection(collection); err != nil {
		t.Fatalf("SaveCollection failed: %v", err)
	}

	path, err := store.CollectionPath("team/api")
	if err != nil {
		t.Fatalf("CollectionPath failed: %v", err)
	}
	if filepath.Dir(path) != proj.CollectionsDir() {
		t.Fatalf("expected collection to stay directly under collections dir, got %s", path)
	}
	if filepath.Base(path) == "api" {
		t.Fatalf("collection name was split as a raw path: %s", path)
	}
}
