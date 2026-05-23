package storage

import (
	"path/filepath"
	"testing"

	"github.com/sreeram/gurl/internal/project"
	"github.com/sreeram/gurl/pkg/types"
)

func TestProjectDBRoutesFileBackedCollectionsToFileStore(t *testing.T) {
	base := NewLMDBWithPath(filepath.Join(t.TempDir(), "gurl.db"))
	if err := base.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer base.Close()

	proj, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	files := NewFileStore(proj)
	if err := files.SaveCollection(types.NewCollection("file-api")); err != nil {
		t.Fatalf("SaveCollection failed: %v", err)
	}
	db := NewProjectDB(base, files)

	req := &types.SavedRequest{
		ID:         "req-file",
		Name:       "list users",
		URL:        "https://file.example.com/users",
		Method:     "GET",
		Collection: "file-api",
	}
	if err := db.SaveRequest(req); err != nil {
		t.Fatalf("SaveRequest failed: %v", err)
	}

	fileRequests, err := files.ListRequests(&ListOptions{Collection: "file-api"})
	if err != nil {
		t.Fatalf("file ListRequests failed: %v", err)
	}
	if len(fileRequests) != 1 {
		t.Fatalf("expected request to be saved to file store, got %d", len(fileRequests))
	}
	baseRequests, err := base.ListRequests(&ListOptions{Collection: "file-api"})
	if err != nil {
		t.Fatalf("base ListRequests failed: %v", err)
	}
	if len(baseRequests) != 0 {
		t.Fatalf("expected DB to stay empty for file-backed collection, got %+v", baseRequests)
	}
}

func TestProjectDBMigrationMakesFileCopyTakePrecedence(t *testing.T) {
	base := NewLMDBWithPath(filepath.Join(t.TempDir(), "gurl.db"))
	if err := base.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer base.Close()

	if err := base.SaveRequest(&types.SavedRequest{
		ID:         "legacy-1",
		Name:       "legacy request",
		URL:        "https://db.example.com",
		Method:     "GET",
		Collection: "legacy",
	}); err != nil {
		t.Fatalf("base SaveRequest failed: %v", err)
	}

	proj, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	files := NewFileStore(proj)
	db := NewProjectDB(base, files)

	count, _, err := db.MigrateCollectionToFiles("legacy")
	if err != nil {
		t.Fatalf("MigrateCollectionToFiles failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one migrated request, got %d", count)
	}

	fileReq, err := files.GetRequest("legacy-1")
	if err != nil {
		t.Fatalf("expected migrated file request: %v", err)
	}
	fileReq.URL = "https://file.example.com"
	if err := files.SaveRequest(fileReq); err != nil {
		t.Fatalf("file SaveRequest failed: %v", err)
	}

	loaded, err := db.GetRequestByName("legacy request")
	if err != nil {
		t.Fatalf("GetRequestByName failed: %v", err)
	}
	if loaded.URL != "https://file.example.com" {
		t.Fatalf("expected fresh file-backed request to win, got %s", loaded.URL)
	}

	requests, err := db.ListRequests(&ListOptions{Collection: "legacy"})
	if err != nil {
		t.Fatalf("ListRequests failed: %v", err)
	}
	if len(requests) != 1 || requests[0].URL != "https://file.example.com" {
		t.Fatalf("expected DB duplicate to be hidden after migration, got %+v", requests)
	}
}

func TestProjectDBDeleteRemovesMigratedDBShadow(t *testing.T) {
	base := NewLMDBWithPath(filepath.Join(t.TempDir(), "gurl.db"))
	if err := base.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer base.Close()

	if err := base.SaveRequest(&types.SavedRequest{
		ID:         "legacy-1",
		Name:       "legacy request",
		URL:        "https://db.example.com",
		Method:     "GET",
		Collection: "legacy",
	}); err != nil {
		t.Fatalf("base SaveRequest failed: %v", err)
	}

	proj, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	db := NewProjectDB(base, NewFileStore(proj))
	if _, _, err := db.MigrateCollectionToFiles("legacy"); err != nil {
		t.Fatalf("MigrateCollectionToFiles failed: %v", err)
	}

	if err := db.DeleteRequest("legacy-1"); err != nil {
		t.Fatalf("DeleteRequest failed: %v", err)
	}
	if _, err := db.GetRequestByName("legacy request"); err == nil {
		t.Fatal("expected migrated request to stay deleted instead of falling back to DB shadow")
	}
	if _, err := base.GetRequest("legacy-1"); err == nil {
		t.Fatal("expected legacy DB shadow row to be removed")
	}
}
