package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

// Panel represents the focused panel in the TUI
type Panel int

const (
	PanelSidebar Panel = iota
	PanelMain
	PanelStatusbar
)

// App is the root bubbletea model
type App struct {
	db              storage.DB
	config          *types.Config
	requests        []*types.SavedRequest
	selectedRequest *types.SavedRequest
	focusedPanel    Panel
	width           int
	height          int
	ready           bool
	quitting        bool
}

// NewApp creates a new TUI application
func NewApp(db storage.DB, config *types.Config) *App {
	return &App{
		db:           db,
		config:       config,
		focusedPanel: PanelSidebar,
		width:        80, // default terminal size
		height:       24,
	}
}

// Init implements tea.Model.Init
func (m *App) Init() tea.Cmd {
	// Load initial data - request list
	requests, err := m.db.ListRequests(nil)
	if err != nil {
		// Log error but continue - we can show welcome screen
		requests = []*types.SavedRequest{}
	}
	m.requests = requests
	return nil
}

// Update implements tea.Model.Update
func (m *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}

	return m, nil
}

// handleKeyPress handles keyboard input using switch on key.String()
func (m *App) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "tab":
		// Cycle focus: sidebar -> main -> statusbar -> sidebar
		switch m.focusedPanel {
		case PanelSidebar:
			m.focusedPanel = PanelMain
		case PanelMain:
			m.focusedPanel = PanelStatusbar
		case PanelStatusbar:
			m.focusedPanel = PanelSidebar
		}
		return m, nil

	case "shift+tab":
		// Reverse cycle
		switch m.focusedPanel {
		case PanelSidebar:
			m.focusedPanel = PanelStatusbar
		case PanelMain:
			m.focusedPanel = PanelSidebar
		case PanelStatusbar:
			m.focusedPanel = PanelMain
		}
		return m, nil

	case "up", "k":
		// Navigate up in current panel
		return m.handleNavigateUp()

	case "down", "j":
		// Navigate down in current panel
		return m.handleNavigateDown()
	}

	return m, nil
}

// handleNavigateUp handles upward navigation
func (m *App) handleNavigateUp() (tea.Model, tea.Cmd) {
	return m, nil
}

// handleNavigateDown handles downward navigation
func (m *App) handleNavigateDown() (tea.Model, tea.Cmd) {
	return m, nil
}

// View implements tea.Model.View - renders the 3-panel layout
func (m *App) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Calculate layout based on current dimensions
	layout := CalculateLayout(m.width, m.height)

	// Build each panel
	sidebar := m.renderSidebar(layout)
	main := m.renderMain(layout)
	status := m.renderStatusBar(layout)

	// Join sidebar and main horizontally
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebar,
		main,
	)

	// Join with status bar at bottom
	fullView := lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		status,
	)

	return fullView
}

// renderSidebar renders the request list panel
func (m *App) renderSidebar(layout Layout) string {
	var sb strings.Builder

	// Sidebar header
	sb.WriteString(Style.Header.Render("Requests"))

	// List requests
	if len(m.requests) == 0 {
		sb.WriteString("\n")
		sb.WriteString(Style.PlainText.Render("  No requests saved"))
	} else {
		for i, req := range m.requests {
			sb.WriteString("\n")
			if i == 0 && m.focusedPanel == PanelSidebar {
				sb.WriteString(Style.SelectedItem.Render(fmt.Sprintf("▶ %s", req.Name)))
			} else {
				sb.WriteString(Style.ListItem.Render(fmt.Sprintf("  %s", req.Name)))
			}
		}
	}

	// Apply border and padding
	sidebarContent := sb.String()
	return Style.Sidebar.
		Width(layout.SidebarWidth).
		Height(layout.MainHeight()).
		Render(sidebarContent)
}

// renderMain renders the main request/response panel
func (m *App) renderMain(layout Layout) string {
	var sb strings.Builder

	// Main header
	sb.WriteString(Style.Header.Render("Request / Response"))

	if m.selectedRequest == nil {
		sb.WriteString("\n")
		sb.WriteString("\n")
		sb.WriteString(Style.WelcomeText.Render("  Welcome to Gurl TUI!"))
		sb.WriteString("\n")
		sb.WriteString("\n")
		sb.WriteString(Style.PlainText.Render("  Use ↑/↓ to navigate requests"))
		sb.WriteString("\n")
		sb.WriteString(Style.PlainText.Render("  Press Enter to select"))
		sb.WriteString("\n")
		sb.WriteString("\n")
		sb.WriteString(Style.Hint.Render("  Press Tab to switch panels"))
	} else {
		sb.WriteString("\n")
		sb.WriteString("\n")
		sb.WriteString(Style.Method.Render(fmt.Sprintf("  %s", m.selectedRequest.Method)))
		sb.WriteString(" ")
		sb.WriteString(Style.URL.Render(m.selectedRequest.URL))
	}

	mainContent := sb.String()
	return Style.Main.
		Width(layout.MainWidth).
		Height(layout.MainHeight()).
		Render(mainContent)
}

// renderStatusBar renders the bottom status bar
func (m *App) renderStatusBar(layout Layout) string {
	statusBar := NewStatusBar(m)
	return Style.StatusBar.
		Width(m.width).
		Height(layout.StatusHeight).
		Render(statusBar.View())
}

// GetRequestCount returns the number of requests
func (m *App) GetRequestCount() int {
	return len(m.requests)
}

// GetCurrentEnv returns the current environment name
func (m *App) GetCurrentEnv() string {
	return "default"
}
