package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sreeram/gurl/pkg/types"
)

var (
	pickerSelected = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)
	pickerNormal   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	pickerFilter   = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	pickerBorder   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1)
	pickerHint = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
)

// Picker is a minimal fuzzy-search request picker for gurl run.
type Picker struct {
	all      []*types.SavedRequest
	filtered []*types.SavedRequest
	cursor   int
	query    string
	chosen   string
	canceled bool
}

// NewPicker creates a picker loaded with all saved requests.
func NewPicker(requests []*types.SavedRequest) *Picker {
	p := &Picker{all: requests}
	p.applyFilter()
	return p
}

// Chosen returns the selected request name, or "" if canceled.
func (p *Picker) Chosen() string {
	if p.canceled {
		return ""
	}
	return p.chosen
}

func (p *Picker) Init() tea.Cmd { return nil }

func (p *Picker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			p.canceled = true
			return p, tea.Quit

		case "enter":
			if len(p.filtered) > 0 {
				p.chosen = p.filtered[p.cursor].Name
			}
			return p, tea.Quit

		case "up", "ctrl+p":
			if p.cursor > 0 {
				p.cursor--
			} else if len(p.filtered) > 0 {
				p.cursor = len(p.filtered) - 1
			}

		case "down", "ctrl+n":
			if p.cursor < len(p.filtered)-1 {
				p.cursor++
			} else if len(p.filtered) > 0 {
				p.cursor = 0
			}

		case "backspace":
			if len(p.query) > 0 {
				p.query = p.query[:len(p.query)-1]
				p.applyFilter()
			}

		default:
			// Only accept printable single chars
			if len(msg.Runes) == 1 {
				p.query += string(msg.Runes)
				p.applyFilter()
			}
		}
	}
	return p, nil
}

func (p *Picker) View() string {
	var sb strings.Builder

	// Filter prompt
	prompt := pickerFilter.Render("> ") + p.query + "█"
	sb.WriteString(prompt)
	sb.WriteString("\n\n")

	if len(p.filtered) == 0 {
		sb.WriteString(pickerNormal.Render("  no requests match"))
	} else {
		for i, req := range p.filtered {
			line := "  " + req.Name
			if req.Collection != "" {
				line += pickerNormal.Render("  " + req.Collection)
			}
			if i == p.cursor {
				sb.WriteString(pickerSelected.Render("▶ " + req.Name))
				if req.Collection != "" {
					sb.WriteString(pickerNormal.Render("  " + req.Collection))
				}
			} else {
				sb.WriteString(pickerNormal.Render(line))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(pickerHint.Render("↑/↓ navigate · enter run · esc cancel"))

	return pickerBorder.Render(sb.String())
}

func (p *Picker) applyFilter() {
	p.cursor = 0
	if p.query == "" {
		p.filtered = make([]*types.SavedRequest, len(p.all))
		copy(p.filtered, p.all)
		return
	}
	q := strings.ToLower(p.query)
	p.filtered = p.filtered[:0]
	for _, r := range p.all {
		if strings.Contains(strings.ToLower(r.Name), q) ||
			strings.Contains(strings.ToLower(r.Collection), q) {
			p.filtered = append(p.filtered, r)
		}
	}
}

// RunPicker opens the interactive picker and returns the chosen request name.
// Returns "" if the user cancels.
func RunPicker(requests []*types.SavedRequest) (string, error) {
	if len(requests) == 0 {
		return "", nil
	}
	picker := NewPicker(requests)
	prog := tea.NewProgram(picker)
	m, err := prog.Run()
	if err != nil {
		return "", err
	}
	return m.(*Picker).Chosen(), nil
}
