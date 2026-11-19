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

// A request saved without an ID replaces the request of the same name in its
// collection, as it does in the LevelDB store. `import --force` relies on
// this: before, every re-import added a second file for each request.
func TestFileStoreSaveWithoutIDReplacesSameName(t *testing.T) {
	proj, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	store := NewFileStore(proj)
	if err := store.SaveCollection(types.NewCollection("payments")); err != nil {
		t.Fatalf("SaveCollection failed: %v", err)
	}
	if err := store.SaveCollection(types.NewCollection("orders")); err != nil {
		t.Fatalf("SaveCollection failed: %v", err)
	}

	first := &types.SavedRequest{Name: "list", URL: "https://api.example.com/v1", Method: "GET", Collection: "payments"}
	if err := store.SaveRequest(first); err != nil {
		t.Fatalf("SaveRequest failed: %v", err)
	}
	second := &types.SavedRequest{Name: "list", URL: "https://api.example.com/v2", Method: "GET", Collection: "payments"}
	if err := store.SaveRequest(second); err != nil {
		t.Fatalf("SaveRequest failed: %v", err)
	}
	other := &types.SavedRequest{Name: "list", URL: "https://api.example.com/orders", Method: "GET", Collection: "orders"}
	if err := store.SaveRequest(other); err != nil {
		t.Fatalf("SaveRequest failed: %v", err)
	}

	if second.ID != first.ID {
		t.Fatalf("expected the second save to reuse ID %q, got %q", first.ID, second.ID)
	}
	if other.ID == first.ID {
		t.Fatal("a request of the same name in another collection must not be replaced")
	}
	requests, err := store.ListRequests(&ListOptions{Collection: "payments"})
	if err != nil {
		t.Fatalf("ListRequests failed: %v", err)
	}
	if len(requests) != 1 || requests[0].URL != "https://api.example.com/v2" {
		t.Fatalf("expected one request with the new URL, got %+v", requests)
	}
}
