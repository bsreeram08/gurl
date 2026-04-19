package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sreeram/gurl/internal/env"
)

// EnvSwitcherMode represents the current mode of the env switcher
type EnvSwitcherMode int

const (
	EnvModeList EnvSwitcherMode = iota
	EnvModeCreate
	EnvModeEdit
	EnvModeDeleteConfirm
	EnvModeImport
	EnvModeVarEdit
)

// EnvSwitcher is a bubbletea sub-model for environment management
type EnvSwitcher struct {
	envStorage    *env.EnvStorage
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
	width         int
	height        int
	msgs          []tea.Msg
	nameInput     textinput.Model
	varsInput     textinput.Model
	importInput   textinput.Model
}

// EnvChangedMsg is sent when the active environment changes
type EnvChangedMsg struct {
	EnvID   string
	EnvName string
}

// EnvCreatedMsg is sent when a new environment is created
type EnvCreatedMsg struct {
	Env *env.Environment
}

// EnvDeletedMsg is sent when an environment is deleted
type EnvDeletedMsg struct {
	EnvID string
}

// NewEnvSwitcher creates a new environment switcher component
func NewEnvSwitcher(envStorage *env.EnvStorage) *EnvSwitcher {
	nameInput := textinput.New()
	nameInput.Placeholder = "Environment name"
	nameInput.Prompt = "> "

	varsInput := textinput.New()
	varsInput.Placeholder = "KEY=value, one per line"
	varsInput.Prompt = "> "

	importInput := textinput.New()
	importInput.Placeholder = "Path to .env file"
	importInput.Prompt = "> "

	es := &EnvSwitcher{
		envStorage:   envStorage,
		envs:         []*env.Environment{},
		cursor:       0,
		mode:         EnvModeList,
		editVarIndex: -1,
		nameInput:    nameInput,
		varsInput:    varsInput,
		importInput:  importInput,
	}

	// Load environments
	es.loadEnvs()

	return es
}

// loadEnvs loads environments from storage
func (es *EnvSwitcher) loadEnvs() {
	envs, err := es.envStorage.ListEnvs()
	if err != nil {
		envs = []*env.Environment{}
	}
	es.envs = envs

	// Get active environment
	activeEnvName, err := es.envStorage.GetActiveEnv()
	if err != nil {
		activeEnvName = ""
	}
	es.activeEnvName = activeEnvName

	// Find active env by name
	for _, e := range es.envs {
		if e.Name == activeEnvName {
			es.activeEnvID = e.ID
			break
		}
	}
}

// Init implements tea.Model.Init
func (es *EnvSwitcher) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.Update
func (es *EnvSwitcher) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		es.width = msg.Width
		es.height = msg.Height
		return es, nil

	case tea.KeyPressMsg:
		return es.handleKeyPress(msg)
	}

	return es, nil
}

// handleKeyPress handles keyboard input using switch on key.String()
func (es *EnvSwitcher) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		es.mode = EnvModeList
		es.editVarIndex = -1
		es.confirmDelete = false
		return es, nil

	case "up", "k":
		return es.handleNavigateUp()

	case "down", "j":
		return es.handleNavigateDown()

	case "enter":
		return es.handleEnter()

	case "d":
		if es.mode == EnvModeList && len(es.envs) > 0 {
			es.mode = EnvModeDeleteConfirm
			es.confirmDelete = false
		}
		return es, nil

	case "n":
		if es.mode == EnvModeList {
			es.mode = EnvModeCreate
			es.newEnvName = ""
			es.newEnvVars = ""
		}
		return es, nil

	case "i":
		if es.mode == EnvModeList {
			es.mode = EnvModeImport
			es.importPath = ""
		}
		return es, nil

	case "e":
		if es.mode == EnvModeList && len(es.envs) > 0 {
			es.mode = EnvModeEdit
			es.editVarIndex = -1
		}
		return es, nil

	case "y":
		if es.mode == EnvModeDeleteConfirm {
			if es.confirmDelete {
				return es.handleDelete()
			}
			es.confirmDelete = true
		}
		return es, nil

	case "backspace":
		return es.handleBackspace()
	}

	// Handle typing in input modes using textinput
	if es.mode == EnvModeCreate || es.mode == EnvModeImport {
		var input textinput.Model
		switch es.mode {
		case EnvModeCreate:
			if es.editVarIndex == 0 {
				input = es.nameInput
			} else {
				input = es.varsInput
			}
		case EnvModeImport:
			input = es.importInput
		}
		input, _ = input.Update(msg)
		switch es.mode {
		case EnvModeCreate:
			if es.editVarIndex == 0 {
				es.nameInput = input
			} else {
				es.varsInput = input
			}
		case EnvModeImport:
			es.importInput = input
		}
	}

	return es, nil
}

// handleNavigateUp handles upward navigation
func (es *EnvSwitcher) handleNavigateUp() (tea.Model, tea.Cmd) {
	switch es.mode {
	case EnvModeList, EnvModeEdit:
		if es.cursor > 0 {
			es.cursor--
		}
	case EnvModeVarEdit:
		if es.editVarIndex > 0 {
			es.editVarIndex--
		}
	case EnvModeCreate:
		if es.editVarIndex > 0 {
			es.editVarIndex--
		}
	}
	return es, nil
}

// handleNavigateDown handles downward navigation
func (es *EnvSwitcher) handleNavigateDown() (tea.Model, tea.Cmd) {
	switch es.mode {
	case EnvModeList:
		if es.cursor < len(es.envs)-1 {
			es.cursor++
		}
	case EnvModeEdit:
		if es.editVarIndex < len(es.envs[es.cursor].Variables)-1 {
			es.editVarIndex++
		}
	case EnvModeVarEdit:
		// Navigate within var editing - name or value field
		// This is handled specially
	}
	return es, nil
}

// handleEnter handles Enter key based on current mode
func (es *EnvSwitcher) handleEnter() (tea.Model, tea.Cmd) {
	switch es.mode {
	case EnvModeList:
		return es.handleSelectEnv()

	case EnvModeCreate:
		return es.handleCreateEnv()

	case EnvModeImport:
		return es.handleImportEnv()

	case EnvModeDeleteConfirm:
		if es.confirmDelete {
			return es.handleDelete()
		}
		es.confirmDelete = true

	case EnvModeEdit:
		// Toggle into var edit mode for selected var
		if es.editVarIndex >= 0 && es.cursor < len(es.envs) {
			es.mode = EnvModeVarEdit
		}

	case EnvModeVarEdit:
		// Exit var edit mode
		es.mode = EnvModeEdit
	}

	return es, nil
}

// handleSelectEnv selects an environment and sets it as active
func (es *EnvSwitcher) handleSelectEnv() (tea.Model, tea.Cmd) {
	if es.cursor < 0 || es.cursor >= len(es.envs) {
		return es, nil
	}

	selectedEnv := es.envs[es.cursor]
	es.activeEnvID = selectedEnv.ID
	es.activeEnvName = selectedEnv.Name

	if err := es.envStorage.SetActiveEnv(selectedEnv.Name); err != nil {
		// Log error but continue
	}

	es.msgs = append(es.msgs, EnvChangedMsg{
		EnvID:   selectedEnv.ID,
		EnvName: selectedEnv.Name,
	})

	return es, nil
}

// handleCreateEnv creates a new environment from input
func (es *EnvSwitcher) handleCreateEnv() (tea.Model, tea.Cmd) {
	name := es.nameInput.Value()
	vars := es.varsInput.Value()

	if name == "" {
		// Just move to vars input
		es.editVarIndex = 1
		return es, nil
	}

	// If we have vars, parse them
	newEnv := env.NewEnvironment(name, "")
	if vars != "" {
		envVars, err := env.ParseDotenv(vars)
		if err == nil {
			for k, v := range envVars {
				newEnv.SetVariable(k, v)
			}
		}
	}

	if err := es.envStorage.SaveEnv(newEnv); err != nil {
		return es, nil
	}

	es.msgs = append(es.msgs, EnvCreatedMsg{Env: newEnv})
	es.loadEnvs()
	es.mode = EnvModeList
	es.editVarIndex = -1
	es.nameInput.SetValue("")
	es.varsInput.SetValue("")

	return es, nil
}

// handleImportEnv imports environment from .env file
func (es *EnvSwitcher) handleImportEnv() (tea.Model, tea.Cmd) {
	path := es.importInput.Value()
	if path == "" {
		return es, nil
	}

	vars, err := env.ParseDotenvFile(path)
	if err != nil {
		return es, nil
	}

	// Get filename as env name
	parts := strings.Split(path, "/")
	filename := parts[len(parts)-1]
	envName := strings.TrimSuffix(filename, ".env")
	if envName == "" {
		envName = "imported"
	}

	newEnv := env.NewEnvironment(envName, "")
	for k, v := range vars {
		newEnv.SetVariable(k, v)
	}

	if err := es.envStorage.SaveEnv(newEnv); err != nil {
		return es, nil
	}

	es.msgs = append(es.msgs, EnvCreatedMsg{Env: newEnv})
	es.loadEnvs()
	es.mode = EnvModeList
	es.importInput.SetValue("")

	return es, nil
}

// handleDelete deletes the selected environment
func (es *EnvSwitcher) handleDelete() (tea.Model, tea.Cmd) {
	if es.cursor < 0 || es.cursor >= len(es.envs) {
		return es, nil
	}

	envToDelete := es.envs[es.cursor]
	envID := envToDelete.ID

	if err := es.envStorage.DeleteEnv(envID); err != nil {
		return es, nil
	}

	// If this was the active env, clear active
	if es.activeEnvID == envID {
		es.activeEnvID = ""
		es.activeEnvName = ""
		es.envStorage.SetActiveEnv("")
	}

	es.msgs = append(es.msgs, EnvDeletedMsg{EnvID: envID})
	es.loadEnvs()
	es.mode = EnvModeList
	es.confirmDelete = false

	if es.cursor >= len(es.envs) && es.cursor > 0 {
		es.cursor = len(es.envs) - 1
	}

	return es, nil
}

// handleBackspace handles backspace in input modes
func (es *EnvSwitcher) handleBackspace() (tea.Model, tea.Cmd) {
	switch es.mode {
	case EnvModeCreate:
		if es.editVarIndex == 1 && len(es.newEnvVars) > 0 {
			es.newEnvVars = es.newEnvVars[:len(es.newEnvVars)-1]
		} else if es.editVarIndex == 0 && len(es.newEnvName) > 0 {
			es.newEnvName = es.newEnvName[:len(es.newEnvName)-1]
		}
	case EnvModeImport:
		if len(es.importPath) > 0 {
			es.importPath = es.importPath[:len(es.importPath)-1]
		}
	}
	return es, nil
}

// View renders the environment switcher
func (es *EnvSwitcher) View() tea.View {
	switch es.mode {
	case EnvModeList:
		return tea.NewView(es.viewList())
	case EnvModeCreate:
		return tea.NewView(es.viewCreate())
	case EnvModeEdit:
		return tea.NewView(es.viewEdit())
	case EnvModeDeleteConfirm:
		return tea.NewView(es.viewDeleteConfirm())
	case EnvModeImport:
		return tea.NewView(es.viewImport())
	case EnvModeVarEdit:
		return tea.NewView(es.viewVarEdit())
	default:
		return tea.NewView(es.viewList())
	}
}

// viewList shows the environment list
func (es *EnvSwitcher) viewList() string {
	var sb strings.Builder

	sb.WriteString(Style.Header.Render("Environments"))
	sb.WriteString("\n")

	if len(es.envs) == 0 {
		sb.WriteString("\n")
		sb.WriteString(Style.PlainText.Render("  No environments"))
		sb.WriteString("\n")
		sb.WriteString("\n")
		sb.WriteString(Style.Hint.Render("  Press n to create new"))
		return sb.String()
	}

	for i, e := range es.envs {
		sb.WriteString("\n")
		prefix := "  "
		if i == es.cursor {
			prefix = "▶ "
		}

		nameDisplay := e.Name
		if e.ID == es.activeEnvID {
			nameDisplay = nameDisplay + " (active)"
		}

		if i == es.cursor {
			sb.WriteString(Style.SelectedItem.Render(prefix + nameDisplay))
		} else {
			sb.WriteString(Style.ListItem.Render(prefix + nameDisplay))
		}

		// Show var count
		varCount := len(e.Variables)
		if varCount > 0 {
			sb.WriteString(Style.Hint.Render(fmt.Sprintf(" (%d vars)", varCount)))
		}
	}

	sb.WriteString("\n")
	sb.WriteString("\n")
	sb.WriteString(Style.Hint.Render("  ↑/↓ navigate  Enter: select  n: new  i: import"))
	sb.WriteString("\n")
	sb.WriteString(Style.Hint.Render("  e: edit vars  d: delete  q: back"))

	return sb.String()
}

// viewCreate shows the create environment form
func (es *EnvSwitcher) viewCreate() string {
	var sb strings.Builder

	sb.WriteString(Style.Header.Render("Create Environment"))
	sb.WriteString("\n")
	sb.WriteString("\n")

	// Environment name input
	sb.WriteString("  Name: ")
	if es.editVarIndex == 0 {
		sb.WriteString(Style.SelectedItem.Render("> " + es.newEnvName + "_"))
	} else {
		if es.newEnvName == "" {
			sb.WriteString(Style.Hint.Render("(enter name)"))
		} else {
			sb.WriteString(es.newEnvName)
		}
	}
	sb.WriteString("\n")

	// Initial vars input
	sb.WriteString("  Vars: ")
	if es.editVarIndex == 1 {
		sb.WriteString(Style.SelectedItem.Render("> " + es.newEnvVars + "_"))
	} else {
		if es.newEnvVars == "" {
			sb.WriteString(Style.Hint.Render("(KEY=value, one per line)"))
		} else {
			// Show masked view of vars
			displayVars := maskSecretsForDisplay(es.newEnvVars, make(map[string]bool))
			sb.WriteString(displayVars)
		}
	}
	sb.WriteString("\n")

	sb.WriteString("\n")
	sb.WriteString(Style.Hint.Render("  Enter: create  q: cancel"))
	sb.WriteString("\n")
	sb.WriteString(Style.Hint.Render("  Tab: switch field"))

	return sb.String()
}

// viewEdit shows the edit environment view
func (es *EnvSwitcher) viewEdit() string {
	if es.cursor < 0 || es.cursor >= len(es.envs) {
		return es.viewList()
	}

	envObj := es.envs[es.cursor]
	var sb strings.Builder

	sb.WriteString(Style.Header.Render(fmt.Sprintf("Edit: %s", envObj.Name)))
	sb.WriteString("\n")

	if len(envObj.Variables) == 0 {
		sb.WriteString("\n")
		sb.WriteString(Style.PlainText.Render("  No variables"))
		sb.WriteString("\n")
	} else {
		i := 0
		for k, v := range envObj.Variables {
			sb.WriteString("\n")
			prefix := "  "
			if i == es.editVarIndex {
				prefix = "▶ "
			}

			displayVal := v
			if envObj.IsSecret(k) {
				displayVal = "*****"
			}

			if i == es.editVarIndex {
				sb.WriteString(Style.SelectedItem.Render(fmt.Sprintf("%s%s = %s", prefix, k, displayVal)))
			} else {
				sb.WriteString(Style.ListItem.Render(fmt.Sprintf("%s%s = %s", prefix, k, displayVal)))
			}
			i++
		}
	}

	sb.WriteString("\n")
	sb.WriteString("\n")
	sb.WriteString(Style.Hint.Render("  ↑/↓ navigate vars  Enter: edit  q: back"))

	return sb.String()
}

// viewVarEdit shows the variable edit form
func (es *EnvSwitcher) viewVarEdit() string {
	if es.cursor < 0 || es.cursor >= len(es.envs) {
		return es.viewList()
	}

	envObj := es.envs[es.cursor]
	var sb strings.Builder

	// Get current var key and value
	keys := make([]string, 0, len(envObj.Variables))
	for k := range envObj.Variables {
		keys = append(keys, k)
	}

	if es.editVarIndex < 0 || es.editVarIndex >= len(keys) {
		return es.viewEdit()
	}

	key := keys[es.editVarIndex]
	value := envObj.Variables[key]
	isSecret := envObj.IsSecret(key)

	sb.WriteString(Style.Header.Render(fmt.Sprintf("Edit Var: %s", key)))
	sb.WriteString("\n")
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("  Key: %s\n", key))
	sb.WriteString(fmt.Sprintf("  Value: "))

	if isSecret {
		sb.WriteString("*****\n")
	} else {
		sb.WriteString(value + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(Style.Hint.Render("  q: back (read-only in this version)"))

	return sb.String()
}

// viewDeleteConfirm shows the delete confirmation
func (es *EnvSwitcher) viewDeleteConfirm() string {
	if es.cursor < 0 || es.cursor >= len(es.envs) {
		return es.viewList()
	}

	envObj := es.envs[es.cursor]
	var sb strings.Builder

	sb.WriteString(Style.Header.Render("Delete Environment?"))
	sb.WriteString("\n")
	sb.WriteString("\n")
	sb.WriteString(Style.PlainText.Render(fmt.Sprintf("  Delete '%s'?\n", envObj.Name)))
	sb.WriteString("\n")

	if es.confirmDelete {
		sb.WriteString(Style.Hint.Render("  Press y again to confirm\n"))
	} else {
		sb.WriteString(Style.Hint.Render("  Press y to confirm\n"))
	}
	sb.WriteString(Style.Hint.Render("  Press q or n to cancel"))

	return sb.String()
}

// viewImport shows the import form
func (es *EnvSwitcher) viewImport() string {
	var sb strings.Builder

	sb.WriteString(Style.Header.Render("Import from .env"))
	sb.WriteString("\n")
	sb.WriteString("\n")

	sb.WriteString("  Path: ")
	if es.importPath == "" {
		sb.WriteString(Style.Hint.Render("(enter file path)"))
	} else {
		sb.WriteString(es.importPath + "_")
	}
	sb.WriteString("\n")

	sb.WriteString("\n")
	sb.WriteString(Style.Hint.Render("  Enter: import  q: cancel"))

	return sb.String()
}

// maskSecretsForDisplay masks secret values in a dotenv-style string
func maskSecretsForDisplay(content string, secretKeys map[string]bool) string {
	if len(secretKeys) == 0 {
		return content
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.Contains(line, "=") {
			eqIndex := strings.Index(line, "=")
			key := strings.TrimSpace(line[:eqIndex])
			if secretKeys[key] {
				value := strings.TrimSpace(line[eqIndex+1:])
				lines[i] = key + " = *****"
				_ = value // avoid unused variable
			}
		}
	}
	return strings.Join(lines, "\n")
}

// GetActiveEnvName returns the currently active environment name
func (es *EnvSwitcher) GetActiveEnvName() string {
	return es.activeEnvName
}

// GetActiveEnvID returns the currently active environment ID
func (es *EnvSwitcher) GetActiveEnvID() string {
	return es.activeEnvID
}

// GetEnvByID returns an environment by ID
func (es *EnvSwitcher) GetEnvByID(id string) *env.Environment {
	for _, e := range es.envs {
		if e.ID == id {
			return e
		}
	}
	return nil
}

// GetMessages returns any messages generated by the switcher
func (es *EnvSwitcher) GetMessages() []tea.Msg {
	msgs := es.msgs
	es.msgs = []tea.Msg{}
	return msgs
}

// SetActiveEnvByName sets the active environment by name
func (es *EnvSwitcher) SetActiveEnvByName(name string) error {
	es.activeEnvName = name

	// Find env by name
	for _, e := range es.envs {
		if e.Name == name {
			es.activeEnvID = e.ID
			break
		}
	}

	return es.envStorage.SetActiveEnv(name)
}

// AddEnvFromFile adds a new environment from a .env file
func (es *EnvSwitcher) AddEnvFromFile(path string) (*env.Environment, error) {
	vars, err := env.ParseDotenvFile(path)
	if err != nil {
		return nil, err
	}

	// Get filename as env name
	parts := strings.Split(path, "/")
	filename := parts[len(parts)-1]
	envName := strings.TrimSuffix(filename, ".env")
	if envName == "" {
		envName = "imported"
	}

	newEnv := env.NewEnvironment(envName, "")
	for k, v := range vars {
		newEnv.SetVariable(k, v)
	}

	if err := es.envStorage.SaveEnv(newEnv); err != nil {
		return nil, err
	}

	es.loadEnvs()
	return newEnv, nil
}

// AddEnv creates a new environment with the given name and vars
func (es *EnvSwitcher) AddEnv(name string, vars map[string]string) (*env.Environment, error) {
	newEnv := env.NewEnvironment(name, "")
	for k, v := range vars {
		newEnv.SetVariable(k, v)
	}

	if err := es.envStorage.SaveEnv(newEnv); err != nil {
		return nil, err
	}

	es.loadEnvs()
	return newEnv, nil
}

// DeleteEnvByID deletes an environment by ID
func (es *EnvSwitcher) DeleteEnvByID(id string) error {
	if err := es.envStorage.DeleteEnv(id); err != nil {
		return err
	}

	if es.activeEnvID == id {
		es.activeEnvID = ""
		es.activeEnvName = ""
		es.envStorage.SetActiveEnv("")
	}

	es.loadEnvs()
	return nil
}

// UpdateEnvVar updates a variable in an environment
func (es *EnvSwitcher) UpdateEnvVar(envID, key, value string, isSecret bool) error {
	envObj := es.GetEnvByID(envID)
	if envObj == nil {
		return fmt.Errorf("environment not found: %s", envID)
	}

	if isSecret {
		envObj.SetSecretVariable(key, value)
	} else {
		envObj.SetVariable(key, value)
	}

	return es.envStorage.SaveEnv(envObj)
}

// EnvSwitcherStyle returns lipgloss styles for the env switcher
var EnvSwitcherStyle = struct {
	EnvName   lipgloss.Style
	EnvActive lipgloss.Style
	EnvVars   lipgloss.Style
	EnvSecret lipgloss.Style
	EnvPrompt lipgloss.Style
	EnvInput  lipgloss.Style
}{
	EnvName: lipgloss.NewStyle().
		Foreground(lipgloss.Color("228")). // Bright yellow
		Bold(true),

	EnvActive: lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")). // Green
		Bold(true),

	EnvVars: lipgloss.NewStyle().
		Foreground(lipgloss.Color("75")), // Cyan

	EnvSecret: lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")), // Dim

	EnvPrompt: lipgloss.NewStyle().
		Foreground(lipgloss.Color("36")). // Cyan
		Bold(true),

	EnvInput: lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")), // White
}

// CreateEnvWithPrompt is a helper to create an env via CLI prompt (for non-TUI use)
func CreateEnvWithPrompt(name string, vars map[string]string) (*env.Environment, error) {
	newEnv := env.NewEnvironment(name, "")
	for k, v := range vars {
		newEnv.SetVariable(k, v)
	}
	return newEnv, nil
}

// Getenv is a simple env getter for use outside TUI
func Getenv(envName string, key string, envStorage *env.EnvStorage) (string, bool) {
	if envName == "" {
		return "", false
	}

	envObj, err := envStorage.GetEnvByName(envName)
	if err != nil {
		return "", false
	}

	val, ok := envObj.GetVariable(key)
	return val, ok
}

// GetActiveEnv returns the active environment instance, if resolved.
func (es *EnvSwitcher) GetActiveEnv() *env.Environment {
	if es == nil {
		return nil
	}
	if es.activeEnvName != "" {
		envObj := es.GetEnvByName(es.activeEnvName)
		if envObj != nil {
			return envObj
		}
	}
	if es.activeEnvID != "" {
		if envObj := es.GetEnvByID(es.activeEnvID); envObj != nil {
			return envObj
		}
	}
	if es.cursor >= 0 && es.cursor < len(es.envs) {
		return es.envs[es.cursor]
	}
	return nil
}

// GetEnvByName returns an environment by name.
func (es *EnvSwitcher) GetEnvByName(name string) *env.Environment {
	if es == nil || name == "" {
		return nil
	}
	for _, e := range es.envs {
		if e.Name == name {
			return e
		}
	}
	if es.envStorage != nil {
		e, err := es.envStorage.GetEnvByName(name)
		if err == nil {
			return e
		}
	}
	return nil
}

// GetActiveEnvVariables returns active environment variables.
func (es *EnvSwitcher) GetActiveEnvVariables() map[string]string {
	vars := make(map[string]string)
	envObj := es.GetActiveEnv()
	if envObj == nil {
		return vars
	}
	for k, v := range envObj.Variables {
		vars[k] = v
	}
	return vars
}

// ReadDotenvFile reads a .env file and returns the variables
func ReadDotenvFile(path string) (map[string]string, error) {
	return env.ParseDotenvFile(path)
}

// Ensure we implement tea.Model
var _ tea.Model = (*EnvSwitcher)(nil)

// GetSelectedEnv returns the currently selected environment
func (es *EnvSwitcher) GetSelectedEnv() *env.Environment {
	if es.cursor < 0 || es.cursor >= len(es.envs) {
		return nil
	}
	return es.envs[es.cursor]
}

// ImportEnvFromFile imports an environment from a .env file path
func ImportEnvFromFile(envStorage *env.EnvStorage, path string) (*env.Environment, error) {
	vars, err := env.ParseDotenvFile(path)
	if err != nil {
		return nil, err
	}

	// Get filename as env name
	parts := strings.Split(path, "/")
	filename := parts[len(parts)-1]
	envName := strings.TrimSuffix(filename, ".env")
	if envName == "" {
		envName = "imported"
	}

	newEnv := env.NewEnvironment(envName, "")
	for k, v := range vars {
		newEnv.SetVariable(k, v)
	}

	if err := envStorage.SaveEnv(newEnv); err != nil {
		return nil, err
	}

	return newEnv, nil
}

// ImportEnvFromContent imports an environment from .env content string
func ImportEnvFromContent(envStorage *env.EnvStorage, name, content string) (*env.Environment, error) {
	vars, err := env.ParseDotenv(content)
	if err != nil {
		return nil, err
	}

	newEnv := env.NewEnvironment(name, "")
	for k, v := range vars {
		newEnv.SetVariable(k, v)
	}

	if err := envStorage.SaveEnv(newEnv); err != nil {
		return nil, err
	}

	return newEnv, nil
}

// FileExists checks if a file exists (simple version)
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
