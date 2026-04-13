package tui

// LayoutMode describes how the workspace is arranged for the current terminal.
type LayoutMode int

const (
	LayoutThreePane LayoutMode = iota
	LayoutSplitRight
	LayoutStacked
)

// Layout holds the calculated dimensions for each visible workspace pane.
type Layout struct {
	Mode            LayoutMode
	Width           int
	Height          int
	FooterHeight    int
	WorkspaceHeight int

	SidebarWidth  int
	SidebarHeight int

	EditorWidth  int
	EditorHeight int

	ResponseWidth  int
	ResponseHeight int
}

// CalculateLayout computes a responsive workspace layout.
func CalculateLayout(width, height int) Layout {
	if width < 1 {
		width = 1
	}
	if height < 3 {
		height = 3
	}

	footerHeight := 1
	workspaceHeight := max(2, height-footerHeight)

	if width >= 140 {
		sidebarWidth := max(32, width*40/100)
		editorWidth := max(42, width*35/100)
		responseWidth := width - sidebarWidth - editorWidth

		if responseWidth < 30 {
			shortfall := 30 - responseWidth

			editorShrink := min(shortfall, max(0, editorWidth-40))
			editorWidth -= editorShrink
			shortfall -= editorShrink

			if shortfall > 0 {
				sidebarShrink := min(shortfall, max(0, sidebarWidth-30))
				sidebarWidth -= sidebarShrink
			}

			responseWidth = width - sidebarWidth - editorWidth
		}

		if responseWidth >= 28 {
			return Layout{
				Mode:            LayoutThreePane,
				Width:           width,
				Height:          height,
				FooterHeight:    footerHeight,
				WorkspaceHeight: workspaceHeight,
				SidebarWidth:    sidebarWidth,
				SidebarHeight:   workspaceHeight,
				EditorWidth:     editorWidth,
				EditorHeight:    workspaceHeight,
				ResponseWidth:   responseWidth,
				ResponseHeight:  workspaceHeight,
			}
		}
	}

	if width >= 100 {
		sidebarWidth := max(30, width*38/100)
		sidebarWidth = min(sidebarWidth, width-46)

		rightWidth := width - sidebarWidth
		editorHeight := max(12, workspaceHeight*58/100)
		editorHeight = min(editorHeight, workspaceHeight-10)
		responseHeight := workspaceHeight - editorHeight

		if responseHeight >= 8 {
			return Layout{
				Mode:            LayoutSplitRight,
				Width:           width,
				Height:          height,
				FooterHeight:    footerHeight,
				WorkspaceHeight: workspaceHeight,
				SidebarWidth:    sidebarWidth,
				SidebarHeight:   workspaceHeight,
				EditorWidth:     rightWidth,
				EditorHeight:    editorHeight,
				ResponseWidth:   rightWidth,
				ResponseHeight:  responseHeight,
			}
		}
	}

	sidebarHeight := max(7, workspaceHeight*28/100)
	editorHeight := max(10, workspaceHeight*40/100)
	responseHeight := workspaceHeight - sidebarHeight - editorHeight

	if responseHeight < 8 {
		shortfall := 8 - responseHeight

		editorShrink := min(shortfall, max(0, editorHeight-8))
		editorHeight -= editorShrink
		shortfall -= editorShrink

		if shortfall > 0 {
			sidebarHeight = max(6, sidebarHeight-shortfall)
		}

		responseHeight = workspaceHeight - sidebarHeight - editorHeight
	}

	return Layout{
		Mode:            LayoutStacked,
		Width:           width,
		Height:          height,
		FooterHeight:    footerHeight,
		WorkspaceHeight: workspaceHeight,
		SidebarWidth:    width,
		SidebarHeight:   sidebarHeight,
		EditorWidth:     width,
		EditorHeight:    editorHeight,
		ResponseWidth:   width,
		ResponseHeight:  responseHeight,
	}
}
