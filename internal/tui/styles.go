package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Style contains all lipgloss styles for the TUI
// Colors are based on the ANSI theme from internal/formatter/theme.go
var Style = struct {
	// Panel styles
	Sidebar   lipgloss.Style
	Main      lipgloss.Style
	StatusBar lipgloss.Style

	// Text styles
	Header       lipgloss.Style
	SelectedItem lipgloss.Style
	ListItem     lipgloss.Style
	PlainText    lipgloss.Style
	WelcomeText  lipgloss.Style
	Hint         lipgloss.Style
	Method       lipgloss.Style
	URL          lipgloss.Style

	// Status bar specific
	StatusEnv     lipgloss.Style
	StatusCount   lipgloss.Style
	StatusVersion lipgloss.Style
}{
	Sidebar: lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(0, 1),

	Main: lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(0, 1),

	StatusBar: lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(lipgloss.Color("238")).
		Foreground(lipgloss.Color("240")),

	Header: lipgloss.NewStyle().
		Foreground(lipgloss.Color("36")). // Cyan from theme
		Bold(true),

	SelectedItem: lipgloss.NewStyle().
		Foreground(lipgloss.Color("34")). // Blue
		Bold(true),

	ListItem: lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")),

	PlainText: lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")),

	WelcomeText: lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")). // Bright green
		Bold(true),

	Hint: lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")). // Dim
		Italic(true),

	Method: lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")). // Yellow
		Bold(true),

	URL: lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")), // Green

	StatusEnv: lipgloss.NewStyle().
		Foreground(lipgloss.Color("228")), // Bright yellow

	StatusCount: lipgloss.NewStyle().
		Foreground(lipgloss.Color("75")), // Cyan

	StatusVersion: lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")), // Dim
}
