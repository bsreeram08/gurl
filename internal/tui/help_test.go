package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbletea"
)

func TestHelp_Toggle(t *testing.T) {
	help := NewHelpPanel()

	// Initially hidden
	if help.IsVisible() {
		t.Error("Help panel should be initially hidden")
	}

	// Toggle with ?
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("?"),
	}
	help.Update(msg)

	if !help.IsVisible() {
		t.Error("Help panel should be visible after pressing ?")
	}

	// Toggle off with ?
	help.Update(msg)

	if help.IsVisible() {
		t.Error("Help panel should be hidden after pressing ? again")
	}
}

func TestHelp_ShowAllShortcuts(t *testing.T) {
	help := NewHelpPanel()
	help.SetSize(100, 40)
	help.Show()

	view := help.View()

	// Should contain all shortcut group titles
	groups := []string{"Global", "Sidebar", "Request Builder", "Response Viewer", "Environment"}
	for _, group := range groups {
		if !strings.Contains(view, group) {
			t.Errorf("Help view should contain group title: %s", group)
		}
	}

	// Should contain specific shortcuts
	shortcuts := []string{"?", "q", "Tab", "Ctrl+Enter", "Ctrl+S", "n", "/"}
	for _, shortcut := range shortcuts {
		if !strings.Contains(view, shortcut) {
			t.Errorf("Help view should contain shortcut: %s", shortcut)
		}
	}
}

func TestHelp_ContextSensitive(t *testing.T) {
	help := NewHelpPanel()
	help.SetSize(100, 40)

	// Test sidebar context
	help.SetFocusedPanel(PanelSidebar)
	help.Show()
	sidebarView := help.View()

	// Sidebar should show navigation shortcuts
	if !strings.Contains(sidebarView, "↑/k") || !strings.Contains(sidebarView, "↓/j") {
		t.Error("Sidebar context should show navigation shortcuts")
	}

	// Test main/request builder context
	help.SetFocusedPanel(PanelMain)
	help.Show()
	mainView := help.View()

	// Main should show Ctrl+Enter for sending
	if !strings.Contains(mainView, "Ctrl+Enter") {
		t.Error("Main context should show send request shortcut")
	}

	// Test statusbar/environment context
	help.SetFocusedPanel(PanelStatusbar)
	help.Show()
	statusView := help.View()

	// Statusbar should show environment switch
	if !strings.Contains(statusView, "e") && !strings.Contains(statusView, "Switch environment") {
		t.Error("Statusbar context should show environment switch")
	}
}

func TestHelp_ShortcutBar(t *testing.T) {
	help := NewHelpPanel()

	// Test sidebar context shortcut bar
	help.SetFocusedPanel(PanelSidebar)
	bar := help.ShortcutBar()
	if !strings.Contains(bar, "select") && !strings.Contains(bar, "filter") {
		t.Errorf("Sidebar shortcut bar should contain select/filter hints, got: %s", bar)
	}

	// Test main context shortcut bar
	help.SetFocusedPanel(PanelMain)
	bar = help.ShortcutBar()
	if !strings.Contains(bar, "send") || !strings.Contains(bar, "save") {
		t.Errorf("Main shortcut bar should contain send/save hints, got: %s", bar)
	}

	// Test statusbar context shortcut bar
	help.SetFocusedPanel(PanelStatusbar)
	bar = help.ShortcutBar()
	if !strings.Contains(bar, "env") {
		t.Errorf("Statusbar shortcut bar should contain env hint, got: %s", bar)
	}
}

func TestHelp_Close(t *testing.T) {
	help := NewHelpPanel()
	help.Show()

	if !help.IsVisible() {
		t.Error("Help panel should be visible before close test")
	}

	// Close with Escape
	escMsg := tea.KeyMsg{
		Type: tea.KeyEscape,
	}
	help.Update(escMsg)

	if help.IsVisible() {
		t.Error("Help panel should be hidden after pressing Escape")
	}

	// Reopen and close with ?
	help.Show()
	helpMsg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("?"),
	}
	help.Update(helpMsg)

	if help.IsVisible() {
		t.Error("Help panel should be hidden after pressing ? (toggle off)")
	}
}

func TestHelp_ModelInterface(t *testing.T) {
	help := NewHelpPanel()

	// HelpPanel should implement tea.Model
	var _ tea.Model = help
}

func TestHelp_SetSize(t *testing.T) {
	help := NewHelpPanel()

	// Set custom size
	help.SetSize(120, 50)

	// Size should be stored (we can verify through View rendering)
	// If dimensions are too small, the panel should handle gracefully
	help.SetSize(30, 10)
	help.Show()
	view := help.View()
	if view == "" {
		t.Error("Help view should not be empty even with small dimensions")
	}
}
