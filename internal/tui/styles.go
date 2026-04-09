package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme represents a color theme variant
type Theme int

const (
	// Light theme
	Light Theme = iota
	// Dark theme
	Dark
)

// ColorScheme holds semantic color tokens for a theme
type ColorScheme struct {
	Primary    string // Main interactive elements (cyan)
	Secondary  string // Secondary text (blue)
	Accent     string // Accent highlights (yellow/amber)
	Success    string // Success states (green)
	Warning    string // Warning states (bright yellow)
	Error      string // Error states (red/orange)
	Border     string // Border color
	Text       string // Primary text (light gray)
	TextDim    string // Dimmed text
	Background string // Background
	Surface    string // Surface/panel background
	Selected   string // Selected item highlight
}

// colorSchemes holds light and dark theme color schemes
var colorSchemes = map[Theme]ColorScheme{
	Light: {
		Primary:    "36",  // Cyan
		Secondary:  "34",  // Blue
		Accent:     "220", // Yellow
		Success:    "82",  // Green
		Warning:    "228", // Bright yellow
		Error:      "160", // Red
		Border:     "238", // Dark gray
		Text:       "252", // Light gray
		TextDim:    "240", // Dim gray
		Background: "255", // White
		Surface:    "254", // Slightly darker
		Selected:   "34",  // Blue (same as secondary for selection)
	},
	Dark: {
		Primary:    "36",  // Cyan
		Secondary:  "34",  // Blue
		Accent:     "220", // Yellow
		Success:    "82",  // Green
		Warning:    "228", // Bright yellow
		Error:      "160", // Red
		Border:     "238", // Dark gray
		Text:       "252", // Light gray
		TextDim:    "240", // Dim gray
		Background: "0",   // Black
		Surface:    "0",   // Black (same in dark mode)
		Selected:   "34",  // Blue
	},
}

// currentTheme stores the active theme
var currentTheme = Dark

// Color returns the semantic color for the given token
func Color(token string) lipgloss.Color {
	return lipgloss.Color(colorSchemes[currentTheme].get(token))
}

// get retrieves a color value from the current theme
func (c ColorScheme) get(token string) string {
	switch token {
	case "primary":
		return c.Primary
	case "secondary":
		return c.Secondary
	case "accent":
		return c.Accent
	case "success":
		return c.Success
	case "warning":
		return c.Warning
	case "error":
		return c.Error
	case "border":
		return c.Border
	case "text":
		return c.Text
	case "textDim":
		return c.TextDim
	case "background":
		return c.Background
	case "surface":
		return c.Surface
	case "selected":
		return c.Selected
	default:
		return c.Text
	}
}

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
	Modal         lipgloss.Style
}{
	Sidebar: lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(Color("border")).
		Padding(0, 1),

	Main: lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(Color("border")).
		Padding(0, 1),

	StatusBar: lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(Color("border")).
		Foreground(Color("textDim")),

	Header: lipgloss.NewStyle().
		Foreground(Color("primary")). // Cyan from theme
		Bold(true),

	SelectedItem: lipgloss.NewStyle().
		Foreground(Color("secondary")). // Blue
		Bold(true),

	ListItem: lipgloss.NewStyle().
		Foreground(Color("text")),

	PlainText: lipgloss.NewStyle().
		Foreground(Color("text")),

	WelcomeText: lipgloss.NewStyle().
		Foreground(Color("success")). // Bright green
		Bold(true),

	Hint: lipgloss.NewStyle().
		Foreground(Color("textDim")). // Dim
		Italic(true),

	Method: lipgloss.NewStyle().
		Foreground(Color("accent")). // Yellow
		Bold(true),

	URL: lipgloss.NewStyle().
		Foreground(Color("success")), // Green

	StatusEnv: lipgloss.NewStyle().
		Foreground(Color("warning")), // Bright yellow

	StatusCount: lipgloss.NewStyle().
		Foreground(Color("primary")), // Cyan

	StatusVersion: lipgloss.NewStyle().
		Foreground(Color("textDim")), // Dim

	Modal: lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(Color("primary")).
		Foreground(Color("text")).
		Padding(1, 2),
}

// SetTheme changes the active color theme for all styles
func SetTheme(theme Theme) {
	currentTheme = theme
}
