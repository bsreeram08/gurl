package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

// mockDB implements storage.DB for testing
type mockDB struct {
	requests []*types.SavedRequest
}

func (m *mockDB) Open() error                               { return nil }
func (m *mockDB) Close() error                              { return nil }
func (m *mockDB) SaveRequest(req *types.SavedRequest) error { return nil }
func (m *mockDB) GetRequest(id string) (*types.SavedRequest, error) {
	for _, r := range m.requests {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, nil
}
func (m *mockDB) GetRequestByName(name string) (*types.SavedRequest, error) {
	for _, r := range m.requests {
		if r.Name == name {
			return r, nil
		}
	}
	return nil, nil
}
func (m *mockDB) ListRequests(opts *storage.ListOptions) ([]*types.SavedRequest, error) {
	return m.requests, nil
}
func (m *mockDB) DeleteRequest(id string) error                     { return nil }
func (m *mockDB) UpdateRequest(req *types.SavedRequest) error       { return nil }
func (m *mockDB) SaveHistory(history *types.ExecutionHistory) error { return nil }
func (m *mockDB) GetHistory(requestID string, limit int) ([]*types.ExecutionHistory, error) {
	return nil, nil
}
func (m *mockDB) ListFolder(path string) ([]*types.SavedRequest, error)          { return nil, nil }
func (m *mockDB) ListFolderRecursive(path string) ([]*types.SavedRequest, error) { return nil, nil }
func (m *mockDB) DeleteFolder(path string) error                                 { return nil }
func (m *mockDB) GetAllFolders() ([]string, error)                               { return nil, nil }

// TestRequestList_Load tests that requests are loaded from DB and displayed in sidebar
func TestRequestList_Load(t *testing.T) {
	db := &mockDB{
		requests: []*types.SavedRequest{
			{ID: "1", Name: "Get Users", Method: "GET", URL: "https://api.example.com/users"},
			{ID: "2", Name: "Create User", Method: "POST", URL: "https://api.example.com/users"},
		},
	}

	rl := NewRequestList(db)

	if len(rl.items) != 2 {
		t.Errorf("expected 2 items, got %d", len(rl.items))
	}

	if rl.items[0].Name != "Get Users" {
		t.Errorf("expected first item to be 'Get Users', got '%s'", rl.items[0].Name)
	}

	if rl.items[1].Method != "POST" {
		t.Errorf("expected second item method to be 'POST', got '%s'", rl.items[1].Method)
	}
}

// TestRequestList_Navigate tests j/k and arrow key navigation via cursor methods
func TestRequestList_Navigate(t *testing.T) {
	db := &mockDB{
		requests: []*types.SavedRequest{
			{ID: "1", Name: "Request 1", Method: "GET", URL: "https://api.example.com/1"},
			{ID: "2", Name: "Request 2", Method: "POST", URL: "https://api.example.com/2"},
			{ID: "3", Name: "Request 3", Method: "PUT", URL: "https://api.example.com/3"},
		},
	}

	rl := NewRequestList(db)

	// Test initial cursor position
	if rl.list.Cursor() != 0 {
		t.Errorf("expected initial cursor at 0, got %d", rl.list.Cursor())
	}

	// Test cursor down
	rl.list.CursorDown()
	if rl.list.Cursor() != 1 {
		t.Errorf("expected cursor at 1 after CursorDown, got %d", rl.list.Cursor())
	}

	// Test cursor down again
	rl.list.CursorDown()
	if rl.list.Cursor() != 2 {
		t.Errorf("expected cursor at 2 after second CursorDown, got %d", rl.list.Cursor())
	}

	// Test cursor up
	rl.list.CursorUp()
	if rl.list.Cursor() != 1 {
		t.Errorf("expected cursor at 1 after CursorUp, got %d", rl.list.Cursor())
	}

	// Test cursor up again
	rl.list.CursorUp()
	if rl.list.Cursor() != 0 {
		t.Errorf("expected cursor at 0 after second CursorUp, got %d", rl.list.Cursor())
	}
}

// TestRequestList_Select tests that Enter selects a request and triggers message
func TestRequestList_Select(t *testing.T) {
	db := &mockDB{
		requests: []*types.SavedRequest{
			{ID: "1", Name: "Get Users", Method: "GET", URL: "https://api.example.com/users"},
			{ID: "2", Name: "Create User", Method: "POST", URL: "https://api.example.com/users"},
		},
	}

	rl := NewRequestList(db)

	// Move cursor down
	rl.list.CursorDown()

	// Get selected item and trigger selection
	selected := rl.list.SelectedItem()
	if selected == nil {
		t.Fatal("expected selected item")
	}

	reqItem, ok := selected.(RequestItem)
	if !ok {
		t.Fatal("expected RequestItem")
	}

	// Manually trigger selection to create message
	rl.msgs = append(rl.msgs, RequestSelectedMsg{Request: reqItem.SavedRequest})

	// Check that a RequestSelectedMsg was sent
	if len(rl.msgs) != 1 {
		t.Error("expected a message to be sent on selection")
	}

	msg, ok := rl.msgs[0].(RequestSelectedMsg)
	if !ok {
		t.Errorf("expected RequestSelectedMsg, got %T", rl.msgs[0])
	}

	if msg.Request.Name != "Create User" {
		t.Errorf("expected selected request 'Create User', got '%s'", msg.Request.Name)
	}
}

// TestRequestList_Filter tests that '/' opens filter and typing filters by name
func TestRequestList_Filter(t *testing.T) {
	db := &mockDB{
		requests: []*types.SavedRequest{
			{ID: "1", Name: "Get Users", Method: "GET", URL: "https://api.example.com/users"},
			{ID: "2", Name: "Get Posts", Method: "GET", URL: "https://api.example.com/posts"},
			{ID: "3", Name: "Create User", Method: "POST", URL: "https://api.example.com/users"},
		},
	}

	rl := NewRequestList(db)

	// Manually test filtering state
	rl.filtering = true
	rl.list.SetFilteringEnabled(true)

	// Filter items
	rl.FilterItems("Get")

	// Check that filter text was set
	if !strings.Contains(rl.filterText, "Get") {
		t.Errorf("expected filter text to contain 'Get', got '%s'", rl.filterText)
	}

	// Reset filter
	rl.filtering = false
	rl.filterText = ""
	rl.list.ResetFilter()
	if rl.filtering {
		t.Error("expected filtering to be false after reset")
	}
}

// TestRequestList_FolderTree tests folder hierarchy with expand/collapse
func TestRequestList_FolderTree(t *testing.T) {
	db := &mockDB{
		requests: []*types.SavedRequest{
			{ID: "1", Name: "Get Users", Method: "GET", URL: "https://api.example.com/users", Folder: "api/users"},
			{ID: "2", Name: "Create User", Method: "POST", URL: "https://api.example.com/users", Folder: "api/users"},
			{ID: "3", Name: "Get Posts", Method: "GET", URL: "https://api.example.com/posts", Folder: "api/posts"},
		},
	}

	rl := NewRequestList(db)

	// Build folder tree
	rl.buildFolderTree()

	// Check that "api" folder exists as root (not "api/users" or "api/posts")
	apiFolder, ok := rl.folders["api"]
	if !ok {
		t.Fatal("expected 'api' folder to exist")
	}

	// Check that "api" has 2 children (users and posts subfolders)
	if len(apiFolder.Children) != 2 {
		t.Errorf("expected 'api' folder to have 2 children, got %d", len(apiFolder.Children))
	}

	// Check children have correct names
	if _, hasUsers := apiFolder.Children["users"]; !hasUsers {
		t.Error("expected 'users' child to exist")
	}
	if _, hasPosts := apiFolder.Children["posts"]; !hasPosts {
		t.Error("expected 'posts' child to exist")
	}

	// Check collapsed state initially
	if !apiFolder.Collapsed {
		t.Error("expected folder to be collapsed initially")
	}

	// Expand folder
	rl.toggleFolder("api")
	if apiFolder.Collapsed {
		t.Error("expected folder to be expanded after toggle")
	}

	// Collapse folder
	rl.toggleFolder("api")
	if !apiFolder.Collapsed {
		t.Error("expected folder to be collapsed after second toggle")
	}
}

// TestRequestList_MethodColor tests HTTP method color mapping
func TestRequestList_MethodColor(t *testing.T) {
	tests := []struct {
		method   string
		expected lipgloss.Color
	}{
		{"GET", lipgloss.Color("green")},
		{"POST", lipgloss.Color("blue")},
		{"PUT", lipgloss.Color("yellow")},
		{"DELETE", lipgloss.Color("red")},
		{"PATCH", lipgloss.Color("magenta")},
		{"HEAD", lipgloss.Color("cyan")},
		{"OPTIONS", lipgloss.Color("cyan")},
	}

	for _, tt := range tests {
		color := getMethodColor(tt.method)
		if color != tt.expected {
			t.Errorf("getMethodColor(%s): expected %v, got %v", tt.method, tt.expected, color)
		}
	}
}

// TestRequestList_CollectionGroup tests requests are grouped by collection
func TestRequestList_CollectionGroup(t *testing.T) {
	db := &mockDB{
		requests: []*types.SavedRequest{
			{ID: "1", Name: "Get Users", Method: "GET", URL: "https://api.example.com/users", Collection: "Users API"},
			{ID: "2", Name: "Create User", Method: "POST", URL: "https://api.example.com/users", Collection: "Users API"},
			{ID: "3", Name: "Get Posts", Method: "GET", URL: "https://api.example.com/posts", Collection: "Posts API"},
			{ID: "4", Name: "No Collection", Method: "GET", URL: "https://api.example.com/other"},
		},
	}

	rl := NewRequestList(db)

	// Group by collection
	rl.groupByCollection()

	// Check collections were created - should have "Users API", "Posts API", and "" (no collection)
	if len(rl.collections) != 3 {
		t.Errorf("expected 3 collections (Users API, Posts API, nil), got %d", len(rl.collections))
	}

	// Check Users API has 2 requests
	usersCollection, ok := rl.collections["Users API"]
	if !ok {
		t.Fatal("expected 'Users API' collection to exist")
	}

	if len(usersCollection.Requests) != 2 {
		t.Errorf("expected 'Users API' to have 2 requests, got %d", len(usersCollection.Requests))
	}

	// Check Posts API has 1 request
	postsCollection, ok := rl.collections["Posts API"]
	if !ok {
		t.Fatal("expected 'Posts API' collection to exist")
	}

	if len(postsCollection.Requests) != 1 {
		t.Errorf("expected 'Posts API' to have 1 request, got %d", len(postsCollection.Requests))
	}
}

// TestRequestList_Empty tests empty state shows "No requests. Save one first."
func TestRequestList_Empty(t *testing.T) {
	db := &mockDB{
		requests: []*types.SavedRequest{},
	}

	rl := NewRequestList(db)

	if len(rl.items) != 0 {
		t.Errorf("expected 0 items for empty DB, got %d", len(rl.items))
	}

	// Check empty state view
	view := rl.emptyStateView()
	if !strings.Contains(view, "No requests") {
		t.Errorf("expected empty state to contain 'No requests', got '%s'", view)
	}
}
