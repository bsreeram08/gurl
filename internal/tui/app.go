package tui

import (
	"context"
	"fmt"
	"image/color"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/google/uuid"
	"github.com/sreeram/gurl/internal/core/template"
	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/importers"
	"github.com/sreeram/gurl/internal/runner"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

// requestsLoadedMsg is sent when requests finish loading
type requestsLoadedMsg struct {
	requests []*types.SavedRequest
}

// Panel represents the focused panel in the TUI
type Panel int

const (
	PanelSidebar Panel = iota
	PanelMain
	PanelResponse
	PanelStatusbar = PanelResponse
)

// Tab represents a single request tab
type Tab struct {
	ID      string
	Name    string
	Request *types.SavedRequest
	Builder *RequestBuilder
	Viewer  *ResponseViewer
}

// SidebarMode controls what the sidebar shows
type SidebarMode int

const (
	SidebarRequests SidebarMode = iota
	SidebarHistory
)

// SearchModal provides a quick-search overlay (Cmd+K)
type SearchModal struct {
	input       textinput.Model
	results     []*types.SavedRequest
	selectedIdx int
	allRequests []*types.SavedRequest
	width       int
	height      int
}

// makeView creates a tea.View with standard TUI settings (alt screen, mouse)
func makeView(content string) tea.View {
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// NewSearchModal creates a new search modal
func NewSearchModal(allRequests []*types.SavedRequest, width, height int) *SearchModal {
	ti := textinput.New()
	ti.Placeholder = "Search requests..."
	ti.Prompt = "> "
	ti.Focus()

	return &SearchModal{
		input:       ti,
		results:     allRequests,
		selectedIdx: 0,
		allRequests: allRequests,
		width:       width,
		height:      height,
	}
}

func (sm *SearchModal) filterResults() {
	query := strings.ToLower(sm.input.Value())
	if query == "" {
		sm.results = sm.allRequests
	} else {
		filtered := []*types.SavedRequest{}
		for _, req := range sm.allRequests {
			if strings.Contains(strings.ToLower(req.Name), query) ||
				strings.Contains(strings.ToLower(req.URL), query) {
				filtered = append(filtered, req)
			}
		}
		sm.results = filtered
	}
	if sm.selectedIdx >= len(sm.results) {
		sm.selectedIdx = 0
	}
}

func (sm *SearchModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if sm.selectedIdx > 0 {
				sm.selectedIdx--
			}
		case "down", "j":
			if sm.selectedIdx < len(sm.results)-1 {
				sm.selectedIdx++
			}
		case "enter":
			// Return nil, nil to indicate "open selected"
			return nil, nil
		case "esc":
			// Return nil, nil to indicate "cancel"
			return nil, nil
		}
	}
	sm.input, _ = sm.input.Update(msg)
	sm.filterResults()
	return sm, nil
}

func (sm *SearchModal) View() tea.View {
	lines := []string{
		Style.Header.Render(" Search "),
		sm.input.View(),
		"",
	}

	maxResults := sm.height - 10
	if maxResults < 5 {
		maxResults = 5
	}

	// Use local variable for display to avoid mutating live state
	displayResults := sm.results
	if len(displayResults) > maxResults {
		displayResults = displayResults[:maxResults]
	}

	for i, req := range displayResults {
		methodColor := methodTextColor(req.Method)
		methodStyle := lipgloss.NewStyle().Foreground(methodColor).Bold(true)
		name := req.Name
		if name == "" {
			name = req.URL
		}
		if len(name) > sm.width-20 {
			name = name[:sm.width-23] + "..."
		}
		if i == sm.selectedIdx {
			lines = append(lines, Style.SelectedItem.Render(fmt.Sprintf("▶ %s %s", methodStyle.Render(req.Method), name)))
		} else {
			lines = append(lines, Style.ListItem.Render(fmt.Sprintf("  %s %s", methodStyle.Render(req.Method), name)))
		}
	}

	if len(displayResults) == 0 {
		lines = append(lines, Style.PlainText.Render("  No matching requests"))
	}

	lines = append(lines, "")
	lines = append(lines, Style.Hint.Render("  ↑↓ navigate  ·  Enter open  ·  Esc close"))

	content := strings.Join(lines, "\n")
	modalWidth := sm.width - 20
	if modalWidth < 40 {
		modalWidth = 40
	}
	return makeView(Style.Modal.
		Width(modalWidth).
		Height(sm.height - 5).
		Render(content))
}

func (sm *SearchModal) GetSelectedRequest() *types.SavedRequest {
	if sm.selectedIdx >= 0 && sm.selectedIdx < len(sm.results) {
		return sm.results[sm.selectedIdx]
	}
	return nil
}

// Init implements tea.Model.Init
func (sm *SearchModal) Init() tea.Cmd {
	return nil
}

type ImportModal struct {
	input    textinput.Model
	errorMsg string
	width    int
	height   int
}

func NewImportModal(width, height int) *ImportModal {
	ti := textinput.New()
	ti.Placeholder = "Enter file path or URL to import..."
	ti.Prompt = "> "
	ti.Focus()

	return &ImportModal{
		input:  ti,
		width:  width,
		height: height,
	}
}

func (im *ImportModal) Update(msg tea.Msg) *ImportModal {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			return im
		case "esc":
			return nil
		}
	}
	im.input, _ = im.input.Update(msg)
	return im
}

func (im *ImportModal) View() tea.View {
	lines := []string{
		Style.Header.Render(" Import "),
		im.input.View(),
		"",
	}

	if im.errorMsg != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		lines = append(lines, errStyle.Render("  Error: "+im.errorMsg))
		lines = append(lines, "")
	}

	formats := "  Supported: .json (Insomnia/Postman/HAR), .yaml/.yml (OpenAPI), .toml (Gurl), .bru (Bruno)"
	lines = append(lines, Style.Hint.Render(formats))
	lines = append(lines, "")
	lines = append(lines, Style.Hint.Render("  Enter import  ·  Esc cancel"))

	content := strings.Join(lines, "\n")
	modalWidth := im.width - 20
	if modalWidth < 50 {
		modalWidth = 50
	}
	return makeView(Style.Modal.
		Width(modalWidth).
		Height(im.height - 5).
		Render(content))
}

func (im *ImportModal) GetPath() string {
	return im.input.Value()
}

func (im *ImportModal) SetError(msg string) {
	im.errorMsg = msg
}

// RunnerModal manages collection runner state
type RunnerModal struct {
	collections []string
	selectedIdx int
	envs        []string
	envIdx      int
	iterations  int
	delay       int
	running     bool
	results     []*runner.RequestResult
	current     int
	total       int
	width       int
	height      int
	mu          sync.Mutex
	cancelCtx   context.Context
	cancelFn    context.CancelFunc
	done        chan struct{}
}

func NewRunnerModal(collections []string, envs []string, width, height int) *RunnerModal {
	idx := 0
	if len(collections) > 0 {
		idx = 0
	}
	return &RunnerModal{
		collections: collections,
		selectedIdx: idx,
		envs:        envs,
		envIdx:      0,
		iterations:  1,
		delay:       0,
		width:       width,
		height:      height,
	}
}

func (rm *RunnerModal) Update(msg tea.Msg) *RunnerModal {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if rm.selectedIdx > 0 {
				rm.selectedIdx--
			}
		case "down", "j":
			if rm.selectedIdx < len(rm.collections)-1 {
				rm.selectedIdx++
			}
		case "enter":
			if !rm.running {
				rm.running = true
			}
		case "esc":
			return nil
		}
	}
	return rm
}

func (rm *RunnerModal) View() tea.View {
	lines := []string{
		Style.Header.Render(" Collection Runner "),
		"",
	}

	if !rm.running {
		if len(rm.collections) == 0 {
			lines = append(lines, Style.PlainText.Render("  No collections found"))
			lines = append(lines, Style.Hint.Render("  Save requests to collections first"))
		} else {
			lines = append(lines, "  Collection:", "")
			for i, col := range rm.collections {
				prefix := "  "
				if i == rm.selectedIdx {
					prefix = "▶ "
					selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Bold(true)
					lines = append(lines, selStyle.Render(prefix+col))
				} else {
					lines = append(lines, Style.ListItem.Render(prefix+col))
				}
			}
			lines = append(lines, "")
			lines = append(lines, fmt.Sprintf("  Iterations: %d", rm.iterations))
			lines = append(lines, fmt.Sprintf("  Delay: %dms", rm.delay))
			lines = append(lines, "")
			lines = append(lines, Style.Hint.Render("  Enter: run  ·  Esc: cancel"))
		}
	} else {
		lines = append(lines, fmt.Sprintf("  Running: %s", rm.getCollectionName()))
		if rm.total > 0 {
			lines = append(lines, fmt.Sprintf("  Progress: %d/%d", rm.current, rm.total))
		}
		lines = append(lines, "")
		for i, res := range rm.results {
			if i >= rm.current {
				break
			}
			status := "✓"
			statusColor := lipgloss.Color("82")
			if res.Error != "" {
				status = "✗"
				statusColor = lipgloss.Color("196")
			} else if res.Skipped {
				status = "⊘"
				statusColor = lipgloss.Color("240")
			}
			statusStyle := lipgloss.NewStyle().Foreground(statusColor).Bold(true)
			name := res.RequestName
			if name == "" {
				name = fmt.Sprintf("Request %d", i+1)
			}
			if len(name) > 40 {
				name = name[:37] + "..."
			}
			duration := res.Duration.Milliseconds()
			lines = append(lines, fmt.Sprintf("  %s %s (%dms)", statusStyle.Render(status), name, duration))
		}
		lines = append(lines, "")
		if rm.current < rm.total || rm.total == 0 {
			lines = append(lines, Style.Hint.Render("  Running... Press Esc to cancel"))
		} else {
			passed := 0
			failed := 0
			for _, r := range rm.results {
				if r.Error != "" || !r.Passed {
					failed++
				} else {
					passed++
				}
			}
			summaryColor := lipgloss.Color("82")
			if failed > 0 {
				summaryColor = lipgloss.Color("196")
			}
			summaryStyle := lipgloss.NewStyle().Foreground(summaryColor).Bold(true)
			lines = append(lines, summaryStyle.Render(fmt.Sprintf("  Done: %d passed, %d failed", passed, failed)))
			lines = append(lines, Style.Hint.Render("  Press Esc to close"))
		}
	}

	content := strings.Join(lines, "\n")
	modalWidth := rm.width - 20
	if modalWidth < 50 {
		modalWidth = 50
	}
	return makeView(Style.Modal.
		Width(modalWidth).
		Height(rm.height - 5).
		Render(content))
}

func (rm *RunnerModal) getCollectionName() string {
	if rm.selectedIdx >= 0 && rm.selectedIdx < len(rm.collections) {
		return rm.collections[rm.selectedIdx]
	}
	return ""
}

func (rm *RunnerModal) GetSelectedCollection() string {
	return rm.getCollectionName()
}

func (rm *RunnerModal) IsRunning() bool {
	return rm.running
}

func (rm *RunnerModal) SetRunning(running bool) {
	rm.running = running
}

func (rm *RunnerModal) SetTotal(total int) {
	rm.total = total
}

// StartRun begins the collection run with cancellation support
func (rm *RunnerModal) StartRun(collection string, db storage.DB, r *runner.Runner) {
	rm.cancelCtx, rm.cancelFn = context.WithCancel(context.Background())
	rm.done = make(chan struct{})
	reqs, _ := db.ListRequests(&storage.ListOptions{Collection: collection})
	rm.total = len(reqs)
	go func() {
		defer close(rm.done)
		res, _ := r.Run(rm.cancelCtx, runner.RunConfig{CollectionName: collection})
		for _, runResult := range res {
			for _, rr := range runResult.RequestResults {
				select {
				case <-rm.cancelCtx.Done():
					return
				default:
					rm.AppendResult(rr)
				}
			}
		}
	}()
}

// CancelRun signals the goroutine to stop
func (rm *RunnerModal) CancelRun() {
	if rm.cancelFn != nil {
		rm.cancelFn()
	}
	if rm.done != nil {
		<-rm.done
	}
}

func (rm *RunnerModal) AppendResult(result *runner.RequestResult) {
	rm.mu.Lock()
	rm.results = append(rm.results, result)
	rm.current++
	rm.mu.Unlock()
}

func (rm *RunnerModal) GetResults() []*runner.RequestResult {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	return rm.results
}

// App is the root bubbletea model
type App struct {
	db           storage.DB
	config       *types.Config
	version      string
	requests     []*types.SavedRequest
	history      []*types.HistoryEntry
	selectedIdx  int
	focusedPanel Panel
	width        int
	height       int
	ready        bool
	quitting     bool

	// Tab state
	tabs        []*Tab
	activeTab   int
	sidebarMode SidebarMode

	requestList     *RequestList
	searchModal     *SearchModal
	importModal     *ImportModal
	runnerModal     *RunnerModal
	envSwitcher     *EnvSwitcher
	envSwitcherOpen bool
	activeEnvName   string
	helpModal       bool
	loadingSpinner  spinner.Model
	variablePrompt  *VariablePrompt
	quickTuiMode    bool
}

// NewApp creates a new TUI application
func NewApp(db storage.DB, config *types.Config) *App {
	return NewAppWithVersion(db, config, "dev")
}

// NewAppWithVersion creates a new TUI application with build metadata.
func NewAppWithVersion(db storage.DB, config *types.Config, version string) *App {
	initialTab := newBlankTab(db)
	rl := NewRequestList(db)
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	envSwitcher, activeEnvName := newAppEnvSwitcher(db)
	return &App{
		db:             db,
		config:         config,
		version:        version,
		focusedPanel:   PanelSidebar,
		width:          80,
		height:         24,
		tabs:           []*Tab{initialTab},
		activeTab:      0,
		sidebarMode:    SidebarRequests,
		requestList:    rl,
		envSwitcher:    envSwitcher,
		activeEnvName:  activeEnvName,
		loadingSpinner: sp,
		quickTuiMode:   false,
	}
}

func newAppEnvSwitcher(db storage.DB) (*EnvSwitcher, string) {
	lmdb, ok := db.(*storage.LMDB)
	if !ok || lmdb == nil || lmdb.DB == nil {
		return nil, ""
	}

	switcher := NewEnvSwitcher(env.NewEnvStorage(lmdb))
	return switcher, switcher.GetActiveEnvName()
}

func newBlankTab(db storage.DB) *Tab {
	return &Tab{
		ID:      uuid.New().String(),
		Name:    "New Request",
		Request: nil,
		Builder: NewRequestBuilder(db),
		Viewer:  NewResponseViewer(),
	}
}

func newTabForRequest(db storage.DB, req *types.SavedRequest) *Tab {
	tab := &Tab{
		ID:      uuid.New().String(),
		Name:    req.Name,
		Request: req,
		Builder: NewRequestBuilder(db),
		Viewer:  NewResponseViewer(),
	}
	tab.Builder.LoadRequest(req)
	return tab
}

// Init implements tea.Model.Init
func (m *App) Init() tea.Cmd {
	return func() tea.Msg {
		requests, _ := m.db.ListRequests(nil)
		if requests == nil {
			requests = []*types.SavedRequest{}
		}
		return requestsLoadedMsg{requests: requests}
	}
}

// Update implements tea.Model.Update
func (m *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Update spinner when loading
	if !m.ready {
		m.loadingSpinner, _ = m.loadingSpinner.Update(msg)
	}

	if m.variablePrompt != nil {
		updated, cmd := m.variablePrompt.Update(msg)
		if updated == nil {
			m.variablePrompt = nil
		} else if vp, ok := updated.(*VariablePrompt); ok {
			m.variablePrompt = vp
		}
		if cmd != nil {
			return m, cmd
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.applyLayout(CalculateLayout(msg.Width, msg.Height))
		if m.envSwitcher != nil {
			m.envSwitcher.Update(msg)
		}
		return m, nil

	case requestsLoadedMsg:
		m.requests = msg.requests
		m.requestList.SetRequests(msg.requests)
		return m, nil

	case tea.KeyPressMsg:
		if m.searchModal != nil {
			result, cmd := m.searchModal.Update(msg)
			if result == nil {
				if req := m.searchModal.GetSelectedRequest(); req != nil {
					m.searchModal = nil
					m.openInNewTab(req)
					return m, nil
				}
				m.searchModal = nil
			}
			return m, cmd
		}
		if m.importModal != nil {
			result := m.importModal.Update(msg)
			if result == nil {
				path := m.importModal.GetPath()
				m.importModal = nil
				if path != "" {
					imported, err := importers.AutoDetectImport(path)
					if err != nil {
						m.importModal = NewImportModal(m.width, m.height)
						m.importModal.SetError(err.Error())
						return m, nil
					}
					for _, req := range imported {
						m.db.SaveRequest(req)
					}
					requests, _ := m.db.ListRequests(nil)
					if requests != nil {
						m.requests = requests
						m.requestList.SetRequests(requests)
					}
				}
				return m, nil
			}
			return m, nil
		}
		if m.runnerModal != nil {
			result := m.runnerModal.Update(msg)
			if result == nil {
				if m.runnerModal.IsRunning() {
					m.runnerModal.CancelRun()
				}
				m.runnerModal = nil
				return m, nil
			}
			if m.runnerModal.IsRunning() {
				col := m.runnerModal.GetSelectedCollection()
				if col != "" && m.runnerModal.total == 0 {
					r := runner.NewRunner(m.db, nil)
					m.runnerModal.StartRun(col, m.db, r)
				}
			}
			return m, nil
		}
		if m.envSwitcherOpen && m.envSwitcher != nil {
			if msg.Code == tea.KeyEscape || msg.String() == "q" {
				m.closeEnvSwitcher()
				return m, nil
			}

			updated, cmd := m.envSwitcher.Update(msg)
			if es, ok := updated.(*EnvSwitcher); ok {
				m.envSwitcher = es
			}
			for _, envMsg := range m.envSwitcher.GetMessages() {
				if changed, ok := envMsg.(EnvChangedMsg); ok {
					m.activeEnvName = changed.EnvName
					m.closeEnvSwitcher()
				}
			}
			return m, cmd
		}
		if m.helpModal {
			if msg.Code == tea.KeyEscape || msg.String() == "q" || msg.String() == "?" {
				m.helpModal = false
			}
			return m, nil
		}
		if handled, cmd := m.handleGlobalKeyPress(msg); handled {
			return m, cmd
		}
		return m.routeFocusedKeyPress(msg)

	case BuilderRequestSelectedMsg:
		m.openInNewTab(msg.Request)
		return m, nil

	case RequestSentMsg:
		if m.activeTab < len(m.tabs) {
			tab := m.tabs[m.activeTab]
			if tab.Builder != nil {
				tab.Builder.sending = false
			}
			if tab.Viewer != nil {
				if msg.Error != nil {
					tab.Viewer.SetError(msg.Error)
				} else {
					tab.Viewer.SetResponse(msg.Response)
				}
			}
		}
		return m, nil

	case RequestSavedMsg:
		requests, _ := m.db.ListRequests(nil)
		if requests == nil {
			requests = []*types.SavedRequest{}
		}
		m.requests = requests
		m.requestList.SetRequests(requests)
		if msg.Request != nil && m.activeTab < len(m.tabs) {
			m.tabs[m.activeTab].Name = msg.Request.Name
			m.tabs[m.activeTab].Request = msg.Request
		}
		return m, nil

	case RequestSendRequestedMsg:
		return m, m.requestSendWithVarsFlow(nil)

	case VariablePromptDoneMsg:
		m.variablePrompt = nil
		if msg.Cancelled {
			return m, nil
		}
		if m.activeTab < len(m.tabs) {
			if cmd := m.sendActiveRequestWithVars(msg.Variables); cmd != nil {
				return m, cmd
			}
		}
		return m, nil
	}

	// Route to active tab's components
	if m.activeTab < len(m.tabs) {
		tab := m.tabs[m.activeTab]
		if tab.Builder != nil {
			updatedBuilder, cmd := tab.Builder.Update(msg)
			if b, ok := updatedBuilder.(*RequestBuilder); ok {
				tab.Builder = b
			}
			if cmd != nil {
				return m, cmd
			}
		}
		if tab.Viewer != nil {
			updatedViewer, cmd := tab.Viewer.Update(msg)
			if v, ok := updatedViewer.(*ResponseViewer); ok {
				tab.Viewer = v
			}
			if cmd != nil {
				return m, cmd
			}
		}
	}

	return m, nil
}

func (m *App) requestSendWithVarsFlow(overrides map[string]string) tea.Cmd {
	if m.activeTab >= len(m.tabs) {
		return nil
	}

	tab := m.tabs[m.activeTab]
	if tab == nil || tab.Builder == nil {
		return nil
	}

	req := tab.Builder.GetEditingRequest()
	if req == nil {
		return nil
	}

	vars := m.collectDefaults(req)
	for k, v := range m.collectActiveEnvVars() {
		vars[k] = v
	}
	for k, v := range overrides {
		vars[k] = v
	}

	missing := m.missingTemplateVars(req, vars)
	if len(missing) > 0 {
		if m.quickTuiMode {
			return func() tea.Msg {
				return RequestSentMsg{Response: nil, Error: fmt.Errorf("missing template variables: %s", strings.Join(missing, ", "))}
			}
		}

		name := req.Name
		if name == "" {
			name = "Request"
		}
		m.variablePrompt = NewVariablePrompt(name, missing, vars)
		return nil
	}

	return m.sendActiveRequestWithVars(vars)
}

func (m *App) sendActiveRequestWithVars(vars map[string]string) tea.Cmd {
	if m.activeTab >= len(m.tabs) {
		return nil
	}

	tab := m.tabs[m.activeTab]
	if tab == nil || tab.Builder == nil {
		return nil
	}

	return tab.Builder.sendRequestWithVars(vars)
}

// SetQuickMode enables a minimal execution mode that skips modal variable prompting.
func (m *App) SetQuickMode(enabled bool) {
	if m != nil {
		m.quickTuiMode = enabled
	}
}

func (m *App) collectDefaults(req *types.SavedRequest) map[string]string {
	vars := make(map[string]string)
	if req == nil {
		return vars
	}

	for _, v := range req.Variables {
		if v.Name == "" {
			continue
		}
		vars[v.Name] = v.Example
	}

	for _, p := range req.PathParams {
		if p.Name == "" {
			continue
		}
		if p.Example != "" {
			vars[p.Name] = p.Example
		}
	}

	return vars
}

func (m *App) collectActiveEnvVars() map[string]string {
	vars := make(map[string]string)
	if m.envSwitcher != nil {
		for k, v := range m.envSwitcher.GetActiveEnvVariables() {
			vars[k] = v
		}
	}
	return vars
}

func (m *App) missingTemplateVars(req *types.SavedRequest, provided map[string]string) []string {
	seen := make(map[string]bool)
	ordered := make([]string, 0)

	add := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		ordered = append(ordered, name)
	}

	appendNames := func(values []string) {
		for _, name := range values {
			add(name)
		}
	}

	appendNames(template.ExtractVarNames(req.URL))
	appendNames(template.ExtractVarNames(req.Body))

	for _, h := range req.Headers {
		appendNames(template.ExtractVarNames(h.Key))
		appendNames(template.ExtractVarNames(h.Value))
	}

	if req.AuthConfig != nil {
		switch req.AuthConfig.Type {
		case "basic":
			appendNames(template.ExtractVarNames(req.AuthConfig.Params["username"]))
			appendNames(template.ExtractVarNames(req.AuthConfig.Params["password"]))
		case "bearer":
			appendNames(template.ExtractVarNames(req.AuthConfig.Params["token"]))
		case "apikey":
			appendNames(template.ExtractVarNames(req.AuthConfig.Params["header"]))
			appendNames(template.ExtractVarNames(req.AuthConfig.Params["value"]))
		}
	}

	for _, p := range req.PathParams {
		add(p.Name)
	}

	if len(ordered) == 0 {
		for _, v := range template.GetVariablesFromRequest(req) {
			add(v.Name)
		}
	}

	missing := make([]string, 0, len(ordered))
	for _, name := range ordered {
		if value, ok := provided[name]; !ok || strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}

	return missing
}

// handleGlobalKeyPress handles app-level shortcuts before a focused pane processes the key.
func (m *App) handleGlobalKeyPress(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	key := msg.String()

	switch key {
	case "ctrl+t":
		m.openNewTab()
		return true, nil
	case "ctrl+w":
		m.closeActiveTab()
		return true, nil
	case "ctrl+d":
		if m.activeTab < len(m.tabs) && m.tabs[m.activeTab].Request != nil {
			m.duplicateActiveTab()
		}
		return true, nil
	case "ctrl+shift+]":
		m.nextTab()
		return true, nil
	case "ctrl+shift+[":
		m.prevTab()
		return true, nil
	case "ctrl+k":
		m.searchModal = NewSearchModal(m.requests, m.width, m.height)
		return true, nil
	case "ctrl+i":
		m.importModal = NewImportModal(m.width, m.height)
		return true, nil
	case "ctrl+r":
		collections := []string{}
		seen := make(map[string]bool)
		for _, req := range m.requests {
			if req.Collection != "" && !seen[req.Collection] {
				seen[req.Collection] = true
				collections = append(collections, req.Collection)
			}
		}
		m.runnerModal = NewRunnerModal(collections, []string{}, m.width, m.height)
		return true, nil
	case "ctrl+1":
		m.focusedPanel = PanelSidebar
		return true, nil
	case "ctrl+2":
		m.focusedPanel = PanelMain
		return true, nil
	case "ctrl+3":
		m.focusedPanel = PanelResponse
		return true, nil
	case "ctrl+e":
		if m.envSwitcher != nil {
			m.envSwitcherOpen = true
			m.envSwitcher.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		}
		return true, nil
	case "ctrl+c":
		m.quitting = true
		return true, tea.Quit
	}

	if m.focusedPanel != PanelMain {
		switch {
		case IsQuitKey(key):
			m.quitting = true
			return true, tea.Quit
		case IsHelpKey(key):
			m.helpModal = !m.helpModal
			return true, nil
		case m.focusedPanel == PanelSidebar && key == "h":
			m.sidebarMode = SidebarHistory
			return true, nil
		case m.focusedPanel == PanelSidebar && key == "r":
			m.sidebarMode = SidebarRequests
			return true, nil
		}
	}

	return false, nil
}

// openNewTab opens a new blank tab
func (m *App) openNewTab() {
	tab := newBlankTab(m.db)
	m.tabs = append(m.tabs, tab)
	m.activeTab = len(m.tabs) - 1
	if m.ready {
		m.applyLayout(CalculateLayout(m.width, m.height))
	}
}

// openInNewTab opens a request in a new tab
func (m *App) openInNewTab(req *types.SavedRequest) {
	tab := newTabForRequest(m.db, req)
	m.tabs = append(m.tabs, tab)
	m.activeTab = len(m.tabs) - 1
	if m.ready {
		m.applyLayout(CalculateLayout(m.width, m.height))
	}
}

// closeActiveTab closes the current tab; keeps at least one tab open
func (m *App) closeActiveTab() {
	if len(m.tabs) <= 1 {
		m.tabs[0] = newBlankTab(m.db)
		m.activeTab = 0
		if m.ready {
			m.applyLayout(CalculateLayout(m.width, m.height))
		}
		return
	}
	m.tabs = append(m.tabs[:m.activeTab], m.tabs[m.activeTab+1:]...)
	if m.activeTab >= len(m.tabs) {
		m.activeTab = len(m.tabs) - 1
	}
	if m.ready {
		m.applyLayout(CalculateLayout(m.width, m.height))
	}
}

// nextTab cycles to the next tab
func (m *App) nextTab() {
	if len(m.tabs) <= 1 {
		return
	}
	m.activeTab = (m.activeTab + 1) % len(m.tabs)
}

// prevTab cycles to the previous tab
func (m *App) prevTab() {
	if len(m.tabs) <= 1 {
		return
	}
	m.activeTab--
	if m.activeTab < 0 {
		m.activeTab = len(m.tabs) - 1
	}
}

// duplicateActiveTab duplicates the current tab's request
func (m *App) duplicateActiveTab() {
	if m.activeTab >= len(m.tabs) {
		return
	}
	tab := m.tabs[m.activeTab]
	var req *types.SavedRequest
	if tab.Request != nil {
		req = tab.Request
	} else {
		req = tab.Builder.GetEditingRequest()
	}
	if req == nil {
		return
	}
	newReq := *req
	newReq.ID = ""
	newReq.Name = req.Name + " (copy)"
	newReq.CreatedAt = time.Now().Unix()
	newReq.UpdatedAt = time.Now().Unix()
	if err := m.db.SaveRequest(&newReq); err == nil {
		requests, _ := m.db.ListRequests(nil)
		if requests != nil {
			m.requests = requests
		}
		m.openInNewTab(&newReq)
	}
}

func (m *App) applyLayout(layout Layout) {
	sidebarWidth, sidebarHeight := panelContentSize(layout.SidebarWidth, layout.SidebarHeight)
	editorWidth, editorHeight := panelContentSize(layout.EditorWidth, layout.EditorHeight)
	responseWidth, responseHeight := panelContentSize(layout.ResponseWidth, layout.ResponseHeight)

	if m.requestList != nil {
		m.requestList.SetSize(sidebarWidth, max(4, sidebarHeight-2))
	}

	for _, tab := range m.tabs {
		if tab.Builder != nil {
			tab.Builder.SetSize(editorWidth, max(8, editorHeight-2))
		}
		if tab.Viewer != nil {
			tab.Viewer.SetSize(responseWidth, max(8, responseHeight-2))
		}
	}
}

func (m *App) closeEnvSwitcher() {
	m.envSwitcherOpen = false
	if m.envSwitcher != nil {
		if active := m.envSwitcher.GetActiveEnvName(); active != "" {
			m.activeEnvName = active
		}
	}
}

func (m *App) routeFocusedKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.focusedPanel {
	case PanelSidebar:
		key := msg.String()
		switch {
		case IsNavigateUpKey(key):
			m.requestList.CursorUp()
			return m, nil
		case IsNavigateDownKey(key):
			m.requestList.CursorDown()
			return m, nil
		case IsEnterKey(key):
			if m.sidebarMode == SidebarRequests {
				if req := m.requestList.GetSelectedRequest(); req != nil {
					m.openInNewTab(req)
				}
			}
			return m, nil
		case key == " ":
			m.requestList.ToggleFolderAtCursor()
			return m, nil
		}
	case PanelMain:
		if tab := m.ActiveTab(); tab != nil && tab.Builder != nil {
			updated, cmd := tab.Builder.Update(msg)
			if builder, ok := updated.(*RequestBuilder); ok {
				tab.Builder = builder
			}
			return m, cmd
		}
	case PanelResponse:
		if tab := m.ActiveTab(); tab != nil && tab.Viewer != nil {
			updated, cmd := tab.Viewer.Update(msg)
			if viewer, ok := updated.(*ResponseViewer); ok {
				tab.Viewer = viewer
			}
			return m, cmd
		}
	}

	return m, nil
}

func (m *App) renderHelpOverlay() string {
	var sb strings.Builder
	sb.WriteString(Style.Header.Render("Keyboard Shortcuts"))
	sb.WriteString("\n\n")

	keyStyle := lipgloss.NewStyle().Foreground(Color("primary")).Bold(true)
	groups := AllShortcuts()
	for idx, group := range groups {
		sb.WriteString(Style.PanelTitle.Render(group.Title))
		sb.WriteString("\n")
		for _, shortcut := range group.Keys {
			line := fmt.Sprintf("  %s  %s", keyStyle.Render(shortcut.Key), shortcut.Description)
			sb.WriteString(Style.ListItem.Render(line))
			sb.WriteString("\n")
		}
		if idx < len(groups)-1 {
			sb.WriteString("\n")
		}
	}

	sb.WriteString(Style.Hint.Render("Press ? or Esc to close"))

	modalWidth := min(max(54, m.width-18), 82)
	return Style.Modal.Width(modalWidth).Render(sb.String())
}

func (m *App) handleSpace() (tea.Model, tea.Cmd) {
	if m.focusedPanel == PanelSidebar && m.sidebarMode == SidebarRequests {
		m.requestList.ToggleFolderAtCursor()
	}
	return m, nil
}

// makeView creates a tea.View with alt screen and mouse mode enabled
func (m *App) makeView(content string) tea.View {
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m *App) View() tea.View {
	if !m.ready {
		return m.makeView(m.loadingSpinner.View() + " Loading...")
	}

	if m.variablePrompt != nil {
		return m.renderCenteredBlock(m.variablePrompt.View().Content)
	}

	if m.searchModal != nil {
		return m.renderCenteredBlock(m.searchModal.View().Content)
	}

	if m.helpModal {
		return m.renderCenteredBlock(m.renderHelpOverlay())
	}

	if m.envSwitcherOpen && m.envSwitcher != nil {
		content := Style.Modal.
			Width(min(max(40, m.width-24), 76)).
			Render(m.envSwitcher.View().Content)
		return m.renderCenteredBlock(content)
	}

	if m.importModal != nil {
		return m.renderCenteredBlock(m.importModal.View().Content)
	}

	if m.runnerModal != nil {
		return m.renderCenteredBlock(m.runnerModal.View().Content)
	}

	return m.makeView(m.renderWorkspace(CalculateLayout(m.width, m.height)))
}

func (m *App) renderCenteredBlock(content string) tea.View {
	topPadding := max(1, (m.height-lipgloss.Height(content))/3)
	leftPadding := max(0, (m.width-lipgloss.Width(content))/2)

	return m.makeView(lipgloss.JoinVertical(
		lipgloss.Left,
		strings.Repeat("\n", topPadding),
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			strings.Repeat(" ", leftPadding),
			content,
		),
	))
}

func panelContentSize(width, height int) (int, int) {
	return max(1, width-4), max(1, height-2)
}

func (m *App) renderWorkspace(layout Layout) string {
	var workspace string

	switch layout.Mode {
	case LayoutThreePane:
		workspace = lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.renderSidebarPane(layout.SidebarWidth, layout.SidebarHeight),
			m.renderEditorPane(layout.EditorWidth, layout.EditorHeight),
			m.renderResponsePane(layout.ResponseWidth, layout.ResponseHeight),
		)
	case LayoutSplitRight:
		right := lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderEditorPane(layout.EditorWidth, layout.EditorHeight),
			m.renderResponsePane(layout.ResponseWidth, layout.ResponseHeight),
		)
		workspace = lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.renderSidebarPane(layout.SidebarWidth, layout.SidebarHeight),
			right,
		)
	default:
		workspace = lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderSidebarPane(layout.SidebarWidth, layout.SidebarHeight),
			m.renderEditorPane(layout.EditorWidth, layout.EditorHeight),
			m.renderResponsePane(layout.ResponseWidth, layout.ResponseHeight),
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		workspace,
		m.renderStatusBar(layout),
	)
}

func (m *App) renderSidebarPane(width, height int) string {
	contentWidth, contentHeight := panelContentSize(width, height)
	header := RenderPanelHeader("Requests", fmt.Sprintf("%d saved", len(m.requests)), contentWidth)
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		m.renderSidebarBody(contentWidth, max(4, contentHeight-2)),
	)

	return PanelStyle(m.focusedPanel == PanelSidebar).
		Width(contentWidth).
		Height(contentHeight).
		Render(content)
}

func (m *App) renderSidebarBody(width, height int) string {
	switcher := lipgloss.JoinHorizontal(
		lipgloss.Center,
		RenderMiniTab("Requests", m.sidebarMode == SidebarRequests),
		"  ",
		RenderMiniTab("History", m.sidebarMode == SidebarHistory),
	)

	body := m.requestList.ViewTree()
	if m.sidebarMode == SidebarHistory {
		body = m.renderHistoryList(width)
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(lipgloss.JoinVertical(lipgloss.Left, switcher, "", body))
}

func (m *App) renderHistoryList(width int) string {
	var lines []string
	for _, req := range m.requests {
		entries, err := m.db.GetHistory(req.ID, 1)
		if err != nil || len(entries) == 0 {
			continue
		}

		entry := entries[0]
		name := req.Name
		if name == "" {
			name = req.URL
		}

		titleWidth := max(10, width-18)
		title := lipgloss.JoinHorizontal(
			lipgloss.Center,
			RenderMethodBadge(req.Method),
			" ",
			Style.PlainText.Render(truncateText(name, titleWidth)),
		)
		lines = append(lines, title+" "+RenderStatusBadge(fmt.Sprintf("%d", entry.StatusCode), entry.StatusCode))
		lines = append(lines, "  "+Style.Hint.Render(fmt.Sprintf("%dms  ·  %s", entry.DurationMs, formatTimestamp(entry.Timestamp))))
		lines = append(lines, "")
	}

	if len(lines) == 0 {
		return Style.Hint.Render("No history yet")
	}

	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

func formatTimestamp(ts int64) string {
	if ts == 0 {
		return ""
	}
	return time.Unix(ts, 0).Format("15:04")
}

// methodTextColor returns the lipgloss color for an HTTP method
func methodTextColor(method string) color.Color {
	switch strings.ToUpper(method) {
	case "GET":
		return lipgloss.Color("82")
	case "POST":
		return lipgloss.Color("226")
	case "PUT":
		return lipgloss.Color("39")
	case "PATCH":
		return lipgloss.Color("214")
	case "DELETE":
		return lipgloss.Color("196")
	case "HEAD":
		return lipgloss.Color("93")
	case "OPTIONS":
		return lipgloss.Color("93")
	default:
		return lipgloss.Color("252")
	}
}

func (m *App) renderEditorPane(width, height int) string {
	contentWidth, contentHeight := panelContentSize(width, height)
	header := RenderPanelHeader("Request", fmt.Sprintf("Tab %d/%d", m.activeTab+1, max(1, len(m.tabs))), contentWidth)
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		m.renderEditorBody(contentWidth, max(8, contentHeight-2)),
	)

	return PanelStyle(m.focusedPanel == PanelMain).
		Width(contentWidth).
		Height(contentHeight).
		Render(content)
}

func (m *App) renderEditorBody(width, height int) string {
	if tab := m.ActiveTab(); tab != nil && tab.Builder != nil {
		return lipgloss.NewStyle().
			Width(width).
			Height(height).
			Render(tab.Builder.View().Content)
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(m.renderWelcome())
}

func (m *App) renderResponsePane(width, height int) string {
	contentWidth, contentHeight := panelContentSize(width, height)
	header := RenderPanelHeader("Response", m.responsePaneMeta(), contentWidth)
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		m.renderResponseBody(contentWidth, max(8, contentHeight-2)),
	)

	return PanelStyle(m.focusedPanel == PanelResponse).
		Width(contentWidth).
		Height(contentHeight).
		Render(content)
}

// renderWelcome renders the welcome screen
func (m *App) renderWelcome() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("\n")
	sb.WriteString(Style.WelcomeText.Render("  Welcome to Gurl"))
	sb.WriteString("\n")
	sb.WriteString("\n")
	sb.WriteString(Style.PlainText.Render("  j/k moves through saved requests in the left pane"))
	sb.WriteString("\n")
	sb.WriteString(Style.PlainText.Render("  Enter opens a request, Tab moves sections, Ctrl+Enter sends"))
	sb.WriteString("\n")
	sb.WriteString(Style.PlainText.Render("  Ctrl+1 / Ctrl+2 / Ctrl+3 jump between requests, editor, and response"))
	sb.WriteString("\n")
	sb.WriteString("\n")
	sb.WriteString(Style.Hint.Render("  Ctrl+T new tab  ·  Ctrl+W close tab  ·  Ctrl+K search  ·  ? help"))
	return sb.String()
}

func (m *App) renderResponseBody(width, height int) string {
	if tab := m.ActiveTab(); tab != nil && tab.Viewer != nil {
		return lipgloss.NewStyle().
			Width(width).
			Height(height).
			Render(tab.Viewer.View().Content)
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(Style.Hint.Render("No response yet"))
}

func (m *App) responsePaneMeta() string {
	tab := m.ActiveTab()
	if tab == nil || tab.Viewer == nil {
		return "Body"
	}

	switch tab.Viewer.ActiveTab() {
	case TabHeaders:
		return "Headers"
	case TabCookies:
		return "Cookies"
	case TabTiming:
		return "Timing"
	case TabDiff:
		return "Diff"
	default:
		return "Body"
	}
}

// renderStatusBar renders the bottom status bar
func (m *App) renderStatusBar(layout Layout) string {
	statusBar := NewStatusBar(m)
	return Style.StatusBar.
		Width(layout.Width).
		Height(layout.FooterHeight).
		Render(statusBar.View())
}

// GetRequestCount returns the number of saved requests
func (m *App) GetRequestCount() int {
	return len(m.requests)
}

// GetCurrentEnv returns the current environment name
func (m *App) GetCurrentEnv() string {
	if m.activeEnvName != "" {
		return m.activeEnvName
	}
	if m.envSwitcher != nil {
		if active := m.envSwitcher.GetActiveEnvName(); active != "" {
			return active
		}
	}
	return "default"
}

// ActiveTab returns the currently active tab
func (m *App) ActiveTab() *Tab {
	if m.activeTab < len(m.tabs) {
		return m.tabs[m.activeTab]
	}
	return nil
}

// TabCount returns the number of open tabs
func (m *App) TabCount() int {
	return len(m.tabs)
}

func (m *App) DisplayVersion() string {
	if strings.TrimSpace(m.version) == "" {
		return "dev"
	}
	return m.version
}

func (m *App) StateLabel() string {
	switch {
	case m.envSwitcherOpen:
		return "ENV"
	case m.searchModal != nil:
		return "SEARCH"
	case m.importModal != nil:
		return "IMPORT"
	case m.runnerModal != nil:
		return "RUNNER"
	case m.helpModal:
		return "HELP"
	}

	if tab := m.ActiveTab(); tab != nil && tab.Builder != nil && tab.Builder.IsSending() {
		return "SENDING"
	}

	return "READY"
}
