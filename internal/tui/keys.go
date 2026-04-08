package tui

import (
	"github.com/charmbracelet/bubbletea"
)

// KeyBindings contains common key bindings for consistent keyboard shortcuts
// Using switch on key.String() as required by the spec
type KeyBindings struct{}

// Global keys
var QuitKeys = []string{"q", "ctrl+c"}
var HelpKey = "?"
var TabKey = "tab"
var ShiftTabKey = "shift+tab"
var EscapeKey = "esc"

// Navigation keys
var NavigateUpKeys = []string{"up", "k"}
var NavigateDownKeys = []string{"down", "j"}

// Enter key for selection
var EnterKey = "enter"

// Request builder keys
var SendRequestKeys = []string{"ctrl+enter"}
var SaveRequestKeys = []string{"ctrl+s"}
var EditRequestKeys = []string{"ctrl+e"}

// New request
var NewRequestKey = "n"

// Filter
var FilterKey = "/"

// Environment switcher
var EnvSwitchKey = "ctrl+e"

// Response viewer keys
var ResponseTabKeys = map[string]string{
	"t": "time",
	"h": "headers",
	"b": "body",
	"c": "copy",
}

// Sidebar specific keys
var CollapseKey = "left"
var ExpandKey = "right"

// IsQuitKey checks if the key should quit the app
func IsQuitKey(key string) bool {
	for _, k := range QuitKeys {
		if key == k {
			return true
		}
	}
	return false
}

// IsHelpKey checks if the key toggles help
func IsHelpKey(key string) bool {
	return key == HelpKey || key == EscapeKey
}

// IsNavigateUpKey checks if the key navigates up
func IsNavigateUpKey(key string) bool {
	for _, k := range NavigateUpKeys {
		if key == k {
			return true
		}
	}
	return false
}

// IsNavigateDownKey checks if the key navigates down
func IsNavigateDownKey(key string) bool {
	for _, k := range NavigateDownKeys {
		if key == k {
			return true
		}
	}
	return false
}

// IsTabKey checks if the key switches panels
func IsTabKey(key string) bool {
	return key == TabKey
}

// IsShiftTabKey checks if the key switches panels in reverse
func IsShiftTabKey(key string) bool {
	return key == ShiftTabKey
}

// IsEnterKey checks if the key selects
func IsEnterKey(key string) bool {
	return key == EnterKey
}

// IsEscapeKey checks if the key is escape
func IsEscapeKey(key string) bool {
	return key == EscapeKey
}

// IsSendRequestKey checks if the key sends a request
func IsSendRequestKey(key string) bool {
	for _, k := range SendRequestKeys {
		if key == k {
			return true
		}
	}
	return false
}

// IsSaveRequestKey checks if the key saves a request
func IsSaveRequestKey(key string) bool {
	for _, k := range SaveRequestKeys {
		if key == k {
			return true
		}
	}
	return false
}

// IsEditRequestKey checks if the key edits a request
func IsEditRequestKey(key string) bool {
	for _, k := range EditRequestKeys {
		if key == k {
			return true
		}
	}
	return false
}

// IsNewRequestKey checks if the key creates a new request
func IsNewRequestKey(key string) bool {
	return key == NewRequestKey
}

// IsFilterKey checks if the key opens filter
func IsFilterKey(key string) bool {
	return key == FilterKey
}

// IsEnvSwitchKey checks if the key switches environment
func IsEnvSwitchKey(key string) bool {
	return key == EnvSwitchKey
}

// IsCollapseKey checks if the key collapses a folder
func IsCollapseKey(key string) bool {
	return key == CollapseKey
}

// IsExpandKey checks if the key expands a folder
func IsExpandKey(key string) bool {
	return key == ExpandKey
}

// IsResponseTabKey checks if the key switches response tabs
func IsResponseTabKey(key string) bool {
	_, ok := ResponseTabKeys[key]
	return ok
}

// GetResponseTab returns the tab name for a key, or empty string if not a response tab key
func GetResponseTab(key string) string {
	return ResponseTabKeys[key]
}

// ShortcutGroup represents a group of shortcuts for help display
type ShortcutGroup struct {
	Title   string
	Keys    []Shortcut
	Context string // "global", "sidebar", "request_builder", "response", "environment"
}

// Shortcut represents a single keyboard shortcut
type Shortcut struct {
	Key         string
	Description string
}

// GlobalShortcuts returns all global shortcuts
func GlobalShortcuts() []ShortcutGroup {
	return []ShortcutGroup{
		{
			Title:   "Global",
			Context: "global",
			Keys: []Shortcut{
				{Key: "?", Description: "Toggle help"},
				{Key: "q", Description: "Quit"},
				{Key: "Tab", Description: "Switch panel"},
				{Key: "Shift+Tab", Description: "Switch panel (reverse)"},
				{Key: "Ctrl+E", Description: "Switch environment"},
				{Key: "Ctrl+Enter", Description: "Send request"},
				{Key: "Ctrl+S", Description: "Save request"},
				{Key: "n", Description: "New request"},
			},
		},
	}
}

// SidebarShortcuts returns shortcuts for sidebar navigation
func SidebarShortcuts() []ShortcutGroup {
	return []ShortcutGroup{
		{
			Title:   "Sidebar",
			Context: "sidebar",
			Keys: []Shortcut{
				{Key: "↑/k", Description: "Move up"},
				{Key: "↓/j", Description: "Move down"},
				{Key: "Enter", Description: "Select request"},
				{Key: "/", Description: "Filter requests"},
				{Key: "Esc", Description: "Clear filter"},
				{Key: "←", Description: "Collapse folder"},
				{Key: "→", Description: "Expand folder"},
			},
		},
	}
}

// RequestBuilderShortcuts returns shortcuts for request builder
func RequestBuilderShortcuts() []ShortcutGroup {
	return []ShortcutGroup{
		{
			Title:   "Request Builder",
			Context: "request_builder",
			Keys: []Shortcut{
				{Key: "Ctrl+Enter", Description: "Send request"},
				{Key: "Ctrl+S", Description: "Save request"},
				{Key: "Ctrl+E", Description: "Edit request"},
			},
		},
	}
}

// ResponseShortcuts returns shortcuts for response viewer
func ResponseShortcuts() []ShortcutGroup {
	return []ShortcutGroup{
		{
			Title:   "Response Viewer",
			Context: "response",
			Keys: []Shortcut{
				{Key: "t", Description: "Time tab"},
				{Key: "h", Description: "Headers tab"},
				{Key: "b", Description: "Body tab"},
				{Key: "c", Description: "Copy to clipboard"},
			},
		},
	}
}

// EnvironmentShortcuts returns shortcuts for environment switcher
func EnvironmentShortcuts() []ShortcutGroup {
	return []ShortcutGroup{
		{
			Title:   "Environment",
			Context: "environment",
			Keys: []Shortcut{
				{Key: "e", Description: "Switch environment"},
			},
		},
	}
}

// AllShortcuts returns all shortcut groups
func AllShortcuts() []ShortcutGroup {
	all := GlobalShortcuts()
	all = append(all, SidebarShortcuts()...)
	all = append(all, RequestBuilderShortcuts()...)
	all = append(all, ResponseShortcuts()...)
	all = append(all, EnvironmentShortcuts()...)
	return all
}

// ShortcutsForContext returns shortcuts for a specific context
func ShortcutsForContext(context string) []ShortcutGroup {
	switch context {
	case "sidebar":
		return append(GlobalShortcuts(), SidebarShortcuts()...)
	case "request_builder":
		return append(GlobalShortcuts(), RequestBuilderShortcuts()...)
	case "response":
		return append(GlobalShortcuts(), ResponseShortcuts()...)
	case "environment":
		return append(GlobalShortcuts(), EnvironmentShortcuts()...)
	default:
		return AllShortcuts()
	}
}

// HandleKeyMsg handles a key message and returns whether it was handled
func (kb KeyBindings) HandleKeyMsg(msg tea.KeyMsg) bool {
	key := msg.String()
	switch key {
	case "q", "ctrl+c", "?", "escape", "tab", "shift+tab",
		"up", "k", "down", "j", "enter",
		"ctrl+enter", "ctrl+s", "ctrl+e", "n", "/",
		"left", "right", "t", "h", "b", "c":
		return true
	}
	return false
}
