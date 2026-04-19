package tui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// Theme represents a color theme variant.
type Theme int

const (
	Light Theme = iota
	Dark
)

// ColorScheme holds semantic color tokens for a theme.
type ColorScheme struct {
	Primary    string
	Secondary  string
	Accent     string
	Success    string
	Warning    string
	Error      string
	Border     string
	Focus      string
	Text       string
	TextDim    string
	Background string
	Surface    string
	SurfaceAlt string
	Selected   string
}

var colorSchemes = map[Theme]ColorScheme{
	Light: {
		Primary:    "31",
		Secondary:  "25",
		Accent:     "214",
		Success:    "34",
		Warning:    "172",
		Error:      "160",
		Border:     "245",
		Focus:      "31",
		Text:       "235",
		TextDim:    "242",
		Background: "255",
		Surface:    "254",
		SurfaceAlt: "252",
		Selected:   "250",
	},
	Dark: {
		Primary:    "51",
		Secondary:  "111",
		Accent:     "221",
		Success:    "120",
		Warning:    "214",
		Error:      "203",
		Border:     "239",
		Focus:      "45",
		Text:       "252",
		TextDim:    "244",
		Background: "235",
		Surface:    "234",
		SurfaceAlt: "236",
		Selected:   "238",
	},
}

var currentTheme = Dark

// Color returns the semantic color for the given token.
func Color(token string) color.Color {
	return lipgloss.Color(colorSchemes[currentTheme].get(token))
}

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
	case "focus":
		return c.Focus
	case "text":
		return c.Text
	case "textDim":
		return c.TextDim
	case "background":
		return c.Background
	case "surface":
		return c.Surface
	case "surfaceAlt":
		return c.SurfaceAlt
	case "selected":
		return c.Selected
	default:
		return c.Text
	}
}

// Style contains all lipgloss styles for the TUI.
var Style = struct {
	Sidebar   lipgloss.Style
	Main      lipgloss.Style
	StatusBar lipgloss.Style

	Header       lipgloss.Style
	PanelTitle   lipgloss.Style
	PanelMeta    lipgloss.Style
	SelectedItem lipgloss.Style
	ListItem     lipgloss.Style
	PlainText    lipgloss.Style
	WelcomeText  lipgloss.Style
	Hint         lipgloss.Style
	Method       lipgloss.Style
	URL          lipgloss.Style

	StatusEnv     lipgloss.Style
	StatusCount   lipgloss.Style
	StatusVersion lipgloss.Style
	StatusState   lipgloss.Style
	FooterLabel   lipgloss.Style
	FooterValue   lipgloss.Style

	Modal           lipgloss.Style
	Card            lipgloss.Style
	CardSelected    lipgloss.Style
	Section         lipgloss.Style
	SectionFocused  lipgloss.Style
	TabActive       lipgloss.Style
	TabInactive     lipgloss.Style
	ButtonPrimary   lipgloss.Style
	ButtonSecondary lipgloss.Style
}{
	Sidebar: lipgloss.NewStyle().
		Background(Color("surface")).
		Foreground(Color("text")),

	Main: lipgloss.NewStyle().
		Background(Color("surface")).
		Foreground(Color("text")),

	StatusBar: lipgloss.NewStyle().
		Background(Color("surface")).
		Foreground(Color("textDim")),

	Header: lipgloss.NewStyle().
		Foreground(Color("primary")).
		Bold(true),

	PanelTitle: lipgloss.NewStyle().
		Foreground(Color("text")).
		Bold(true),

	PanelMeta: lipgloss.NewStyle().
		Foreground(Color("textDim")),

	SelectedItem: lipgloss.NewStyle().
		Foreground(Color("primary")).
		Bold(true),

	ListItem: lipgloss.NewStyle().
		Foreground(Color("text")),

	PlainText: lipgloss.NewStyle().
		Foreground(Color("text")),

	WelcomeText: lipgloss.NewStyle().
		Foreground(Color("success")).
		Bold(true),

	Hint: lipgloss.NewStyle().
		Foreground(Color("textDim")),

	Method: lipgloss.NewStyle().
		Foreground(Color("accent")).
		Bold(true),

	URL: lipgloss.NewStyle().
		Foreground(Color("text")),

	StatusEnv: lipgloss.NewStyle().
		Foreground(Color("primary")).
		Bold(true),

	StatusCount: lipgloss.NewStyle().
		Foreground(Color("secondary")).
		Bold(true),

	StatusVersion: lipgloss.NewStyle().
		Foreground(Color("textDim")),

	StatusState: lipgloss.NewStyle().
		Foreground(Color("success")).
		Bold(true),

	FooterLabel: lipgloss.NewStyle().
		Foreground(Color("textDim")),

	FooterValue: lipgloss.NewStyle().
		Foreground(Color("text")),

	Modal: lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(Color("focus")).
		Foreground(Color("text")).
		Padding(1, 2),

	Card: lipgloss.NewStyle().
		Foreground(Color("text")).
		Padding(0, 0),

	CardSelected: lipgloss.NewStyle().
		Foreground(Color("primary")).
		Bold(true).
		Padding(0, 0),

	Section: lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(Color("border")).
		Padding(0, 1),

	SectionFocused: lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(Color("focus")).
		Padding(0, 1),

	TabActive: lipgloss.NewStyle().
		Foreground(Color("primary")).
		Bold(true),

	TabInactive: lipgloss.NewStyle().
		Foreground(Color("textDim")),

	ButtonPrimary: lipgloss.NewStyle().
		Foreground(Color("primary")).
		Bold(true).
		Padding(0, 0),

	ButtonSecondary: lipgloss.NewStyle().
		Foreground(Color("textDim")).
		Padding(0, 0),
}

// SetTheme changes the active color theme for all styles.
func SetTheme(theme Theme) {
	currentTheme = theme
}

// PanelStyle returns the standard panel frame.
func PanelStyle(focused bool) lipgloss.Style {
	borderColor := Color("border")
	if focused {
		borderColor = Color("focus")
	}

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Foreground(Color("text")).
		Padding(0, 1)
}

// SectionStyle returns the nested content box style used inside panes.
func SectionStyle(focused bool) lipgloss.Style {
	if focused {
		return Style.SectionFocused
	}
	return Style.Section
}

// RenderPanelHeader renders a panel title row with trailing metadata.
func RenderPanelHeader(title, meta string, width int) string {
	left := Style.PanelTitle.Render(strings.ToUpper(title))
	if meta == "" {
		return left
	}

	right := Style.PanelMeta.Render(meta)
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + right
}

// RenderMiniTab renders a compact tab label.
func RenderMiniTab(label string, active bool) string {
	if active {
		return Style.TabActive.Render("[" + strings.ToLower(label) + "]")
	}
	return Style.TabInactive.Render(strings.ToLower(label))
}

// RenderActionButton renders a toolbar button.
func RenderActionButton(label string, primary bool, active bool) string {
	style := Style.ButtonSecondary
	if primary {
		style = Style.ButtonPrimary
	}
	if active {
		style = style.Copy().Underline(true)
	}
	return style.Render("[" + label + "]")
}

// RenderMethodBadge renders a filled HTTP method pill.
func RenderMethodBadge(method string) string {
	fg, _ := methodBadgePalette(method)
	label := strings.ToUpper(method)
	if len(label) < 6 {
		label += strings.Repeat(" ", 6-len(label))
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(fg)).
		Bold(true).
		Render(label)
}

// RenderStatusBadge renders a response status badge.
func RenderStatusBadge(label string, statusCode int) string {
	fg, _ := statusBadgePalette(statusCode)
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(fg)).
		Bold(true).
		Render(label)
}

func methodBadgePalette(method string) (string, string) {
	switch strings.ToUpper(method) {
	case "GET":
		return "16", "45"
	case "POST":
		return "16", "221"
	case "PUT":
		return "16", "214"
	case "PATCH":
		return "16", "177"
	case "DELETE":
		return "16", "203"
	case "HEAD", "OPTIONS":
		return "16", "111"
	default:
		return "16", "250"
	}
}

func statusBadgePalette(statusCode int) (string, string) {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return "16", "120"
	case statusCode >= 300 && statusCode < 400:
		return "16", "214"
	case statusCode >= 400 && statusCode < 500:
		return "16", "221"
	default:
		return "16", "203"
	}
}

func truncateText(value string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= maxWidth {
		return value
	}
	if maxWidth == 1 {
		return "…"
	}

	runes := []rune(value)
	for len(runes) > 0 {
		candidate := string(runes) + "…"
		if lipgloss.Width(candidate) <= maxWidth {
			return candidate
		}
		runes = runes[:len(runes)-1]
	}

	return "…"
}
