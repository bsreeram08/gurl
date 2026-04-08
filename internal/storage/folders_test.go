package storage

import (
	"path/filepath"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

// TestFolder_Create tests creating a folder and storing a request in it
func TestFolder_Create(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_folder.db")
	db := &LMDB{dbPath: dbPath}
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	// Create a request with a folder path
	req := &types.SavedRequest{
		Name:   "get-user",
		URL:    "https://api.example.com/v2/users",
		Method: "GET",
		Folder: "api/v2/users",
	}

	if err := db.SaveRequest(req); err != nil {
		t.Fatalf("failed to save request: %v", err)
	}

	// List requests in the folder
	requests, err := db.ListFolder("api/v2/users")
	if err != nil {
		t.Fatalf("failed to list folder: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("expected 1 request in folder, got %d", len(requests))
	}

	if requests[0].Name != "get-user" {
		t.Errorf("expected request name 'get-user', got '%s'", requests[0].Name)
	}

	if requests[0].Folder != "api/v2/users" {
		t.Errorf("expected folder 'api/v2/users', got '%s'", requests[0].Folder)
	}
}

// TestFolder_Nested tests folder inside folder structure
func TestFolder_Nested(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_nested.db")
	db := &LMDB{dbPath: dbPath}
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	// Create nested folder requests
	req1 := &types.SavedRequest{
		Name:   "admin-list",
		URL:    "https://api.example.com/v2/users/admin",
		Method: "GET",
		Folder: "api/v2/users/admin",
	}
	req2 := &types.SavedRequest{
		Name:   "admin-create",
		URL:    "https://api.example.com/v2/users/admin",
		Method: "POST",
		Folder: "api/v2/users/admin",
	}

	if err := db.SaveRequest(req1); err != nil {
		t.Fatalf("failed to save request 1: %v", err)
	}
	if err := db.SaveRequest(req2); err != nil {
		t.Fatalf("failed to save request 2: %v", err)
	}

	// List recursive - should find both
	requests, err := db.ListFolderRecursive("api/v2/users")
	if err != nil {
		t.Fatalf("failed to list recursive: %v", err)
	}

	if len(requests) != 2 {
		t.Errorf("expected 2 requests in recursive listing, got %d", len(requests))
	}

	// List specific nested folder only
	nested, err := db.ListFolder("api/v2/users/admin")
	if err != nil {
		t.Fatalf("failed to list nested folder: %v", err)
	}

	if len(nested) != 2 {
		t.Errorf("expected 2 requests in nested folder, got %d", len(nested))
	}
}

// TestFolder_MoveRequest tests moving a request into a different folder
func TestFolder_MoveRequest(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_move.db")
	db := &LMDB{dbPath: dbPath}
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	// Create request in original folder
	req := &types.SavedRequest{
		Name:   "get-user",
		URL:    "https://api.example.com/v2/users",
		Method: "GET",
		Folder: "api/v2/users",
	}

	if err := db.SaveRequest(req); err != nil {
		t.Fatalf("failed to save request: %v", err)
	}

	// Verify it's in the original folder
	requests, err := db.ListFolder("api/v2/users")
	if err != nil {
		t.Fatalf("failed to list original folder: %v", err)
	}
	if len(requests) != 1 {
		t.Errorf("expected 1 request in original folder, got %d", len(requests))
	}

	// Move the request to a new folder by updating it
	req.Folder = "api/v2/admin"
	if err := db.UpdateRequest(req); err != nil {
		t.Fatalf("failed to update request: %v", err)
	}

	// Verify it's no longer in the original folder
	requests, err = db.ListFolder("api/v2/users")
	if err != nil {
		t.Fatalf("failed to list original folder after move: %v", err)
	}
	if len(requests) != 0 {
		t.Errorf("expected 0 requests in original folder after move, got %d", len(requests))
	}

	// Verify it's in the new folder
	requests, err = db.ListFolder("api/v2/admin")
	if err != nil {
		t.Fatalf("failed to list new folder: %v", err)
	}
	if len(requests) != 1 {
		t.Errorf("expected 1 request in new folder, got %d", len(requests))
	}
}

// TestFolder_ListFolder tests listing requests in a specific folder
func TestFolder_ListFolder(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_list_folder.db")
	db := &LMDB{dbPath: dbPath}
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	// Create requests in different folders
	req1 := &types.SavedRequest{
		Name:   "get-users",
		URL:    "https://api.example.com/v2/users",
		Method: "GET",
		Folder: "api/v2/users",
	}
	req2 := &types.SavedRequest{
		Name:   "get-posts",
		URL:    "https://api.example.com/v2/posts",
		Method: "GET",
		Folder: "api/v2/posts",
	}
	req3 := &types.SavedRequest{
		Name:   "get-user",
		URL:    "https://api.example.com/v2/users/123",
		Method: "GET",
		Folder: "api/v2/users",
	}

	for _, req := range []*types.SavedRequest{req1, req2, req3} {
		if err := db.SaveRequest(req); err != nil {
			t.Fatalf("failed to save request: %v", err)
		}
	}

	// List only api/v2/users folder
	requests, err := db.ListFolder("api/v2/users")
	if err != nil {
		t.Fatalf("failed to list folder: %v", err)
	}

	if len(requests) != 2 {
		t.Errorf("expected 2 requests in api/v2/users, got %d", len(requests))
	}

	// Verify the correct requests are returned
	names := make(map[string]bool)
	for _, r := range requests {
		names[r.Name] = true
	}

	if !names["get-users"] {
		t.Errorf("expected get-users in folder listing")
	}
	if !names["get-user"] {
		t.Errorf("expected get-user in folder listing")
	}
	if names["get-posts"] {
		t.Errorf("get-posts should not be in api/v2/users folder")
	}
}

// TestFolder_ListRecursive tests recursive listing of requests in folder and subfolders
func TestFolder_ListRecursive(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_list_recursive.db")
	db := &LMDB{dbPath: dbPath}
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	// Create nested folder structure
	req1 := &types.SavedRequest{
		Name:   "get-users",
		URL:    "https://api.example.com/v2/users",
		Method: "GET",
		Folder: "api/v2/users",
	}
	req2 := &types.SavedRequest{
		Name:   "get-admin",
		URL:    "https://api.example.com/v2/users/admin",
		Method: "GET",
		Folder: "api/v2/users/admin",
	}
	req3 := &types.SavedRequest{
		Name:   "get-superadmin",
		URL:    "https://api.example.com/v2/users/admin/super",
		Method: "GET",
		Folder: "api/v2/users/admin/super",
	}
	req4 := &types.SavedRequest{
		Name:   "get-posts",
		URL:    "https://api.example.com/v2/posts",
		Method: "GET",
		Folder: "api/v2/posts",
	}

	for _, req := range []*types.SavedRequest{req1, req2, req3, req4} {
		if err := db.SaveRequest(req); err != nil {
			t.Fatalf("failed to save request: %v", err)
		}
	}

	// Recursive listing from api/v2/users should include all subfolders
	requests, err := db.ListFolderRecursive("api/v2/users")
	if err != nil {
		t.Fatalf("failed to list recursive: %v", err)
	}

	if len(requests) != 3 {
		t.Errorf("expected 3 requests in recursive listing, got %d", len(requests))
	}

	// Verify all nested requests are included
	names := make(map[string]bool)
	for _, r := range requests {
		names[r.Name] = true
	}

	if !names["get-users"] {
		t.Errorf("expected get-users in recursive listing")
	}
	if !names["get-admin"] {
		t.Errorf("expected get-admin in recursive listing")
	}
	if !names["get-superadmin"] {
		t.Errorf("expected get-superadmin in recursive listing")
	}
	if names["get-posts"] {
		t.Errorf("get-posts should not be in api/v2/users recursive listing")
	}
}

// TestFolder_DeleteFolder tests deleting a folder moves requests to parent
func TestFolder_DeleteFolder(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_delete_folder.db")
	db := &LMDB{dbPath: dbPath}
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	// Create requests in nested folder
	req1 := &types.SavedRequest{
		Name:   "get-admin",
		URL:    "https://api.example.com/v2/users/admin",
		Method: "GET",
		Folder: "api/v2/users/admin",
	}
	req2 := &types.SavedRequest{
		Name:   "list-admin",
		URL:    "https://api.example.com/v2/users/admin",
		Method: "GET",
		Folder: "api/v2/users/admin",
	}

	for _, req := range []*types.SavedRequest{req1, req2} {
		if err := db.SaveRequest(req); err != nil {
			t.Fatalf("failed to save request: %v", err)
		}
	}

	// Delete the folder
	if err := db.DeleteFolder("api/v2/users/admin"); err != nil {
		t.Fatalf("failed to delete folder: %v", err)
	}

	// Requests should now have empty folder (moved to root)
	allReqs, err := db.ListRequests(&ListOptions{})
	if err != nil {
		t.Fatalf("failed to list all requests: %v", err)
	}

	for _, r := range allReqs {
		if r.Folder == "api/v2/users/admin" {
			t.Errorf("request %s should have empty folder after folder deletion", r.Name)
		}
	}
}

// TestFolder_FolderPath tests that requests display with full path
func TestFolder_FolderPath(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_folder_path.db")
	db := &LMDB{dbPath: dbPath}
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	// Create request with folder
	req := &types.SavedRequest{
		Name:   "get-user",
		URL:    "https://api.example.com/v2/users",
		Method: "GET",
		Folder: "api/v2/users",
	}

	if err := db.SaveRequest(req); err != nil {
		t.Fatalf("failed to save request: %v", err)
	}

	// Verify the request has the folder field set
	savedReq, err := db.GetRequest(req.ID)
	if err != nil {
		t.Fatalf("failed to get request: %v", err)
	}

	if savedReq.Folder != "api/v2/users" {
		t.Errorf("expected folder 'api/v2/users', got '%s'", savedReq.Folder)
	}

	// List folder and verify path
	requests, err := db.ListFolder("api/v2/users")
	if err != nil {
		t.Fatalf("failed to list folder: %v", err)
	}

	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}

	if requests[0].Folder != "api/v2/users" {
		t.Errorf("expected folder path 'api/v2/users', got '%s'", requests[0].Folder)
	}
}

// TestFolder_RootRequests tests that requests without folder appear at root level
func TestFolder_RootRequests(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_root.db")
	db := &LMDB{dbPath: dbPath}
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	// Create requests - some with folders, some without
	req1 := &types.SavedRequest{
		Name:   "root-request",
		URL:    "https://api.example.com/root",
		Method: "GET",
		Folder: "", // No folder - root level
	}
	req2 := &types.SavedRequest{
		Name:   "foldered-request",
		URL:    "https://api.example.com/v2/users",
		Method: "GET",
		Folder: "api/v2/users",
	}
	req3 := &types.SavedRequest{
		Name:   "another-root",
		URL:    "https://api.example.com/another",
		Method: "GET",
		Folder: "", // No folder - root level
	}

	for _, req := range []*types.SavedRequest{req1, req2, req3} {
		if err := db.SaveRequest(req); err != nil {
			t.Fatalf("failed to save request: %v", err)
		}
	}

	// List all unique folders
	folders, err := db.GetAllFolders()
	if err != nil {
		t.Fatalf("failed to get all folders: %v", err)
	}

	// Should only have one unique folder (empty string is not a folder)
	found := false
	for _, f := range folders {
		if f == "api/v2/users" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'api/v2/users' in folders list")
	}

	// List all requests and check root requests
	allReqs, err := db.ListRequests(&ListOptions{})
	if err != nil {
		t.Fatalf("failed to list all requests: %v", err)
	}

	rootCount := 0
	folderCount := 0
	for _, r := range allReqs {
		if r.Folder == "" {
			rootCount++
		} else {
			folderCount++
		}
	}

	if rootCount != 2 {
		t.Errorf("expected 2 root requests, got %d", rootCount)
	}
	if folderCount != 1 {
		t.Errorf("expected 1 foldered request, got %d", folderCount)
	}
}
