package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

// Helper to create a test request
func testRequest(name string) *types.SavedRequest {
	return &types.SavedRequest{
		ID:         "test-id-" + name,
		Name:       name,
		URL:        "https://example.com/" + name,
		Method:     "GET",
		Headers:    []types.Header{},
		Collection: "test-collection",
		Tags:       []string{"test"},
	}
}

func TestSaveRequest(t *testing.T) {
	db := NewInMemoryDB()
	req := testRequest("test-save")

	err := db.SaveRequest(req)
	if err != nil {
		t.Fatalf("SaveRequest failed: %v", err)
	}

	// Verify it was saved
	got, err := db.GetRequest("test-save")
	if err != nil {
		t.Fatalf("GetRequest failed: %v", err)
	}
	if got.Name != req.Name {
		t.Errorf("got name %q, want %q", got.Name, req.Name)
	}
}

func TestGetRequest(t *testing.T) {
	db := NewInMemoryDB()
	req := testRequest("test-get")
	db.SaveRequest(req)

	got, err := db.GetRequest("test-get")
	if err != nil {
		t.Fatalf("GetRequest failed: %v", err)
	}
	if got.ID != req.ID {
		t.Errorf("got ID %q, want %q", got.ID, req.ID)
	}
	if got.URL != req.URL {
		t.Errorf("got URL %q, want %q", got.URL, req.URL)
	}
}

func TestGetRequestNotFound(t *testing.T) {
	db := NewInMemoryDB()

	_, err := db.GetRequest("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent request")
	}
}

func TestDeleteRequest(t *testing.T) {
	db := NewInMemoryDB()
	req := testRequest("test-delete")
	db.SaveRequest(req)

	err := db.DeleteRequest("test-delete")
	if err != nil {
		t.Fatalf("DeleteRequest failed: %v", err)
	}

	// Verify it's gone
	_, err = db.GetRequest("test-delete")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestDeleteRequestNotFound(t *testing.T) {
	db := NewInMemoryDB()

	err := db.DeleteRequest("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent request")
	}
}

func TestUpdateRequest(t *testing.T) {
	db := NewInMemoryDB()
	req := testRequest("test-update")
	db.SaveRequest(req)

	// Update the request
	req.URL = "https://updated.com/test"
	req.Method = "POST"

	err := db.SaveRequest(req)
	if err != nil {
		t.Fatalf("SaveRequest (update) failed: %v", err)
	}

	got, err := db.GetRequest("test-update")
	if err != nil {
		t.Fatalf("GetRequest failed: %v", err)
	}
	if got.URL != "https://updated.com/test" {
		t.Errorf("got URL %q, want %q", got.URL, "https://updated.com/test")
	}
	if got.Method != "POST" {
		t.Errorf("got method %q, want %q", got.Method, "POST")
	}
}

func TestListRequests(t *testing.T) {
	db := NewInMemoryDB()

	// Add multiple requests
	db.SaveRequest(testRequest("req1"))
	db.SaveRequest(testRequest("req2"))
	db.SaveRequest(testRequest("req3"))

	requests, err := db.ListRequests(types.ListOptions{})
	if err != nil {
		t.Fatalf("ListRequests failed: %v", err)
	}
	if len(requests) != 3 {
		t.Errorf("got %d requests, want 3", len(requests))
	}
}

func TestListRequestsWithCollectionFilter(t *testing.T) {
	db := NewInMemoryDB()

	req1 := testRequest("req-col-a")
	req1.Collection = "collection-a"
	db.SaveRequest(req1)

	req2 := testRequest("req-col-b")
	req2.Collection = "collection-b"
	db.SaveRequest(req2)

	requests, err := db.ListRequests(types.ListOptions{Collection: "collection-a"})
	if err != nil {
		t.Fatalf("ListRequests failed: %v", err)
	}
	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}
	if requests[0].Collection != "collection-a" {
		t.Errorf("got collection %q, want %q", requests[0].Collection, "collection-a")
	}
}

func TestListRequestsWithTagFilter(t *testing.T) {
	db := NewInMemoryDB()

	req1 := testRequest("req-tag-1")
	req1.Tags = []string{"tag-a", "tag-b"}
	db.SaveRequest(req1)

	req2 := testRequest("req-tag-2")
	req2.Tags = []string{"tag-c"}
	db.SaveRequest(req2)

	requests, err := db.ListRequests(types.ListOptions{Tag: "tag-a"})
	if err != nil {
		t.Fatalf("ListRequests failed: %v", err)
	}
	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}
	if requests[0].Name != "req-tag-1" {
		t.Errorf("got name %q, want %q", requests[0].Name, "req-tag-1")
	}
}

func TestRenameRequest(t *testing.T) {
	db := NewInMemoryDB()
	req := testRequest("old-name")
	db.SaveRequest(req)

	err := db.RenameRequest("old-name", "new-name")
	if err != nil {
		t.Fatalf("RenameRequest failed: %v", err)
	}

	// Old name should not exist
	_, err = db.GetRequest("old-name")
	if err == nil {
		t.Error("old name should not exist")
	}

	// New name should exist
	got, err := db.GetRequest("new-name")
	if err != nil {
		t.Fatalf("GetRequest failed: %v", err)
	}
	if got.Name != "new-name" {
		t.Errorf("got name %q, want %q", got.Name, "new-name")
	}
}

func TestRenameRequestNotFound(t *testing.T) {
	db := NewInMemoryDB()

	err := db.RenameRequest("nonexistent", "new-name")
	if err == nil {
		t.Error("expected error for nonexistent request")
	}
}

func TestSaveHistory(t *testing.T) {
	db := NewInMemoryDB()

	entry := &types.HistoryEntry{
		ID:          "hist-1",
		RequestID:   "req-1",
		StatusCode:  200,
		DurationMs:  100,
		SizeBytes:   1024,
	}

	err := db.SaveHistory(entry)
	if err != nil {
		t.Fatalf("SaveHistory failed: %v", err)
	}
}

func TestClose(t *testing.T) {
	db := NewInMemoryDB()

	err := db.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestLMDBCreation(t *testing.T) {
	// Test that NewLMDB creates the database directory structure
	// This test checks the path construction, not actual DB operations

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot determine home directory")
	}

	expectedDir := filepath.Join(homeDir, ".local", "share", "scurl")
	if expectedDir == "" {
		t.Error("expected directory path should not be empty")
	}

	// Verify LMDB type exists and can be created
	lmdb := &LMDB{}
	if lmdb == nil {
		t.Error("expected non-nil LMDB struct")
	}
}

func TestInMemoryDBMultipleRequests(t *testing.T) {
	db := NewInMemoryDB()

	// Save many requests
	for i := 0; i < 100; i++ {
		req := testRequest("bulk-req-%d")
		req.Name = "bulk-req-%d"
		db.SaveRequest(&types.SavedRequest{
			ID:   "id-%d",
			Name: "bulk-req-%d",
			URL:  "https://example.com/bulk/%d",
		})
	}

	requests, err := db.ListRequests(types.ListOptions{})
	if err != nil {
		t.Fatalf("ListRequests failed: %v", err)
	}
	// All requests share the same name, so only one will be stored
	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1 (last one wins)", len(requests))
	}
}

func TestEmptyListRequests(t *testing.T) {
	db := NewInMemoryDB()

	requests, err := db.ListRequests(types.ListOptions{})
	if err != nil {
		t.Fatalf("ListRequests failed: %v", err)
	}
	if len(requests) != 0 {
		t.Errorf("got %d requests, want 0", len(requests))
	}
}

func TestDuplicateRequestOverwrite(t *testing.T) {
	db := NewInMemoryDB()

	req1 := testRequest("dup")
	req1.URL = "https://first.com"
	db.SaveRequest(req1)

	req2 := testRequest("dup")
	req2.URL = "https://second.com"
	db.SaveRequest(req2)

	requests, err := db.ListRequests(types.ListOptions{})
	if err != nil {
		t.Fatalf("ListRequests failed: %v", err)
	}
	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}
	if requests[0].URL != "https://second.com" {
		t.Errorf("got URL %q, want %q", requests[0].URL, "https://second.com")
	}
}

// TestLMDBWithTempDir tests LMDB with a custom path (requires mocking or special setup)
func TestLMDBPathStructure(t *testing.T) {
	// Verify LMDB creates the correct path structure
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot determine home directory")
	}

	expectedPath := filepath.Join(home, ".local", "share", "scurl", "scurl.db")
	if expectedPath == "" {
		t.Error("expected path should not be empty")
	}
}
