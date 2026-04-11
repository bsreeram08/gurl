package tui

import (
	"strings"
	"testing"

	"charm.land/bubbletea/v2"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

type MockDB struct {
	requests []*types.SavedRequest
}

func (m *MockDB) Open() error  { return nil }
func (m *MockDB) Close() error { return nil }
func (m *MockDB) SaveRequest(req *types.SavedRequest) error {
	if req.ID == "" {
		req.ID = "mock-id-1"
	}
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

func strKey(s string) tea.KeyPressMsg {
	if len(s) == 0 {
		return tea.KeyPressMsg{}
	}
	return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
}

func ctrlKey(ch rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: rune(int('a') + int(ch-'a')), Mod: tea.ModCtrl}
}

func TestApp_NewApp(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}

	app := NewApp(db, config)

	if app == nil {
		t.Fatal("NewApp returned nil")
	}
	if app.db != db {
		t.Error("db not set correctly")
	}
	if app.config != config {
		t.Error("config not set correctly")
	}
	if app.focusedPanel != PanelSidebar {
		t.Errorf("focusedPanel should be PanelSidebar, got %v", app.focusedPanel)
	}
	if app.width != 80 {
		t.Errorf("width should be 80, got %d", app.width)
	}
	if app.height != 24 {
		t.Errorf("height should be 24, got %d", app.height)
	}
	if app.sidebarMode != SidebarRequests {
		t.Errorf("sidebarMode should be SidebarRequests, got %v", app.sidebarMode)
	}
	if len(app.tabs) != 1 {
		t.Errorf("should have 1 initial tab, got %d", len(app.tabs))
	}
	if app.activeTab != 0 {
		t.Errorf("activeTab should be 0, got %d", app.activeTab)
	}
	if app.requestList == nil {
		t.Error("requestList should be initialized")
	}

	var _ tea.Model = app
}

func TestApp_Init(t *testing.T) {
	db := &MockDB{requests: []*types.SavedRequest{
		{ID: "1", Name: "Test Request", URL: "https://example.com"},
	}}
	config := &types.Config{}
	app := NewApp(db, config)

	cmd := app.Init()
	if cmd == nil {
		t.Error("Init should return a command for initial load")
	}

	msg := cmd()
	loadedMsg, ok := msg.(requestsLoadedMsg)
	if !ok {
		t.Fatalf("expected requestsLoadedMsg, got %T", msg)
	}

	if len(loadedMsg.requests) != 1 {
		t.Errorf("expected 1 request, got %d", len(loadedMsg.requests))
	}
}

func TestApp_OpenNewTab(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	initialCount := len(app.tabs)
	app.openNewTab()

	if len(app.tabs) != initialCount+1 {
		t.Errorf("expected %d tabs, got %d", initialCount+1, len(app.tabs))
	}
	if app.activeTab != len(app.tabs)-1 {
		t.Errorf("activeTab should be %d, got %d", len(app.tabs)-1, app.activeTab)
	}
}

func TestApp_OpenInNewTab(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	req := &types.SavedRequest{ID: "test-id", Name: "Test Request", URL: "https://example.com"}
	initialCount := len(app.tabs)

	app.openInNewTab(req)

	if len(app.tabs) != initialCount+1 {
		t.Errorf("expected %d tabs, got %d", initialCount+1, len(app.tabs))
	}

	newTab := app.tabs[len(app.tabs)-1]
	if newTab.Request != req {
		t.Error("tab should have the request set")
	}
}

func TestApp_CloseActiveTab(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.openNewTab()
	app.openNewTab()

	initialCount := len(app.tabs)
	app.activeTab = 1
	app.closeActiveTab()

	if len(app.tabs) != initialCount-1 {
		t.Errorf("expected %d tabs, got %d", initialCount-1, len(app.tabs))
	}
}

func TestApp_CloseActiveTab_KeepsOne(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.closeActiveTab()

	if len(app.tabs) != 1 {
		t.Errorf("closing last tab should keep one, got %d", len(app.tabs))
	}
}

func TestApp_CloseActiveTab_AdjustsIndex(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.openNewTab()
	app.openNewTab()
	app.activeTab = 0
	originalID := app.tabs[0].ID

	app.closeActiveTab()

	if app.activeTab != 0 {
		t.Errorf("activeTab should be 0, got %d", app.activeTab)
	}
	if len(app.tabs) == 1 && app.tabs[0].ID == originalID {
		t.Error("tab should have been replaced when closing last")
	}
}

func TestApp_NextTab(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.openNewTab()
	app.openNewTab()
	app.activeTab = 0

	app.nextTab()

	if app.activeTab != 1 {
		t.Errorf("expected activeTab 1, got %d", app.activeTab)
	}
}

func TestApp_NextTab_Wraps(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.openNewTab()
	app.activeTab = len(app.tabs) - 1

	app.nextTab()

	if app.activeTab != 0 {
		t.Errorf("expected activeTab 0 after wrap, got %d", app.activeTab)
	}
}

func TestApp_PrevTab(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.openNewTab()
	app.activeTab = 1

	app.prevTab()

	if app.activeTab != 0 {
		t.Errorf("expected activeTab 0, got %d", app.activeTab)
	}
}

func TestApp_PrevTab_Wraps(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.openNewTab()
	app.activeTab = 0

	app.prevTab()

	if app.activeTab != len(app.tabs)-1 {
		t.Errorf("expected activeTab %d after wrap, got %d", len(app.tabs)-1, app.activeTab)
	}
}

func TestApp_TabCount(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	if app.TabCount() != 1 {
		t.Errorf("initial TabCount should be 1, got %d", app.TabCount())
	}

	app.openNewTab()
	if app.TabCount() != 2 {
		t.Errorf("TabCount should be 2, got %d", app.TabCount())
	}

	app.closeActiveTab()
	if app.TabCount() != 1 {
		t.Errorf("TabCount should be 1, got %d", app.TabCount())
	}
}

func TestApp_ActiveTab(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	tab := app.ActiveTab()
	if tab == nil {
		t.Error("ActiveTab should not return nil")
	}
}

func TestApp_PanelFocus_TabCyclesForward(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	if app.focusedPanel != PanelSidebar {
		t.Errorf("initial focus should be Sidebar, got %v", app.focusedPanel)
	}

	app.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if app.focusedPanel != PanelMain {
		t.Errorf("after Tab, focus should be Main, got %v", app.focusedPanel)
	}

	app.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if app.focusedPanel != PanelStatusbar {
		t.Errorf("after second Tab, focus should be Statusbar, got %v", app.focusedPanel)
	}

	app.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if app.focusedPanel != PanelSidebar {
		t.Errorf("after third Tab, focus should cycle to Sidebar, got %v", app.focusedPanel)
	}
}

func TestApp_PanelFocus_ShiftTabCyclesReverse(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if app.focusedPanel != PanelStatusbar {
		t.Errorf("after Shift+Tab, focus should be Statusbar, got %v", app.focusedPanel)
	}

	app.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if app.focusedPanel != PanelMain {
		t.Errorf("after second Shift+Tab, focus should be Main, got %v", app.focusedPanel)
	}

	app.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if app.focusedPanel != PanelSidebar {
		t.Errorf("after third Shift+Tab, focus should be Sidebar, got %v", app.focusedPanel)
	}
}

func TestApp_Keyboard_ctrlT_OpensNewTab(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	initial := app.TabCount()
	app.Update(ctrlKey('t'))

	if app.TabCount() != initial+1 {
		t.Errorf("Ctrl+T should add a new tab")
	}
}

func TestApp_Keyboard_ctrlW_ClosesTab(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.openNewTab()
	initial := app.TabCount()

	app.Update(ctrlKey('w'))

	if app.TabCount() != initial-1 {
		t.Errorf("Ctrl+W should close a tab")
	}
}

func TestApp_Keyboard_ctrlW_KeepsOneTab(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.Update(ctrlKey('w'))

	if app.TabCount() != 1 {
		t.Errorf("Ctrl+W with single tab should keep one tab")
	}
}

func TestApp_Keyboard_ctrlD_DuplicatesTab(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	req := &types.SavedRequest{ID: "dup-test", Name: "Original", URL: "https://example.com"}
	app.openInNewTab(req)

	initial := app.TabCount()
	app.Update(ctrlKey('d'))

	if app.TabCount() != initial+1 {
		t.Errorf("Ctrl+D should duplicate tab")
	}

	newTab := app.tabs[len(app.tabs)-1]
	if newTab.Request == nil {
		t.Errorf("duplicated tab should have a request")
	} else if !strings.Contains(newTab.Name, "(copy)") {
		t.Errorf("duplicated tab should have '(copy)' suffix, got %q", newTab.Name)
	}
}

func TestApp_Keyboard_ctrlD_NoRequestNoDuplicate(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	initial := app.TabCount()
	app.Update(ctrlKey('d'))

	if app.TabCount() != initial {
		t.Error("Ctrl+D on blank tab should not duplicate")
	}
}

func TestApp_Keyboard_q_Quits(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	_, cmd := app.Update(strKey("q"))

	if !app.quitting {
		t.Error("App should be in quitting state after 'q'")
	}
	if cmd == nil {
		t.Error("Update with 'q' should return tea.Quit command")
	}
}

func TestApp_Keyboard_ctrlC_Quits(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	_, cmd := app.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})

	if !app.quitting {
		t.Error("App should be in quitting state after Ctrl+C")
	}
	if cmd == nil {
		t.Error("Update with Ctrl+C should return tea.Quit command")
	}
}

func TestApp_Keyboard_h_SwitchesToHistory(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	if app.sidebarMode != SidebarRequests {
		t.Error("initial sidebarMode should be SidebarRequests")
	}

	app.Update(strKey("h"))

	if app.sidebarMode != SidebarHistory {
		t.Errorf("after 'h', sidebarMode should be SidebarHistory, got %v", app.sidebarMode)
	}
}

func TestApp_Keyboard_r_SwitchesToRequests(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.sidebarMode = SidebarHistory
	app.Update(strKey("r"))

	if app.sidebarMode != SidebarRequests {
		t.Errorf("after 'r', sidebarMode should be SidebarRequests, got %v", app.sidebarMode)
	}
}

func TestApp_Keyboard_question_TogglesHelp(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	if app.helpModal {
		t.Error("helpModal should initially be false")
	}

	app.Update(strKey("?"))

	if !app.helpModal {
		t.Error("helpModal should be true after '?'")
	}

	app.Update(strKey("?"))

	if app.helpModal {
		t.Error("helpModal should be false after second '?'")
	}
}

func TestApp_Keyboard_helpModal_Esc_Closes(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.helpModal = true
	app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})

	if app.helpModal {
		t.Error("helpModal should be false after Escape")
	}
}

func TestApp_Keyboard_ctrlShiftBrackets(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.openNewTab()
	app.openNewTab()
	app.activeTab = 0

	app.nextTab()
	if app.activeTab != 1 {
		t.Errorf("nextTab should go to next tab, got %d", app.activeTab)
	}

	app.prevTab()
	if app.activeTab != 0 {
		t.Errorf("prevTab should go to prev tab, got %d", app.activeTab)
	}
}

func TestApp_Keyboard_ctrlK_OpensSearchModal(t *testing.T) {
	db := &MockDB{requests: []*types.SavedRequest{
		{ID: "1", Name: "Test Request", URL: "https://example.com"},
	}}
	config := &types.Config{}
	app := NewApp(db, config)

	app.width = 100
	app.height = 40

	app.Update(ctrlKey('k'))

	if app.searchModal == nil {
		t.Error("Ctrl+K should open search modal")
	}
}

func TestApp_SearchModal_FiltersResults(t *testing.T) {
	db := &MockDB{requests: []*types.SavedRequest{
		{ID: "1", Name: "Get Users", URL: "https://api.example.com/users"},
		{ID: "2", Name: "Create User", URL: "https://api.example.com/users"},
		{ID: "3", Name: "Get Posts", URL: "https://api.example.com/posts"},
	}}
	config := &types.Config{}
	app := NewApp(db, config)

	app.width = 100
	app.height = 40

	app.searchModal = NewSearchModal(db.requests, app.width, app.height)

	app.searchModal.input.SetValue("Get")
	app.searchModal.filterResults()

	if len(app.searchModal.results) != 2 {
		t.Errorf("expected 2 results for 'Get', got %d", len(app.searchModal.results))
	}

	app.searchModal.input.SetValue("posts")
	app.searchModal.filterResults()

	if len(app.searchModal.results) != 1 {
		t.Errorf("expected 1 result for 'posts', got %d", len(app.searchModal.results))
	}

	app.searchModal.input.SetValue("")
	app.searchModal.filterResults()

	if len(app.searchModal.results) != 3 {
		t.Errorf("expected 3 results for empty filter, got %d", len(app.searchModal.results))
	}
}

func TestApp_SearchModal_EscapeCloses(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.width = 100
	app.height = 40

	app.searchModal = NewSearchModal(nil, app.width, app.height)

	_, cmd := app.searchModal.Update(tea.KeyPressMsg{Code: tea.KeyEscape})

	if cmd != nil {
		t.Error("SearchModal.Update should return nil on Escape")
	}
}

func TestApp_Keyboard_ctrlI_OpensImportModal(t *testing.T) {
	t.Skip("Ctrl+I conflicts with Tab in terminals - tested manually")
}

func TestApp_ImportModal_EscapeCloses(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.width = 100
	app.height = 40

	app.importModal = NewImportModal(app.width, app.height)

	result := app.importModal.Update(tea.KeyPressMsg{Code: tea.KeyEscape})

	if result != nil {
		t.Error("ImportModal.Update should return nil on Escape")
	}
}

func TestApp_CalculateLayout(t *testing.T) {
	layout := CalculateLayout(120, 40)

	if layout.SidebarWidth < 30 {
		t.Errorf("SidebarWidth should be at least 30, got %d", layout.SidebarWidth)
	}

	if layout.StatusHeight != 1 {
		t.Errorf("StatusHeight should be 1, got %d", layout.StatusHeight)
	}

	if layout.SidebarWidth+layout.MainWidth != 120 {
		t.Errorf("SidebarWidth + MainWidth should equal total width")
	}
}

func TestApp_CalculateLayout_SmallTerminal(t *testing.T) {
	layout := CalculateLayout(40, 10)

	if layout.SidebarWidth < 30 {
		t.Errorf("SidebarWidth should be at least 30 even on small terminals")
	}
}

func TestApp_View_NotReady(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	view := app.View()
	if !strings.Contains(view.Content, "Loading...") {
		t.Errorf("View before Init should contain 'Loading...', got %q", view.Content)
	}
}

func TestApp_View_WithSize(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := app.View()
	if strings.Contains(view.Content, "Loading...") {
		t.Error("View after size set should not contain 'Loading...' (still loading)")
	}
}

func TestApp_View_HelpModal(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app.helpModal = true

	view := app.View()

	found := false
	for _, line := range []string{"Ctrl+T", "New tab", "Ctrl+W", "Ctrl+K"} {
		if containsStr(view.Content, line) {
			found = true
			break
		}
	}
	if !found {
		t.Error("Help modal should show keyboard shortcuts")
	}
}

func TestApp_View_SearchModal(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app.searchModal = NewSearchModal(nil, app.width, app.height)

	view := app.View()

	if !containsStr(view.Content, "Search") {
		t.Error("View should show search modal")
	}
}

func TestApp_View_ImportModal(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app.importModal = NewImportModal(app.width, app.height)

	view := app.View()

	if !containsStr(view.Content, "Import") {
		t.Error("View should show import modal")
	}
}

func TestApp_NavigateWithNoSidebarFocus(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.focusedPanel = PanelMain

	app.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	app.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	if app.focusedPanel != PanelMain {
		t.Error("focusedPanel should not change when not in sidebar")
	}
}

func TestApp_SpaceToggleFolder(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app.focusedPanel = PanelSidebar
	app.sidebarMode = SidebarRequests

	app.Update(tea.KeyPressMsg{Code: tea.KeySpace})

	if app.focusedPanel != PanelSidebar {
		t.Error("focusedPanel should not change when handling Space")
	}
}

func TestApp_GetRequestCount(t *testing.T) {
	db := &MockDB{requests: []*types.SavedRequest{
		{ID: "1"},
		{ID: "2"},
		{ID: "3"},
	}}
	config := &types.Config{}
	app := NewApp(db, config)

	cmd := app.Init()
	if cmd != nil {
		if msg, ok := cmd().(requestsLoadedMsg); ok {
			updated, _ := app.Update(msg)
			if a, ok := updated.(*App); ok {
				app = a
			}
		}
	}

	if app.GetRequestCount() != 3 {
		t.Errorf("GetRequestCount should return 3, got %d", app.GetRequestCount())
	}
}

func TestApp_GetCurrentEnv(t *testing.T) {
	db := &MockDB{}
	config := &types.Config{}
	app := NewApp(db, config)

	env := app.GetCurrentEnv()
	if env != "default" {
		t.Errorf("GetCurrentEnv should return 'default', got %q", env)
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
