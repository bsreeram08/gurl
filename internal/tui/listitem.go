package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sreeram/gurl/pkg/types"
)

// RequestItem represents a request in the list
type RequestItem struct {
	*types.SavedRequest
	FolderPath string
}

// FilterValue implements list.Item for filtering
func (i RequestItem) FilterValue() string {
	return i.Name
}

// Title returns the display title for the item
func (i RequestItem) Title() string {
	return i.Name
}

// Description returns the folder path as description
func (i RequestItem) Description() string {
	if i.FolderPath != "" {
		return i.FolderPath
	}
	return i.URL
}

// MethodBadge returns a styled method badge with color based on HTTP method
func (i RequestItem) MethodBadge() string {
	color := getMethodColor(i.Method)
	methodStyle := lipgloss.NewStyle().
		Foreground(color).
		Bold(true).
		Padding(0, 1)

	return methodStyle.Render(fmt.Sprintf("%-6s", i.Method))
}

// getMethodColor returns the lipgloss color for an HTTP method using switch
func getMethodColor(method string) lipgloss.Color {
	switch method {
	case "GET":
		return lipgloss.Color("green")
	case "POST":
		return lipgloss.Color("blue")
	case "PUT":
		return lipgloss.Color("yellow")
	case "DELETE":
		return lipgloss.Color("red")
	case "PATCH":
		return lipgloss.Color("magenta")
	case "HEAD", "OPTIONS":
		return lipgloss.Color("cyan")
	default:
		return lipgloss.Color("white")
	}
}

// RenderRequestItem renders a request item with method badge and name
func RenderRequestItem(item RequestItem, selected bool) string {
	var sb strings.Builder

	methodBadge := item.MethodBadge()

	if selected {
		sb.WriteString("▶ ")
	} else {
		sb.WriteString("  ")
	}

	sb.WriteString(methodBadge)
	sb.WriteString(" ")
	sb.WriteString(item.Name)

	return sb.String()
}

// FolderNode represents a folder in the tree
type FolderNode struct {
	Name      string
	Path      string
	Requests  []*types.SavedRequest
	Children  map[string]*FolderNode
	Collapsed bool
}

// CollectionGroup represents a collection with its requests
type CollectionGroup struct {
	Name     string
	Requests []*types.SavedRequest
}
