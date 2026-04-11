package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
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
	PanelStatusbar
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
	case tea.KeyMsg:
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

func (sm *SearchModal) View() string {
	lines := []string{
		Style.Header.Render(" Search "),
		sm.input.View(),
		"",
	}

	maxResults := sm.height - 10
	if maxResults < 5 {
		maxResults = 5
	}
	if len(sm.results) > maxResults {
		sm.results = sm.results[:maxResults]
	}

	for i, req := range sm.results {
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

	if len(sm.results) == 0 {
		lines = append(lines, Style.PlainText.Render("  No matching requests"))
	}

	lines = append(lines, "")
	lines = append(lines, Style.Hint.Render("  ↑↓ navigate  ·  Enter open  ·  Esc close"))

	content := strings.Join(lines, "\n")
	modalWidth := sm.width - 20
	if modalWidth < 40 {
		modalWidth = 40
	}
	return Style.Modal.
		Width(modalWidth).
		Height(sm.height - 5).
		Render(content)
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
	case tea.KeyMsg:
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

func (im *ImportModal) View() string {
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
	return Style.Modal.
		Width(modalWidth).
		Height(im.height - 5).
		Render(content)
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
	case tea.KeyMsg:
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

func (rm *RunnerModal) View() string {
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
	return Style.Modal.
		Width(modalWidth).
		Height(rm.height - 5).
		Render(content)
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

	requestList *RequestList
	searchModal *SearchModal
	importModal *ImportModal
	runnerModal *RunnerModal
	helpModal   bool
}

// NewApp creates a new TUI application
func NewApp(db storage.DB, config *types.Config) *App {
	initialTab := newBlankTab(db)
	rl := NewRequestList(db)
	return &App{
		db:           db,
		config:       config,
		focusedPanel: PanelSidebar,
		width:        80,
		height:       24,
		tabs:         []*Tab{initialTab},
		activeTab:    0,
		sidebarMode:  SidebarRequests,
		requestList:  rl,
	}
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
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.requestList.SetSize(msg.Width, msg.Height)
		// Always update all components regardless of active tab
		for _, tab := range m.tabs {
			if tab.Builder != nil {
				tab.Builder.Update(msg)
			}
			if tab.Viewer != nil {
				tab.Viewer.Update(msg)
			}
		}
		return m, nil

	case requestsLoadedMsg:
		m.requests = msg.requests
		m.requestList.SetRequests(msg.requests)
		return m, nil

	case tea.KeyMsg:
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
		if m.helpModal {
			if msg.Type == tea.KeyEsc || msg.String() == "q" || msg.String() == "?" {
				m.helpModal = false
			}
			return m, nil
		}
		return m.handleKeyPress(msg)

	case BuilderRequestSelectedMsg:
		m.openInNewTab(msg.Request)
		return m, nil

	case RequestSentMsg:
		if m.activeTab < len(m.tabs) {
			m.tabs[m.activeTab].Viewer.SetResponse(msg.Response)
		}
		return m, nil

	case RequestSavedMsg:
		requests, _ := m.db.ListRequests(nil)
		if requests == nil {
			requests = []*types.SavedRequest{}
		}
		m.requests = requests
		if msg.Request != nil && m.activeTab < len(m.tabs) {
			m.tabs[m.activeTab].Name = msg.Request.Name
			m.tabs[m.activeTab].Request = msg.Request
		}
		return m, nil
	}

	// Route to active tab's components
	if m.activeTab < len(m.tabs) {
		tab := m.tabs[m.activeTab]
		if tab.Builder != nil {
			updatedBuilder, cmd := tab.Builder.Update(msg)
			if updatedBuilder != nil {
				switch b := updatedBuilder.(type) {
				case *RequestBuilder:
					tab.Builder = b
				}
			}
			for _, subMsg := range tab.Builder.GetMessages() {
				if result, _ := m.Update(subMsg); result != nil {
					if app, ok := result.(*App); ok {
						m = app
					}
				}
			}
			if cmd != nil {
				return m, cmd
			}
		}
		if tab.Viewer != nil {
			updatedViewer, cmd := tab.Viewer.Update(msg)
			if updatedViewer != nil {
				switch v := updatedViewer.(type) {
				case *ResponseViewer:
					tab.Viewer = v
				}
			}
			if cmd != nil {
				return m, cmd
			}
		}
	}

	return m, nil
}

// handleKeyPress handles all keyboard input
func (m *App) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case IsQuitKey(key):
		m.quitting = true
		return m, tea.Quit

	case key == "ctrl+t":
		m.openNewTab()
		return m, nil

	case key == "ctrl+w":
		m.closeActiveTab()
		return m, nil

	case key == "ctrl+d":
		if m.activeTab < len(m.tabs) && m.tabs[m.activeTab].Request != nil {
			m.duplicateActiveTab()
		}
		return m, nil

	case key == "ctrl+shift+]":
		m.nextTab()
		return m, nil

	case key == "ctrl+shift+[":
		m.prevTab()
		return m, nil

	case key == "ctrl+k":
		m.searchModal = NewSearchModal(m.requests, m.width, m.height)
		return m, nil

	case key == "ctrl+i":
		m.importModal = NewImportModal(m.width, m.height)
		return m, nil

	case key == "ctrl+r":
		collections := []string{}
		seen := make(map[string]bool)
		for _, req := range m.requests {
			if req.Collection != "" && !seen[req.Collection] {
				seen[req.Collection] = true
				collections = append(collections, req.Collection)
			}
		}
		m.runnerModal = NewRunnerModal(collections, []string{}, m.width, m.height)
		return m, nil

	case IsHelpKey(key):
		m.helpModal = !m.helpModal
		return m, nil

	case key == "ctrl+e":
		// TODO: Open env switcher modal
		return m, nil

	case key == "h":
		m.sidebarMode = SidebarHistory
		return m, nil

	case key == "r":
		m.sidebarMode = SidebarRequests
		return m, nil

	case IsTabKey(key):
		switch m.focusedPanel {
		case PanelSidebar:
			m.focusedPanel = PanelMain
		case PanelMain:
			m.focusedPanel = PanelStatusbar
		case PanelStatusbar:
			m.focusedPanel = PanelSidebar
		}
		return m, nil

	case IsShiftTabKey(key):
		switch m.focusedPanel {
		case PanelSidebar:
			m.focusedPanel = PanelStatusbar
		case PanelMain:
			m.focusedPanel = PanelSidebar
		case PanelStatusbar:
			m.focusedPanel = PanelMain
		}
		return m, nil

	case IsNavigateUpKey(key):
		return m.handleNavigateUp()

	case IsNavigateDownKey(key):
		return m.handleNavigateDown()

	case IsEnterKey(key):
		return m.handleEnter()

	case key == " ":
		return m.handleSpace()
	}

	return m, nil
}

// openNewTab opens a new blank tab
func (m *App) openNewTab() {
	tab := newBlankTab(m.db)
	m.tabs = append(m.tabs, tab)
	m.activeTab = len(m.tabs) - 1
}

// openInNewTab opens a request in a new tab
func (m *App) openInNewTab(req *types.SavedRequest) {
	tab := newTabForRequest(m.db, req)
	m.tabs = append(m.tabs, tab)
	m.activeTab = len(m.tabs) - 1
}

// closeActiveTab closes the current tab; keeps at least one tab open
func (m *App) closeActiveTab() {
	if len(m.tabs) <= 1 {
		m.tabs[0] = newBlankTab(m.db)
		m.activeTab = 0
		return
	}
	m.tabs = append(m.tabs[:m.activeTab], m.tabs[m.activeTab+1:]...)
	if m.activeTab >= len(m.tabs) {
		m.activeTab = len(m.tabs) - 1
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

func (m *App) handleNavigateUp() (tea.Model, tea.Cmd) {
	if m.focusedPanel == PanelSidebar {
		if _, cmd := m.requestList.Update(tea.KeyMsg{Type: tea.KeyUp}); cmd != nil {
			return m, cmd
		}
	}
	return m, nil
}

func (m *App) handleNavigateDown() (tea.Model, tea.Cmd) {
	if m.focusedPanel == PanelSidebar {
		if _, cmd := m.requestList.Update(tea.KeyMsg{Type: tea.KeyDown}); cmd != nil {
			return m, cmd
		}
	}
	return m, nil
}

func (m *App) handleEnter() (tea.Model, tea.Cmd) {
	if m.focusedPanel == PanelSidebar && m.sidebarMode == SidebarRequests {
		if req := m.requestList.GetSelectedRequest(); req != nil {
			m.openInNewTab(req)
		}
	}
	return m, nil
}

func (m *App) renderHelpOverlay() string {
	var sb strings.Builder
	sb.WriteString(Style.Header.Render(" Keyboard Shortcuts "))
	sb.WriteString("\n\n")

	shortcuts := []struct {
		key  string
		desc string
	}{
		{"Ctrl+T", "New tab"},
		{"Ctrl+W", "Close tab"},
		{"Ctrl+D", "Duplicate tab"},
		{"Ctrl+Tab", "Next tab"},
		{"Ctrl+Shift+[", "Previous tab"},
		{"Ctrl+K", "Global search"},
		{"?", "Toggle this help"},
		{"h", "History sidebar"},
		{"r", "Requests sidebar"},
		{"Tab / Shift+Tab", "Cycle focus"},
		{"↑ / ↓ or j/k", "Navigate list"},
		{"Space", "Toggle folder expand"},
		{"Enter", "Open request in new tab"},
		{"q / Ctrl+C", "Quit"},
	}

	for _, s := range shortcuts {
		keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
		line := fmt.Sprintf("  %s  %s", keyStyle.Render(s.key), s.desc)
		sb.WriteString(Style.ListItem.Render(line))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(Style.Hint.Render("  Press ? or q to close"))

	content := sb.String()
	modalWidth := 45
	if modalWidth > m.width-10 {
		modalWidth = m.width - 10
	}
	return lipgloss.NewStyle().
		Width(modalWidth).
		Height(m.height-5).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Render(content)
}

func (m *App) handleSpace() (tea.Model, tea.Cmd) {
	if m.focusedPanel == PanelSidebar && m.sidebarMode == SidebarRequests {
		m.requestList.ToggleFolderAtCursor()
	}
	return m, nil
}

func (m *App) View() string {
	if !m.ready {
		return "Loading..."
	}

	if m.searchModal != nil {
		overlay := m.searchModal.View()
		bg := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Background(lipgloss.Color("0"))
		padding := (m.width - 60) / 2
		if padding < 5 {
			padding = 5
		}
		return lipgloss.JoinVertical(
			lipgloss.Top,
			bg.Render(strings.Repeat("\n", 5)),
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				strings.Repeat(" ", padding),
				lipgloss.NewStyle().Width(60).Render(overlay),
			),
		)
	}

	if m.helpModal {
		return m.renderHelpOverlay()
	}

	if m.importModal != nil {
		overlay := m.importModal.View()
		bg := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Background(lipgloss.Color("0"))
		padding := (m.width - 60) / 2
		if padding < 5 {
			padding = 5
		}
		return lipgloss.JoinVertical(
			lipgloss.Top,
			bg.Render(strings.Repeat("\n", 5)),
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				strings.Repeat(" ", padding),
				lipgloss.NewStyle().Width(60).Render(overlay),
			),
		)
	}

	if m.runnerModal != nil {
		overlay := m.runnerModal.View()
		bg := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Background(lipgloss.Color("0"))
		padding := (m.width - 60) / 2
		if padding < 5 {
			padding = 5
		}
		return lipgloss.JoinVertical(
			lipgloss.Top,
			bg.Render(strings.Repeat("\n", 5)),
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				strings.Repeat(" ", padding),
				lipgloss.NewStyle().Width(60).Render(overlay),
			),
		)
	}

	layout := CalculateLayout(m.width, m.height)
	sidebar := m.renderSidebar(layout)
	main := m.renderMain(layout)
	status := m.renderStatusBar(layout)

	content := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
	return lipgloss.JoinVertical(lipgloss.Left, content, status)
}

// renderSidebar renders the sidebar (requests list or history)
func (m *App) renderSidebar(layout Layout) string {
	var sb strings.Builder

	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	if m.sidebarMode == SidebarRequests {
		sb.WriteString(activeStyle.Render(" Requests "))
		sb.WriteString(inactiveStyle.Render(" History "))
	} else {
		sb.WriteString(inactiveStyle.Render(" Requests "))
		sb.WriteString(activeStyle.Render(" History "))
	}
	sb.WriteString("\n")

	if m.sidebarMode == SidebarHistory {
		m.renderHistoryList(&sb)
	} else {
		sb.WriteString(m.requestList.ViewTree())
	}

	sidebarContent := sb.String()
	return Style.Sidebar.
		Width(layout.SidebarWidth).
		Height(layout.MainHeight(m.height)).
		Render(sidebarContent)
}

func (m *App) renderHistoryList(sb *strings.Builder) {
	hasHistory := false
	for _, req := range m.requests {
		entries, err := m.db.GetHistory(req.ID, 1)
		if err != nil || len(entries) == 0 {
			continue
		}
		entry := entries[0]
		hasHistory = true
		prefix := "  "
		itemStyle := Style.ListItem
		methodColor := methodTextColor(req.Method)
		methodStyle := lipgloss.NewStyle().Foreground(methodColor).Bold(true)
		name := req.Name
		if name == "" {
			name = req.URL
		}
		if len(name) > 28 {
			name = name[:25] + "..."
		}
		statusColor := lipgloss.Color("82")
		if entry.StatusCode >= 400 {
			statusColor = lipgloss.Color("196")
		} else if entry.StatusCode >= 300 {
			statusColor = lipgloss.Color("226")
		}
		statusStyle := lipgloss.NewStyle().Foreground(statusColor)
		durationStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		line := fmt.Sprintf("%s%s %s [%s %dms] %s",
			prefix, methodStyle.Render(req.Method), name, statusStyle.Render(fmt.Sprintf("%d", entry.StatusCode)), entry.DurationMs, durationStyle.Render("@"+formatTimestamp(entry.Timestamp)))
		sb.WriteString(itemStyle.Render(line))
		sb.WriteString("\n")
	}
	if !hasHistory {
		sb.WriteString(Style.PlainText.Render("\n  No history yet"))
	}
}

func formatTimestamp(ts int64) string {
	if ts == 0 {
		return ""
	}
	return time.Unix(ts, 0).Format("15:04")
}

// methodTextColor returns the lipgloss color for an HTTP method
func methodTextColor(method string) lipgloss.Color {
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

func (m *App) renderMain(layout Layout) string {
	var sb strings.Builder

	sb.WriteString(m.renderTabBar())
	sb.WriteString(m.renderToolbar())

	if m.activeTab < len(m.tabs) {
		tab := m.tabs[m.activeTab]
		if tab.Builder != nil {
			sb.WriteString(tab.Builder.View())
		}
	} else {
		sb.WriteString(m.renderWelcome())
	}

	mainContent := sb.String()
	return Style.Main.
		Width(layout.MainWidth).
		Height(m.height).
		Render(mainContent)
}

func (m *App) renderToolbar() string {
	var sb strings.Builder

	toolbarBg := lipgloss.NewStyle().
		Background(lipgloss.Color("238")).
		Foreground(lipgloss.Color("252"))

	var method, url string
	if m.activeTab < len(m.tabs) {
		tab := m.tabs[m.activeTab]
		if tab.Request != nil {
			method = tab.Request.Method
			url = tab.Request.URL
		}
	}

	saveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39"))
	dupStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("226"))
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	if method != "" {
		methodColor := methodTextColor(method)
		methodStyle := lipgloss.NewStyle().Foreground(methodColor).Bold(true)
		displayURL := url
		if len(displayURL) > 50 {
			displayURL = displayURL[:47] + "..."
		}
		toolbarLine := fmt.Sprintf("  %s  %s    %s  %s  %s",
			methodStyle.Render(method),
			saveStyle.Render("[Ctrl+S save]"),
			dupStyle.Render("[Ctrl+D dup]"),
			hintStyle.Render("[Ctrl+Enter send]"),
			Style.Hint.Render(displayURL),
		)
		sb.WriteString(toolbarBg.Render(toolbarLine))
	} else {
		sb.WriteString(toolbarBg.Render("  New Request    " + hintStyle.Render("[Ctrl+Enter send]")))
	}

	sb.WriteString("\n")
	return sb.String()
}

// renderTabBar renders the tab bar
func (m *App) renderTabBar() string {
	var sb strings.Builder

	bgStyle := lipgloss.NewStyle().Background(lipgloss.Color("236"))

	for i, tab := range m.tabs {
		name := tab.Name
		if len(name) > 18 {
			name = name[:15] + "..."
		}
		if i == m.activeTab {
			activeStyle := lipgloss.NewStyle().
				Background(lipgloss.Color("39")).
				Foreground(lipgloss.Color("15")).
				Bold(true).
				Padding(0, 1)
			sb.WriteString(activeStyle.Render(" " + name + " "))
		} else {
			inactiveStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Padding(0, 1)
			sb.WriteString(inactiveStyle.Render(" " + name + " "))
		}
	}

	newTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Padding(0, 1)
	sb.WriteString(newTabStyle.Render("[+]"))

	sb.WriteString("\n")
	return bgStyle.Render(sb.String())
}

// renderWelcome renders the welcome screen
func (m *App) renderWelcome() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("\n")
	sb.WriteString(Style.WelcomeText.Render("  Welcome to Gurl TUI!"))
	sb.WriteString("\n")
	sb.WriteString("\n")
	sb.WriteString(Style.PlainText.Render("  Use ↑/↓ to navigate requests"))
	sb.WriteString("\n")
	sb.WriteString(Style.PlainText.Render("  Press Enter to open in new tab"))
	sb.WriteString("\n")
	sb.WriteString(Style.PlainText.Render("  Press Tab to switch panels"))
	sb.WriteString("\n")
	sb.WriteString("\n")
	sb.WriteString(Style.Hint.Render("  Ctrl+T: new tab  |  Ctrl+W: close tab  |  Ctrl+D: duplicate"))
	return sb.String()
}

// renderStatusBar renders the bottom status bar
func (m *App) renderStatusBar(layout Layout) string {
	statusBar := NewStatusBar(m)
	return Style.StatusBar.
		Width(m.width).
		Height(layout.StatusHeight).
		Render(statusBar.View())
}

// GetRequestCount returns the number of saved requests
func (m *App) GetRequestCount() int {
	return len(m.requests)
}

// GetCurrentEnv returns the current environment name
func (m *App) GetCurrentEnv() string {
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
