package tui

import (
	"fmt"
	"net/url"
	"strings"

	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

const (
	MinVisibleItems   = 1
	DefaultItemHeight = 3
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

	var sb strings.Builder
	end := rl.scrollOffset + rl.visibleCount
	if end > len(rl.items) {
		end = len(rl.items)
	}

	if rl.scrollOffset > 0 {
		sb.WriteString(Style.Hint.Render(fmt.Sprintf("Showing %d-%d of %d", rl.scrollOffset+1, end, len(rl.items))))
		sb.WriteString("\n\n")
	}

	for i := rl.scrollOffset; i < end; i++ {
		req := rl.items[i]
		contentWidth := max(20, rl.width-3)
		row := rl.renderRequestCard(req, i == rl.cursor, contentWidth)
		sb.WriteString(row)
		if i < end-1 {
			sb.WriteString("\n")
		}
	}

	if end < len(rl.items) {
		sb.WriteString("\n")
		sb.WriteString(Style.Hint.Render(fmt.Sprintf("More below (%d remaining)", len(rl.items)-end)))
	}

	return sb.String()
}

func (rl *RequestList) View() tea.View { return tea.NewView(rl.ViewTree()) }

func (rl *RequestList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Code {
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

func (rl *RequestList) renderRequestCard(req *types.SavedRequest, selected bool, width int) string {
	title := req.Name
	if title == "" {
		title = req.URL
	}

	primaryWidth := max(12, width-10)
	subtitle, meta := requestCardLines(req)
	prefix := "  "
	titleStyle := Style.PlainText.Copy().Bold(true)
	if selected {
		prefix = "> "
		titleStyle = Style.SelectedItem.Copy().Bold(true)
	}

	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		prefix,
		RenderMethodBadge(req.Method),
		" ",
		titleStyle.Render(truncateText(title, primaryWidth)),
	)

	lines := []string{
		header,
		Style.Hint.Render("  " + truncateText(subtitle, max(10, width-2))),
	}
	if meta != "" {
		lines = append(lines, Style.Hint.Render("  "+truncateText(meta, max(10, width-2))))
	}

	style := Style.Card.Width(width)
	if selected {
		style = Style.CardSelected.Width(width)
	}
	return style.Render(strings.Join(lines, "\n"))
}

func requestCardLines(req *types.SavedRequest) (string, string) {
	host := req.URL
	path := req.URL

	if parsed, err := url.Parse(req.URL); err == nil {
		if parsed.Host != "" {
			host = parsed.Host
		}
		switch {
		case parsed.Path != "":
			path = parsed.Path
		case parsed.Host != "":
			path = parsed.Host
		}
		if parsed.RawQuery != "" {
			path += "?" + parsed.RawQuery
		}
	}

	metaParts := []string{host}
	if req.Collection != "" {
		metaParts = append(metaParts, "Collection: "+req.Collection)
	}
	if req.Folder != "" {
		metaParts = append(metaParts, "Folder: "+req.Folder)
	}

	return path, strings.Join(metaParts, "  ·  ")
}
