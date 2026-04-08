package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

// RequestList is a bubbletea sub-model wrapping bubbles/list for request navigation
type RequestList struct {
	list        list.Model
	db          storage.DB
	items       []RequestItem
	folders     map[string]*FolderNode
	collections map[string]*CollectionGroup
	filtering   bool
	filterText  string
	msgs        []tea.Msg
	width       int
	height      int
}

// RequestSelectedMsg is sent when a request is selected
type RequestSelectedMsg struct {
	Request *types.SavedRequest
}

// NewRequestList creates a new RequestList component
func NewRequestList(db storage.DB) *RequestList {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false

	// Start with empty list, dimensions will be set via SetSize
	l := list.New([]list.Item{}, delegate, 30, 20)
	l.SetShowTitle(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)

	rl := &RequestList{
		list:        l,
		db:          db,
		items:       []RequestItem{},
		folders:     make(map[string]*FolderNode),
		collections: make(map[string]*CollectionGroup),
		filtering:   false,
		filterText:  "",
		msgs:        []tea.Msg{},
	}

	// Load requests from DB
	rl.loadRequests()

	return rl
}

// loadRequests loads requests from the database
func (rl *RequestList) loadRequests() {
	requests, err := rl.db.ListRequests(nil)
	if err != nil {
		requests = []*types.SavedRequest{}
	}

	rl.items = make([]RequestItem, len(requests))
	for i, req := range requests {
		rl.items[i] = RequestItem{
			SavedRequest: req,
			FolderPath:   req.Folder,
		}
	}

	// Update list items
	listItems := make([]list.Item, len(rl.items))
	for i, item := range rl.items {
		listItems[i] = item
	}
	rl.list.SetItems(listItems)
}

// buildFolderTree builds a folder hierarchy from requests
func (rl *RequestList) buildFolderTree() {
	rl.folders = make(map[string]*FolderNode)

	for _, req := range rl.items {
		if req.Folder == "" {
			continue
		}

		parts := strings.Split(req.Folder, "/")
		currentPath := ""
		parent := (*FolderNode)(nil)

		for _, part := range parts {
			if currentPath == "" {
				currentPath = part
			} else {
				currentPath = currentPath + "/" + part
			}

			if _, exists := rl.folders[currentPath]; !exists {
				rl.folders[currentPath] = &FolderNode{
					Name:      part,
					Path:      currentPath,
					Children:  make(map[string]*FolderNode),
					Collapsed: true, // Folders start collapsed as per spec
				}
			}

			if parent != nil {
				parent.Children[part] = rl.folders[currentPath]
			}
			parent = rl.folders[currentPath]
		}

		if parent != nil {
			parent.Requests = append(parent.Requests, req.SavedRequest)
		}
	}
}

// toggleFolder expands or collapses a folder
func (rl *RequestList) toggleFolder(path string) {
	if folder, exists := rl.folders[path]; exists {
		folder.Collapsed = !folder.Collapsed
	}
}

// groupByCollection groups requests by collection
func (rl *RequestList) groupByCollection() {
	rl.collections = make(map[string]*CollectionGroup)

	// Add to "" (no collection) group first
	rl.collections[""] = &CollectionGroup{Name: "", Requests: []*types.SavedRequest{}}

	for _, req := range rl.items {
		collectionName := req.Collection
		if collectionName == "" {
			collectionName = ""
			rl.collections[""].Requests = append(rl.collections[""].Requests, req.SavedRequest)
		} else {
			if _, exists := rl.collections[collectionName]; !exists {
				rl.collections[collectionName] = &CollectionGroup{
					Name:     collectionName,
					Requests: []*types.SavedRequest{},
				}
			}
			rl.collections[collectionName].Requests = append(rl.collections[collectionName].Requests, req.SavedRequest)
		}
	}
}

// handleKeyPress handles keyboard input for navigation and filtering
func (rl *RequestList) handleKeyPress(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyRunes:
		runeStr := string(msg.Runes)
		switch runeStr {
		case "/":
			rl.filtering = true
			rl.list.SetFilterText("")
			rl.list.SetFilteringEnabled(true)
			return
		case "j":
			rl.list.CursorDown()
			return
		case "k":
			rl.list.CursorUp()
			return
		}
	case tea.KeyUp:
		rl.list.CursorUp()
		return
	case tea.KeyDown:
		rl.list.CursorDown()
		return
	case tea.KeyEnter:
		// Select current item
		if selected := rl.list.SelectedItem(); selected != nil {
			if reqItem, ok := selected.(RequestItem); ok {
				rl.msgs = append(rl.msgs, RequestSelectedMsg{Request: reqItem.SavedRequest})
			}
		}
		return
	case tea.KeyEscape:
		if rl.filtering {
			rl.filtering = false
			rl.filterText = ""
			rl.list.ResetFilter()
		}
		return
	}

	// Pass to list for filter input
	if rl.filtering {
		rl.list, _ = rl.list.Update(msg)
		rl.filterText = rl.list.FilterValue()
	}
}

// emptyStateView returns the view for empty state
func (rl *RequestList) emptyStateView() string {
	emptyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(1, 0)

	return emptyStyle.Render("No requests. Save one first.")
}

// View renders the request list
func (rl *RequestList) View() string {
	if len(rl.items) == 0 {
		return rl.emptyStateView()
	}

	return rl.list.View()
}

// Update implements tea.Model.Update
func (rl *RequestList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		rl.width = msg.Width
		rl.height = msg.Height
		rl.list.SetSize(msg.Width, msg.Height)
		return rl, nil

	case tea.KeyMsg:
		rl.handleKeyPress(msg)
	}

	rl.list, _ = rl.list.Update(msg)
	return rl, nil
}

// Init implements tea.Model.Init
func (rl *RequestList) Init() tea.Cmd {
	return nil
}

// GetSelectedRequest returns the currently selected request
func (rl *RequestList) GetSelectedRequest() *types.SavedRequest {
	selected := rl.list.SelectedItem()
	if selected == nil {
		return nil
	}

	if reqItem, ok := selected.(RequestItem); ok {
		return reqItem.SavedRequest
	}
	return nil
}

// GetMessages returns any messages generated by the list
func (rl *RequestList) GetMessages() []tea.Msg {
	msgs := rl.msgs
	rl.msgs = []tea.Msg{}
	return msgs
}

// SetSize updates the list dimensions
func (rl *RequestList) SetSize(width, height int) {
	rl.width = width
	rl.height = height
	rl.list.SetSize(width, height)
}

// FilterItems filters the list by a pattern
func (rl *RequestList) FilterItems(pattern string) {
	rl.list.SetFilterText(pattern)
	rl.filterText = pattern
}
