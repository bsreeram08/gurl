package tui

import (
	"fmt"
	"strings"
)

const gurlVersion = "dev"

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
	var sb strings.Builder

	// Current environment
	env := s.app.GetCurrentEnv()
	sb.WriteString(Style.StatusEnv.Render(fmt.Sprintf(" env: %s ", env)))

	sb.WriteString(" │ ")

	// Request count
	count := s.app.GetRequestCount()
	sb.WriteString(Style.StatusCount.Render(fmt.Sprintf(" %d requests ", count)))

	// Fill remaining space
	sb.WriteString(strings.Repeat(" ", 40))

	// Version on the right
	sb.WriteString(Style.StatusVersion.Render(fmt.Sprintf(" gurl %s ", gurlVersion)))

	return sb.String()
}
