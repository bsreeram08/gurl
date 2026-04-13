package tui

import (
	"charm.land/bubbletea/v2"
)

// KeyBindings contains common key bindings for consistent keyboard shortcuts.
type KeyBindings struct{}

var QuitKeys = []string{"q", "ctrl+c"}
var HelpKey = "?"
var TabKey = "tab"
var ShiftTabKey = "shift+tab"
var EscapeKey = "esc"

var NavigateUpKeys = []string{"up", "k"}
var NavigateDownKeys = []string{"down", "j"}

var EnterKey = "enter"

var SendRequestKeys = []string{"ctrl+enter"}
var SaveRequestKeys = []string{"ctrl+s"}
var EditRequestKeys = []string{"ctrl+e"}

var NewRequestKey = "n"
var FilterKey = "/"
var EnvSwitchKey = "ctrl+e"

var ResponseTabKeys = map[string]string{
	"t": "time",
	"h": "headers",
	"b": "body",
	"d": "diff",
}

var CollapseKey = "left"
var ExpandKey = "right"

func IsQuitKey(key string) bool {
	for _, k := range QuitKeys {
		if key == k {
			return true
		}
	}
	return false
}

func IsHelpKey(key string) bool {
	return key == HelpKey
}

func IsNavigateUpKey(key string) bool {
	for _, k := range NavigateUpKeys {
		if key == k {
			return true
		}
	}
	return false
}

func IsNavigateDownKey(key string) bool {
	for _, k := range NavigateDownKeys {
		if key == k {
			return true
		}
	}
	return false
}

func IsTabKey(key string) bool {
	return key == TabKey
}

func IsShiftTabKey(key string) bool {
	return key == ShiftTabKey
}

func IsEnterKey(key string) bool {
	return key == EnterKey
}

func IsEscapeKey(key string) bool {
	return key == EscapeKey
}

func IsSendRequestKey(key string) bool {
	for _, k := range SendRequestKeys {
		if key == k {
			return true
		}
	}
	return false
}

func IsSaveRequestKey(key string) bool {
	for _, k := range SaveRequestKeys {
		if key == k {
			return true
		}
	}
	return false
}

func IsEditRequestKey(key string) bool {
	for _, k := range EditRequestKeys {
		if key == k {
			return true
		}
	}
	return false
}

func IsNewRequestKey(key string) bool {
	return key == NewRequestKey
}

func IsFilterKey(key string) bool {
	return key == FilterKey
}

func IsEnvSwitchKey(key string) bool {
	return key == EnvSwitchKey
}

func IsCollapseKey(key string) bool {
	return key == CollapseKey
}

func IsExpandKey(key string) bool {
	return key == ExpandKey
}

func IsResponseTabKey(key string) bool {
	_, ok := ResponseTabKeys[key]
	return ok
}

func GetResponseTab(key string) string {
	return ResponseTabKeys[key]
}

type ShortcutGroup struct {
	Title   string
	Keys    []Shortcut
	Context string
}

type Shortcut struct {
	Key         string
	Description string
}

func GlobalShortcuts() []ShortcutGroup {
	return []ShortcutGroup{
		{
			Title:   "Global",
			Context: "global",
			Keys: []Shortcut{
				{Key: "Ctrl+1 / Ctrl+2 / Ctrl+3", Description: "Focus requests, editor, or response"},
				{Key: "Ctrl+T", Description: "New tab"},
				{Key: "Ctrl+W", Description: "Close tab"},
				{Key: "Ctrl+D", Description: "Duplicate tab"},
				{Key: "Ctrl+Shift+[ / ]", Description: "Cycle open tabs"},
				{Key: "Ctrl+K", Description: "Search requests"},
				{Key: "Ctrl+E", Description: "Environment switcher"},
				{Key: "Ctrl+C", Description: "Quit"},
			},
		},
	}
}

func SidebarShortcuts() []ShortcutGroup {
	return []ShortcutGroup{
		{
			Title:   "Sidebar",
			Context: "sidebar",
			Keys: []Shortcut{
				{Key: "↑/k", Description: "Move up"},
				{Key: "↓/j", Description: "Move down"},
				{Key: "Enter", Description: "Open request"},
				{Key: "h / r", Description: "Toggle history or requests"},
				{Key: "?", Description: "Open help"},
				{Key: "q", Description: "Quit"},
			},
		},
	}
}

func RequestBuilderShortcuts() []ShortcutGroup {
	return []ShortcutGroup{
		{
			Title:   "Request Builder",
			Context: "request_builder",
			Keys: []Shortcut{
				{Key: "Tab / Shift+Tab", Description: "Move between editor sections"},
				{Key: "Ctrl+Enter", Description: "Send request"},
				{Key: "Ctrl+S", Description: "Save request"},
				{Key: "[ / ]", Description: "Cycle auth type in auth view"},
			},
		},
	}
}

func ResponseShortcuts() []ShortcutGroup {
	return []ShortcutGroup{
		{
			Title:   "Response Viewer",
			Context: "response",
			Keys: []Shortcut{
				{Key: "b", Description: "Preview tab"},
				{Key: "h", Description: "Headers tab"},
				{Key: "t", Description: "Timing tab"},
				{Key: "d", Description: "Diff tab"},
				{Key: "y", Description: "Copy response body"},
			},
		},
	}
}

func EnvironmentShortcuts() []ShortcutGroup {
	return []ShortcutGroup{
		{
			Title:   "Environment",
			Context: "environment",
			Keys: []Shortcut{
				{Key: "↑/↓", Description: "Navigate environments"},
				{Key: "Enter", Description: "Activate environment"},
				{Key: "q / Esc", Description: "Close switcher"},
			},
		},
	}
}

func AllShortcuts() []ShortcutGroup {
	all := GlobalShortcuts()
	all = append(all, SidebarShortcuts()...)
	all = append(all, RequestBuilderShortcuts()...)
	all = append(all, ResponseShortcuts()...)
	all = append(all, EnvironmentShortcuts()...)
	return all
}

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

func (kb KeyBindings) HandleKeyMsg(msg tea.KeyMsg) bool {
	key := msg.String()
	switch key {
	case "q", "ctrl+c", "?", "up", "k", "down", "j", "enter",
		"ctrl+enter", "ctrl+s", "ctrl+e", "n", "/",
		"left", "right", "t", "h", "b", "d", "y",
		"ctrl+1", "ctrl+2", "ctrl+3", "tab", "shift+tab":
		return true
	}
	return false
}
