package tui

import (
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

// MockDB implements storage.DB for testing
type MockDB struct {
	requests []*types.SavedRequest
}

func (m *MockDB) Open() error  { return nil }
func (m *MockDB) Close() error { return nil }
func (m *MockDB) SaveRequest(req *types.SavedRequest) error {
	m.requests = append(m.requests, req)
	return nil
}
func (m *MockDB) GetRequest(id string) (*types.SavedRequest, error) {
	for _, r := range m.requests {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, nil
}
func (m *MockDB) GetRequestByName(name string) (*types.SavedRequest, error) {
	for _, r := range m.requests {
		if r.Name == name {
			return r, nil
		}
	}
	return nil, nil
}
func (m *MockDB) ListRequests(opts *storage.ListOptions) ([]*types.SavedRequest, error) {
	return m.requests, nil
}
func (m *MockDB) DeleteRequest(id string) error                     { return nil }
func (m *MockDB) UpdateRequest(req *types.SavedRequest) error       { return nil }
func (m *MockDB) SaveHistory(history *types.ExecutionHistory) error { return nil }
func (m *MockDB) GetHistory(requestID string, limit int) ([]*types.ExecutionHistory, error) {
	return nil, nil
}
func (m *MockDB) ListFolder(path string) ([]*types.SavedRequest, error)          { return nil, nil }
func (m *MockDB) ListFolderRecursive(path string) ([]*types.SavedRequest, error) { return nil, nil }
func (m *MockDB) DeleteFolder(path string) error                                 { return nil }
func (m *MockDB) GetAllFolders() ([]string, error)                               { return nil, nil }

func TestTUI_Init(t *testing.T) {
	db := &MockDB{requests: []*types.SavedRequest{
		{ID: "1", Name: "Test Request", URL: "https://example.com"},
	}}
	config := &types.Config{}

	app := NewApp(db, config)

	if app == nil {
		t.Fatal("NewApp returned nil")
	}

	if app.db == nil {
		t.Error("db was not set")
	}

	if app.config == nil {
		t.Error("config was not set")
	}

	// App should implement tea.Model
	var _ tea.Model = app

	// Init should return a command that loads requests from DB
	cmd := app.Init()
	if cmd == nil {
		t.Error("Init should return a command for initial load")
	}
}

func TestTUI_QuitOnQ(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}

	app := NewApp(db, config)

	// Create a quit message - "q" key
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("q"),
	}

	newModel, cmd := app.Update(msg)

	// Should return a non-nil command (tea.Quit)
	if cmd == nil {
		t.Error("Update with 'q' key should return a command (tea.Quit)")
	}

	// Model should still be the same app
	if newModel != app {
		t.Error("Model should be the same app instance")
	}

	// App should be in quitting state
	if !app.quitting {
		t.Error("App should be in quitting state after 'q'")
	}
}

func TestTUI_QuitOnCtrlC(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}

	app := NewApp(db, config)

	// Create Ctrl+C message
	msg := tea.KeyMsg{
		Type: tea.KeyCtrlC,
	}

	newModel, cmd := app.Update(msg)

	// Should return a non-nil command (tea.Quit)
	if cmd == nil {
		t.Error("Update with Ctrl+C should return a command (tea.Quit)")
	}

	// Model should still be the same app
	if newModel != app {
		t.Error("Model should be the same app instance")
	}

	// App should be in quitting state
	if !app.quitting {
		t.Error("App should be in quitting state after Ctrl+C")
	}
}

func TestTUI_Layout(t *testing.T) {
	db := &MockDB{requests: []*types.SavedRequest{
		{ID: "1", Name: "Request 1", URL: "https://example.com/1"},
		{ID: "2", Name: "Request 2", URL: "https://example.com/2"},
	}}
	config := &types.Config{}

	app := NewApp(db, config)

	// Simulate window size message
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	app.Update(msg)

	// Calculate layout
	layout := CalculateLayout(120, 40)

	// Verify layout dimensions
	if layout.SidebarWidth < 30 {
		t.Errorf("SidebarWidth should be at least 30, got %d", layout.SidebarWidth)
	}

	if layout.StatusHeight != 1 {
		t.Errorf("StatusHeight should be 1, got %d", layout.StatusHeight)
	}

	// Main width should be the remainder
	expectedMainWidth := 120 - layout.SidebarWidth
	if layout.MainWidth != expectedMainWidth {
		t.Errorf("MainWidth should be %d, got %d", expectedMainWidth, layout.MainWidth)
	}
}

func TestTUI_Resize(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}

	app := NewApp(db, config)

	// Initial size
	msg1 := tea.WindowSizeMsg{Width: 100, Height: 30}
	app.Update(msg1)

	layout1 := CalculateLayout(100, 30)
	sidebarWidth1 := layout1.SidebarWidth

	// Resize to larger terminal
	msg2 := tea.WindowSizeMsg{Width: 200, Height: 50}
	app.Update(msg2)

	layout2 := CalculateLayout(200, 50)
	sidebarWidth2 := layout2.SidebarWidth

	// Sidebar should scale proportionally but respect minimum
	if sidebarWidth2 <= sidebarWidth1 {
		t.Errorf("SidebarWidth should increase with terminal size, was %d now %d", sidebarWidth1, sidebarWidth2)
	}

	// Main width should adjust
	if layout2.MainWidth <= layout1.MainWidth {
		t.Errorf("MainWidth should increase with terminal size")
	}

	// Resize to very small terminal (minimum enforced)
	layout3 := CalculateLayout(40, 10)

	// Sidebar should at least be 30
	if layout3.SidebarWidth < 30 {
		t.Errorf("SidebarWidth should be at least 30 even on small terminals, got %d", layout3.SidebarWidth)
	}
}

func TestTUI_FocusSwitch(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}

	app := NewApp(db, config)

	// Initial focus should be sidebar
	if app.focusedPanel != PanelSidebar {
		t.Errorf("Initial focus should be sidebar, got %v", app.focusedPanel)
	}

	// Send Tab key
	msg := tea.KeyMsg{
		Type: tea.KeyTab,
	}

	app.Update(msg)

	// Focus should move to main
	if app.focusedPanel != PanelMain {
		t.Errorf("After Tab, focus should be main, got %v", app.focusedPanel)
	}

	// Send Tab again
	app.Update(msg)

	// Focus should move to status (if applicable) or cycle back
	// For 3-panel, Tab should cycle: sidebar -> main -> statusbar -> sidebar
	if app.focusedPanel != PanelStatusbar {
		t.Errorf("After second Tab, focus should be statusbar, got %v", app.focusedPanel)
	}

	// Third Tab should cycle back to sidebar
	app.Update(msg)
	if app.focusedPanel != PanelSidebar {
		t.Errorf("After third Tab, focus should cycle back to sidebar, got %v", app.focusedPanel)
	}
}

func TestTUI_StatusBar(t *testing.T) {
	db := &MockDB{requests: []*types.SavedRequest{
		{ID: "1", Name: "Request 1", URL: "https://example.com/1"},
		{ID: "2", Name: "Request 2", URL: "https://example.com/2"},
		{ID: "3", Name: "Request 3", URL: "https://example.com/3"},
	}}
	config := &types.Config{}

	app := NewApp(db, config)

	// Create status bar
	statusBar := NewStatusBar(app)

	// Verify it has the expected content
	view := statusBar.View()

	// Should contain gurl version
	if !contains(view, "gurl") {
		t.Error("StatusBar should contain 'gurl'")
	}

	// Should contain request count
	if !contains(view, "3") && !contains(view, "request") {
		t.Error("StatusBar should contain request count")
	}

	// Should show current environment
	if !contains(view, "env") && !contains(view, "default") {
		t.Error("StatusBar should show environment")
	}
}

// Helper to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
