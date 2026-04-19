package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// StatusBar displays environment, request count, and version
type StatusBar struct {
	app *App
}

// NewStatusBar creates a new status bar component
func NewStatusBar(app *App) *StatusBar {
	return &StatusBar{app: app}
}

// View renders the status bar content
func (s *StatusBar) View() string {
	left := strings.Join([]string{
		Style.FooterLabel.Render("env"),
		Style.StatusEnv.Render(s.app.GetCurrentEnv()),
		Style.FooterLabel.Render("·"),
		Style.FooterLabel.Render("req"),
		Style.StatusCount.Render(fmt.Sprintf("%d", s.app.GetRequestCount())),
		Style.FooterLabel.Render("·"),
		Style.FooterLabel.Render("tabs"),
		Style.FooterValue.Render(fmt.Sprintf("%d", s.app.TabCount())),
		Style.FooterLabel.Render("·"),
		Style.FooterLabel.Render("focus"),
		Style.FooterValue.Render(s.focusLabel()),
	}, " ")

	right := strings.Join([]string{
		Style.Hint.Render(s.contextHint()),
		Style.FooterLabel.Render("·"),
		Style.StatusVersion.Render(s.app.DisplayVersion()),
		Style.FooterLabel.Render("·"),
		Style.StatusState.Render(strings.ToLower(s.app.StateLabel())),
	}, " ")

	gap := s.app.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + right
}

func (s *StatusBar) focusLabel() string {
	switch s.app.focusedPanel {
	case PanelSidebar:
		return "requests"
	case PanelMain:
		return "editor"
	case PanelResponse:
		return "response"
	default:
		return "global"
	}
}

func (s *StatusBar) contextHint() string {
	switch {
	case s.app.searchModal != nil || s.app.importModal != nil || s.app.runnerModal != nil || s.app.variablePrompt != nil || s.app.envSwitcherOpen || s.app.helpModal:
		return "enter confirm · esc close"
	case s.app.focusedPanel == PanelSidebar:
		return "j/k move · enter open · h/r toggle"
	case s.app.focusedPanel == PanelMain:
		return "tab section · ctrl+s save · ctrl+enter send"
	case s.app.focusedPanel == PanelResponse:
		return "j/k scroll · b/h/c/t/d view · y copy"
	default:
		return "? help"
	}
}
