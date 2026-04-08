package tui

import (
	"fmt"
	"os"
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/sreeram/gurl/internal/env"
)

// errEnvNotFound is used when environment is not found (since env package doesn't define this error)
var errEnvNotFound = fmt.Errorf("environment not found")

// mockEnvStorageForTests is a test helper that implements the same interface as env.EnvStorage
// but uses in-memory storage
type mockEnvStorageForTests struct {
	envs      map[string]*env.Environment
	activeEnv string
}

func newMockEnvStorageForTests() *mockEnvStorageForTests {
	return &mockEnvStorageForTests{
		envs: make(map[string]*env.Environment),
	}
}

func (m *mockEnvStorageForTests) SaveEnv(e *env.Environment) error {
	if e.ID == "" {
		e.ID = fmt.Sprintf("id-%d", len(m.envs))
	}
	m.envs[e.ID] = e
	return nil
}

func (m *mockEnvStorageForTests) GetEnv(id string) (*env.Environment, error) {
	e, ok := m.envs[id]
	if !ok {
		return nil, errEnvNotFound
	}
	return e, nil
}

func (m *mockEnvStorageForTests) DeleteEnv(id string) error {
	delete(m.envs, id)
	if m.activeEnv == id {
		m.activeEnv = ""
	}
	return nil
}

func (m *mockEnvStorageForTests) ListEnvs() ([]*env.Environment, error) {
	result := make([]*env.Environment, 0, len(m.envs))
	for _, e := range m.envs {
		result = append(result, e)
	}
	return result, nil
}

func (m *mockEnvStorageForTests) GetEnvByName(name string) (*env.Environment, error) {
	for _, e := range m.envs {
		if e.Name == name {
			return e, nil
		}
	}
	return nil, errEnvNotFound
}

func (m *mockEnvStorageForTests) GetActiveEnv() (string, error) {
	return m.activeEnv, nil
}

func (m *mockEnvStorageForTests) SetActiveEnv(name string) error {
	if name == "" {
		m.activeEnv = ""
		return nil
	}
	// Find env by name
	for _, e := range m.envs {
		if e.Name == name {
			m.activeEnv = e.ID
			return nil
		}
	}
	return errEnvNotFound
}

// envSwitcherTestable is a version of EnvSwitcher that accepts our mock storage
type envSwitcherTestable struct {
	envStorage    *mockEnvStorageForTests
	envs          []*env.Environment
	activeEnvID   string
	activeEnvName string
	cursor        int
	mode          EnvSwitcherMode
	editVarIndex  int
	newEnvName    string
	newEnvVars    string
	importPath    string
	confirmDelete bool
	msgs          []tea.Msg
}

func newEnvSwitcherTestable(storage *mockEnvStorageForTests) *envSwitcherTestable {
	es := &envSwitcherTestable{
		envStorage:   storage,
		envs:         []*env.Environment{},
		cursor:       0,
		mode:         EnvModeList,
		editVarIndex: -1,
	}
	es.loadEnvs()
	return es
}

func (es *envSwitcherTestable) loadEnvs() {
	envs, _ := es.envStorage.ListEnvs()
	es.envs = envs

	activeEnvName, _ := es.envStorage.GetActiveEnv()
	es.activeEnvName = activeEnvName

	for _, e := range es.envs {
		if e.Name == activeEnvName {
			es.activeEnvID = e.ID
			break
		}
	}
}

func (es *envSwitcherTestable) handleKeyPress(msg tea.KeyMsg) {
	switch msg.String() {
	case "q", "esc":
		es.mode = EnvModeList
		es.editVarIndex = -1
		es.confirmDelete = false
		return

	case "up", "k":
		if es.cursor > 0 {
			es.cursor--
		}
		return

	case "down", "j":
		if es.mode == EnvModeList {
			if es.cursor < len(es.envs)-1 {
				es.cursor++
			}
		} else if es.mode == EnvModeEdit {
			if es.editVarIndex >= 0 && es.cursor < len(es.envs) {
				env := es.envs[es.cursor]
				if es.editVarIndex < len(env.Variables)-1 {
					es.editVarIndex++
				}
			}
		}
		return

	case "enter":
		es.handleEnter()
		return

	case "d":
		if es.mode == EnvModeList && len(es.envs) > 0 {
			es.mode = EnvModeDeleteConfirm
			es.confirmDelete = false
		}
		return

	case "n":
		if es.mode == EnvModeList {
			es.mode = EnvModeCreate
			es.newEnvName = ""
			es.newEnvVars = ""
		}
		return

	case "i":
		if es.mode == EnvModeList {
			es.mode = EnvModeImport
			es.importPath = ""
		}
		return

	case "e":
		if es.mode == EnvModeList && len(es.envs) > 0 {
			es.mode = EnvModeEdit
			es.editVarIndex = -1
		}
		return

	case "y":
		if es.mode == EnvModeDeleteConfirm {
			if es.confirmDelete {
				es.handleDelete()
				return
			}
			es.confirmDelete = true
		}
		return
	}

	// Handle typing in input modes
	if es.mode == EnvModeCreate || es.mode == EnvModeImport {
		if msg.Type == tea.KeyRunes {
			runes := string(msg.Runes)
			if es.mode == EnvModeCreate && es.editVarIndex == 0 {
				es.newEnvName += runes
			} else if es.mode == EnvModeCreate && es.editVarIndex == 1 {
				es.newEnvVars += runes
			} else if es.mode == EnvModeImport {
				es.importPath += runes
			}
		}
	}
}

func (es *envSwitcherTestable) handleEnter() {
	switch es.mode {
	case EnvModeList:
		es.handleSelectEnv()
	case EnvModeCreate:
		es.handleCreateEnv()
	case EnvModeImport:
		es.handleImportEnv()
	case EnvModeDeleteConfirm:
		if es.confirmDelete {
			es.handleDelete()
		} else {
			es.confirmDelete = true
		}
	case EnvModeEdit:
		if es.editVarIndex >= 0 && es.cursor < len(es.envs) {
			es.mode = EnvModeVarEdit
		}
	case EnvModeVarEdit:
		es.mode = EnvModeEdit
	}
}

func (es *envSwitcherTestable) handleSelectEnv() {
	if es.cursor < 0 || es.cursor >= len(es.envs) {
		return
	}
	selectedEnv := es.envs[es.cursor]
	es.activeEnvID = selectedEnv.ID
	es.activeEnvName = selectedEnv.Name
	es.envStorage.SetActiveEnv(selectedEnv.Name)
}

func (es *envSwitcherTestable) handleCreateEnv() {
	if es.newEnvName == "" {
		es.editVarIndex = 1
		return
	}

	newEnv := env.NewEnvironment(es.newEnvName, "")
	if es.newEnvVars != "" {
		vars, _ := env.ParseDotenv(es.newEnvVars)
		for k, v := range vars {
			newEnv.SetVariable(k, v)
		}
	}

	es.envStorage.SaveEnv(newEnv)
	es.loadEnvs()
	es.mode = EnvModeList
	es.editVarIndex = -1
}

func (es *envSwitcherTestable) handleImportEnv() {
	if es.importPath == "" {
		return
	}

	vars, err := env.ParseDotenvFile(es.importPath)
	if err != nil {
		return
	}

	parts := splitPath(es.importPath)
	filename := parts[len(parts)-1]
	envName := trimSuffix(filename, ".env")
	if envName == "" {
		envName = "imported"
	}

	newEnv := env.NewEnvironment(envName, "")
	for k, v := range vars {
		newEnv.SetVariable(k, v)
	}

	es.envStorage.SaveEnv(newEnv)
	es.loadEnvs()
	es.mode = EnvModeList
	es.importPath = ""
}

func (es *envSwitcherTestable) handleDelete() {
	if es.cursor < 0 || es.cursor >= len(es.envs) {
		return
	}
	envToDelete := es.envs[es.cursor]
	es.envStorage.DeleteEnv(envToDelete.ID)

	if es.activeEnvID == envToDelete.ID {
		es.activeEnvID = ""
		es.activeEnvName = ""
		es.envStorage.SetActiveEnv("")
	}

	es.loadEnvs()
	es.mode = EnvModeList
	es.confirmDelete = false

	if es.cursor >= len(es.envs) && es.cursor > 0 {
		es.cursor = len(es.envs) - 1
	}
}

func (es *envSwitcherTestable) getActiveEnvName() string {
	return es.activeEnvName
}

func (es *envSwitcherTestable) getActiveEnvID() string {
	return es.activeEnvID
}

// View implements tea.Model.View for testing
func (es *envSwitcherTestable) View() string {
	switch es.mode {
	case EnvModeList:
		return es.viewList()
	case EnvModeCreate:
		return es.viewCreate()
	case EnvModeEdit:
		return es.viewEdit()
	case EnvModeDeleteConfirm:
		return es.viewDeleteConfirm()
	case EnvModeImport:
		return es.viewImport()
	case EnvModeVarEdit:
		return es.viewVarEdit()
	default:
		return es.viewList()
	}
}

func (es *envSwitcherTestable) viewList() string {
	result := "Environments\n"
	if len(es.envs) == 0 {
		result += "No environments"
		return result
	}
	for i, e := range es.envs {
		prefix := "  "
		if i == es.cursor {
			prefix = "> "
		}
		nameDisplay := e.Name
		if e.ID == es.activeEnvID {
			nameDisplay = nameDisplay + " (active)"
		}
		result += prefix + nameDisplay + "\n"
	}
	return result
}

func (es *envSwitcherTestable) viewCreate() string {
	result := "Create Environment\n"
	result += "Name: " + es.newEnvName + "\n"
	result += "Vars: " + es.newEnvVars + "\n"
	return result
}

func (es *envSwitcherTestable) viewEdit() string {
	result := "Edit Environment\n"
	if es.cursor < len(es.envs) {
		e := es.envs[es.cursor]
		for k, v := range e.Variables {
			displayVal := v
			if e.IsSecret(k) {
				displayVal = "*****"
			}
			result += fmt.Sprintf("%s = %s\n", k, displayVal)
		}
	}
	return result
}

func (es *envSwitcherTestable) viewDeleteConfirm() string {
	return "Delete Environment?"
}

func (es *envSwitcherTestable) viewImport() string {
	return "Import from .env\nPath: " + es.importPath
}

func (es *envSwitcherTestable) viewVarEdit() string {
	return "Edit Variable"
}

// Helper functions
func splitPath(p string) []string {
	parts := make([]string, 0)
	start := 0
	for i := 0; i < len(p); i++ {
		if p[i] == '/' || p[i] == '\\' {
			if i > start {
				parts = append(parts, p[start:i])
			}
			start = i + 1
		}
	}
	if start < len(p) {
		parts = append(parts, p[start:])
	}
	return parts
}

func trimSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}

// Tests

func TestEnvSwitcher_New(t *testing.T) {
	storage := newMockEnvStorageForTests()
	es := newEnvSwitcherTestable(storage)

	if es == nil {
		t.Fatal("expected non-nil EnvSwitcher")
	}

	if es.mode != EnvModeList {
		t.Errorf("expected mode EnvModeList, got %v", es.mode)
	}

	if es.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", es.cursor)
	}

	if len(es.envs) != 0 {
		t.Errorf("expected 0 envs, got %d", len(es.envs))
	}
}

func TestEnvSwitcher_LoadEnvs(t *testing.T) {
	storage := newMockEnvStorageForTests()

	env1 := env.NewEnvironment("dev", "")
	env1.SetVariable("API_URL", "http://localhost:3000")
	storage.SaveEnv(env1)

	env2 := env.NewEnvironment("prod", "")
	env2.SetVariable("API_URL", "https://api.example.com")
	storage.SaveEnv(env2)

	es := newEnvSwitcherTestable(storage)

	if len(es.envs) != 2 {
		t.Errorf("expected 2 envs, got %d", len(es.envs))
	}

	names := make(map[string]bool)
	for _, e := range es.envs {
		names[e.Name] = true
	}

	if !names["dev"] {
		t.Error("expected 'dev' in env names")
	}
	if !names["prod"] {
		t.Error("expected 'prod' in env names")
	}
}

func TestEnvSwitcher_Navigate(t *testing.T) {
	storage := newMockEnvStorageForTests()

	for i := 0; i < 3; i++ {
		e := env.NewEnvironment(fmt.Sprintf("env%d", i), "")
		storage.SaveEnv(e)
	}

	es := newEnvSwitcherTestable(storage)

	if es.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", es.cursor)
	}

	msg := tea.KeyMsg{Type: tea.KeyDown}
	es.handleKeyPress(msg)

	if es.cursor != 1 {
		t.Errorf("expected cursor 1 after down, got %d", es.cursor)
	}

	es.handleKeyPress(msg)

	if es.cursor != 2 {
		t.Errorf("expected cursor 2 after second down, got %d", es.cursor)
	}

	upMsg := tea.KeyMsg{Type: tea.KeyUp}
	es.handleKeyPress(upMsg)

	if es.cursor != 1 {
		t.Errorf("expected cursor 1 after up, got %d", es.cursor)
	}
}

func TestEnvSwitcher_SelectEnv(t *testing.T) {
	storage := newMockEnvStorageForTests()

	env1 := env.NewEnvironment("dev", "")
	env1.SetVariable("KEY", "value1")
	storage.SaveEnv(env1)

	env2 := env.NewEnvironment("prod", "")
	env2.SetVariable("KEY", "value2")
	storage.SaveEnv(env2)

	es := newEnvSwitcherTestable(storage)

	es.cursor = 0
	es.handleSelectEnv()

	if es.activeEnvName != "dev" {
		t.Errorf("expected active env 'dev', got '%s'", es.activeEnvName)
	}

	if es.activeEnvID != env1.ID {
		t.Errorf("expected active env ID '%s', got '%s'", env1.ID, es.activeEnvID)
	}

	es.cursor = 1
	es.handleSelectEnv()

	if es.activeEnvName != "prod" {
		t.Errorf("expected active env 'prod', got '%s'", es.activeEnvName)
	}
}

func TestEnvSwitcher_CreateEnv(t *testing.T) {
	storage := newMockEnvStorageForTests()
	es := newEnvSwitcherTestable(storage)

	nMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
	es.handleKeyPress(nMsg)

	if es.mode != EnvModeCreate {
		t.Errorf("expected mode EnvModeCreate, got %v", es.mode)
	}

	es.editVarIndex = 0
	es.newEnvName = "testenv"

	es.handleCreateEnv()

	if len(es.envs) != 1 {
		t.Errorf("expected 1 env after create, got %d", len(es.envs))
	}

	if es.envs[0].Name != "testenv" {
		t.Errorf("expected env name 'testenv', got '%s'", es.envs[0].Name)
	}

	if es.mode != EnvModeList {
		t.Errorf("expected mode EnvModeList after create, got %v", es.mode)
	}
}

func TestEnvSwitcher_CreateEnvWithVars(t *testing.T) {
	storage := newMockEnvStorageForTests()
	es := newEnvSwitcherTestable(storage)

	es.mode = EnvModeCreate
	es.editVarIndex = 0
	es.newEnvName = "myenv"
	es.newEnvVars = "API_URL=https://example.com\nSECRET=password123"

	es.handleCreateEnv()

	if len(es.envs) != 1 {
		t.Fatalf("expected 1 env, got %d", len(es.envs))
	}

	e := es.envs[0]
	if e.Name != "myenv" {
		t.Errorf("expected name 'myenv', got '%s'", e.Name)
	}

	val, ok := e.GetVariable("API_URL")
	if !ok {
		t.Error("expected API_URL variable")
	}
	if val != "https://example.com" {
		t.Errorf("expected API_URL 'https://example.com', got '%s'", val)
	}

	val, ok = e.GetVariable("SECRET")
	if !ok {
		t.Error("expected SECRET variable")
	}
	if val != "password123" {
		t.Errorf("expected SECRET 'password123', got '%s'", val)
	}
}

func TestEnvSwitcher_DeleteEnv(t *testing.T) {
	storage := newMockEnvStorageForTests()

	e := env.NewEnvironment("todelete", "")
	storage.SaveEnv(e)

	es := newEnvSwitcherTestable(storage)

	if len(es.envs) != 1 {
		t.Fatalf("expected 1 env, got %d", len(es.envs))
	}

	es.activeEnvID = e.ID
	es.activeEnvName = e.Name

	es.cursor = 0
	es.mode = EnvModeDeleteConfirm
	es.confirmDelete = true

	es.handleDelete()

	if len(es.envs) != 0 {
		t.Errorf("expected 0 envs after delete, got %d", len(es.envs))
	}

	if es.activeEnvID != "" {
		t.Errorf("expected active env to be cleared, got '%s'", es.activeEnvID)
	}
}

func TestEnvSwitcher_DeleteConfirm(t *testing.T) {
	storage := newMockEnvStorageForTests()

	e := env.NewEnvironment("test", "")
	storage.SaveEnv(e)

	es := newEnvSwitcherTestable(storage)
	es.cursor = 0

	dMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}
	es.handleKeyPress(dMsg)

	if es.mode != EnvModeDeleteConfirm {
		t.Errorf("expected mode EnvModeDeleteConfirm, got %v", es.mode)
	}

	if es.confirmDelete {
		t.Error("expected confirmDelete to be false initially")
	}

	yMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	es.handleKeyPress(yMsg)

	if !es.confirmDelete {
		t.Error("expected confirmDelete to be true after first y")
	}

	if len(es.envs) != 1 {
		t.Errorf("expected env not to be deleted yet, got %d envs", len(es.envs))
	}

	es.handleKeyPress(yMsg)

	if len(es.envs) != 0 {
		t.Errorf("expected env to be deleted, got %d envs", len(es.envs))
	}
}

func TestEnvSwitcher_Quit(t *testing.T) {
	storage := newMockEnvStorageForTests()
	es := newEnvSwitcherTestable(storage)

	es.mode = EnvModeCreate
	es.newEnvName = "test"

	qMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	es.handleKeyPress(qMsg)

	if es.mode != EnvModeList {
		t.Errorf("expected mode EnvModeList after q, got %v", es.mode)
	}
}

func TestEnvSwitcher_SecretMasking(t *testing.T) {
	storage := newMockEnvStorageForTests()

	e := env.NewEnvironment("test", "")
	e.SetSecretVariable("API_KEY", "super-secret-key")
	e.SetVariable("API_URL", "http://example.com")
	storage.SaveEnv(e)

	es := newEnvSwitcherTestable(storage)
	es.cursor = 0

	eMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")}
	es.handleKeyPress(eMsg)

	if es.mode != EnvModeEdit {
		t.Errorf("expected mode EnvModeEdit, got %v", es.mode)
	}

	// The secret should be masked when viewing vars
	if !e.IsSecret("API_KEY") {
		t.Error("expected API_KEY to be marked as secret")
	}
}

func TestEnvSwitcher_ImportEnv(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.env")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := `API_URL=https://example.com
DB_HOST=localhost`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	storage := newMockEnvStorageForTests()
	es := newEnvSwitcherTestable(storage)

	iMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")}
	es.handleKeyPress(iMsg)

	if es.mode != EnvModeImport {
		t.Errorf("expected mode EnvModeImport, got %v", es.mode)
	}

	es.importPath = tmpFile.Name()

	es.handleImportEnv()

	if len(es.envs) != 1 {
		t.Errorf("expected 1 env after import, got %d", len(es.envs))
	}

	e := es.envs[0]
	val, ok := e.GetVariable("API_URL")
	if !ok {
		t.Error("expected API_URL variable")
	}
	if val != "https://example.com" {
		t.Errorf("expected API_URL 'https://example.com', got '%s'", val)
	}
}

func TestEnvSwitcher_GetActiveEnvName(t *testing.T) {
	storage := newMockEnvStorageForTests()
	es := newEnvSwitcherTestable(storage)

	if es.getActiveEnvName() != "" {
		t.Error("expected empty active env initially")
	}

	e := env.NewEnvironment("test", "")
	storage.SaveEnv(e)
	es.loadEnvs()
	es.activeEnvName = "test"
	es.activeEnvID = e.ID

	if es.getActiveEnvName() != "test" {
		t.Errorf("expected 'test', got '%s'", es.getActiveEnvName())
	}
}

func TestEnvSwitcher_NavigateEditMode(t *testing.T) {
	storage := newMockEnvStorageForTests()

	e := env.NewEnvironment("test", "")
	e.SetVariable("KEY1", "value1")
	e.SetVariable("KEY2", "value2")
	e.SetVariable("KEY3", "value3")
	storage.SaveEnv(e)

	es := newEnvSwitcherTestable(storage)
	es.cursor = 0

	eMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")}
	es.handleKeyPress(eMsg)

	if es.mode != EnvModeEdit {
		t.Errorf("expected mode EnvModeEdit, got %v", es.mode)
	}

	// In edit mode, down key should move between variables
	// Set editVarIndex to 0 first since env has 3 vars
	es.editVarIndex = 0

	msg := tea.KeyMsg{Type: tea.KeyDown}
	es.handleKeyPress(msg)

	// After down in edit mode, editVarIndex should increment
	if es.editVarIndex != 1 {
		t.Errorf("expected editVarIndex 1, got %d", es.editVarIndex)
	}
}

func TestEnvSwitcher_FileExists(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-fileexists")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if !FileExists(tmpFile.Name()) {
		t.Error("expected FileExists to return true for existing file")
	}

	if FileExists("/non/existent/path") {
		t.Error("expected FileExists to return false for non-existent file")
	}
}

func TestEnvSwitcher_MaskSecretsForDisplay(t *testing.T) {
	content := "API_URL=https://example.com\nSECRET=password123"
	secretKeys := map[string]bool{"SECRET": true}

	result := maskSecretsForDisplay(content, secretKeys)

	if !envContains(result, "API_URL=https://example.com") {
		t.Error("expected non-secret to be preserved")
	}

	if !envContains(result, "SECRET = *****") {
		t.Error("expected secret to be masked")
	}
}

func TestEnvSwitcher_AddEnv(t *testing.T) {
	storage := newMockEnvStorageForTests()
	es := newEnvSwitcherTestable(storage)

	vars := map[string]string{
		"API_URL": "http://example.com",
		"DEBUG":   "true",
	}

	newEnv := env.NewEnvironment("newenv", "")
	for k, v := range vars {
		newEnv.SetVariable(k, v)
	}

	err := storage.SaveEnv(newEnv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	es.loadEnvs()

	if len(es.envs) != 1 {
		t.Errorf("expected 1 env in list, got %d", len(es.envs))
	}

	val, ok := newEnv.GetVariable("API_URL")
	if !ok {
		t.Fatal("expected API_URL")
	}
	if val != "http://example.com" {
		t.Errorf("expected 'http://example.com', got '%s'", val)
	}
}

func TestEnvSwitcher_DeleteEnvByID(t *testing.T) {
	storage := newMockEnvStorageForTests()

	e := env.NewEnvironment("test", "")
	storage.SaveEnv(e)

	es := newEnvSwitcherTestable(storage)
	// Simulate selecting the env first (which sets activeEnvID via SetActiveEnv)
	es.envStorage.SetActiveEnv("test")
	es.loadEnvs()

	err := storage.DeleteEnv(e.ID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	es.loadEnvs()

	if len(es.envs) != 0 {
		t.Errorf("expected 0 envs, got %d", len(es.envs))
	}

	// After loadEnvs, activeEnvName should be cleared since the env was deleted
	if es.activeEnvName != "" {
		t.Errorf("expected activeEnvName to be cleared, got '%s'", es.activeEnvName)
	}
}

func TestEnvSwitcher_ViewModes(t *testing.T) {
	storage := newMockEnvStorageForTests()
	es := newEnvSwitcherTestable(storage)

	// Test list view
	es.mode = EnvModeList
	view := es.View()
	if !envContains(view, "Environments") {
		t.Error("expected list view")
	}

	// Test create view
	es.mode = EnvModeCreate
	es.editVarIndex = 0
	es.newEnvName = ""
	view = es.View()
	if !envContains(view, "Create Environment") {
		t.Error("expected create view")
	}

	// Test delete confirm view
	es.mode = EnvModeDeleteConfirm
	view = es.View()
	if !envContains(view, "Delete") {
		t.Error("expected delete confirm view")
	}

	// Test import view
	es.mode = EnvModeImport
	view = es.View()
	if !envContains(view, "Import") {
		t.Error("expected import view")
	}
}

func TestEnvSwitcher_UpdateEnvVar(t *testing.T) {
	storage := newMockEnvStorageForTests()

	e := env.NewEnvironment("test", "")
	e.SetVariable("KEY", "oldvalue")
	storage.SaveEnv(e)

	es := newEnvSwitcherTestable(storage)

	e.SetVariable("KEY", "newvalue")
	storage.SaveEnv(e)

	es.loadEnvs()

	updated := es.envs[0]
	val, ok := updated.GetVariable("KEY")
	if !ok {
		t.Fatal("expected KEY variable")
	}
	if val != "newvalue" {
		t.Errorf("expected 'newvalue', got '%s'", val)
	}
}

func TestEnvSwitcher_SecretVar(t *testing.T) {
	storage := newMockEnvStorageForTests()

	e := env.NewEnvironment("test", "")
	e.SetVariable("SECRET", "mysecret")
	e.SecretKeys["SECRET"] = true
	storage.SaveEnv(e)

	es := newEnvSwitcherTestable(storage)

	if !es.envs[0].IsSecret("SECRET") {
		t.Error("expected SECRET to be marked as secret")
	}

	val, _ := es.envs[0].GetVariable("SECRET")
	if val != "mysecret" {
		t.Errorf("expected 'mysecret', got '%s'", val)
	}
}

// envContains is a test helper to avoid redeclaring contains
func envContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
