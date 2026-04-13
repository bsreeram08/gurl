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
		Style.FooterLabel.Render("ENV:"),
		Style.StatusEnv.Render(s.app.GetCurrentEnv()),
		Style.FooterLabel.Render("| Requests:"),
		Style.StatusCount.Render(fmt.Sprintf("%d", s.app.GetRequestCount())),
		Style.FooterLabel.Render("| Tabs:"),
		Style.FooterValue.Render(fmt.Sprintf("%d", s.app.TabCount())),
	}, " ")

	right := strings.Join([]string{
		Style.StatusVersion.Render(s.app.DisplayVersion()),
		Style.FooterLabel.Render("|"),
		Style.StatusState.Render(s.app.StateLabel()),
	}, " ")

	gap := s.app.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + right
}
