package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

const (
	MinVisibleItems   = 5
	DefaultItemHeight = 1
)

// RequestList is a simple sidebar list with keyboard navigation and custom virtualization.
type RequestList struct {
	items        []*types.SavedRequest
	cursor       int
	width        int
	height       int
	scrollOffset int
	visibleCount int
	db           storage.DB
	filtering    bool
	filterText   string
	msgs         []tea.Msg
}

// RequestSelectedMsg is sent when a request is selected
type RequestSelectedMsg struct {
	Request *types.SavedRequest
}

func NewRequestList(db storage.DB) *RequestList {
	return &RequestList{
		db:           db,
		items:        []*types.SavedRequest{},
		cursor:       0,
		scrollOffset: 0,
		visibleCount: MinVisibleItems,
	}
}

func (rl *RequestList) SetRequests(requests []*types.SavedRequest) {
	rl.items = requests
	if rl.cursor >= len(rl.items) {
		rl.cursor = len(rl.items) - 1
	}
	if rl.cursor < 0 {
		rl.cursor = 0
	}
	rl.ensureScrollOffset()
}

func (rl *RequestList) SetSize(w, h int) {
	rl.width = w
	rl.height = h
	rl.visibleCount = h / DefaultItemHeight
	if rl.visibleCount < MinVisibleItems {
		rl.visibleCount = MinVisibleItems
	}
	rl.ensureScrollOffset()
}

func (rl *RequestList) CursorUp() {
	if rl.cursor > 0 {
		rl.cursor--
		rl.ensureScrollOffset()
	}
}

func (rl *RequestList) CursorDown() {
	if rl.cursor < len(rl.items)-1 {
		rl.cursor++
		rl.ensureScrollOffset()
	}
}

func (rl *RequestList) ensureScrollOffset() {
	if rl.visibleCount == 0 {
		return
	}
	if rl.scrollOffset > rl.cursor {
		rl.scrollOffset = rl.cursor
	}
	if rl.scrollOffset+rl.visibleCount <= rl.cursor {
		rl.scrollOffset = rl.cursor - rl.visibleCount + 1
	}
	maxOffset := len(rl.items) - rl.visibleCount
	if maxOffset < 0 {
		maxOffset = 0
	}
	if rl.scrollOffset > maxOffset {
		rl.scrollOffset = maxOffset
	}
}

func (rl *RequestList) GetSelectedRequest() *types.SavedRequest {
	if len(rl.items) == 0 || rl.cursor >= len(rl.items) {
		return nil
	}
	return rl.items[rl.cursor]
}

func (rl *RequestList) ToggleFolderAtCursor() {}

func (rl *RequestList) FilterItems(pattern string) {
	rl.filterText = pattern
}

func (rl *RequestList) GetMessages() []tea.Msg { return nil }

func (rl *RequestList) ViewTree() string {
	if len(rl.items) == 0 {
		return Style.PlainText.Render("\n  No requests saved.\n  Use 'gurl save' to add one.")
	}

	scrollIndicator := ""
	if rl.scrollOffset > 0 {
		scrollIndicator += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("▲\n")
	}

	var sb strings.Builder
	end := rl.scrollOffset + rl.visibleCount
	if end > len(rl.items) {
		end = len(rl.items)
	}

	for i := rl.scrollOffset; i < end; i++ {
		req := rl.items[i]
		methodColor := getMethodColor(req.Method)
		methodStyle := lipgloss.NewStyle().Foreground(methodColor).Bold(true)

		name := req.Name
		maxName := rl.width - 12
		if maxName < 10 {
			maxName = 10
		}
		if len(name) > maxName {
			name = name[:maxName-3] + "..."
		}

		line := fmt.Sprintf("%s %s", methodStyle.Render(req.Method), name)

		if req.Collection != "" {
			collStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			line += collStyle.Render(" [" + req.Collection + "]")
		}

		if i == rl.cursor {
			sb.WriteString(Style.SelectedItem.Render("▶ " + line))
		} else {
			sb.WriteString(Style.ListItem.Render("  " + line))
		}
		sb.WriteString("\n")
	}

	if end < len(rl.items) {
		scrollIndicator += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("▼")
	}

	return scrollIndicator + sb.String()
}

func (rl *RequestList) View() string { return rl.ViewTree() }

func (rl *RequestList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			rl.CursorUp()
		case tea.KeyDown:
			rl.CursorDown()
		}
	case tea.WindowSizeMsg:
		rl.SetSize(msg.Width, msg.Height)
	}
	return rl, nil
}

func (rl *RequestList) Init() tea.Cmd { return nil }
