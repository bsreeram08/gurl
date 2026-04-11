package tui

import (
	"strings"

	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// HelpPanel is a sub-model that displays keyboard shortcuts
// It appears as an overlay when the user presses ?
type HelpPanel struct {
	visible      bool
	focusedPanel Panel // The currently focused panel to show context-sensitive help
	width        int
	height       int
}

// NewHelpPanel creates a new help panel
func NewHelpPanel() *HelpPanel {
	return &HelpPanel{
		visible:      false,
		focusedPanel: PanelSidebar,
		width:        60,
		height:       20,
	}
}

// SetFocusedPanel sets which panel is focused for context-sensitive help
func (h *HelpPanel) SetFocusedPanel(panel Panel) {
	h.focusedPanel = panel
}

// SetSize sets the dimensions of the help panel
func (h *HelpPanel) SetSize(width, height int) {
	h.width = width
	h.height = height
}

// Toggle visibility of the help panel
func (h *HelpPanel) Toggle() {
	h.visible = !h.visible
}

// Show displays the help panel
func (h *HelpPanel) Show() {
	h.visible = true
}

// Hide hides the help panel
func (h *HelpPanel) Hide() {
	h.visible = false
}

// IsVisible returns whether the help panel is visible
func (h *HelpPanel) IsVisible() bool {
	return h.visible
}

// Init implements tea.Model.Init
func (h *HelpPanel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.Update
func (h *HelpPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return h.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		h.SetSize(msg.Width, msg.Height)
	}
	return h, nil
}

// handleKeyPress handles keyboard input for the help panel
func (h *HelpPanel) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "?":
		h.Toggle()
		return h, nil
	case "esc":
		h.Hide()
		return h, nil
	}
	return h, nil
}

// View implements tea.Model.View - renders the help overlay
func (h *HelpPanel) View() tea.View {
	if !h.visible {
		return tea.NewView("")
	}
	return tea.NewView(h.renderHelpOverlay())
}

// renderHelpOverlay renders the help panel as an overlay
func (h *HelpPanel) renderHelpOverlay() string {
	var sb strings.Builder

	// Get context-sensitive shortcuts based on focused panel
	shortcuts := h.getContextShortcuts()

	// Calculate overlay dimensions
	panelWidth := min(h.width-10, 70)
	panelHeight := min(h.height-5, 25)

	if panelWidth < 20 || panelHeight < 5 {
		return styleHelpTitle.Render(" ? Keyboard Shortcuts ") + styleHelpFooter.Render(" (window too small)")
	}

	// Center the panel
	leftPadding := max(0, (h.width-panelWidth)/2)
	topPadding := max(0, (h.height-panelHeight)/2)

	// Draw top border
	sb.WriteString(strings.Repeat("\n", topPadding))
	sb.WriteString(strings.Repeat(" ", leftPadding))
	sb.WriteString(styleHelpBorder.Render("┌" + strings.Repeat("─", panelWidth-2) + "┐"))
	sb.WriteString("\n")

	// Title
	sb.WriteString(strings.Repeat(" ", leftPadding))
	sb.WriteString(styleHelpBorder.Render("│"))
	sb.WriteString(styleHelpTitle.Render(" Keyboard Shortcuts "))
	sb.WriteString(styleHelpBorder.Render("│"))
	sb.WriteString("\n")

	// Separator
	sb.WriteString(strings.Repeat(" ", leftPadding))
	sb.WriteString(styleHelpBorder.Render("├" + strings.Repeat("─", panelWidth-2) + "┤"))
	sb.WriteString("\n")

	// Draw each shortcut group
	for _, group := range shortcuts {
		// Group title
		sb.WriteString(strings.Repeat(" ", leftPadding))
		sb.WriteString(styleHelpBorder.Render("│"))
		sb.WriteString(styleHelpGroupTitle.Render(" " + group.Title + " "))
		sb.WriteString(styleHelpBorder.Render("│"))
		sb.WriteString("\n")

		// Shortcuts in this group
		for _, sc := range group.Keys {
			contentLen := len(sc.Key) + len(sc.Description) + 6
			padding := max(0, panelWidth-contentLen)
			sb.WriteString(strings.Repeat(" ", leftPadding))
			sb.WriteString(styleHelpBorder.Render("│ "))
			sb.WriteString(styleHelpKey.Render(sc.Key))
			sb.WriteString(styleHelpSeparator.Render(" : "))
			sb.WriteString(styleHelpDescription.Render(sc.Description))
			sb.WriteString(strings.Repeat(" ", padding))
			sb.WriteString(styleHelpBorder.Render("│"))
			sb.WriteString("\n")
		}
	}

	// Bottom border
	sb.WriteString(strings.Repeat(" ", leftPadding))
	sb.WriteString(styleHelpBorder.Render("└" + strings.Repeat("─", panelWidth-2) + "┘"))
	sb.WriteString("\n")

	// Footer hint
	sb.WriteString(strings.Repeat(" ", leftPadding))
	sb.WriteString(styleHelpFooter.Render(" Press ? or Esc to close "))
	sb.WriteString("\n")

	return sb.String()
}

// getContextShortcuts returns all keyboard shortcuts for the help overlay.
func (h *HelpPanel) getContextShortcuts() []ShortcutGroup {
	return AllShortcuts()
}

// ShortcutBar returns a short summary of shortcuts for the status bar
// This shows the most important shortcuts for the current context
func (h *HelpPanel) ShortcutBar() string {
	switch h.focusedPanel {
	case PanelSidebar:
		return "Enter:select /:filter ?:help"
	case PanelMain:
		return "Ctrl+Enter:send Ctrl+S:save Ctrl+E:edit ?:help"
	case PanelStatusbar:
		return "e:switch env ?:help"
	default:
		return "Tab:switch ?:help q:quit"
	}
}

// HelpStyles contains styles for the help panel
var styleHelpBorder = lipgloss.NewStyle().
	Foreground(lipgloss.Color("240"))

var styleHelpTitle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("36")).
	Bold(true)

var styleHelpGroupTitle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("228")).
	Bold(true)

var styleHelpKey = lipgloss.NewStyle().
	Foreground(lipgloss.Color("82")).
	Bold(true)

var styleHelpSeparator = lipgloss.NewStyle().
	Foreground(lipgloss.Color("240"))

var styleHelpDescription = lipgloss.NewStyle().
	Foreground(lipgloss.Color("252"))

var styleHelpFooter = lipgloss.NewStyle().
	Foreground(lipgloss.Color("240")).
	Italic(true)
