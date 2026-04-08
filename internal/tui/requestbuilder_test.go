package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

// mockDB implements storage.DB for testing RequestBuilder
type mockBuilderDB struct {
	requests []*types.SavedRequest
	saved    []*types.SavedRequest
}

func (m *mockBuilderDB) Open() error  { return nil }
func (m *mockBuilderDB) Close() error { return nil }
func (m *mockBuilderDB) SaveRequest(req *types.SavedRequest) error {
	m.saved = append(m.saved, req)
	return nil
}
func (m *mockBuilderDB) GetRequest(id string) (*types.SavedRequest, error) {
	for _, r := range m.requests {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, nil
}
func (m *mockBuilderDB) GetRequestByName(name string) (*types.SavedRequest, error) {
	for _, r := range m.requests {
		if r.Name == name {
			return r, nil
		}
	}
	return nil, nil
}
func (m *mockBuilderDB) ListRequests(opts *storage.ListOptions) ([]*types.SavedRequest, error) {
	return m.requests, nil
}
func (m *mockBuilderDB) DeleteRequest(id string) error { return nil }
func (m *mockBuilderDB) UpdateRequest(req *types.SavedRequest) error {
	// Find and update
	for i, r := range m.requests {
		if r.ID == req.ID {
			m.requests[i] = req
			break
		}
	}
	// Also record in saved (real DB calls SaveRequest internally for Update)
	m.saved = append(m.saved, req)
	return nil
}
func (m *mockBuilderDB) SaveHistory(history *types.ExecutionHistory) error { return nil }
func (m *mockBuilderDB) GetHistory(requestID string, limit int) ([]*types.ExecutionHistory, error) {
	return nil, nil
}
func (m *mockBuilderDB) ListFolder(path string) ([]*types.SavedRequest, error) { return nil, nil }
func (m *mockBuilderDB) ListFolderRecursive(path string) ([]*types.SavedRequest, error) {
	return nil, nil
}
func (m *mockBuilderDB) DeleteFolder(path string) error   { return nil }
func (m *mockBuilderDB) GetAllFolders() ([]string, error) { return nil, nil }

// TestBuilder_DisplayRequest tests that a selected request is loaded into the builder
func TestBuilder_DisplayRequest(t *testing.T) {
	db := &mockBuilderDB{
		requests: []*types.SavedRequest{
			{
				ID:      "1",
				Name:    "Get Users",
				Method:  "GET",
				URL:     "https://api.example.com/users",
				Headers: []types.Header{{Key: "Accept", Value: "application/json"}},
				Body:    "",
			},
		},
	}

	rb := NewRequestBuilder(db)

	// Load the request
	req := db.requests[0]
	rb.LoadRequest(req)

	// Verify editing state
	if rb.editing == nil {
		t.Fatal("editing should not be nil after LoadRequest")
	}

	if rb.editing.Method != "GET" {
		t.Errorf("expected method GET, got %s", rb.editing.Method)
	}

	if rb.editing.URL != "https://api.example.com/users" {
		t.Errorf("expected URL https://api.example.com/users, got %s", rb.editing.URL)
	}

	if len(rb.editing.Headers) != 1 {
		t.Errorf("expected 1 header, got %d", len(rb.editing.Headers))
	}

	if rb.editing.Headers[0].Key != "Accept" {
		t.Errorf("expected header key Accept, got %s", rb.editing.Headers[0].Key)
	}

	if rb.editing.Headers[0].Value != "application/json" {
		t.Errorf("expected header value application/json, got %s", rb.editing.Headers[0].Value)
	}

	if rb.editing.Body != "" {
		t.Errorf("expected empty body, got %s", rb.editing.Body)
	}
}

// TestBuilder_EditURL tests URL editing
func TestBuilder_EditURL(t *testing.T) {
	db := &mockBuilderDB{}
	rb := NewRequestBuilder(db)

	// Start with a blank request
	rb.NewRequest()

	// Check initial URL is empty
	if rb.urlInput.Value() != "" {
		t.Errorf("expected empty URL, got %s", rb.urlInput.Value())
	}

	// Set URL via text input
	rb.urlInput.SetValue("https://api.example.com/users")

	// Verify URL was set
	if rb.urlInput.Value() != "https://api.example.com/users" {
		t.Errorf("expected URL to be set, got %s", rb.urlInput.Value())
	}

	// Sync to editing and verify
	rb.syncEditingFromForm()
	if rb.editing.URL != "https://api.example.com/users" {
		t.Errorf("expected editing URL to be https://api.example.com/users, got %s", rb.editing.URL)
	}
}

// TestBuilder_EditMethod tests method cycling
func TestBuilder_EditMethod(t *testing.T) {
	db := &mockBuilderDB{}
	rb := NewRequestBuilder(db)

	rb.NewRequest()

	// Start with GET (index 0)
	if rb.methodIndex != 0 {
		t.Errorf("expected initial method index 0 (GET), got %d", rb.methodIndex)
	}
	if rb.methods[rb.methodIndex] != "GET" {
		t.Errorf("expected method GET, got %s", rb.methods[rb.methodIndex])
	}

	// Cycle to next method (POST)
	rb.methodIndex = (rb.methodIndex + 1) % len(rb.methods)
	if rb.methods[rb.methodIndex] != "POST" {
		t.Errorf("expected method POST, got %s", rb.methods[rb.methodIndex])
	}

	// Cycle to PUT
	rb.methodIndex = (rb.methodIndex + 1) % len(rb.methods)
	if rb.methods[rb.methodIndex] != "PUT" {
		t.Errorf("expected method PUT, got %s", rb.methods[rb.methodIndex])
	}

	// Cycle through all methods via direct index manipulation
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	rb.methodIndex = 0
	for i, expected := range methods {
		if rb.methods[rb.methodIndex] != expected {
			t.Errorf("method cycle index %d: expected %s, got %s", i, expected, rb.methods[rb.methodIndex])
		}
		rb.methodIndex = (rb.methodIndex + 1) % len(rb.methods)
	}

	// After full cycle, should be back at GET
	if rb.methodIndex != 0 {
		t.Errorf("expected methodIndex 0 after full cycle, got %d", rb.methodIndex)
	}

	// Test keyboard handling - cycle via 'm' key press
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")}
	rb.handleMethodKey(msg)

	if rb.methodIndex != 1 {
		t.Errorf("expected method index 1 after 'm' key, got %d", rb.methodIndex)
	}
	if rb.methods[rb.methodIndex] != "POST" {
		t.Errorf("expected POST after 'm' key, got %s", rb.methods[rb.methodIndex])
	}
}

// TestBuilder_AddHeader tests adding header rows
func TestBuilder_AddHeader(t *testing.T) {
	db := &mockBuilderDB{}
	rb := NewRequestBuilder(db)

	rb.NewRequest()

	// Should have 1 header row by default after NewRequest
	if len(rb.headerInputs) != 1 {
		t.Errorf("expected 1 header row after NewRequest, got %d", len(rb.headerInputs))
	}

	// Add a header via 'a' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}
	rb.handleHeadersKey(msg)

	if len(rb.headerInputs) != 2 {
		t.Errorf("expected 2 header rows after 'a' key, got %d", len(rb.headerInputs))
	}

	// Add another
	rb.handleHeadersKey(msg)
	if len(rb.headerInputs) != 3 {
		t.Errorf("expected 3 header rows, got %d", len(rb.headerInputs))
	}

	// Set some values
	rb.headerInputs[0].keyInput.SetValue("Content-Type")
	rb.headerInputs[0].valueInput.SetValue("application/json")

	// Sync and verify
	rb.syncEditingFromForm()
	if len(rb.editing.Headers) != 1 {
		t.Errorf("expected 1 header after sync, got %d", len(rb.editing.Headers))
	}
	if rb.editing.Headers[0].Key != "Content-Type" {
		t.Errorf("expected header key Content-Type, got %s", rb.editing.Headers[0].Key)
	}
}

// TestBuilder_RemoveHeader tests removing header rows
func TestBuilder_RemoveHeader(t *testing.T) {
	db := &mockBuilderDB{}
	rb := NewRequestBuilder(db)

	rb.NewRequest()

	// Add some headers
	rb.addHeaderRow()
	rb.addHeaderRow()

	if len(rb.headerInputs) != 3 {
		t.Fatalf("expected 3 header rows, got %d", len(rb.headerInputs))
	}

	// Set values on all rows
	rb.headerInputs[0].keyInput.SetValue("Header1")
	rb.headerInputs[1].keyInput.SetValue("Header2")
	rb.headerInputs[2].keyInput.SetValue("Header3")

	// Focus the second row
	rb.headerInputs[1].keyInput.Focus()

	// Remove via 'd' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}
	rb.handleHeadersKey(msg)

	// Should have 2 rows now (removed the focused one)
	if len(rb.headerInputs) != 2 {
		t.Errorf("expected 2 header rows after 'd' key, got %d", len(rb.headerInputs))
	}

	// First row should still have Header1
	if rb.headerInputs[0].keyInput.Value() != "Header1" {
		t.Errorf("expected first header to be Header1, got %s", rb.headerInputs[0].keyInput.Value())
	}

	// Third row should have become second and still have Header3
	if rb.headerInputs[1].keyInput.Value() != "Header3" {
		t.Errorf("expected second header to be Header3 (was third), got %s", rb.headerInputs[1].keyInput.Value())
	}
}

// TestBuilder_EditBody tests body editing
func TestBuilder_EditBody(t *testing.T) {
	db := &mockBuilderDB{}
	rb := NewRequestBuilder(db)

	rb.NewRequest()

	// Check initial body is empty
	if rb.bodyInput.Value() != "" {
		t.Errorf("expected empty body, got %s", rb.bodyInput.Value())
	}

	// Set body
	body := `{"name": "test", "value": 123}`
	rb.bodyInput.SetValue(body)

	if rb.bodyInput.Value() != body {
		t.Errorf("expected body to be set, got %s", rb.bodyInput.Value())
	}

	// Sync to editing
	rb.syncEditingFromForm()
	if rb.editing.Body != body {
		t.Errorf("expected editing body to be %s, got %s", body, rb.editing.Body)
	}

	// Check content type detection
	contentType := detectContentType(rb.editing.Headers, body)
	if contentType != "json" {
		t.Errorf("expected content type 'json', got %s", contentType)
	}
}

// TestBuilder_SendRequest tests the send functionality
func TestBuilder_SendRequest(t *testing.T) {
	db := &mockBuilderDB{}
	rb := NewRequestBuilder(db)

	rb.NewRequest()
	rb.urlInput.SetValue("https://api.example.com/test")
	rb.methodIndex = 0 // GET

	// Initially not sending
	if rb.sending {
		t.Error("expected sending to be false initially")
	}

	// Check that sendRequest doesn't panic and returns nil (async)
	cmd := rb.sendRequest()
	if cmd != nil {
		t.Error("sendRequest should return nil cmd (async operation)")
	}

	// Sending should be true now
	if !rb.sending {
		t.Error("expected sending to be true after sendRequest")
	}
}

// TestBuilder_SaveChanges tests saving request to DB
func TestBuilder_SaveChanges(t *testing.T) {
	db := &mockBuilderDB{}
	rb := NewRequestBuilder(db)

	// Create a request and modify it
	rb.NewRequest()
	rb.urlInput.SetValue("https://api.example.com/save-test")
	rb.methodIndex = 2 // PUT
	rb.bodyInput.SetValue(`{"saved": true}`)

	// Save should work for new request
	cmd := rb.saveRequest()
	if cmd != nil {
		t.Error("saveRequest should return nil cmd")
	}

	// Check that DB was called
	if len(db.saved) != 1 {
		t.Errorf("expected 1 saved request, got %d", len(db.saved))
	}

	savedReq := db.saved[0]
	if savedReq.URL != "https://api.example.com/save-test" {
		t.Errorf("expected URL https://api.example.com/save-test, got %s", savedReq.URL)
	}

	if savedReq.Method != "PUT" {
		t.Errorf("expected method PUT, got %s", savedReq.Method)
	}

	if savedReq.Body != `{"saved": true}` {
		t.Errorf("expected body {\"saved\": true}, got %s", savedReq.Body)
	}

	// Test updating existing request
	existingReq := &types.SavedRequest{
		ID:      "existing-1",
		Name:    "Existing",
		Method:  "GET",
		URL:     "https://api.example.com/old",
		Headers: []types.Header{},
	}
	db.requests = []*types.SavedRequest{existingReq}

	rb.LoadRequest(existingReq)
	rb.urlInput.SetValue("https://api.example.com/updated")
	rb.syncEditingFromForm()

	// Save should call UpdateRequest, not SaveRequest
	cmd = rb.saveRequest()
	if cmd != nil {
		t.Error("saveRequest should return nil cmd")
	}

	// saved should have 2 items (new + updated existing)
	if len(db.saved) != 2 {
		t.Errorf("expected 2 saved requests, got %d", len(db.saved))
	}
}

// TestBuilder_NewRequest tests creating a new blank request
func TestBuilder_NewRequest(t *testing.T) {
	db := &mockBuilderDB{}
	rb := NewRequestBuilder(db)

	// First load an existing request
	existingReq := &types.SavedRequest{
		ID:      "1",
		Name:    "Existing",
		Method:  "POST",
		URL:     "https://api.example.com/existing",
		Headers: []types.Header{{Key: "X-Custom", Value: "header"}},
		Body:    `{"existing": true}`,
	}
	rb.LoadRequest(existingReq)

	// Now create new request
	rb.NewRequest()

	// Should be blank
	if rb.request != nil {
		t.Error("expected request to be nil for new request")
	}

	if rb.editing == nil {
		t.Fatal("editing should not be nil after NewRequest")
	}

	if rb.editing.Name != "New Request" {
		t.Errorf("expected name 'New Request', got %s", rb.editing.Name)
	}

	if rb.editing.Method != "GET" {
		t.Errorf("expected method GET, got %s", rb.editing.Method)
	}

	if rb.editing.URL != "" {
		t.Errorf("expected empty URL, got %s", rb.editing.URL)
	}

	if len(rb.editing.Headers) != 0 {
		t.Errorf("expected 0 headers, got %d", len(rb.editing.Headers))
	}

	// URL input should be empty
	if rb.urlInput.Value() != "" {
		t.Errorf("expected urlInput to be empty, got %s", rb.urlInput.Value())
	}

	// Method should be reset to GET (index 0)
	if rb.methodIndex != 0 {
		t.Errorf("expected methodIndex 0, got %d", rb.methodIndex)
	}
}

// TestBuilder_Auth tests auth type selection and inputs
func TestBuilder_Auth(t *testing.T) {
	db := &mockBuilderDB{}
	rb := NewRequestBuilder(db)

	rb.NewRequest()

	// Default auth should be none
	if rb.authType != "none" {
		t.Errorf("expected auth type 'none', got %s", rb.authType)
	}

	// Switch to basic auth
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")}
	rb.handleAuthKey(msg)

	if rb.authType != "basic" {
		t.Errorf("expected auth type 'basic', got %s", rb.authType)
	}

	// Set auth values - need to reassign back to map since Model is value type
	usernameInput := rb.authInputs["username"]
	usernameInput.SetValue("testuser")
	rb.authInputs["username"] = usernameInput

	passwordInput := rb.authInputs["password"]
	passwordInput.SetValue("testpass")
	rb.authInputs["password"] = passwordInput

	// Sync and verify
	rb.syncEditingFromForm()
	if rb.editing.AuthConfig == nil {
		t.Fatal("expected AuthConfig to be set")
	}

	if rb.editing.AuthConfig.Type != "basic" {
		t.Errorf("expected auth type 'basic', got %s", rb.editing.AuthConfig.Type)
	}

	if rb.editing.AuthConfig.Params["username"] != "testuser" {
		t.Errorf("expected username 'testuser', got %s", rb.editing.AuthConfig.Params["username"])
	}

	if rb.editing.AuthConfig.Params["password"] != "testpass" {
		t.Errorf("expected password 'testpass', got %s", rb.editing.AuthConfig.Params["password"])
	}

	// Switch to bearer
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")}
	rb.handleAuthKey(msg)

	if rb.authType != "bearer" {
		t.Errorf("expected auth type 'bearer', got %s", rb.authType)
	}

	// Switch to apikey
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")}
	rb.handleAuthKey(msg)

	if rb.authType != "apikey" {
		t.Errorf("expected auth type 'apikey', got %s", rb.authType)
	}

	// Switch back to none
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")}
	rb.handleAuthKey(msg)

	if rb.authType != "none" {
		t.Errorf("expected auth type 'none', got %s", rb.authType)
	}
}

// TestBuilder_SectionNavigation tests Tab and Shift+Tab navigation
func TestBuilder_SectionNavigation(t *testing.T) {
	db := &mockBuilderDB{}
	rb := NewRequestBuilder(db)

	rb.NewRequest()

	// Start at method section
	if rb.activeSection != SectionMethod {
		t.Errorf("expected initial section SectionMethod, got %v", rb.activeSection)
	}

	// Tab should move to URL
	rb.nextSection()
	if rb.activeSection != SectionURL {
		t.Errorf("expected SectionURL after Tab, got %v", rb.activeSection)
	}

	// Tab should move to Headers
	rb.nextSection()
	if rb.activeSection != SectionHeaders {
		t.Errorf("expected SectionHeaders after Tab, got %v", rb.activeSection)
	}

	// Tab should move to QueryParams
	rb.nextSection()
	if rb.activeSection != SectionQueryParams {
		t.Errorf("expected SectionQueryParams after Tab, got %v", rb.activeSection)
	}

	// Tab should move to Body
	rb.nextSection()
	if rb.activeSection != SectionBody {
		t.Errorf("expected SectionBody after Tab, got %v", rb.activeSection)
	}

	// Tab should move to Auth
	rb.nextSection()
	if rb.activeSection != SectionAuth {
		t.Errorf("expected SectionAuth after Tab, got %v", rb.activeSection)
	}

	// Tab should move to Send
	rb.nextSection()
	if rb.activeSection != SectionSend {
		t.Errorf("expected SectionSend after Tab, got %v", rb.activeSection)
	}

	// Tab should cycle back to Method
	rb.nextSection()
	if rb.activeSection != SectionMethod {
		t.Errorf("expected SectionMethod after cycle, got %v", rb.activeSection)
	}

	// Shift+Tab should go backwards
	rb.prevSection()
	if rb.activeSection != SectionSend {
		t.Errorf("expected SectionSend after Shift+Tab, got %v", rb.activeSection)
	}
}

// TestBuilder_QueryParams tests query parameter editing
func TestBuilder_QueryParams(t *testing.T) {
	db := &mockBuilderDB{}
	rb := NewRequestBuilder(db)

	rb.NewRequest()

	// Should have no query params initially
	if len(rb.queryInputs) != 0 {
		t.Errorf("expected 0 query params initially, got %d", len(rb.queryInputs))
	}

	// Add a query param
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}
	rb.handleQueryParamsKey(msg)

	if len(rb.queryInputs) != 1 {
		t.Errorf("expected 1 query param after 'a' key, got %d", len(rb.queryInputs))
	}

	// Set values
	rb.queryInputs[0].keyInput.SetValue("page")
	rb.queryInputs[0].valueInput.SetValue("1")

	// Verify URL building would include query params
	// (Actual URL building includes query params in full URL)
}

// TestBuilder_WelcomeView tests the welcome view when no request is loaded
func TestBuilder_WelcomeView(t *testing.T) {
	db := &mockBuilderDB{}
	rb := NewRequestBuilder(db)

	// Don't load any request
	view := rb.welcomeView()

	if !strings.Contains(view, "Welcome to Gurl TUI!") {
		t.Error("welcome view should contain 'Welcome to Gurl TUI!'")
	}

	if !strings.Contains(view, "sidebar") {
		t.Error("welcome view should mention sidebar")
	}
}

// TestBuilder_View tests the View method with a loaded request
func TestBuilder_View(t *testing.T) {
	db := &mockBuilderDB{}
	rb := NewRequestBuilder(db)

	// Create and load a request
	req := &types.SavedRequest{
		ID:      "1",
		Name:    "Test Request",
		Method:  "POST",
		URL:     "https://api.example.com/test",
		Headers: []types.Header{{Key: "Content-Type", Value: "application/json"}},
		Body:    `{"key": "value"}`,
	}
	rb.LoadRequest(req)

	view := rb.View()

	// Should show the URL
	if !strings.Contains(view, "https://api.example.com/test") {
		t.Error("view should contain the URL")
	}

	// Should show the method
	if !strings.Contains(view, "POST") {
		t.Error("view should contain the method")
	}

	// Should show headers section
	if !strings.Contains(view, "Headers") {
		t.Error("view should contain 'Headers'")
	}

	// Should show Body section
	if !strings.Contains(view, "Body") {
		t.Error("view should contain 'Body'")
	}

	// Should show Send section
	if !strings.Contains(view, "Send") {
		t.Error("view should contain 'Send'")
	}
}

// TestBuilder_GetMessages tests message generation
func TestBuilder_GetMessages(t *testing.T) {
	db := &mockBuilderDB{}
	rb := NewRequestBuilder(db)

	// Initially no messages
	msgs := rb.GetMessages()
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages initially, got %d", len(msgs))
	}

	// Messages should be cleared after GetMessages
	msgs = rb.GetMessages()
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages after GetMessages, got %d", len(msgs))
	}
}

// TestBuilder_LoadingSpinner tests the loading spinner
func TestBuilder_LoadingSpinner(t *testing.T) {
	db := &mockBuilderDB{}
	rb := NewRequestBuilder(db)

	// spinner.Model is a struct, not a pointer - check it was initialized
	// by verifying sending state instead
	if rb.sending {
		t.Error("sending should be false initially")
	}
}

// TestBuilder_IsSending tests the IsSending method
func TestBuilder_IsSending(t *testing.T) {
	db := &mockBuilderDB{}
	rb := NewRequestBuilder(db)

	if rb.IsSending() {
		t.Error("IsSending should be false initially")
	}

	rb.sending = true
	if !rb.IsSending() {
		t.Error("IsSending should be true when sending")
	}
}

// TestBuilder_GetEditingRequest tests getting the current editing request
func TestBuilder_GetEditingRequest(t *testing.T) {
	db := &mockBuilderDB{}
	rb := NewRequestBuilder(db)

	// No editing request initially
	req := rb.GetEditingRequest()
	if req != nil {
		t.Error("expected nil editing request initially")
	}

	// Create a request
	rb.NewRequest()
	rb.urlInput.SetValue("https://api.example.com/get")
	rb.syncEditingFromForm()

	req = rb.GetEditingRequest()
	if req == nil {
		t.Fatal("expected non-nil editing request after NewRequest")
	}

	if req.URL != "https://api.example.com/get" {
		t.Errorf("expected URL https://api.example.com/get, got %s", req.URL)
	}
}

// TestBuilder_BuildClientRequest tests building a client request
func TestBuilder_BuildClientRequest(t *testing.T) {
	db := &mockBuilderDB{}
	rb := NewRequestBuilder(db)

	rb.NewRequest()
	rb.urlInput.SetValue("https://api.example.com/build")
	rb.methodIndex = 0 // GET
	rb.bodyInput.SetValue(`{"built": true}`)

	// Add a header
	rb.headerInputs[0].keyInput.SetValue("X-Custom")
	rb.headerInputs[0].valueInput.SetValue("custom-value")
	rb.syncEditingFromForm()

	// Build the client request
	clientReq := rb.buildClientRequest()

	if clientReq.Method != "GET" {
		t.Errorf("expected method GET, got %s", clientReq.Method)
	}

	if clientReq.URL != "https://api.example.com/build" {
		t.Errorf("expected URL https://api.example.com/build, got %s", clientReq.URL)
	}

	if clientReq.Body != `{"built": true}` {
		t.Errorf("expected body {\"built\": true}, got %s", clientReq.Body)
	}

	if len(clientReq.Headers) != 1 {
		t.Fatalf("expected 1 header, got %d", len(clientReq.Headers))
	}

	if clientReq.Headers[0].Key != "X-Custom" {
		t.Errorf("expected header key X-Custom, got %s", clientReq.Headers[0].Key)
	}

	if clientReq.Headers[0].Value != "custom-value" {
		t.Errorf("expected header value custom-value, got %s", clientReq.Headers[0].Value)
	}
}

// TestBuilder_AuthHeaders tests auth header injection
func TestBuilder_AuthHeaders(t *testing.T) {
	db := &mockBuilderDB{}
	rb := NewRequestBuilder(db)

	rb.NewRequest()
	rb.urlInput.SetValue("https://api.example.com/auth")

	// Test Bearer auth
	rb.authType = "bearer"
	tokenInput := rb.authInputs["token"]
	tokenInput.SetValue("my-token")
	rb.authInputs["token"] = tokenInput
	rb.syncEditingFromForm()

	clientReq := rb.buildClientRequest()

	// Should have Authorization header
	found := false
	for _, h := range clientReq.Headers {
		if h.Key == "Authorization" && strings.Contains(h.Value, "my-token") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Authorization header with Bearer token")
	}
}

// TestBuilder_MethodColors tests method color function
func TestBuilder_MethodColors(t *testing.T) {
	methodColorTests := []struct {
		method   string
		expected string
	}{
		{"GET", "green"},
		{"POST", "blue"},
		{"PUT", "yellow"},
		{"DELETE", "red"},
		{"PATCH", "magenta"},
		{"HEAD", "cyan"},
		{"OPTIONS", "cyan"},
	}

	for _, tt := range methodColorTests {
		color := getMethodColor(tt.method)
		expectedColor := lipgloss.Color(tt.expected)
		if color != expectedColor {
			t.Errorf("getMethodColor(%s): expected %v, got %v", tt.method, expectedColor, color)
		}
	}
}
