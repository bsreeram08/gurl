package tui

// Layout holds the calculated dimensions for each panel
type Layout struct {
	SidebarWidth int
	MainWidth    int
	StatusHeight int
}

// CalculateLayout computes responsive panel dimensions
// Sidebar: 25% width (minimum 30 chars)
// Main: remaining space
// Status: 1 line at bottom
func CalculateLayout(width, height int) Layout {
	// Minimum sidebar width
	minSidebar := 30
	// Calculate 25% for sidebar
	sidebar25 := width * 25 / 100

	var sidebarWidth int
	if width < 60 {
		sidebarWidth = width / 2
		if sidebarWidth < minSidebar {
			sidebarWidth = minSidebar
		}
		if sidebarWidth > 30 {
			sidebarWidth = 30
		}
	} else {
		sidebarWidth = max(minSidebar, sidebar25)
		sidebarWidth = min(sidebarWidth, width/2)
	}

	// Main panel gets the remainder
	mainWidth := width - sidebarWidth

	// Status bar is always 1 line
	statusHeight := 1

	return Layout{
		SidebarWidth: sidebarWidth,
		MainWidth:    mainWidth,
		StatusHeight: statusHeight,
	}
}

// MainHeight returns the height available for the main content area
// (total height minus status bar and padding)
func (l Layout) MainHeight(totalHeight int) int {
	h := totalHeight - l.StatusHeight - 1 // 1 for border/padding
	if h < 1 {
		return 1
	}
	return h
}
